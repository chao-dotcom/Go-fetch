package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap/zaptest"

	"taskqueue/internal/api/handlers"
	"taskqueue/internal/broker"
	"taskqueue/internal/models"
	"taskqueue/internal/storage"
)

type mockStorage struct{ mock.Mock }

func (m *mockStorage) CreateJob(ctx context.Context, job *models.Job) error {
	return m.Called(ctx, job).Error(0)
}
func (m *mockStorage) UpdateJob(ctx context.Context, job *models.Job) error { return nil }
func (m *mockStorage) GetJob(ctx context.Context, id string) (*models.Job, error) {
	args := m.Called(ctx, id)
	if job, ok := args.Get(0).(*models.Job); ok {
		return job, args.Error(1)
	}
	return nil, args.Error(1)
}
func (m *mockStorage) ListJobs(ctx context.Context, filter storage.ListFilter) ([]*models.Job, int, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).([]*models.Job), args.Int(1), args.Error(2)
}
func (m *mockStorage) AppendJobLog(context.Context, models.JobLog) error { return nil }
func (m *mockStorage) ListJobLogs(context.Context, string) ([]models.JobLog, error) {
	return nil, nil
}
func (m *mockStorage) RegisterWorker(context.Context, models.WorkerInfo) error { return nil }
func (m *mockStorage) RecordHeartbeat(context.Context, string, int) error       { return nil }
func (m *mockStorage) ListWorkers(context.Context) ([]models.WorkerInfo, error) { return nil, nil }

type mockBroker struct{ mock.Mock }

func (m *mockBroker) Enqueue(ctx context.Context, msg *broker.Message) error {
	return m.Called(ctx, msg).Error(0)
}
func (m *mockBroker) Consume(context.Context, string, string, int, time.Duration) ([]*broker.Message, error) {
	return nil, nil
}
func (m *mockBroker) Ack(context.Context, string, string) error                     { return nil }
func (m *mockBroker) Requeue(context.Context, string, *broker.Message, time.Duration) error {
	return nil
}
func (m *mockBroker) SendToDeadLetter(context.Context, string, *broker.Message) error { return nil }

type mockMetrics struct{ mock.Mock }

func (m *mockMetrics) RecordJobSubmitted(jobType string) {
	m.Called(jobType)
}

func TestCreateJobHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	store := new(mockStorage)
	brokerMock := new(mockBroker)
	metrics := new(mockMetrics)
	logger := zaptest.NewLogger(t)

	handler := handlers.NewJobHandler(store, brokerMock, logger, "job_stream", metrics)
	router := gin.New()
	router.POST("/v1/jobs", handler.CreateJob)

	store.On("CreateJob", mock.Anything, mock.AnythingOfType("*models.Job")).Return(nil)
	brokerMock.On("Enqueue", mock.Anything, mock.AnythingOfType("*broker.Message")).Return(nil)
	metrics.On("RecordJobSubmitted", "send_email").Return()

	body := map[string]any{
		"type":    "send_email",
		"payload": map[string]any{"to": "user@example.com"},
	}
	payload, _ := json.Marshal(body)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/jobs", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusAccepted, rec.Code)

	store.AssertExpectations(t)
	brokerMock.AssertExpectations(t)
	metrics.AssertExpectations(t)
}

func TestGetJobHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	store := new(mockStorage)
	brokerMock := new(mockBroker)
	logger := zaptest.NewLogger(t)

	handler := handlers.NewJobHandler(store, brokerMock, logger, "job_stream", nil)
	router := gin.New()
	router.GET("/v1/jobs/:id", handler.GetJob)

	job := &models.Job{ID: "job-123", Type: "send_email", Status: models.JobStatusQueued}
	store.On("GetJob", mock.Anything, "job-123").Return(job, nil)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/jobs/job-123", nil)
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	store.AssertExpectations(t)
}

func TestListJobs(t *testing.T) {
	gin.SetMode(gin.TestMode)

	store := new(mockStorage)
	brokerMock := new(mockBroker)
	logger := zaptest.NewLogger(t)

	handler := handlers.NewJobHandler(store, brokerMock, logger, "job_stream", nil)
	router := gin.New()
	router.GET("/v1/jobs", handler.ListJobs)

	store.On("ListJobs", mock.Anything, mock.AnythingOfType("storage.ListFilter")).
		Return([]*models.Job{{ID: "job-1"}}, 1, nil)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/jobs?type=send_email", nil)
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	store.AssertExpectations(t)
}

