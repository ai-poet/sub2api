package handler

import (
	"log/slog"
	"net/url"
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ip"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// 封禁申诉会话的签发点（fork 本地，见 service/appeal.go 与 CLAUDE.md「封禁申诉会话」）。
//
// 上游登录 handler 里每个「身份已验证、但账号被禁用」的出口只加一行：
//   if respondAppealIfDisabled(c, h.appealService, err) { return }      // JSON 响应
//   if h.redirectAppealIfDisabled(c, frontendCallback, err) { return }  // 第三方回调重定向
// 错误必须是 service.UserDisabledError（只在凭证验证之后构造），其余错误一律原样走原来的分支。
// appealService 为 nil、账号不是 disabled、后端模式下的普通用户时不签发，行为与原来完全一致。

// SetAppealService 注入申诉服务（ProvideAuthHandler）。
func (h *AuthHandler) SetAppealService(svc *service.AppealService) {
	if h != nil {
		h.appealService = svc
	}
}

// SetAppealService 注入申诉服务（ProvidePasskeyHandler）。
func (h *PasskeyHandler) SetAppealService(svc *service.AppealService) {
	if h != nil {
		h.appealService = svc
	}
}

// issueAppealToken 被禁用账号的错误 → 申诉令牌；不满足条件返回 false。
func issueAppealToken(c *gin.Context, appeal *service.AppealService, err error, via string) (string, int, bool) {
	if appeal == nil || err == nil {
		return "", 0, false
	}
	disabled, ok := service.AsUserDisabledError(err)
	if !ok {
		return "", 0, false
	}
	token, ttl, issueErr := appeal.Issue(c.Request.Context(), disabled.UserID)
	if issueErr != nil {
		slog.Warn("appeal.issue_failed", "user_id", disabled.UserID, "via", via, "error", issueErr)
		return "", 0, false
	}
	if token == "" {
		return "", 0, false
	}
	middleware2.SetAuditActor(c, disabled.UserID, disabled.Email)
	middleware2.SetAuditExtra(c, map[string]any{"result": "appeal_issued"})
	slog.Info("appeal.session_issued", "user_id", disabled.UserID, "via", via, "client_ip", ip.GetClientIP(c))
	return token, int(ttl.Seconds()), true
}

// respondAppealIfDisabled JSON 出口：照旧返回 403 USER_NOT_ACTIVE，metadata 里带上申诉令牌。
func respondAppealIfDisabled(c *gin.Context, appeal *service.AppealService, err error) bool {
	token, expiresIn, ok := issueAppealToken(c, appeal, err, "json")
	if !ok {
		return false
	}
	response.ErrorFrom(c, service.ErrUserNotActive.WithMetadata(map[string]string{
		"appeal_token": token,
		"expires_in":   strconv.Itoa(expiresIn),
	}))
	return true
}

// redirectAppealIfDisabled 第三方登录回调出口：重定向到前端回调页并在 fragment 里带上申诉令牌（不带任何 access token）。
func (h *AuthHandler) redirectAppealIfDisabled(c *gin.Context, frontendCallback string, err error) bool {
	if h == nil {
		return false
	}
	token, expiresIn, ok := issueAppealToken(c, h.appealService, err, "oauth")
	if !ok {
		return false
	}
	fragment := url.Values{}
	fragment.Set("appeal_token", token)
	fragment.Set("expires_in", strconv.Itoa(expiresIn))
	redirectWithFragment(c, frontendCallback, fragment)
	return true
}

// handleDisabledPasswordLogin 密码登录：密码已验证但账号被禁用。
// 开了 2FA 的账号先走正常的 2FA 挑战（Login2FA 里再签发申诉令牌），申诉不能比正常登录更宽松。
func (h *AuthHandler) handleDisabledPasswordLogin(c *gin.Context, err error) bool {
	if h == nil || h.appealService == nil {
		return false
	}
	disabled, ok := service.AsUserDisabledError(err)
	if !ok {
		return false
	}
	if h.totpService != nil && h.settingSvc != nil && h.userService != nil && h.settingSvc.IsTotpEnabled(c.Request.Context()) {
		user, getErr := h.userService.GetByID(c.Request.Context(), disabled.UserID)
		if getErr == nil && user != nil && user.TotpEnabled {
			tempToken, sessErr := h.totpService.CreateLoginSession(c.Request.Context(), user.ID, user.Email)
			if sessErr != nil {
				response.InternalError(c, "Failed to create 2FA session")
				return true
			}
			response.Success(c, TotpLoginResponse{
				Requires2FA:     true,
				TempToken:       tempToken,
				UserEmailMasked: service.MaskEmail(user.Email),
			})
			return true
		}
	}
	return respondAppealIfDisabled(c, h.appealService, err)
}

// rejectInactivePasskeyUser Passkey 登录：Passkey 服务对 disabled 账号放行到这里，
// 被禁用就发申诉令牌，其它非 active 状态照旧 403；返回 true 表示已写响应。
func (h *PasskeyHandler) rejectInactivePasskeyUser(c *gin.Context, user *service.User) bool {
	err := ensureLoginUserActive(user)
	if err == nil {
		return false
	}
	if respondAppealIfDisabled(c, h.appealService, err) {
		return true
	}
	response.ErrorFrom(c, err)
	return true
}
