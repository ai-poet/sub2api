package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSupportTicketsMigration(t *testing.T) {
	content, err := FS.ReadFile("240_support_tickets.sql")
	require.NoError(t, err)

	sql := strings.Join(strings.Fields(string(content)), " ")
	require.Contains(t, sql, "CREATE TABLE IF NOT EXISTS support_tickets (")
	require.Contains(t, sql, "status VARCHAR(20) NOT NULL DEFAULT 'open'")
	require.Contains(t, sql, "category VARCHAR(32) NOT NULL DEFAULT 'other'")
	require.Contains(t, sql, "user_unread BOOLEAN NOT NULL DEFAULT FALSE")
	require.Contains(t, sql, "message_count INT NOT NULL DEFAULT 0")
	require.Contains(t, sql, "CREATE TABLE IF NOT EXISTS support_ticket_messages (")
	require.Contains(t, sql, "ticket_id BIGINT NOT NULL REFERENCES support_tickets(id) ON DELETE CASCADE")
	require.Contains(t, sql, "author_role VARCHAR(20) NOT NULL")
	require.Contains(t, sql, "CREATE INDEX IF NOT EXISTS idx_support_tickets_user_last_message ON support_tickets (user_id, last_message_at DESC)")
	require.Contains(t, sql, "CREATE INDEX IF NOT EXISTS idx_support_tickets_status_last_message ON support_tickets (status, last_message_at DESC)")
	require.Contains(t, sql, "CREATE INDEX IF NOT EXISTS idx_support_tickets_open ON support_tickets (last_message_at DESC) WHERE status = 'open'")
	require.Contains(t, sql, "CREATE INDEX IF NOT EXISTS idx_support_tickets_user_unread ON support_tickets (user_id) WHERE user_unread = TRUE")
	require.Contains(t, sql, "CREATE INDEX IF NOT EXISTS idx_support_tickets_user_active ON support_tickets (user_id) WHERE status <> 'closed'")
	require.Contains(t, sql, "CREATE INDEX IF NOT EXISTS idx_support_ticket_messages_ticket_created ON support_ticket_messages (ticket_id, created_at ASC, id ASC)")
}
