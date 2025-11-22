package storage

import (
	"context"
	"errors"

	"taskqueue/internal/models"
)

var (
	// ErrJobNotFound indicates the job does not exist.
	ErrJobNotFound = errors.New("job not found")
)

// ListFilter allows filtering job lists.
type ListFilter struct {
	Status models.JobStatus
	Type   string
	Limit  int
	Offset int
}

// Storage defines persistence requirements.
type Storage interface {
	CreateJob(ctx context.Context, job *models.Job) error
	UpdateJob(ctx context.Context, job *models.Job) error
	GetJob(ctx context.Context, id string) (*models.Job, error)
	ListJobs(ctx context.Context, filter ListFilter) ([]*models.Job, int, error)

	AppendJobLog(ctx context.Context, log models.JobLog) error
	ListJobLogs(ctx context.Context, jobID string) ([]models.JobLog, error)

	RegisterWorker(ctx context.Context, info models.WorkerInfo) error
	RecordHeartbeat(ctx context.Context, workerID string, activeJobs int) error
	ListWorkers(ctx context.Context) ([]models.WorkerInfo, error)
}

