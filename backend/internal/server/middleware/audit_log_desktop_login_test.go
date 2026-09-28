package middleware

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// 客户端登录码（fork）：申请体带网关 API Key 与 PKCE challenge，兑换体带一次性码与
// verifier。键级脱敏认不出裸键 "code" / "code_verifier"，两条路由都必须整体不入库。
func TestDesktopLoginRoutesOmitAuditBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	codeRoute := "POST /api/v1/auth/desktop-session/code"
	exchangeRoute := "POST /api/v1/auth/desktop-session/exchange"
	require.Contains(t, auditBodyOmittedRoutes, codeRoute)
	require.Contains(t, auditBodyOmittedRoutes, exchangeRoute)

	repository := &auditCaptureRepository{}
	auditService := service.NewAuditLogService(repository, nil)
	auditService.Start()

	router := gin.New()
	router.Use(gin.HandlerFunc(NewAuditLogMiddleware(auditService)))
	router.POST("/api/v1/auth/desktop-session/code", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"code": "K7QM-3XPD", "expires_in": 600})
	})
	// 兑换是公开路由：没有登录主体，由 handler 在成功后补上操作者
	router.POST("/api/v1/auth/desktop-session/exchange", func(c *gin.Context) {
		SetAuditActor(c, 41, "")
		c.JSON(http.StatusOK, gin.H{"access_token": "audit-canary-access", "api_key": "audit-canary-key"})
	})

	for path, body := range map[string]string{
		"/api/v1/auth/desktop-session/code":     `{"code_challenge":"audit-canary-challenge","code_challenge_method":"S256","api_key":"audit-canary-key"}`,
		"/api/v1/auth/desktop-session/exchange": `{"code":"audit-canary-code","code_verifier":"audit-canary-verifier"}`,
	} {
		request := httptest.NewRequest(http.MethodPost, path, bytes.NewBufferString(body))
		request.Header.Set("Content-Type", "application/json")
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, request)
		require.Equal(t, http.StatusOK, recorder.Code)
	}
	auditService.Stop()

	repository.mu.Lock()
	logs := append([]*service.AuditLog(nil), repository.logs...)
	repository.mu.Unlock()
	require.Len(t, logs, 2)
	for _, entry := range logs {
		require.Equal(t, "<credential-bearing body omitted>", entry.RequestBody, entry.Path)
		require.NotContains(t, entry.RequestBody, "audit-canary")
		require.NotContains(t, fmt.Sprint(entry.Extra), "audit-canary")
		if entry.Path == "/api/v1/auth/desktop-session/exchange" {
			require.NotNil(t, entry.ActorUserID)
			require.Equal(t, int64(41), *entry.ActorUserID)
		}
	}
}
