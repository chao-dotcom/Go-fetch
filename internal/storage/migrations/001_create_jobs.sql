CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";

CREATE TYPE job_status AS ENUM (
    'QUEUED',
    'RUNNING',
    'SUCCEEDED',
    'FAILED',
    'RETRYING',
    'CANCELLED'
);

CREATE TABLE jobs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    type VARCHAR(100) NOT NULL,
    payload JSONB NOT NULL,
    status job_status NOT NULL DEFAULT 'QUEUED',
    priority INTEGER DEFAULT 0,
    attempts INTEGER DEFAULT 0,
    max_attempts INTEGER DEFAULT 3,
    timeout_seconds INTEGER DEFAULT 300,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    started_at TIMESTAMPTZ,
    finished_at TIMESTAMPTZ,
    scheduled_for TIMESTAMPTZ,
    result JSONB,
    error TEXT,
    worker_id VARCHAR(100),
    worker_hostname VARCHAR(255),
    processing_time_ms BIGINT,
    queue_time_ms BIGINT,
    idempotency_key VARCHAR(255) UNIQUE,
    metadata JSONB DEFAULT '{}'::jsonb,
    tags TEXT[] DEFAULT '{}',
    CONSTRAINT valid_attempts CHECK (attempts >= 0),
    CONSTRAINT valid_max_attempts CHECK (max_attempts >= 1 AND max_attempts <= 10),
    CONSTRAINT valid_priority CHECK (priority >= -10 AND priority <= 10)
);

CREATE INDEX idx_jobs_status ON jobs(status);
CREATE INDEX idx_jobs_type ON jobs(type);
CREATE INDEX idx_jobs_created_at ON jobs(created_at DESC);
CREATE INDEX idx_jobs_updated_at ON jobs(updated_at DESC);
CREATE INDEX idx_jobs_priority ON jobs(priority DESC) WHERE status = 'QUEUED';
CREATE INDEX idx_jobs_worker_id ON jobs(worker_id) WHERE worker_id IS NOT NULL;
CREATE INDEX idx_jobs_scheduled_for ON jobs(scheduled_for) WHERE scheduled_for IS NOT NULL AND status = 'QUEUED';
CREATE INDEX idx_jobs_status_type ON jobs(status, type);
CREATE INDEX idx_jobs_status_created ON jobs(status, created_at DESC);
CREATE INDEX idx_jobs_type_created ON jobs(type, created_at DESC);
CREATE INDEX idx_jobs_payload_gin ON jobs USING gin(payload jsonb_path_ops);
CREATE INDEX idx_jobs_metadata_gin ON jobs USING gin(metadata jsonb_path_ops);
CREATE INDEX idx_jobs_error_gin ON jobs USING gin(error gin_trgm_ops) WHERE error IS NOT NULL;
CREATE INDEX idx_jobs_active ON jobs(id) WHERE status IN ('QUEUED', 'RUNNING', 'RETRYING');

CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_jobs_updated_at
    BEFORE UPDATE ON jobs
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE OR REPLACE FUNCTION calculate_processing_time()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.finished_at IS NOT NULL AND NEW.started_at IS NOT NULL THEN
        NEW.processing_time_ms = EXTRACT(EPOCH FROM (NEW.finished_at - NEW.started_at)) * 1000;
    END IF;

    IF NEW.started_at IS NOT NULL AND NEW.created_at IS NOT NULL THEN
        NEW.queue_time_ms = EXTRACT(EPOCH FROM (NEW.started_at - NEW.created_at)) * 1000;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER calculate_job_times
    BEFORE UPDATE ON jobs
    FOR EACH ROW
    WHEN (NEW.finished_at IS NOT NULL OR NEW.started_at IS NOT NULL)
    EXECUTE FUNCTION calculate_processing_time();

