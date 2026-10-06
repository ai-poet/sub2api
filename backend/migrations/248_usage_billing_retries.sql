-- 248: 扣费失败重试队列（fork 本地功能）
--
-- 后扣费事务（余额 / 订阅 / Key 额度 / 账号配额）失败时，扣费命令不再丢失：
-- 服务把序列化后的 UsageBillingCommand 入队，后台按退避重放。request_id + api_key_id
-- 与 usage_billing_dedup 同键，重放经去重表天然幂等。行只在成功或放弃后保留作审计。
CREATE TABLE IF NOT EXISTS usage_billing_retries (
    id BIGSERIAL PRIMARY KEY,
    request_id VARCHAR(255) NOT NULL,
    api_key_id BIGINT NOT NULL,
    user_id BIGINT NOT NULL,
    account_id BIGINT NOT NULL,
    group_id BIGINT NULL,
    platform VARCHAR(64) NOT NULL DEFAULT '',
    -- 应扣金额快照（倍率后），只用于展示与告警汇总；真正重放的是 command
    actual_cost NUMERIC(20,8) NOT NULL DEFAULT 0,
    -- 序列化的 service.UsageBillingCommand（含去重指纹，重放时原样提交）
    command JSONB NOT NULL,
    -- pending | settled | failed
    status VARCHAR(16) NOT NULL DEFAULT 'pending',
    attempts INT NOT NULL DEFAULT 0,
    last_error TEXT NOT NULL DEFAULT '',
    next_retry_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    settled_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_usage_billing_retries_request UNIQUE (request_id, api_key_id)
);

-- 重放扫描只看 pending 行
CREATE INDEX IF NOT EXISTS idx_usage_billing_retries_due
    ON usage_billing_retries (next_retry_at) WHERE status = 'pending';
-- 告警按时间窗统计失败笔数
CREATE INDEX IF NOT EXISTS idx_usage_billing_retries_created
    ON usage_billing_retries (created_at);
CREATE INDEX IF NOT EXISTS idx_usage_billing_retries_user
    ON usage_billing_retries (user_id, created_at DESC);

-- 默认告警规则：5 分钟窗口内出现任何一笔扣费失败即触发（critical，30 分钟冷却）。
-- 只在还没有同类规则时插入，管理员改过的规则不会被覆盖。
INSERT INTO ops_alert_rules (name, description, enabled, severity, metric_type, operator, threshold, window_minutes, sustained_minutes, cooldown_minutes)
SELECT '扣费失败',
       '统计窗口内扣费事务失败并进入重试队列的请求数。出现即说明有请求未能即时扣费，请到使用记录页核对。',
       TRUE, 'critical', 'billing_failure_count', '>=', 1, 5, 1, 30
WHERE NOT EXISTS (SELECT 1 FROM ops_alert_rules WHERE metric_type = 'billing_failure_count');
