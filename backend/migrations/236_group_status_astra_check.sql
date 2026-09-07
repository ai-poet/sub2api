-- 分组运行状态：Astra 指纹验证（meow 基准，行为指纹），本 fork 自有功能。
-- 与 Sol Juice 探针并列、字段独立；仅 OpenAI 分组可开启，按 astra_check_interval_seconds 低频运行。
ALTER TABLE group_status_configs
    ADD COLUMN IF NOT EXISTS astra_check_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS astra_check_request_model VARCHAR(255) NOT NULL DEFAULT 'gpt-6-astra',
    ADD COLUMN IF NOT EXISTS astra_check_tier VARCHAR(16) NOT NULL DEFAULT 'low',
    ADD COLUMN IF NOT EXISTS astra_check_interval_seconds INTEGER NOT NULL DEFAULT 3600;

ALTER TABLE group_status_states
    ADD COLUMN IF NOT EXISTS astra_check_verdict VARCHAR(32) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS astra_check_stable_status VARCHAR(32) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS astra_check_winner VARCHAR(255) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS astra_check_matches JSONB NOT NULL DEFAULT '[]'::jsonb,
    ADD COLUMN IF NOT EXISTS astra_check_reasons JSONB NOT NULL DEFAULT '[]'::jsonb,
    ADD COLUMN IF NOT EXISTS astra_check_detail TEXT NULL,
    ADD COLUMN IF NOT EXISTS astra_check_checked_at TIMESTAMPTZ NULL,
    ADD COLUMN IF NOT EXISTS astra_check_consecutive_mismatch INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS astra_check_valid_samples INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS astra_check_planned_samples INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS astra_check_input_tokens BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS astra_check_output_tokens BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS astra_check_reasoning_tokens BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS astra_check_last_run_id BIGINT NULL;

CREATE TABLE IF NOT EXISTS group_status_astra_check_runs (
    id BIGSERIAL PRIMARY KEY,
    group_id BIGINT NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    config_id BIGINT NOT NULL REFERENCES group_status_configs(id) ON DELETE CASCADE,
    benchmark_package_id VARCHAR(255) NOT NULL DEFAULT '',
    benchmark_version VARCHAR(64) NOT NULL DEFAULT '',
    benchmark_sha256 VARCHAR(64) NOT NULL DEFAULT '',
    request_model VARCHAR(255) NOT NULL DEFAULT '',
    tier VARCHAR(16) NOT NULL DEFAULT 'low',
    account_id BIGINT NULL,
    verdict VARCHAR(32) NOT NULL,
    winner_model VARCHAR(255) NOT NULL DEFAULT '',
    matches JSONB NOT NULL DEFAULT '[]'::jsonb,
    cells JSONB NOT NULL DEFAULT '[]'::jsonb,
    reasons JSONB NOT NULL DEFAULT '[]'::jsonb,
    requests_planned INTEGER NOT NULL DEFAULT 0,
    requests_completed INTEGER NOT NULL DEFAULT 0,
    valid_samples INTEGER NOT NULL DEFAULT 0,
    input_tokens BIGINT NOT NULL DEFAULT 0,
    output_tokens BIGINT NOT NULL DEFAULT 0,
    reasoning_tokens BIGINT NOT NULL DEFAULT 0,
    latency_ms BIGINT NULL,
    http_code INTEGER NULL,
    error_detail TEXT NULL,
    started_at TIMESTAMPTZ NOT NULL,
    finished_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_group_status_astra_check_runs_group_finished_at
    ON group_status_astra_check_runs(group_id, finished_at DESC);

COMMENT ON COLUMN group_status_configs.astra_check_enabled IS
    'Whether the low-frequency Astra behavioral-fingerprint check runs for this group (OpenAI groups only)';
COMMENT ON COLUMN group_status_states.astra_check_stable_status IS
    'Stable Astra fingerprint verdict: empty | pass | mismatch';
