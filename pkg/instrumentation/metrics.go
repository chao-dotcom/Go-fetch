package instrumentation

import (
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics holds all Prometheus metrics
type Metrics struct {
	// Job metrics
	jobsSubmittedTotal     *prometheus.CounterVec
	jobsProcessedTotal     *prometheus.CounterVec
	jobsFailedTotal        *prometheus.CounterVec
	jobsRetriedTotal       *prometheus.CounterVec
	jobProcessingDuration  *prometheus.HistogramVec
	jobQueueLatency        *prometheus.HistogramVec

	// Worker metrics
	workerActiveJobs       *prometheus.GaugeVec
	workerTotalProcessed   *prometheus.CounterVec
	workerTotalFailed      *prometheus.CounterVec
	workerHeartbeat        *prometheus.GaugeVec
	workerUptime           *prometheus.GaugeVec

	// Queue metrics
	queueDepth             *prometheus.GaugeVec
	queuePendingMessages   *prometheus.GaugeVec
	queueProcessingRate    *prometheus.GaugeVec

	// API metrics
	httpRequestsTotal      *prometheus.CounterVec
	httpRequestDuration    *prometheus.HistogramVec
	httpRequestsInFlight   *prometheus.GaugeVec

	// System metrics
	databaseConnectionsActive prometheus.Gauge
	redisConnectionsActive    prometheus.Gauge
}

// NewMetrics creates and registers all Prometheus metrics
func NewMetrics(namespace string) *Metrics {
	if namespace == "" {
		namespace = "taskqueue"
	}

	m := &Metrics{
		// Job submission metrics
		jobsSubmittedTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "jobs_submitted_total",
				Help:      "Total number of jobs submitted",
			},
			[]string{"type"},
		),

		// Job processing metrics
		jobsProcessedTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "jobs_processed_total",
				Help:      "Total number of jobs processed",
			},
			[]string{"type", "status"}, // status: success, failure
		),

		jobsFailedTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "jobs_failed_total",
				Help:      "Total number of jobs that failed",
			},
			[]string{"type", "reason"}, // reason: timeout, panic, error
		),

		jobsRetriedTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "jobs_retried_total",
				Help:      "Total number of job retry attempts",
			},
			[]string{"type"},
		),

		// Job duration histogram
		jobProcessingDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "job_processing_duration_seconds",
				Help:      "Job processing duration in seconds",
				Buckets:   []float64{0.1, 0.5, 1, 2, 5, 10, 30, 60, 120, 300}, // 100ms to 5min
			},
			[]string{"type"},
		),

		// Queue latency (time from submission to start)
		jobQueueLatency: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "job_queue_latency_seconds",
				Help:      "Time jobs spend waiting in queue",
				Buckets:   []float64{0.1, 0.5, 1, 5, 10, 30, 60, 300, 600}, // 100ms to 10min
			},
			[]string{"type"},
		),

		// Worker metrics
		workerActiveJobs: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "worker_active_jobs",
				Help:      "Number of jobs currently being processed by worker",
			},
			[]string{"worker_id", "hostname"},
		),

		workerTotalProcessed: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "worker_jobs_processed_total",
				Help:      "Total number of jobs processed by worker",
			},
			[]string{"worker_id", "hostname"},
		),

		workerTotalFailed: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "worker_jobs_failed_total",
				Help:      "Total number of jobs failed by worker",
			},
			[]string{"worker_id", "hostname"},
		),

		workerHeartbeat: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "worker_heartbeat_timestamp",
				Help:      "Unix timestamp of last worker heartbeat",
			},
			[]string{"worker_id", "hostname"},
		),

		workerUptime: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "worker_uptime_seconds",
				Help:      "Worker uptime in seconds",
			},
			[]string{"worker_id", "hostname"},
		),

		// Queue metrics
		queueDepth: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "queue_depth",
				Help:      "Number of jobs waiting in queue",
			},
			[]string{"queue"},
		),

		queuePendingMessages: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "queue_pending_messages",
				Help:      "Number of pending (unacknowledged) messages in queue",
			},
			[]string{"queue"},
		),

		queueProcessingRate: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "queue_processing_rate",
				Help:      "Queue processing rate (jobs per second)",
			},
			[]string{"queue"},
		),

		// HTTP API metrics
		httpRequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "http_requests_total",
				Help:      "Total number of HTTP requests",
			},
			[]string{"method", "path", "status"},
		),

		httpRequestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "http_request_duration_seconds",
				Help:      "HTTP request latency in seconds",
				Buckets:   prometheus.DefBuckets, // 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10
			},
			[]string{"method", "path"},
		),

		httpRequestsInFlight: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "http_requests_in_flight",
				Help:      "Number of HTTP requests currently being served",
			},
			[]string{"method"},
		),

		// System metrics
		databaseConnectionsActive: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "database_connections_active",
				Help:      "Number of active database connections",
			},
		),

		redisConnectionsActive: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "redis_connections_active",
				Help:      "Number of active Redis connections",
			},
		),
	}

	return m
}

// Job Metrics

func (m *Metrics) RecordJobSubmitted(jobType string) {
	m.jobsSubmittedTotal.WithLabelValues(jobType).Inc()
}

func (m *Metrics) RecordJobReceived(jobType string) {
	// Can add specific metric if needed, or combine with submitted
	m.jobsSubmittedTotal.WithLabelValues(jobType).Inc()
}

func (m *Metrics) RecordJobProcessed(jobType string, duration time.Duration, success bool) {
	status := "success"
	if !success {
		status = "failure"
	}

	m.jobsProcessedTotal.WithLabelValues(jobType, status).Inc()
	m.jobProcessingDuration.WithLabelValues(jobType).Observe(duration.Seconds())
}

func (m *Metrics) RecordJobFailure(jobType string) {
	m.jobsFailedTotal.WithLabelValues(jobType, "error").Inc()
}

func (m *Metrics) RecordJobFailureWithReason(jobType, reason string) {
	m.jobsFailedTotal.WithLabelValues(jobType, reason).Inc()
}

func (m *Metrics) RecordJobRetry(jobType string) {
	m.jobsRetriedTotal.WithLabelValues(jobType).Inc()
}

func (m *Metrics) RecordQueueLatency(jobType string, latency time.Duration) {
	m.jobQueueLatency.WithLabelValues(jobType).Observe(latency.Seconds())
}

// Worker Metrics

func (m *Metrics) RecordWorkerHeartbeat(workerID, hostname string, activeJobs int) {
	m.workerActiveJobs.WithLabelValues(workerID, hostname).Set(float64(activeJobs))
	m.workerHeartbeat.WithLabelValues(workerID, hostname).SetToCurrentTime()
}

func (m *Metrics) RecordWorkerJobProcessed(workerID, hostname string) {
	m.workerTotalProcessed.WithLabelValues(workerID, hostname).Inc()
}

func (m *Metrics) RecordWorkerJobFailed(workerID, hostname string) {
	m.workerTotalFailed.WithLabelValues(workerID, hostname).Inc()
}

func (m *Metrics) RecordWorkerUptime(workerID, hostname string, uptime time.Duration) {
	m.workerUptime.WithLabelValues(workerID, hostname).Set(uptime.Seconds())
}

func (m *Metrics) SetWorkerActiveJobs(workerID, hostname string, count int) {
	m.workerActiveJobs.WithLabelValues(workerID, hostname).Set(float64(count))
}

// Queue Metrics

func (m *Metrics) SetQueueDepth(queueName string, depth int64) {
	m.queueDepth.WithLabelValues(queueName).Set(float64(depth))
}

func (m *Metrics) SetQueuePendingMessages(queueName string, pending int64) {
	m.queuePendingMessages.WithLabelValues(queueName).Set(float64(pending))
}

func (m *Metrics) SetQueueProcessingRate(queueName string, rate float64) {
	m.queueProcessingRate.WithLabelValues(queueName).Set(rate)
}

// HTTP API Metrics

func (m *Metrics) RecordHTTPRequest(method, path string, statusCode int, duration time.Duration) {
	m.httpRequestsTotal.WithLabelValues(method, path, statusCodeToString(statusCode)).Inc()
	m.httpRequestDuration.WithLabelValues(method, path).Observe(duration.Seconds())
}

func (m *Metrics) IncrementHTTPRequestsInFlight(method string) {
	m.httpRequestsInFlight.WithLabelValues(method).Inc()
}

func (m *Metrics) DecrementHTTPRequestsInFlight(method string) {
	m.httpRequestsInFlight.WithLabelValues(method).Dec()
}

// System Metrics

func (m *Metrics) SetDatabaseConnectionsActive(count int) {
	m.databaseConnectionsActive.Set(float64(count))
}

func (m *Metrics) SetRedisConnectionsActive(count int) {
	m.redisConnectionsActive.Set(float64(count))
}

// Custom metric examples for specific use cases

// RecordJobTimeout records a job timeout event
func (m *Metrics) RecordJobTimeout(jobType string) {
	m.jobsFailedTotal.WithLabelValues(jobType, "timeout").Inc()
}

// RecordJobPanic records a job panic event
func (m *Metrics) RecordJobPanic(jobType string) {
	m.jobsFailedTotal.WithLabelValues(jobType, "panic").Inc()
}

// RecordJobCancelled records a job cancellation
func (m *Metrics) RecordJobCancelled(jobType string) {
	m.jobsFailedTotal.WithLabelValues(jobType, "cancelled").Inc()
}

// Helper functions

func extractHostname(workerID string) string {
	// Extract hostname from worker ID format: "worker-hostname-pid"
	// Try to extract hostname from common formats
	parts := strings.Split(workerID, "-")
	if len(parts) >= 2 {
		return parts[1]
	}
	return "unknown"
}

func statusCodeToString(code int) string {
	switch {
	case code >= 200 && code < 300:
		return "2xx"
	case code >= 300 && code < 400:
		return "3xx"
	case code >= 400 && code < 500:
		return "4xx"
	case code >= 500:
		return "5xx"
	default:
		return strconv.Itoa(code)
	}
}

