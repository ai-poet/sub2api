-- 添加易支付渠道配置
-- epay_channels 存储启用的支付渠道，逗号分隔
-- 例如: alipay,wxpay,usdt.plasma,usdt.polygon

INSERT INTO settings (key, value, created_at, updated_at)
VALUES ('epay_channels', 'alipay,wxpay', NOW(), NOW())
ON CONFLICT (key) DO NOTHING;
