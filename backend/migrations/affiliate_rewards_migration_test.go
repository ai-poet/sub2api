package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAffiliateRewardsMigration(t *testing.T) {
	content, err := FS.ReadFile("253_affiliate_rewards.sql")
	require.NoError(t, err)

	sql := strings.Join(strings.Fields(string(content)), " ")
	require.Contains(t, sql, "CREATE TABLE IF NOT EXISTS user_affiliates (")
	require.Contains(t, sql, "aff_code VARCHAR(32) NOT NULL UNIQUE")
	require.Contains(t, sql, "aff_rebate_rate_percent NUMERIC(5,2) NULL")
	require.Contains(t, sql, "CREATE TABLE IF NOT EXISTS user_affiliate_ledger (")
	require.Contains(t, sql, "CREATE UNIQUE INDEX IF NOT EXISTS idx_user_affiliate_ledger_order_unique ON user_affiliate_ledger(source_order_ref) WHERE action = 'accrue' AND source_order_ref IS NOT NULL")
	require.Contains(t, sql, "VALUES (u.id, candidate, FALSE,")
	require.Contains(t, sql, "UPDATE user_affiliates ua SET inviter_id = ur.referrer_id")
	require.Contains(t, sql, "SELECT 'affiliate_enabled', value")
	require.Contains(t, sql, "INSERT INTO settings (key, value, updated_at)")
	require.NotContains(t, sql, "INSERT INTO settings (key, value, created_at")
	require.NotContains(t, sql, "DROP TABLE user_referrals")
	require.NotContains(t, sql, "referral_code = NULL")
}
