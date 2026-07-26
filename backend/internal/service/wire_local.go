package service

import (
	"github.com/Wei-Shaw/sub2api/internal/config"
)

func ProvideGroupStatusRunnerService(
	repo GroupStatusRepository,
	probeSvc *GroupStatusProbeService,
	cfg *config.Config,
) *GroupStatusRunnerService {
	svc := NewGroupStatusRunnerService(repo, probeSvc, cfg)
	svc.Start()
	return svc
}

// ProvideReferralRewardRecordRepository reuses redeem code persistence for referral rewards.
func ProvideReferralRewardRecordRepository(redeemRepo RedeemCodeRepository) ReferralRewardRecordRepository {
	return redeemRepo
}
