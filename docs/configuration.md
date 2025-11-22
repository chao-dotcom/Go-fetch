# Configuration Reference

## Environment Variables

### Required Variables

| Variable | Description | Example |
|----------|-------------|---------|
| `DATABASE_URL` | PostgreSQL connection string (when using postgres storage) | `postgres://user:pass@host:5432/db?sslmode=disable` |
| `REDIS_ADDR` | Redis connection address (when using redis broker) | `redis:6379` |

### Optional Variables

#### API Server Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `API_ADDR` | HTTP server listen address | `:8080` |
| `GRPC_ADDR` | gRPC server listen address | `:9090` |
| `STORAGE_DRIVER` | Storage backend (`memory` or `postgres`) | `memory` |
| `BROKER_DRIVER` | Message broker (`memory` or `redis`) | `memory` |
| `QUEUE_STREAM` | Redis stream/queue name | `job_stream` |
| `CONSUMER_GROUP` | Redis consumer group name | `workers` |
| `JWT_SECRET` | JWT signing secret (empty to disable) | `""` |
| `ENABLE_RATE_LIMITER` | Enable rate limiting | `false` |
| `RATE_LIMIT_PER_MINUTE` | Requests per minute per IP | `120` |
| `CORS_ORIGINS` | Allowed CORS origins (comma-separated) | `*` |
| `LOG_LEVEL` | Logging level (`DEBUG`, `INFO`, `WARN`, `ERROR`) | `INFO` |
| `SERVICE_NAME` | Service name for telemetry | `taskqueue-api` |
| `GOMAXPROCS` | Go runtime CPU limit | (system default) |

#### Worker Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `WORKER_ID` | Unique worker identifier | (auto-generated) |
| `HOSTNAME` | Worker hostname | (system hostname) |
| `WORKER_POOL_SIZE` | Goroutine pool size | `8` |
| `WORKER_MAX_RETRIES` | Maximum retry attempts per job | `5` |
| `SHUTDOWN_TIMEOUT` | Graceful shutdown timeout | `30s` |
| `STORAGE_DRIVER` | Storage backend | `memory` |
| `BROKER_DRIVER` | Message broker | `memory` |
| `QUEUE_STREAM` | Redis stream/queue name | `job_stream` |
| `CONSUMER_GROUP` | Redis consumer group name | `workers` |
| `LOG_LEVEL` | Logging level | `INFO` |
| `SERVICE_NAME` | Service name for telemetry | `taskqueue-worker` |

#### Redis Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `REDIS_ADDR` | Redis server address | `127.0.0.1:6379` |
| `REDIS_USERNAME` | Redis username (if required) | `""` |
| `REDIS_PASSWORD` | Redis password (if required) | `""` |
| `REDIS_DB` | Redis database number | `0` |

## Configuration Examples

### Development Configuration

```bash
# .env (development)
STORAGE_DRIVER=memory
BROKER_DRIVER=memory
API_ADDR=:8080
ENABLE_RATE_LIMITER=false
JWT_SECRET=
LOG_LEVEL=DEBUG
```

### Production Configuration

```bash
# .env (production)
STORAGE_DRIVER=postgres
BROKER_DRIVER=redis
DATABASE_URL=postgres://user:pass@postgres:5432/taskqueue?sslmode=require&pool_max_conns=200
REDIS_ADDR=redis:6379
API_ADDR=:8080
ENABLE_RATE_LIMITER=true
RATE_LIMIT_PER_MINUTE=120
JWT_SECRET=your-secure-secret-key-here
LOG_LEVEL=INFO
GOMAXPROCS=4
```

### High-Performance Configuration

```bash
# .env (high-performance)
STORAGE_DRIVER=postgres
BROKER_DRIVER=redis
DATABASE_URL=postgres://user:pass@postgres:5432/taskqueue?sslmode=require&pool_max_conns=200
REDIS_ADDR=redis:6379
API_ADDR=:8080
ENABLE_RATE_LIMITER=true
RATE_LIMIT_PER_MINUTE=10000
JWT_SECRET=your-secure-secret-key-here
WORKER_POOL_SIZE=16
WORKER_MAX_RETRIES=3
LOG_LEVEL=WARN
GOMAXPROCS=8
```

## Storage Drivers

### Memory Storage

**Use Case**: Development, testing, single-process applications

**Configuration:**
```bash
STORAGE_DRIVER=memory
```

**Characteristics:**
- No persistence (data lost on restart)
- Fast in-memory operations
- No external dependencies
- Not suitable for production

### PostgreSQL Storage

**Use Case**: Production, multi-process, distributed systems

**Configuration:**
```bash
STORAGE_DRIVER=postgres
DATABASE_URL=postgres://user:pass@host:5432/db?sslmode=require&pool_max_conns=200
```

**Connection String Parameters:**
- `sslmode`: `disable` (dev) or `require` (prod)
- `pool_max_conns`: Maximum connection pool size (default: 200)
- `pool_max_idle_conns`: Maximum idle connections (default: 50)

**Characteristics:**
- Full persistence
- ACID guarantees
- Scalable and reliable
- Requires PostgreSQL 16+

## Broker Drivers

### Memory Broker

**Use Case**: Development, testing, single-process applications

**Configuration:**
```bash
BROKER_DRIVER=memory
```

**Characteristics:**
- In-process Go channels
- No external dependencies
- Fast for development
- Not suitable for distributed systems

### Redis Broker

**Use Case**: Production, distributed systems, high availability

**Configuration:**
```bash
BROKER_DRIVER=redis
REDIS_ADDR=redis:6379
QUEUE_STREAM=job_stream
CONSUMER_GROUP=workers
```

**Optional Redis Authentication:**
```bash
REDIS_USERNAME=redis_user
REDIS_PASSWORD=redis_password
REDIS_DB=0
```

**Characteristics:**
- Redis Streams for reliable messaging
- Consumer groups for load balancing
- Message acknowledgment
- Dead letter queue support
- Requires Redis 7+

## Docker Compose Configuration

### Environment Variables in docker-compose.yml

```yaml
services:
  api:
    environment:
      STORAGE_DRIVER: postgres
      BROKER_DRIVER: redis
      DATABASE_URL: postgres://taskqueue:taskqueue@postgres:5432/taskqueue?sslmode=disable&pool_max_conns=200
      REDIS_ADDR: redis:6379
      QUEUE_STREAM: job_stream
      CONSUMER_GROUP: workers
      ENABLE_RATE_LIMITER: "false"
      RATE_LIMIT_PER_MINUTE: "10000"
      JWT_SECRET: ""
      GOMAXPROCS: "4"

  worker:
    environment:
      STORAGE_DRIVER: postgres
      BROKER_DRIVER: redis
      DATABASE_URL: postgres://taskqueue:taskqueue@postgres:5432/taskqueue?sslmode=disable
      REDIS_ADDR: redis:6379
      QUEUE_STREAM: job_stream
      CONSUMER_GROUP: workers
      WORKER_POOL_SIZE: 8
```

## Configuration Best Practices

### Security

1. **Never commit secrets to version control**
   - Use environment variables or secrets management
   - Use `.env` files with `.gitignore`

2. **Use strong JWT secrets**
   - Minimum 32 characters
   - Randomly generated
   - Rotated regularly

3. **Enable SSL for production**
   - Set `sslmode=require` in DATABASE_URL
   - Use TLS for Redis if exposed

4. **Restrict CORS origins**
   - Don't use `*` in production
   - Specify exact allowed origins

### Performance

1. **Optimize connection pools**
   - Set `pool_max_conns` based on expected load
   - Monitor connection usage
   - Adjust based on database capacity

2. **Tune worker pool size**
   - Start with CPU cores × 2
   - Monitor CPU utilization (target: 70-80%)
   - Adjust based on job processing time

3. **Configure appropriate timeouts**
   - Job timeout based on expected duration
   - HTTP timeouts for external calls
   - Database query timeouts

### Reliability

1. **Enable rate limiting in production**
   - Prevents abuse and overload
   - Set appropriate limits per use case

2. **Use structured logging**
   - Set appropriate log level
   - Use JSON format for production
   - Include correlation IDs

3. **Configure health checks**
   - Enable health check endpoints
   - Set appropriate intervals
   - Configure alerting thresholds

## Configuration Validation

The system validates configuration on startup:

- **Storage driver**: Must be `memory` or `postgres`
- **Broker driver**: Must be `memory` or `redis`
- **Database URL**: Required if using `postgres` storage
- **Redis address**: Required if using `redis` broker
- **Worker pool size**: Must be > 0
- **Max retries**: Must be between 1 and 10

Invalid configuration will cause the service to fail on startup with a clear error message.

## Dynamic Configuration

Some settings can be changed at runtime:

- **Worker pool size**: Requires worker restart
- **Rate limit**: Requires API restart
- **Log level**: Can be changed via environment variable (requires restart)

Most configuration changes require a service restart to take effect.

