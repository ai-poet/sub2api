package handler

import (
	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/handler/admin"
	"github.com/Wei-Shaw/sub2api/internal/securityaudit"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/google/wire"
)

// ProvideAdminHandlers creates the AdminHandlers struct
func ProvideAdminHandlers(
	dashboardHandler *admin.DashboardHandler,
	userHandler *admin.UserHandler,
	groupHandler *admin.GroupHandler,
	accountHandler *admin.AccountHandler,
	announcementHandler *admin.AnnouncementHandler,
	dataManagementHandler *admin.DataManagementHandler,
	backupHandler *admin.BackupHandler,
	oauthHandler *admin.OAuthHandler,
	openaiOAuthHandler *admin.OpenAIOAuthHandler,
	geminiOAuthHandler *admin.GeminiOAuthHandler,
	antigravityOAuthHandler *admin.AntigravityOAuthHandler,
	grokOAuthHandler *admin.GrokOAuthHandler,
	cnProviderHandler *admin.CNProviderHandler,
	proxyHandler *admin.ProxyHandler,
	redeemHandler *admin.RedeemHandler,
	promoHandler *admin.PromoHandler,
	settingHandler *admin.SettingHandler,
	opsHandler *admin.OpsHandler,
	systemHandler *admin.SystemHandler,
	subscriptionHandler *admin.SubscriptionHandler,
	usageHandler *admin.UsageHandler,
	userAttributeHandler *admin.UserAttributeHandler,
	errorPassthroughHandler *admin.ErrorPassthroughHandler,
	referralHandler *admin.ReferralHandler,
	tlsFingerprintProfileHandler *admin.TLSFingerprintProfileHandler,
	pluginHandler *admin.PluginHandler,
	apiKeyHandler *admin.AdminAPIKeyHandler,
	scheduledTestHandler *admin.ScheduledTestHandler,
	channelHandler *admin.ChannelHandler,
	contentModerationHandler *admin.ContentModerationHandler,
	promptAuditHandler *securityaudit.PromptAdminHandler,
	complianceHandler *admin.ComplianceHandler,
	auditLogHandler *admin.AuditLogHandler,
	upstreamBillingProbe *service.UpstreamBillingProbeService,
	ollamaCloudUsage *service.OllamaCloudUsageService,
) *AdminHandlers {
	accountHandler.SetUpstreamBillingProbeService(upstreamBillingProbe)
	accountHandler.SetOllamaCloudUsageService(ollamaCloudUsage)
	return &AdminHandlers{
		Dashboard:             dashboardHandler,
		User:                  userHandler,
		Group:                 groupHandler,
		Account:               accountHandler,
		Announcement:          announcementHandler,
		DataManagement:        dataManagementHandler,
		Backup:                backupHandler,
		OAuth:                 oauthHandler,
		OpenAIOAuth:           openaiOAuthHandler,
		GeminiOAuth:           geminiOAuthHandler,
		AntigravityOAuth:      antigravityOAuthHandler,
		GrokOAuth:             grokOAuthHandler,
		CNProvider:            cnProviderHandler,
		Proxy:                 proxyHandler,
		Redeem:                redeemHandler,
		Promo:                 promoHandler,
		Setting:               settingHandler,
		Ops:                   opsHandler,
		System:                systemHandler,
		Subscription:          subscriptionHandler,
		Usage:                 usageHandler,
		UserAttribute:         userAttributeHandler,
		ErrorPassthrough:      errorPassthroughHandler,
		Referral:              referralHandler,
		TLSFingerprintProfile: tlsFingerprintProfileHandler,
		APIKey:                apiKeyHandler,
		ScheduledTest:         scheduledTestHandler,
		Channel:               channelHandler,
		ContentModeration:     contentModerationHandler,
		PromptAudit:           promptAuditHandler,
		Compliance:            complianceHandler,
		AuditLog:              auditLogHandler,
		Plugin:                pluginHandler,
	}
}

func ProvideGatewayHandler(
	gatewayService *service.GatewayService,
	openAIGatewayService *service.OpenAIGatewayService,
	geminiCompatService *service.GeminiMessagesCompatService,
	antigravityGatewayService *service.AntigravityGatewayService,
	userService *service.UserService,
	concurrencyService *service.ConcurrencyService,
	billingCacheService *service.BillingCacheService,
	usageService *service.UsageService,
	apiKeyService *service.APIKeyService,
	usageRecordWorkerPool *service.UsageRecordWorkerPool,
	errorPassthroughService *service.ErrorPassthroughService,
	contentModerationService *service.ContentModerationService,
	userMsgQueueService *service.UserMessageQueueService,
	cfg *config.Config,
	settingService *service.SettingService,
	coordinator *securityaudit.Coordinator,
) *GatewayHandler {
	h := NewGatewayHandler(gatewayService, openAIGatewayService, geminiCompatService, antigravityGatewayService,
		userService, concurrencyService, billingCacheService, usageService, apiKeyService, usageRecordWorkerPool,
		errorPassthroughService, contentModerationService, userMsgQueueService, cfg, settingService)
	h.securityAuditCoordinator = coordinator
	return h
}

func ProvideOpenAIGatewayHandler(
	gatewayService *service.OpenAIGatewayService,
	pluginManager *service.PluginManager,
	concurrencyService *service.ConcurrencyService,
	billingCacheService *service.BillingCacheService,
	apiKeyService *service.APIKeyService,
	usageRecordWorkerPool *service.UsageRecordWorkerPool,
	errorPassthroughService *service.ErrorPassthroughService,
	contentModerationService *service.ContentModerationService,
	opsService *service.OpsService,
	grokQuotaService *service.GrokQuotaService,
	cfg *config.Config,
	coordinator *securityaudit.Coordinator,
) *OpenAIGatewayHandler {
	gatewayService.SetPluginManager(pluginManager)
	h := NewOpenAIGatewayHandler(gatewayService, concurrencyService, billingCacheService, apiKeyService,
		usageRecordWorkerPool, errorPassthroughService, contentModerationService, opsService, cfg)
	h.securityAuditCoordinator = coordinator
	h.grokMediaEligibilityProber = grokQuotaService
	return h
}

func ProvideBatchImageHandler(
	batchService *service.BatchImagePublicService,
	download *service.BatchImageDownloadService,
	cleanup *service.BatchImageCleanupService,
	openAI *OpenAIGatewayHandler,
) *BatchImageHandler {
	h := NewBatchImageHandler(batchService, download, cleanup)
	h.openAI = openAI
	return h
}

// ProvideSystemHandler creates admin.SystemHandler with UpdateService
func ProvideSystemHandler(updateService *service.UpdateService, lockService *service.SystemOperationLockService) *admin.SystemHandler {
	return admin.NewSystemHandler(updateService, lockService)
}

// ProvideSettingHandler creates SettingHandler with version from BuildInfo
func ProvideSettingHandler(settingService *service.SettingService, buildInfo BuildInfo, notificationEmailService *service.NotificationEmailService) *SettingHandler {
	h := NewSettingHandler(settingService, buildInfo.Version)
	h.SetNotificationEmailService(notificationEmailService)
	return h
}

// ProvideAdminSettingHandler creates admin.SettingHandler with notification template APIs.
func ProvideAdminSettingHandler(settingService *service.SettingService, emailService *service.EmailService, turnstileService *service.TurnstileService, aliyunCaptchaService *service.AliyunCaptchaService, opsService *service.OpsService, userAttributeService *service.UserAttributeService, notificationEmailService *service.NotificationEmailService, totpService *service.TotpService, userService *service.UserService) *admin.SettingHandler {
	h := admin.NewSettingHandler(settingService, emailService, turnstileService, opsService, userAttributeService)
	h.SetNotificationEmailService(notificationEmailService)
	h.SetAliyunCaptchaService(aliyunCaptchaService)
	h.SetStepUpDeps(totpService, userService)
	return h
}

func ProvideAdminGroupHandler(
	adminService service.AdminService,
	dashboardService *service.DashboardService,
	groupCapacityService *service.GroupCapacityService,
	groupStatusService *service.GroupStatusService,
	groupStatusProbeSvc *service.GroupStatusProbeService,
) *admin.GroupHandler {
	h := admin.NewGroupHandler(adminService, dashboardService, groupCapacityService)
	h.SetGroupStatusServices(groupStatusService, groupStatusProbeSvc)
	return h
}

// ProvideHandlers creates the Handlers struct
func ProvideHandlers(
	authHandler *AuthHandler,
	userHandler *UserHandler,
	apiKeyHandler *APIKeyHandler,
	usageHandler *UsageHandler,
	redeemHandler *RedeemHandler,
	subscriptionHandler *SubscriptionHandler,
	announcementHandler *AnnouncementHandler,
	adminHandlers *AdminHandlers,
	gatewayHandler *GatewayHandler,
	openaiGatewayHandler *OpenAIGatewayHandler,
	settingHandler *SettingHandler,
	totpHandler *TotpHandler,
	referralHandler *ReferralHandler,
	modelCatalogHandler *ModelCatalogHandler,
	publicPricingHandler *PublicPricingHandler,
	groupStatusHandler *GroupStatusHandler,
	passkeyHandler *PasskeyHandler,
	availableChannelHandler *AvailableChannelHandler,
	asyncImageHandler *AsyncImageHandler,
	batchImageHandler *BatchImageHandler,
	payBridgeHandler *PayBridgeHandler,
	_ *service.IdempotencyCoordinator,
	_ *service.IdempotencyCleanupService,
	_ *service.OpenAIQuotaAutoResetService,
) *Handlers {
	return &Handlers{
		Auth:             authHandler,
		User:             userHandler,
		APIKey:           apiKeyHandler,
		Usage:            usageHandler,
		Redeem:           redeemHandler,
		Subscription:     subscriptionHandler,
		Announcement:     announcementHandler,
		Admin:            adminHandlers,
		Gateway:          gatewayHandler,
		OpenAIGateway:    openaiGatewayHandler,
		Setting:          settingHandler,
		Totp:             totpHandler,
		Referral:         referralHandler,
		ModelCatalog:     modelCatalogHandler,
		PublicPricing:    publicPricingHandler,
		GroupStatus:      groupStatusHandler,
		Passkey:          passkeyHandler,
		AvailableChannel: availableChannelHandler,
		AsyncImage:       asyncImageHandler,
		BatchImage:       batchImageHandler,
		PayBridge:        payBridgeHandler,
	}
}

// ProviderSet is the Wire provider set for all handlers
var ProviderSet = wire.NewSet(
	// Top-level handlers
	NewAuthHandler,
	NewUserHandler,
	NewAPIKeyHandler,
	NewUsageHandler,
	NewRedeemHandler,
	NewSubscriptionHandler,
	NewAnnouncementHandler,
	ProvideGatewayHandler,
	ProvideOpenAIGatewayHandler,
	NewTotpHandler,
	NewReferralHandler,
	NewModelCatalogHandler,
	NewPublicPricingHandler,
	NewGroupStatusHandler,
	NewPasskeyHandler,
	ProvideSettingHandler,
	NewAvailableChannelHandler,
	NewAsyncImageHandler,
	ProvideBatchImageHandler,
	NewPayBridgeHandler,

	// Admin handlers
	admin.NewDashboardHandler,
	admin.NewUserHandler,
	ProvideAdminGroupHandler,
	admin.ProvideAccountHandler,
	admin.NewAnnouncementHandler,
	admin.NewDataManagementHandler,
	admin.NewBackupHandler,
	admin.NewOAuthHandler,
	admin.NewOpenAIOAuthHandler,
	admin.NewGeminiOAuthHandler,
	admin.NewAntigravityOAuthHandler,
	admin.NewGrokOAuthHandler,
	admin.NewCNProviderHandler,
	admin.NewProxyHandler,
	admin.NewRedeemHandler,
	admin.NewPromoHandler,
	ProvideAdminSettingHandler,
	admin.NewOpsHandler,
	ProvideSystemHandler,
	admin.NewSubscriptionHandler,
	admin.NewUsageHandler,
	admin.NewUserAttributeHandler,
	admin.NewErrorPassthroughHandler,
	admin.NewReferralHandler,
	admin.NewTLSFingerprintProfileHandler,
	admin.NewPluginHandler,
	admin.NewAdminAPIKeyHandler,
	admin.NewScheduledTestHandler,
	admin.NewChannelHandler,
	admin.NewContentModerationHandler,
	admin.NewComplianceHandler,
	admin.NewAuditLogHandler,

	// AdminHandlers and Handlers constructors
	ProvideAdminHandlers,
	ProvideHandlers,
)
