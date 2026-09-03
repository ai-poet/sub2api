-- 分组运行状态：稳定状态变红 / 从红恢复时是否推送提醒（Server酱³），默认开启。
ALTER TABLE group_status_configs
    ADD COLUMN IF NOT EXISTS notify_enabled BOOLEAN NOT NULL DEFAULT TRUE;

COMMENT ON COLUMN group_status_configs.notify_enabled IS
    'Whether stable down/up transitions of this group trigger push notifications (ServerChan3)';
