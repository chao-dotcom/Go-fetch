CREATE TYPE worker_status AS ENUM ('active', 'idle', 'dead', 'draining');

CREATE TABLE workers (
    id VARCHAR(100) PRIMARY KEY,
    hostname VARCHAR(255) NOT NULL,
    status worker_status NOT NULL DEFAULT 'active',
    pool_size INTEGER NOT NULL,
    queues TEXT[] NOT NULL DEFAULT '{"default"}',
    active_jobs INTEGER DEFAULT 0,
    total_processed BIGINT DEFAULT 0,
    total_failed BIGINT DEFAULT 0,
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_heartbeat TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    stopped_at TIMESTAMPTZ,
    version VARCHAR(50),
    metadata JSONB DEFAULT '{}'::jsonb,
    CONSTRAINT valid_pool_size CHECK (pool_size > 0 AND pool_size <= 256),
    CONSTRAINT valid_active_jobs CHECK (active_jobs >= 0)
);

CREATE INDEX idx_workers_status ON workers(status);
CREATE INDEX idx_workers_last_heartbeat ON workers(last_heartbeat DESC);
CREATE INDEX idx_workers_started_at ON workers(started_at DESC);

CREATE OR REPLACE FUNCTION mark_dead_workers()
RETURNS void AS $$
BEGIN
    UPDATE workers
    SET status = 'dead'
    WHERE status IN ('active', 'idle')
      AND last_heartbeat < NOW() - INTERVAL '30 seconds';
END;
$$ LANGUAGE plpgsql;

