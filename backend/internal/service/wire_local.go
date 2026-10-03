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

// ProvideAdminApprovalService 构造运维写操作审批服务（fork 本地）。
// 推送钩子（Server酱³）通过 SetNotifier 挂载、可配置的数量上限通过 SetSettingRepository 挂载，
// 保持构造函数签名稳定。
func ProvideAdminApprovalService(
	repo AdminApprovalRepository,
	users *UserService,
	subs *SubscriptionService,
	groups GroupRepository,
	apiKeys APIKeyRepository,
	encryptor SecretEncryptor,
	notifier *ApprovalNotifyService,
	settingRepo SettingRepository,
) *AdminApprovalService {
	svc := NewAdminApprovalService(repo, users, subs, groups, apiKeys, encryptor)
	// 显式判空，避免 nil 指针包进非 nil 接口
	if notifier != nil {
		svc.SetNotifier(notifier)
	}
	if settingRepo != nil {
		svc.SetSettingRepository(settingRepo)
	}
	return svc
}

// ProvideAdminApprovalSweeper 构造并启动过期 / 卡住申请的回收器。
func ProvideAdminApprovalSweeper(svc *AdminApprovalService) *AdminApprovalSweeper {
	w := NewAdminApprovalSweeper(svc, 0)
	w.Start()
	return w
}

// ProvideTicketService 构造工单服务（fork 本地）。
// 推送钩子（Server酱³）通过 SetNotifier 挂载，保持构造函数签名稳定。
func ProvideTicketService(repo SupportTicketRepository, notifier *TicketNotifyService) *TicketService {
	svc := NewTicketService(repo)
	// 显式判空，避免 nil 指针包进非 nil 接口
	if notifier != nil {
		svc.SetNotifier(notifier)
	}
	return svc
}

// ProvideContentModerationService 构造内容审计服务并挂上站内信投递（fork 本地）。
// 用 setter 而非改上游构造函数签名，保持 NewContentModerationService 的现有用法与测试不变。
func ProvideContentModerationService(
	settingRepo SettingRepository,
	repo ContentModerationRepository,
	hashCache ContentModerationHashCache,
	groupRepo GroupRepository,
	userRepo UserRepository,
	proxyRepo ProxyRepository,
	authCacheInvalidator APIKeyAuthCacheInvalidator,
	emailService *EmailService,
	siteMessages *SiteMessageService,
) *ContentModerationService {
	svc := NewContentModerationService(settingRepo, repo, hashCache, groupRepo, userRepo, proxyRepo, authCacheInvalidator, emailService)
	// 显式判空，避免 nil 指针包进非 nil 接口
	if siteMessages != nil {
		svc.SetSiteMessageNotifier(siteMessages)
	}
	return svc
}

// ProvideAppealService 构造封禁申诉服务（fork 本地），并把自己注册为账号状态观察者：
// 管理员修改用户状态（含运维经审批的重放）与风控中心解封时，作废申诉会话并发「账户已恢复」站内信。
// 用类型断言 + setter，保持上游 NewAdminService / NewContentModerationService 签名不变。
func ProvideAppealService(
	store AppealSessionStore,
	users UserRepository,
	settings *SettingService,
	tickets *TicketService,
	siteMessages *SiteMessageService,
	adminService AdminService,
	moderation *ContentModerationService,
) *AppealService {
	svc := NewAppealService(store, users, settings, tickets, siteMessages)
	if setter, ok := adminService.(interface{ SetUserStatusObserver(UserStatusObserver) }); ok {
		setter.SetUserStatusObserver(svc)
	}
	if moderation != nil {
		moderation.SetUserStatusObserver(svc)
	}
	return svc
}

// ProvidePersonalTokenService 构造运维管理员个人令牌服务（fork 本地）。
// 同时把自己作为吊销器挂到 AdminService 上：角色离开 operator 时吊销其令牌。
// 用类型断言 + setter，保持上游 NewAdminService 签名不变。
func ProvidePersonalTokenService(
	repo PersonalTokenRepository,
	userService *UserService,
	settingService *SettingService,
	adminService AdminService,
) *PersonalTokenService {
	svc := NewPersonalTokenService(repo, userService, settingService)
	if setter, ok := adminService.(interface{ SetPersonalTokenRevoker(PersonalTokenRevoker) }); ok {
		setter.SetPersonalTokenRevoker(svc)
	}
	return svc
}
