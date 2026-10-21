package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
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

// 没传日期时取服务器时区的今天，并把实际范围、排名指标和条数回给前端；用户榜默认关闭时不查库。
func TestInsightsUserLeaderboard_DefaultsToTodayAndEchoesTheRange(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := service.NewInsightsService(nil, nil, nil, &config.Config{})
	h := NewInsightsHandler(svc, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/leaderboard/users?metric=cost", nil)
	c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 7})
	h.UserLeaderboard(c)

	require.Equal(t, http.StatusOK, w.Code)
	var body struct {
		Data map[string]any `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	today := timezone.Now().Format("2006-01-02")
	require.Equal(t, false, body.Data["enabled"])
	require.Equal(t, today, body.Data["start_date"])
	require.Equal(t, today, body.Data["end_date"])
	require.Equal(t, "tokens", body.Data["sort_by"], "the metric query parameter is ignored")
	require.Equal(t, false, body.Data["show_cost"])
	require.EqualValues(t, 20, body.Data["limit"])
	require.NotContains(t, body.Data, "metric")
	require.NotContains(t, body.Data, "window")
}

func TestInsightsLeaderboards_RejectBadDateRanges(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := service.NewInsightsService(nil, nil, nil, &config.Config{})
	h := NewInsightsHandler(svc, nil)

	cases := map[string]string{
		"bad format":     "start_date=2026-13-01&end_date=2026-10-12",
		"reversed":       "start_date=2026-10-12&end_date=2026-10-01",
		"over 90 days":   "start_date=2026-01-01&end_date=2026-10-12",
		"not a date":     "start_date=today",
		"slashed format": "start_date=2026/10/01&end_date=2026/10/02",
	}
	for name, query := range cases {
		for _, endpoint := range []string{"models", "users"} {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/leaderboard/"+endpoint+"?"+query, nil)
			c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 7})
			if endpoint == "models" {
				h.ModelLeaderboard(c)
			} else {
				h.UserLeaderboard(c)
			}
			require.Equal(t, http.StatusBadRequest, w.Code, "%s %s", endpoint, name)
		}
	}
}

// 只有管理员和运维管理员拿到实名视角；普通用户和没有角色的上下文都是匿名视角。
func TestInsightsLeaderboardViewer_OnlyPrivilegedRolesAreIdentified(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cases := map[string]bool{
		service.RoleAdmin:    true,
		service.RoleOperator: true,
		service.RoleUser:     false,
		"":                   false,
	}
	for role, identified := range cases {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		if role != "" {
			c.Set(string(middleware2.ContextKeyUserRole), role)
		}
		viewer := leaderboardViewer(c, 9)
		require.Equal(t, int64(9), viewer.UserID, role)
		require.Equal(t, identified, viewer.Identified, role)
	}
}
