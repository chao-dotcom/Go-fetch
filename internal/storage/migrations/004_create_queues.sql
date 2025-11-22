CREATE TABLE queues (
    name VARCHAR(100) PRIMARY KEY,
    description TEXT,
    priority INTEGER DEFAULT 0,
    depth BIGINT DEFAULT 0,
    total_processed BIGINT DEFAULT 0,
    total_failed BIGINT DEFAULT 0,
    avg_processing_time_ms DECIMAL(10,2),
    p50_processing_time_ms BIGINT,
    p95_processing_time_ms BIGINT,
    p99_processing_time_ms BIGINT,
    max_jobs_per_second INTEGER,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    config JSONB DEFAULT '{}'::jsonb,
    CONSTRAINT valid_queue_priority CHECK (priority >= -10 AND priority <= 10)
);

INSERT INTO queues (name, description, priority) VALUES
    ('default', 'Default job queue', 0),
    ('high_priority', 'High priority jobs', 10),
    ('low_priority', 'Low priority background jobs', -5)
ON CONFLICT DO NOTHING;

CREATE INDEX idx_queues_priority ON queues(priority DESC);

CREATE OR REPLACE FUNCTION update_queue_stats()
RETURNS TRIGGER AS $$
DECLARE
    queue_name TEXT;
BEGIN
    queue_name := COALESCE(NEW.metadata->>'queue', 'default');

    IF TG_OP = 'INSERT' AND NEW.status = 'QUEUED' THEN
        UPDATE queues SET depth = depth + 1 WHERE name = queue_name;
    ELSIF TG_OP = 'UPDATE' AND OLD.status = 'QUEUED' AND NEW.status != 'QUEUED' THEN
        UPDATE queues SET depth = GREATEST(depth - 1, 0) WHERE name = queue_name;
    END IF;

    IF TG_OP = 'UPDATE' AND NEW.status = 'SUCCEEDED' AND OLD.status != 'SUCCEEDED' THEN
        UPDATE queues
        SET total_processed = total_processed + 1,
            updated_at = NOW()
        WHERE name = queue_name;
    ELSIF TG_OP = 'UPDATE' AND NEW.status = 'FAILED' AND OLD.status != 'FAILED' THEN
        UPDATE queues
        SET total_failed = total_failed + 1,
            updated_at = NOW()
        WHERE name = queue_name;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_queue_stats_trigger
    AFTER INSERT OR UPDATE ON jobs
    FOR EACH ROW
    EXECUTE FUNCTION update_queue_stats();

