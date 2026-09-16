package admin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/usagestats"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// 分组 / 账号返回的 admin service 桩：故意带上倍率、凭证等敏感字段，验证投影不会把它们带出去。
type filterOptionsAdminStub struct {
	service.AdminService
	gotSearch   string
	gotPageSize int
}

func (s *filterOptionsAdminStub) GetAllGroupsIncludingInactive(ctx context.Context) ([]service.Group, error) {
	return []service.Group{
		{ID: 2, Name: "zeta", Platform: "openai", RateMultiplier: 1.5, Status: "active", SubscriptionType: "subscription", IsExclusive: true},
		{ID: 1, Name: "alpha", Platform: "anthropic", RateMultiplier: 2, Status: "inactive", SubscriptionType: "standard"},
	}, nil
}

func (s *filterOptionsAdminStub) ListAccounts(ctx context.Context, page, pageSize int, platform, accountType, status, search string, groupID int64, privacyMode string, sortBy, sortOrder string) ([]service.Account, int64, error) {
	s.gotSearch = search
	s.gotPageSize = pageSize
	return []service.Account{
		{ID: 7, Name: "acc-prod", Platform: "openai", Credentials: map[string]any{"api_key": "sk-secret"}},
	}, 1, nil
}

type filterModelsRepoStub struct {
	service.UsageLogRepository
	gotStart, gotEnd time.Time
	gotFilters       usagestats.UsageLogFilters
	gotSource        string
}

func (s *filterModelsRepoStub) GetModelStatsWithUsageFiltersBySource(ctx context.Context, startTime, endTime time.Time, filters usagestats.UsageLogFilters, source string) ([]usagestats.ModelStat, error) {
	s.gotStart, s.gotEnd, s.gotFilters, s.gotSource = startTime, endTime, filters, source
	return []usagestats.ModelStat{
		{Model: "gpt-5.6", Requests: 10},
		{Model: "claude-opus-5", Requests: 3},
		{Model: " gpt-5.6 ", Requests: 1},
		{Model: "", Requests: 1},
	}, nil
}

func newUsageFilterOptionsTestRouter(t *testing.T, adminStub *filterOptionsAdminStub, repo *filterModelsRepoStub) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	usageSvc := service.NewUsageService(repo, nil, nil, nil)
	handler := NewUsageHandler(usageSvc, nil, adminStub, nil)
	router := gin.New()
	router.GET("/admin/usage/filter-groups", handler.FilterGroups)
	router.GET("/admin/usage/search-accounts", handler.SearchAccounts)
	router.GET("/admin/usage/filter-models", handler.FilterModels)
	return router
}

func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func TestAdminUsageFilterGroups_MinimalProjectionSortedByName(t *testing.T) {
	stub := &filterOptionsAdminStub{}
	router := newUsageFilterOptionsTestRouter(t, stub, &filterModelsRepoStub{})

	req := httptest.NewRequest(http.MethodGet, "/admin/usage/filter-groups", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var resp struct {
		Data []map[string]any `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Len(t, resp.Data, 2)
	require.Equal(t, "alpha", resp.Data[0]["name"])
	require.Equal(t, "inactive", resp.Data[0]["status"])
	require.Equal(t, "standard", resp.Data[0]["subscription_type"])
	require.Equal(t, false, resp.Data[0]["is_exclusive"])
	require.Equal(t, "zeta", resp.Data[1]["name"])
	require.Equal(t, "active", resp.Data[1]["status"])
	require.Equal(t, "subscription", resp.Data[1]["subscription_type"])
	require.Equal(t, true, resp.Data[1]["is_exclusive"])
	for _, row := range resp.Data {
		require.Equal(t, []string{"id", "is_exclusive", "name", "platform", "status", "subscription_type"}, sortedKeys(row), "分组筛选项只能带 id / name / platform 与非敏感元数据（状态 / 订阅类型 / 专属）")
	}
}

func TestAdminUsageSearchAccounts_MinimalProjectionAndEmptyKeyword(t *testing.T) {
	stub := &filterOptionsAdminStub{}
	router := newUsageFilterOptionsTestRouter(t, stub, &filterModelsRepoStub{})

	req := httptest.NewRequest(http.MethodGet, "/admin/usage/search-accounts?q=", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	require.JSONEq(t, `{"code":0,"message":"success","data":[]}`, rec.Body.String())
	require.Empty(t, stub.gotSearch, "空关键字不应查库")

	req = httptest.NewRequest(http.MethodGet, "/admin/usage/search-accounts?q=prod", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "prod", stub.gotSearch)
	require.Equal(t, usageFilterAccountSearchLimit, stub.gotPageSize)

	var resp struct {
		Data []map[string]any `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Len(t, resp.Data, 1)
	require.Equal(t, []string{"id", "name", "platform"}, sortedKeys(resp.Data[0]), "账号筛选项只能带 id / name / platform")
	require.NotContains(t, rec.Body.String(), "sk-secret")
}

func TestAdminUsageFilterModels_DedupSortedWithinDateRange(t *testing.T) {
	repo := &filterModelsRepoStub{}
	router := newUsageFilterOptionsTestRouter(t, &filterOptionsAdminStub{}, repo)

	req := httptest.NewRequest(http.MethodGet, "/admin/usage/filter-models?start_date=2026-09-01&end_date=2026-09-02&timezone=UTC", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var resp struct {
		Data []string `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, []string{"claude-opus-5", "gpt-5.6"}, resp.Data)

	require.Equal(t, time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC), repo.gotStart.UTC())
	require.Equal(t, time.Date(2026, 9, 3, 0, 0, 0, 0, time.UTC), repo.gotEnd.UTC(), "上边界应为 end_date 次日 00:00")
	require.Equal(t, usagestats.UsageLogFilters{}, repo.gotFilters, "模型筛选项不应带任何用户 / key / 账号过滤")
	require.Equal(t, usagestats.ModelSourceRequested, repo.gotSource)
}

func TestAdminUsageFilterModels_RejectsBadDates(t *testing.T) {
	router := newUsageFilterOptionsTestRouter(t, &filterOptionsAdminStub{}, &filterModelsRepoStub{})

	for _, query := range []string{
		"start_date=2026-13-01&end_date=2026-09-02",
		"start_date=2026-09-05&end_date=2026-09-02",
		"start_date=2024-01-01&end_date=2026-09-02",
	} {
		req := httptest.NewRequest(http.MethodGet, "/admin/usage/filter-models?"+query+"&timezone=UTC", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		require.Equalf(t, http.StatusBadRequest, rec.Code, "query %q", query)
	}
}
