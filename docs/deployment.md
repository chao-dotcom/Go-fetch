# Deployment Guide

## System Requirements

### Minimum Requirements

- **CPU**: 2 cores
- **Memory**: 4GB RAM
- **Storage**: 20GB (for PostgreSQL data)
- **Network**: 100 Mbps

### Recommended Requirements

- **CPU**: 4+ cores
- **Memory**: 8GB+ RAM
- **Storage**: 100GB SSD (for PostgreSQL data)
- **Network**: 1 Gbps

### Database Requirements

- **PostgreSQL**: 16+ with 300 max connections
- **Redis**: 7+ with persistence enabled
- **Disk**: SSD recommended for database

## Docker Deployment

### Production Docker Compose

Create a `docker-compose.prod.yml`:

```yaml
services:
  postgres:
    image: postgres:16.1-alpine
    environment:
      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD}
      POSTGRES_USER: ${POSTGRES_USER}
      POSTGRES_DB: ${POSTGRES_DB}
    volumes:
      - pgdata:/var/lib/postgresql/data
    command:
      - "postgres"
      - "-c"
      - "max_connections=300"
      - "-c"
      - "shared_buffers=256MB"
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U ${POSTGRES_USER}"]
      interval: 10s
      timeout: 5s
      retries: 5

  redis:
    image: redis:7.2-alpine
    command: redis-server --appendonly yes
    volumes:
      - redisdata:/data
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 3s
      retries: 5

  api:
    build:
      context: .
      target: api
    environment:
      STORAGE_DRIVER: postgres
      BROKER_DRIVER: redis
      DATABASE_URL: postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@postgres:5432/${POSTGRES_DB}?sslmode=require&pool_max_conns=200
      REDIS_ADDR: redis:6379
      ENABLE_RATE_LIMITER: "true"
      RATE_LIMIT_PER_MINUTE: "120"
      JWT_SECRET: ${JWT_SECRET}
    ports:
      - "8080:8080"
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy
    restart: unless-stopped

  worker:
    build:
      context: .
      target: worker
    environment:
      STORAGE_DRIVER: postgres
      BROKER_DRIVER: redis
      DATABASE_URL: postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@postgres:5432/${POSTGRES_DB}?sslmode=require
      REDIS_ADDR: redis:6379
      WORKER_POOL_SIZE: 8
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy
    restart: unless-stopped
    deploy:
      replicas: 2

volumes:
  pgdata:
  redisdata:
```

### Environment Variables

Create a `.env` file:

```bash
POSTGRES_USER=taskqueue
POSTGRES_PASSWORD=your-secure-password
POSTGRES_DB=taskqueue
JWT_SECRET=your-jwt-secret-key
```

### Deploy

```bash
docker-compose -f docker-compose.prod.yml up -d
```

## Kubernetes Deployment

### Prerequisites

- Kubernetes cluster (1.20+)
- kubectl configured
- Helm 3+ (optional)

### Deployment Manifests

See `k8s/` directory for Kubernetes manifests (if created).

### Helm Chart

```bash
helm install taskqueue ./helm-chart
```

## Monitoring Setup

### Prometheus Configuration

Create `prometheus.yml`:

```yaml
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: 'taskqueue-api'
    static_configs:
      - targets: ['api:8080']
```

### Grafana Dashboards

Import dashboard from `grafana/dashboard.json` (if created).

### Alert Rules

Create `prometheus/alerts.yml`:

```yaml
groups:
  - name: taskqueue
    rules:
      - alert: HighLatency
        expr: histogram_quantile(0.95, http_request_duration_seconds_bucket) > 0.01
        for: 5m
        annotations:
          summary: "High API latency detected"

      - alert: HighFailureRate
        expr: rate(job_failures_total[5m]) > 0.01
        for: 5m
        annotations:
          summary: "High job failure rate"

      - alert: QueueDepthHigh
        expr: queue_depth > 1000
        for: 5m
        annotations:
          summary: "Queue depth is high"
```

## Security Checklist

- [ ] JWT authentication enabled (`JWT_SECRET` set)
- [ ] Rate limiting configured (`ENABLE_RATE_LIMITER=true`)
- [ ] Database credentials secured (use secrets management)
- [ ] TLS/SSL configured for API (reverse proxy with nginx/traefik)
- [ ] Database connections use SSL (`sslmode=require`)
- [ ] Redis password protected (if exposed)
- [ ] Firewall rules configured
- [ ] Regular security updates applied
- [ ] Secrets stored in secure vault (not in code)
- [ ] CORS origins restricted to known domains

## Performance Tuning

### Database Tuning

**PostgreSQL Configuration:**
```sql
-- Increase shared buffers
shared_buffers = 256MB

-- Increase work memory
work_mem = 16MB

-- Increase maintenance work memory
maintenance_work_mem = 128MB

-- Enable connection pooling
max_connections = 300
```

### Connection Pooling

**API Service:**
- `pool_max_conns=200` in DATABASE_URL
- `pool_max_idle_conns=50`

**Worker Service:**
- Default connection pool settings
- Monitor connection usage

### Worker Configuration

**Optimal Pool Size:**
- CPU cores × 2 = recommended pool size
- Example: 4 cores → 8 workers
- Monitor CPU utilization (target: 70-80%)

**Scaling Workers:**
```bash
docker-compose up -d --scale worker=3
```

## Backup & Recovery

### Database Backup

**Automated Backup Script:**
```bash
#!/bin/bash
docker-compose exec postgres pg_dump -U taskqueue taskqueue > backup_$(date +%Y%m%d).sql
```

**Schedule with Cron:**
```bash
0 2 * * * /path/to/backup.sh
```

### Redis Backup

Redis persistence is enabled with `appendonly yes`. For manual backup:

```bash
docker-compose exec redis redis-cli BGSAVE
```

### Recovery

**Restore Database:**
```bash
docker-compose exec -T postgres psql -U taskqueue taskqueue < backup_20240101.sql
```

## Troubleshooting

### Common Issues

**1. High Latency**
- Check database connection pool usage
- Monitor queue depth
- Increase worker pool size
- Check network latency

**2. High Failure Rate**
- Check worker logs for errors
- Verify job handlers are working
- Check database connectivity
- Review Redis connection

**3. Queue Depth Growing**
- Add more worker instances
- Increase worker pool size
- Check for stuck jobs
- Review job processing time

**4. Database Connection Errors**
- Check max_connections setting
- Monitor connection pool usage
- Review connection timeout settings
- Check database health

**5. Worker Not Processing Jobs**
- Check worker logs
- Verify Redis connection
- Check worker registration in database
- Review consumer group configuration

### Logs

**View API Logs:**
```bash
docker-compose logs -f api
```

**View Worker Logs:**
```bash
docker-compose logs -f worker
```

**View Database Logs:**
```bash
docker-compose logs -f postgres
```

**View Redis Logs:**
```bash
docker-compose logs -f redis
```

## Health Checks

### API Health

```bash
curl http://localhost:8080/health
```

### Database Health

```bash
docker-compose exec postgres pg_isready -U taskqueue
```

### Redis Health

```bash
docker-compose exec redis redis-cli ping
```

## Scaling

### Horizontal Scaling

**Add More API Instances:**
```bash
docker-compose up -d --scale api=2
```

**Add More Workers:**
```bash
docker-compose up -d --scale worker=5
```

### Vertical Scaling

**Increase Resources:**
- Edit `docker-compose.yml` to add resource limits
- Increase database connection limits
- Adjust worker pool sizes

## Production Checklist

- [ ] All services running and healthy
- [ ] Monitoring and alerts configured
- [ ] Backups scheduled and tested
- [ ] Security hardening applied
- [ ] Performance tuning completed
- [ ] Load testing performed
- [ ] Documentation reviewed
- [ ] Team trained on operations
- [ ] Incident response plan ready
- [ ] Rollback procedure tested

