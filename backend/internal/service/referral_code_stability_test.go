//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

// conditionalWriteUserRepoStub 模拟支持条件写入的仓储。
// 内嵌 mockUserRepo:其 GetByID 返回的 User 不带 ReferralCode,
// 恰好复现"读取路径映射丢失"的回归场景。
type conditionalWriteUserRepoStub struct {
	mockUserRepo
	dbCode string
}

func (s *conditionalWriteUserRepoStub) SetReferralCodeIfEmpty(_ context.Context, _ int64, code string) (bool, error) {
	if s.dbCode != "" {
		return false, nil
	}
	s.dbCode = code
	return true, nil
}

// 推荐码一经生成必须终身稳定:2026-07-26 的上游合并曾丢失 userEntityToService 的
// ReferralCode 映射,导致每次访问推荐页都重新生成覆盖。以下用例钉住幂等语义。

func TestReferralService_GenerateReferralCode_ReturnsExistingWithoutUpdate(t *testing.T) {
	repo := &mockUserRepo{getByIDUser: &User{ID: 7, ReferralCode: "EXISTING1"}}
	svc := NewReferralService(nil, nil, repo, nil, nil, nil)

	code, err := svc.GenerateReferralCode(context.Background(), 7)
	require.NoError(t, err)
	require.Equal(t, "EXISTING1", code)
	require.Zero(t, repo.updateCalls, "已有推荐码时不得触发任何写入")
}

func TestReferralService_GenerateReferralCode_StableAcrossCalls(t *testing.T) {
	repo := &mockUserRepo{getByIDUser: &User{ID: 7}}
	repo.updateFn = func(_ context.Context, u *User) error {
		repo.getByIDUser.ReferralCode = u.ReferralCode
		return nil
	}
	svc := NewReferralService(nil, nil, repo, nil, nil, nil)

	first, err := svc.GenerateReferralCode(context.Background(), 7)
	require.NoError(t, err)
	require.NotEmpty(t, first)
	require.Equal(t, 1, repo.updateCalls)

	second, err := svc.GenerateReferralCode(context.Background(), 7)
	require.NoError(t, err)
	require.Equal(t, first, second, "连续两次调用必须返回同一个码")
	require.Equal(t, 1, repo.updateCalls, "第二次调用不得重写推荐码")
}

func TestReferralService_GenerateReferralCode_ConditionalWriteHappyPath(t *testing.T) {
	repo := &conditionalWriteUserRepoStub{}
	svc := NewReferralService(nil, nil, repo, nil, nil, nil)

	code, err := svc.GenerateReferralCode(context.Background(), 7)
	require.NoError(t, err)
	require.NotEmpty(t, code)
	require.Equal(t, code, repo.dbCode)
	require.Zero(t, repo.updateCalls, "支持条件写入时不得走掩码 Update 路径")
}

func TestReferralService_GenerateReferralCode_RefusesOverwriteWhenReadPathEmpty(t *testing.T) {
	// 库中已有码(dbCode 非空),但 GetByID 读回的 ReferralCode 为空——
	// 即映射回归重演。此时必须拒绝覆盖并报错,而不是轮换用户的码。
	repo := &conditionalWriteUserRepoStub{dbCode: "DBCODE01"}
	svc := NewReferralService(nil, nil, repo, nil, nil, nil)

	_, err := svc.GenerateReferralCode(context.Background(), 7)
	require.Error(t, err)
	require.Contains(t, err.Error(), "refusing to overwrite")
	require.Equal(t, "DBCODE01", repo.dbCode, "库中的码必须原样保留")
}
