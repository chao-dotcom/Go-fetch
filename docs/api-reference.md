# API Reference

## Base URL

```
http://localhost:8080/v1
```

## Authentication

JWT authentication is **optional** and disabled by default for development. To enable:

1. Set `JWT_SECRET` environment variable
2. Include `Authorization: Bearer <token>` header in requests

**Note**: For production, JWT authentication should be enabled.

## Rate Limiting

Rate limiting is **disabled by default** for development. To enable:

1. Set `ENABLE_RATE_LIMITER=true`
2. Configure `RATE_LIMIT_PER_MINUTE` (default: 120)

Rate limits are applied per IP address.

## Endpoints

### Jobs API

#### POST /jobs

Submit a new job for processing.

**Request Body:**
```json
{
  "type": "send_email",
  "payload": {
    "to": "user@example.com",
    "subject": "Hello",
    "body": "Welcome to the task queue!"
  },
  "priority": 0,
  "max_attempts": 3,
  "timeout_seconds": 300,
  "idempotency_key": "optional-unique-key"
}
```

**Response:** `202 Accepted`
```json
{
  "job_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "QUEUED",
  "message": "Job accepted for processing"
}
```

**Job Types:**
- `send_email` - Send email notification
- `process_video` - Process video file
- `generate_thumbnail` - Generate video thumbnail
- `webhook` - Send HTTP webhook

**Fields:**
- `type` (required): Job type identifier
- `payload` (required): Job-specific data
- `priority` (optional): Job priority (higher = more important, default: 0)
- `max_attempts` (optional): Maximum retry attempts (1-10, default: 5)
- `timeout_seconds` (optional): Job timeout in seconds
- `idempotency_key` (optional): Ensures job is only processed once

#### GET /jobs

List all jobs with optional filtering.

**Query Parameters:**
- `status` (optional): Filter by status (`QUEUED`, `RUNNING`, `SUCCEEDED`, `FAILED`, `RETRYING`)
- `type` (optional): Filter by job type
- `limit` (optional): Maximum number of results (default: 100, max: 1000)
- `offset` (optional): Pagination offset (default: 0)

**Example:**
```
GET /v1/jobs?status=QUEUED&limit=50&offset=0
```

**Response:** `200 OK`
```json
{
  "jobs": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "type": "send_email",
      "status": "QUEUED",
      "attempts": 0,
      "max_attempts": 3,
      "created_at": "2024-01-01T12:00:00Z",
      "updated_at": "2024-01-01T12:00:00Z"
    }
  ],
  "total": 100,
  "limit": 50,
  "offset": 0
}
```

#### GET /jobs/:id

Get detailed information about a specific job.

**Response:** `200 OK`
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "type": "send_email",
  "payload": {
    "to": "user@example.com",
    "subject": "Hello",
    "body": "Welcome!"
  },
  "status": "SUCCEEDED",
  "attempts": 1,
  "max_attempts": 3,
  "result": {
    "message_id": "<123@taskqueue.local>",
    "sent_at": "2024-01-01T12:00:05Z"
  },
  "created_at": "2024-01-01T12:00:00Z",
  "updated_at": "2024-01-01T12:00:05Z",
  "started_at": "2024-01-01T12:00:01Z",
  "finished_at": "2024-01-01T12:00:05Z",
  "worker_id": "worker-123"
}
```

**Response:** `404 Not Found` (if job doesn't exist)
```json
{
  "error": "job not found"
}
```

#### DELETE /jobs/:id

Cancel a queued or running job.

**Response:** `200 OK`
```json
{
  "message": "Job cancelled successfully"
}
```

**Response:** `400 Bad Request` (if job cannot be cancelled)
```json
{
  "error": "job cannot be cancelled (already finished)"
}
```

#### POST /jobs/:id/retry

Manually retry a failed job.

**Response:** `200 OK`
```json
{
  "message": "Job queued for retry",
  "job_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "QUEUED"
}
```

#### GET /jobs/:id/logs

Get execution logs for a job.

**Response:** `200 OK`
```json
{
  "logs": [
    {
      "id": 1,
      "job_id": "550e8400-e29b-41d4-a716-446655440000",
      "level": "INFO",
      "message": "Job started processing",
      "timestamp": "2024-01-01T12:00:01Z"
    },
    {
      "id": 2,
      "job_id": "550e8400-e29b-41d4-a716-446655440000",
      "level": "INFO",
      "message": "Job completed successfully",
      "timestamp": "2024-01-01T12:00:05Z"
    }
  ]
}
```

### Queues API

#### GET /queues

Get queue statistics and status.

**Response:** `200 OK`
```json
{
  "queues": [
    {
      "name": "job_stream",
      "depth": 150,
      "pending_count": 10,
      "broker_depth": 140,
      "consumers": 3
    }
  ]
}
```

**Fields:**
- `name`: Queue/stream name
- `depth`: Total number of jobs in queue
- `pending_count`: Jobs being processed
- `broker_depth`: Jobs in message broker
- `consumers`: Number of active workers

### Workers API

#### GET /workers

List all registered workers.

**Response:** `200 OK`
```json
{
  "workers": [
    {
      "id": "worker-123",
      "hostname": "worker-1",
      "pool_size": 8,
      "active_jobs": 5,
      "total_jobs": 1000,
      "total_processed": 950,
      "total_failed": 50,
      "status": "busy",
      "last_heartbeat": "2024-01-01T12:00:00Z",
      "started_at": "2024-01-01T10:00:00Z"
    }
  ]
}
```

**Worker Status:**
- `starting` - Worker is initializing
- `idle` - Worker is running but has no active jobs
- `busy` - Worker is processing jobs
- `offline` - Worker hasn't sent heartbeat recently (> 30s)

### System API

#### GET /health

Health check endpoint.

**Response:** `200 OK`
```json
{
  "status": "healthy",
  "timestamp": "2024-01-01T12:00:00Z"
}
```

#### GET /metrics

Prometheus metrics endpoint.

**Response:** `200 OK` (Prometheus format)
```
# HELP job_processing_seconds Job processing duration
# TYPE job_processing_seconds histogram
job_processing_seconds_bucket{le="0.005"} 100
job_processing_seconds_bucket{le="0.01"} 200
...

# HELP http_requests_total Total HTTP requests
# TYPE http_requests_total counter
http_requests_total{method="POST",endpoint="/v1/jobs"} 1000
...
```

## Error Handling

### Error Response Format

All errors follow this format:

```json
{
  "error": "error message",
  "code": "ERROR_CODE",
  "details": {}
}
```

### HTTP Status Codes

- `200 OK` - Request successful
- `202 Accepted` - Job accepted for processing
- `400 Bad Request` - Invalid request (validation error)
- `401 Unauthorized` - Authentication required
- `404 Not Found` - Resource not found
- `429 Too Many Requests` - Rate limit exceeded
- `500 Internal Server Error` - Server error

### Common Error Codes

- `JOB_NOT_FOUND` - Job ID doesn't exist
- `INVALID_JOB_TYPE` - Job type not registered
- `VALIDATION_ERROR` - Request validation failed
- `RATE_LIMIT_EXCEEDED` - Too many requests
- `INTERNAL_ERROR` - Server error

## gRPC API

### Protobuf Definition

See `internal/grpc/proto/taskqueue.proto` for complete gRPC service definitions.

### Generate Go Code

```bash
make proto
```

### gRPC Services

- `JobService` - Job management operations
- `QueueService` - Queue statistics
- `WorkerService` - Worker management

## Examples

### Submit Job (cURL)

```bash
curl -X POST http://localhost:8080/v1/jobs \
  -H "Content-Type: application/json" \
  -d '{
    "type": "send_email",
    "payload": {
      "to": "user@example.com",
      "subject": "Hello",
      "body": "Welcome!"
    }
  }'
```

### Submit Job (PowerShell)

```powershell
Invoke-RestMethod -Uri http://localhost:8080/v1/jobs `
  -Method POST `
  -ContentType "application/json" `
  -Body '{
    "type": "send_email",
    "payload": {
      "to": "user@example.com",
      "subject": "Hello",
      "body": "Welcome!"
    }
  }'
```

### Check Job Status

```bash
curl http://localhost:8080/v1/jobs/550e8400-e29b-41d4-a716-446655440000
```

### List Queued Jobs

```bash
curl "http://localhost:8080/v1/jobs?status=QUEUED&limit=10"
```

### Get Queue Statistics

```bash
curl http://localhost:8080/v1/queues
```

### List Workers

```bash
curl http://localhost:8080/v1/workers
```

