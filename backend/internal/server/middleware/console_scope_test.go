//go:build unit

package middleware

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestOperatorScopeAllows(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		method string
		path   string
		want   bool
	}{
		{name: "ops errors list", method: http.MethodGet, path: "/api/v1/admin/ops/errors", want: true},
		{name: "ops error detail template", method: http.MethodGet, path: "/api/v1/admin/ops/errors/:id", want: true},
		{name: "ops ws", method: http.MethodGet, path: "/api/v1/admin/ops/ws/qps", want: true},
		{name: "usage list", method: http.MethodGet, path: "/api/v1/admin/usage", want: true},
		{name: "console session", method: http.MethodGet, path: "/api/v1/admin/console/session", want: true},
		{name: "compliance status is not needed by operators", method: http.MethodGet, path: "/api/v1/admin/compliance", want: false},
		{name: "compliance accept is a write", method: http.MethodPost, path: "/api/v1/admin/compliance/accept", want: false},
		{name: "lowercase method normalized", method: "get", path: "/api/v1/admin/usage", want: true},

		{name: "concrete path instead of template", method: http.MethodGet, path: "/api/v1/admin/ops/errors/42", want: false},
		{name: "empty path (unmatched route) fails closed", method: http.MethodGet, path: "", want: false},
		{name: "empty method", method: "", path: "/api/v1/admin/usage", want: false},
		{name: "write on whitelisted read path", method: http.MethodPut, path: "/api/v1/admin/ops/errors/:id", want: false},
		{name: "resolve is a write", method: http.MethodPut, path: "/api/v1/admin/ops/errors/:id/resolve", want: false},
		{name: "usage cleanup write", method: http.MethodPost, path: "/api/v1/admin/usage/cleanup-tasks", want: false},
		{name: "usage cleanup read is not whitelisted", method: http.MethodGet, path: "/api/v1/admin/usage/cleanup-tasks", want: false},
		{name: "balance is approval-only, never directly allowed", method: http.MethodPost, path: "/api/v1/admin/users/:id/balance", want: false},
		{name: "users list is readable", method: http.MethodGet, path: "/api/v1/admin/users", want: true},
		{name: "subscriptions list is readable", method: http.MethodGet, path: "/api/v1/admin/subscriptions", want: true},
		{name: "attribute batch is a read-shaped POST", method: http.MethodPost, path: "/api/v1/admin/user-attributes/batch", want: true},
		{name: "cancel own approval", method: http.MethodPost, path: "/api/v1/admin/approvals/:id/cancel", want: true},
		{name: "approve is admin only", method: http.MethodPost, path: "/api/v1/admin/approvals/:id/approve", want: false},
		{name: "delete user is refused, not allowed", method: http.MethodDelete, path: "/api/v1/admin/users/:id", want: false},
		{name: "settings", method: http.MethodGet, path: "/api/v1/admin/settings", want: false},
		{name: "accounts", method: http.MethodGet, path: "/api/v1/admin/accounts", want: false},
		{name: "groups", method: http.MethodGet, path: "/api/v1/admin/groups", want: false},
		{name: "dashboard", method: http.MethodGet, path: "/api/v1/admin/dashboard/stats", want: false},
		{name: "ops alert rules", method: http.MethodGet, path: "/api/v1/admin/ops/alert-rules", want: false},
		{name: "ops email config", method: http.MethodGet, path: "/api/v1/admin/ops/email-notification/config", want: false},
		{name: "pay admin probe", method: http.MethodGet, path: "/api/internal/pay/auth/admin", want: false},
		{name: "pages list", method: http.MethodGet, path: "/api/v1/pages", want: false},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tc.want, OperatorScopeAllows(tc.method, tc.path))
		})
	}
}

// 白名单表本身的不变量：只读条目全是 GET；直接放行的写条目只有读语义的 POST 与撤回自己的申请；
// 直接放行 / 须审批 / 永不允许三张表两两不相交。
func TestOperatorScopeTableInvariants(t *testing.T) {
	t.Parallel()

	for key := range operatorReadScope {
		require.Truef(t, strings.HasPrefix(key, "GET "), "read scope entry must be GET: %s", key)
		require.Truef(t, strings.HasPrefix(key, "GET /api/v1/admin/"), "read scope entry must live under /api/v1/admin: %s", key)
	}
	require.Equal(t, map[string]struct{}{
		"POST /api/v1/admin/user-attributes/batch": {},
		"POST /api/v1/admin/approvals/:id/cancel":  {},
	}, operatorWriteScope, "直接放行的写条目必须保持最小；新的写接口应走审批范围")

	routes := OperatorScopeRoutes()
	require.Len(t, routes, len(operatorReadScope)+len(operatorWriteScope))
	require.IsIncreasing(t, routes)

	for key := range operatorApprovalScope {
		require.Falsef(t, strings.HasPrefix(key, "GET "), "approval scope must be write-only: %s", key)
		_, allowed := operatorReadScope[key]
		_, allowedWrite := operatorWriteScope[key]
		_, refused := operatorRefusedScope[key]
		require.Falsef(t, allowed || allowedWrite || refused, "approval entry overlaps another table: %s", key)
	}
	for key := range operatorRefusedScope {
		_, allowed := operatorReadScope[key]
		_, allowedWrite := operatorWriteScope[key]
		require.Falsef(t, allowed || allowedWrite, "refused entry overlaps allow tables: %s", key)
	}
	require.IsIncreasing(t, OperatorApprovalScopeRoutes())
	require.IsIncreasing(t, OperatorRefusedScopeRoutes())
}

func TestOperatorApprovalAndRefusedScopes(t *testing.T) {
	t.Parallel()

	require.True(t, OperatorApprovalRequired(http.MethodPost, "/api/v1/admin/users/:id/balance"))
	require.True(t, OperatorApprovalRequired("put", "/api/v1/admin/users/:id"))
	require.True(t, OperatorApprovalRequired(http.MethodPut, "/api/v1/admin/api-keys/:id"))
	require.True(t, OperatorApprovalRequired(http.MethodDelete, "/api/v1/admin/subscriptions/:id"))
	require.False(t, OperatorApprovalRequired(http.MethodGet, "/api/v1/admin/users"))
	require.False(t, OperatorApprovalRequired(http.MethodPost, "/api/v1/admin/users/7/balance"), "concrete path instead of template")
	require.False(t, OperatorApprovalRequired(http.MethodPost, ""))
	require.False(t, OperatorApprovalRequired(http.MethodPost, "/api/v1/admin/compliance/accept"))

	require.True(t, OperatorActionRefused(http.MethodDelete, "/api/v1/admin/users/:id"))
	require.False(t, OperatorActionRefused(http.MethodDelete, "/api/v1/admin/subscriptions/:id"))
	require.False(t, OperatorActionRefused(http.MethodGet, "/api/v1/admin/users/:id"))
}

func TestOperatorDenyAuditLimiter(t *testing.T) {
	t.Parallel()

	limiter := &operatorDenyAuditLimiter{buckets: make(map[int64]*operatorDenyBucket)}
	base := time.Date(2026, 9, 12, 10, 0, 0, 0, time.UTC)

	for i := 0; i < operatorDenyAuditLimit; i++ {
		require.True(t, limiter.allow(7, base), "call %d within limit", i)
	}
	require.False(t, limiter.allow(7, base), "limit exhausted")
	require.True(t, limiter.allow(8, base), "other users have their own bucket")
	require.True(t, limiter.allow(7, base.Add(operatorDenyAuditWindow)), "new window resets")
	require.NotContains(t, limiter.buckets, int64(8), "expired buckets are collected on rollover")
}
