package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// 首页公开统计是匿名接口：前端共享的 apiClient 收到 401 会把访客踢去登录页，
// 所以开关关闭时也必须是 200，并带上可被 CDN 缓存的头。
func TestInsightsPublicStats_DisabledIs200AndCacheable(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := service.NewInsightsService(nil, nil, nil, &config.Config{})
	h := NewInsightsHandler(svc, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/stats/public", nil)
	h.PublicStats(c)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "public, max-age=60", w.Header().Get("Cache-Control"))
	var body struct {
		Data service.PublicStatsResult `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.False(t, body.Data.Enabled)
}

func TestInsightsUserLeaderboard_RequiresAuthSubject(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := service.NewInsightsService(nil, nil, nil, &config.Config{})
	h := NewInsightsHandler(svc, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/leaderboard/users", nil)
	h.UserLeaderboard(c)

	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestInsightsModelLeaderboard_RejectsUnknownWindow(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := service.NewInsightsService(nil, nil, nil, &config.Config{})
	h := NewInsightsHandler(svc, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/leaderboard/models?window=1y", nil)
	h.ModelLeaderboard(c)

	require.Equal(t, http.StatusBadRequest, w.Code)
}
