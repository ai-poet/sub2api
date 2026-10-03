package routes

import (
	"net/http"
	"strconv"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/middleware"
	servermiddleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// RegisterAppealRoutes 封禁申诉会话（fork 本地，见 service/appeal.go）。
//
// 只用 X-Appeal-Token 认证，路由集合由 middleware/appeal_scope.go 的白名单钉死
// （routes/appeal_scope_coverage_test.go 校验两者一一对应）。工单 / 附件 / 站内信复用用户侧 handler。
func RegisterAppealRoutes(
	v1 *gin.RouterGroup,
	h *handler.Handlers,
	auditLog servermiddleware.AuditLogMiddleware,
	redisClient *redis.Client,
	settingService *service.SettingService,
	panelRateLimiter *servermiddleware.PanelRateLimiter,
) {
	if h == nil || h.Appeal == nil {
		return
	}
	limiter := middleware.NewRateLimiter(redisClient)

	appeal := v1.Group("/appeal")
	appeal.Use(limiter.LimitWithOptions("appeal-api", 120, time.Minute, middleware.RateLimitOptions{
		FailureMode: middleware.RateLimitFailClose,
	}))
	appeal.Use(servermiddleware.AppealAuth(h.Appeal.Authenticator()))
	appeal.Use(servermiddleware.BackendModeUserGuard(settingService))
	appeal.Use(panelRateLimiter.Global())
	appeal.Use(gin.HandlerFunc(auditLog))
	{
		appeal.GET("/session", h.Appeal.Session)
		appeal.POST("/logout", h.Appeal.Logout)

		siteMessages := appeal.Group("/site-messages")
		{
			siteMessages.GET("", h.SiteMessage.List)
			siteMessages.GET("/unread-count", h.SiteMessage.UnreadCount)
			siteMessages.POST("/read-all", h.SiteMessage.MarkAllRead)
			siteMessages.POST("/:id/read", h.SiteMessage.MarkRead)
		}

		tickets := appeal.Group("/tickets")
		{
			tickets.GET("", h.Ticket.List)
			tickets.POST("",
				appealUserLimit(limiter, "appeal-ticket-create", 1, time.Minute),
				appealUserLimit(limiter, "appeal-ticket-create-day", 5, 24*time.Hour),
				h.Appeal.CreateTicket)
			tickets.POST("/attachments", panelRateLimiter.Heavy(),
				appealUserLimit(limiter, "appeal-ticket-write", 30, time.Hour),
				servermiddleware.RequestBodyLimit(service.MaxTicketAttachmentBytes+(1<<20)), h.TicketAttachment.Upload)
			tickets.GET("/attachments/content", h.TicketAttachment.Content)
			tickets.GET("/:id", h.Ticket.Get)
			tickets.POST("/:id/messages", panelRateLimiter.Heavy(),
				appealUserLimit(limiter, "appeal-ticket-write", 30, time.Hour), h.Ticket.Reply)
		}
	}
}

// appealUserLimit 按申诉会话的用户限流，Redis 故障时拒绝（申诉入口宁可暂时不可用也不被刷）。
func appealUserLimit(limiter *middleware.RateLimiter, key string, limit int, window time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		subject, ok := servermiddleware.GetAuthSubjectFromContext(c)
		if !ok || subject.UserID <= 0 || limiter == nil {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"code": http.StatusTooManyRequests, "reason": "APPEAL_RATE_LIMITED", "message": "too many requests"})
			return
		}
		result, err := limiter.Allow(c.Request.Context(), key+":uid:"+strconv.FormatInt(subject.UserID, 10), limit, window)
		if err != nil || !result.Allowed {
			if result.RetryAfter > 0 {
				c.Header("Retry-After", strconv.Itoa(int((result.RetryAfter+time.Second-1)/time.Second)))
			}
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"code": http.StatusTooManyRequests, "reason": "APPEAL_RATE_LIMITED", "message": "too many requests"})
			return
		}
		c.Next()
	}
}
