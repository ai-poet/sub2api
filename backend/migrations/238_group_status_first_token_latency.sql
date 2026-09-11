-- 分组运行状态：存活探测的 latency_ms 改为流式首字延迟（首个内容 token 到达时间），
-- 另存一份完整返回耗时 total_latency_ms 供参考。本 fork 自有功能。
ALTER TABLE group_status_records
    ADD COLUMN IF NOT EXISTS total_latency_ms BIGINT NULL;

ALTER TABLE group_status_states
    ADD COLUMN IF NOT EXISTS total_latency_ms BIGINT NULL;
