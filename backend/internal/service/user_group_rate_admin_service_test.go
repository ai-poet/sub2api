package service

import (
	"context"
	"errors"
	"math"
	"testing"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/stretchr/testify/require"
)

type userGroupRateAdminRepoStub struct {
	listFilter UserGroupRateListFilter
	usageIDs   []int64
	usageSince time.Time
	users      map[int64]bool
	groups     map[int64]bool
	applied    []UserGroupRateChange
	applyCalls int
	applyErr   error
}

func (s *userGroupRateAdminRepoStub) List(_ context.Context, filter UserGroupRateListFilter) (*UserGroupRateList, error) {
	s.listFilter = filter
	return &UserGroupRateList{Page: filter.Page, PageSize: filter.PageSize}, nil
}

func (s *userGroupRateAdminRepoStub) ListGroups(context.Context) ([]UserGroupRateGroupOption, error) {
	return nil, nil
}

func (s *userGroupRateAdminRepoStub) Usage(_ context.Context, ids []int64, since time.Time) ([]UserGroupRateUsage, error) {
	s.usageIDs = ids
	s.usageSince = since
	return []UserGroupRateUsage{}, nil
}

func (s *userGroupRateAdminRepoStub) ExistingUserIDs(_ context.Context, ids []int64) ([]int64, error) {
	return filterExisting(ids, s.users), nil
}

func (s *userGroupRateAdminRepoStub) ExistingGroupIDs(_ context.Context, ids []int64) ([]int64, error) {
	return filterExisting(ids, s.groups), nil
}

func (s *userGroupRateAdminRepoStub) ApplyBatch(_ context.Context, changes []UserGroupRateChange) error {
	s.applyCalls++
	s.applied = changes
	return s.applyErr
}

func filterExisting(ids []int64, present map[int64]bool) []int64 {
	var out []int64
	for _, id := range ids {
		if present[id] {
			out = append(out, id)
		}
	}
	return out
}

func newUserGroupRateAdminServiceForTest() (*UserGroupRateAdminService, *userGroupRateAdminRepoStub) {
	repo := &userGroupRateAdminRepoStub{
		users:  map[int64]bool{1: true, 2: true},
		groups: map[int64]bool{10: true, 11: true},
	}
	return NewUserGroupRateAdminService(repo), repo
}

func ugrRate(v float64) *float64 { return &v }

func TestUserGroupRateAdminService_ListNormalizesFilter(t *testing.T) {
	svc, repo := newUserGroupRateAdminServiceForTest()

	_, err := svc.List(context.Background(), UserGroupRateListFilter{Page: 0, PageSize: 5000, Search: "  a@b  ", SortBy: "password; DROP", SortOrder: "ASC"})
	require.NoError(t, err)
	require.Equal(t, 1, repo.listFilter.Page)
	require.Equal(t, UserGroupRateBatchMaxItems, repo.listFilter.PageSize)
	require.Equal(t, "a@b", repo.listFilter.Search)
	require.Equal(t, "updated_at", repo.listFilter.SortBy, "unknown sort fields fall back")
	require.Equal(t, "asc", repo.listFilter.SortOrder)

	_, err = svc.List(context.Background(), UserGroupRateListFilter{SortBy: "discount", SortOrder: "weird"})
	require.NoError(t, err)
	require.Equal(t, "discount", repo.listFilter.SortBy)
	require.Equal(t, "desc", repo.listFilter.SortOrder)
	require.Equal(t, 20, repo.listFilter.PageSize)
}

func TestUserGroupRateAdminService_ApplyBatchWritesNormalizedChanges(t *testing.T) {
	svc, repo := newUserGroupRateAdminServiceForTest()

	result, err := svc.ApplyBatch(context.Background(), []UserGroupRateChange{
		{UserID: 1, GroupID: 10, RateMultiplier: ugrRate(0.123456)},
		{UserID: 1, GroupID: 11, RateMultiplier: nil},
		{UserID: 2, GroupID: 10, RateMultiplier: ugrRate(1.5)},
	})
	require.NoError(t, err)
	require.Equal(t, &UserGroupRateBatchResult{Updated: 2, Cleared: 1}, result)
	require.Equal(t, 1, repo.applyCalls)
	require.Len(t, repo.applied, 3)
	require.InDelta(t, 0.1235, *repo.applied[0].RateMultiplier, 1e-12, "rounded to DECIMAL(10,4)")
	require.Nil(t, repo.applied[1].RateMultiplier)
}

func TestUserGroupRateAdminService_ApplyBatchRejectsInvalidInput(t *testing.T) {
	tooMany := make([]UserGroupRateChange, UserGroupRateBatchMaxItems+1)
	for i := range tooMany {
		tooMany[i] = UserGroupRateChange{UserID: 1, GroupID: int64(i + 1), RateMultiplier: ugrRate(1)}
	}
	cases := []struct {
		name    string
		changes []UserGroupRateChange
		want    error
	}{
		{"empty", nil, ErrUserGroupRateBatchEmpty},
		{"too many", tooMany, ErrUserGroupRateBatchTooLarge},
		{"bad id", []UserGroupRateChange{{UserID: 0, GroupID: 10, RateMultiplier: ugrRate(1)}}, ErrUserGroupRateInvalidID},
		{"duplicate", []UserGroupRateChange{{UserID: 1, GroupID: 10, RateMultiplier: ugrRate(1)}, {UserID: 1, GroupID: 10}}, ErrUserGroupRateDuplicatePair},
		{"zero", []UserGroupRateChange{{UserID: 1, GroupID: 10, RateMultiplier: ugrRate(0)}}, ErrUserGroupRateInvalidRate},
		{"rounds to zero", []UserGroupRateChange{{UserID: 1, GroupID: 10, RateMultiplier: ugrRate(0.00004)}}, ErrUserGroupRateInvalidRate},
		{"negative", []UserGroupRateChange{{UserID: 1, GroupID: 10, RateMultiplier: ugrRate(-1)}}, ErrUserGroupRateInvalidRate},
		{"overflow", []UserGroupRateChange{{UserID: 1, GroupID: 10, RateMultiplier: ugrRate(1e6)}}, ErrUserGroupRateInvalidRate},
		{"nan", []UserGroupRateChange{{UserID: 1, GroupID: 10, RateMultiplier: ugrRate(math.NaN())}}, ErrUserGroupRateInvalidRate},
		{"inf", []UserGroupRateChange{{UserID: 1, GroupID: 10, RateMultiplier: ugrRate(math.Inf(1))}}, ErrUserGroupRateInvalidRate},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc, repo := newUserGroupRateAdminServiceForTest()
			_, err := svc.ApplyBatch(context.Background(), tc.changes)
			require.ErrorIs(t, err, tc.want)
			require.Equal(t, 400, infraerrors.Code(err))
			require.Zero(t, repo.applyCalls, "nothing is written on validation failure")
		})
	}
}

func TestUserGroupRateAdminService_ApplyBatchRejectsMissingUsersAndGroups(t *testing.T) {
	svc, repo := newUserGroupRateAdminServiceForTest()

	_, err := svc.ApplyBatch(context.Background(), []UserGroupRateChange{
		{UserID: 1, GroupID: 10, RateMultiplier: ugrRate(1)},
		{UserID: 9, GroupID: 10, RateMultiplier: ugrRate(1)},
		{UserID: 7, GroupID: 10, RateMultiplier: ugrRate(1)},
	})
	require.ErrorIs(t, err, ErrUserGroupRateUserNotFound)
	var appErr *infraerrors.ApplicationError
	require.True(t, errors.As(err, &appErr))
	require.Equal(t, "7,9", appErr.Metadata["ids"])

	_, err = svc.ApplyBatch(context.Background(), []UserGroupRateChange{{UserID: 1, GroupID: 99, RateMultiplier: ugrRate(1)}})
	require.ErrorIs(t, err, ErrUserGroupRateGroupNotFound)
	require.Zero(t, repo.applyCalls)
}

func TestUserGroupRateAdminService_ApplyBatchPropagatesRepoError(t *testing.T) {
	svc, repo := newUserGroupRateAdminServiceForTest()
	repo.applyErr = errors.New("db down")

	_, err := svc.ApplyBatch(context.Background(), []UserGroupRateChange{{UserID: 1, GroupID: 10, RateMultiplier: ugrRate(1)}})
	require.EqualError(t, err, "db down")
}

func TestUserGroupRateAdminService_UsageDeduplicatesAndCaps(t *testing.T) {
	svc, repo := newUserGroupRateAdminServiceForTest()
	now := time.Date(2026, 10, 14, 12, 0, 0, 0, time.UTC)
	svc.now = func() time.Time { return now }

	_, err := svc.Usage(context.Background(), []int64{3, 3, 0, -1, 4})
	require.NoError(t, err)
	require.Equal(t, []int64{3, 4}, repo.usageIDs)
	require.Equal(t, now.Add(-30*24*time.Hour), repo.usageSince)

	out, err := svc.Usage(context.Background(), nil)
	require.NoError(t, err)
	require.Empty(t, out)

	many := make([]int64, UserGroupRateUsageMaxUsers+1)
	for i := range many {
		many[i] = int64(i + 1)
	}
	_, err = svc.Usage(context.Background(), many)
	require.ErrorIs(t, err, ErrUserGroupRateUsageTooMany)
}

func TestUserGroupRateAdminService_NilRepoIsUnavailable(t *testing.T) {
	svc := NewUserGroupRateAdminService(nil)
	_, err := svc.ApplyBatch(context.Background(), []UserGroupRateChange{{UserID: 1, GroupID: 10, RateMultiplier: ugrRate(1)}})
	require.ErrorIs(t, err, ErrUserGroupRateAdminUnavailable)
}
