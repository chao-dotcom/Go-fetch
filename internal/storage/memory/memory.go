package memory

import (
	"context"
	"slices"
	"sync"
	"time"

	"taskqueue/internal/models"
	"taskqueue/internal/storage"
)

// Store is an in-memory storage implementation useful for development and tests.
type Store struct {
	mu       sync.RWMutex
	jobs     map[string]*models.Job
	jobLogs  map[string][]models.JobLog
	workers  map[string]models.WorkerInfo
	totalJob int
}

// New creates a new Store.
func New() *Store {
	return &Store{
		jobs:    make(map[string]*models.Job),
		jobLogs: make(map[string][]models.JobLog),
		workers: make(map[string]models.WorkerInfo),
	}
}

// CreateJob inserts a job.
func (s *Store) CreateJob(ctx context.Context, job *models.Job) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	job.CreatedAt = now
	job.UpdatedAt = now
	if job.Status == "" {
		job.Status = models.JobStatusQueued
	}
	s.jobs[job.ID] = cloneJob(job)
	s.totalJob++
	return nil
}

// UpdateJob replaces a job record.
func (s *Store) UpdateJob(ctx context.Context, job *models.Job) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.jobs[job.ID]; !ok {
		return storage.ErrJobNotFound
	}
	job.UpdatedAt = time.Now().UTC()
	s.jobs[job.ID] = cloneJob(job)
	return nil
}

// GetJob retrieves a job by ID.
func (s *Store) GetJob(ctx context.Context, id string) (*models.Job, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	job, ok := s.jobs[id]
	if !ok {
		return nil, storage.ErrJobNotFound
	}
	return cloneJob(job), nil
}

// GetJobByIdempotencyKey retrieves a job by its idempotency key.
func (s *Store) GetJobByIdempotencyKey(ctx context.Context, key string) (*models.Job, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, job := range s.jobs {
		if job.IdempotencyKey == key {
			return cloneJob(job), nil
		}
	}
	return nil, storage.ErrJobNotFound
}

// ListJobs returns filtered jobs.
func (s *Store) ListJobs(ctx context.Context, filter storage.ListFilter) ([]*models.Job, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var jobs []*models.Job
	for _, job := range s.jobs {
		if filter.Status != "" && job.Status != filter.Status {
			continue
		}
		if filter.Type != "" && job.Type != filter.Type {
			continue
		}
		jobs = append(jobs, cloneJob(job))
	}

	total := len(jobs)
	slices.SortFunc(jobs, func(a, b *models.Job) int {
		if a.CreatedAt.Equal(b.CreatedAt) {
			return 0
		}
		if a.CreatedAt.After(b.CreatedAt) {
			return -1
		}
		return 1
	})

	start := filter.Offset
	if start > total {
		start = total
	}
	limit := filter.Limit
	if limit <= 0 || limit > total-start {
		limit = total - start
	}
	return jobs[start : start+limit], total, nil
}

// AppendJobLog adds a record to job logs.
func (s *Store) AppendJobLog(ctx context.Context, log models.JobLog) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	log.Timestamp = time.Now().UTC()
	s.jobLogs[log.JobID] = append(s.jobLogs[log.JobID], log)
	return nil
}

// ListJobLogs returns logs for a job.
func (s *Store) ListJobLogs(ctx context.Context, jobID string) ([]models.JobLog, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	logs := s.jobLogs[jobID]
	result := make([]models.JobLog, len(logs))
	copy(result, logs)
	return result, nil
}

// RegisterWorker stores worker metadata.
func (s *Store) RegisterWorker(ctx context.Context, info models.WorkerInfo) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if info.StartedAt.IsZero() {
		info.StartedAt = time.Now().UTC()
	}
	info.LastHeartbeat = time.Now().UTC()
	info.Status = "active"
	s.workers[info.ID] = info
	return nil
}

// RecordHeartbeat updates heartbeat info.
func (s *Store) RecordHeartbeat(ctx context.Context, workerID string, activeJobs int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	info, ok := s.workers[workerID]
	if !ok {
		info = models.WorkerInfo{ID: workerID, StartedAt: time.Now().UTC()}
	}
	info.LastHeartbeat = time.Now().UTC()
	info.ActiveJobs = activeJobs
	if activeJobs > 0 {
		info.Status = "busy"
	} else {
		info.Status = "idle"
	}
	s.workers[workerID] = info
	return nil
}

// ListWorkers returns worker state.
func (s *Store) ListWorkers(ctx context.Context) ([]models.WorkerInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]models.WorkerInfo, 0, len(s.workers))
	for _, info := range s.workers {
		result = append(result, info)
	}
	return result, nil
}

func cloneJob(job *models.Job) *models.Job {
	if job == nil {
		return nil
	}
	cp := *job
	if job.Payload != nil {
		cp.Payload = append([]byte(nil), job.Payload...)
	}
	if job.Result != nil {
		cp.Result = append([]byte(nil), job.Result...)
	}
	return &cp
}

