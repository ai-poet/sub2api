package handler

import (
	"context"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// DesktopLoginCodeRequest POST /api/v1/auth/desktop-session/code 请求体。
// claude_api_key / codex_api_key 可省略或为 null。
type DesktopLoginCodeRequest struct {
	CodeChallenge       string `json:"code_challenge"`
	CodeChallengeMethod string `json:"code_challenge_method"`
	APIKey              string `json:"api_key"`
	ClaudeAPIKey        string `json:"claude_api_key"`
	CodexAPIKey         string `json:"codex_api_key"`
}

// DesktopLoginCodeResponse 登录码与剩余有效秒数。
type DesktopLoginCodeResponse struct {
	Code      string `json:"code"`
	ExpiresIn int    `json:"expires_in"`
}

// DesktopLoginExchangeRequest POST /api/v1/auth/desktop-session/exchange 请求体。
type DesktopLoginExchangeRequest struct {
	Code         string `json:"code"`
	CodeVerifier string `json:"code_verifier"`
}

// DesktopLoginExchangeResponse 兑换结果：一对独立会话家族的桌面 token 与页面留下的 API Key。
type DesktopLoginExchangeResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"` // Access Token 有效期（秒）
	TokenType    string `json:"token_type"`
	APIKey       string `json:"api_key"`
	ClaudeAPIKey string `json:"claude_api_key,omitempty"`
	CodexAPIKey  string `json:"codex_api_key,omitempty"`
}

// desktopLoginBackendMode 后台模式开关（*service.SettingService 实现）。
type desktopLoginBackendMode interface {
	IsBackendModeEnabled(ctx context.Context) bool
}

// DesktopLoginHandler 客户端一次性登录码（fork 本地功能）。
//
// 访问不到 127.0.0.1 回调的客户端改用登录码：浏览器桥接页在已登录会话里申请一个码
// （绑定客户端的 S256 PKCE challenge），用户把码粘贴进客户端，客户端带 verifier 兑换 token。
type DesktopLoginHandler struct {
	svc         *service.DesktopLoginService
	backendMode desktopLoginBackendMode
}

// NewDesktopLoginHandler 构造客户端登录码 handler。
func NewDesktopLoginHandler(svc *service.DesktopLoginService, settingService *service.SettingService) *DesktopLoginHandler {
	h := &DesktopLoginHandler{svc: svc}
	// 显式判空，避免 nil 指针包进非 nil 接口
	if settingService != nil {
		h.backendMode = settingService
	}
	return h
}

// CreateCode 为当前登录用户签发一个客户端登录码。
// POST /api/v1/auth/desktop-session/code（需登录）
func (h *DesktopLoginHandler) CreateCode(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	var req DesktopLoginCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request body")
		return
	}
	issued, err := h.svc.CreateCode(c.Request.Context(), subject.UserID, req.CodeChallenge, req.CodeChallengeMethod, service.DesktopLoginKeys{
		APIKey:       req.APIKey,
		ClaudeAPIKey: req.ClaudeAPIKey,
		CodexAPIKey:  req.CodexAPIKey,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	setDesktopLoginNoStore(c)
	response.Success(c, DesktopLoginCodeResponse{
		Code:      issued.Code,
		ExpiresIn: issued.ExpiresIn,
	})
}

// Exchange 用登录码 + PKCE verifier 换 token 对与 API Key。
// POST /api/v1/auth/desktop-session/exchange（公开）
//
// 与码相关的任何失败（不存在、过期、已用、verifier 不符或格式错误，
// 连同请求体解析失败）都返回同一个 DESKTOP_LOGIN_CODE_INVALID。
func (h *DesktopLoginHandler) Exchange(c *gin.Context) {
	var req DesktopLoginExchangeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorFrom(c, service.ErrDesktopLoginCodeInvalid)
		return
	}
	ctx := c.Request.Context()
	result, err := h.svc.Exchange(ctx, req.Code, req.CodeVerifier)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	middleware2.SetAuditActor(c, result.UserID, "")

	// 与 RefreshToken 一致：后台模式下只允许管理员 / 运维拿到会话。
	if h.backendMode != nil && h.backendMode.IsBackendModeEnabled(ctx) && !service.IsPrivilegedRole(result.UserRole) {
		response.Forbidden(c, "Backend mode is active. Only admin login is allowed.")
		return
	}

	setDesktopLoginNoStore(c)
	response.Success(c, DesktopLoginExchangeResponse{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		ExpiresIn:    result.ExpiresIn,
		TokenType:    "Bearer",
		APIKey:       result.Keys.APIKey,
		ClaudeAPIKey: result.Keys.ClaudeAPIKey,
		CodexAPIKey:  result.Keys.CodexAPIKey,
	})
}

// setDesktopLoginNoStore 登录码与 token 都是一次性凭证，不允许被任何中间层缓存。
func setDesktopLoginNoStore(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	c.Header("Pragma", "no-cache")
}
