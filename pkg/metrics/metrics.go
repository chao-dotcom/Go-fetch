package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// JobProcessingSeconds tracks worker execution latency.
	JobProcessingSeconds = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "taskqueue",
		Name:      "job_processing_seconds",
		Help:      "Time spent executing jobs",
		Buckets:   prometheus.DefBuckets,
	}, []string{"job_type", "status"})

	// JobFailuresTotal counts failed jobs per type.
	JobFailuresTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "taskqueue",
		Name:      "job_failures_total",
		Help:      "Number of failed jobs",
	}, []string{"job_type"})

	// JobEnqueuedTotal counts enqueued jobs.
	JobEnqueuedTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "taskqueue",
		Name:      "jobs_enqueued_total",
		Help:      "Number of jobs enqueued by the API",
	}, []string{"job_type"})

	// WorkerActiveGauge tracks active jobs per worker.
	WorkerActiveGauge = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "taskqueue",
		Name:      "worker_active_jobs",
		Help:      "Number of active jobs processed by a worker",
	}, []string{"worker_id"})

	// QueueDepthGauge tracks queue depth (memory broker only).
	QueueDepthGauge = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "taskqueue",
		Name:      "queue_depth",
		Help:      "Approximate queue depth",
	}, []string{"queue"})

	// WorkerHeartbeatGauge tracks heartbeat timestamps.
	WorkerHeartbeatGauge = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "taskqueue",
		Name:      "worker_heartbeat_seconds",
		Help:      "Unix timestamp of the last worker heartbeat",
	}, []string{"worker_id"})

	// WorkerUptimeGauge tracks worker uptime in seconds.
	WorkerUptimeGauge = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "taskqueue",
		Name:      "worker_uptime_seconds",
		Help:      "Worker uptime in seconds",
	}, []string{"worker_id"})

	// APIRequestDuration tracks HTTP request latency.
	APIRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "taskqueue",
		Name:      "api_request_duration_seconds",
		Help:      "HTTP request latency in seconds",
		Buckets:   prometheus.DefBuckets,
	}, []string{"method", "endpoint", "status"})
)

// ObserveJobDuration records job duration with status and type labels.
func ObserveJobDuration(jobType, status string, duration time.Duration) {
	JobProcessingSeconds.WithLabelValues(jobType, status).Observe(duration.Seconds())
}

// ObserveAPIRequest records API request duration.
func ObserveAPIRequest(method, endpoint, status string, duration time.Duration) {
	APIRequestDuration.WithLabelValues(method, endpoint, status).Observe(duration.Seconds())
}


