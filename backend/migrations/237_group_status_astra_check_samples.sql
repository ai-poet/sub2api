-- 分组运行状态：Astra 指纹验证运行记录补逐请求样本（题目、答案、归一类别、HTTP 码、延迟），
-- 供管理端事后回看每一次验证的过程。本 fork 自有功能。
ALTER TABLE group_status_astra_check_runs
    ADD COLUMN IF NOT EXISTS samples JSONB NOT NULL DEFAULT '[]'::jsonb;
