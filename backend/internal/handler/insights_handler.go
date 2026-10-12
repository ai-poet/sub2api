package handler

import (
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// fork：排行榜、首页公开统计与它们的后台配置（路由见 server/routes/insights.go）。

// insightsDefaultWindow 是排行榜不传 window 时的档位。
const insightsDefaultWindow = service.InsightsWindow7d

type InsightsHandler struct {
	insights *service.InsightsService
	settings *service.SettingService
}

func NewInsightsHandler(insights *service.InsightsService, settings *service.SettingService) *InsightsHandler {
	return &InsightsHandler{insights: insights, settings: settings}
}

// ModelLeaderboard 模型热度榜。
// GET /api/v1/leaderboard/models?window=24h|7d|30d
func (h *InsightsHandler) ModelLeaderboard(c *gin.Context) {
	window := c.DefaultQuery("window", insightsDefaultWindow)
	result, err := h.insights.ModelRanking(c.Request.Context(), window)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

// UserLeaderboard 用户用量榜（名字脱敏，不含用户 ID 与邮箱）。
// GET /api/v1/leaderboard/users?window=24h|7d|30d&metric=tokens|requests|cost
func (h *InsightsHandler) UserLeaderboard(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "Unauthorized")
		return
	}
	window := c.DefaultQuery("window", insightsDefaultWindow)
	metric := c.DefaultQuery("metric", service.InsightsMetricTokens)
	result, err := h.insights.UserRanking(c.Request.Context(), window, metric, subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

// PublicStats 首页公开统计。匿名可见，开关关闭或数据不可用时也返回 200，从不返回 401。
// GET /api/v1/stats/public
func (h *InsightsHandler) PublicStats(c *gin.Context) {
	result := h.insights.PublicStats(c.Request.Context())
	c.Header("Cache-Control", "public, max-age=60")
	response.Success(c, result)
}

// GetConfig 后台读取排行榜 / 公开统计配置。
// GET /api/v1/admin/insights/config
func (h *InsightsHandler) GetConfig(c *gin.Context) {
	cfg, err := h.settings.GetInsightsConfig(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, cfg)
}

// UpdateConfig 后台保存配置；保存后公开设置随即刷新。
// PUT /api/v1/admin/insights/config
func (h *InsightsHandler) UpdateConfig(c *gin.Context) {
	var req service.InsightsConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request body")
		return
	}
	cfg, err := h.settings.UpdateInsightsConfig(c.Request.Context(), req)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	h.insights.InvalidateConfig()
	response.Success(c, cfg)
}
