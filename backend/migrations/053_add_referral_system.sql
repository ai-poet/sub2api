-- 用户推荐邀请奖励系统
-- 用户表新增 referral_code 字段
ALTER TABLE users ADD COLUMN IF NOT EXISTS referral_code VARCHAR(32) DEFAULT '';

-- 部分唯一索引：referral_code 非空且未软删除时唯一
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_referral_code ON users (referral_code)
  WHERE referral_code != '' AND deleted_at IS NULL;

-- 推荐关系表
CREATE TABLE IF NOT EXISTS user_referrals (
    id BIGSERIAL PRIMARY KEY,
    referrer_id BIGINT NOT NULL REFERENCES users(id),
    referee_id BIGINT NOT NULL REFERENCES users(id),
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    referrer_balance_reward DECIMAL(20,8) DEFAULT 0,
    referrer_group_id BIGINT,
    referrer_subscription_days INT DEFAULT 0,
    referrer_rewarded_at TIMESTAMPTZ,
    referee_balance_reward DECIMAL(20,8) DEFAULT 0,
    referee_group_id BIGINT,
    referee_subscription_days INT DEFAULT 0,
    referee_rewarded_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_user_referrals_referee_id ON user_referrals (referee_id);
CREATE INDEX IF NOT EXISTS idx_user_referrals_referrer_id ON user_referrals (referrer_id);
CREATE INDEX IF NOT EXISTS idx_user_referrals_status ON user_referrals (status);
