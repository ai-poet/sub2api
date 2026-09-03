package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/imroc/req/v3"
)

// 分组运行状态 → Server酱³ 推送（本 fork 自有功能）。
//
// 探测结果落库后若产生稳定状态切换事件（down / up），由 GroupStatusProbeService 调用
// NotifyTransition；这里只做廉价判断然后异步投递，推送失败只写日志，绝不影响探测流程。
// 发送协议移植自 flow-router 的 serverchan3：POST https://{uid}.push.ft07.com/send/{sendkey}.send，
// 表单字段 title / desp，成功 = HTTP 2xx 且响应 JSON code == 0。

const (
	groupStatusNotifyAttemptTimeout = 15 * time.Second
	groupStatusNotifyMaxAttempts    = 3
	groupStatusNotifyDeliveryBudget = 60 * time.Second
	groupStatusNotifyErrorDetailMax = 200
	groupStatusNotifyLogComponent   = "service.group_status_notify"
	groupStatusNotifyDefaultSite    = "Sub2API"
	groupStatusNotifyTimeLayout     = "2006-01-02 15:04:05 MST"
)

var (
	// UID 会被拼进推送域名，只允许安全字符，防止把请求打到别的主机上。
	groupStatusServerChanUIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)

	ErrGroupStatusNotifyInvalidUID    = infraerrors.BadRequest("GROUP_STATUS_NOTIFY_INVALID_UID", "Server酱 UID may only contain letters, digits, '-' or '_'")
	ErrGroupStatusNotifyNotConfigured = infraerrors.BadRequest("GROUP_STATUS_NOTIFY_NOT_CONFIGURED", "Server酱 UID and SendKey are required")
)

// ValidateServerChanUID 校验 Server酱³ UID 是否只包含可安全拼入主机名的字符。
func ValidateServerChanUID(uid string) error {
	if !groupStatusServerChanUIDPattern.MatchString(strings.TrimSpace(uid)) {
		return ErrGroupStatusNotifyInvalidUID
	}
	return nil
}

// serverChanClient 只负责 HTTP 传输；baseURL 为空时按 uid 拼默认域名，测试可指向 httptest.Server。
type serverChanClient struct {
	baseURL string
	sleep   func(ctx context.Context, d time.Duration) error
}

func newServerChanClient() *serverChanClient {
	return &serverChanClient{sleep: serverChanSleep}
}

func serverChanSleep(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func (c *serverChanClient) endpoint(uid, sendkey string) string {
	base := strings.TrimRight(strings.TrimSpace(c.baseURL), "/")
	if base == "" {
		base = fmt.Sprintf("https://%s.push.ft07.com", strings.TrimSpace(uid))
	}
	return fmt.Sprintf("%s/send/%s.send", base, url.PathEscape(strings.TrimSpace(sendkey)))
}

// send 单次发送，不重试。返回的错误已脱敏（不含 sendkey）。
func (c *serverChanClient) send(ctx context.Context, uid, sendkey, title, desp string) error {
	resp, err := req.C().
		SetTimeout(groupStatusNotifyAttemptTimeout).
		R().
		SetContext(ctx).
		SetFormData(map[string]string{
			"title": title,
			"desp":  desp,
		}).
		Post(c.endpoint(uid, sendkey))
	if err != nil {
		return redactServerChanSecret(fmt.Errorf("serverchan request failed: %w", err), sendkey)
	}
	if !resp.IsSuccessState() {
		return fmt.Errorf("serverchan returned http %d", resp.StatusCode)
	}

	var result struct {
		Code    *int   `json:"code"`
		Message string `json:"message"`
		Msg     string `json:"msg"`
	}
	if err := json.Unmarshal(resp.Bytes(), &result); err == nil && result.Code != nil && *result.Code != 0 {
		reason := strings.TrimSpace(result.Message)
		if reason == "" {
			reason = strings.TrimSpace(result.Msg)
		}
		if reason == "" {
			reason = fmt.Sprintf("code %d", *result.Code)
		}
		return redactServerChanSecret(errors.New("serverchan returned "+reason), sendkey)
	}
	return nil
}

// sendWithRetry 最多 groupStatusNotifyMaxAttempts 次，指数退避 1s / 2s / 4s …（上限 30s），ctx 取消即停。
func (c *serverChanClient) sendWithRetry(ctx context.Context, uid, sendkey, title, desp string) error {
	var lastErr error
	for attempt := 1; attempt <= groupStatusNotifyMaxAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			if lastErr != nil {
				return lastErr
			}
			return err
		}
		err := c.send(ctx, uid, sendkey, title, desp)
		if err == nil {
			return nil
		}
		lastErr = err
		if attempt == groupStatusNotifyMaxAttempts {
			break
		}
		if err := c.sleep(ctx, serverChanBackoff(attempt)); err != nil {
			return lastErr
		}
	}
	return lastErr
}

// serverChanBackoff 第 1 次重试等 1s，第 2 次 2s，第 3 次 4s …，上限 30s。
func serverChanBackoff(attempt int) time.Duration {
	const maxDelay = 30 * time.Second
	if attempt < 1 {
		attempt = 1
	}
	if attempt > 6 {
		return maxDelay
	}
	delay := time.Duration(1<<uint(attempt-1)) * time.Second
	if delay > maxDelay || delay <= 0 {
		return maxDelay
	}
	return delay
}

// redactServerChanSecret 把错误信息里的 sendkey（含 URL 转义形式）替换成 ***。
// req 的 *url.Error 会把完整请求 URL 带进 Error()，而 URL 里就有 sendkey。
func redactServerChanSecret(err error, sendkey string) error {
	if err == nil {
		return nil
	}
	key := strings.TrimSpace(sendkey)
	if key == "" {
		return err
	}
	msg := err.Error()
	replaced := strings.ReplaceAll(msg, key, "***")
	if escaped := url.PathEscape(key); escaped != key {
		replaced = strings.ReplaceAll(replaced, escaped, "***")
	}
	if replaced == msg {
		return err
	}
	return errors.New(replaced)
}

type groupStatusServerChanConfig struct {
	Enabled  bool
	UID      string
	SendKey  string
	SiteName string
}

// GroupStatusNotifyService 在分组稳定状态变红 / 恢复时通过 Server酱³ 推送提醒。
type GroupStatusNotifyService struct {
	settingRepo SettingRepository
	client      *serverChanClient
}

func NewGroupStatusNotifyService(settingRepo SettingRepository) *GroupStatusNotifyService {
	return &GroupStatusNotifyService{
		settingRepo: settingRepo,
		client:      newServerChanClient(),
	}
}

func (s *GroupStatusNotifyService) loadConfig(ctx context.Context) (groupStatusServerChanConfig, error) {
	cfg := groupStatusServerChanConfig{SiteName: groupStatusNotifyDefaultSite}
	if s == nil || s.settingRepo == nil {
		return cfg, errors.New("setting repository is not configured")
	}
	values, err := s.settingRepo.GetMultiple(ctx, []string{
		SettingKeyGroupStatusNotifyServerChanEnabled,
		SettingKeyGroupStatusNotifyServerChanUID,
		SettingKeyGroupStatusNotifyServerChanSendKey,
		SettingKeySiteName,
	})
	if err != nil {
		return cfg, err
	}
	cfg.Enabled = values[SettingKeyGroupStatusNotifyServerChanEnabled] == "true"
	cfg.UID = strings.TrimSpace(values[SettingKeyGroupStatusNotifyServerChanUID])
	cfg.SendKey = strings.TrimSpace(values[SettingKeyGroupStatusNotifyServerChanSendKey])
	if site := strings.TrimSpace(values[SettingKeySiteName]); site != "" {
		cfg.SiteName = site
	}
	return cfg, nil
}

// NotifyTransition 由探测落库后调用：只对 down / up 事件、且分组开启了推送时异步投递。
// nil 接收者安全，调用方无需判空。
func (s *GroupStatusNotifyService) NotifyTransition(group *Group, cfg *GroupStatusConfig, event *GroupStatusEvent) {
	if s == nil || group == nil || cfg == nil || event == nil {
		return
	}
	if event.EventType != GroupStatusEventDown && event.EventType != GroupStatusEventUp {
		return
	}
	if !cfg.NotifyEnabled {
		return
	}
	groupCopy := *group
	eventCopy := *event
	go s.deliver(&groupCopy, &eventCopy)
}

// deliver 同步投递一条状态切换推送，供 NotifyTransition 在 goroutine 里调用；测试可直接调用。
func (s *GroupStatusNotifyService) deliver(group *Group, event *GroupStatusEvent) {
	ctx, cancel := context.WithTimeout(context.Background(), groupStatusNotifyDeliveryBudget)
	defer cancel()

	cfg, err := s.loadConfig(ctx)
	if err != nil {
		logger.LegacyPrintf(groupStatusNotifyLogComponent, "[GroupStatusNotify] group=%d event=%s load config failed: %v", group.ID, event.EventType, err)
		return
	}
	if !cfg.Enabled {
		return
	}
	if cfg.UID == "" || cfg.SendKey == "" {
		logger.LegacyPrintf(groupStatusNotifyLogComponent, "[GroupStatusNotify] group=%d event=%s skipped: serverchan uid/sendkey not configured", group.ID, event.EventType)
		return
	}
	if err := ValidateServerChanUID(cfg.UID); err != nil {
		logger.LegacyPrintf(groupStatusNotifyLogComponent, "[GroupStatusNotify] group=%d event=%s skipped: invalid serverchan uid", group.ID, event.EventType)
		return
	}

	title, desp := buildGroupStatusNotifyMessage(cfg.SiteName, group, event)
	if err := s.client.sendWithRetry(ctx, cfg.UID, cfg.SendKey, title, desp); err != nil {
		logger.LegacyPrintf(groupStatusNotifyLogComponent, "[GroupStatusNotify] group=%d event=%s push failed: %v", group.ID, event.EventType, redactServerChanSecret(err, cfg.SendKey))
		return
	}
	logger.LegacyPrintf(groupStatusNotifyLogComponent, "[GroupStatusNotify] group=%d event=%s pushed via serverchan", group.ID, event.EventType)
}

// SendTest 发送一条测试推送。uid / sendkey 留空时回退到已保存的配置；只发一次，不重试。
func (s *GroupStatusNotifyService) SendTest(ctx context.Context, uid, sendkey string) error {
	if s == nil {
		return errors.New("group status notify service is not configured")
	}
	saved, err := s.loadConfig(ctx)
	if err != nil {
		return err
	}
	uid = strings.TrimSpace(uid)
	sendkey = strings.TrimSpace(sendkey)
	if uid == "" {
		uid = saved.UID
	}
	if sendkey == "" {
		sendkey = saved.SendKey
	}
	if uid == "" || sendkey == "" {
		return ErrGroupStatusNotifyNotConfigured
	}
	if err := ValidateServerChanUID(uid); err != nil {
		return err
	}

	title := fmt.Sprintf("[%s] 分组运行状态测试推送", saved.SiteName)
	desp := strings.Join([]string{
		fmt.Sprintf("这是一条来自 %s 的测试消息，收到即表示 Server酱³ 推送配置正确。", saved.SiteName),
		fmt.Sprintf("**时间**：%s", time.Now().Format(groupStatusNotifyTimeLayout)),
	}, "\n\n")
	if err := s.client.send(ctx, uid, sendkey, title, desp); err != nil {
		return redactServerChanSecret(err, sendkey)
	}
	return nil
}

// buildGroupStatusNotifyMessage 生成推送标题与 Markdown 正文。
func buildGroupStatusNotifyMessage(siteName string, group *Group, event *GroupStatusEvent) (string, string) {
	siteName = strings.TrimSpace(siteName)
	if siteName == "" {
		siteName = groupStatusNotifyDefaultSite
	}
	groupName := strings.TrimSpace(group.Name)
	if groupName == "" {
		groupName = fmt.Sprintf("#%d", group.ID)
	}

	var title string
	switch event.EventType {
	case GroupStatusEventDown:
		title = fmt.Sprintf("[%s] 分组「%s」状态变红", siteName, groupName)
	case GroupStatusEventUp:
		title = fmt.Sprintf("[%s] 分组「%s」已恢复", siteName, groupName)
	default:
		title = fmt.Sprintf("[%s] 分组「%s」状态变化", siteName, groupName)
	}

	platform := strings.TrimSpace(group.Platform)
	if platform == "" {
		platform = "-"
	}
	httpCode := "-"
	if event.HTTPCode != nil {
		httpCode = fmt.Sprintf("%d", *event.HTTPCode)
	}
	latency := "-"
	if event.LatencyMS != nil {
		latency = fmt.Sprintf("%d ms", *event.LatencyMS)
	}
	subStatus := strings.TrimSpace(event.SubStatus)
	if subStatus == "" {
		subStatus = "-"
	}
	observedAt := event.ObservedAt
	if observedAt.IsZero() {
		observedAt = time.Now()
	}

	lines := []string{
		fmt.Sprintf("**分组**：%s（#%d / %s）", groupName, group.ID, platform),
		fmt.Sprintf("**状态**：%s → %s", groupStatusStatusLabel(event.FromStatus), groupStatusStatusLabel(event.ToStatus)),
		fmt.Sprintf("**子状态**：%s", subStatus),
		fmt.Sprintf("**HTTP**：%s", httpCode),
		fmt.Sprintf("**延迟**：%s", latency),
		fmt.Sprintf("**错误**：%s", truncateGroupStatusErrorDetail(event.ErrorDetail)),
		fmt.Sprintf("**时间**：%s", observedAt.Local().Format(groupStatusNotifyTimeLayout)),
	}
	return title, strings.Join(lines, "\n\n")
}

func groupStatusStatusLabel(status string) string {
	switch strings.TrimSpace(status) {
	case GroupRuntimeStatusUp:
		return "正常"
	case GroupRuntimeStatusDegraded:
		return "降级"
	case GroupRuntimeStatusDown:
		return "不可用"
	case "":
		return "未知"
	default:
		return strings.TrimSpace(status)
	}
}

// truncateGroupStatusErrorDetail 压平换行并截断，避免推送正文被超长错误撑爆。
func truncateGroupStatusErrorDetail(detail string) string {
	flat := strings.Join(strings.Fields(detail), " ")
	if flat == "" {
		return "-"
	}
	runes := []rune(flat)
	if len(runes) <= groupStatusNotifyErrorDetailMax {
		return flat
	}
	return string(runes[:groupStatusNotifyErrorDetailMax]) + "..."
}
