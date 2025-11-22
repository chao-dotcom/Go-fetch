package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"

	"taskqueue/internal/api/middleware"
	"taskqueue/internal/broker"
	"taskqueue/internal/models"
	"taskqueue/internal/storage"
	"taskqueue/pkg/metrics"
)

var tracer = otel.Tracer("taskqueue-api")

// JobHandler hosts job endpoints.
type JobHandler struct {
	store    storage.Storage
	broker   broker.Broker
	logger   *zap.Logger
	queue    string
	recorder MetricsRecorder
}

// NewJobHandler creates a handler.
func NewJobHandler(store storage.Storage, broker broker.Broker, logger *zap.Logger, queueName string, recorder MetricsRecorder) *JobHandler {
	return &JobHandler{
		store:    store,
		broker:   broker,
		logger:   logger,
		queue:    queueName,
		recorder: recorder,
	}
}

// CreateJobRequest is the POST body.
type CreateJobRequest struct {
	Type           string      `json:"type" binding:"required"`
	Payload        interface{} `json:"payload" binding:"required"`
	Priority       int         `json:"priority"`
	MaxAttempts    int         `json:"max_attempts" binding:"omitempty,min=1,max=10"`
	TimeoutSeconds int         `json:"timeout_seconds"`
	IdempotencyKey string      `json:"idempotency_key"`
	ScheduledFor   *time.Time  `json:"scheduled_for"`
}

// CreateJob handles job submission.
func (h *JobHandler) CreateJob(c *gin.Context) {
	ctx, span := tracer.Start(c.Request.Context(), "CreateJob")
	defer span.End()
	logger := h.loggerWithContext(c)

	var req CreateJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		span.RecordError(err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.MaxAttempts == 0 {
		req.MaxAttempts = 3
	}
	if req.MaxAttempts < 1 || req.MaxAttempts > 10 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "max_attempts must be between 1 and 10"})
		return
	}
	if req.Priority < -10 || req.Priority > 10 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "priority must be between -10 and 10"})
		return
	}
	if req.TimeoutSeconds <= 0 {
		req.TimeoutSeconds = 300
	}

	if req.IdempotencyKey != "" {
		if s, ok := h.store.(idempotentStore); ok {
			if existing, err := s.GetJobByIdempotencyKey(ctx, req.IdempotencyKey); err == nil && existing != nil {
				c.JSON(http.StatusOK, gin.H{
					"job_id":  existing.ID,
					"status":  existing.Status,
					"message": "job already exists for idempotency key",
				})
				return
			}
		}
	}

	payload, err := json.Marshal(req.Payload)
	if err != nil {
		span.RecordError(err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}

	job := &models.Job{
		ID:             uuid.NewString(),
		Type:           req.Type,
		Payload:        payload,
		Priority:       req.Priority,
		MaxAttempts:    req.MaxAttempts,
		Status:         models.JobStatusQueued,
		TimeoutSeconds: req.TimeoutSeconds,
		IdempotencyKey: req.IdempotencyKey,
		ScheduledFor:   req.ScheduledFor,
	}
	if err := h.store.CreateJob(ctx, job); err != nil {
		span.RecordError(err)
		logger.Error("create job failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to persist job"})
		return
	}

	msg := &broker.Message{
		JobID:       job.ID,
		Type:        job.Type,
		Payload:     job.Payload,
		MaxAttempts: job.MaxAttempts,
		Priority:    job.Priority,
	}
	if err := h.broker.Enqueue(ctx, msg); err != nil {
		span.RecordError(err)
		logger.Error("enqueue failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to enqueue"})
		return
	}
	if h.recorder != nil {
		h.recorder.RecordJobSubmitted(job.Type)
	}
	metrics.JobEnqueuedTotal.WithLabelValues(job.Type).Inc()

	span.SetAttributes(attribute.String("job.id", job.ID), attribute.String("job.type", job.Type))
	c.JSON(http.StatusAccepted, gin.H{"job_id": job.ID, "status": job.Status})
}

// GetJob retrieves job details.
func (h *JobHandler) GetJob(c *gin.Context) {
	ctx, span := tracer.Start(c.Request.Context(), "GetJob")
	defer span.End()

	logger := h.loggerWithContext(c)
	jobID := c.Param("id")
	span.SetAttributes(attribute.String("job.id", jobID))

	job, err := h.store.GetJob(ctx, jobID)
	if err != nil {
		if err == storage.ErrJobNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
			return
		}
		logger.Error("failed to fetch job", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch job"})
		return
	}
	c.JSON(http.StatusOK, job)
}

// ListJobs returns paginated jobs.
func (h *JobHandler) ListJobs(c *gin.Context) {
	ctx, span := tracer.Start(c.Request.Context(), "ListJobs")
	defer span.End()

	logger := h.loggerWithContext(c)
	filter := storage.ListFilter{
		Status: models.JobStatus(c.Query("status")),
		Type:   c.Query("type"),
		Limit:  queryInt(c, "limit", 50),
		Offset: queryInt(c, "offset", 0),
	}
	jobs, total, err := h.store.ListJobs(ctx, filter)
	if err != nil {
		logger.Error("failed to list jobs", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list jobs"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"jobs":   jobs,
		"total":  total,
		"limit":  filter.Limit,
		"offset": filter.Offset,
	})
}

// CancelJob marks a job as cancelled if still queued.
func (h *JobHandler) CancelJob(c *gin.Context) {
	ctx := c.Request.Context()
	job, err := h.store.GetJob(ctx, c.Param("id"))
	if err != nil {
		if err == storage.ErrJobNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch job"})
		return
	}
	if job.Status != models.JobStatusQueued {
		c.JSON(http.StatusBadRequest, gin.H{"error": "job already running or finished"})
		return
	}
	job.Status = models.JobStatusFailed
	job.Error = "cancelled"
	if err := h.store.UpdateJob(ctx, job); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to cancel"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "cancelled"})
}

// RetryJob allows manual retry.
func (h *JobHandler) RetryJob(c *gin.Context) {
	logger := h.loggerWithContext(c)
	ctx := c.Request.Context()
	job, err := h.store.GetJob(ctx, c.Param("id"))
	if err != nil {
		if err == storage.ErrJobNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch job"})
		return
	}
	if job.Status != models.JobStatusFailed {
		c.JSON(http.StatusBadRequest, gin.H{"error": "job is not failed"})
		return
	}
	job.Status = models.JobStatusQueued
	job.Attempts = 0
	job.Error = ""
	if err := h.store.UpdateJob(ctx, job); err != nil {
		logger.Error("failed to update job", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update job"})
		return
	}
	msg := &broker.Message{
		JobID:       job.ID,
		Type:        job.Type,
		Payload:     job.Payload,
		MaxAttempts: job.MaxAttempts,
		Priority:    job.Priority,
	}
	if err := h.broker.Enqueue(ctx, msg); err != nil {
		logger.Error("failed to enqueue job", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to enqueue"})
		return
	}
	metrics.JobEnqueuedTotal.WithLabelValues(job.Type).Inc()
	c.JSON(http.StatusOK, gin.H{"status": "requeued"})
}

// GetJobLogs returns stored logs.
func (h *JobHandler) GetJobLogs(c *gin.Context) {
	logs, err := h.store.ListJobLogs(c.Request.Context(), c.Param("id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load logs"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"logs": logs})
}

// QueueStats returns basic queue metrics (memory implementation only).
func (h *JobHandler) QueueStats(c *gin.Context) {
	depth := -1
	if dr, ok := h.broker.(interface{ Depth() int }); ok {
		depth = dr.Depth()
	}
	c.JSON(http.StatusOK, []gin.H{
		{"name": h.queue, "depth": depth, "processing_rate": 0, "avg_latency_ms": 0},
	})
}

// ListWorkers returns heartbeat information.
func (h *JobHandler) ListWorkers(c *gin.Context) {
	workers, err := h.store.ListWorkers(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list workers"})
		return
	}
	c.JSON(http.StatusOK, workers)
}

func queryInt(c *gin.Context, key string, fallback int) int {
	if val := c.Query(key); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil {
			return parsed
		}
	}
	return fallback
}

// MetricsRecorder captures interesting API-level metrics.
type MetricsRecorder interface {
	RecordJobSubmitted(jobType string)
}

type idempotentStore interface {
	GetJobByIdempotencyKey(ctx context.Context, key string) (*models.Job, error)
}

func (h *JobHandler) loggerWithContext(c *gin.Context) *zap.Logger {
	return h.logger.With(zap.String("request_id", middleware.RequestIDValue(c)))
}

