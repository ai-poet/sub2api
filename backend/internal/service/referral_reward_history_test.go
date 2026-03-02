//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

type referralRepoStubForRewardHistory struct {
	referral       *UserReferral
	updatedStatus  string
	updatedRewards *ReferralRewardSnapshot
}

func (s *referralRepoStubForRewardHistory) Create(context.Context, *UserReferral) error { return nil }
func (s *referralRepoStubForRewardHistory) GetByRefereeID(_ context.Context, refereeID int64) (*UserReferral, error) {
	if s.referral == nil || s.referral.RefereeID != refereeID {
		return nil, ErrReferralNotFound
	}
	copyRef := *s.referral
	return &copyRef, nil
}
func (s *referralRepoStubForRewardHistory) GetByID(context.Context, int64) (*UserReferral, error) {
	return nil, ErrReferralNotFound
}
func (s *referralRepoStubForRewardHistory) UpdateStatus(_ context.Context, id int64, status string, rewards *ReferralRewardSnapshot) error {
	if s.referral != nil && s.referral.ID == id {
		s.referral.Status = status
	}
	s.updatedStatus = status
	s.updatedRewards = rewards
	return nil
}
func (s *referralRepoStubForRewardHistory) RunInTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}
func (s *referralRepoStubForRewardHistory) CountByReferrerID(context.Context, int64) (int, error) {
	return 0, nil
}
func (s *referralRepoStubForRewardHistory) CountByReferrerIDAndStatus(context.Context, int64, string) (int, error) {
	return 0, nil
}
func (s *referralRepoStubForRewardHistory) SumReferrerBalanceReward(context.Context, int64) (float64, error) {
	return 0, nil
}
func (s *referralRepoStubForRewardHistory) ListByReferrerID(context.Context, int64, pagination.PaginationParams) ([]UserReferral, *pagination.PaginationResult, error) {
	return nil, &pagination.PaginationResult{}, nil
}
func (s *referralRepoStubForRewardHistory) ListAll(context.Context, pagination.PaginationParams) ([]UserReferral, *pagination.PaginationResult, error) {
	return nil, &pagination.PaginationResult{}, nil
}

type referralCacheStubForRewardHistory struct{}

func (s *referralCacheStubForRewardHistory) AcquireRewardLock(context.Context, int64) (bool, error) {
	return true, nil
}
func (s *referralCacheStubForRewardHistory) ReleaseRewardLock(context.Context, int64) error {
	return nil
}

type rewardRecordRepoStub struct {
	created []*RedeemCode
}

func (s *rewardRecordRepoStub) Create(_ context.Context, code *RedeemCode) error {
	copyCode := *code
	s.created = append(s.created, &copyCode)
	return nil
}

func TestReferralService_TriggerReferralReward_CreatesRewardHistoryRecords(t *testing.T) {
	referralRepo := &referralRepoStubForRewardHistory{
		referral: &UserReferral{
			ID:         1,
			ReferrerID: 1001,
			RefereeID:  2002,
			Status:     ReferralStatusPending,
		},
	}
	settingRepo := &stubSettingRepoForReferralService{
		values: map[string]string{
			SettingKeyReferralEnabled:               "true",
			SettingKeyReferralReferrerBalanceReward: "12.50",
			SettingKeyReferralRefereeBalanceReward:  "7.75",
		},
	}
	rewardRepo := &rewardRecordRepoStub{}

	type balanceUpdate struct {
		userID int64
		amount float64
	}
	updates := make([]balanceUpdate, 0, 2)
	userRepo := &mockUserRepo{
		updateBalanceFn: func(_ context.Context, id int64, amount float64) error {
			updates = append(updates, balanceUpdate{userID: id, amount: amount})
			return nil
		},
	}

	svc := NewReferralService(
		referralRepo,
		&referralCacheStubForRewardHistory{},
		userRepo,
		settingRepo,
		rewardRepo,
		nil,
	)

	svc.TriggerReferralReward(context.Background(), 2002)

	require.Equal(t, ReferralStatusRewarded, referralRepo.referral.Status)
	require.Equal(t, ReferralStatusRewarded, referralRepo.updatedStatus)
	require.NotNil(t, referralRepo.updatedRewards)
	require.InDelta(t, 12.5, referralRepo.updatedRewards.ReferrerBalanceReward, 0.000001)
	require.InDelta(t, 7.75, referralRepo.updatedRewards.RefereeBalanceReward, 0.000001)

	require.Len(t, updates, 2)
	require.Equal(t, int64(1001), updates[0].userID)
	require.InDelta(t, 12.5, updates[0].amount, 0.000001)
	require.Equal(t, int64(2002), updates[1].userID)
	require.InDelta(t, 7.75, updates[1].amount, 0.000001)

	require.Len(t, rewardRepo.created, 2)
	require.Equal(t, AdjustmentTypeReferralReward, rewardRepo.created[0].Type)
	require.Equal(t, StatusUsed, rewardRepo.created[0].Status)
	require.Equal(t, int64(1001), *rewardRepo.created[0].UsedBy)
	require.InDelta(t, 12.5, rewardRepo.created[0].Value, 0.000001)
	require.Contains(t, rewardRepo.created[0].Notes, "首次充值")
	require.NotNil(t, rewardRepo.created[0].UsedAt)

	require.Equal(t, AdjustmentTypeReferralReward, rewardRepo.created[1].Type)
	require.Equal(t, StatusUsed, rewardRepo.created[1].Status)
	require.Equal(t, int64(2002), *rewardRepo.created[1].UsedBy)
	require.InDelta(t, 7.75, rewardRepo.created[1].Value, 0.000001)
	require.Contains(t, rewardRepo.created[1].Notes, "首次充值")
	require.NotNil(t, rewardRepo.created[1].UsedAt)
}
