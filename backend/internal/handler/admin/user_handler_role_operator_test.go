package admin

import (
	"net/http"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// 运维管理员（operator）与管理员一样属于控制台角色：授予该角色需要 step-up。

func TestUpdateUserPromoteToOperatorRequiresStepUp(t *testing.T) {
	router, _ := setupRoleStepUpRouter(t)

	rec := doJSON(t, router, http.MethodPut, "/api/v1/admin/users/1", map[string]any{"role": service.RoleOperator})
	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestUpdateUserAdminToOperatorRequiresStepUp(t *testing.T) {
	router, _ := setupRoleStepUpRouter(t)

	// 目标是其他管理员：改成 operator 也是敏感的角色变更。
	rec := doJSON(t, router, http.MethodPut, "/api/v1/admin/users/2", map[string]any{"role": service.RoleOperator})
	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestUpdateUserKeepOperatorRoleSkipsStepUp(t *testing.T) {
	router, adminSvc := setupRoleStepUpRouter(t)
	adminSvc.users = append(adminSvc.users, service.User{ID: 3, Email: "ops@example.com", Role: service.RoleOperator, Status: service.StatusActive})

	rec := doJSON(t, router, http.MethodPut, "/api/v1/admin/users/3", map[string]any{"role": service.RoleOperator, "notes": "edit"})
	require.Equal(t, http.StatusOK, rec.Code)
}

func TestCreateOperatorUserRequiresStepUp(t *testing.T) {
	router, _ := setupRoleStepUpRouter(t)

	rec := doJSON(t, router, http.MethodPost, "/api/v1/admin/users", map[string]any{
		"email": "ops@example.com", "password": "pass123", "role": service.RoleOperator,
	})
	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

// 管理员不能把自己改成 operator：原来只拦 user，改成 operator 同样会失去后台全权。
func TestUpdateUserSelfDemotionToOperatorRejected(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	adminSvc := newStubAdminService()
	adminSvc.users = append(adminSvc.users, service.User{ID: 2, Email: "admin@example.com", Role: service.RoleAdmin, Status: service.StatusActive})
	h := NewUserHandler(adminSvc, nil, nil, nil, nil, nil, nil)
	router.PUT("/api/v1/admin/users/:id", func(c *gin.Context) {
		c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: 2})
		c.Set(string(middleware.ContextKeyUserRole), service.RoleAdmin)
		h.Update(c)
	})

	for _, role := range []string{service.RoleOperator, service.RoleUser} {
		rec := doJSON(t, router, http.MethodPut, "/api/v1/admin/users/2", map[string]any{"role": role})
		require.Equalf(t, http.StatusBadRequest, rec.Code, "self demotion to %s must be rejected", role)
		require.Contains(t, rec.Body.String(), "cannot demote yourself")
	}

	// 自己改自己但角色仍是 admin：正常编辑。
	rec := doJSON(t, router, http.MethodPut, "/api/v1/admin/users/2", map[string]any{"role": service.RoleAdmin})
	require.Equal(t, http.StatusOK, rec.Code)
}
