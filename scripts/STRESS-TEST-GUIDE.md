# Stress Testing Guide

This guide explains how to perform stress testing on the Distributed Task Queue system using k6.

## Prerequisites

1. **k6 installed** - Download from [k6.io](https://k6.io/docs/getting-started/installation/)
2. **Docker** (optional but recommended) - For running tests in isolated environment
3. **System running** - Ensure all services are up: `docker-compose up -d`

## Quick Start

### Basic Stress Test (15 VU)

```bash
k6 run scripts/quick-stress-test.js
```

### High Load Test (100 VU)

```bash
k6 run scripts/stress-test-high-load.js
```

## Test Scripts Overview

| Script | Description | Concurrency | Duration |
|--------|-------------|-------------|----------|
| `quick-stress-test.js` | Standard load test | 15 VU | ~2 minutes |
| `quick-stress-test-30.js` | Quick 30-second test | 3-5 VU | 30 seconds |
| `quick-stress-test-10.js` | Quick 10-second test | 3-10 VU | 10 seconds |
| `stress-test-high-load.js` | High load test | 100 VU | 8 minutes |
| `load-test.js` | Extended load test | 200 VU | 24 minutes |
| `fixed-stress-test.js` | Fixed connection test | 50 VU | ~5 minutes |

## Running Tests

### Method 1: Local k6 (Recommended for Development)

```bash
# Standard test
k6 run scripts/quick-stress-test.js

# High load test
k6 run scripts/stress-test-high-load.js

# Custom parameters
k6 run --vus 50 --duration 5m scripts/quick-stress-test.js
```

### Method 2: Docker (Recommended for Production-like Testing)

**Using Docker's internal network (best performance):**

```bash
# Get the Docker network name
docker network ls | grep go_work

# Run test inside Docker network
docker run --rm -i --network go_work_default \
  -v ${PWD}/scripts:/scripts \
  grafana/k6 run \
  --batch 30 --batch-per-host 30 \
  --no-connection-reuse \
  --env API_URL=http://go_work-api-1:8080/v1 \
  /scripts/stress-test-high-load.js
```

**PowerShell (Windows):**

```powershell
docker run --rm -i --network go_work_default `
  -v ${PWD}/scripts:/scripts `
  grafana/k6 run `
  --batch 30 --batch-per-host 30 `
  --no-connection-reuse `
  --env API_URL=http://go_work-api-1:8080/v1 `
  /scripts/stress-test-high-load.js
```

### Method 3: Local Network (For Testing from Host)

```bash
k6 run --env API_URL=http://localhost:8080/v1 scripts/stress-test-high-load.js
```

## Understanding Test Results

### Key Metrics

1. **HTTP Request Duration**
   - `p95`: 95th percentile response time (target: < 50ms)
   - `p99`: 99th percentile response time (target: < 100ms)
   - `avg`: Average response time

2. **HTTP Request Failed Rate**
   - Target: < 5% (0.05)
   - Our system achieves: 0.00%

3. **Job Submit Success Rate**
   - Target: > 95%
   - Our system achieves: 100%

4. **Throughput**
   - Requests per second (req/s)
   - Higher is better

### Example Output Interpretation

```
http_req_duration
  ✓ 'p(95)<50' p(95)=7.17ms  ✅ Passed (7x better than threshold)
  ✓ 'p(99)<100' p(99)=17.2ms ✅ Passed (5.8x better than threshold)

http_req_failed
  ✓ 'rate<0.05' rate=0.00%   ✅ Perfect (0 failures)

job_submit_success
  ✓ 'rate>0.95' rate=100.00% ✅ Perfect (all jobs submitted)
```

## Test Scenarios

### Scenario 1: Gradual Load Increase

Tests system behavior under gradually increasing load:

```bash
k6 run scripts/stress-test-high-load.js
```

This script:
- Starts at 15 VU
- Gradually increases to 100 VU
- Tests system stability at each level

### Scenario 2: Quick Validation

Quick test to verify system is working:

```bash
k6 run scripts/quick-stress-test-10.js
```

Runs for 10 seconds with low concurrency.

### Scenario 3: Sustained High Load

Tests system under sustained high load:

```bash
k6 run scripts/load-test.js
```

Runs for 24 minutes with up to 200 VU.

## Customizing Tests

### Adjusting Concurrency

Edit the test script's `stages`:

```javascript
export const options = {
  stages: [
    { duration: '1m', target: 10 },   // Start with 10 VU
    { duration: '2m', target: 50 },   // Ramp to 50 VU
    { duration: '2m', target: 50 },    // Hold at 50 VU
    { duration: '1m', target: 0 },     // Ramp down
  ],
  // ...
};
```

### Adjusting Thresholds

Modify threshold values:

```javascript
thresholds: {
  http_req_duration: ['p(95)<100', 'p(99)<200'],  // More lenient
  http_req_failed: ['rate<0.10'],                  // Allow 10% failure
  job_submit_success: ['rate>0.90'],               // 90% success rate
},
```

### Changing Job Types

Modify the `JOB_TYPES` array:

```javascript
const JOB_TYPES = [
  { type: 'send_email', weight: 50 },      // 50% of jobs
  { type: 'process_video', weight: 30 },   // 30% of jobs
  { type: 'generate_thumbnail', weight: 20 }, // 20% of jobs
];
```

## Troubleshooting

### High Failure Rate

If you see high failure rates:

1. **Check API is running:**
   ```bash
   curl http://localhost:8080/health
   ```

2. **Check rate limiting:**
   - Ensure `ENABLE_RATE_LIMITER=false` in docker-compose.yml
   - Or increase `RATE_LIMIT_PER_MINUTE`

3. **Check worker capacity:**
   ```bash
   docker-compose ps worker
   curl http://localhost:8080/v1/workers
   ```

4. **Increase worker count:**
   ```bash
   docker-compose up -d --scale worker=3
   ```

### Connection Errors

If you see connection errors:

1. **Use Docker network:**
   - Run k6 inside Docker network
   - Use service names instead of localhost

2. **Check Docker network:**
   ```bash
   docker network inspect go_work_default
   ```

3. **Increase timeouts:**
   - Edit script: `timeout: '30s'` in http.post calls

### Slow Response Times

If response times are high:

1. **Check database connections:**
   ```bash
   docker-compose exec postgres psql -U taskqueue -d taskqueue -c "SELECT count(*) FROM pg_stat_activity;"
   ```

2. **Check queue depth:**
   ```bash
   curl http://localhost:8080/v1/queues
   ```

3. **Increase worker pool size:**
   - Set `WORKER_POOL_SIZE=16` in docker-compose.yml
   - Restart workers: `docker-compose restart worker`

## Best Practices

1. **Start Small**: Begin with low concurrency (5-10 VU) and gradually increase
2. **Monitor Resources**: Watch CPU, memory, and database connections during tests
3. **Use Docker Network**: For most accurate results, run k6 in Docker network
4. **Check Logs**: Monitor API and worker logs during tests
5. **Warm Up**: Allow system to warm up before starting high-load tests

## Performance Targets

Based on our test results, the system should achieve:

- **p95 Latency**: < 10ms (we achieve 7.17ms)
- **p99 Latency**: < 20ms (we achieve 17.2ms)
- **Failure Rate**: < 0.1% (we achieve 0.00%)
- **Throughput**: > 100 req/s (we achieve 116 req/s)

## Advanced: Custom Test Script

Create your own test script:

```javascript
import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  stages: [
    { duration: '30s', target: 10 },
    { duration: '1m', target: 50 },
    { duration: '30s', target: 0 },
  ],
  thresholds: {
    http_req_duration: ['p(95)<50'],
    http_req_failed: ['rate<0.05'],
  },
};

const BASE_URL = __ENV.API_URL || 'http://localhost:8080/v1';

export default function () {
  const res = http.post(
    `${BASE_URL}/jobs`,
    JSON.stringify({
      type: 'send_email',
      payload: { to: 'test@example.com', subject: 'Test', body: 'Test' },
    }),
    { headers: { 'Content-Type': 'application/json' } }
  );
  
  check(res, {
    'status is 202': (r) => r.status === 202,
  });
  
  sleep(1);
}
```

Save as `scripts/custom-test.js` and run:

```bash
k6 run scripts/custom-test.js
```

## See Also

- [k6 Documentation](https://k6.io/docs/)
- [Performance Analysis](docs/performance-analysis.md)
- [Chinese Stress Test Guide](docs/stress-test-zh.md)

