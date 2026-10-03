package handler

import (
	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

// ProvideAuthHandler 构造认证 handler 并挂上封禁申诉服务（fork 本地）。
// 用 setter 而非改上游构造函数签名，保持现有测试的 NewAuthHandler 用法不变。
func ProvideAuthHandler(
	cfg *config.Config,
	authService *service.AuthService,
	userService *service.UserService,
	settingService *service.SettingService,
	promoService *service.PromoService,
	redeemService *service.RedeemService,
	totpService *service.TotpService,
	userAttributeService *service.UserAttributeService,
	appealService *service.AppealService,
) *AuthHandler {
	h := NewAuthHandler(cfg, authService, userService, settingService, promoService, redeemService, totpService, userAttributeService)
	if appealService != nil {
		h.SetAppealService(appealService)
	}
	return h
}

// ProvidePasskeyHandler 构造 Passkey handler 并挂上封禁申诉服务（fork 本地）。
func ProvidePasskeyHandler(
	passkeys *service.PasskeyService,
	authService *service.AuthService,
	settingService *service.SettingService,
	appealService *service.AppealService,
) *PasskeyHandler {
	h := NewPasskeyHandler(passkeys, authService, settingService)
	if appealService != nil {
		h.SetAppealService(appealService)
	}
	return h
}
