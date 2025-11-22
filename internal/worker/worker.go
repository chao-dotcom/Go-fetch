package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"

	"taskqueue/internal/broker"
	"taskqueue/internal/jobs"
	"taskqueue/internal/models"
	"taskqueue/internal/storage"
	"taskqueue/pkg/metrics"
)

// Config configures a worker instance.
type Config struct {
	WorkerID        string
	Hostname        string
	PoolSize        int
	MaxRetries      int
	Stream          string
	ConsumerGroup   string
	ShutdownTimeout time.Duration

	Broker   broker.Broker
	Storage  storage.Storage
	Registry *jobs.Registry
	Logger   *zap.Logger

	Tracer trace.Tracer
}

// Worker consumes jobs from the queue and executes them.
type Worker struct {
	cfg       Config
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	jobChan   chan *broker.Message
	logger    *zap.Logger
	tracer    trace.Tracer
	activeMu  sync.Mutex
	activeCnt int
	startTime time.Time
}

// New creates a worker with sane defaults.
func New(cfg Config) (*Worker, error) {
	if cfg.Registry == nil {
		cfg.Registry = jobs.NewRegistry()
	}
	if cfg.Logger == nil {
		logger, err := zap.NewProduction()
		if err != nil {
			return nil, err
		}
		cfg.Logger = logger
	}
	if cfg.PoolSize <= 0 {
		cfg.PoolSize = 8
	}
	if cfg.MaxRetries <= 0 {
		cfg.MaxRetries = 5
	}
	if cfg.ShutdownTimeout <= 0 {
		cfg.ShutdownTimeout = 30 * time.Second
	}
	if cfg.WorkerID == "" {
		cfg.WorkerID = fmt.Sprintf("worker-%s", uuid.NewString())
	}
	if cfg.Hostname == "" {
		hostname, _ := os.Hostname()
		cfg.Hostname = hostname
	}
	if cfg.Tracer == nil {
		cfg.Tracer = otel.Tracer("taskqueue-worker")
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &Worker{
		cfg:     cfg,
		ctx:     ctx,
		cancel:  cancel,
		jobChan: make(chan *broker.Message, cfg.PoolSize*2),
		logger:  cfg.Logger,
		tracer:  cfg.Tracer,
		startTime: time.Now(),
	}, nil
}

// Start begins consuming and processing jobs until the provided context is cancelled.
func (w *Worker) Start(parent context.Context) error {
	ctx, cancel := context.WithCancel(parent)
	defer cancel()

	if err := w.cfg.Storage.RegisterWorker(ctx, models.WorkerInfo{
		ID:       w.cfg.WorkerID,
		Hostname: w.cfg.Hostname,
		PoolSize: w.cfg.PoolSize,
		Status:   "starting",
	}); err != nil {
		return err
	}

	w.logger.Info("worker starting",
		zap.String("worker_id", w.cfg.WorkerID),
		zap.Int("pool_size", w.cfg.PoolSize),
	)

	// Start pool
	for i := 0; i < w.cfg.PoolSize; i++ {
		w.wg.Add(1)
		go w.processLoop(ctx, i)
	}

	// Start consumer
	w.wg.Add(1)
	go w.consumeLoop(ctx)

	<-ctx.Done()
	w.logger.Info("shutdown signal received")
	waitCtx, waitCancel := context.WithTimeout(context.Background(), w.cfg.ShutdownTimeout)
	done := make(chan struct{})
	go func() {
		w.wg.Wait()
		close(done)
	}()
	select {
	case <-waitCtx.Done():
		w.logger.Warn("worker shutdown timed out")
	case <-done:
		w.logger.Info("worker shutdown complete")
	}
	waitCancel()
	return nil
}

func (w *Worker) consumeLoop(ctx context.Context) {
	defer w.wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		default:
			msgs, err := w.cfg.Broker.Consume(ctx, w.cfg.Stream, w.cfg.WorkerID, w.cfg.PoolSize, time.Second)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				w.logger.Error("consume error", zap.Error(err))
				time.Sleep(time.Second)
				continue
			}
			for _, msg := range msgs {
				select {
				case w.jobChan <- msg:
					w.incrementActive(1)
				case <-ctx.Done():
					return
				}
			}
		}
	}
}

func (w *Worker) processLoop(ctx context.Context, workerIdx int) {
	defer w.wg.Done()
	workerLogger := w.logger.With(zap.Int("goroutine", workerIdx))

	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-w.jobChan:
			w.executeJob(ctx, workerLogger, msg)
			w.incrementActive(-1)
		}
	}
}

func (w *Worker) executeJob(ctx context.Context, logger *zap.Logger, msg *broker.Message) {
	ctx, span := w.tracer.Start(ctx, "worker.execute",
		trace.WithAttributes(
			attribute.String("job.id", msg.JobID),
			attribute.String("job.type", msg.Type),
			attribute.String("queue.stream", w.cfg.Stream),
		))
	defer span.End()

	handler, ok := w.cfg.Registry.HandlerFor(msg.Type)
	if !ok {
		logger.Error("handler not found", zap.String("type", msg.Type))
		span.RecordError(fmt.Errorf("handler not found for %s", msg.Type))
		_ = w.cfg.Broker.Ack(ctx, w.cfg.Stream, msg.ID)
		return
	}

	job, err := w.cfg.Storage.GetJob(ctx, msg.JobID)
	if err != nil {
		logger.Error("job missing", zap.Error(err), zap.String("job_id", msg.JobID))
		span.RecordError(err)
		_ = w.cfg.Broker.Ack(ctx, w.cfg.Stream, msg.ID)
		return
	}

	start := time.Now()
	job.Status = models.JobStatusRunning
	job.StartedAt = ptrTime(start)
	job.WorkerID = w.cfg.WorkerID
	_ = w.cfg.Storage.UpdateJob(ctx, job)
	_ = w.cfg.Storage.AppendJobLog(ctx, models.JobLog{
		JobID:   job.ID,
		Level:   "info",
		Message: "job started",
	})
	finalStatus := models.JobStatusSucceeded

	defer func() {
		if r := recover(); r != nil {
			logger.Error("handler panic", zap.Any("panic", r))
			span.RecordError(fmt.Errorf("panic: %v", r))
			_ = w.handleFailure(ctx, job, msg, fmt.Errorf("panic: %v", r))
			finalStatus = job.Status
		}
	}()

	defer func() {
		metrics.ObserveJobDuration(job.Type, string(finalStatus), time.Since(start))
	}()

	result, err := handler(ctx, msg.Payload)
	if err != nil {
		logger.Warn("job failed", zap.Error(err), zap.String("job_id", job.ID))
		metrics.JobFailuresTotal.WithLabelValues(job.Type).Inc()
		span.RecordError(err)
		_ = w.handleFailure(ctx, job, msg, err)
		finalStatus = job.Status
		return
	}

	job.Status = models.JobStatusSucceeded
	finished := time.Now()
	job.FinishedAt = &finished
	job.Result, _ = json.Marshal(result)
	_ = w.cfg.Storage.UpdateJob(ctx, job)
	_ = w.cfg.Storage.AppendJobLog(ctx, models.JobLog{JobID: job.ID, Level: "info", Message: "job succeeded"})
	_ = w.cfg.Broker.Ack(ctx, w.cfg.Stream, msg.ID)
	logger.Info("job completed", zap.String("job_id", job.ID), zap.Duration("duration", time.Since(start)))
}

func (w *Worker) handleFailure(ctx context.Context, job *models.Job, msg *broker.Message, err error) error {
	job.Attempts++
	job.Error = err.Error()
	msg.Attempts++
	if job.Attempts >= job.MaxAttempts {
		job.Status = models.JobStatusFailed
		finished := time.Now()
		job.FinishedAt = &finished
		_ = w.cfg.Storage.UpdateJob(ctx, job)
		_ = w.cfg.Storage.AppendJobLog(ctx, models.JobLog{JobID: job.ID, Level: "error", Message: err.Error()})
		_ = w.cfg.Broker.SendToDeadLetter(ctx, w.cfg.Stream, msg)
		return nil
	}

	job.Status = models.JobStatusRetrying
	_ = w.cfg.Storage.UpdateJob(ctx, job)
	_ = w.cfg.Storage.AppendJobLog(ctx, models.JobLog{JobID: job.ID, Level: "warn", Message: "job retry scheduled"})
	delay := calculateBackoff(job.Attempts)
	return w.cfg.Broker.Requeue(ctx, w.cfg.Stream, msg, delay)
}

func (w *Worker) incrementActive(delta int) {
	w.activeMu.Lock()
	defer w.activeMu.Unlock()
	w.activeCnt += delta
	if w.activeCnt < 0 {
		w.activeCnt = 0
	}
	_ = w.cfg.Storage.RecordHeartbeat(context.Background(), w.cfg.WorkerID, w.activeCnt)
	metrics.WorkerActiveGauge.WithLabelValues(w.cfg.WorkerID).Set(float64(w.activeCnt))
	metrics.WorkerHeartbeatGauge.WithLabelValues(w.cfg.WorkerID).Set(float64(time.Now().Unix()))
	metrics.WorkerUptimeGauge.WithLabelValues(w.cfg.WorkerID).Set(time.Since(w.startTime).Seconds())
}

func ptrTime(t time.Time) *time.Time {
	return &t
}

// calculateBackoff implements exponential backoff.
func calculateBackoff(attempts int) time.Duration {
	if attempts <= 0 {
		return 100 * time.Millisecond
	}
	delay := time.Duration(1<<uint(attempts)) * 100 * time.Millisecond
	if delay > 30*time.Second {
		delay = 30 * time.Second
	}
	jitter := time.Duration(float64(delay) * 0.2)
	return delay - jitter + time.Duration(float64(jitter)*0.5)
}

