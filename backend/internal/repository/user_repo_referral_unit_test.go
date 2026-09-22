package repository

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

// 钉住 userEntityToService 的 ReferralCode 映射:2026-07-26 的上游合并曾把该字段
// 从映射结构体字面量里吞掉,导致 GetByID 恒返回空码、推荐码被反复重新生成覆盖、
// 已分发的邀请链接全部失效。此用例在映射再次丢失时立即变红。
func TestUserRepositoryGetByIDRoundTripsReferralCode(t *testing.T) {
	repo, client := newUserEntRepo(t)
	ctx := context.Background()

	require.NoError(t, repo.Create(ctx, &service.User{
		Email:        "referral@example.com",
		Username:     "referral-user",
		PasswordHash: "hash",
		Role:         service.RoleUser,
		Status:       service.StatusActive,
	}))
	created, err := repo.GetByEmail(ctx, "referral@example.com")
	require.NoError(t, err)

	_, err = client.User.UpdateOneID(created.ID).SetReferralCode("REFCODE1").Save(ctx)
	require.NoError(t, err)

	got, err := repo.GetByID(ctx, created.ID)
	require.NoError(t, err)
	require.Equal(t, "REFCODE1", got.ReferralCode)

	byCode, err := repo.GetByReferralCode(ctx, "REFCODE1")
	require.NoError(t, err)
	require.Equal(t, created.ID, byCode.ID)
	require.Equal(t, "REFCODE1", byCode.ReferralCode)
}

func TestUserRepositorySetReferralCodeIfEmptyRefusesOverwrite(t *testing.T) {
	repo, _ := newUserEntRepo(t)
	ctx := context.Background()

	require.NoError(t, repo.Create(ctx, &service.User{
		Email:        "cond@example.com",
		Username:     "cond-user",
		PasswordHash: "hash",
		Role:         service.RoleUser,
		Status:       service.StatusActive,
	}))
	created, err := repo.GetByEmail(ctx, "cond@example.com")
	require.NoError(t, err)

	written, err := repo.SetReferralCodeIfEmpty(ctx, created.ID, "FIRST001")
	require.NoError(t, err)
	require.True(t, written)

	// 已有码时条件写入必须拒绝,库中的码原样保留
	written, err = repo.SetReferralCodeIfEmpty(ctx, created.ID, "SECOND02")
	require.NoError(t, err)
	require.False(t, written)

	got, err := repo.GetByID(ctx, created.ID)
	require.NoError(t, err)
	require.Equal(t, "FIRST001", got.ReferralCode)
}
