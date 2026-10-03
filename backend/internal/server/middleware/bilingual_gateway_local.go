package middleware

import (
	"errors"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/bilingual"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"

	"github.com/gin-gonic/gin"
)

// 网关错误中英双语（fork 本地，见 CLAUDE.md「网关错误中英双语」）。
//
// AbortWithError 同时被网关（API Key 认证）和面板（jwtAuth / adminAuth）使用。
// 面板接口由前端按错误码做 i18n，原样返回；其余路径（/v1、/v1beta、/api/v3、/antigravity …）都是网关。
var bilingualSkipPrefixes = []string{
	"/api/v1/",
	"/api/internal/",
}

// localizeGatewayMessage 把网关报错改写成「中文 / English」；面板接口与不认识的消息原样返回。
func localizeGatewayMessage(c *gin.Context, message string) string {
	if c == nil || c.Request == nil || c.Request.URL == nil {
		return message
	}
	path := c.Request.URL.Path
	for _, prefix := range bilingualSkipPrefixes {
		if strings.HasPrefix(path, prefix) {
			return message
		}
	}
	return bilingual.Gateway(message)
}

// gatewayErrorMessage 取 ApplicationError 的 Message 返回给客户端；
// err.Error() 是调试串（error: code=429 reason="..." message="..."），不能直接给客户端看。
func gatewayErrorMessage(err error) string {
	if err == nil {
		return ""
	}
	var appErr *infraerrors.ApplicationError
	if errors.As(err, &appErr) && strings.TrimSpace(appErr.Message) != "" {
		return appErr.Message
	}
	return err.Error()
}
