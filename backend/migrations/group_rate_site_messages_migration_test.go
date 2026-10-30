package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGroupRateSiteMessagesMigration(t *testing.T) {
	content, err := FS.ReadFile("254_group_rate_site_messages.sql")
	require.NoError(t, err)

	sql := strings.Join(strings.Fields(string(content)), " ")

	require.Equal(t, 5, strings.Count(sql, "CREATE OR REPLACE FUNCTION"))
	require.Contains(t, sql, "CREATE OR REPLACE FUNCTION group_rate_site_message_recipients(target_group_id BIGINT, at_time TIMESTAMPTZ)")
	require.Contains(t, sql, "RETURNS TABLE (user_id BIGINT, rate_multiplier NUMERIC)")
	require.Contains(t, sql, "STABLE AS $$")
	require.Contains(t, sql, "CREATE OR REPLACE FUNCTION group_rate_site_message_escape(value TEXT)")
	require.Contains(t, sql, "IMMUTABLE")
	require.Contains(t, sql, "STRICT")
	require.Contains(t, sql, "left(regexp_replace(value, '[[:space:]]+', ' ', 'g'), 120)")
	require.Contains(t, sql, "chr(92) || '`*_{}[]()#+-.!|<>~'")
	require.Contains(t, sql, "CREATE OR REPLACE FUNCTION insert_group_rate_site_message(")
	require.Contains(t, sql, "CREATE OR REPLACE FUNCTION notify_group_rate_site_messages()")
	require.Contains(t, sql, "CREATE OR REPLACE FUNCTION notify_user_group_rate_site_message()")

	require.Contains(t, sql, "k.deleted_at IS NULL")
	require.Contains(t, sql, "k.status = 'active'")
	require.Contains(t, sql, "(k.expires_at IS NULL OR k.expires_at > at_time)")
	require.Contains(t, sql, "s.deleted_at IS NULL")
	require.Contains(t, sql, "s.status = 'active'")
	require.Contains(t, sql, "s.starts_at <= at_time")
	require.Contains(t, sql, "s.expires_at > at_time")
	require.Contains(t, sql, "UNION")
	require.Contains(t, sql, "u.deleted_at IS NULL")
	require.Contains(t, sql, "u.status = 'active'")
	require.Contains(t, sql, "g.deleted_at IS NULL")
	require.Contains(t, sql, "g.status = 'active'")
	require.Contains(t, sql, "g.subscription_type = 'subscription'")
	require.Contains(t, sql, "NOT g.is_exclusive AND NOT u.restrict_public_groups")
	require.Contains(t, sql, "FROM user_allowed_groups a")
	require.Contains(t, sql, "LEFT JOIN user_group_rate_multipliers ugr ON ugr.user_id = u.id AND ugr.group_id = g.id")

	require.Contains(t, sql, "COALESCE(previous_override, previous_group.rate_multiplier)")
	require.Contains(t, sql, "COALESCE(current_override, current_group.rate_multiplier)")
	require.Contains(t, sql, "GREATEST(0, previous_group.image_rate_multiplier)")
	require.Contains(t, sql, "GREATEST(0, current_group.image_rate_multiplier)")
	require.Contains(t, sql, "GREATEST(0, previous_group.video_rate_multiplier)")
	require.Contains(t, sql, "GREATEST(0, current_group.video_rate_multiplier)")
	require.Contains(t, sql, "previous_group.image_rate_independent IS DISTINCT FROM current_group.image_rate_independent")
	require.Contains(t, sql, "previous_group.video_rate_independent IS DISTINCT FROM current_group.video_rate_independent")
	require.Contains(t, sql, "'跟随基础'")
	require.Contains(t, sql, "'独立'")
	require.Contains(t, sql, "'follows base'")
	require.Contains(t, sql, "'independent'")

	require.Equal(t, 1, strings.Count(sql, "INSERT INTO"), "the migration must not backfill or snapshot historical mail")
	require.Contains(t, sql, "INSERT INTO site_messages (user_id, category, title, content, source_type, source_id, created_at)")
	require.Contains(t, sql, "'system'")
	require.Contains(t, sql, "'分组倍率调整通知 / Group rate update'")
	require.Contains(t, sql, "'专属倍率调整通知 / Personal rate update'")
	require.Contains(t, sql, "'group_rate_changed'")
	require.Contains(t, sql, "'user_group_rate_changed'")
	require.Contains(t, sql, "format('group:%s:tx:%s', current_group.id, txid_current())")
	require.Contains(t, sql, "| 项目 | 调整前 | 调整后 |")
	require.Contains(t, sql, "| Item | Before | After |")
	require.Contains(t, sql, "AT TIME ZONE 'UTC'")
	require.Contains(t, sql, "'YYYY-MM-DD HH24:MI:SS'")
	require.Contains(t, sql, "statement_timestamp()")
	require.NotContains(t, sql, "sender_user_id")
	require.NotContains(t, sql, "read_at")

	require.Equal(t, 2, strings.Count(sql, "CREATE TRIGGER"))
	require.Contains(t, sql, "DROP TRIGGER IF EXISTS trg_groups_rate_site_messages ON groups")
	require.Contains(t, sql, "CREATE TRIGGER trg_groups_rate_site_messages AFTER UPDATE OF rate_multiplier, image_rate_independent, image_rate_multiplier, video_rate_independent, video_rate_multiplier ON groups FOR EACH ROW WHEN (")
	require.Contains(t, sql, "OLD.rate_multiplier IS DISTINCT FROM NEW.rate_multiplier")
	require.Contains(t, sql, "OLD.image_rate_independent IS DISTINCT FROM NEW.image_rate_independent")
	require.Contains(t, sql, "OLD.image_rate_multiplier IS DISTINCT FROM NEW.image_rate_multiplier")
	require.Contains(t, sql, "OLD.video_rate_independent IS DISTINCT FROM NEW.video_rate_independent")
	require.Contains(t, sql, "OLD.video_rate_multiplier IS DISTINCT FROM NEW.video_rate_multiplier")
	require.Contains(t, sql, "EXECUTE FUNCTION notify_group_rate_site_messages()")
	require.Contains(t, sql, "DROP TRIGGER IF EXISTS trg_user_group_rates_site_messages ON user_group_rate_multipliers")
	require.Contains(t, sql, "CREATE TRIGGER trg_user_group_rates_site_messages AFTER INSERT OR UPDATE OR DELETE ON user_group_rate_multipliers FOR EACH ROW EXECUTE FUNCTION notify_user_group_rate_site_message()")

	require.NotContains(t, sql, "CREATE TABLE")
	require.NotContains(t, sql, "ALTER TABLE")
}
