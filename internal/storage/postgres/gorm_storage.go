package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/lib/pq"
	"go.uber.org/zap"
	"gorm.io/datatypes"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"taskqueue/internal/models"
	"taskqueue/internal/storage"
)

// GormStore implements the storage.Storage interface using GORM.
type GormStore struct {
	db     *gorm.DB
	logger *zap.Logger
}

// Config holds PostgreSQL configuration for GORM.
type Config struct {
	Host            string
	Port            int
	User            string
	Password        string
	Database        string
	DSN             string
	MaxConnections  int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	Logger          *zap.Logger
}

// NewGormStoreFromDSN creates a GORM store from a DSN string.
func NewGormStoreFromDSN(dsn string, logger *zap.Logger) (*GormStore, error) {
	cfg := &Config{
		DSN:    dsn,
		Logger: logger,
	}
	return NewGormStore(cfg)
}

// NewGormStore creates a new GORM-based PostgreSQL storage instance.
func NewGormStore(cfg *Config) (*GormStore, error) {
	var dsn string
	if cfg.DSN != "" {
		dsn = cfg.DSN
	} else {
		dsn = fmt.Sprintf(
			"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
			cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Database,
		)
	}

	// Configure GORM logger
	gormLogger := logger.Default.LogMode(logger.Silent)
	if cfg.Logger != nil {
		gormLogger = logger.Default.LogMode(logger.Info)
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger:                 gormLogger,
		SkipDefaultTransaction: true,
		PrepareStmt:            true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Get underlying sql.DB for connection pool configuration
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database instance: %w", err)
	}

	// Configure connection pool - 增加连接池大小以支持更高并发
	if cfg.MaxConnections > 0 {
		sqlDB.SetMaxOpenConns(cfg.MaxConnections)
	} else {
		sqlDB.SetMaxOpenConns(200)  // 从 100 增加到 200
	}

	if cfg.MaxIdleConns > 0 {
		sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	} else {
		sqlDB.SetMaxIdleConns(50)  // 从 10 增加到 50
	}

	if cfg.ConnMaxLifetime > 0 {
		sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	} else {
		sqlDB.SetConnMaxLifetime(time.Hour)
	}
	
	// 新增：设置连接空闲超时
	sqlDB.SetConnMaxIdleTime(10 * time.Minute)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := sqlDB.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	store := &GormStore{
		db:     db,
		logger: cfg.Logger,
	}

	// Auto-migrate schema
	if err := store.migrate(); err != nil {
		return nil, fmt.Errorf("failed to migrate schema: %w", err)
	}

	if cfg.Logger != nil {
		cfg.Logger.Info("PostgreSQL storage initialized with GORM",
			zap.String("host", cfg.Host),
			zap.Int("port", cfg.Port),
			zap.String("database", cfg.Database),
		)
	}

	return store, nil
}

// GORM Models

type gormJob struct {
	ID               string         `gorm:"type:text;primaryKey"`
	Type             string         `gorm:"type:varchar(100);not null;index:idx_jobs_type"`
	Payload          []byte         `gorm:"type:jsonb;not null"`
	Status           string         `gorm:"type:varchar(20);not null;default:'QUEUED';index:idx_jobs_status"`
	Priority         int            `gorm:"default:0;index:idx_jobs_priority"`
	Attempts         int            `gorm:"default:0"`
	MaxAttempts      int            `gorm:"default:3"`
	TimeoutSeconds   int            `gorm:"default:300"`
	CreatedAt        time.Time      `gorm:"not null;default:now();index:idx_jobs_created_at"`
	UpdatedAt        time.Time      `gorm:"not null;default:now()"`
	StartedAt        *time.Time     `gorm:"index:idx_jobs_started_at"`
	FinishedAt       *time.Time
	ScheduledFor     *time.Time     `gorm:"index:idx_jobs_scheduled_for"`
	Result           []byte         `gorm:"type:jsonb"`
	Error            *string        `gorm:"type:text"`
	WorkerID         *string        `gorm:"type:varchar(100);index:idx_jobs_worker_id"`
	ProcessingTimeMS *int64
	QueueTimeMS      *int64
	IdempotencyKey   *string        `gorm:"type:varchar(255);uniqueIndex:idx_jobs_idempotency_key"`
	Metadata         datatypes.JSON `gorm:"type:jsonb;default:'{}'"`
	Tags             pq.StringArray `gorm:"type:text[]"`
}

func (gormJob) TableName() string {
	return "jobs"
}

type gormJobLog struct {
	ID        int64          `gorm:"primaryKey;autoIncrement"`
	JobID     string         `gorm:"type:text;not null;index:idx_job_logs_job_id"`
	Level     string         `gorm:"type:varchar(20);not null;default:'info'"`
	Message   string         `gorm:"type:text;not null"`
	Timestamp time.Time      `gorm:"not null;default:now();index:idx_job_logs_timestamp"`
	TraceID   *string         `gorm:"type:varchar(100);index:idx_job_logs_trace_id"`
	SpanID    *string         `gorm:"type:varchar(100)"`
	WorkerID  *string         `gorm:"type:varchar(100)"`
	Metadata  datatypes.JSON  `gorm:"type:jsonb;default:'{}'"`
}

func (gormJobLog) TableName() string {
	return "job_logs"
}

type gormWorker struct {
	ID             string         `gorm:"type:varchar(100);primaryKey"`
	Hostname       string         `gorm:"type:varchar(255);not null"`
	Status         string         `gorm:"type:varchar(20);not null;default:'active';index:idx_workers_status"`
	PoolSize       int            `gorm:"not null"`
	Queues         pq.StringArray `gorm:"type:text[];not null;default:'{default}'"`
	ActiveJobs      int            `gorm:"default:0"`
	TotalProcessed int64          `gorm:"default:0"`
	TotalFailed     int64          `gorm:"default:0"`
	StartedAt       time.Time      `gorm:"not null;default:now();index:idx_workers_started_at"`
	LastHeartbeat   time.Time      `gorm:"not null;default:now();index:idx_workers_last_heartbeat"`
	StoppedAt       *time.Time
	Version         *string        `gorm:"type:varchar(50)"`
	Metadata        datatypes.JSON `gorm:"type:jsonb;default:'{}'"`
}

func (gormWorker) TableName() string {
	return "workers"
}

func (s *GormStore) migrate() error {
	return s.db.AutoMigrate(&gormJob{}, &gormJobLog{}, &gormWorker{})
}

// Storage interface implementation

func (s *GormStore) CreateJob(ctx context.Context, job *models.Job) error {
	now := time.Now().UTC()
	if job.CreatedAt.IsZero() {
		job.CreatedAt = now
	}
	if job.UpdatedAt.IsZero() {
		job.UpdatedAt = now
	}
	if job.Status == "" {
		job.Status = models.JobStatusQueued
	}

	timeoutSeconds := job.TimeoutSeconds
	if timeoutSeconds == 0 {
		timeoutSeconds = 300
	}

	var idempotencyKey *string
	if job.IdempotencyKey != "" {
		idempotencyKey = &job.IdempotencyKey
	}

	var workerID *string
	if job.WorkerID != "" {
		workerID = &job.WorkerID
	}

	var errorMsg *string
	if job.Error != "" {
		errorMsg = &job.Error
	}

	dbJob := &gormJob{
		ID:             job.ID,
		Type:           job.Type,
		Payload:        job.Payload,
		Status:         string(job.Status),
		Priority:       job.Priority,
		MaxAttempts:    job.MaxAttempts,
		TimeoutSeconds: timeoutSeconds,
		Attempts:       job.Attempts,
		CreatedAt:      job.CreatedAt,
		UpdatedAt:      job.UpdatedAt,
		IdempotencyKey: idempotencyKey,
		ScheduledFor:   job.ScheduledFor,
		WorkerID:       workerID,
		Error:          errorMsg,
		Result:         job.Result,
		StartedAt:      job.StartedAt,
		FinishedAt:     job.FinishedAt,
	}

	result := s.db.WithContext(ctx).Create(dbJob)
	if result.Error != nil {
		return fmt.Errorf("failed to create job: %w", result.Error)
	}

	return nil
}

func (s *GormStore) UpdateJob(ctx context.Context, job *models.Job) error {
	job.UpdatedAt = time.Now().UTC()

	timeoutSeconds := job.TimeoutSeconds
	if timeoutSeconds == 0 {
		timeoutSeconds = 300
	}

	var idempotencyKey *string
	if job.IdempotencyKey != "" {
		idempotencyKey = &job.IdempotencyKey
	}

	var workerID *string
	if job.WorkerID != "" {
		workerID = &job.WorkerID
	}

	var errorMsg *string
	if job.Error != "" {
		errorMsg = &job.Error
	}

	updates := map[string]interface{}{
		"type":            job.Type,
		"payload":         job.Payload,
		"priority":        job.Priority,
		"status":          string(job.Status),
		"attempts":        job.Attempts,
		"max_attempts":    job.MaxAttempts,
		"result":          job.Result,
		"error":           errorMsg,
		"updated_at":      job.UpdatedAt,
		"started_at":      job.StartedAt,
		"finished_at":     job.FinishedAt,
		"worker_id":        workerID,
		"timeout_seconds":  timeoutSeconds,
		"idempotency_key":  idempotencyKey,
		"scheduled_for":    job.ScheduledFor,
	}

	result := s.db.WithContext(ctx).
		Model(&gormJob{}).
		Where("id = ?", job.ID).
		Updates(updates)

	if result.Error != nil {
		return fmt.Errorf("failed to update job: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return storage.ErrJobNotFound
	}

	return nil
}

func (s *GormStore) GetJob(ctx context.Context, id string) (*models.Job, error) {
	var job gormJob
	result := s.db.WithContext(ctx).Where("id = ?", id).First(&job)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, storage.ErrJobNotFound
		}
		return nil, fmt.Errorf("failed to get job: %w", result.Error)
	}

	return s.gormJobToModel(&job), nil
}

func (s *GormStore) GetJobByIdempotencyKey(ctx context.Context, key string) (*models.Job, error) {
	var job gormJob
	result := s.db.WithContext(ctx).Where("idempotency_key = ?", key).First(&job)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, storage.ErrJobNotFound
		}
		return nil, fmt.Errorf("failed to get job by idempotency key: %w", result.Error)
	}

	return s.gormJobToModel(&job), nil
}

func (s *GormStore) ListJobs(ctx context.Context, filter storage.ListFilter) ([]*models.Job, int, error) {
	query := s.db.WithContext(ctx).Model(&gormJob{})

	// Apply filters
	if filter.Status != "" {
		query = query.Where("status = ?", string(filter.Status))
	}
	if filter.Type != "" {
		query = query.Where("type = ?", filter.Type)
	}

	// Get total count
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count jobs: %w", err)
	}

	// Apply pagination and ordering
	limit := filter.Limit
	if limit == 0 {
		limit = 50
	}

	var jobs []gormJob
	result := query.
		Order("created_at DESC").
		Limit(limit).
		Offset(filter.Offset).
		Find(&jobs)

	if result.Error != nil {
		return nil, 0, fmt.Errorf("failed to list jobs: %w", result.Error)
	}

	// Convert to models
	models := make([]*models.Job, len(jobs))
	for i, job := range jobs {
		models[i] = s.gormJobToModel(&job)
	}

	return models, int(total), nil
}

func (s *GormStore) AppendJobLog(ctx context.Context, log models.JobLog) error {
	ts := log.Timestamp
	if ts.IsZero() {
		ts = time.Now().UTC()
	}

	dbLog := &gormJobLog{
		JobID:     log.JobID,
		Level:     log.Level,
		Message:   log.Message,
		Timestamp: ts,
	}

	result := s.db.WithContext(ctx).Create(dbLog)
	if result.Error != nil {
		return fmt.Errorf("failed to append job log: %w", result.Error)
	}

	return nil
}

func (s *GormStore) ListJobLogs(ctx context.Context, jobID string) ([]models.JobLog, error) {
	var logs []gormJobLog
	result := s.db.WithContext(ctx).
		Where("job_id = ?", jobID).
		Order("timestamp ASC").
		Find(&logs)

	if result.Error != nil {
		return nil, fmt.Errorf("failed to list job logs: %w", result.Error)
	}

	jobLogs := make([]models.JobLog, len(logs))
	for i, log := range logs {
		jobLogs[i] = models.JobLog{
			ID:        log.ID,
			JobID:     log.JobID,
			Level:     log.Level,
			Message:   log.Message,
			Timestamp: log.Timestamp,
		}
	}

	return jobLogs, nil
}

func (s *GormStore) RegisterWorker(ctx context.Context, info models.WorkerInfo) error {
	if info.StartedAt.IsZero() {
		info.StartedAt = time.Now().UTC()
	}
	if info.LastHeartbeat.IsZero() {
		info.LastHeartbeat = time.Now().UTC()
	}

	worker := &gormWorker{
		ID:             info.ID,
		Hostname:       info.Hostname,
		Status:         info.Status,
		PoolSize:       info.PoolSize,
		ActiveJobs:     info.ActiveJobs,
		TotalProcessed: int64(info.TotalJobs),
		StartedAt:      info.StartedAt,
		LastHeartbeat:  info.LastHeartbeat,
	}

	result := s.db.WithContext(ctx).
		Where("id = ?", info.ID).
		Assign(map[string]interface{}{
			"hostname":       info.Hostname,
			"pool_size":      info.PoolSize,
			"last_heartbeat": info.LastHeartbeat,
			"started_at":     info.StartedAt,
			"status":         info.Status,
		}).
		FirstOrCreate(worker)

	if result.Error != nil {
		return fmt.Errorf("failed to register worker: %w", result.Error)
	}

	return nil
}

func (s *GormStore) RecordHeartbeat(ctx context.Context, workerID string, activeJobs int) error {
	status := "idle"
	if activeJobs > 0 {
		status = "busy"
	}

	updates := map[string]interface{}{
		"active_jobs":    activeJobs,
		"last_heartbeat": time.Now().UTC(),
		"status":         status,
	}

	result := s.db.WithContext(ctx).
		Model(&gormWorker{}).
		Where("id = ?", workerID).
		Updates(updates)

	if result.Error != nil {
		return fmt.Errorf("failed to record heartbeat: %w", result.Error)
	}

	return nil
}

func (s *GormStore) ListWorkers(ctx context.Context) ([]models.WorkerInfo, error) {
	var workers []gormWorker
	result := s.db.WithContext(ctx).
		Order("started_at DESC").
		Find(&workers)

	if result.Error != nil {
		return nil, fmt.Errorf("failed to list workers: %w", result.Error)
	}

	workerInfos := make([]models.WorkerInfo, len(workers))
	for i, worker := range workers {
		workerInfos[i] = models.WorkerInfo{
			ID:            worker.ID,
			Hostname:      worker.Hostname,
			PoolSize:      worker.PoolSize,
			ActiveJobs:    worker.ActiveJobs,
			TotalJobs:     int(worker.TotalProcessed),
			LastHeartbeat: worker.LastHeartbeat,
			StartedAt:     worker.StartedAt,
			Status:        worker.Status,
		}
	}

	return workerInfos, nil
}

// Additional methods from guide

func (s *GormStore) UpdateJobStatus(ctx context.Context, jobID, status, workerID string, timestamp time.Time) error {
	updates := map[string]interface{}{
		"status":     status,
		"updated_at": time.Now(),
	}

	if status == "RUNNING" {
		updates["started_at"] = timestamp
		if workerID != "" {
			updates["worker_id"] = workerID
		}
	}

	result := s.db.WithContext(ctx).
		Model(&gormJob{}).
		Where("id = ?", jobID).
		Updates(updates)

	if result.Error != nil {
		return fmt.Errorf("failed to update job status: %w", result.Error)
	}

	return nil
}

func (s *GormStore) UpdateJobSuccess(ctx context.Context, jobID string, result []byte, finishedAt time.Time, duration time.Duration) error {
	processingTimeMS := duration.Milliseconds()

	updates := map[string]interface{}{
		"status":             "SUCCEEDED",
		"result":             result,
		"finished_at":        finishedAt,
		"processing_time_ms": processingTimeMS,
		"updated_at":          time.Now(),
	}

	dbResult := s.db.WithContext(ctx).
		Model(&gormJob{}).
		Where("id = ?", jobID).
		Updates(updates)

	if dbResult.Error != nil {
		return fmt.Errorf("failed to update job success: %w", dbResult.Error)
	}

	return nil
}

func (s *GormStore) UpdateJobFailed(ctx context.Context, jobID, errorMsg string, finishedAt time.Time) error {
	updates := map[string]interface{}{
		"status":      "FAILED",
		"error":       errorMsg,
		"finished_at": finishedAt,
		"updated_at":   time.Now(),
	}

	result := s.db.WithContext(ctx).
		Model(&gormJob{}).
		Where("id = ?", jobID).
		Updates(updates)

	if result.Error != nil {
		return fmt.Errorf("failed to update job failure: %w", result.Error)
	}

	return nil
}

func (s *GormStore) ResetJobForRetry(ctx context.Context, jobID string) error {
	updates := map[string]interface{}{
		"status":     "QUEUED",
		"attempts":   0,
		"error":      nil,
		"started_at": nil,
		"worker_id":  nil,
		"updated_at":  time.Now(),
	}

	result := s.db.WithContext(ctx).
		Model(&gormJob{}).
		Where("id = ?", jobID).
		Updates(updates)

	if result.Error != nil {
		return fmt.Errorf("failed to reset job for retry: %w", result.Error)
	}

	return nil
}

func (s *GormStore) GetActiveWorkerCount(ctx context.Context) (int, error) {
	var count int64
	result := s.db.WithContext(ctx).
		Model(&gormWorker{}).
		Where("status = ? AND last_heartbeat > ?", "active", time.Now().Add(-30*time.Second)).
		Count(&count)

	if result.Error != nil {
		return 0, fmt.Errorf("failed to count active workers: %w", result.Error)
	}

	return int(count), nil
}

func (s *GormStore) Close() error {
	sqlDB, err := s.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// DB returns the underlying *sql.DB for metrics collection
func (s *GormStore) DB() (*sql.DB, error) {
	return s.db.DB()
}

// Helper functions

func (s *GormStore) gormJobToModel(job *gormJob) *models.Job {
	model := &models.Job{
		ID:             job.ID,
		Type:           job.Type,
		Payload:        job.Payload,
		Status:         models.JobStatus(job.Status),
		Priority:       job.Priority,
		Attempts:       job.Attempts,
		MaxAttempts:    job.MaxAttempts,
		TimeoutSeconds: job.TimeoutSeconds,
		CreatedAt:      job.CreatedAt,
		UpdatedAt:      job.UpdatedAt,
		StartedAt:      job.StartedAt,
		FinishedAt:     job.FinishedAt,
		Result:         job.Result,
		WorkerID:       "",
		ScheduledFor:   job.ScheduledFor,
	}

	if job.WorkerID != nil {
		model.WorkerID = *job.WorkerID
	}
	if job.Error != nil {
		model.Error = *job.Error
	}
	if job.IdempotencyKey != nil {
		model.IdempotencyKey = *job.IdempotencyKey
	}

	return model
}

