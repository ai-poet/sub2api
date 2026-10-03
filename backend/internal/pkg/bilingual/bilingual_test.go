package bilingual

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/require"
)

func TestGatewayMessagesHaveUniqueKeys(t *testing.T) {
	seen := map[string]string{}
	for _, m := range gatewayMessages {
		require.NotEmpty(t, strings.TrimSpace(m.ZH), "empty zh for %q", m.EN)
		require.NotEmpty(t, strings.TrimSpace(m.EN), "empty en for %q", m.ZH)
		require.NotContains(t, strings.ToLower(m.EN), "upstream", "paywall.rs skips messages mentioning upstream")
		for _, key := range append([]string{m.ZH, m.EN}, m.Aliases...) {
			k := normalizeKey(key)
			prev, dup := seen[k]
			require.False(t, dup, "key %q used by both %q and %q", key, prev, m.String())
			seen[k] = m.String()
		}
	}
}

func TestGatewayRendersBothHalvesAndIsIdempotent(t *testing.T) {
	for _, m := range gatewayMessages {
		want := m.ZH + " / " + m.EN
		require.Equal(t, want, Gateway(m.ZH))
		require.Equal(t, want, Gateway(m.EN))
		require.Equal(t, want, Gateway(strings.ToUpper(m.EN)), "matching is case-insensitive")
		require.Equal(t, want, Gateway("  "+m.EN+"\n"))
		require.Equal(t, want, Gateway(want), "rendering twice must not change the message")
		for _, alias := range m.Aliases {
			require.Equal(t, want, Gateway(alias))
		}
	}
}

func TestGatewayKnownMessages(t *testing.T) {
	require.Equal(t,
		"账户已被停用，请登录网站查看通知或提交申诉 / User account is not active. Sign in to the website to view notices or submit an appeal.",
		Gateway("User account is not active"))
	require.Equal(t, "API key 已过期 / API key has expired", Gateway("api key 已过期"))
	require.Equal(t, "账户余额不足 / Insufficient account balance", Gateway("Insufficient account balance"))
	require.Equal(t, "已超出每日用量上限 / daily usage limit exceeded", Gateway("daily usage limit exceeded"))
}

func TestGatewayPassesUnknownMessagesThrough(t *testing.T) {
	for _, msg := range []string{
		"",
		"   ",
		"自定义拦截：请勿发送敏感内容",
		"Model \"gpt-x\" is not available for this group",
		"upstream returned 500",
	} {
		require.Equal(t, msg, Gateway(msg))
	}
}

func TestGatewayIPPrefixRule(t *testing.T) {
	got := Gateway("Access denied. Your IP is 203.0.113.9")
	require.Equal(t, "访问被拒绝，当前 IP：203.0.113.9 / Access denied. Your IP is 203.0.113.9", got)
	require.Equal(t, got, Gateway(got))
}

func TestGatewayHashSuffix(t *testing.T) {
	got := Gateway("内容审计命中风险规则，请调整输入后重试（hash: abc123）")
	require.Equal(t, "内容审计命中风险规则，请调整输入后重试 / Content audit matched a risk rule. Please adjust your input and try again.（hash: abc123）", got)
	require.Equal(t, got, Gateway(got))

	custom := "自定义文案（hash: abc123）"
	require.Equal(t, custom, Gateway(custom))
}

// 运维错误分类与桌面客户端按小写子串识别这些报错，双语化之后必须仍然命中。
func TestGatewayKeepsClassifierSubstrings(t *testing.T) {
	cases := map[string]string{
		"Invalid API key":                        "invalid api key",
		"API key is required":                    "api key is required",
		"API key is disabled":                    "api key is disabled",
		"User associated with API key not found": "user associated with api key not found",
		"User account is not active":             "user account is not active",
		"API Key 所属分组已删除":                        "api key 所属分组已删除",
		"API Key 所属分组已停用":                        "api key 所属分组已停用",
		"API Key is not assigned to any group and cannot be used. Please contact the administrator to assign it to a group.": "api key is not assigned to any group",
		"API key in query parameter is deprecated. Please use Authorization header instead.":                                 "api key in query parameter is deprecated",
		"Query parameter api_key is deprecated. Use Authorization header or key instead.":                                    "query parameter api_key is deprecated",
		"No active subscription found for this group":                                                                        "no active subscription found for this group",
		"subscription is invalid or expired":                                                                                 "subscription is invalid or expired",
		"insufficient balance":                                                                                               "insufficient balance",
		"Insufficient account balance":                                                                                       "insufficient account balance",
		"API key 额度已用完":                                                                                                      "api key 额度已用完",
		"api key 5小时限额已用完":                                                                                                   "api key 5小时限额已用完",
		"api key 日限额已用完":                                                                                                     "api key 日限额已用完",
		"api key 7天限额已用完":                                                                                                    "api key 7天限额已用完",
		"daily usage limit exceeded":                                                                                         "daily usage limit exceeded",
		"weekly usage limit exceeded":                                                                                        "weekly usage limit exceeded",
		"monthly usage limit exceeded":                                                                                       "monthly usage limit exceeded",
		"Daily usage quota exhausted for this platform.":                                                                     "usage quota exhausted for this platform",
		"group requests-per-minute limit exceeded":                                                                           "requests-per-minute limit exceeded",
		"user requests-per-minute limit exceeded":                                                                            "requests-per-minute limit exceeded",
	}
	for legacy, substr := range cases {
		require.Contains(t, strings.ToLower(Gateway(legacy)), substr, "legacy %q", legacy)
	}
}

func TestTruncateUTF8(t *testing.T) {
	require.Equal(t, "", TruncateUTF8("abc", 0))
	require.Equal(t, "abc", TruncateUTF8("abc", 10))
	require.Equal(t, "ab", TruncateUTF8("abc", 2))

	zh := "账户已被停用"
	for max := 0; max <= len(zh); max++ {
		got := TruncateUTF8(zh, max)
		require.True(t, utf8.ValidString(got), "max=%d", max)
		require.LessOrEqual(t, len(got), max)
		require.True(t, strings.HasPrefix(zh, got))
	}

	// 现有 cyber 会话屏蔽文案在第 120 字节处落在 ASCII 上，截断结果与按字节切一致。
	cyber := "该会话已被网络安全策略屏蔽，请开启新会话 / This session is blocked by cyber-security policy, please start a new session"
	require.Equal(t, cyber[:120], TruncateUTF8(cyber, 120))

	long := Gateway("User account is not active")
	require.True(t, utf8.ValidString(TruncateUTF8(long, 120)))
}
