//go:build unit

package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// 后台模式下 operator 作为控制台角色可以使用自服务接口（/auth/me、个人资料等），普通用户仍被拒绝。
func TestBackendModeUserGuardAllowsOperator(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := newBackendModeSettingService(t, "true")

	for role, want := range map[string]int{
		service.RoleAdmin:    http.StatusOK,
		service.RoleOperator: http.StatusOK,
		service.RoleUser:     http.StatusForbidden,
	} {
		r := gin.New()
		r.Use(func(c *gin.Context) {
			c.Set(string(ContextKeyUserRole), role)
			c.Next()
		})
		r.Use(BackendModeUserGuard(svc))
		r.GET("/auth/me", func(c *gin.Context) { c.Status(http.StatusOK) })

		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/auth/me", nil))
		require.Equalf(t, want, rec.Code, "role %s", role)
	}
}
