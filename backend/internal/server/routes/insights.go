package routes

import (
	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"

	"github.com/gin-gonic/gin"
)

// fork：排行榜与首页公开统计。

// RegisterInsightsPublicRoutes 注册首页公开统计：无需登录，按客户端 IP 限流，从不返回 401。
func RegisterInsightsPublicRoutes(v1 *gin.RouterGroup, h *handler.Handlers, panelRateLimiter *middleware.PanelRateLimiter) {
	stats := v1.Group("/stats")
	stats.Use(panelRateLimiter.PublicIP())
	{
		stats.GET("/public", h.Insights.PublicStats)
	}
}

// registerInsightsUserRoutes 注册排行榜（登录可见），挂在用户路由的 authenticated 分组下。
func registerInsightsUserRoutes(authenticated *gin.RouterGroup, h *handler.Handlers) {
	leaderboard := authenticated.Group("/leaderboard")
	{
		leaderboard.GET("/models", h.Insights.ModelLeaderboard)
		leaderboard.GET("/users", h.Insights.UserLeaderboard)
	}
}

// registerInsightsAdminRoutes 后台配置：仅管理员（operator 不在白名单里，这里再显式 AdminOnly）。
func registerInsightsAdminRoutes(admin *gin.RouterGroup, h *handler.Handlers) {
	insights := admin.Group("/insights")
	insights.Use(middleware.AdminOnly())
	{
		insights.GET("/config", h.Insights.GetConfig)
		insights.PUT("/config", h.Insights.UpdateConfig)
	}
}
