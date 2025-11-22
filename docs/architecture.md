# System Architecture

## Overview

The Distributed Task Queue System is built using a microservices architecture pattern, separating concerns into distinct, scalable components. The system is designed for high availability, fault tolerance, and horizontal scalability.

## Component Architecture

### API Server

The API server (`cmd/api/main.go`) serves as the entry point for all client requests. It handles:

- **HTTP REST API** (Gin framework) on port 8080
- **gRPC API** (optional) on port 9090
- **Request routing** and middleware chain
- **Authentication** (JWT, optional)
- **Rate limiting** (configurable per IP)
- **Request validation** and error handling

**Key Components:**
- `internal/api/server.go` - HTTP server setup and routing
- `internal/api/handlers/jobs.go` - Job management endpoints
- `internal/api/middleware/` - Authentication, CORS, logging, recovery

### Worker Pool

Workers (`cmd/worker/main.go`) consume jobs from the message queue and execute them:

- **Goroutine-based concurrency** - Configurable pool size (default: 8)
- **Job consumption** from Redis Streams or memory queue
- **Job execution** via registered handlers
- **State management** - Updates job status in database
- **Retry logic** - Exponential backoff with max attempts
- **Heartbeat monitoring** - Reports status every 5 seconds

**Key Components:**
- `internal/worker/worker.go` - Worker pool implementation
- `internal/jobs/registry.go` - Job type registry and handlers

### Message Broker

The message broker provides reliable job queuing:

**Redis Streams Implementation:**
- **Consumer Groups** - Multiple workers can consume from same stream
- **Message Acknowledgment** - Ensures at-least-once delivery
- **Dead Letter Queue** - Handles permanently failed jobs
- **Requeue Support** - Retry mechanism with delay

**Memory Implementation:**
- Go channels for in-process communication
- Suitable for development and testing

**Key Components:**
- `internal/broker/broker.go` - Broker interface
- `internal/broker/redis/redis.go` - Redis Streams implementation
- `internal/broker/memory/memory.go` - In-memory implementation

### Storage Layer

The storage layer provides persistent job state management:

**PostgreSQL Implementation:**
- **Job persistence** - Complete job lifecycle tracking
- **Job logs** - Execution history and debugging
- **Worker registration** - Worker discovery and health monitoring
- **Connection pooling** - 200 max connections, 50 idle
- **Transactions** - ACID guarantees for state updates

**Memory Implementation:**
- In-memory maps for development
- No persistence (data lost on restart)

**Key Components:**
- `internal/storage/storage.go` - Storage interface
- `internal/storage/postgres/gorm_storage.go` - PostgreSQL implementation
- `internal/storage/memory/memory.go` - In-memory implementation

## Data Flow

### Job Submission Flow

```
1. Client → POST /v1/jobs
   ↓
2. API Handler validates request
   ↓
3. Storage.CreateJob() - Creates job record (status: QUEUED)
   ↓
4. Broker.Enqueue() - Adds job to message queue
   ↓
5. Returns 202 Accepted with job_id
```

### Job Processing Flow

```
1. Worker.consumeLoop() - Reads from message queue
   ↓
2. Message sent to jobChan (buffered channel)
   ↓
3. Worker pool goroutine receives message
   ↓
4. processJob():
   a. Storage.UpdateJob() - Status: QUEUED → RUNNING
   b. Registry.HandlerFor() - Gets job handler
   c. Execute handler function
   d. Storage.UpdateJob() - Status: RUNNING → SUCCEEDED/FAILED
   e. Broker.Ack() - Acknowledges message
   ↓
5. If failed and attempts < max_attempts:
   - Calculate backoff delay (2^attempts * 100ms)
   - Broker.Requeue() - Re-queue with delay
   - Status: FAILED → RETRYING → QUEUED
   ↓
6. If failed and attempts >= max_attempts:
   - Broker.SendToDeadLetter() - Move to DLQ
   - Status: FAILED (permanent)
```

### State Machine

```
QUEUED → RUNNING → SUCCEEDED (end)
           │
           └──→ FAILED
                 │
                 ├─→ RETRYING → QUEUED (if attempts < max)
                 │
                 └─→ FAILED (permanent, if attempts >= max)
                      └─→ Dead Letter Queue
```

## Design Decisions

### Why Redis Streams?

- **Ordered log** - Maintains message order
- **Consumer Groups** - Multiple workers can process same stream
- **At-least-once delivery** - Message acknowledgment ensures reliability
- **High performance** - In-memory with optional persistence
- **Built-in features** - Dead letter queue, message replay

**Alternatives Considered:**
- RabbitMQ: More complex setup, heavier resource usage
- NATS JetStream: Good alternative, but Redis is more widely adopted
- Kafka: Overkill for this use case, higher latency

### Why PostgreSQL?

- **ACID guarantees** - Reliable job state management
- **Rich querying** - Complex filtering and reporting
- **Mature ecosystem** - Proven reliability and tooling
- **JSONB support** - Flexible payload storage
- **Connection pooling** - Efficient resource usage

**Alternatives Considered:**
- MongoDB: NoSQL, but ACID transactions are important
- SQLite: Not suitable for distributed systems
- In-memory: Good for development, not production

### Goroutine Pool vs Traditional Thread Pool

**Advantages:**
- **Lightweight** - Goroutines use ~2KB stack (vs 1MB+ for threads)
- **Efficient scheduling** - Go runtime handles context switching
- **High concurrency** - Can handle thousands of concurrent jobs
- **Native Go** - No external dependencies

**Trade-offs:**
- **Single process** - Cannot scale across machines (solved by multiple worker instances)
- **Memory limits** - Still bound by available RAM

## Scalability

### Horizontal Scaling

**API Servers:**
- Stateless design allows multiple instances
- Load balancer distributes requests
- Shared PostgreSQL and Redis

**Workers:**
- Multiple worker instances consume from same Redis Stream
- Consumer groups ensure load balancing
- Each worker maintains its own connection pool

**Database:**
- PostgreSQL read replicas for read-heavy workloads
- Connection pooling limits per instance
- Vertical scaling for write-heavy workloads

### Vertical Scaling

**Optimizations Applied:**
- PostgreSQL: 300 max connections, 256MB shared buffers
- Database pool: 200 max connections, 50 idle
- File descriptors: 65,536 limit
- HTTP server: Optimized timeouts (30s read, 30s write, 120s idle)
- GOMAXPROCS: 4 (utilizes multi-core CPUs)

### Bottlenecks and Limitations

**Current Bottlenecks:**
- Database write throughput (mitigated by connection pooling)
- Redis memory (for large queue depths)
- Network bandwidth (for high-throughput scenarios)

**Limitations:**
- Single PostgreSQL instance (can be sharded if needed)
- Redis single-threaded (can use Redis Cluster)
- Worker pool size per instance (can run multiple instances)

## Trade-offs

### Consistency vs Availability

- **Chosen: Strong Consistency**
  - Job state is always consistent
  - Trade-off: Slightly higher latency for state updates
  - Justification: Job state accuracy is critical

### Latency vs Throughput

- **Chosen: Optimize for Latency**
  - Sub-10ms p95 latency achieved
  - Trade-off: Lower throughput than batch processing
  - Justification: Real-time job processing requirements

### Memory vs Persistence

- **Chosen: Full Persistence**
  - All jobs persisted to PostgreSQL
  - Trade-off: Higher storage costs
  - Justification: Job history and debugging needs

## Monitoring and Observability

### Metrics (Prometheus)

- `job_processing_seconds` - Job execution duration histogram
- `job_failures_total` - Total job failures counter
- `worker_uptime_seconds` - Worker uptime gauge
- `queue_depth` - Current queue depth gauge
- `http_requests_total` - HTTP request counter
- `http_request_duration_seconds` - HTTP latency histogram

### Tracing (OpenTelemetry)

- Distributed traces across API → Broker → Worker → Database
- Jaeger integration for trace visualization
- Correlation IDs for request tracking

### Logging

- Structured JSON logs (Zap)
- Request correlation IDs
- Log levels: DEBUG, INFO, WARN, ERROR
- Contextual information (job_id, worker_id, etc.)

## Security Considerations

- **JWT Authentication** - Optional, configurable
- **Rate Limiting** - Per-IP request throttling
- **CORS** - Configurable allowed origins
- **Input Validation** - Request payload validation
- **SQL Injection Protection** - Parameterized queries
- **TLS/SSL** - Recommended for production (not included in Docker setup)

## Future Enhancements

- **Kubernetes Deployment** - Helm charts for K8s
- **Job Scheduling** - CRON-like scheduled jobs (partially implemented)
- **Priority Queues** - Multiple priority levels
- **Job Dependencies** - DAG-based job execution
- **Webhook Notifications** - Job completion callbacks
- **Multi-region Support** - Geographic distribution

