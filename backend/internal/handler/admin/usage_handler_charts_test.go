package admin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/usagestats"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// 图表聚合的 repo 桩：返回带账号成本的数据，验证 operator 投影会抹掉它而管理员保留。
type usageChartsRepoStub struct {
	service.UsageLogRepository
	trendStart, trendEnd time.Time
	trendGranularity     string
	trendFilters         usagestats.UsageLogFilters
	groupFilters         usagestats.UsageLogFilters
	modelFilters         usagestats.UsageLogFilters
	modelSource          string
}

func (s *usageChartsRepoStub) GetUsageTrendWithUsageFilters(ctx context.Context, startTime, endTime time.Time, granularity string, filters usagestats.UsageLogFilters) ([]usagestats.TrendDataPoint, error) {
	s.trendStart, s.trendEnd, s.trendGranularity, s.trendFilters = startTime, endTime, granularity, filters
	return []usagestats.TrendDataPoint{{Date: "2026-09-01", Requests: 5, TotalTokens: 100, Cost: 1, ActualCost: 0.5}}, nil
}

func (s *usageChartsRepoStub) GetGroupStatsWithUsageFilters(ctx context.Context, startTime, endTime time.Time, filters usagestats.UsageLogFilters) ([]usagestats.GroupStat, error) {
	s.groupFilters = filters
	return []usagestats.GroupStat{{GroupID: 1, GroupName: "g1", Requests: 5, TotalTokens: 100, Cost: 1, ActualCost: 0.5, AccountCost: 0.3}}, nil
}

func (s *usageChartsRepoStub) GetModelStatsWithUsageFiltersBySource(ctx context.Context, startTime, endTime time.Time, filters usagestats.UsageLogFilters, source string) ([]usagestats.ModelStat, error) {
	s.modelFilters, s.modelSource = filters, source
	return []usagestats.ModelStat{{Model: "gpt-5.6", Requests: 5, TotalTokens: 100, Cost: 1, ActualCost: 0.5, AccountCost: 0.3}}, nil
}

func (s *usageChartsRepoStub) GetStatsWithFilters(ctx context.Context, filters usagestats.UsageLogFilters) (*usagestats.UsageStats, error) {
	accountCost := 0.3
	return &usagestats.UsageStats{TotalAccountCost: &accountCost}, nil
}

func newUsageChartsTestRouter(repo *usageChartsRepoStub, role string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	usageSvc := service.NewUsageService(repo, nil, nil, nil)
	handler := NewUsageHandler(usageSvc, nil, nil, nil)
	router := gin.New()
	if role != "" {
		router.Use(func(c *gin.Context) {
			c.Set(string(middleware.ContextKeyUserRole), role)
			c.Next()
		})
	}
	router.GET("/admin/usage/charts", handler.Charts)
	router.GET("/admin/usage/model-stats", handler.ModelStats)
	router.GET("/admin/usage/stats", handler.Stats)
	return router
}

func getUsageChartsJSON(t *testing.T, router *gin.Engine, target string) (int, map[string]any) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, target, nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		return rec.Code, nil
	}
	var resp struct {
		Data map[string]any `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	return rec.Code, resp.Data
}

func TestAdminUsageCharts_AdminKeepsAccountCostAndPassesFilters(t *testing.T) {
	repo := &usageChartsRepoStub{}
	router := newUsageChartsTestRouter(repo, service.RoleAdmin)

	code, data := getUsageChartsJSON(t, router, "/admin/usage/charts?start_date=2026-09-01&end_date=2026-09-02&granularity=hour&timezone=UTC&user_id=9&model=gpt-5.6&native_compaction_v2=true&request_type=stream")
	require.Equal(t, http.StatusOK, code)

	require.Equal(t, "hour", repo.trendGranularity)
	require.Equal(t, time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC), repo.trendStart.UTC())
	require.Equal(t, time.Date(2026, 9, 3, 0, 0, 0, 0, time.UTC), repo.trendEnd.UTC(), "上边界应为 end_date 次日 00:00")
	require.Equal(t, int64(9), repo.trendFilters.UserID)
	require.Equal(t, "gpt-5.6", repo.trendFilters.Model)
	require.Equal(t, usagestats.ModelSourceRequested, repo.trendFilters.ModelFilterSource)
	require.NotNil(t, repo.trendFilters.NativeCompactionV2)
	require.True(t, *repo.trendFilters.NativeCompactionV2)
	require.NotNil(t, repo.trendFilters.RequestType)
	require.Equal(t, int16(service.RequestTypeStream), *repo.trendFilters.RequestType)
	require.Equal(t, repo.trendFilters, repo.groupFilters, "趋势与分组应使用同一组筛选")

	require.Equal(t, "hour", data["granularity"])
	trend := data["trend"].([]any)
	require.Len(t, trend, 1)
	groups := data["groups"].([]any)
	require.Len(t, groups, 1)
	require.InDelta(t, 0.3, groups[0].(map[string]any)["account_cost"], 1e-9, "管理员保留账号成本")
}

func TestAdminUsageCharts_OperatorStripsAccountCost(t *testing.T) {
	repo := &usageChartsRepoStub{}
	router := newUsageChartsTestRouter(repo, service.RoleOperator)

	code, data := getUsageChartsJSON(t, router, "/admin/usage/charts?start_date=2026-09-01&end_date=2026-09-02&timezone=UTC")
	require.Equal(t, http.StatusOK, code)

	require.Equal(t, "day", repo.trendGranularity)
	groups := data["groups"].([]any)
	require.Len(t, groups, 1)
	group := groups[0].(map[string]any)
	require.InDelta(t, 0, group["account_cost"], 1e-9, "operator 不应看到账号成本")
	require.InDelta(t, 0.5, group["actual_cost"], 1e-9, "其余字段保持不变")
	require.Equal(t, "g1", group["group_name"])
	require.Len(t, data["trend"].([]any), 1)
}

func TestAdminUsageCharts_RejectsBadInput(t *testing.T) {
	router := newUsageChartsTestRouter(&usageChartsRepoStub{}, service.RoleOperator)

	for _, query := range []string{
		"granularity=week",
		"user_id=abc",
		"stream=maybe",
		"start_date=2024-01-01&end_date=2026-09-02&timezone=UTC",
		"start_date=2026-09-05&end_date=2026-09-02&timezone=UTC",
	} {
		code, _ := getUsageChartsJSON(t, router, "/admin/usage/charts?"+query)
		require.Equalf(t, http.StatusBadRequest, code, "query %q", query)
	}
}

func TestAdminUsageModelStats_SourceAndOperatorProjection(t *testing.T) {
	repo := &usageChartsRepoStub{}
	operatorRouter := newUsageChartsTestRouter(repo, service.RoleOperator)

	code, _ := getUsageChartsJSON(t, operatorRouter, "/admin/usage/model-stats?model_source=bogus")
	require.Equal(t, http.StatusBadRequest, code)

	code, data := getUsageChartsJSON(t, operatorRouter, "/admin/usage/model-stats?model_source=upstream&model=gpt-5.6&group_id=4&timezone=UTC")
	require.Equal(t, http.StatusOK, code)
	require.Equal(t, "upstream", repo.modelSource)
	require.Equal(t, "upstream", data["model_source"])
	require.Equal(t, int64(4), repo.modelFilters.GroupID)
	require.Empty(t, repo.modelFilters.Model, "与 dashboard 一致：model 筛选不作用于模型分布本身")
	models := data["models"].([]any)
	require.Len(t, models, 1)
	require.InDelta(t, 0, models[0].(map[string]any)["account_cost"], 1e-9, "operator 不应看到账号成本")
	require.Equal(t, "gpt-5.6", models[0].(map[string]any)["model"])

	adminRouter := newUsageChartsTestRouter(&usageChartsRepoStub{}, service.RoleAdmin)
	code, data = getUsageChartsJSON(t, adminRouter, "/admin/usage/model-stats?timezone=UTC")
	require.Equal(t, http.StatusOK, code)
	require.Equal(t, usagestats.ModelSourceRequested, data["model_source"])
	require.InDelta(t, 0.3, data["models"].([]any)[0].(map[string]any)["account_cost"], 1e-9, "管理员保留账号成本")
}

func TestAdminUsageStats_OperatorHidesTotalAccountCost(t *testing.T) {
	operatorRouter := newUsageChartsTestRouter(&usageChartsRepoStub{}, service.RoleOperator)
	code, data := getUsageChartsJSON(t, operatorRouter, "/admin/usage/stats?nocache=1&timezone=UTC")
	require.Equal(t, http.StatusOK, code)
	require.NotContains(t, data, "total_account_cost", "operator 的汇总不应带账号成本")

	adminRouter := newUsageChartsTestRouter(&usageChartsRepoStub{}, service.RoleAdmin)
	code, data = getUsageChartsJSON(t, adminRouter, "/admin/usage/stats?nocache=1&timezone=UTC")
	require.Equal(t, http.StatusOK, code)
	require.InDelta(t, 0.3, data["total_account_cost"], 1e-9)
}
