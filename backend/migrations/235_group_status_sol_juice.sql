-- 分组运行状态：纯 Sol 验证（Juice 指纹探测），本 fork 自有功能。
-- 仅 OpenAI 分组可开启；与 60s 存活探测独立，按 sol_juice_interval_seconds 低频运行。
ALTER TABLE group_status_configs
    ADD COLUMN IF NOT EXISTS sol_juice_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS sol_juice_interval_seconds INTEGER NOT NULL DEFAULT 900,
    ADD COLUMN IF NOT EXISTS sol_juice_model VARCHAR(255) NOT NULL DEFAULT 'gpt-5.6-sol';

ALTER TABLE group_status_states
    ADD COLUMN IF NOT EXISTS sol_juice_status VARCHAR(32) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS sol_juice_stable_status VARCHAR(32) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS sol_juice_value VARCHAR(64) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS sol_juice_detail TEXT NULL,
    ADD COLUMN IF NOT EXISTS sol_juice_checked_at TIMESTAMPTZ NULL,
    ADD COLUMN IF NOT EXISTS sol_juice_consecutive_mismatch INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS sol_juice_input_tokens BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS sol_juice_output_tokens BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS sol_juice_reasoning_tokens BIGINT NOT NULL DEFAULT 0;

CREATE TABLE IF NOT EXISTS group_status_juice_records (
    id BIGSERIAL PRIMARY KEY,
    group_id BIGINT NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    config_id BIGINT NOT NULL REFERENCES group_status_configs(id) ON DELETE CASCADE,
    model VARCHAR(255) NOT NULL DEFAULT '',
    effort VARCHAR(16) NOT NULL DEFAULT 'high',
    classification VARCHAR(32) NOT NULL,
    normalized_value VARCHAR(64) NOT NULL DEFAULT '',
    answer_excerpt TEXT NULL,
    http_code INTEGER NULL,
    latency_ms BIGINT NULL,
    input_tokens BIGINT NOT NULL DEFAULT 0,
    output_tokens BIGINT NOT NULL DEFAULT 0,
    reasoning_tokens BIGINT NOT NULL DEFAULT 0,
    error_detail TEXT NULL,
    observed_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_group_status_juice_records_group_observed_at
    ON group_status_juice_records(group_id, observed_at DESC);

COMMENT ON COLUMN group_status_configs.sol_juice_enabled IS
    'Whether the low-frequency Sol Juice identity probe runs for this group (OpenAI groups only)';
COMMENT ON COLUMN group_status_states.sol_juice_stable_status IS
    'Stable Sol Juice verdict: empty | pass | mismatch';
