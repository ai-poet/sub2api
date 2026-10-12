package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type historiesAvailableGroups struct {
	groups []Group
}

func (r *historiesAvailableGroups) GetAvailableGroups(context.Context, int64) ([]Group, error) {
	return r.groups, nil
}

// historiesRepo 只实现 ListUserHistories 用到的方法；batch 为 true 时额外暴露批量查询。
type historiesRepo struct {
	GroupStatusRepository
	summaries    []GroupStatusSummary
	records      map[int64][]GroupStatusRecord
	batchCalls   [][]int64
	singleCalls  []int64
	summaryCalls [][]int64
}

func (r *historiesRepo) ListSummaries(_ context.Context, groupIDs []int64) ([]GroupStatusSummary, error) {
	r.summaryCalls = append(r.summaryCalls, groupIDs)
	out := make([]GroupStatusSummary, 0, len(r.summaries))
	for _, summary := range r.summaries {
		for _, id := range groupIDs {
			if summary.GroupID == id {
				out = append(out, summary)
			}
		}
	}
	return out, nil
}

func (r *historiesRepo) ListRecordsSince(_ context.Context, groupID int64, _ time.Time) ([]GroupStatusRecord, error) {
	r.singleCalls = append(r.singleCalls, groupID)
	return r.records[groupID], nil
}

type batchHistoriesRepo struct {
	*historiesRepo
}

func (r batchHistoriesRepo) ListRecordStatusesSince(_ context.Context, groupIDs []int64, _ time.Time) (map[int64][]GroupStatusRecord, error) {
	r.batchCalls = append(r.batchCalls, groupIDs)
	out := make(map[int64][]GroupStatusRecord, len(groupIDs))
	for _, id := range groupIDs {
		out[id] = r.records[id]
	}
	return out, nil
}

func newHistoriesFixture() *historiesRepo {
	now := time.Now()
	return &historiesRepo{
		summaries: []GroupStatusSummary{
			{GroupID: 1, Enabled: true},
			{GroupID: 2, Enabled: false},
		},
		records: map[int64][]GroupStatusRecord{
			1: {
				{GroupID: 1, Status: GroupRuntimeStatusUp, ObservedAt: now.Add(-2 * time.Hour)},
				{GroupID: 1, Status: GroupRuntimeStatusUp, ObservedAt: now.Add(-90 * time.Minute)},
				{GroupID: 1, Status: GroupRuntimeStatusDown, ObservedAt: now.Add(-time.Hour)},
				{GroupID: 1, Status: GroupRuntimeStatusUp, ObservedAt: now.Add(-10 * time.Minute)},
			},
		},
	}
}

func TestListUserHistories_OnlyVisibleEnabledGroupsInOneBatch(t *testing.T) {
	repo := newHistoriesFixture()
	svc := &GroupStatusService{
		repo:            batchHistoriesRepo{repo},
		availableGroups: &historiesAvailableGroups{groups: []Group{{ID: 1}, {ID: 2}, {ID: 3}}},
	}

	got, err := svc.ListUserHistories(context.Background(), 7, GroupStatusPeriod24h)
	require.NoError(t, err)

	// 分组 2 关闭了检测、分组 3 没有配置，都不返回；整个请求只发一次批量查询
	require.Len(t, got, 1)
	require.Equal(t, int64(1), got[0].GroupID)
	require.Equal(t, 4, got[0].TotalCount)
	require.InDelta(t, 75.0, got[0].Availability, 0.001)
	require.GreaterOrEqual(t, len(got[0].Buckets), 24)
	require.Equal(t, [][]int64{{1}}, repo.batchCalls)
	require.Empty(t, repo.singleCalls)

	down := 0
	for _, bucket := range got[0].Buckets {
		down += bucket.DownCount
	}
	require.Equal(t, 1, down)
}

func TestListUserHistories_FallsBackToPerGroupReads(t *testing.T) {
	repo := newHistoriesFixture()
	svc := &GroupStatusService{
		repo:            repo,
		availableGroups: &historiesAvailableGroups{groups: []Group{{ID: 1}, {ID: 2}}},
	}

	got, err := svc.ListUserHistories(context.Background(), 7, GroupStatusPeriod7d)
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, []int64{1}, repo.singleCalls)
	require.LessOrEqual(t, len(got[0].Buckets), 8)
}

func TestListUserHistories_GroupWithoutRecordsHasZeroTotal(t *testing.T) {
	repo := newHistoriesFixture()
	repo.records = nil
	svc := &GroupStatusService{
		repo:            batchHistoriesRepo{repo},
		availableGroups: &historiesAvailableGroups{groups: []Group{{ID: 1}}},
	}

	got, err := svc.ListUserHistories(context.Background(), 7, GroupStatusPeriod24h)
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Zero(t, got[0].TotalCount)
	require.Zero(t, got[0].Availability)
}

func TestListUserHistories_RejectsUnknownPeriod(t *testing.T) {
	svc := &GroupStatusService{
		repo:            newHistoriesFixture(),
		availableGroups: &historiesAvailableGroups{},
	}
	_, err := svc.ListUserHistories(context.Background(), 7, "30d")
	require.Error(t, err)
}

func TestListUserHistories_NoVisibleGroupsSkipsRecordQuery(t *testing.T) {
	repo := newHistoriesFixture()
	svc := &GroupStatusService{
		repo:            batchHistoriesRepo{repo},
		availableGroups: &historiesAvailableGroups{},
	}
	got, err := svc.ListUserHistories(context.Background(), 7, GroupStatusPeriod24h)
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Empty(t, got)
	require.Empty(t, repo.batchCalls)
	require.Empty(t, repo.summaryCalls)
}
