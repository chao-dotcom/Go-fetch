CREATE OR REPLACE VIEW active_jobs AS
SELECT 
    j.*,
    w.hostname AS worker_hostname,
    w.status AS worker_status,
    EXTRACT(EPOCH FROM (NOW() - j.created_at)) * 1000 AS age_ms
FROM jobs j
LEFT JOIN workers w ON j.worker_id = w.id
WHERE j.status IN ('QUEUED', 'RUNNING', 'RETRYING');

CREATE OR REPLACE VIEW job_stats_by_type AS
SELECT 
    type,
    COUNT(*) AS total_jobs,
    COUNT(*) FILTER (WHERE status = 'SUCCEEDED') AS succeeded,
    COUNT(*) FILTER (WHERE status = 'FAILED') AS failed,
    COUNT(*) FILTER (WHERE status IN ('QUEUED', 'RUNNING', 'RETRYING')) AS in_progress,
    AVG(processing_time_ms) AS avg_processing_time_ms,
    PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY processing_time_ms) AS p50_processing_time_ms,
    PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY processing_time_ms) AS p95_processing_time_ms,
    PERCENTILE_CONT(0.99) WITHIN GROUP (ORDER BY processing_time_ms) AS p99_processing_time_ms
FROM jobs
WHERE finished_at > NOW() - INTERVAL '24 hours'
GROUP BY type;

CREATE OR REPLACE VIEW worker_performance AS
SELECT 
    w.id,
    w.hostname,
    w.status,
    w.active_jobs,
    w.total_processed,
    w.total_failed,
    ROUND(100.0 * w.total_failed / NULLIF(w.total_processed, 0), 2) AS failure_rate_percent,
    EXTRACT(EPOCH FROM (NOW() - w.started_at)) / 3600 AS uptime_hours,
    ROUND(w.total_processed / NULLIF(EXTRACT(EPOCH FROM (NOW() - w.started_at)) / 3600, 0), 2) AS jobs_per_hour
FROM workers w
WHERE w.status <> 'dead';

GRANT SELECT ON active_jobs TO taskqueue;
GRANT SELECT ON job_stats_by_type TO taskqueue;
GRANT SELECT ON worker_performance TO taskqueue;

