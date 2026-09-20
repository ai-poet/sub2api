-- 工单系统（fork 本地功能，见 docs/TICKETS.md）：用户提交工单并与客服（admin / operator）在同一线程往返；
-- 首条消息即工单正文。不对 users 建外键（users 为软删除），消息表随工单级联删除。
CREATE TABLE IF NOT EXISTS support_tickets (
    id                BIGSERIAL PRIMARY KEY,
    user_id           BIGINT       NOT NULL,
    user_email        VARCHAR(255) NOT NULL DEFAULT '',
    title             VARCHAR(200) NOT NULL,
    category          VARCHAR(32)  NOT NULL DEFAULT 'other',
    status            VARCHAR(20)  NOT NULL DEFAULT 'open',
    user_unread       BOOLEAN      NOT NULL DEFAULT FALSE,
    message_count     INT          NOT NULL DEFAULT 0,
    last_message_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    closed_at         TIMESTAMPTZ  NULL,
    closed_by_user_id BIGINT       NULL,
    closed_by_role    VARCHAR(20)  NOT NULL DEFAULT '',
    created_at        TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS support_ticket_messages (
    id             BIGSERIAL PRIMARY KEY,
    ticket_id      BIGINT       NOT NULL REFERENCES support_tickets(id) ON DELETE CASCADE,
    author_user_id BIGINT       NOT NULL,
    author_email   VARCHAR(255) NOT NULL DEFAULT '',
    author_role    VARCHAR(20)  NOT NULL,
    body           TEXT         NOT NULL,
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- 用户自己的工单列表 / 客服按状态 tab 的列表
CREATE INDEX IF NOT EXISTS idx_support_tickets_user_last_message
    ON support_tickets (user_id, last_message_at DESC);
CREATE INDEX IF NOT EXISTS idx_support_tickets_status_last_message
    ON support_tickets (status, last_message_at DESC);
-- 客服角标：待处理数
CREATE INDEX IF NOT EXISTS idx_support_tickets_open
    ON support_tickets (last_message_at DESC) WHERE status = 'open';
-- 用户角标：有未读客服回复的工单数
CREATE INDEX IF NOT EXISTS idx_support_tickets_user_unread
    ON support_tickets (user_id) WHERE user_unread = TRUE;
-- 每人未关闭工单上限
CREATE INDEX IF NOT EXISTS idx_support_tickets_user_active
    ON support_tickets (user_id) WHERE status <> 'closed';
CREATE INDEX IF NOT EXISTS idx_support_ticket_messages_ticket_created
    ON support_ticket_messages (ticket_id, created_at ASC, id ASC);
