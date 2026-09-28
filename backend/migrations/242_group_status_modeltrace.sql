-- 分组运行状态：ModelTrace 指纹验证（数字分布指纹），本 fork 自有功能，替换纯 Sol 验证（Juice）。
-- OpenAI / Anthropic 分组可开启；管理员按分组选择预期模型，按 modeltrace_interval_seconds 低频运行。
--
-- 235 的 sol_juice_* 列与 group_status_juice_records 表不再读写，但刻意保留：
-- SKIP_SETUP 实例与镜像回滚要求「新 schema 配旧镜像」仍能工作，旧镜像的 SELECT 仍引用这些列。
-- 等所有副本都升级到本镜像之后，再用单独的清理迁移删除。
ALTER TABLE group_status_configs
    ADD COLUMN IF NOT EXISTS modeltrace_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS modeltrace_expected_model VARCHAR(255) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS modeltrace_request_model VARCHAR(255) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS modeltrace_interval_seconds INTEGER NOT NULL DEFAULT 3600;

ALTER TABLE group_status_states
    ADD COLUMN IF NOT EXISTS modeltrace_verdict VARCHAR(32) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS modeltrace_stable_status VARCHAR(32) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS modeltrace_run_expected_model VARCHAR(255) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS modeltrace_top_model VARCHAR(255) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS modeltrace_top_probability DOUBLE PRECISION NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS modeltrace_expected_probability DOUBLE PRECISION NULL,
    ADD COLUMN IF NOT EXISTS modeltrace_ranking JSONB NOT NULL DEFAULT '[]'::jsonb,
    ADD COLUMN IF NOT EXISTS modeltrace_reasons JSONB NOT NULL DEFAULT '[]'::jsonb,
    ADD COLUMN IF NOT EXISTS modeltrace_detail TEXT NULL,
    ADD COLUMN IF NOT EXISTS modeltrace_checked_at TIMESTAMPTZ NULL,
    ADD COLUMN IF NOT EXISTS modeltrace_consecutive_mismatch INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS modeltrace_valid_outputs INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS modeltrace_input_tokens BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS modeltrace_output_tokens BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS modeltrace_reasoning_tokens BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS modeltrace_last_cost_usd DOUBLE PRECISION NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS modeltrace_last_run_id BIGINT NULL;

CREATE TABLE IF NOT EXISTS group_status_modeltrace_runs (
    id BIGSERIAL PRIMARY KEY,
    group_id BIGINT NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    config_id BIGINT NOT NULL REFERENCES group_status_configs(id) ON DELETE CASCADE,
    platform VARCHAR(32) NOT NULL DEFAULT '',
    bank_sha256 VARCHAR(64) NOT NULL DEFAULT '',
    bank_built_at VARCHAR(64) NOT NULL DEFAULT '',
    expected_model VARCHAR(255) NOT NULL DEFAULT '',
    request_model VARCHAR(255) NOT NULL DEFAULT '',
    account_id BIGINT NULL,
    account_type VARCHAR(32) NOT NULL DEFAULT '',
    round INTEGER NOT NULL DEFAULT 1,
    verdict VARCHAR(32) NOT NULL,
    outcome VARCHAR(32) NOT NULL DEFAULT '',
    top_model VARCHAR(255) NOT NULL DEFAULT '',
    top_probability DOUBLE PRECISION NOT NULL DEFAULT 0,
    expected_probability DOUBLE PRECISION NULL,
    calibration_queries INTEGER NOT NULL DEFAULT 0,
    beta DOUBLE PRECISION NOT NULL DEFAULT 0,
    ranking JSONB NOT NULL DEFAULT '[]'::jsonb,
    family_probabilities JSONB NOT NULL DEFAULT '[]'::jsonb,
    reasons JSONB NOT NULL DEFAULT '[]'::jsonb,
    outputs JSONB NOT NULL DEFAULT '[]'::jsonb,
    attempts_planned INTEGER NOT NULL DEFAULT 0,
    attempts_made INTEGER NOT NULL DEFAULT 0,
    valid_outputs INTEGER NOT NULL DEFAULT 0,
    input_tokens BIGINT NOT NULL DEFAULT 0,
    output_tokens BIGINT NOT NULL DEFAULT 0,
    reasoning_tokens BIGINT NOT NULL DEFAULT 0,
    cost_usd DOUBLE PRECISION NOT NULL DEFAULT 0,
    latency_ms BIGINT NULL,
    http_code INTEGER NULL,
    error_detail TEXT NULL,
    started_at TIMESTAMPTZ NOT NULL,
    finished_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_group_status_modeltrace_runs_group_finished_at
    ON group_status_modeltrace_runs(group_id, finished_at DESC);

-- 已开启纯 Sol 验证的 OpenAI 分组改由 ModelTrace 接手，预期模型 gpt-5.6-sol；
-- 自定义过的 Juice 请求模型沿用为 ModelTrace 请求模型。同时关掉旧开关，避免未升级的副本继续跑 Juice。
UPDATE group_status_configs c
SET modeltrace_enabled = TRUE,
    modeltrace_expected_model = 'gpt-5.6-sol',
    modeltrace_request_model = CASE
        WHEN c.sol_juice_model IN ('', 'gpt-5.6-sol') THEN ''
        ELSE c.sol_juice_model
    END,
    modeltrace_interval_seconds = GREATEST(c.sol_juice_interval_seconds, 3600),
    sol_juice_enabled = FALSE
WHERE c.sol_juice_enabled = TRUE
  AND c.group_id IN (SELECT id FROM groups WHERE platform = 'openai');

COMMENT ON COLUMN group_status_configs.modeltrace_enabled IS
    'Whether the low-frequency ModelTrace number-fingerprint check runs for this group (OpenAI / Anthropic groups)';
COMMENT ON COLUMN group_status_configs.modeltrace_expected_model IS
    'Model the ModelTrace check expects this group to serve (fingerprint bank id)';
COMMENT ON COLUMN group_status_states.modeltrace_stable_status IS
    'Stable ModelTrace fingerprint verdict: empty | pass | mismatch';
COMMENT ON COLUMN group_status_configs.sol_juice_enabled IS
    'Dormant since 242: the Sol Juice probe was replaced by the ModelTrace check; kept for older images';
COMMENT ON TABLE group_status_juice_records IS
    'Dormant since 242: Sol Juice samples are no longer written; kept for older images';
