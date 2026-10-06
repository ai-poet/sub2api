package admin

import "github.com/Wei-Shaw/sub2api/internal/service"

// ProvideUsageHandler 构造使用记录 handler 并挂上扣费失败状态标注（fork 本地）。
// 构造函数签名保持上游原样，测试与 wire 的其它调用点不受影响。
func ProvideUsageHandler(
	usageService *service.UsageService,
	apiKeyService *service.APIKeyService,
	adminService service.AdminService,
	cleanupService *service.UsageCleanupService,
	billingRetry *service.UsageBillingRetryService,
) *UsageHandler {
	h := NewUsageHandler(usageService, apiKeyService, adminService, cleanupService)
	// 显式判空，避免 nil 指针包进非 nil 接口
	if billingRetry != nil {
		h.SetBillingRetryAnnotator(billingRetry)
	}
	return h
}
