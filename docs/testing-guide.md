# Testing Guide

## Overview

This guide covers all testing strategies for the Distributed Task Queue System, including unit tests, integration tests, and load testing.

## Unit Testing

### Running Unit Tests

```bash
# Run all tests
make test

# Run specific package
go test ./internal/worker/...

# Run with verbose output
go test -v ./...

# Run with coverage
go test -cover ./...
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Writing Unit Tests

**Example test structure:**
```go
package worker_test

import (
    "context"
    "testing"
    "time"
    
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestWorker_ProcessJob(t *testing.T) {
    // Setup
    ctx := context.Background()
    worker := setupTestWorker(t)
    
    // Test
    job := createTestJob()
    err := worker.ProcessJob(ctx, job)
    
    // Assert
    require.NoError(t, err)
    assert.Equal(t, "SUCCEEDED", job.Status)
}
```

### Test Utilities

**Mock storage:**
```go
type mockStorage struct {
    jobs map[string]*models.Job
}

func (m *mockStorage) CreateJob(ctx context.Context, job *models.Job) error {
    m.jobs[job.ID] = job
    return nil
}
```

**Mock broker:**
```go
type mockBroker struct {
    messages []*broker.Message
}

func (m *mockBroker) Enqueue(ctx context.Context, msg *broker.Message) error {
    m.messages = append(m.messages, msg)
    return nil
}
```

## Integration Testing

### Setup Test Environment

**Create `docker-compose.test.yml`:**
```yaml
services:
  postgres-test:
    image: postgres:16.1-alpine
    environment:
      POSTGRES_PASSWORD: test
      POSTGRES_USER: test
      POSTGRES_DB: test
    ports:
      - "5433:5432"

  redis-test:
    image: redis:7.2-alpine
    ports:
      - "6380:6379"
```

**Start test services:**
```bash
docker-compose -f docker-compose.test.yml up -d
```

**Run integration tests:**
```bash
go test -tags=integration ./...
```

**Cleanup:**
```bash
docker-compose -f docker-compose.test.yml down -v
```

### Integration Test Example

```go
// +build integration

package integration

import (
    "context"
    "testing"
    
    "taskqueue/internal/storage/postgres"
)

func TestPostgreSQLStorage(t *testing.T) {
    store := postgres.NewGormStore(
        "postgres://test:test@localhost:5433/test?sslmode=disable",
        logger,
    )
    
    ctx := context.Background()
    job := &models.Job{
        ID:     "test-id",
        Type:   "test",
        Status: models.JobStatusQueued,
    }
    
    err := store.CreateJob(ctx, job)
    require.NoError(t, err)
    
    retrieved, err := store.GetJob(ctx, "test-id")
    require.NoError(t, err)
    assert.Equal(t, job.ID, retrieved.ID)
}
```

## Load Testing

### k6 Setup

**Install k6:**
```bash
# Mac
brew install k6

# Linux
sudo gpg -k
sudo gpg --no-default-keyring --keyring /usr/share/keyrings/k6-archive-keyring.gpg --keyserver hkp://keyserver.ubuntu.com:80 --recv-keys C5AD17C747E3415A3642D57D77C6C491D6AC1D69
echo "deb [signed-by=/usr/share/keyrings/k6-archive-keyring.gpg] https://dl.k6.io/deb stable main" | sudo tee /etc/apt/sources.list.d/k6.list
sudo apt-get update
sudo apt-get install k6

# Windows
choco install k6
```

### Running Load Tests

**Quick test (15 VU):**
```bash
k6 run scripts/quick-stress-test.js
```

**High load test (100 VU):**
```bash
k6 run scripts/stress-test-high-load.js
```

**Custom test:**
```bash
k6 run --vus 50 --duration 5m scripts/quick-stress-test.js
```

**Using Docker:**
```bash
docker run --rm -i --network go_work_default \
  -v ${PWD}/scripts:/scripts \
  grafana/k6 run \
  --env API_URL=http://go_work-api-1:8080/v1 \
  /scripts/stress-test-high-load.js
```

### Test Scripts

| Script | Description | Concurrency | Duration |
|--------|-------------|-------------|----------|
| `quick-stress-test.js` | Standard load test | 15 VU | ~2 min |
| `stress-test-high-load.js` | High load test | 100 VU | 8 min |
| `load-test.js` | Extended test | 200 VU | 24 min |
| `quick-stress-test-30.js` | Quick validation | 3-5 VU | 30 sec |
| `quick-stress-test-10.js` | Very quick test | 3-10 VU | 10 sec |

### Understanding Results

**Key metrics:**
- **p95 latency**: 95% of requests faster than this value
- **p99 latency**: 99% of requests faster than this value
- **Failure rate**: Percentage of failed requests
- **Throughput**: Requests per second

**Target thresholds:**
- p95 latency < 50ms
- p99 latency < 100ms
- Failure rate < 1%
- Throughput > 100 req/s

**Our results:**
- p95 latency: 7.17ms ✅
- p99 latency: 17.2ms ✅
- Failure rate: 0.00% ✅
- Throughput: 116 req/s ✅

## Performance Testing

### Benchmark Tests

**Run Go benchmarks:**
```bash
go test -bench=. -benchmem ./...
```

**Example benchmark:**
```go
func BenchmarkWorker_ProcessJob(b *testing.B) {
    worker := setupTestWorker(b)
    job := createTestJob()
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        worker.ProcessJob(context.Background(), job)
    }
}
```

### Profiling

**CPU profile:**
```bash
go test -cpuprofile=cpu.prof -bench=. ./...
go tool pprof cpu.prof
```

**Memory profile:**
```bash
go test -memprofile=mem.prof -bench=. ./...
go tool pprof mem.prof
```

## End-to-End Testing

### Test Scenarios

**1. Job Submission and Processing:**
```bash
# Submit job
JOB_ID=$(curl -X POST http://localhost:8080/v1/jobs \
  -H "Content-Type: application/json" \
  -d '{"type": "send_email", "payload": {"to": "test@example.com"}}' \
  | jq -r '.job_id')

# Wait for processing
sleep 5

# Check status
curl http://localhost:8080/v1/jobs/$JOB_ID
```

**2. Retry Logic:**
```bash
# Submit job that will fail
JOB_ID=$(curl -X POST http://localhost:8080/v1/jobs \
  -H "Content-Type: application/json" \
  -d '{"type": "failing_job", "payload": {}}' \
  | jq -r '.job_id')

# Wait for retries
sleep 10

# Check attempts
curl http://localhost:8080/v1/jobs/$JOB_ID | jq '.attempts'
```

**3. Worker Scaling:**
```bash
# Start with 1 worker
docker-compose up -d --scale worker=1

# Submit jobs
for i in {1..100}; do
  curl -X POST http://localhost:8080/v1/jobs \
    -H "Content-Type: application/json" \
    -d "{\"type\": \"send_email\", \"payload\": {\"to\": \"test$i@example.com\"}}"
done

# Scale to 3 workers
docker-compose up -d --scale worker=3

# Monitor queue depth
watch -n 1 'curl -s http://localhost:8080/v1/queues | jq'
```

## Test Data Management

### Fixtures

**Create test fixtures:**
```go
func createTestJob() *models.Job {
    return &models.Job{
        ID:          uuid.NewString(),
        Type:        "send_email",
        Status:      models.JobStatusQueued,
        MaxAttempts: 3,
        Payload:     json.RawMessage(`{"to": "test@example.com"}`),
    }
}
```

### Database Cleanup

**Clean test database:**
```go
func cleanupTestDB(t *testing.T, store storage.Storage) {
    ctx := context.Background()
    jobs, _, _ := store.ListJobs(ctx, storage.ListFilter{Limit: 1000})
    for _, job := range jobs {
        // Delete job logic
    }
}
```

## Continuous Integration

### GitHub Actions Example

```yaml
name: Test

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.22'
      - run: go test ./...
      - run: go test -coverprofile=coverage.out ./...
      - uses: codecov/codecov-action@v3
        with:
          file: ./coverage.out
```

## Best Practices

1. **Write tests first** (TDD) for critical paths
2. **Keep tests fast** - Unit tests should run in < 1s
3. **Use table-driven tests** for multiple scenarios
4. **Mock external dependencies** in unit tests
5. **Use real dependencies** in integration tests
6. **Run load tests regularly** to catch performance regressions
7. **Maintain test coverage** > 80% for critical packages
8. **Document test scenarios** and expected behavior

## Troubleshooting Tests

### Common Issues

**1. Tests failing intermittently:**
- Check for race conditions: `go test -race ./...`
- Ensure proper cleanup between tests
- Use proper synchronization

**2. Integration tests timing out:**
- Increase timeout values
- Check service health before tests
- Verify network connectivity

**3. Load tests showing high failure rate:**
- Check system resources (CPU, memory)
- Verify rate limiting is disabled
- Check database connection pool
- Monitor worker capacity

See [Stress Test Guide](../scripts/STRESS-TEST-GUIDE.md) for more load testing details.

