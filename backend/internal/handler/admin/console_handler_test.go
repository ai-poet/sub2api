package admin

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestConsoleSession_ScopesByRole(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewConsoleHandler(nil, nil)

	for role, wantScopes := range map[string][]string{
		service.RoleAdmin:    {ConsoleScopeAll},
		service.RoleOperator: {ConsoleScopeOpsRead, ConsoleScopeUsageRead},
		service.RoleUser:     {},
	} {
		router := gin.New()
		router.GET("/api/v1/admin/console/session", func(c *gin.Context) {
			c.Set(string(middleware.ContextKeyUserRole), role)
			h.GetSession(c)
		})
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/admin/console/session", nil))
		require.Equal(t, http.StatusOK, rec.Code)

		var resp struct {
			Data ConsoleSessionResponse `json:"data"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
		require.Equal(t, role, resp.Data.Role)
		require.Equal(t, wantScopes, resp.Data.Scopes)
		// 没有 ops service 时视为硬开关关闭：与 GET /admin/settings 的 opsEnabled 语义一致。
		require.False(t, resp.Data.OpsMonitoringEnabled)
		require.Equal(t, "auto", resp.Data.OpsQueryModeDefault)
	}
}
