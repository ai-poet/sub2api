package routes

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

const internalPayTokenHeader = "X-Sub2API-Pay-Token"

func RegisterPayRoutes(
	r *gin.Engine,
	h *handler.Handlers,
	jwtAuth middleware.JWTAuthMiddleware,
	adminAuth middleware.AdminAuthMiddleware,
	userService *service.UserService,
	cfg *config.Config,
) {
	registerPayProxyRoutes(r)

	if cfg == nil || strings.TrimSpace(cfg.JWT.Secret) == "" {
		return
	}

	internal := r.Group("/api/internal/pay")
	internal.Use(internalPayAuthMiddleware(cfg.JWT.Secret), internalPayNeutralSessionBinding())
	{
		authenticated := internal.Group("/auth")
		authenticated.Use(gin.HandlerFunc(jwtAuth))
		authenticated.GET("/me", h.Auth.GetCurrentUser)

		adminCheck := internal.Group("/auth")
		adminCheck.Use(gin.HandlerFunc(adminAuth))
		adminCheck.GET("/admin", func(c *gin.Context) {
			subject, _ := middleware.GetAuthSubjectFromContext(c)
			c.JSON(http.StatusOK, gin.H{
				"ok":      true,
				"user_id": subject.UserID,
			})
		})
	}

	// 附件与通知桥接：只需内部令牌，不需要 internalPayAdminContextMiddleware 合成的
	// 管理员身份（它们不复用任何 admin handler）。
	attachments := internal.Group("/attachments")
	attachments.Use(middleware.RequestBodyLimit(service.MaxPayAttachmentBytes))
	{
		attachments.POST("", h.PayBridge.UploadAttachment)
		attachments.GET("/content", h.PayBridge.DownloadAttachment)
		attachments.DELETE("", h.PayBridge.DeleteAttachment)
	}
	internal.POST("/notifications/invoice-ready", h.PayBridge.SendInvoiceReadyEmail)

	adminInternal := internal.Group("")
	adminInternal.Use(internalPayAdminContextMiddleware(userService))
	{
		adminInternal.GET("/users", h.Admin.User.List)
		adminInternal.GET("/users/:id", h.Admin.User.GetByID)
		adminInternal.POST("/users/:id/balance", h.Admin.User.UpdateBalance)
		adminInternal.GET("/users/:id/subscriptions", h.Admin.Subscription.ListByUser)

		adminInternal.GET("/groups/all", h.Admin.Group.GetAll)
		adminInternal.GET("/groups/:id", h.Admin.Group.GetByID)

		adminInternal.GET("/subscriptions", h.Admin.Subscription.List)
		adminInternal.POST("/subscriptions/assign", h.Admin.Subscription.Assign)
		adminInternal.POST("/subscriptions/:id/extend", h.Admin.Subscription.Extend)

		adminInternal.POST("/redeem-codes/create-and-redeem", h.Admin.Redeem.CreateAndRedeem)
	}
}

func registerPayProxyRoutes(r *gin.Engine) {
	target, err := url.Parse(resolvePayProxyTarget())
	if err != nil {
		panic("invalid SUB2APIPAY_INTERNAL_URL: " + err.Error())
	}

	proxy := httputil.NewSingleHostReverseProxy(target)
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalHost := req.Host
		originalPath := req.URL.Path
		originalQuery := req.URL.RawQuery
		forwardedProto := payProxyForwardedProto(req)
		req.URL.Path = rewritePayProxyPath(req.URL.Path)
		req.URL.RawPath = rewritePayProxyPath(req.URL.RawPath)
		originalDirector(req)
		if originalHost != "" {
			req.Header.Set("X-Forwarded-Host", originalHost)
		}
		req.Header.Set("X-Forwarded-Proto", forwardedProto)
		// Always force the integrated pay prefix so embedded navigation
		// consistently generates /pay/... URLs even behind extra reverse proxies.
		req.Header.Set("X-Forwarded-Prefix", "/pay")
		req.Header.Set("X-Pathname", originalPath)
		req.Header.Set("X-Search", originalQuery)
	}
	proxy.ErrorHandler = func(w http.ResponseWriter, req *http.Request, proxyErr error) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`{"error":"pay service unavailable"}`))
	}

	handler := func(c *gin.Context) {
		proxy.ServeHTTP(c.Writer, c.Request)
		c.Abort()
	}

	r.Any("/_next/*proxyPath", handler)
	r.Any("/pay", handler)
	r.Any("/pay/*proxyPath", handler)
}

// payProxyForwardedProto 决定透传给支付服务的协议。
//
// 上游反代（Caddy / Nginx）已经设置的 X-Forwarded-Proto 优先：本服务自己不终止 TLS，以前无条件按
// req.TLS 判断，经边缘终止 TLS 后支付服务永远看到 http，据此生成的 notify_url / return_url 都指向
// http://，被边缘 308 跳到 https 后第三方支付的回调客户端拿不到 "success"，订单只能靠页面轮询兜底。
// 没有上游头（直连本服务）时才回退到 req.TLS。值只认 http / https，其它一律按无头处理。
func payProxyForwardedProto(req *http.Request) string {
	raw := req.Header.Get("X-Forwarded-Proto")
	if i := strings.IndexByte(raw, ','); i >= 0 {
		raw = raw[:i]
	}
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "https":
		return "https"
	case "http":
		return "http"
	}
	if req.TLS != nil {
		return "https"
	}
	return "http"
}

func rewritePayProxyPath(path string) string {
	switch {
	case path == "/pay/admin" || strings.HasPrefix(path, "/pay/admin/"):
		return strings.TrimPrefix(path, "/pay")
	case path == "/pay/api" || strings.HasPrefix(path, "/pay/api/"):
		return strings.TrimPrefix(path, "/pay")
	case path == "/pay/icons" || strings.HasPrefix(path, "/pay/icons/"):
		return strings.TrimPrefix(path, "/pay")
	default:
		return path
	}
}

func resolvePayProxyTarget() string {
	if value := strings.TrimSpace(os.Getenv("SUB2APIPAY_INTERNAL_URL")); value != "" {
		return value
	}
	return "http://127.0.0.1:3000"
}

func internalPayAuthMiddleware(jwtSecret string) gin.HandlerFunc {
	expectedTokens := deriveAcceptedInternalPayTokens(jwtSecret)
	return func(c *gin.Context) {
		received := strings.TrimSpace(c.GetHeader(internalPayTokenHeader))
		if received == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid internal pay token"})
			return
		}

		for _, expected := range expectedTokens {
			if hmac.Equal([]byte(received), []byte(expected)) {
				c.Next()
				return
			}
		}

		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid internal pay token"})
	}
}

// internalPayNeutralSessionBinding 让内部支付桥接的请求不参与会话绑定校验。
//
// 这些请求由 sub2apipay 在服务端发起（127.0.0.1 + Node UA），带的是用户的 JWT。会话绑定开启时，
// SessionBindingContext 按这个 IP/UA 算出的指纹与 token 登录时的指纹必然不同，enforceSessionBinding
// 会据此吊销整个会话族——用户在网页或桌面端打开支付页就被踢下线。桌面端还会把自己的 token 交给
// 浏览器里的收银台，所以转发真实客户端 IP/UA 也救不了。
//
// 这里注入空指纹（Hash() == ""），enforceSessionBinding 对空指纹直接放行。内部令牌已经证明调用方
// 是 sub2apipay，jwtAuth 仍然校验签名、过期、TokenVersion 与用户状态。必须挂在
// internalPayAuthMiddleware 之后：没有内部令牌的请求不能拿到这份豁免。
func internalPayNeutralSessionBinding() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request = c.Request.WithContext(service.WithSessionBinding(c.Request.Context(), &service.SessionBinding{}))
		c.Next()
	}
}

func deriveAcceptedInternalPayTokens(jwtSecret string) []string {
	secrets := []string{strings.TrimSpace(jwtSecret)}
	if envSecret := strings.TrimSpace(os.Getenv("JWT_SECRET")); envSecret != "" {
		secrets = append(secrets, envSecret)
	}

	tokens := make([]string, 0, len(secrets))
	seen := make(map[string]struct{}, len(secrets))
	for _, secret := range secrets {
		if secret == "" {
			continue
		}
		if _, exists := seen[secret]; exists {
			continue
		}
		seen[secret] = struct{}{}
		tokens = append(tokens, deriveInternalPayToken(secret))
	}

	if len(secrets) >= 2 && len(tokens) >= 2 && strings.TrimSpace(jwtSecret) != strings.TrimSpace(os.Getenv("JWT_SECRET")) {
		log.Printf("[pay_integration] detected JWT secret mismatch between runtime env and active config; accepting both internal pay tokens")
	}

	return tokens
}

func internalPayAdminContextMiddleware(userService *service.UserService) gin.HandlerFunc {
	return func(c *gin.Context) {
		if userService == nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "user service unavailable"})
			return
		}

		adminUser, err := userService.GetFirstAdmin(c.Request.Context())
		if err != nil || adminUser == nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "admin user unavailable"})
			return
		}

		c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{
			UserID:      adminUser.ID,
			Concurrency: adminUser.Concurrency,
		})
		c.Set(string(middleware.ContextKeyUserRole), adminUser.Role)
		c.Set("auth_method", "internal_pay")
		c.Next()
	}
}

func deriveInternalPayToken(jwtSecret string) string {
	mac := hmac.New(sha256.New, []byte(jwtSecret))
	_, _ = mac.Write([]byte("sub2api-pay-internal-bridge:v1"))
	return hex.EncodeToString(mac.Sum(nil))
}
