-- 253: 推荐奖励升级为按比例返利（fork 本地，参考 TokenRouter affiliate 模型）
-- 保留 users.referral_code 与 user_referrals 作为回滚快照；新逻辑只读写以下两表。
CREATE TABLE IF NOT EXISTS user_affiliates (
    user_id BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    aff_code VARCHAR(32) NOT NULL UNIQUE,
    inviter_id BIGINT NULL REFERENCES users(id) ON DELETE SET NULL,
    aff_count INT NOT NULL DEFAULT 0,
    aff_quota NUMERIC(20,8) NOT NULL DEFAULT 0,
    aff_frozen_quota NUMERIC(20,8) NOT NULL DEFAULT 0,
    aff_history_quota NUMERIC(20,8) NOT NULL DEFAULT 0,
    aff_rebate_rate_percent NUMERIC(5,2) NULL,
    aff_code_custom BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT user_affiliates_quotas_nonnegative CHECK (aff_quota >= 0 AND aff_frozen_quota >= 0 AND aff_history_quota >= 0),
    CONSTRAINT user_affiliates_rebate_rate_valid CHECK (aff_rebate_rate_percent IS NULL OR (aff_rebate_rate_percent >= 0 AND aff_rebate_rate_percent <= 100))
);

CREATE TABLE IF NOT EXISTS user_affiliate_ledger (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    action VARCHAR(32) NOT NULL,
    amount NUMERIC(20,8) NOT NULL,
    source_user_id BIGINT NULL REFERENCES users(id) ON DELETE SET NULL,
    source_order_ref VARCHAR(64) NULL,
    frozen_until TIMESTAMPTZ NULL,
    balance_after NUMERIC(20,8) NULL,
    aff_quota_after NUMERIC(20,8) NULL,
    aff_frozen_quota_after NUMERIC(20,8) NULL,
    aff_history_quota_after NUMERIC(20,8) NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT user_affiliate_ledger_action_valid CHECK (action IN ('accrue', 'transfer')),
    CONSTRAINT user_affiliate_ledger_amount_positive CHECK (amount > 0)
);

CREATE INDEX IF NOT EXISTS idx_user_affiliates_inviter_id ON user_affiliates(inviter_id);
CREATE INDEX IF NOT EXISTS idx_user_affiliates_admin_settings ON user_affiliates(updated_at DESC) WHERE aff_code_custom = TRUE OR aff_rebate_rate_percent IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_user_affiliate_ledger_user ON user_affiliate_ledger(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_user_affiliate_ledger_source_user ON user_affiliate_ledger(source_user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_user_affiliate_ledger_frozen_thaw ON user_affiliate_ledger(user_id, frozen_until) WHERE action = 'accrue' AND frozen_until IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_user_affiliate_ledger_order_unique ON user_affiliate_ledger(source_order_ref) WHERE action = 'accrue' AND source_order_ref IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_user_affiliate_ledger_action_created ON user_affiliate_ledger(action, created_at DESC);

DO $$
DECLARE
    u RECORD;
    candidate TEXT;
    suffix TEXT;
BEGIN
    FOR u IN SELECT id, referral_code, created_at FROM users ORDER BY id LOOP
        candidate := upper(btrim(COALESCE(u.referral_code, '')));
        IF candidate !~ '^[A-Z0-9_-]{4,32}$' OR EXISTS (SELECT 1 FROM user_affiliates WHERE aff_code = candidate AND user_id <> u.id) THEN
            suffix := upper(substr(md5(u.id::text || ':affiliate'), 1, 12));
            candidate := 'CR' || suffix;
            WHILE EXISTS (SELECT 1 FROM user_affiliates WHERE aff_code = candidate AND user_id <> u.id) LOOP
                suffix := upper(substr(md5(suffix || ':retry'), 1, 12));
                candidate := 'CR' || suffix;
            END LOOP;
        END IF;
        -- 旧系统的邀请码均为自动生成，不属于管理员自定义配置。
        INSERT INTO user_affiliates (user_id, aff_code, aff_code_custom, created_at, updated_at)
        VALUES (u.id, candidate, FALSE, COALESCE(u.created_at, NOW()), NOW())
        ON CONFLICT (user_id) DO NOTHING;
    END LOOP;
END $$;

UPDATE user_affiliates ua SET inviter_id = ur.referrer_id, updated_at = NOW()
FROM user_referrals ur
WHERE ua.user_id = ur.referee_id AND ua.inviter_id IS NULL AND ur.referrer_id <> ur.referee_id
  AND EXISTS (SELECT 1 FROM users WHERE id = ur.referrer_id);

UPDATE user_affiliates ua SET aff_count = counts.total, updated_at = NOW()
FROM (SELECT inviter_id, COUNT(*)::INT AS total FROM user_affiliates WHERE inviter_id IS NOT NULL GROUP BY inviter_id) counts
WHERE ua.user_id = counts.inviter_id;

INSERT INTO settings (key, value, updated_at)
SELECT 'affiliate_enabled', value, NOW() FROM settings WHERE key = 'referral_enabled'
ON CONFLICT (key) DO NOTHING;

DELETE FROM settings WHERE key IN (
    'referral_enabled', 'referral_referrer_balance_reward', 'referral_referrer_group_id',
    'referral_referrer_subscription_days', 'referral_referee_balance_reward',
    'referral_referee_group_id', 'referral_referee_subscription_days', 'referral_max_per_user'
);
