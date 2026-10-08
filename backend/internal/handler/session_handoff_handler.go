package handler

import (
	"context"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// SessionHandoffCodeRequest POST /api/v1/auth/session-handoff/code 请求体。
type SessionHandoffCodeRequest struct {
	CodeChallenge       string `json:"code_challenge"`
	CodeChallengeMethod string `json:"code_challenge_method"`
	TargetOrigin        string `json:"target_origin"`
}

// SessionHandoffCodeResponse 交接码、规范化后的接收方 origin 与剩余有效秒数。
type SessionHandoffCodeResponse struct {
	Code         string `json:"code"`
	TargetOrigin string `json:"target_origin"`
	ExpiresIn    int    `json:"expires_in"`
}

// SessionHandoffExchangeRequest POST /api/v1/auth/session-handoff/exchange 请求体。
type SessionHandoffExchangeRequest struct {
	Code         string `json:"code"`
	CodeVerifier string `json:"code_verifier"`
}

// SessionHandoffExchangeResponse 一对独立会话家族的 token。
type SessionHandoffExchangeResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"` // Access Token 有效期（秒）
	TokenType    string `json:"token_type"`
}

// sessionHandoffBackendMode 后台模式开关（*service.SettingService 实现）。
type sessionHandoffBackendMode interface {
	IsBackendModeEnabled(ctx context.Context) bool
}

// SessionHandoffHandler 多域名登录交接（fork 本地功能），见 service.SessionHandoffService。
type SessionHandoffHandler struct {
	svc         *service.SessionHandoffService
	backendMode sessionHandoffBackendMode
}

// NewSessionHandoffHandler 构造交接 handler。
func NewSessionHandoffHandler(svc *service.SessionHandoffService, settingService *service.SettingService) *SessionHandoffHandler {
	h := &SessionHandoffHandler{svc: svc}
	// 显式判空，避免 nil 指针包进非 nil 接口
	if settingService != nil {
		h.backendMode = settingService
	}
	return h
}

// CreateCode 为当前登录用户签发一个交接码。
// POST /api/v1/auth/session-handoff/code（需登录）
func (h *SessionHandoffHandler) CreateCode(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	var req SessionHandoffCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request body")
		return
	}
	issued, err := h.svc.CreateCode(c.Request.Context(), subject.UserID, req.CodeChallenge, req.CodeChallengeMethod, req.TargetOrigin)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	setDesktopLoginNoStore(c)
	response.Success(c, SessionHandoffCodeResponse{
		Code:         issued.Code,
		TargetOrigin: issued.TargetOrigin,
		ExpiresIn:    issued.ExpiresIn,
	})
}

// Exchange 用交接码 + PKCE verifier 换一对新的 token。
// POST /api/v1/auth/session-handoff/exchange（公开）
//
// 与码相关的任何失败（连同请求体解析失败）都返回同一个 SESSION_HANDOFF_CODE_INVALID。
func (h *SessionHandoffHandler) Exchange(c *gin.Context) {
	var req SessionHandoffExchangeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorFrom(c, service.ErrSessionHandoffCodeInvalid)
		return
	}
	ctx := c.Request.Context()
	result, err := h.svc.Exchange(ctx, req.Code, req.CodeVerifier, c.GetHeader("Origin"))
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
	response.Success(c, SessionHandoffExchangeResponse{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		ExpiresIn:    result.ExpiresIn,
		TokenType:    "Bearer",
	})
}
