package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// userGroupRateRepoMem 内存版专属倍率管理仓储。
type userGroupRateRepoMem struct {
	listFilter service.UserGroupRateListFilter
	rows       []service.UserGroupRateRow
	usageIDs   []int64
	applied    []service.UserGroupRateChange
}

func (r *userGroupRateRepoMem) List(_ context.Context, f service.UserGroupRateListFilter) (*service.UserGroupRateList, error) {
	r.listFilter = f
	return &service.UserGroupRateList{Items: r.rows, Total: int64(len(r.rows)), UserCount: 1, Page: f.Page, PageSize: f.PageSize}, nil
}

func (r *userGroupRateRepoMem) ListGroups(context.Context) ([]service.UserGroupRateGroupOption, error) {
	return []service.UserGroupRateGroupOption{{ID: 10, Name: "Claude", Platform: "anthropic", Status: "active", SubscriptionType: "standard", RateMultiplier: 1}}, nil
}

func (r *userGroupRateRepoMem) Usage(_ context.Context, ids []int64, _ time.Time) ([]service.UserGroupRateUsage, error) {
	r.usageIDs = ids
	return []service.UserGroupRateUsage{{UserID: 1, GroupID: 10, Requests: 3, ActualCost: 1.25}}, nil
}

func (r *userGroupRateRepoMem) ExistingUserIDs(_ context.Context, ids []int64) ([]int64, error) {
	return ids, nil
}

func (r *userGroupRateRepoMem) ExistingGroupIDs(_ context.Context, ids []int64) ([]int64, error) {
	return ids, nil
}

func (r *userGroupRateRepoMem) ApplyBatch(_ context.Context, changes []service.UserGroupRateChange) error {
	r.applied = changes
	return nil
}

func newUserGroupRateRouter(repo *userGroupRateRepoMem) *gin.Engine {
	gin.SetMode(gin.TestMode)
	h := NewUserGroupRateHandler(service.NewUserGroupRateAdminService(repo))
	router := gin.New()
	router.GET("/api/v1/admin/users/group-rates", h.List)
	router.GET("/api/v1/admin/users/group-rates/groups", h.Groups)
	router.GET("/api/v1/admin/users/group-rates/usage", h.Usage)
	router.POST("/api/v1/admin/users/batch-group-rates", h.ApplyBatch)
	return router
}

func doUserGroupRateRequest(t *testing.T, router *gin.Engine, method, path, body string) (*httptest.ResponseRecorder, map[string]any) {
	t.Helper()
	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	var parsed map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &parsed)
	return rec, parsed
}

func TestUserGroupRateHandler_ListPassesFiltersAndReturnsUserCount(t *testing.T) {
	repo := &userGroupRateRepoMem{rows: []service.UserGroupRateRow{{
		UserID: 1, UserEmail: "a@b.c", GroupID: 10, GroupName: "Claude", GroupRateMultiplier: 1, RateMultiplier: 0.8,
	}}}
	router := newUserGroupRateRouter(repo)

	rec, body := doUserGroupRateRequest(t, router, http.MethodGet,
		"/api/v1/admin/users/group-rates?page=2&page_size=50&search=vip&group_id=10&user_status=active&sort_by=rate_multiplier&sort_order=asc", "")
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, service.UserGroupRateListFilter{
		Page: 2, PageSize: 50, Search: "vip", GroupID: 10, UserStatus: "active", SortBy: "rate_multiplier", SortOrder: "asc",
	}, repo.listFilter)

	data := body["data"].(map[string]any)
	require.EqualValues(t, 1, data["user_count"])
	require.EqualValues(t, 1, data["total"])
	items := data["items"].([]any)
	require.Len(t, items, 1)
	row := items[0].(map[string]any)
	require.Equal(t, "a@b.c", row["user_email"])
	require.EqualValues(t, 0.8, row["rate_multiplier"])
	require.EqualValues(t, 1, row["group_rate_multiplier"])
	require.Contains(t, row, "rpm_override")

	rec, _ = doUserGroupRateRequest(t, router, http.MethodGet, "/api/v1/admin/users/group-rates?group_id=abc", "")
	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestUserGroupRateHandler_GroupsAndUsage(t *testing.T) {
	repo := &userGroupRateRepoMem{}
	router := newUserGroupRateRouter(repo)

	rec, body := doUserGroupRateRequest(t, router, http.MethodGet, "/api/v1/admin/users/group-rates/groups", "")
	require.Equal(t, http.StatusOK, rec.Code)
	groups := body["data"].([]any)
	require.Len(t, groups, 1)
	require.EqualValues(t, 1, groups[0].(map[string]any)["rate_multiplier"])

	rec, body = doUserGroupRateRequest(t, router, http.MethodGet, "/api/v1/admin/users/group-rates/usage?user_ids=1,%202,,1", "")
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, []int64{1, 2}, repo.usageIDs)
	usage := body["data"].([]any)
	require.EqualValues(t, 1.25, usage[0].(map[string]any)["actual_cost"])

	rec, _ = doUserGroupRateRequest(t, router, http.MethodGet, "/api/v1/admin/users/group-rates/usage?user_ids=1,x", "")
	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestUserGroupRateHandler_ApplyBatch(t *testing.T) {
	repo := &userGroupRateRepoMem{}
	router := newUserGroupRateRouter(repo)

	rec, body := doUserGroupRateRequest(t, router, http.MethodPost, "/api/v1/admin/users/batch-group-rates",
		`{"items":[{"user_id":1,"group_id":10,"rate_multiplier":0.75},{"user_id":2,"group_id":10,"rate_multiplier":null}]}`)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	data := body["data"].(map[string]any)
	require.EqualValues(t, 1, data["updated"])
	require.EqualValues(t, 1, data["cleared"])
	require.Len(t, repo.applied, 2)
	require.InDelta(t, 0.75, *repo.applied[0].RateMultiplier, 1e-12)
	require.Nil(t, repo.applied[1].RateMultiplier)
}

func TestUserGroupRateHandler_ApplyBatchRejectsBadInput(t *testing.T) {
	cases := map[string]string{
		"missing rate":   `{"items":[{"user_id":1,"group_id":10}]}`,
		"string rate":    `{"items":[{"user_id":1,"group_id":10,"rate_multiplier":"0.8"}]}`,
		"invalid rate":   `{"items":[{"user_id":1,"group_id":10,"rate_multiplier":0}]}`,
		"empty":          `{"items":[]}`,
		"not json":       `nope`,
		"duplicate pair": `{"items":[{"user_id":1,"group_id":10,"rate_multiplier":1},{"user_id":1,"group_id":10,"rate_multiplier":null}]}`,
	}
	for name, payload := range cases {
		t.Run(name, func(t *testing.T) {
			repo := &userGroupRateRepoMem{}
			rec, _ := doUserGroupRateRequest(t, newUserGroupRateRouter(repo), http.MethodPost, "/api/v1/admin/users/batch-group-rates", payload)
			require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
			require.Nil(t, repo.applied, "nothing is written")
		})
	}
}
