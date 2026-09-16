package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAdminApprovalRequestsMigration(t *testing.T) {
	content, err := FS.ReadFile("239_admin_approval_requests.sql")
	require.NoError(t, err)

	sql := strings.Join(strings.Fields(string(content)), " ")
	require.Contains(t, sql, "CREATE TABLE IF NOT EXISTS admin_approval_requests (")
	require.Contains(t, sql, "status VARCHAR(20) NOT NULL DEFAULT 'pending'")
	require.Contains(t, sql, "request_body_enc TEXT NOT NULL DEFAULT ''")
	require.Contains(t, sql, "requester_user_id BIGINT NOT NULL")
	require.Contains(t, sql, "expires_at TIMESTAMPTZ NOT NULL")
	require.Contains(t, sql, "CREATE INDEX IF NOT EXISTS idx_admin_approval_requests_status_created ON admin_approval_requests (status, created_at DESC)")
	require.Contains(t, sql, "CREATE INDEX IF NOT EXISTS idx_admin_approval_requests_requester_created ON admin_approval_requests (requester_user_id, created_at DESC)")
	require.Contains(t, sql, "CREATE INDEX IF NOT EXISTS idx_admin_approval_requests_pending_expires ON admin_approval_requests (expires_at) WHERE status = 'pending'")
}
