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
	"GET /api/v1/admin/compliance",
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
	"GET /api/v1/admin/usage/search-api-keys",
	"GET /api/v1/admin/usage/search-users",
	"GET /api/v1/admin/usage/stats",
	"POST /api/v1/admin/compliance/accept",
}

// operatorForbiddenPrefixes 永远不允许出现在白名单里的管理域（余额 / 账号 / 分组 / 设置 / 备份 / 系统 / 仪表盘）。
var operatorForbiddenPrefixes = []string{
	"/api/v1/admin/users",
	"/api/v1/admin/accounts",
	"/api/v1/admin/groups",
	"/api/v1/admin/settings",
	"/api/v1/admin/backups",
	"/api/v1/admin/system",
	"/api/v1/admin/dashboard",
	"/api/v1/admin/data-management",
	"/api/v1/admin/audit-logs",
	"/api/v1/admin/redeem-codes",
	"/api/v1/admin/subscriptions",
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
			require.Equal(t, "POST /api/v1/admin/compliance/accept", entry, "the only allowed write is the compliance acknowledgement")
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
