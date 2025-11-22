package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"taskqueue/internal/models"
	"taskqueue/internal/storage"
)

// Store implements the storage.Storage interface backed by PostgreSQL.
type Store struct {
	pool *pgxpool.Pool
}

// New creates a PostgreSQL-backed store and ensures the schema exists.
// If USE_GORM environment variable is set to "true", it uses GORM-based storage.
// logger is optional - if nil, a default logger will be used for GORM.
func New(ctx context.Context, dsn string, logger *zap.Logger) (storage.Storage, error) {
	if dsn == "" {
		return nil, fmt.Errorf("postgres: DSN cannot be empty")
	}

	// Check if GORM should be used
	if os.Getenv("USE_GORM") == "true" {
		if logger == nil {
			logger, _ = zap.NewDevelopment()
		}
		return NewGormStoreFromDSN(dsn, logger)
	}

	// Use pgxpool implementation (legacy)
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("postgres: parse config: %w", err)
	}
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("postgres: connect: %w", err)
	}

	store := &Store{pool: pool}
	if err := store.ensureSchema(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return store, nil
}

// Close releases the underlying pool resources.
func (s *Store) Close() {
	if s.pool != nil {
		s.pool.Close()
	}
}

func (s *Store) ensureSchema(ctx context.Context) error {
	// Execute each statement separately to avoid issues with duplicate constraints
	statements := []string{
		`CREATE TABLE IF NOT EXISTS jobs (
			id TEXT PRIMARY KEY,
			type TEXT NOT NULL,
			payload JSONB NOT NULL,
			priority INT NOT NULL DEFAULT 0,
			status TEXT NOT NULL,
			attempts INT NOT NULL DEFAULT 0,
			max_attempts INT NOT NULL DEFAULT 3,
			result JSONB,
			error TEXT,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			started_at TIMESTAMPTZ,
			finished_at TIMESTAMPTZ,
			worker_id TEXT,
			timeout_seconds INT NOT NULL DEFAULT 300,
			idempotency_key TEXT,
			scheduled_for TIMESTAMPTZ
		)`,
		`CREATE INDEX IF NOT EXISTS idx_jobs_status ON jobs(status)`,
		`CREATE INDEX IF NOT EXISTS idx_jobs_type ON jobs(type)`,
		`CREATE INDEX IF NOT EXISTS idx_jobs_idempotency_key ON jobs(idempotency_key) WHERE idempotency_key IS NOT NULL`,
		`CREATE TABLE IF NOT EXISTS job_logs (
			id BIGSERIAL PRIMARY KEY,
			job_id TEXT NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
			level TEXT NOT NULL,
			message TEXT NOT NULL,
			timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS workers (
			id TEXT PRIMARY KEY,
			hostname TEXT,
			pool_size INT,
			active_jobs INT NOT NULL DEFAULT 0,
			total_jobs INT NOT NULL DEFAULT 0,
			last_heartbeat TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			status TEXT NOT NULL DEFAULT 'idle'
		)`,
	}

	for _, stmt := range statements {
		if _, err := s.pool.Exec(ctx, stmt); err != nil {
			// Ignore errors about duplicate keys/indexes/types - these are expected if schema already exists
			errStr := err.Error()
			if contains(errStr, "duplicate key") || contains(errStr, "already exists") || contains(errStr, "pg_type_typname_nsp_index") {
				continue
			}
			return fmt.Errorf("postgres: ensure schema: %w", err)
		}
	}
	return nil
}

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

// CreateJob stores the job definition.
func (s *Store) CreateJob(ctx context.Context, job *models.Job) error {
	now := time.Now().UTC()
	job.CreatedAt = now
	job.UpdatedAt = now
	if job.Status == "" {
		job.Status = models.JobStatusQueued
	}

	query := `INSERT INTO jobs (
		id, type, payload, priority, status, attempts, max_attempts, result, error,
		created_at, updated_at, started_at, finished_at, worker_id, timeout_seconds, idempotency_key, scheduled_for
	) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)`

	timeoutSeconds := job.TimeoutSeconds
	if timeoutSeconds == 0 {
		timeoutSeconds = 300
	}

	_, err := s.pool.Exec(ctx, query,
		job.ID, job.Type, job.Payload, job.Priority, job.Status, job.Attempts, job.MaxAttempts,
		job.Result, job.Error, job.CreatedAt, job.UpdatedAt, job.StartedAt, job.FinishedAt, job.WorkerID,
		timeoutSeconds, job.IdempotencyKey, job.ScheduledFor,
	)
	return err
}

// UpdateJob updates a job record.
func (s *Store) UpdateJob(ctx context.Context, job *models.Job) error {
	job.UpdatedAt = time.Now().UTC()
	query := `UPDATE jobs
		SET type=$2, payload=$3, priority=$4, status=$5, attempts=$6, max_attempts=$7,
			result=$8, error=$9, updated_at=$10, started_at=$11, finished_at=$12, worker_id=$13,
			timeout_seconds=$14, idempotency_key=$15, scheduled_for=$16
		WHERE id=$1`
	timeoutSeconds := job.TimeoutSeconds
	if timeoutSeconds == 0 {
		timeoutSeconds = 300
	}
	cmd, err := s.pool.Exec(ctx, query,
		job.ID, job.Type, job.Payload, job.Priority, job.Status, job.Attempts, job.MaxAttempts,
		job.Result, job.Error, job.UpdatedAt, job.StartedAt, job.FinishedAt, job.WorkerID,
		timeoutSeconds, job.IdempotencyKey, job.ScheduledFor,
	)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return storage.ErrJobNotFound
	}
	return nil
}

// GetJob fetches a single job.
func (s *Store) GetJob(ctx context.Context, id string) (*models.Job, error) {
	query := `SELECT id, type, payload, priority, status, attempts, max_attempts,
		result, error, created_at, updated_at, started_at, finished_at, worker_id,
		timeout_seconds, idempotency_key, scheduled_for
		FROM jobs WHERE id=$1`
	row := s.pool.QueryRow(ctx, query, id)
	job, err := scanJob(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, storage.ErrJobNotFound
		}
		return nil, err
	}
	return job, nil
}

// GetJobByIdempotencyKey fetches a job by its idempotency key.
func (s *Store) GetJobByIdempotencyKey(ctx context.Context, key string) (*models.Job, error) {
	query := `SELECT id, type, payload, priority, status, attempts, max_attempts,
		result, error, created_at, updated_at, started_at, finished_at, worker_id,
		timeout_seconds, idempotency_key, scheduled_for
		FROM jobs WHERE idempotency_key=$1 LIMIT 1`
	row := s.pool.QueryRow(ctx, query, key)
	job, err := scanJob(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, storage.ErrJobNotFound
		}
		return nil, err
	}
	return job, nil
}

// ListJobs returns jobs filtered by status/type with pagination.
func (s *Store) ListJobs(ctx context.Context, filter storage.ListFilter) ([]*models.Job, int, error) {
	where := ""
	args := make([]any, 0, 4)
	argPos := 1

	if filter.Status != "" {
		where += fmt.Sprintf("status = $%d", argPos)
		args = append(args, filter.Status)
		argPos++
	}
	if filter.Type != "" {
		if where != "" {
			where += " AND "
		}
		where += fmt.Sprintf("type = $%d", argPos)
		args = append(args, filter.Type)
		argPos++
	}

	countQuery := "SELECT COUNT(*) FROM jobs"
	if where != "" {
		countQuery += " WHERE " + where
	}
	var total int
	if err := s.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := `SELECT id, type, payload, priority, status, attempts, max_attempts,
		result, error, created_at, updated_at, started_at, finished_at, worker_id,
		timeout_seconds, idempotency_key, scheduled_for
		FROM jobs`
	if where != "" {
		query += " WHERE " + where
	}
	query += " ORDER BY created_at DESC"

	limit := filter.Limit
	if limit == 0 {
		limit = 50
	}
	offset := filter.Offset

	query = fmt.Sprintf("%s LIMIT %d OFFSET %d", query, limit, offset)

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var jobs []*models.Job
	for rows.Next() {
		job, err := scanJob(rows)
		if err != nil {
			return nil, 0, err
		}
		jobs = append(jobs, job)
	}
	return jobs, total, rows.Err()
}

// AppendJobLog inserts a job log record.
func (s *Store) AppendJobLog(ctx context.Context, log models.JobLog) error {
	ts := log.Timestamp
	if ts.IsZero() {
		ts = time.Now().UTC()
	}
	query := `INSERT INTO job_logs (job_id, level, message, timestamp) VALUES ($1,$2,$3,$4)`
	_, err := s.pool.Exec(ctx, query, log.JobID, log.Level, log.Message, ts)
	return err
}

// ListJobLogs returns logs for a job ordered by time.
func (s *Store) ListJobLogs(ctx context.Context, jobID string) ([]models.JobLog, error) {
	query := `SELECT id, job_id, level, message, timestamp FROM job_logs WHERE job_id=$1 ORDER BY timestamp ASC`
	rows, err := s.pool.Query(ctx, query, jobID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []models.JobLog
	for rows.Next() {
		var log models.JobLog
		if err := rows.Scan(&log.ID, &log.JobID, &log.Level, &log.Message, &log.Timestamp); err != nil {
			return nil, err
		}
		logs = append(logs, log)
	}
	return logs, rows.Err()
}

// RegisterWorker inserts or updates a worker record.
func (s *Store) RegisterWorker(ctx context.Context, info models.WorkerInfo) error {
	if info.StartedAt.IsZero() {
		info.StartedAt = time.Now().UTC()
	}
	if info.LastHeartbeat.IsZero() {
		info.LastHeartbeat = time.Now().UTC()
	}
	query := `INSERT INTO workers (id, hostname, pool_size, active_jobs, total_jobs, last_heartbeat, started_at, status)
			  VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
			  ON CONFLICT (id) DO UPDATE SET
				hostname=EXCLUDED.hostname,
				pool_size=EXCLUDED.pool_size,
				last_heartbeat=EXCLUDED.last_heartbeat,
				started_at=EXCLUDED.started_at,
				status=EXCLUDED.status`
	_, err := s.pool.Exec(ctx, query,
		info.ID, info.Hostname, info.PoolSize, info.ActiveJobs, info.TotalJobs,
		info.LastHeartbeat, info.StartedAt, info.Status,
	)
	return err
}

// RecordHeartbeat updates worker heartbeat/status.
func (s *Store) RecordHeartbeat(ctx context.Context, workerID string, activeJobs int) error {
	status := "idle"
	if activeJobs > 0 {
		status = "busy"
	}
	query := `UPDATE workers SET active_jobs=$2, last_heartbeat=$3, status=$4 WHERE id=$1`
	_, err := s.pool.Exec(ctx, query, workerID, activeJobs, time.Now().UTC(), status)
	return err
}

// ListWorkers returns worker states.
func (s *Store) ListWorkers(ctx context.Context) ([]models.WorkerInfo, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, hostname, pool_size, active_jobs, total_jobs, last_heartbeat, started_at, status FROM workers`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var workers []models.WorkerInfo
	for rows.Next() {
		var info models.WorkerInfo
		if err := rows.Scan(&info.ID, &info.Hostname, &info.PoolSize, &info.ActiveJobs, &info.TotalJobs,
			&info.LastHeartbeat, &info.StartedAt, &info.Status); err != nil {
			return nil, err
		}
		workers = append(workers, info)
	}
	return workers, rows.Err()
}

func scanJob(row interface {
	Scan(dest ...any) error
}) (*models.Job, error) {
	var job models.Job
	var startedAt, finishedAt, scheduledFor sql.NullTime
	var workerID, idempotencyKey sql.NullString
	var timeoutSeconds sql.NullInt32
	if err := row.Scan(
		&job.ID,
		&job.Type,
		&job.Payload,
		&job.Priority,
		&job.Status,
		&job.Attempts,
		&job.MaxAttempts,
		&job.Result,
		&job.Error,
		&job.CreatedAt,
		&job.UpdatedAt,
		&startedAt,
		&finishedAt,
		&workerID,
		&timeoutSeconds,
		&idempotencyKey,
		&scheduledFor,
	); err != nil {
		return nil, err
	}
	if startedAt.Valid {
		st := startedAt.Time
		job.StartedAt = &st
	}
	if finishedAt.Valid {
		ft := finishedAt.Time
		job.FinishedAt = &ft
	}
	if workerID.Valid {
		job.WorkerID = workerID.String
	}
	if timeoutSeconds.Valid {
		job.TimeoutSeconds = int(timeoutSeconds.Int32)
	}
	if idempotencyKey.Valid {
		job.IdempotencyKey = idempotencyKey.String
	}
	if scheduledFor.Valid {
		sf := scheduledFor.Time
		job.ScheduledFor = &sf
	}
	return &job, nil
}


