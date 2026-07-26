-- 上游原始版本同时写入支付可见渠道与调度器开关；本仓库不含内置支付栈，
-- 这里只保留网关侧的 openai_advanced_scheduler_enabled 默认值。
INSERT INTO settings (key, value)
VALUES
    ('openai_advanced_scheduler_enabled', 'false')
ON CONFLICT (key) DO NOTHING;
