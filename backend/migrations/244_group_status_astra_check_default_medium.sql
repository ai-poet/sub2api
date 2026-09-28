-- meow 指纹验证的默认档位改为中档（meow 官方推荐；低档证据少，Sol 等答案分散的模型容易被带偏）。
-- 只改列默认值，已保存的分组保持原档位，由管理员在分组运行状态里自行调整。
ALTER TABLE group_status_configs ALTER COLUMN astra_check_tier SET DEFAULT 'medium';
