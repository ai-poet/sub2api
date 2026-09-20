package routes

import (
	"net/http"
	"sort"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/handler"
	servermiddleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// operatorScopeGolden 是运维管理员（operator）白名单的快照。
// 任何放大（新增条目）都必须显式改这里，防止上游合并或顺手改动悄悄扩大 operator 的可达面。
var operatorScopeGolden = []string{
	"GET /api/v1/admin/console/session",
	"GET /api/v1/admin/ops/account-availability",
	"GET /api/v1/admin/ops/advanced-settings",
	"GET /api/v1/admin/ops/alert-events",
	"GET /api/v1/admin/ops/alert-events/:id",
	"GET /api/v1/admin/ops/concurrency",
	"GET /api/v1/admin/ops/dashboard/error-distribution",
	"GET /api/v1/admin/ops/dashboard/error-trend",
	"GET /api/v1/admin/ops/dashboard/latency-histogram",
	"GET /api/v1/admin/ops/dashboard/openai-token-stats",
	"GET /api/v1/admin/ops/dashboard/overview",
	"GET /api/v1/admin/ops/dashboard/snapshot-v2",
	"GET /api/v1/admin/ops/dashboard/throughput-trend",
	"GET /api/v1/admin/ops/errors",
	"GET /api/v1/admin/ops/errors/:id",
	"GET /api/v1/admin/ops/realtime-traffic",
	"GET /api/v1/admin/ops/request-errors",
	"GET /api/v1/admin/ops/request-errors/:id",
	"GET /api/v1/admin/ops/request-errors/:id/upstream-errors",
	"GET /api/v1/admin/ops/requests",
	"GET /api/v1/admin/ops/settings/metric-thresholds",
	"GET /api/v1/admin/ops/system-logs",
	"GET /api/v1/admin/ops/system-logs/health",
	"GET /api/v1/admin/ops/upstream-errors",
	"GET /api/v1/admin/ops/upstream-errors/:id",
	"GET /api/v1/admin/ops/user-concurrency",
	"GET /api/v1/admin/ops/ws/qps",
	"GET /api/v1/admin/usage",
	"GET /api/v1/admin/usage/charts",
	"GET /api/v1/admin/usage/filter-groups",
	"GET /api/v1/admin/usage/filter-models",
	"GET /api/v1/admin/usage/model-stats",
	"GET /api/v1/admin/usage/search-accounts",
	"GET /api/v1/admin/usage/search-api-keys",
	"GET /api/v1/admin/usage/search-users",
	"GET /api/v1/admin/usage/stats",
	// 用户 / 订阅管理（只读部分）与审批申请
	"GET /api/v1/admin/approvals",
	"GET /api/v1/admin/approvals/:id",
	"GET /api/v1/admin/approvals/pending-count",
	"GET /api/v1/admin/subscriptions",
	"GET /api/v1/admin/subscriptions/:id",
	"GET /api/v1/admin/subscriptions/:id/progress",
	"GET /api/v1/admin/user-attributes",
	"GET /api/v1/admin/users",
	"GET /api/v1/admin/users/:id",
	"GET /api/v1/admin/users/:id/api-keys",
	"GET /api/v1/admin/users/:id/attributes",
	"GET /api/v1/admin/users/:id/balance-history",
	"GET /api/v1/admin/users/:id/platform-quotas",
	"GET /api/v1/admin/users/:id/rpm-status",
	"GET /api/v1/admin/users/:id/subscriptions",
	"GET /api/v1/admin/users/:id/usage",
	// 工单（fork 本地）：admin / operator 同权
	"GET /api/v1/admin/tickets",
	"GET /api/v1/admin/tickets/:id",
	"GET /api/v1/admin/tickets/attachments/content",
	"GET /api/v1/admin/tickets/open-count",
	// 显式放开的非 GET：读语义的 POST、撤回自己的申请、客服工单的回复 / 关闭 / 重开 / 图片上传
	"POST /api/v1/admin/approvals/:id/cancel",
	"POST /api/v1/admin/tickets/:id/close",
	"POST /api/v1/admin/tickets/:id/messages",
	"POST /api/v1/admin/tickets/:id/reopen",
	"POST /api/v1/admin/tickets/attachments",
	"POST /api/v1/admin/user-attributes/batch",
}

// operatorWriteScopeGolden 显式放开、不经审批的非 GET 条目；operator 白名单里出现的非 GET 必须在此登记。
var operatorWriteScopeGolden = []string{
	"POST /api/v1/admin/approvals/:id/cancel",
	"POST /api/v1/admin/tickets/:id/close",
	"POST /api/v1/admin/tickets/:id/messages",
	"POST /api/v1/admin/tickets/:id/reopen",
	"POST /api/v1/admin/tickets/attachments",
	"POST /api/v1/admin/user-attributes/batch",
}

// operatorApprovalScopeGolden operator 可发起、须管理员一键通过后重放的写接口快照。
var operatorApprovalScopeGolden = []string{
	"DELETE /api/v1/admin/subscriptions/:id",
	"POST /api/v1/admin/subscriptions/:id/extend",
	"POST /api/v1/admin/subscriptions/:id/reset-quota",
	"POST /api/v1/admin/subscriptions/:id/restore",
	"POST /api/v1/admin/subscriptions/:id/revoke",
	"POST /api/v1/admin/subscriptions/assign",
	"POST /api/v1/admin/subscriptions/bulk-assign",
	"POST /api/v1/admin/users",
	"POST /api/v1/admin/users/:id/auth-identities",
	"POST /api/v1/admin/users/:id/balance",
	"POST /api/v1/admin/users/:id/platform-quotas/reset",
	"POST /api/v1/admin/users/:id/replace-group",
	"POST /api/v1/admin/users/batch-concurrency",
	"POST /api/v1/admin/users/batch-limits",
	"PUT /api/v1/admin/api-keys/:id",
	"PUT /api/v1/admin/users/:id",
	"PUT /api/v1/admin/users/:id/attributes",
	"PUT /api/v1/admin/users/:id/platform-quotas",
}

// operatorRefusedScopeGolden operator 永远不能发起、也不入队的动作。
var operatorRefusedScopeGolden = []string{
	"DELETE /api/v1/admin/users/:id",
}

// operatorForbiddenPrefixes 永远不允许出现在白名单里的管理域（余额 / 账号 / 分组 / 设置 / 备份 / 系统 / 仪表盘）。
var operatorForbiddenPrefixes = []string{
	"/api/v1/admin/accounts",
	"/api/v1/admin/groups",
	"/api/v1/admin/settings",
	"/api/v1/admin/backups",
	"/api/v1/admin/system",
	"/api/v1/admin/dashboard",
	"/api/v1/admin/data-management",
	"/api/v1/admin/audit-logs",
	"/api/v1/admin/redeem-codes",
}

// operatorApprovalAllowedPrefixes 审批范围只能落在用户 / 订阅管理与用户 API Key 上。
var operatorApprovalAllowedPrefixes = []string{
	"/api/v1/admin/users",
	"/api/v1/admin/subscriptions",
	"/api/v1/admin/api-keys/:id",
}

func registerAllAdminAuthRoutesForTest(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handlers := &handler.Handlers{Admin: &handler.AdminHandlers{}}
	passthrough := func(c *gin.Context) { c.Next() }
	adminAuth := servermiddleware.AdminAuthMiddleware(passthrough)
	jwtAuth := servermiddleware.JWTAuthMiddleware(passthrough)
	auditLog := servermiddleware.AuditLogMiddleware(passthrough)
	stepUp := servermiddleware.StepUpAuthMiddleware(passthrough)

	v1 := router.Group("/api/v1")
	RegisterAdminRoutes(v1, handlers, adminAuth, auditLog, stepUp, nil, nil)
	handler.RegisterPageRoutes(v1, t.TempDir(), gin.HandlerFunc(jwtAuth), gin.HandlerFunc(adminAuth), nil)
	RegisterPayRoutes(router, handlers, jwtAuth, adminAuth, nil, &config.Config{JWT: config.JWTConfig{Secret: "test-secret"}})
	return router
}

func TestOperatorScopeMatchesGoldenList(t *testing.T) {
	actual := servermiddleware.OperatorScopeRoutes()
	golden := append([]string(nil), operatorScopeGolden...)
	sort.Strings(golden)
	require.Equal(t, golden, actual, "operator 白名单变了：放大或缩小 operator 可达面都必须同步更新 golden 列表并评估返回字段")
}

func TestOperatorScopeEntriesExistAsRegisteredRoutes(t *testing.T) {
	router := registerAllAdminAuthRoutesForTest(t)
	registered := map[string]struct{}{}
	for _, route := range router.Routes() {
		registered[route.Method+" "+route.Path] = struct{}{}
	}

	for _, entry := range servermiddleware.OperatorScopeRoutes() {
		_, ok := registered[entry]
		require.Truef(t, ok, "stale operator scope entry %q: route no longer registered (upstream renamed/removed it?)", entry)
	}
}

func TestOperatorScopeIsReadOnlyAndStaysOutOfForbiddenDomains(t *testing.T) {
	for _, entry := range servermiddleware.OperatorScopeRoutes() {
		method, path, ok := strings.Cut(entry, " ")
		require.True(t, ok, entry)
		if method != http.MethodGet {
			require.Containsf(t, operatorWriteScopeGolden, entry, "operator 白名单里的非 GET 条目必须显式登记在 operatorWriteScopeGolden：%s", entry)
		}
		for _, prefix := range operatorForbiddenPrefixes {
			require.Falsef(t, strings.HasPrefix(path, prefix), "operator scope must never include %s (entry %s)", prefix, entry)
		}
	}
}

// 每一条经 adminAuth 保护的路由，要么在白名单里，要么对 operator 默认拒绝——
// 这里只是把"默认拒绝"的规模打印成断言，确保白名单远小于全部管理路由。
func TestOperatorScopeIsASmallSubsetOfAdminRoutes(t *testing.T) {
	router := registerAllAdminAuthRoutesForTest(t)
	adminRoutes := 0
	for _, route := range router.Routes() {
		if strings.HasPrefix(route.Path, "/api/v1/admin/") {
			adminRoutes++
		}
	}
	whitelisted := len(servermiddleware.OperatorScopeRoutes())
	require.Greater(t, adminRoutes, whitelisted*5, "operator whitelist (%d) should stay a small fraction of admin routes (%d)", whitelisted, adminRoutes)
}

// 审批范围 golden：operator 可发起的写接口一旦变化必须显式改这里。
func TestOperatorApprovalScopeMatchesGolden(t *testing.T) {
	actual := servermiddleware.OperatorApprovalScopeRoutes()
	golden := append([]string(nil), operatorApprovalScopeGolden...)
	sort.Strings(golden)
	require.Equal(t, golden, actual, "operator 审批范围变了：必须同步更新 golden 并评估该写接口是否适合由运维发起")

	refused := servermiddleware.OperatorRefusedScopeRoutes()
	refusedGolden := append([]string(nil), operatorRefusedScopeGolden...)
	sort.Strings(refusedGolden)
	require.Equal(t, refusedGolden, refused)
}

// 审批范围里的每一条都必须是已注册的非 GET 路由，且只落在用户 / 订阅 / 用户 API Key 域内；
// 三张表（直接放行 / 须审批 / 永不允许）两两不相交。
func TestOperatorApprovalScopeInvariants(t *testing.T) {
	router := registerAllAdminAuthRoutesForTest(t)
	registered := map[string]struct{}{}
	for _, route := range router.Routes() {
		registered[route.Method+" "+route.Path] = struct{}{}
	}

	allowed := map[string]struct{}{}
	for _, entry := range servermiddleware.OperatorScopeRoutes() {
		allowed[entry] = struct{}{}
	}
	refused := map[string]struct{}{}
	for _, entry := range servermiddleware.OperatorRefusedScopeRoutes() {
		refused[entry] = struct{}{}
		_, ok := registered[entry]
		require.Truef(t, ok, "stale refused entry %q", entry)
		_, overlap := allowed[entry]
		require.Falsef(t, overlap, "refused entry must not be whitelisted: %s", entry)
	}
	for _, entry := range servermiddleware.OperatorApprovalScopeRoutes() {
		method, path, ok := strings.Cut(entry, " ")
		require.True(t, ok, entry)
		require.NotEqualf(t, http.MethodGet, method, "approval scope must be write-only: %s", entry)
		_, ok = registered[entry]
		require.Truef(t, ok, "stale approval scope entry %q: route no longer registered", entry)
		_, overlap := allowed[entry]
		require.Falsef(t, overlap, "approval entry must not also be directly allowed: %s", entry)
		_, overlap = refused[entry]
		require.Falsef(t, overlap, "approval entry must not also be refused: %s", entry)
		inDomain := false
		for _, prefix := range operatorApprovalAllowedPrefixes {
			if strings.HasPrefix(path, prefix) {
				inDomain = true
				break
			}
		}
		require.Truef(t, inDomain, "approval scope must stay within users / subscriptions / api-keys: %s", entry)
	}
	require.True(t, servermiddleware.OperatorActionRefused(http.MethodDelete, "/api/v1/admin/users/:id"), "deleting users must stay refused")
	require.False(t, servermiddleware.OperatorApprovalRequired(http.MethodDelete, "/api/v1/admin/users/:id"))
}
