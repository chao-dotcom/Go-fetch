CREATE TYPE log_level AS ENUM ('debug', 'info', 'warn', 'error');

CREATE TABLE job_logs (
    id BIGSERIAL PRIMARY KEY,
    job_id UUID NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    level log_level NOT NULL DEFAULT 'info',
    message TEXT NOT NULL,
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    trace_id VARCHAR(100),
    span_id VARCHAR(100),
    worker_id VARCHAR(100),
    metadata JSONB DEFAULT '{}'::jsonb
);

CREATE INDEX idx_job_logs_job_id ON job_logs(job_id, timestamp DESC);
CREATE INDEX idx_job_logs_level ON job_logs(level);
CREATE INDEX idx_job_logs_timestamp ON job_logs(timestamp DESC);
CREATE INDEX idx_job_logs_trace_id ON job_logs(trace_id) WHERE trace_id IS NOT NULL;
CREATE INDEX idx_job_logs_metadata_gin ON job_logs USING gin(metadata jsonb_path_ops);

CREATE TABLE job_logs_y2024m01 PARTITION OF job_logs
    FOR VALUES FROM ('2024-01-01') TO ('2024-02-01');

CREATE OR REPLACE FUNCTION create_log_partition()
RETURNS void AS $$
DECLARE
    partition_date DATE;
    partition_name TEXT;
    start_date TEXT;
    end_date TEXT;
BEGIN
    partition_date := DATE_TRUNC('month', CURRENT_DATE);
    partition_name := 'job_logs_y' || TO_CHAR(partition_date, 'YYYY') || 'm' || TO_CHAR(partition_date, 'MM');
    start_date := partition_date::TEXT;
    end_date := (partition_date + INTERVAL '1 month')::TEXT;

    IF NOT EXISTS (SELECT 1 FROM pg_class WHERE relname = partition_name) THEN
        EXECUTE format(
            'CREATE TABLE %I PARTITION OF job_logs FOR VALUES FROM (%L) TO (%L)',
            partition_name, start_date, end_date
        );
    END IF;
END;
$$ LANGUAGE plpgsql;

