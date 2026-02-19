-- +goose Up
-- +goose StatementBegin

-- 支付回调日志表：记录所有支付回调的原始数据和处理结果
CREATE TABLE IF NOT EXISTS payment_callback_logs (
    id BIGSERIAL PRIMARY KEY,
    
    -- 订单信息
    order_no VARCHAR(100) NOT NULL,
    
    -- 支付提供商: epay, creem
    provider VARCHAR(20) NOT NULL,
    
    -- 原始回调数据 (JSON)
    raw_data JSONB NOT NULL,
    
    -- 签名/验证信息
    signature TEXT,
    
    -- 验证结果
    verified BOOLEAN NOT NULL DEFAULT FALSE,
    
    -- 处理结果
    processed BOOLEAN NOT NULL DEFAULT FALSE,
    
    -- 处理结果消息
    result_message TEXT,
    
    -- 客户端IP
    client_ip VARCHAR(45),
    
    -- 时间戳
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    processed_at TIMESTAMP WITH TIME ZONE
);

-- 索引
CREATE INDEX idx_payment_callback_logs_order_no ON payment_callback_logs(order_no);
CREATE INDEX idx_payment_callback_logs_provider ON payment_callback_logs(provider);
CREATE INDEX idx_payment_callback_logs_created_at ON payment_callback_logs(created_at);
CREATE INDEX idx_payment_callback_logs_verified ON payment_callback_logs(verified);
CREATE INDEX idx_payment_callback_logs_processed ON payment_callback_logs(processed);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_payment_callback_logs_processed;
DROP INDEX IF EXISTS idx_payment_callback_logs_verified;
DROP INDEX IF EXISTS idx_payment_callback_logs_created_at;
DROP INDEX IF EXISTS idx_payment_callback_logs_provider;
DROP INDEX IF EXISTS idx_payment_callback_logs_order_no;
DROP TABLE IF EXISTS payment_callback_logs;
-- +goose StatementEnd
