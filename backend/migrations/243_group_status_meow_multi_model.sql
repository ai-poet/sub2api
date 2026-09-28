-- 分组运行状态：meow 指纹验证改为多模型（本 fork 自有功能），并统一取代 ModelTrace 与纯 Sol 验证。
--
-- 历史名称是 Astra 指纹验证，表与列沿用 astra_check：
--   * group_status_configs.astra_check_models：分组要检测的预期模型列表 [{expected_model, request_model}]，
--     共用 astra_check_tier 与 astra_check_interval_seconds；
--   * group_status_astra_check_states：每个（分组, 预期模型）各自的最近结果与稳定结论；
--   * group_status_astra_check_runs 补 expected_model 等列，一次运行对应一个预期模型。
--
-- 以下列 / 表从本迁移起不再读写，但刻意保留：SKIP_SETUP 实例与镜像回滚要求「新 schema 配旧镜像」仍能工作。
--   * group_status_configs.astra_check_request_model、group_status_states.astra_check_*（单模型 Astra）；
--   * 242 的 modeltrace_* 列与 group_status_modeltrace_runs（ModelTrace）；
--   * 235 的 sol_juice_* 列与 group_status_juice_records（纯 Sol 验证，242 已关闭开关）。
-- 等所有副本都升级到本镜像之后，再用单独的清理迁移删除。
ALTER TABLE group_status_configs
    ADD COLUMN IF NOT EXISTS astra_check_models JSONB NOT NULL DEFAULT '[]'::jsonb;

CREATE TABLE IF NOT EXISTS group_status_astra_check_states (
    id BIGSERIAL PRIMARY KEY,
    group_id BIGINT NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    config_id BIGINT NOT NULL REFERENCES group_status_configs(id) ON DELETE CASCADE,
    expected_model VARCHAR(255) NOT NULL,
    verdict VARCHAR(32) NOT NULL DEFAULT '',
    stable_status VARCHAR(32) NOT NULL DEFAULT '',
    winner_model VARCHAR(255) NOT NULL DEFAULT '',
    matches JSONB NOT NULL DEFAULT '[]'::jsonb,
    reasons JSONB NOT NULL DEFAULT '[]'::jsonb,
    detail TEXT NULL,
    checked_at TIMESTAMPTZ NULL,
    consecutive_mismatch INTEGER NOT NULL DEFAULT 0,
    valid_samples INTEGER NOT NULL DEFAULT 0,
    planned_samples INTEGER NOT NULL DEFAULT 0,
    input_tokens BIGINT NOT NULL DEFAULT 0,
    output_tokens BIGINT NOT NULL DEFAULT 0,
    reasoning_tokens BIGINT NOT NULL DEFAULT 0,
    last_cost_usd DOUBLE PRECISION NOT NULL DEFAULT 0,
    last_run_id BIGINT NULL,
    benchmark_package_id VARCHAR(255) NOT NULL DEFAULT '',
    benchmark_version VARCHAR(64) NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_group_status_astra_check_states_group_model
    ON group_status_astra_check_states(group_id, expected_model);

ALTER TABLE group_status_astra_check_runs
    ADD COLUMN IF NOT EXISTS platform VARCHAR(32) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS expected_model VARCHAR(255) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS round INTEGER NOT NULL DEFAULT 1,
    ADD COLUMN IF NOT EXISTS scoring_version VARCHAR(64) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS cost_usd DOUBLE PRECISION NOT NULL DEFAULT 0;

-- 旧的单模型运行记录都是 GPT-6 Astra（v2 评分）
UPDATE group_status_astra_check_runs
SET expected_model = 'gpt-6-astra',
    platform = 'openai',
    scoring_version = 'meow-fingerprint-v2'
WHERE expected_model = '';

CREATE INDEX IF NOT EXISTS idx_group_status_astra_check_runs_group_model_finished_at
    ON group_status_astra_check_runs(group_id, expected_model, finished_at DESC);

-- 1) 已开启的单模型 Astra 检测：转成只含 GPT-6 Astra 的列表，自定义过的请求模型沿用
UPDATE group_status_configs
SET astra_check_models = jsonb_build_array(jsonb_build_object(
        'expected_model', 'gpt-6-astra',
        'request_model', CASE
            WHEN astra_check_request_model IN ('', 'gpt-6-astra') THEN ''
            ELSE astra_check_request_model
        END
    ))
WHERE astra_check_enabled = TRUE
  AND astra_check_models = '[]'::jsonb;

-- 2) 已开启的 ModelTrace（含 242 从纯 Sol 验证迁过来的分组）：把它的预期模型并入列表并开启指纹验证；
--    ModelTrace 的 claude-opus-5-5 对应 meow 的 claude-opus-5.5。未开启 Astra 的分组沿用 ModelTrace 的间隔（不低于 900 秒）。
UPDATE group_status_configs c
SET astra_check_models = c.astra_check_models || jsonb_build_array(jsonb_build_object(
        'expected_model', m.expected_model,
        'request_model', c.modeltrace_request_model
    )),
    astra_check_interval_seconds = CASE
        WHEN c.astra_check_enabled THEN c.astra_check_interval_seconds
        ELSE GREATEST(c.modeltrace_interval_seconds, 900)
    END,
    astra_check_enabled = TRUE
FROM (
    SELECT id,
           CASE modeltrace_expected_model
               WHEN 'claude-opus-5-5' THEN 'claude-opus-5.5'
               ELSE modeltrace_expected_model
           END AS expected_model
    FROM group_status_configs
) m
WHERE m.id = c.id
  AND c.modeltrace_enabled = TRUE
  AND m.expected_model IN ('gpt-5.6-sol', 'gpt-6-sol', 'gpt-6-astra', 'claude-opus-5.5')
  AND NOT EXISTS (
      SELECT 1 FROM jsonb_array_elements(c.astra_check_models) e
      WHERE e->>'expected_model' = m.expected_model
  );

-- 3) ModelTrace 一律关闭，避免未升级的副本继续跑
UPDATE group_status_configs
SET modeltrace_enabled = FALSE
WHERE modeltrace_enabled = TRUE;

COMMENT ON COLUMN group_status_configs.astra_check_models IS
    'Models the meow fingerprint check expects this group to serve: [{expected_model, request_model}]';
COMMENT ON COLUMN group_status_astra_check_states.stable_status IS
    'Stable meow fingerprint verdict for this (group, expected model): empty | pass | mismatch';
COMMENT ON COLUMN group_status_configs.astra_check_request_model IS
    'Dormant since 243: replaced by astra_check_models; kept for older images';
COMMENT ON COLUMN group_status_states.astra_check_stable_status IS
    'Dormant since 243: per-model state lives in group_status_astra_check_states; kept for older images';
COMMENT ON COLUMN group_status_configs.modeltrace_enabled IS
    'Dormant since 243: the ModelTrace check was folded into the meow fingerprint check; kept for older images';
COMMENT ON TABLE group_status_modeltrace_runs IS
    'Dormant since 243: ModelTrace runs are no longer written; kept for older images';
