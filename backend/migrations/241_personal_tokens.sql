-- 运维管理员个人令牌（fork 本地功能，见 docs/PERSONAL_TOKENS.md）：operator 在个人资料页自助生成，
-- 脚本以 Authorization: Bearer pat-... 调用管理 API，可达面与其 JWT 会话完全一致。
-- 每人至多一个（user_id 唯一，重新生成即覆盖）；只存 SHA-256，明文只在生成时返回一次。
-- user_token_version 记录签发时的密码 / 邮箱指纹，改密或改邮箱后令牌自动失效。
CREATE TABLE IF NOT EXISTS personal_tokens (
    id                 BIGSERIAL PRIMARY KEY,
    user_id            BIGINT       NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    token_hash         CHAR(64)     NOT NULL UNIQUE,
    token_hint         VARCHAR(32)  NOT NULL,
    user_token_version BIGINT       NOT NULL,
    expires_at         TIMESTAMPTZ  NULL,
    last_used_at       TIMESTAMPTZ  NULL,
    last_used_ip       VARCHAR(64)  NOT NULL DEFAULT '',
    created_ip         VARCHAR(64)  NOT NULL DEFAULT '',
    created_at         TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
