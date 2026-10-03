package handler

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/securityaudit"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/stretchr/testify/require"
)

// 网关错误中英双语（fork 本地）：计费 / 内容审计报错返回给客户端时是「中文 / English」。

func TestBillingErrorDetails_MessagesAreBilingual(t *testing.T) {
	cases := map[error]string{
		service.ErrInsufficientBalance:               "余额不足 / insufficient balance",
		service.ErrGroupRPMExceeded:                  "分组每分钟请求数已达上限 / group requests-per-minute limit exceeded",
		service.ErrAPIKeyRateLimit1dExceeded:         "api key 日限额已用完 / API key daily limit exhausted",
		service.ErrBillingServiceUnavailable:         "计费服务暂时不可用，请稍后重试 / Billing service temporarily unavailable. Please retry later.",
		service.ErrUserPlatformDailyQuotaExhausted:   "该平台今日用量额度已用完 / Daily usage quota exhausted for this platform.",
		service.ErrSubscriptionInvalid:               "订阅无效或已过期 / subscription is invalid or expired",
		service.ErrUserPlatformMonthlyQuotaExhausted: "该平台本月用量额度已用完 / Monthly usage quota exhausted for this platform.",
	}
	for err, want := range cases {
		_, _, msg, _ := billingErrorDetails(err)
		require.Equal(t, want, msg, "err %v", err)
	}
}

func TestSecurityAuditMessage_DefaultsAreBilingual(t *testing.T) {
	zhDefault := &securityaudit.Decision{ClientMessage: "内容审计命中风险规则，请调整输入后重试"}
	enDefault := &securityaudit.Decision{ClientMessage: "Content audit matched a risk rule. Please adjust your input and try again."}
	want := "内容审计命中风险规则，请调整输入后重试 / Content audit matched a risk rule. Please adjust your input and try again."
	require.Equal(t, want, securityAuditMessage(zhDefault))
	require.Equal(t, want, securityAuditMessage(enDefault))

	hashed := &securityaudit.Decision{ClientMessage: "内容审计命中风险规则，请调整输入后重试（hash: deadbeef）"}
	require.Equal(t, want+"（hash: deadbeef）", securityAuditMessage(hashed))

	require.Equal(t, "请求被内容安全策略拦截 / Request blocked by content policy", securityAuditMessage(nil))

	prompt := &securityaudit.Decision{ClientMessage: "提示词安全审计拒绝了该请求，请调整输入后重试"}
	require.Equal(t, "提示词安全审计拒绝了该请求，请调整输入后重试 / Prompt security audit rejected this request. Please adjust your input and try again.", securityAuditMessage(prompt))
}

func TestSecurityAuditMessage_CustomMessagePassesThrough(t *testing.T) {
	custom := &securityaudit.Decision{ClientMessage: "本站禁止此类内容 / This content is not allowed here"}
	require.Equal(t, "本站禁止此类内容 / This content is not allowed here", securityAuditMessage(custom))
}
