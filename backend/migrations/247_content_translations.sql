-- 内容自动翻译（fork 本地功能，见 CLAUDE.md「内容自动翻译」与 docs/CONTENT_TRANSLATION.md）：
-- 管理员手写的文案（分组名称 / 描述、公告、站点文案、渠道描述、支付服务登记的套餐与活动文案）
-- 用管理员自己的网关 API Key 翻译后永久缓存在这里。
-- 键是「规范化原文的 sha256 + 目标语言」：原文不变就永远命中，不再请求模型；不做自动清理，
-- last_seen_at 只用于后台标出暂时用不到的译文。manual 为真的行是管理员改写过的，自动翻译不会覆盖。
CREATE TABLE IF NOT EXISTS content_translations (
    id              BIGSERIAL    PRIMARY KEY,
    source_hash     CHAR(64)     NOT NULL,
    target_lang     VARCHAR(8)   NOT NULL,
    source_text     TEXT         NOT NULL,
    translated_text TEXT         NOT NULL,
    model           VARCHAR(128) NOT NULL DEFAULT '',
    manual          BOOLEAN      NOT NULL DEFAULT FALSE,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    last_seen_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS ux_content_translations_hash_lang
    ON content_translations (source_hash, target_lang);

-- 外部登记的文案源（目前只有支付服务，namespace = 'pay'）。后端自己的文案每轮扫描时现场收集，不落这张表。
CREATE TABLE IF NOT EXISTS content_translation_sources (
    namespace   VARCHAR(32) NOT NULL,
    source_hash CHAR(64)    NOT NULL,
    source_text TEXT        NOT NULL,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (namespace, source_hash)
);
