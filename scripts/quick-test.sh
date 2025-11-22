#!/bin/bash
# Quick test script for the task queue system

set -e

API_URL="${API_URL:-http://localhost:8080/v1}"
JWT_TOKEN="${JWT_TOKEN:-}"

echo "🧪 Quick Test Script"
echo "==================="
echo "API URL: $API_URL"
echo ""

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Helper function
check_response() {
    local response=$1
    local expected_status=$2
    local test_name=$3
    
    if echo "$response" | grep -q "\"status\":\"$expected_status\"" || [ "$expected_status" = "200" ]; then
        echo -e "${GREEN}✅ $test_name${NC}"
        return 0
    else
        echo -e "${RED}❌ $test_name${NC}"
        echo "Response: $response"
        return 1
    fi
}

# Build auth header
AUTH_HEADER=""
if [ -n "$JWT_TOKEN" ]; then
    AUTH_HEADER="-H 'Authorization: Bearer $JWT_TOKEN'"
fi

# 1. Health Check
echo "1. Testing health endpoint..."
HEALTH_RESPONSE=$(curl -s "$API_URL/../health" || curl -s "http://localhost:8080/health")
if echo "$HEALTH_RESPONSE" | grep -q "ok"; then
    echo -e "${GREEN}✅ Health check passed${NC}"
else
    echo -e "${RED}❌ Health check failed${NC}"
    echo "Response: $HEALTH_RESPONSE"
    exit 1
fi

# 2. Submit a job
echo ""
echo "2. Submitting test job..."
JOB_RESPONSE=$(curl -s -X POST "$API_URL/jobs" \
    -H "Content-Type: application/json" \
    $AUTH_HEADER \
    -d '{
        "type": "send_email",
        "payload": {
            "to": "test@example.com",
            "subject": "Test Email",
            "body": "This is a test email from the quick test script"
        },
        "priority": 0,
        "max_attempts": 3
    }')

JOB_ID=$(echo "$JOB_RESPONSE" | grep -o '"id":"[^"]*"' | cut -d'"' -f4 || echo "")

if [ -z "$JOB_ID" ]; then
    echo -e "${RED}❌ Failed to submit job${NC}"
    echo "Response: $JOB_RESPONSE"
    exit 1
else
    echo -e "${GREEN}✅ Job submitted successfully${NC}"
    echo "   Job ID: $JOB_ID"
fi

# 3. Wait a bit for processing
echo ""
echo "3. Waiting for job processing (5 seconds)..."
sleep 5

# 4. Check job status
echo ""
echo "4. Checking job status..."
JOB_STATUS_RESPONSE=$(curl -s "$API_URL/jobs/$JOB_ID" $AUTH_HEADER)
JOB_STATUS=$(echo "$JOB_STATUS_RESPONSE" | grep -o '"status":"[^"]*"' | cut -d'"' -f4 || echo "")

if [ -n "$JOB_STATUS" ]; then
    echo -e "${GREEN}✅ Job status retrieved${NC}"
    echo "   Status: $JOB_STATUS"
else
    echo -e "${YELLOW}⚠️  Could not retrieve job status${NC}"
fi

# 5. List jobs
echo ""
echo "5. Listing jobs..."
JOBS_RESPONSE=$(curl -s "$API_URL/jobs?limit=5" $AUTH_HEADER)
JOB_COUNT=$(echo "$JOBS_RESPONSE" | grep -o '"jobs"' | wc -l || echo "0")

if [ "$JOB_COUNT" -gt 0 ] || echo "$JOBS_RESPONSE" | grep -q "jobs"; then
    echo -e "${GREEN}✅ Jobs list retrieved${NC}"
else
    echo -e "${YELLOW}⚠️  Could not retrieve jobs list${NC}"
fi

# 6. List workers
echo ""
echo "6. Listing workers..."
WORKERS_RESPONSE=$(curl -s "$API_URL/workers" $AUTH_HEADER)
if echo "$WORKERS_RESPONSE" | grep -q "\[" || echo "$WORKERS_RESPONSE" | grep -q "id"; then
    echo -e "${GREEN}✅ Workers list retrieved${NC}"
else
    echo -e "${YELLOW}⚠️  Could not retrieve workers list${NC}"
fi

# 7. Get queue stats
echo ""
echo "7. Getting queue statistics..."
QUEUE_RESPONSE=$(curl -s "$API_URL/queues" $AUTH_HEADER)
if echo "$QUEUE_RESPONSE" | grep -q "\[" || echo "$QUEUE_RESPONSE" | grep -q "name"; then
    echo -e "${GREEN}✅ Queue stats retrieved${NC}"
else
    echo -e "${YELLOW}⚠️  Could not retrieve queue stats${NC}"
fi

# 8. Check metrics
echo ""
echo "8. Checking Prometheus metrics..."
METRICS_RESPONSE=$(curl -s "http://localhost:8080/metrics" 2>/dev/null || echo "")
if echo "$METRICS_RESPONSE" | grep -q "job_processing" || echo "$METRICS_RESPONSE" | grep -q "# HELP"; then
    echo -e "${GREEN}✅ Metrics endpoint accessible${NC}"
else
    echo -e "${YELLOW}⚠️  Metrics endpoint not accessible${NC}"
fi

echo ""
echo "==================="
echo -e "${GREEN}✅ Quick test completed!${NC}"
echo ""
echo "To view job details:"
echo "  curl $API_URL/jobs/$JOB_ID $AUTH_HEADER"
echo ""
echo "To view job logs:"
echo "  curl $API_URL/jobs/$JOB_ID/logs $AUTH_HEADER"

