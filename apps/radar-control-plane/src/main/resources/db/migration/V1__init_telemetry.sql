CREATE TABLE IF NOT EXISTS telemetry_logs (
    id BIGSERIAL PRIMARY KEY,
    query_id VARCHAR(64) NOT NULL,
    prompt TEXT NOT NULL,
    cache_hit BOOLEAN NOT NULL,
    similarity REAL,
    threshold REAL,
    latency_ms BIGINT NOT NULL,
    baseline_ms BIGINT DEFAULT 1250,
    tokens_input INT,
    tokens_output INT,
    tokens_saved INT DEFAULT 0,
    cost_saved DOUBLE PRECISION DEFAULT 0.0,
    model VARCHAR(64),
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_telemetry_timestamp ON telemetry_logs (timestamp);
CREATE INDEX IF NOT EXISTS idx_telemetry_cache_hit ON telemetry_logs (cache_hit);
