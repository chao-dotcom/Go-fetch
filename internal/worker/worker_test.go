package worker

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap/zaptest"

	"taskqueue/internal/broker"
	"taskqueue/internal/jobs"
	"taskqueue/internal/models"
	"taskqueue/internal/storage"
)

// ----------------------------------------------------------------------
// Mocks
// ----------------------------------------------------------------------

type mockBroker struct{ mock.Mock }

func (m *mockBroker) Enqueue(ctx context.Context, msg *broker.Message) error {
	args := m.Called(ctx, msg)
	return args.Error(0)
}

func (m *mockBroker) Consume(ctx context.Context, stream string, consumerID string, count int, block time.Duration) ([]*broker.Message, error) {
	args := m.Called(ctx, stream, consumerID, count, block)
	if arr, ok := args.Get(0).([]*broker.Message); ok {
		return arr, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockBroker) Ack(ctx context.Context, stream, messageID string) error {
	return m.Called(ctx, stream, messageID).Error(0)
}

func (m *mockBroker) Requeue(ctx context.Context, stream string, msg *broker.Message, delay time.Duration) error {
	return m.Called(ctx, stream, msg, delay).Error(0)
}

func (m *mockBroker) SendToDeadLetter(ctx context.Context, stream string, msg *broker.Message) error {
	return m.Called(ctx, stream, msg).Error(0)
}

type mockStorage struct{ mock.Mock }

func (m *mockStorage) CreateJob(context.Context, *models.Job) error { return nil }
func (m *mockStorage) AppendJobLog(ctx context.Context, log models.JobLog) error {
	return m.Called(ctx, log).Error(0)
}
func (m *mockStorage) ListJobLogs(context.Context, string) ([]models.JobLog, error) { return nil, nil }
func (m *mockStorage) ListJobs(context.Context, storage.ListFilter) ([]*models.Job, int, error) {
	return nil, 0, nil
}
func (m *mockStorage) ListWorkers(context.Context) ([]models.WorkerInfo, error) { return nil, nil }
func (m *mockStorage) RegisterWorker(context.Context, models.WorkerInfo) error  { return nil }
func (m *mockStorage) RecordHeartbeat(context.Context, string, int) error       { return nil }
func (m *mockStorage) UpdateJob(ctx context.Context, job *models.Job) error {
	return m.Called(ctx, job).Error(0)
}
func (m *mockStorage) GetJob(ctx context.Context, id string) (*models.Job, error) {
	args := m.Called(ctx, id)
	if job, ok := args.Get(0).(*models.Job); ok {
		return job, args.Error(1)
	}
	return nil, args.Error(1)
}

// ----------------------------------------------------------------------
// Tests
// ----------------------------------------------------------------------

func TestWorkerExecuteJobSuccess(t *testing.T) {
	logger := zaptest.NewLogger(t)
	b := new(mockBroker)
	s := new(mockStorage)
	reg := jobs.NewRegistry()
	reg.Register("test_job", func(ctx context.Context, payload json.RawMessage) (any, error) {
		return map[string]string{"status": "ok"}, nil
	})

	jobModel := &models.Job{ID: "job-123", Type: "test_job", MaxAttempts: 3}
	message := &broker.Message{ID: "msg-1", JobID: "job-123", Type: "test_job", Payload: json.RawMessage(`{}`)}

	s.On("GetJob", mock.Anything, "job-123").Return(jobModel, nil)
	s.On("UpdateJob", mock.Anything, mock.AnythingOfType("*models.Job")).Return(nil).Times(2)
	s.On("AppendJobLog", mock.Anything, mock.AnythingOfType("models.JobLog")).Return(nil).Times(2)
	b.On("Ack", mock.Anything, "job_stream", "msg-1").Return(nil)

	w, err := New(Config{
		WorkerID: "worker-1",
		Stream:   "job_stream",
		Broker:   b,
		Storage:  s,
		Registry: reg,
		Logger:   logger,
	})
	assert.NoError(t, err)

	w.executeJob(context.Background(), logger, message)

	s.AssertExpectations(t)
	b.AssertExpectations(t)
}

func TestWorkerExecuteJobFailureRetries(t *testing.T) {
	logger := zaptest.NewLogger(t)
	b := new(mockBroker)
	s := new(mockStorage)
	reg := jobs.NewRegistry()
	reg.Register("fail_job", func(ctx context.Context, payload json.RawMessage) (any, error) {
		return nil, errors.New("boom")
	})

	jobModel := &models.Job{ID: "job-456", Type: "fail_job", MaxAttempts: 3}
	message := &broker.Message{ID: "msg-2", JobID: "job-456", Type: "fail_job", Payload: json.RawMessage(`{}`)}

	s.On("GetJob", mock.Anything, "job-456").Return(jobModel, nil)
	s.On("UpdateJob", mock.Anything, mock.AnythingOfType("*models.Job")).Return(nil).Times(2)
	s.On("AppendJobLog", mock.Anything, mock.AnythingOfType("models.JobLog")).Return(nil).Times(2)
	b.On("Requeue", mock.Anything, "job_stream", message, mock.AnythingOfType("time.Duration")).Return(nil)
	b.On("Ack", mock.Anything, "job_stream", "msg-2").Return(nil)

	w, err := New(Config{
		WorkerID: "worker-2",
		Stream:   "job_stream",
		Broker:   b,
		Storage:  s,
		Registry: reg,
		Logger:   logger,
	})
	assert.NoError(t, err)

	w.executeJob(context.Background(), logger, message)

	s.AssertExpectations(t)
	b.AssertExpectations(t)
	assert.Equal(t, 1, jobModel.Attempts)
}

func TestWorkerExecuteJobPermanentFailure(t *testing.T) {
	logger := zaptest.NewLogger(t)
	b := new(mockBroker)
	s := new(mockStorage)
	reg := jobs.NewRegistry()
	reg.Register("permanent_fail", func(ctx context.Context, payload json.RawMessage) (any, error) {
		return nil, errors.New("fatal")
	})

	jobModel := &models.Job{ID: "job-789", Type: "permanent_fail", Attempts: 2, MaxAttempts: 3}
	message := &broker.Message{ID: "msg-3", JobID: "job-789", Type: "permanent_fail", Payload: json.RawMessage(`{}`)}

	s.On("GetJob", mock.Anything, "job-789").Return(jobModel, nil)
	s.On("UpdateJob", mock.Anything, mock.AnythingOfType("*models.Job")).Return(nil)
	s.On("AppendJobLog", mock.Anything, mock.AnythingOfType("models.JobLog")).Return(nil).Times(2)
	b.On("SendToDeadLetter", mock.Anything, "job_stream", message).Return(nil)
	b.On("Ack", mock.Anything, "job_stream", "msg-3").Return(nil)

	w, err := New(Config{
		WorkerID: "worker-3",
		Stream:   "job_stream",
		Broker:   b,
		Storage:  s,
		Registry: reg,
		Logger:   logger,
	})
	assert.NoError(t, err)

	w.executeJob(context.Background(), logger, message)

	s.AssertExpectations(t)
	b.AssertExpectations(t)
	assert.Equal(t, models.JobStatusFailed, jobModel.Status)
}

func TestWorkerPanicRecovery(t *testing.T) {
	logger := zaptest.NewLogger(t)
	b := new(mockBroker)
	s := new(mockStorage)
	reg := jobs.NewRegistry()
	reg.Register("panic_job", func(ctx context.Context, payload json.RawMessage) (any, error) {
		panic("boom")
	})

	jobModel := &models.Job{ID: "job-panic", Type: "panic_job", MaxAttempts: 3}
	message := &broker.Message{ID: "msg-panic", JobID: "job-panic", Type: "panic_job", Payload: json.RawMessage(`{}`)}

	s.On("GetJob", mock.Anything, "job-panic").Return(jobModel, nil)
	s.On("UpdateJob", mock.Anything, mock.AnythingOfType("*models.Job")).Return(nil).Times(2)
	s.On("AppendJobLog", mock.Anything, mock.AnythingOfType("models.JobLog")).Return(nil).Times(2)
	b.On("Requeue", mock.Anything, "job_stream", message, mock.AnythingOfType("time.Duration")).Return(nil)
	b.On("Ack", mock.Anything, "job_stream", "msg-panic").Return(nil)

	w, err := New(Config{
		WorkerID: "worker-4",
		Stream:   "job_stream",
		Broker:   b,
		Storage:  s,
		Registry: reg,
		Logger:   logger,
	})
	assert.NoError(t, err)

	assert.NotPanics(t, func() {
		w.executeJob(context.Background(), logger, message)
	})

	s.AssertExpectations(t)
	b.AssertExpectations(t)
}

func TestCalculateBackoffBounds(t *testing.T) {
	backoff := calculateBackoff(0)
	assert.Greater(t, backoff, 0*time.Millisecond)
	assert.LessOrEqual(t, backoff, 30*time.Second)

	backoff = calculateBackoff(10)
	assert.LessOrEqual(t, backoff, 30*time.Second)
}

