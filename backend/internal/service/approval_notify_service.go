package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

// 运维写操作审批 → Server酱³ 推送（fork 本地功能）。
//
// 新申请落库后异步推一条消息给管理员，正文带审批页链接；推送失败只写日志，不影响入队。
// 传输层复用分组运行状态推送的 serverChanClient（同一个 UID / SendKey），只多一个独立的开关
// approval_notify_serverchan_enabled。

const (
	approvalNotifyDeliveryBudget = 60 * time.Second
	approvalNotifyLogComponent   = "service.approval_notify"
	approvalNotifyTimeLayout     = "2006-01-02 15:04:05 MST"
	approvalNotifyDefaultSite    = "Sub2API"
)

type approvalNotifyConfig struct {
	Enabled     bool
	UID         string
	SendKey     string
	SiteName    string
	FrontendURL string
}

// ApprovalNotifyService 新审批申请的 Server酱³ 推送。
type ApprovalNotifyService struct {
	settingRepo SettingRepository
	repo        AdminApprovalRepository
	client      *serverChanClient
}

// NewApprovalNotifyService 构造推送服务；repo 用于回写 notified_at，允许为 nil。
func NewApprovalNotifyService(settingRepo SettingRepository, repo AdminApprovalRepository) *ApprovalNotifyService {
	return &ApprovalNotifyService{
		settingRepo: settingRepo,
		repo:        repo,
		client:      newServerChanClient(),
	}
}

func (s *ApprovalNotifyService) loadConfig(ctx context.Context) (approvalNotifyConfig, error) {
	cfg := approvalNotifyConfig{SiteName: approvalNotifyDefaultSite}
	if s == nil || s.settingRepo == nil {
		return cfg, errors.New("setting repository is not configured")
	}
	values, err := s.settingRepo.GetMultiple(ctx, []string{
		SettingKeyApprovalNotifyServerChanEnabled,
		SettingKeyGroupStatusNotifyServerChanUID,
		SettingKeyGroupStatusNotifyServerChanSendKey,
		SettingKeySiteName,
		SettingKeyFrontendURL,
	})
	if err != nil {
		return cfg, err
	}
	cfg.Enabled = values[SettingKeyApprovalNotifyServerChanEnabled] == "true"
	cfg.UID = strings.TrimSpace(values[SettingKeyGroupStatusNotifyServerChanUID])
	cfg.SendKey = strings.TrimSpace(values[SettingKeyGroupStatusNotifyServerChanSendKey])
	cfg.FrontendURL = strings.TrimSpace(values[SettingKeyFrontendURL])
	if site := strings.TrimSpace(values[SettingKeySiteName]); site != "" {
		cfg.SiteName = site
	}
	return cfg, nil
}

// NotifyRequested 实现 ApprovalNotifier：异步投递，nil 接收者安全。
func (s *ApprovalNotifyService) NotifyRequested(req *AdminApprovalRequest, fallbackOrigin string) {
	if s == nil || req == nil {
		return
	}
	clone := *req
	go func() {
		if err := s.deliver(&clone, fallbackOrigin); err != nil {
			logger.LegacyPrintf(approvalNotifyLogComponent, "[ApprovalNotify] approval=%d push failed: %v", clone.ID, err)
		}
	}()
}

// deliver 同步投递一条推送；测试可直接调用。开关关闭 / 未配置时静默返回 nil。
func (s *ApprovalNotifyService) deliver(req *AdminApprovalRequest, fallbackOrigin string) error {
	if s == nil || req == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), approvalNotifyDeliveryBudget)
	defer cancel()

	cfg, err := s.loadConfig(ctx)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	if !cfg.Enabled {
		return nil
	}
	if cfg.UID == "" || cfg.SendKey == "" {
		logger.LegacyPrintf(approvalNotifyLogComponent, "[ApprovalNotify] approval=%d skipped: serverchan uid/sendkey not configured", req.ID)
		return nil
	}
	if err := ValidateServerChanUID(cfg.UID); err != nil {
		logger.LegacyPrintf(approvalNotifyLogComponent, "[ApprovalNotify] approval=%d skipped: invalid serverchan uid", req.ID)
		return nil
	}

	link := approvalPageLink(cfg.FrontendURL, fallbackOrigin, req.ID)
	title, desp := buildApprovalNotifyMessage(cfg.SiteName, req, link)
	if err := s.client.sendWithRetry(ctx, cfg.UID, cfg.SendKey, title, desp); err != nil {
		return redactServerChanSecret(err, cfg.SendKey)
	}
	if s.repo != nil {
		if err := s.repo.MarkNotified(ctx, req.ID, time.Now()); err != nil {
			logger.LegacyPrintf(approvalNotifyLogComponent, "[ApprovalNotify] approval=%d mark notified failed: %v", req.ID, err)
		}
	}
	logger.LegacyPrintf(approvalNotifyLogComponent, "[ApprovalNotify] approval=%d pushed via serverchan", req.ID)
	return nil
}

// approvalPageLink 审批页深链：优先站点配置的 frontend_url，其次捕获时的请求 origin；两者都没有则不带链接。
func approvalPageLink(frontendURL, fallbackOrigin string, id int64) string {
	base := strings.TrimRight(strings.TrimSpace(frontendURL), "/")
	if base == "" {
		base = strings.TrimRight(strings.TrimSpace(fallbackOrigin), "/")
	}
	if base == "" {
		return ""
	}
	return base + "/admin/approvals?id=" + strconv.FormatInt(id, 10)
}

// approvalActionLabels 审计动作名 → 推送里的中文标签；未知动作回退为动作名本身。
var approvalActionLabels = map[string]string{
	"admin.users.create":                       "新建用户",
	"admin.users.update":                       "编辑用户",
	"admin.users.balance.create":               "调整余额",
	"admin.users.replace_group.create":         "更换专属分组",
	"admin.users.batch_concurrency.create":     "批量调整并发",
	"admin.users.batch_limits.create":          "批量调整限额",
	"admin.users.platform_quotas.update":       "修改平台额度",
	"admin.users.platform_quotas.reset.create": "重置平台额度窗口",
	"admin.users.attributes.update":            "修改用户属性",
	"admin.users.auth_identities.create":       "绑定登录身份",
	"admin.api_keys.update":                    "调整 API Key 分组",
	"admin.subscriptions.assign.create":        "分配订阅",
	"admin.subscriptions.bulk_assign.create":   "批量分配订阅",
	"admin.subscriptions.extend.create":        "调整订阅有效期",
	"admin.subscriptions.reset_quota.create":   "重置订阅额度",
	"admin.subscriptions.revoke.create":        "撤销订阅",
	"admin.subscriptions.restore.create":       "恢复订阅",
	"admin.subscriptions.delete":               "撤销订阅",
}

func approvalActionLabel(action string) string {
	if label, ok := approvalActionLabels[strings.TrimSpace(action)]; ok {
		return label
	}
	if strings.TrimSpace(action) == "" {
		return "写操作"
	}
	return action
}

// buildApprovalNotifyMessage 生成推送标题与 Markdown 正文。
func buildApprovalNotifyMessage(siteName string, req *AdminApprovalRequest, link string) (string, string) {
	siteName = strings.TrimSpace(siteName)
	if siteName == "" {
		siteName = approvalNotifyDefaultSite
	}
	label := approvalActionLabel(req.Action)
	title := fmt.Sprintf("[%s] 新的审批申请 #%d：%s", siteName, req.ID, label)

	requester := strings.TrimSpace(req.RequesterEmail)
	if requester == "" {
		requester = "user #" + strconv.FormatInt(req.RequesterUserID, 10)
	}
	target := strings.TrimSpace(req.TargetSummary)
	if target == "" {
		target = "-"
	}
	createdAt := req.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now()
	}
	lines := []string{
		fmt.Sprintf("**申请人**：%s", requester),
		fmt.Sprintf("**操作**：%s（%s %s）", label, req.Method, req.RequestPath),
		fmt.Sprintf("**对象**：%s", target),
		fmt.Sprintf("**时间**：%s", createdAt.Local().Format(approvalNotifyTimeLayout)),
		fmt.Sprintf("**有效期至**：%s", req.ExpiresAt.Local().Format(approvalNotifyTimeLayout)),
	}
	if link != "" {
		lines = append(lines, fmt.Sprintf("[前往审批](%s)", link))
	}
	return title, strings.Join(lines, "\n\n")
}
