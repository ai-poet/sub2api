-- 站内信（fork 本地功能，见 CLAUDE.md「站内信」）：点对点发给单个账号（user / operator / admin），不是广播。
-- 来源：内容审计风控通知（category=security，受 content_moderation_config.site_message_on_hit 控制，不依赖 SMTP）、
-- 管理员在用户管理里手动发送（category=admin；运维发送经审批重放，sender 记为发起的 operator 并带 approval_id）、
-- 系统通知（category=system，如申诉后账户恢复）。
-- 不对 users 建外键（users 为软删除）；不建去重唯一索引（request_id 可由客户端指定，风控日志 id 在通知时尚未生成）。
CREATE TABLE IF NOT EXISTS site_messages (
    id             BIGSERIAL    PRIMARY KEY,
    user_id        BIGINT       NOT NULL,
    category       VARCHAR(20)  NOT NULL DEFAULT 'admin',
    title          VARCHAR(200) NOT NULL,
    content        TEXT         NOT NULL,
    source_type    VARCHAR(32)  NOT NULL DEFAULT '',
    source_id      VARCHAR(128) NOT NULL DEFAULT '',
    sender_user_id BIGINT       NULL,
    sender_role    VARCHAR(20)  NOT NULL DEFAULT '',
    approval_id    BIGINT       NULL,
    read_at        TIMESTAMPTZ  NULL,
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- 收件箱列表（按时间倒序分页）
CREATE INDEX IF NOT EXISTS idx_site_messages_user_created
    ON site_messages (user_id, created_at DESC, id DESC);
-- 未读角标 / 未读筛选 / 自动弹窗
CREATE INDEX IF NOT EXISTS idx_site_messages_user_unread
    ON site_messages (user_id, created_at DESC) WHERE read_at IS NULL;
