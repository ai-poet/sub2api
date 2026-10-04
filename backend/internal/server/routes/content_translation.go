package routes

import (
	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"

	"github.com/gin-gonic/gin"
)

// 内容自动翻译（fork 本地功能，接口约定见 docs/CONTENT_TRANSLATION.md）。

// contentTranslationLookupBodyLimit 一次查询最多 200 条文本，请求体上限 256 KiB。
const contentTranslationLookupBodyLimit = 256 << 10

// contentTranslationSourcesBodyLimit 支付服务一次最多登记 1000 条文案。
const contentTranslationSourcesBodyLimit = 4 << 20

// RegisterContentTranslationRoutes 注册公开查询接口：无需登录、只读缓存、永远不调模型，
// 与 /settings/public 一样按客户端 IP 兜底限流（反代与支付服务的内网地址自动跳过）。
func RegisterContentTranslationRoutes(v1 *gin.RouterGroup, h *handler.Handlers, panelRateLimiter *middleware.PanelRateLimiter) {
	translations := v1.Group("/content-translations")
	translations.Use(panelRateLimiter.PublicIP())
	{
		translations.POST("/lookup", middleware.RequestBodyLimit(contentTranslationLookupBodyLimit), h.ContentTranslation.Lookup)
	}
}

// registerContentTranslationAdminRoutes 后台配置与译文管理：仅管理员（operator 不在白名单里，这里再显式 AdminOnly）。
func registerContentTranslationAdminRoutes(admin *gin.RouterGroup, h *handler.Handlers) {
	translations := admin.Group("/content-translations")
	translations.Use(middleware.AdminOnly())
	{
		translations.GET("/config", h.Admin.ContentTranslation.GetConfig)
		translations.PUT("/config", h.Admin.ContentTranslation.UpdateConfig)
		translations.POST("/config/test", h.Admin.ContentTranslation.TestConfig)
		translations.GET("/status", h.Admin.ContentTranslation.Status)
		translations.POST("/sync", h.Admin.ContentTranslation.Sync)
		translations.GET("", h.Admin.ContentTranslation.List)
		translations.DELETE("", h.Admin.ContentTranslation.Clear)
		translations.PUT("/:id", h.Admin.ContentTranslation.Update)
		translations.DELETE("/:id", h.Admin.ContentTranslation.Delete)
	}
}

// registerContentTranslationPayRoutes 支付服务登记自己的文案（只认内部令牌，挂在 /api/internal/pay 下）。
func registerContentTranslationPayRoutes(internal *gin.RouterGroup, h *handler.Handlers) {
	internal.PUT("/content-translations/sources",
		middleware.RequestBodyLimit(contentTranslationSourcesBodyLimit),
		h.ContentTranslation.RegisterPaySources)
}
