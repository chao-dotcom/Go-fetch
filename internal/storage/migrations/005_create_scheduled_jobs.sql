CREATE TABLE scheduled_jobs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL UNIQUE,
    type VARCHAR(100) NOT NULL,
    payload JSONB NOT NULL,
    cron_expression VARCHAR(100),
    interval_seconds INTEGER,
    enabled BOOLEAN DEFAULT true,
    last_run_at TIMESTAMPTZ,
    next_run_at TIMESTAMPTZ NOT NULL,
    last_job_id UUID REFERENCES jobs(id),
    max_attempts INTEGER DEFAULT 3,
    timeout_seconds INTEGER DEFAULT 300,
    metadata JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT valid_schedule CHECK (
        (cron_expression IS NOT NULL AND interval_seconds IS NULL) OR
        (cron_expression IS NULL AND interval_seconds IS NOT NULL)
    )
);

CREATE INDEX idx_scheduled_jobs_next_run ON scheduled_jobs(next_run_at) WHERE enabled = true;
CREATE INDEX idx_scheduled_jobs_enabled ON scheduled_jobs(enabled);

