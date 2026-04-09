CREATE TABLE IF NOT EXISTS group_status_configs (
    id BIGSERIAL PRIMARY KEY,
    group_id BIGINT NOT NULL UNIQUE REFERENCES groups(id) ON DELETE CASCADE,
    enabled BOOLEAN NOT NULL DEFAULT FALSE,
    probe_model VARCHAR(255) NOT NULL DEFAULT '',
    probe_prompt TEXT NOT NULL DEFAULT '',
    validation_mode VARCHAR(32) NOT NULL DEFAULT 'non_empty',
    expected_keywords JSONB NOT NULL DEFAULT '[]'::jsonb,
    interval_seconds INTEGER NOT NULL DEFAULT 60,
    timeout_seconds INTEGER NOT NULL DEFAULT 30,
    slow_latency_ms BIGINT NOT NULL DEFAULT 15000,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS group_status_records (
    id BIGSERIAL PRIMARY KEY,
    group_id BIGINT NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    config_id BIGINT NOT NULL REFERENCES group_status_configs(id) ON DELETE CASCADE,
    status VARCHAR(32) NOT NULL,
    response_excerpt TEXT NULL,
    latency_ms BIGINT NULL,
    http_code INTEGER NULL,
    sub_status VARCHAR(64) NOT NULL DEFAULT '',
    error_detail TEXT NULL,
    observed_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS group_status_states (
    id BIGSERIAL PRIMARY KEY,
    group_id BIGINT NOT NULL UNIQUE REFERENCES groups(id) ON DELETE CASCADE,
    config_id BIGINT NOT NULL REFERENCES group_status_configs(id) ON DELETE CASCADE,
    latest_status VARCHAR(32) NOT NULL DEFAULT '',
    stable_status VARCHAR(32) NOT NULL DEFAULT '',
    response_excerpt TEXT NULL,
    latency_ms BIGINT NULL,
    http_code INTEGER NULL,
    sub_status VARCHAR(64) NOT NULL DEFAULT '',
    error_detail TEXT NULL,
    observed_at TIMESTAMPTZ NULL,
    consecutive_down INTEGER NOT NULL DEFAULT 0,
    consecutive_non_down INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS group_status_events (
    id BIGSERIAL PRIMARY KEY,
    group_id BIGINT NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    config_id BIGINT NOT NULL REFERENCES group_status_configs(id) ON DELETE CASCADE,
    event_type VARCHAR(32) NOT NULL,
    from_status VARCHAR(32) NOT NULL DEFAULT '',
    to_status VARCHAR(32) NOT NULL DEFAULT '',
    latency_ms BIGINT NULL,
    http_code INTEGER NULL,
    sub_status VARCHAR(64) NOT NULL DEFAULT '',
    error_detail TEXT NULL,
    observed_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_group_status_records_group_observed_at
    ON group_status_records(group_id, observed_at DESC);

CREATE INDEX IF NOT EXISTS idx_group_status_records_config_observed_at
    ON group_status_records(config_id, observed_at DESC);

CREATE INDEX IF NOT EXISTS idx_group_status_events_group_observed_at
    ON group_status_events(group_id, observed_at DESC);

CREATE INDEX IF NOT EXISTS idx_group_status_events_config_observed_at
    ON group_status_events(config_id, observed_at DESC);
