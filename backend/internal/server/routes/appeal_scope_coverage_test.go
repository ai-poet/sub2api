package routes

import (
	"sort"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/handler"
	servermiddleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// 封禁申诉会话（fork 本地）：申诉令牌可达的路由集合。改动这张表等于改变被封账号能做的事，
// 必须同时更新 middleware/appeal_scope.go 并评估每条路由的返回内容。
var appealScopeGolden = []string{
	"GET /api/v1/appeal/session",
	"POST /api/v1/appeal/logout",
	"GET /api/v1/appeal/site-messages",
	"GET /api/v1/appeal/site-messages/unread-count",
	"POST /api/v1/appeal/site-messages/read-all",
	"POST /api/v1/appeal/site-messages/:id/read",
	"GET /api/v1/appeal/tickets",
	"POST /api/v1/appeal/tickets",
	"GET /api/v1/appeal/tickets/:id",
	"POST /api/v1/appeal/tickets/:id/messages",
	"POST /api/v1/appeal/tickets/attachments",
	"GET /api/v1/appeal/tickets/attachments/content",
}

func registeredAppealRoutes(t *testing.T) []string {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handlers := &handler.Handlers{Appeal: &handler.AppealHandler{}}
	passthrough := func(c *gin.Context) { c.Next() }
	RegisterAppealRoutes(router.Group("/api/v1"), handlers, servermiddleware.AuditLogMiddleware(passthrough), nil, nil, nil)

	out := []string{}
	for _, route := range router.Routes() {
		out = append(out, route.Method+" "+route.Path)
	}
	sort.Strings(out)
	return out
}

func TestAppealScopeMatchesGoldenList(t *testing.T) {
	golden := append([]string(nil), appealScopeGolden...)
	sort.Strings(golden)
	require.Equal(t, golden, servermiddleware.AppealScopeRoutes(), "申诉白名单变了：被封账号可达面必须同步更新 golden 列表")
}

func TestAppealScopeEqualsRegisteredAppealRoutes(t *testing.T) {
	require.Equal(t, servermiddleware.AppealScopeRoutes(), registeredAppealRoutes(t),
		"每条 /api/v1/appeal/* 路由都必须在白名单里，白名单里也不能有未注册的路由")
}

func TestAppealScopeNeverReachesPrivilegedSurfaces(t *testing.T) {
	for _, route := range servermiddleware.AppealScopeRoutes() {
		parts := strings.SplitN(route, " ", 2)
		require.Len(t, parts, 2)
		path := parts[1]
		require.True(t, strings.HasPrefix(path, "/api/v1/appeal/"), route)
		for _, forbidden := range []string{"/admin", "/auth", "/user/", "/keys", "/personal-token", "/redeem", "/payment"} {
			require.NotContains(t, path, forbidden, route)
		}
	}
}

func TestAppealRoutesSkippedWithoutHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	RegisterAppealRoutes(router.Group("/api/v1"), &handler.Handlers{}, servermiddleware.AuditLogMiddleware(func(c *gin.Context) { c.Next() }), nil, nil, nil)
	require.Empty(t, router.Routes())
}
