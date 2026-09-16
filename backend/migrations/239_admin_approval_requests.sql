-- 运维管理员（operator）写操作审批队列（fork 本地功能，见 docs/OPERATOR_ROLE.md「审批」一节）。
-- operator 对用户管理 / 订阅管理的写请求在认证层被捕获入队，由唯一管理员一键通过后以管理员身份重放。
CREATE TABLE IF NOT EXISTS admin_approval_requests (
    id                     BIGSERIAL PRIMARY KEY,
    status                 VARCHAR(20)   NOT NULL DEFAULT 'pending',
    action                 VARCHAR(128)  NOT NULL,
    method                 VARCHAR(10)   NOT NULL,
    route_template         VARCHAR(255)  NOT NULL,
    request_path           VARCHAR(1024) NOT NULL,
    request_query          TEXT          NOT NULL DEFAULT '',
    content_type           VARCHAR(128)  NOT NULL DEFAULT '',
    request_body_enc       TEXT          NOT NULL DEFAULT '',
    request_body_redacted  TEXT          NOT NULL DEFAULT '',
    request_body_sha256    CHAR(64)      NOT NULL DEFAULT '',
    target_type            VARCHAR(32)   NOT NULL DEFAULT '',
    target_id              BIGINT        NULL,
    target_summary         VARCHAR(255)  NOT NULL DEFAULT '',
    requester_user_id      BIGINT        NOT NULL,
    requester_email        VARCHAR(255)  NOT NULL DEFAULT '',
    requester_ip           VARCHAR(64)   NOT NULL DEFAULT '',
    request_id             VARCHAR(64)   NOT NULL DEFAULT '',
    decided_by_user_id     BIGINT        NULL,
    decided_by_email       VARCHAR(255)  NOT NULL DEFAULT '',
    decided_at             TIMESTAMPTZ   NULL,
    decision_reason        TEXT          NOT NULL DEFAULT '',
    executed_at            TIMESTAMPTZ   NULL,
    result_status_code     INT           NULL,
    result_body            TEXT          NOT NULL DEFAULT '',
    result_error           VARCHAR(255)  NOT NULL DEFAULT '',
    notified_at            TIMESTAMPTZ   NULL,
    expires_at             TIMESTAMPTZ   NOT NULL,
    created_at             TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at             TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_admin_approval_requests_status_created
    ON admin_approval_requests (status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_admin_approval_requests_requester_created
    ON admin_approval_requests (requester_user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_admin_approval_requests_pending_expires
    ON admin_approval_requests (expires_at) WHERE status = 'pending';
