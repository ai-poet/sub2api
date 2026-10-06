//go:build unit

package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

// fork：API Key 删除频率限制 + admin / operator 对创建 / 删除限制的豁免。

type deleteLimitCacheStub struct {
	*apiKeyCacheStub
	deleteCounts map[int64]int64
	incrErr      error
	windows      []time.Duration
}

func (s *deleteLimitCacheStub) IncrementDeleteCount(ctx context.Context, userID int64, window time.Duration) (int64, error) {
	s.windows = append(s.windows, window)
	if s.incrErr != nil {
		return 0, s.incrErr
	}
	s.deleteCounts[userID]++
	return s.deleteCounts[userID], nil
}

func newDeleteLimitService(repo *apiKeyRepoStub, cache *deleteLimitCacheStub, maxDeletes int, role string) *APIKeyService {
	cfg := &config.Config{}
	cfg.APIKeyCreate.MaxDeletesPerUserPerHour = maxDeletes
	return &APIKeyService{
		apiKeyRepo: repo,
		userRepo:   &userRepoStub{user: &User{ID: 7, Role: role}},
		cache:      cache,
		cfg:        cfg,
	}
}

func newDeleteLimitStubs() (*apiKeyRepoStub, *deleteLimitCacheStub) {
	return &apiKeyRepoStub{apiKey: &APIKey{ID: 1, UserID: 7, Key: "k"}},
		&deleteLimitCacheStub{apiKeyCacheStub: &apiKeyCacheStub{}, deleteCounts: map[int64]int64{}}
}

func TestAPIKeyServiceDelete_RateLimited(t *testing.T) {
	repo, cache := newDeleteLimitStubs()
	svc := newDeleteLimitService(repo, cache, 2, RoleUser)

	for i := 0; i < 2; i++ {
		require.NoError(t, svc.Delete(context.Background(), 1, 7))
	}
	err := svc.Delete(context.Background(), 1, 7)
	require.ErrorIs(t, err, ErrAPIKeyDeleteLimited)
	require.Len(t, repo.deletedIDs, 2, "超限的删除不能落库")
	require.Equal(t, apiKeyDeleteCountWindow, cache.windows[0])
}

func TestAPIKeyServiceDelete_PrivilegedRoleExempt(t *testing.T) {
	for _, role := range []string{RoleAdmin, RoleOperator} {
		repo, cache := newDeleteLimitStubs()
		svc := newDeleteLimitService(repo, cache, 1, role)
		for i := 0; i < 3; i++ {
			require.NoError(t, svc.Delete(context.Background(), 1, 7), role)
		}
		require.Len(t, repo.deletedIDs, 3, role)
		require.Empty(t, cache.windows, "特权角色不计数", role)
	}
}

func TestAPIKeyServiceDelete_ZeroLimitDisablesCheck(t *testing.T) {
	repo, cache := newDeleteLimitStubs()
	svc := newDeleteLimitService(repo, cache, 0, RoleUser)
	for i := 0; i < 5; i++ {
		require.NoError(t, svc.Delete(context.Background(), 1, 7))
	}
	require.Empty(t, cache.windows)
}

func TestAPIKeyServiceDelete_RedisErrorFailsOpen(t *testing.T) {
	repo, cache := newDeleteLimitStubs()
	cache.incrErr = errors.New("redis down")
	svc := newDeleteLimitService(repo, cache, 1, RoleUser)
	for i := 0; i < 3; i++ {
		require.NoError(t, svc.Delete(context.Background(), 1, 7))
	}
	require.Len(t, repo.deletedIDs, 3)
}

// 非所有者的删除在频率检查之前就被拒绝，不消耗配额。
func TestAPIKeyServiceDelete_OwnerMismatchDoesNotConsumeDeleteCount(t *testing.T) {
	repo, cache := newDeleteLimitStubs()
	svc := newDeleteLimitService(repo, cache, 1, RoleUser)
	require.ErrorIs(t, svc.Delete(context.Background(), 1, 8), ErrInsufficientPerms)
	require.Zero(t, cache.deleteCounts[8])
}

// admin / operator 同样豁免上游的创建数量与频率限制。
func TestAPIKeyServiceCreate_PrivilegedRoleExempt(t *testing.T) {
	for _, role := range []string{RoleAdmin, RoleOperator} {
		repo, cache := newCreateLimitStubs()
		repo.activeCount = 500
		svc := newCreateLimitService(repo, cache, 1, 1)
		svc.userRepo = &userRepoStub{user: &User{ID: 7, Role: role}}

		for i := 0; i < 3; i++ {
			_, err := svc.Create(context.Background(), 7, CreateAPIKeyRequest{Name: "k"})
			require.NoError(t, err, role)
		}
		require.Len(t, repo.created, 3, role)
		require.Empty(t, cache.windows, "特权角色不计数", role)
	}
}
