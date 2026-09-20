package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

// 工单 → Server酱³ 推送（fork 本地功能）。
//
// 新工单落库、用户追加回复后异步推一条消息给管理员，正文带工单页链接；推送失败只写日志，不影响主流程。
// 传输层复用分组运行状态推送的 serverChanClient（同一个 UID / SendKey），只多一个独立的开关
// ticket_notify_serverchan_enabled。客服自己的回复不推送。

const (
	ticketNotifyDeliveryBudget = 60 * time.Second
	ticketNotifyLogComponent   = "service.ticket_notify"
	ticketNotifyTimeLayout     = "2006-01-02 15:04:05 MST"
	ticketNotifyDefaultSite    = "Sub2API"
	ticketNotifyExcerptRunes   = 200

	ticketNotifyKindCreated     = "created"
	ticketNotifyKindUserReplied = "user_replied"
)

type ticketNotifyConfig struct {
	Enabled     bool
	UID         string
	SendKey     string
	SiteName    string
	FrontendURL string
}

// TicketNotifyService 工单事件的 Server酱³ 推送。
type TicketNotifyService struct {
	settingRepo SettingRepository
	client      *serverChanClient
}

// NewTicketNotifyService 构造推送服务。
func NewTicketNotifyService(settingRepo SettingRepository) *TicketNotifyService {
	return &TicketNotifyService{
		settingRepo: settingRepo,
		client:      newServerChanClient(),
	}
}

func (s *TicketNotifyService) loadConfig(ctx context.Context) (ticketNotifyConfig, error) {
	cfg := ticketNotifyConfig{SiteName: ticketNotifyDefaultSite}
	if s == nil || s.settingRepo == nil {
		return cfg, errors.New("setting repository is not configured")
	}
	values, err := s.settingRepo.GetMultiple(ctx, []string{
		SettingKeyTicketNotifyServerChanEnabled,
		SettingKeyGroupStatusNotifyServerChanUID,
		SettingKeyGroupStatusNotifyServerChanSendKey,
		SettingKeySiteName,
		SettingKeyFrontendURL,
	})
	if err != nil {
		return cfg, err
	}
	cfg.Enabled = values[SettingKeyTicketNotifyServerChanEnabled] == "true"
	cfg.UID = strings.TrimSpace(values[SettingKeyGroupStatusNotifyServerChanUID])
	cfg.SendKey = strings.TrimSpace(values[SettingKeyGroupStatusNotifyServerChanSendKey])
	cfg.FrontendURL = strings.TrimSpace(values[SettingKeyFrontendURL])
	if site := strings.TrimSpace(values[SettingKeySiteName]); site != "" {
		cfg.SiteName = site
	}
	return cfg, nil
}

// NotifyTicketCreated 实现 TicketNotifier：新工单，异步投递。
func (s *TicketNotifyService) NotifyTicketCreated(ticket *SupportTicket, first *SupportTicketMessage, fallbackOrigin string) {
	s.notifyAsync(ticketNotifyKindCreated, ticket, first, fallbackOrigin)
}

// NotifyTicketUserReplied 实现 TicketNotifier：用户追加回复，异步投递。
func (s *TicketNotifyService) NotifyTicketUserReplied(ticket *SupportTicket, msg *SupportTicketMessage, fallbackOrigin string) {
	s.notifyAsync(ticketNotifyKindUserReplied, ticket, msg, fallbackOrigin)
}

// notifyAsync 克隆入参后在 goroutine 里投递；nil 接收者安全。
func (s *TicketNotifyService) notifyAsync(kind string, ticket *SupportTicket, msg *SupportTicketMessage, fallbackOrigin string) {
	if s == nil || ticket == nil {
		return
	}
	ticketClone := *ticket
	var msgClone *SupportTicketMessage
	if msg != nil {
		m := *msg
		msgClone = &m
	}
	go func() {
		if err := s.deliver(kind, &ticketClone, msgClone, fallbackOrigin); err != nil {
			logger.LegacyPrintf(ticketNotifyLogComponent, "[TicketNotify] ticket=%d kind=%s push failed: %v", ticketClone.ID, kind, err)
		}
	}()
}

// deliver 同步投递一条推送；测试可直接调用。开关关闭 / 未配置时静默返回 nil。
func (s *TicketNotifyService) deliver(kind string, ticket *SupportTicket, msg *SupportTicketMessage, fallbackOrigin string) error {
	if s == nil || ticket == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), ticketNotifyDeliveryBudget)
	defer cancel()

	cfg, err := s.loadConfig(ctx)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	if !cfg.Enabled {
		return nil
	}
	if cfg.UID == "" || cfg.SendKey == "" {
		logger.LegacyPrintf(ticketNotifyLogComponent, "[TicketNotify] ticket=%d skipped: serverchan uid/sendkey not configured", ticket.ID)
		return nil
	}
	if err := ValidateServerChanUID(cfg.UID); err != nil {
		logger.LegacyPrintf(ticketNotifyLogComponent, "[TicketNotify] ticket=%d skipped: invalid serverchan uid", ticket.ID)
		return nil
	}

	link := ticketPageLink(cfg.FrontendURL, fallbackOrigin, ticket.ID)
	title, desp := buildTicketNotifyMessage(kind, cfg.SiteName, ticket, msg, link)
	if err := s.client.sendWithRetry(ctx, cfg.UID, cfg.SendKey, title, desp); err != nil {
		return redactServerChanSecret(err, cfg.SendKey)
	}
	logger.LegacyPrintf(ticketNotifyLogComponent, "[TicketNotify] ticket=%d kind=%s pushed via serverchan", ticket.ID, kind)
	return nil
}

// ticketPageLink 工单页深链：优先站点配置的 frontend_url，其次请求 origin；两者都没有则不带链接。
func ticketPageLink(frontendURL, fallbackOrigin string, id int64) string {
	base := strings.TrimRight(strings.TrimSpace(frontendURL), "/")
	if base == "" {
		base = strings.TrimRight(strings.TrimSpace(fallbackOrigin), "/")
	}
	if base == "" {
		return ""
	}
	return base + "/admin/tickets?id=" + strconv.FormatInt(id, 10)
}

// ticketCategoryLabels 分类 → 推送里的中文标签；未知分类回退为分类名本身。
var ticketCategoryLabels = map[string]string{
	TicketCategoryAccount: "账号",
	TicketCategoryBilling: "计费 / 额度",
	TicketCategoryAPI:     "API 调用",
	TicketCategoryOther:   "其他",
}

func ticketCategoryLabel(category string) string {
	category = strings.TrimSpace(category)
	if label, ok := ticketCategoryLabels[category]; ok {
		return label
	}
	if category == "" {
		return "-"
	}
	return category
}

// ticketNotifyExcerpt 把正文折成单行摘要并按 rune 截断。
func ticketNotifyExcerpt(body string, maxRunes int) string {
	text := strings.Join(strings.Fields(body), " ")
	if text == "" {
		return "-"
	}
	if maxRunes > 0 && utf8.RuneCountInString(text) > maxRunes {
		r := []rune(text)
		return string(r[:maxRunes]) + "…"
	}
	return text
}

// buildTicketNotifyMessage 生成推送标题与 Markdown 正文。
func buildTicketNotifyMessage(kind, siteName string, ticket *SupportTicket, msg *SupportTicketMessage, link string) (string, string) {
	siteName = strings.TrimSpace(siteName)
	if siteName == "" {
		siteName = ticketNotifyDefaultSite
	}
	var title string
	switch kind {
	case ticketNotifyKindUserReplied:
		title = fmt.Sprintf("[%s] 工单 #%d 有新回复：%s", siteName, ticket.ID, ticket.Title)
	default:
		title = fmt.Sprintf("[%s] 新工单 #%d：%s", siteName, ticket.ID, ticket.Title)
	}

	user := strings.TrimSpace(ticket.UserEmail)
	if user == "" {
		user = "user #" + strconv.FormatInt(ticket.UserID, 10)
	}
	body := ""
	at := ticket.LastMessageAt
	if msg != nil {
		body = msg.Body
		if !msg.CreatedAt.IsZero() {
			at = msg.CreatedAt
		}
	}
	if at.IsZero() {
		at = time.Now()
	}
	lines := []string{
		fmt.Sprintf("**用户**：%s", user),
		fmt.Sprintf("**分类**：%s", ticketCategoryLabel(ticket.Category)),
		fmt.Sprintf("**内容**：%s", ticketNotifyExcerpt(body, ticketNotifyExcerptRunes)),
		fmt.Sprintf("**时间**：%s", at.Local().Format(ticketNotifyTimeLayout)),
	}
	if link != "" {
		lines = append(lines, fmt.Sprintf("[查看工单](%s)", link))
	}
	return title, strings.Join(lines, "\n\n")
}
