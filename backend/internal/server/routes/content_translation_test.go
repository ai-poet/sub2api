package routes

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/handler"
	adminhandler "github.com/Wei-Shaw/sub2api/internal/handler/admin"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// 没有配置（未启用）的服务：查询返回空译文，不碰任何仓储。
func newContentTranslationTestHandlers() *handler.Handlers {
	svc := service.NewContentTranslationService(nil, nil, nil, nil, nil, nil, nil, nil, &config.Config{})
	return &handler.Handlers{
		ContentTranslation: handler.NewContentTranslationHandler(svc),
		Admin:              &handler.AdminHandlers{ContentTranslation: adminhandler.NewContentTranslationHandler(svc)},
	}
}

func TestContentTranslationLookupRouteIsPublicAndBounded(t *testing.T) {
	router := gin.New()
	RegisterContentTranslationRoutes(router.Group("/api/v1"), newContentTranslationTestHandlers(), nil)

	body, _ := json.Marshal(map[string]any{"lang": "en", "texts": []string{"专用分组"}})
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/v1/content-translations/lookup", bytes.NewReader(body)))
	require.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Code int `json:"code"`
		Data struct {
			Lang         string            `json:"lang"`
			Translations map[string]string `json:"translations"`
			Pending      bool              `json:"pending"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)
	require.Equal(t, "en", resp.Data.Lang)
	require.NotNil(t, resp.Data.Translations)

	w = httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/v1/content-translations/lookup",
		strings.NewReader(`{"lang":"fr","texts":[]}`)))
	require.Equal(t, http.StatusBadRequest, w.Code)

	huge := `{"lang":"en","texts":["` + strings.Repeat("a", contentTranslationLookupBodyLimit) + `"]}`
	w = httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/v1/content-translations/lookup", strings.NewReader(huge)))
	require.Equal(t, http.StatusRequestEntityTooLarge, w.Code)
}

func TestContentTranslationAdminRoutesAreAdminOnly(t *testing.T) {
	for role, want := range map[string]int{
		service.RoleOperator: http.StatusForbidden,
		service.RoleUser:     http.StatusForbidden,
		service.RoleAdmin:    http.StatusOK,
	} {
		router := gin.New()
		admin := router.Group("/api/v1/admin")
		admin.Use(func(c *gin.Context) {
			c.Set(string(middleware.ContextKeyUserRole), role)
			c.Next()
		})
		registerContentTranslationAdminRoutes(admin, newContentTranslationTestHandlers())

		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/admin/content-translations/status", nil))
		require.Equal(t, want, w.Code, role)
	}
}

func TestContentTranslationPaySourcesRouteRequiresInternalToken(t *testing.T) {
	t.Setenv("JWT_SECRET", "")
	secret := strings.Repeat("s", 32)
	cfg := &config.Config{}
	cfg.JWT.Secret = secret
	router := gin.New()
	passthrough := func(c *gin.Context) { c.Next() }
	RegisterPayRoutes(router, newContentTranslationTestHandlers(),
		middleware.JWTAuthMiddleware(passthrough), middleware.AdminAuthMiddleware(passthrough), nil, cfg)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodPut, "/api/internal/pay/content-translations/sources",
		strings.NewReader(`{"texts":["套餐"]}`)))
	require.Equal(t, http.StatusUnauthorized, w.Code)

	// 带内部令牌：进入 handler（这里用坏 body 停在参数校验，不碰仓储）
	req := httptest.NewRequest(http.MethodPut, "/api/internal/pay/content-translations/sources", strings.NewReader(`not json`))
	req.Header.Set(internalPayTokenHeader, deriveInternalPayToken(secret))
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}
