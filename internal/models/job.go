package models

import (
	"encoding/json"
	"time"
)

// JobStatus represents lifecycle state.
type JobStatus string

const (
	JobStatusQueued    JobStatus = "QUEUED"
	JobStatusRunning   JobStatus = "RUNNING"
	JobStatusSucceeded JobStatus = "SUCCEEDED"
	JobStatusFailed    JobStatus = "FAILED"
	JobStatusRetrying  JobStatus = "RETRYING"
)

// Job describes a persisted job record.
type Job struct {
	ID             string          `json:"id"`
	Type           string          `json:"type"`
	Payload        json.RawMessage `json:"payload"`
	Priority       int             `json:"priority"`
	Status         JobStatus       `json:"status"`
	Attempts       int             `json:"attempts"`
	MaxAttempts    int             `json:"max_attempts"`
	Result         json.RawMessage `json:"result,omitempty"`
	Error          string          `json:"error,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
	StartedAt      *time.Time      `json:"started_at,omitempty"`
	FinishedAt     *time.Time      `json:"finished_at,omitempty"`
	WorkerID       string          `json:"worker_id,omitempty"`
	TimeoutSeconds int             `json:"timeout_seconds,omitempty"`
	IdempotencyKey string          `json:"idempotency_key,omitempty"`
	ScheduledFor   *time.Time      `json:"scheduled_for,omitempty"`
}

// JobLog captures per job events.
type JobLog struct {
	ID        int64     `json:"id"`
	JobID     string    `json:"job_id"`
	Level     string    `json:"level"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}

// WorkerInfo stores worker metadata and heartbeat fields.
type WorkerInfo struct {
	ID           string    `json:"id"`
	Hostname     string    `json:"hostname"`
	PoolSize     int       `json:"pool_size"`
	ActiveJobs   int       `json:"active_jobs"`
	TotalJobs    int       `json:"total_jobs"`
	LastHeartbeat time.Time `json:"last_heartbeat"`
	StartedAt    time.Time `json:"started_at"`
	Status       string    `json:"status"`
}

