// Package bilingual 把网关返回给 API 客户端的报错改写成「中文 / English」单串（fork 本地功能，
// 见 CLAUDE.md「网关错误中英双语」）。
//
// 网关客户端（SDK、CLI、各类工具）没有语言协商通道，所以沿用 cyberSessionBlockedClientMsg 的写法：
// 一条消息同时给出中文和英文。只翻译账户 / 风控 / 计费这类需要用户自己处理的报错；
// 不认识的消息（包括管理员自定义的拦截文案、上游透传的报错）原样返回。
//
// 约束：原有文本必须逐字保留在它所属语言那一半的开头。运维错误分类
// （handler/ops_error_logger.go）和桌面客户端（client/crates/sub2api/src/paywall.rs）
// 都按小写子串匹配这些文本，bilingual_test.go 钉住了这一点。
package bilingual

import (
	"strings"
	"unicode/utf8"
)

// Separator 中英两半之间的分隔符。
const Separator = " / "

// Text 一条双语消息。Aliases 是额外的查找键：原有文本在翻译时被扩写过（例如补了一句引导），
// 原文仍然要能查到这一条。
type Text struct {
	ZH      string
	EN      string
	Aliases []string
}

// String 渲染为「中文 / English」。
func (t Text) String() string {
	return t.ZH + Separator + t.EN
}

// prefixRule 处理带动态尾巴的消息（如 IP）：英文前缀 + 动态部分 → 中文前缀 + 动态部分 / 英文前缀 + 动态部分。
type prefixRule struct {
	zhPrefix string
	enPrefix string
}

var prefixRules = []prefixRule{
	{zhPrefix: "访问被拒绝，当前 IP：", enPrefix: "Access denied. Your IP is "},
}

// hashSuffixMarker 内容审计 hash 命中时追加在拦截文案后面的尾巴：fmt.Sprintf("%s（hash: %s）", message, hash)。
const hashSuffixMarker = "（hash: "

// index 小写键 → 渲染结果。中文、英文、渲染后的整串都能作为键，所以 Gateway 是幂等的。
var index = buildIndex(gatewayMessages)

func buildIndex(messages []Text) map[string]string {
	out := make(map[string]string, len(messages)*3)
	for _, m := range messages {
		rendered := m.String()
		keys := append([]string{m.ZH, m.EN, rendered}, m.Aliases...)
		for _, key := range keys {
			out[normalizeKey(key)] = rendered
		}
	}
	return out
}

func normalizeKey(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

func lookup(msg string) (string, bool) {
	out, ok := index[normalizeKey(msg)]
	return out, ok
}

// Gateway 把一条网关报错改写成双语；不认识的消息原样返回。
func Gateway(msg string) string {
	trimmed := strings.TrimSpace(msg)
	if trimmed == "" {
		return msg
	}
	if out, ok := lookup(trimmed); ok {
		return out
	}
	// 内容审计 hash 拦截：前半翻译，hash 尾巴原样接回去
	if i := strings.Index(trimmed, hashSuffixMarker); i > 0 {
		if out, ok := lookup(trimmed[:i]); ok {
			return out + trimmed[i:]
		}
	}
	for _, rule := range prefixRules {
		if strings.HasPrefix(trimmed, rule.zhPrefix) {
			return msg // 已经是双语
		}
		if strings.HasPrefix(trimmed, rule.enPrefix) {
			tail := trimmed[len(rule.enPrefix):]
			return rule.zhPrefix + tail + Separator + trimmed
		}
	}
	return msg
}

// TruncateUTF8 把 s 截到不超过 max 字节，且不会切断一个 UTF-8 字符。
func TruncateUTF8(s string, max int) string {
	if max <= 0 {
		return ""
	}
	if len(s) <= max {
		return s
	}
	cut := max
	for cut > 0 && !utf8.RuneStart(s[cut]) {
		cut--
	}
	return s[:cut]
}
