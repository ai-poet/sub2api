package handler

import (
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"

	"github.com/gin-gonic/gin"
)

// CreateDesktopSession 为原生桌面客户端签发独立会话家族的 token 对。
// POST /api/v1/auth/desktop-session（需登录）
//
// 桌面端桥接登录页在浏览器会话内调用本接口，把返回的 token 对交给桌面端，
// 而不是把浏览器自己的 token 交出去——两端共用一个轮转制 refresh token
// 会让后刷新的一方掉登录。
func (h *AuthHandler) CreateDesktopSession(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	pair, err := h.authService.GenerateDesktopTokenPair(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, RefreshTokenResponse{
		AccessToken:  pair.AccessToken,
		RefreshToken: pair.RefreshToken,
		ExpiresIn:    pair.ExpiresIn,
		TokenType:    "Bearer",
	})
}
