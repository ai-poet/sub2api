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

// ProvideGroupStatusProbeService 构造探测服务并挂上 Server酱³ 推送；
// 用 setter 而非改构造函数签名，保持现有测试的 NewGroupStatusProbeService 用法不变。
func ProvideGroupStatusProbeService(
	repo GroupStatusRepository,
	groupRepo GroupRepository,
	scheduler *SchedulerSnapshotService,
	accountTestSvc *AccountTestService,
	gatewaySvc *GatewayService,
	openAIGatewaySvc *OpenAIGatewayService,
	notifySvc *GroupStatusNotifyService,
) *GroupStatusProbeService {
	svc := NewGroupStatusProbeService(repo, groupRepo, scheduler, accountTestSvc, gatewaySvc, openAIGatewaySvc)
	// 显式判空，避免 nil 指针包进非 nil 接口
	if notifySvc != nil {
		svc.SetTransitionNotifier(notifySvc)
	}
	return svc
}

// ProvideReferralRewardRecordRepository reuses redeem code persistence for referral rewards.
func ProvideReferralRewardRecordRepository(redeemRepo RedeemCodeRepository) ReferralRewardRecordRepository {
	return redeemRepo
}
