package service

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"
)

// 内容审计 → 站内信（fork 本地功能，见 CLAUDE.md「站内信」）。
//
// 上游 content_moderation.go 里只有三处 fork 改动：服务结构体的 siteMessageNotifier 字段、
// persistContentModerationLog 里的 deliverFlaggedSiteMessages 调用、RecordCyberPolicyEvent 末尾的
// deliverCyberPolicySiteMessages 调用（以及配置里紧挨 email_on_hit 的 site_message_on_hit 开关）。
//
// 规则：受 cfg.SiteMessageOnHit 控制（默认开），与邮件开关互相独立，也不依赖 SMTP / 用户邮箱：
//   - 违规提醒：每次命中（applySideEffects 时）都发；
//   - 封禁通知：本次刚触发自动封禁时发；
//   - cyber 通知：非 LogOnly 的 cyber 拦截发。
// 「仅记录」模式与白名单不发（上游已把 applySideEffects / LogOnly 置好）。
// 正文由服务端渲染为中英双语 Markdown：收件人语言偏好目前没有任何地方写入，无法按语言渲染。

// ContentModerationSiteMessageNotifier 站内信投递端口（实现是 *SiteMessageService）。
type ContentModerationSiteMessageNotifier interface {
	DeliverSystemMessage(ctx context.Context, in SystemSiteMessageInput) error
}

// SetSiteMessageNotifier 挂上站内信投递（由 wire_local.go 的 ProvideContentModerationService 在启动时注入）。
func (s *ContentModerationService) SetSiteMessageNotifier(n ContentModerationSiteMessageNotifier) {
	if s == nil {
		return
	}
	s.siteMessageNotifier = n
}

const (
	siteMessageFieldMaxRunes       = 120
	siteMessageCyberDetailMaxRunes = 500
	siteMessageTimeLayout          = "2006-01-02 15:04:05 UTC"
	siteMessageBilingualDivider    = "\n\n---\n\n"
	siteMessageAppealHintZH        = "如需申诉，请使用该账号登录网站，按页面提示提交申诉工单；处理结果会通过工单回复和站内信通知您。"
	siteMessageAppealHintEN        = "To appeal, sign in to the website with this account and submit an appeal ticket as prompted. You will be notified of the result in the ticket and by site message."
	siteMessageBannedBannerZH      = "**账户当前处于封禁状态，所有 API 请求将被拒绝。**"
	siteMessageBannedBannerEN      = "**This account is currently disabled. All API requests will be rejected.**"
)

// deliverFlaggedSiteMessages 内容审核命中后的站内信（违规提醒 + 刚封禁时的封禁通知）。
func (s *ContentModerationService) deliverFlaggedSiteMessages(ctx context.Context, cfg *ContentModerationConfig, log *ContentModerationLog, autoBanJustApplied bool) {
	if !s.siteMessageDeliverable(cfg, log) {
		return
	}
	title, content := buildContentModerationViolationSiteMessage(log, cfg)
	s.deliverContentModerationSiteMessage(ctx, log, SiteMessageSourceContentModeration, title, content)
	if autoBanJustApplied {
		title, content = buildContentModerationDisabledSiteMessage(log, cfg)
		s.deliverContentModerationSiteMessage(ctx, log, SiteMessageSourceContentModerationBan, title, content)
	}
}

// deliverCyberPolicySiteMessages cyber 拦截后的站内信（LogOnly 不发）。
func (s *ContentModerationService) deliverCyberPolicySiteMessages(ctx context.Context, cfg *ContentModerationConfig, log *ContentModerationLog, logOnly bool, autoBanned bool) {
	if logOnly || !s.siteMessageDeliverable(cfg, log) {
		return
	}
	title, content := buildCyberPolicySiteMessage(log)
	s.deliverContentModerationSiteMessage(ctx, log, SiteMessageSourceCyberPolicy, title, content)
	if autoBanned {
		title, content = buildContentModerationDisabledSiteMessage(log, cfg)
		s.deliverContentModerationSiteMessage(ctx, log, SiteMessageSourceCyberPolicyBan, title, content)
	}
}

func (s *ContentModerationService) siteMessageDeliverable(cfg *ContentModerationConfig, log *ContentModerationLog) bool {
	if s == nil || s.siteMessageNotifier == nil || cfg == nil || log == nil {
		return false
	}
	if !cfg.SiteMessageOnHit || !log.Flagged {
		return false
	}
	return log.UserID != nil && *log.UserID > 0
}

func (s *ContentModerationService) deliverContentModerationSiteMessage(ctx context.Context, log *ContentModerationLog, source, title, content string) {
	err := s.siteMessageNotifier.DeliverSystemMessage(ctx, SystemSiteMessageInput{
		UserID:     contentModerationEmailUserID(log),
		Category:   SiteMessageCategorySecurity,
		Title:      title,
		Content:    content,
		SourceType: source,
		SourceID:   contentModerationSiteMessageSourceID(log),
	})
	if err != nil {
		slog.Warn("content_moderation.site_message_failed",
			"user_id", contentModerationEmailUserID(log), "source", source, "error", err)
	}
}

// contentModerationSiteMessageSourceID 便于追溯：已落库用日志 id，否则用请求 id（仅追溯，不做去重）。
func contentModerationSiteMessageSourceID(log *ContentModerationLog) string {
	if log == nil {
		return ""
	}
	if log.ID > 0 {
		return fmt.Sprintf("log:%d", log.ID)
	}
	if rid := strings.TrimSpace(log.RequestID); rid != "" {
		return clampRunes("req:"+rid, siteMessageSourceIDMaxRunes)
	}
	return ""
}

// ---------- 渲染（纯函数） ----------

func buildContentModerationViolationSiteMessage(log *ContentModerationLog, cfg *ContentModerationConfig) (string, string) {
	title := "账户风控提醒 / Risk control notice"
	f := siteMessageModerationFields(log, cfg)

	var zh, en strings.Builder
	zh.WriteString("您的 API 请求在内容审计中触发了平台风控策略。\n\n")
	zh.WriteString("| 项目 | 详情 |\n| --- | --- |\n")
	fmt.Fprintf(&zh, "| 触发时间 | %s |\n", f.at)
	fmt.Fprintf(&zh, "| 所属分组 | %s |\n", f.group)
	fmt.Fprintf(&zh, "| 命中类别 | %s / %s |\n", f.category, f.score)
	fmt.Fprintf(&zh, "| 累计触发次数 | %s |\n", f.countZH)
	if f.requestID != "" {
		fmt.Fprintf(&zh, "| 请求 ID | %s |\n", f.requestID)
	}

	en.WriteString("Your API request triggered the platform's risk-control policy during content audit.\n\n")
	en.WriteString("| Item | Details |\n| --- | --- |\n")
	fmt.Fprintf(&en, "| Triggered at | %s |\n", f.at)
	fmt.Fprintf(&en, "| Group | %s |\n", f.group)
	fmt.Fprintf(&en, "| Category | %s / %s |\n", f.category, f.score)
	fmt.Fprintf(&en, "| Violations | %s |\n", f.countEN)
	if f.requestID != "" {
		fmt.Fprintf(&en, "| Request ID | %s |\n", f.requestID)
	}

	switch {
	case log != nil && log.AutoBanned:
		zh.WriteString("\n" + siteMessageBannedBannerZH + "\n\n" + siteMessageAppealHintZH)
		en.WriteString("\n" + siteMessageBannedBannerEN + "\n\n" + siteMessageAppealHintEN)
	case cfg != nil && cfg.AutoBanEnabled:
		zh.WriteString("\n多次触发将导致账户被自动禁用，请调整请求内容。")
		en.WriteString("\nRepeated violations will disable the account automatically. Please adjust your requests.")
	default:
		zh.WriteString("\n请调整请求内容。")
		en.WriteString("\nPlease adjust your requests.")
	}
	return title, strings.TrimSpace(zh.String()) + siteMessageBilingualDivider + strings.TrimSpace(en.String())
}

func buildContentModerationDisabledSiteMessage(log *ContentModerationLog, cfg *ContentModerationConfig) (string, string) {
	title := "账户已被禁用 / Account disabled"
	f := siteMessageModerationFields(log, cfg)

	var zh, en strings.Builder
	zh.WriteString("您的账户在计数周期内多次触发平台风控策略，系统已自动禁用该账户。\n\n")
	zh.WriteString("| 项目 | 详情 |\n| --- | --- |\n")
	fmt.Fprintf(&zh, "| 封禁时间 | %s |\n", f.at)
	fmt.Fprintf(&zh, "| 所属分组 | %s |\n", f.group)
	fmt.Fprintf(&zh, "| 命中类别 | %s / %s |\n", f.category, f.score)
	fmt.Fprintf(&zh, "| 累计触发次数 | %s |\n", f.countZH)
	zh.WriteString("\n" + siteMessageBannedBannerZH + "\n\n" + siteMessageAppealHintZH)

	en.WriteString("Your account triggered the platform's risk-control policy repeatedly within the counting window and has been disabled automatically.\n\n")
	en.WriteString("| Item | Details |\n| --- | --- |\n")
	fmt.Fprintf(&en, "| Disabled at | %s |\n", f.at)
	fmt.Fprintf(&en, "| Group | %s |\n", f.group)
	fmt.Fprintf(&en, "| Category | %s / %s |\n", f.category, f.score)
	fmt.Fprintf(&en, "| Violations | %s |\n", f.countEN)
	en.WriteString("\n" + siteMessageBannedBannerEN + "\n\n" + siteMessageAppealHintEN)

	return title, strings.TrimSpace(zh.String()) + siteMessageBilingualDivider + strings.TrimSpace(en.String())
}

func buildCyberPolicySiteMessage(log *ContentModerationLog) (string, string) {
	title := "网络安全策略拦截提醒 / Cyber-security policy notice"
	at := siteMessageEventTime(log)
	model, group, requestID, detail := "-", "-", "", ""
	if log != nil {
		model = siteMessageField(log.Model)
		group = siteMessageField(log.GroupName)
		if rid := strings.TrimSpace(log.RequestID); rid != "" {
			requestID = siteMessageMarkdownEscape(clampRunes(rid, siteMessageFieldMaxRunes))
		}
		detail = siteMessageFirstLine(log.Error, siteMessageCyberDetailMaxRunes)
	}

	var zh, en strings.Builder
	zh.WriteString("您的请求被网络安全策略（cyber policy）拦截。\n\n")
	zh.WriteString("| 项目 | 详情 |\n| --- | --- |\n")
	fmt.Fprintf(&zh, "| 触发时间 | %s |\n", at)
	fmt.Fprintf(&zh, "| 模型 | %s |\n", model)
	fmt.Fprintf(&zh, "| 所属分组 | %s |\n", group)
	if requestID != "" {
		fmt.Fprintf(&zh, "| 请求 ID | %s |\n", requestID)
	}
	if detail != "" {
		zh.WriteString("\n拦截说明：\n\n" + siteMessageCodeBlock(detail) + "\n")
	}
	zh.WriteString("\n请勿发送涉及网络攻击、恶意代码等违反安全策略的内容，多次触发可能导致账户被禁用。")

	en.WriteString("Your request was blocked by the cyber-security policy.\n\n")
	en.WriteString("| Item | Details |\n| --- | --- |\n")
	fmt.Fprintf(&en, "| Triggered at | %s |\n", at)
	fmt.Fprintf(&en, "| Model | %s |\n", model)
	fmt.Fprintf(&en, "| Group | %s |\n", group)
	if requestID != "" {
		fmt.Fprintf(&en, "| Request ID | %s |\n", requestID)
	}
	if detail != "" {
		en.WriteString("\nDetails:\n\n" + siteMessageCodeBlock(detail) + "\n")
	}
	en.WriteString("\nDo not send content that violates the security policy, such as cyber attacks or malicious code. Repeated violations may disable the account.")

	return title, strings.TrimSpace(zh.String()) + siteMessageBilingualDivider + strings.TrimSpace(en.String())
}

type siteMessageModerationValues struct {
	at, group, category, score, countZH, countEN, requestID string
}

func siteMessageModerationFields(log *ContentModerationLog, cfg *ContentModerationConfig) siteMessageModerationValues {
	v := siteMessageModerationValues{
		at:       siteMessageEventTime(log),
		group:    "-",
		category: "-",
		score:    "0.000",
		countZH:  "0 次",
		countEN:  "0",
	}
	if log == nil {
		return v
	}
	v.group = siteMessageField(log.GroupName)
	v.category = siteMessageField(log.HighestCategory)
	v.score = fmt.Sprintf("%.3f", log.HighestScore)
	if rid := strings.TrimSpace(log.RequestID); rid != "" {
		v.requestID = siteMessageMarkdownEscape(clampRunes(rid, siteMessageFieldMaxRunes))
	}
	count := log.ViolationCount
	if cfg != nil && cfg.AutoBanEnabled {
		threshold := cfg.BanThreshold
		if threshold <= 0 {
			threshold = defaultContentModerationBanThreshold
		}
		v.countZH = fmt.Sprintf("%d 次（阈值 %d）", count, threshold)
		v.countEN = fmt.Sprintf("%d (threshold %d)", count, threshold)
	} else {
		v.countZH = fmt.Sprintf("%d 次", count)
		v.countEN = fmt.Sprintf("%d", count)
	}
	return v
}

func siteMessageEventTime(log *ContentModerationLog) string {
	at := time.Now()
	if log != nil && !log.CreatedAt.IsZero() {
		at = log.CreatedAt
	}
	return at.UTC().Format(siteMessageTimeLayout)
}

// siteMessageField 表格单元格：单行、截断、转义；空值显示 "-"。
func siteMessageField(v string) string {
	v = strings.Join(strings.Fields(v), " ")
	if v == "" {
		return "-"
	}
	return siteMessageMarkdownEscape(clampRunes(v, siteMessageFieldMaxRunes))
}

// siteMessageMarkdownEscape 转义 Markdown 元字符。模型名、分组名、请求 ID 有一部分来自请求方，
// 不转义就能在站内信里注入链接、图片或打乱表格。
func siteMessageMarkdownEscape(v string) string {
	var b strings.Builder
	b.Grow(len(v))
	for _, r := range v {
		switch r {
		case '\\', '`', '*', '_', '{', '}', '[', ']', '(', ')', '#', '+', '-', '.', '!', '|', '<', '>', '~':
			b.WriteByte('\\')
		}
		b.WriteRune(r)
	}
	return b.String()
}

// siteMessageFirstLine 取第一行非空文本并截断（cyber 拦截的 log.Error 第一行是上游提示，后面是原始 body / 用量）。
func siteMessageFirstLine(v string, max int) string {
	for _, line := range strings.Split(strings.ReplaceAll(v, "\r\n", "\n"), "\n") {
		if line = strings.TrimSpace(line); line != "" {
			return clampRunes(line, max)
		}
	}
	return ""
}

// siteMessageCodeBlock 用比内容里最长的反引号串更长的围栏包起来，内容无法提前结束代码块。
func siteMessageCodeBlock(v string) string {
	longest, run := 0, 0
	for _, r := range v {
		if r == '`' {
			run++
			if run > longest {
				longest = run
			}
		} else {
			run = 0
		}
	}
	fenceLen := longest + 1
	if fenceLen < 3 {
		fenceLen = 3
	}
	fence := strings.Repeat("`", fenceLen)
	return fence + "text\n" + v + "\n" + fence
}
