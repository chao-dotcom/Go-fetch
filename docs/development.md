# Development Guide

## Prerequisites

- **Go 1.22+** - [Download](https://golang.org/dl/)
- **PostgreSQL 16+** - [Download](https://www.postgresql.org/download/)
- **Redis 7+** - [Download](https://redis.io/download)
- **Node.js 18+** - [Download](https://nodejs.org/) (for dashboard)
- **Docker & Docker Compose** - [Download](https://www.docker.com/) (optional but recommended)
- **Make** - Usually pre-installed on Linux/Mac, [download for Windows](https://www.gnu.org/software/make/)

## Local Setup

### Option 1: With Docker (Recommended)

**Start all services:**
```bash
docker-compose up -d
```

**View logs:**
```bash
docker-compose logs -f api
docker-compose logs -f worker
```

**Stop services:**
```bash
docker-compose down
```

### Option 2: Without Docker

**1. Start PostgreSQL:**
```bash
# Linux/Mac
sudo service postgresql start

# Or using Docker
docker run -d -p 5432:5432 \
  -e POSTGRES_PASSWORD=taskqueue \
  -e POSTGRES_USER=taskqueue \
  -e POSTGRES_DB=taskqueue \
  postgres:16.1-alpine
```

**2. Start Redis:**
```bash
# Linux/Mac
redis-server

# Or using Docker
docker run -d -p 6379:6379 redis:7.2-alpine
```

**3. Set environment variables:**
```bash
export STORAGE_DRIVER=postgres
export BROKER_DRIVER=redis
export DATABASE_URL=postgres://taskqueue:taskqueue@localhost:5432/taskqueue?sslmode=disable
export REDIS_ADDR=localhost:6379
```

**4. Run API server:**
```bash
make run-api
# Or
go run cmd/api/main.go
```

**5. Run worker:**
```bash
make run-worker
# Or
go run cmd/worker/main.go
```

**6. Run dashboard:**
```bash
cd dashboard
npm install
npm run dev
```

## Development Workflow

### Running Services

**Start all services:**
```bash
make docker-up
```

**Start individual services:**
```bash
make run-api      # API server only
make run-worker   # Worker only
```

**View service logs:**
```bash
docker-compose logs -f api
docker-compose logs -f worker
```

### Hot Reload

**For Go services:**
Install [air](https://github.com/cosmtrek/air):
```bash
go install github.com/cosmtrek/air@latest
```

Create `.air.toml`:
```toml
[build]
  cmd = "go run cmd/api/main.go"
  bin = "tmp/main"
  include_ext = ["go"]
  exclude_dir = ["vendor", "dashboard"]
```

Run with hot reload:
```bash
air
```

**For Dashboard:**
```bash
cd dashboard
npm run dev  # Vite hot reload enabled
```

### Debugging

**Debug API server:**
```bash
# Using Delve debugger
dlv debug cmd/api/main.go

# Set breakpoints
(dlv) break handlers/jobs.go:57
(dlv) continue
```

**Debug worker:**
```bash
dlv debug cmd/worker/main.go
```

**View database:**
```bash
# Connect to PostgreSQL
docker-compose exec postgres psql -U taskqueue -d taskqueue

# View jobs
SELECT * FROM jobs ORDER BY created_at DESC LIMIT 10;

# View workers
SELECT * FROM workers;
```

**View Redis:**
```bash
# Connect to Redis
docker-compose exec redis redis-cli

# View stream
XINFO STREAM job_stream

# View consumer group
XINFO GROUPS job_stream
```

## Code Organization

### Project Structure

```
.
├── cmd/                    # Application entry points
│   ├── api/               # API server
│   ├── worker/            # Worker process
│   └── cli/               # CLI client
├── internal/              # Private application code
│   ├── api/               # HTTP handlers and middleware
│   ├── broker/            # Message broker implementations
│   ├── storage/           # Storage implementations
│   ├── worker/            # Worker pool logic
│   ├── jobs/              # Job handlers
│   └── models/            # Data models
├── pkg/                   # Public reusable packages
│   ├── config/            # Configuration management
│   ├── logger/            # Logging utilities
│   └── instrumentation/  # Metrics and tracing
└── dashboard/             # React frontend
```

### Package Guidelines

- **`cmd/`**: Entry points only, minimal logic
- **`internal/`**: Private code, not importable by external packages
- **`pkg/`**: Public packages, can be imported by external code
- **`dashboard/`**: Frontend code, separate build process

## Adding Features

### Adding a New Job Type

**1. Create handler function in `internal/jobs/registry.go`:**

```go
func ExecuteCustomJob(ctx context.Context, payload json.RawMessage) (any, error) {
    var params CustomJobPayload
    if err := json.Unmarshal(payload, &params); err != nil {
        return nil, fmt.Errorf("invalid payload: %w", err)
    }
    
    // Your job logic here
    result := processCustomJob(ctx, params)
    
    return CustomJobResult{
        // Result fields
    }, nil
}
```

**2. Register in `NewRegistry()`:**

```go
func NewRegistry() *Registry {
    r := &Registry{
        handlers: make(map[string]Handler),
    }
    r.Register("send_email", ExecuteEmailJob)
    r.Register("custom_job", ExecuteCustomJob)  // Add this
    return r
}
```

**3. Test your handler:**

```go
func TestExecuteCustomJob(t *testing.T) {
    payload := json.RawMessage(`{"param": "value"}`)
    result, err := ExecuteCustomJob(context.Background(), payload)
    assert.NoError(t, err)
    assert.NotNil(t, result)
}
```

### Adding a New Storage Driver

**1. Implement the `Storage` interface in `internal/storage/storage.go`:**

```go
type CustomStorage struct {
    // Your storage fields
}

func (s *CustomStorage) CreateJob(ctx context.Context, job *models.Job) error {
    // Implementation
}

func (s *CustomStorage) UpdateJob(ctx context.Context, job *models.Job) error {
    // Implementation
}

// Implement all interface methods
```

**2. Add driver selection in `cmd/api/main.go`:**

```go
var store storage.Storage
switch cfg.StorageDriver {
case "postgres":
    store = pgstorage.NewGormStore(cfg.DBURL, zapLogger)
case "custom":
    store = customstorage.New(cfg.CustomConfig)
default:
    store = memory.New()
}
```

### Adding a New API Endpoint

**1. Add handler method in `internal/api/handlers/jobs.go`:**

```go
func (h *JobHandler) CustomEndpoint(c *gin.Context) {
    // Handler logic
    c.JSON(http.StatusOK, gin.H{"message": "success"})
}
```

**2. Register route in `internal/api/server.go`:**

```go
v1.GET("/custom", handlers.CustomEndpoint)
```

## Testing

### Unit Tests

**Run all tests:**
```bash
make test
# Or
go test ./...
```

**Run specific package:**
```bash
go test ./internal/worker/...
```

**Run with coverage:**
```bash
go test -cover ./...
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Integration Tests

**Test with Docker:**
```bash
# Start test services
docker-compose -f docker-compose.test.yml up -d

# Run integration tests
go test -tags=integration ./...

# Cleanup
docker-compose -f docker-compose.test.yml down
```

### Load Tests

**Run k6 tests:**
```bash
k6 run scripts/quick-stress-test.js
```

See [Testing Guide](testing-guide.md) for more details.

## Code Style

### Formatting

**Format code:**
```bash
go fmt ./...
```

**Use goimports:**
```bash
go install golang.org/x/tools/cmd/goimports@latest
goimports -w .
```

### Linting

**Install golangci-lint:**
```bash
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

**Run linter:**
```bash
golangci-lint run
```

### Code Review Checklist

- [ ] Code formatted with `go fmt`
- [ ] No linting errors
- [ ] Tests added/updated
- [ ] Documentation updated
- [ ] Error handling implemented
- [ ] Logging added where appropriate
- [ ] No hardcoded values (use config)
- [ ] Context propagation for cancellation

## Common Development Tasks

### View Job Status

```bash
curl http://localhost:8080/v1/jobs/{job_id}
```

### Submit Test Job

```bash
curl -X POST http://localhost:8080/v1/jobs \
  -H "Content-Type: application/json" \
  -d '{
    "type": "send_email",
    "payload": {"to": "test@example.com", "subject": "Test", "body": "Test"}
  }'
```

### Check Queue Depth

```bash
curl http://localhost:8080/v1/queues
```

### View Worker Status

```bash
curl http://localhost:8080/v1/workers
```

### Check Metrics

```bash
curl http://localhost:8080/metrics
```

## Troubleshooting

### Common Issues

**1. Port already in use:**
```bash
# Find process using port
lsof -i :8080  # Mac/Linux
netstat -ano | findstr :8080  # Windows

# Kill process
kill -9 <PID>
```

**2. Database connection error:**
- Check PostgreSQL is running
- Verify DATABASE_URL is correct
- Check database exists: `psql -U taskqueue -d taskqueue`

**3. Redis connection error:**
- Check Redis is running: `redis-cli ping`
- Verify REDIS_ADDR is correct

**4. Import errors:**
```bash
# Download dependencies
go mod download

# Tidy modules
go mod tidy
```

## Contributing

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/my-feature`
3. Make your changes
4. Add tests
5. Run tests: `make test`
6. Commit changes: `git commit -am 'Add feature'`
7. Push to branch: `git push origin feature/my-feature`
8. Submit pull request

