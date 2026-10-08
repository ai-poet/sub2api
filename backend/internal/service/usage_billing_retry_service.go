package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
)

// 扣费失败重试服务（fork 本地功能）。
//
//   - 网关后扣费失败 → EnqueueFailedBilling 把扣费命令落到 usage_billing_retries，
//     usage_log 保留应扣金额（不再清零）。
//   - 时间轮每 30 秒领取到期行，用同一个 UsageBillingRepository.Apply 原样重放：
//     去重表让重放幂等，成功后只做缓存失效与 user×platform 配额补记。
//   - 不可重试的错误（去重指纹冲突、用户 / 订阅已不存在）或重试耗尽 → failed，
//     留在表里供人工处理；使用记录页按 (request_id, api_key_id) 标注状态。
//   - 失败入队与放弃都会触发 Server酱³ 推送（开关 billing_failure_notify_serverchan_enabled，
//     复用分组运行状态推送的 UID / SendKey），同一实例至少间隔 10 分钟一条。

const (
	usageBillingRetryLogComponent = "service.usage_billing_retry"
	usageBillingRetryWorkerName   = "usage_billing_retry"
	usageBillingRetryInterval     = 30 * time.Second
	usageBillingRetryBatchSize    = 200
	usageBillingRetryLease        = 2 * time.Minute
	usageBillingRetryBaseBackoff  = 30 * time.Second
	usageBillingRetryMaxBackoff   = time.Hour
	usageBillingRetryMaxAttempts  = 24
	usageBillingRetryRunTimeout   = 5 * time.Minute
	usageBillingRetryNotifyMinGap = 10 * time.Minute
	usageBillingRetryNotifyBudget = 60 * time.Second
	usageBillingRetryDefaultSite  = "Sub2API"
)

// UsageBillingRetryService 扣费失败的落库、重放、标注与告警。
type UsageBillingRetryService struct {
	repo                  UsageBillingRetryRepository
	billingRepo           UsageBillingRepository
	billingCache          *BillingCacheService
	userPlatformQuotaRepo UserPlatformQuotaRepository
	settingRepo           SettingRepository
	timingWheel           *TimingWheelService
	cfg                   *config.Config
	client                *serverChanClient
	now                   func() time.Time

	startOnce sync.Once
	stopOnce  sync.Once
	running   atomic.Bool

	notifyMu      sync.Mutex
	lastNotifyAt  time.Time
	notifyInFly   atomic.Bool
	newFailures   atomic.Int64 // 自上次推送以来入队的失败笔数
	abandoned     atomic.Int64 // 自上次推送以来放弃的笔数
	notifyDeliver func(ctx context.Context, cfg billingFailureNotifyConfig, title, desp string) error
}

type billingFailureNotifyConfig struct {
	Enabled     bool
	UID         string
	SendKey     string
	SiteName    string
	FrontendURL string
}

func NewUsageBillingRetryService(
	repo UsageBillingRetryRepository,
	billingRepo UsageBillingRepository,
	billingCache *BillingCacheService,
	userPlatformQuotaRepo UserPlatformQuotaRepository,
	settingRepo SettingRepository,
	timingWheel *TimingWheelService,
	cfg *config.Config,
) *UsageBillingRetryService {
	svc := &UsageBillingRetryService{
		repo:                  repo,
		billingRepo:           billingRepo,
		billingCache:          billingCache,
		userPlatformQuotaRepo: userPlatformQuotaRepo,
		settingRepo:           settingRepo,
		timingWheel:           timingWheel,
		cfg:                   cfg,
		client:                newServerChanClient(),
		now:                   time.Now,
	}
	svc.notifyDeliver = svc.deliverServerChan
	return svc
}

// Start 挂上时间轮；缺依赖时不启动（只落库不重放，日志提示）。
func (s *UsageBillingRetryService) Start() {
	if s == nil {
		return
	}
	if s.repo == nil || s.billingRepo == nil || s.timingWheel == nil {
		logger.LegacyPrintf(usageBillingRetryLogComponent, "[BillingRetry] not started (missing deps)")
		return
	}
	s.startOnce.Do(func() {
		s.timingWheel.ScheduleRecurring(usageBillingRetryWorkerName, usageBillingRetryInterval, s.runOnce)
		logger.LegacyPrintf(usageBillingRetryLogComponent, "[BillingRetry] started (interval=%s batch=%d max_attempts=%d)", usageBillingRetryInterval, usageBillingRetryBatchSize, usageBillingRetryMaxAttempts)
	})
}

func (s *UsageBillingRetryService) Stop() {
	if s == nil {
		return
	}
	s.stopOnce.Do(func() {
		if s.timingWheel != nil {
			s.timingWheel.Cancel(usageBillingRetryWorkerName)
		}
	})
}

// EnqueueFailedBilling 实现 BillingRetryEnqueuer：把失败的扣费命令落库。
// 落库本身也失败（数据库不可用）时把完整命令打进 ALERT 日志，便于事后人工补录。
func (s *UsageBillingRetryService) EnqueueFailedBilling(ctx context.Context, requestID string, usageLog *UsageLog, p *postUsageBillingParams, cause error) {
	if s == nil || s.repo == nil || p == nil || p.Cost == nil {
		return
	}
	if p.SimpleModeKeyRateLimitOnly {
		// 简单模式只记 Key 窗口用量，没有钱可补
		return
	}
	cmd := buildUsageBillingCommand(requestID, usageLog, p)
	if cmd == nil {
		return
	}
	cmd.Normalize()
	if cmd.RequestID == "" || !usageBillingCommandHasEffect(cmd) {
		return
	}
	item := &UsageBillingRetry{
		RequestID:   cmd.RequestID,
		APIKeyID:    cmd.APIKeyID,
		UserID:      cmd.UserID,
		AccountID:   cmd.AccountID,
		Platform:    strings.TrimSpace(p.Platform),
		ActualCost:  QuantizeUsageBillingAmount(p.Cost.ActualCost),
		Command:     cmd,
		Status:      UsageBillingRetryStatusPending,
		LastError:   errorText(cause),
		NextRetryAt: s.now().Add(usageBillingRetryBaseBackoff),
	}
	if usageLog != nil && usageLog.GroupID != nil {
		gid := *usageLog.GroupID
		item.GroupID = &gid
	} else if p.APIKey != nil && p.APIKey.GroupID != nil {
		gid := *p.APIKey.GroupID
		item.GroupID = &gid
	}

	dbCtx, cancel := detachedBillingContext(ctx)
	defer cancel()
	queued, err := s.repo.Enqueue(dbCtx, item)
	if err != nil {
		payload, _ := json.Marshal(cmd)
		logger.LegacyPrintf(usageBillingRetryLogComponent, "ALERT: enqueue failed billing failed user=%d api_key=%d request=%s actual_cost=%.8f cause=%v enqueue_err=%v command=%s",
			item.UserID, item.APIKeyID, item.RequestID, item.ActualCost, cause, err, string(payload))
		return
	}
	if !queued {
		return
	}
	logger.LegacyPrintf(usageBillingRetryLogComponent, "[BillingRetry] queued user=%d api_key=%d request=%s actual_cost=%.8f cause=%v",
		item.UserID, item.APIKeyID, item.RequestID, item.ActualCost, cause)
	s.newFailures.Add(1)
	s.maybeNotify()
}

// runOnce 领取到期行并逐条重放；时间轮回调，单实例内不重入。
func (s *UsageBillingRetryService) runOnce() {
	if s == nil || s.repo == nil || s.billingRepo == nil {
		return
	}
	if !s.running.CompareAndSwap(false, true) {
		return
	}
	defer s.running.Store(false)

	ctx, cancel := context.WithTimeout(context.Background(), usageBillingRetryRunTimeout)
	defer cancel()

	items, err := s.repo.ClaimDue(ctx, s.now(), usageBillingRetryLease, usageBillingRetryBatchSize)
	if err != nil {
		logger.LegacyPrintf(usageBillingRetryLogComponent, "[BillingRetry] claim due failed: %v", err)
		return
	}
	for _, item := range items {
		if ctx.Err() != nil {
			break
		}
		s.replay(ctx, item)
	}
	s.maybeNotify()
}

// replay 重放一笔命令；返回是否已结清。
func (s *UsageBillingRetryService) replay(ctx context.Context, item *UsageBillingRetry) bool {
	if item == nil || item.Command == nil {
		return false
	}
	cmd := cloneUsageBillingCommand(item.Command)
	applyCtx, cancel := context.WithTimeout(ctx, postUsageBillingTimeout)
	result, err := s.billingRepo.Apply(applyCtx, cmd)
	cancel()
	if errors.Is(err, ErrAccountNotFound) && cmd.AccountQuotaCost > 0 {
		// 上游账号在请求进行中被删：账号配额无处可记，但用户的钱照扣。
		// 指纹由原始金额派生且已固化在命令里，去重仍按原键匹配。
		retry := cloneUsageBillingCommand(item.Command)
		retry.AccountQuotaCost = 0
		applyCtx, cancel = context.WithTimeout(ctx, postUsageBillingTimeout)
		result, err = s.billingRepo.Apply(applyCtx, retry)
		cancel()
	}
	if err == nil {
		if markErr := s.repo.MarkSettled(ctx, item.ID, s.now()); markErr != nil {
			logger.LegacyPrintf(usageBillingRetryLogComponent, "[BillingRetry] mark settled failed id=%d: %v", item.ID, markErr)
		}
		if result != nil && result.Applied {
			s.afterSettled(ctx, item, cmd)
			logger.LegacyPrintf(usageBillingRetryLogComponent, "[BillingRetry] settled id=%d user=%d api_key=%d request=%s actual_cost=%.8f attempt=%d",
				item.ID, item.UserID, item.APIKeyID, item.RequestID, item.ActualCost, item.Attempts)
		} else {
			logger.LegacyPrintf(usageBillingRetryLogComponent, "[BillingRetry] already billed id=%d request=%s (dedup hit), settled without charge", item.ID, item.RequestID)
		}
		return true
	}
	if isPermanentUsageBillingError(err) || item.Attempts >= usageBillingRetryMaxAttempts {
		if markErr := s.repo.MarkFailed(ctx, item.ID, errorText(err)); markErr != nil {
			logger.LegacyPrintf(usageBillingRetryLogComponent, "[BillingRetry] mark failed failed id=%d: %v", item.ID, markErr)
		}
		logger.LegacyPrintf(usageBillingRetryLogComponent, "ALERT: billing retry abandoned id=%d user=%d api_key=%d request=%s actual_cost=%.8f attempts=%d err=%v",
			item.ID, item.UserID, item.APIKeyID, item.RequestID, item.ActualCost, item.Attempts, err)
		s.abandoned.Add(1)
		return false
	}
	next := s.now().Add(usageBillingRetryBackoff(item.Attempts))
	if markErr := s.repo.MarkRetry(ctx, item.ID, errorText(err), next); markErr != nil {
		logger.LegacyPrintf(usageBillingRetryLogComponent, "[BillingRetry] mark retry failed id=%d: %v", item.ID, markErr)
	}
	logger.LegacyPrintf(usageBillingRetryLogComponent, "[BillingRetry] retry later id=%d request=%s attempt=%d next=%s err=%v",
		item.ID, item.RequestID, item.Attempts, next.Format(time.RFC3339), err)
	return false
}

// afterSettled 重放成功后的缓存收尾：余额 / 订阅 / Key 限流缓存失效，user×platform 配额补记。
// 低余额、账号配额等通知不在这里补发，下一次正常请求会按新值触发。
func (s *UsageBillingRetryService) afterSettled(ctx context.Context, item *UsageBillingRetry, cmd *UsageBillingCommand) {
	if s.billingCache == nil || item == nil || cmd == nil {
		return
	}
	if cmd.BalanceCost > 0 {
		if err := s.billingCache.InvalidateUserBalance(ctx, item.UserID); err != nil {
			logger.LegacyPrintf(usageBillingRetryLogComponent, "[BillingRetry] invalidate balance cache failed user=%d: %v", item.UserID, err)
		}
	}
	if cmd.SubscriptionCost > 0 && item.GroupID != nil {
		if err := s.billingCache.InvalidateSubscription(ctx, item.UserID, *item.GroupID); err != nil {
			logger.LegacyPrintf(usageBillingRetryLogComponent, "[BillingRetry] invalidate subscription cache failed user=%d group=%d: %v", item.UserID, *item.GroupID, err)
		}
	}
	if cmd.APIKeyRateLimitCost > 0 {
		if err := s.billingCache.InvalidateAPIKeyRateLimit(ctx, item.APIKeyID); err != nil {
			logger.LegacyPrintf(usageBillingRetryLogComponent, "[BillingRetry] invalidate api key rate limit cache failed key=%d: %v", item.APIKeyID, err)
		}
	}
	// 与 finalizePostUsageBilling 同一条件：余额模式、有平台、有 limit 的用户才累加
	platform := strings.TrimSpace(item.Platform)
	if cmd.SubscriptionCost == 0 && cmd.BalanceCost > 0 && platform != "" && item.ActualCost > 0 &&
		s.billingCache.HasUserPlatformQuotaLimit(ctx, item.UserID, platform) {
		s.billingCache.IncrementUserPlatformQuotaUsage(item.UserID, platform, item.ActualCost)
		if s.userPlatformQuotaRepo != nil && (s.cfg == nil || !s.cfg.Database.UserPlatformQuotaFlusherEnabled) {
			if err := s.userPlatformQuotaRepo.IncrementUsageWithReset(ctx, item.UserID, platform, item.ActualCost, time.Now().UTC()); err != nil {
				logger.LegacyPrintf(usageBillingRetryLogComponent, "ALERT: incr user platform quota DB failed user=%d platform=%s cost=%f: %v", item.UserID, platform, item.ActualCost, err)
			}
		}
	}
}

// Annotate 按 (request_id, api_key_id) 给使用记录标上扣费状态（只读，不落库）。
func (s *UsageBillingRetryService) Annotate(ctx context.Context, logs []UsageLog) {
	if s == nil || s.repo == nil || len(logs) == 0 {
		return
	}
	keys := make([]UsageBillingRetryKey, 0, len(logs))
	for i := range logs {
		if strings.TrimSpace(logs[i].RequestID) == "" {
			continue
		}
		keys = append(keys, UsageBillingRetryKey{RequestID: logs[i].RequestID, APIKeyID: logs[i].APIKeyID})
	}
	if len(keys) == 0 {
		return
	}
	items, err := s.repo.ListByKeys(ctx, keys)
	if err != nil {
		logger.LegacyPrintf(usageBillingRetryLogComponent, "[BillingRetry] annotate usage logs failed: %v", err)
		return
	}
	if len(items) == 0 {
		return
	}
	byKey := make(map[UsageBillingRetryKey]*UsageBillingRetry, len(items))
	for _, item := range items {
		byKey[UsageBillingRetryKey{RequestID: item.RequestID, APIKeyID: item.APIKeyID}] = item
	}
	for i := range logs {
		item, ok := byKey[UsageBillingRetryKey{RequestID: logs[i].RequestID, APIKeyID: logs[i].APIKeyID}]
		if !ok {
			continue
		}
		logs[i].BillingStatus = item.Status
		logs[i].BillingError = item.LastError
		logs[i].BillingAttempts = item.Attempts
	}
}

// CountBillingFailures 实现 BillingFailureSource：窗口内入队的失败笔数。
func (s *UsageBillingRetryService) CountBillingFailures(ctx context.Context, start, end time.Time) (int64, error) {
	if s == nil || s.repo == nil {
		return 0, errors.New("usage billing retry repository not configured")
	}
	return s.repo.CountCreatedBetween(ctx, start, end)
}

// UnsettledBillingCount 实现 BillingFailureSource：当前待补扣 + 已放弃的笔数。
func (s *UsageBillingRetryService) UnsettledBillingCount(ctx context.Context) (int64, error) {
	if s == nil || s.repo == nil {
		return 0, errors.New("usage billing retry repository not configured")
	}
	summary, err := s.repo.Summary(ctx)
	if err != nil {
		return 0, err
	}
	if summary == nil {
		return 0, nil
	}
	return summary.Pending + summary.Failed, nil
}

// Summary 当前未结清的笔数与金额（管理接口 / 推送用）。
func (s *UsageBillingRetryService) Summary(ctx context.Context) (*UsageBillingRetrySummary, error) {
	if s == nil || s.repo == nil {
		return &UsageBillingRetrySummary{}, nil
	}
	return s.repo.Summary(ctx)
}

// maybeNotify 有新失败 / 新放弃且距上次推送超过最小间隔时异步推一条摘要。
func (s *UsageBillingRetryService) maybeNotify() {
	if s == nil || s.settingRepo == nil {
		return
	}
	if s.newFailures.Load() == 0 && s.abandoned.Load() == 0 {
		return
	}
	s.notifyMu.Lock()
	if !s.lastNotifyAt.IsZero() && s.now().Sub(s.lastNotifyAt) < usageBillingRetryNotifyMinGap {
		s.notifyMu.Unlock()
		return
	}
	s.lastNotifyAt = s.now()
	s.notifyMu.Unlock()

	if !s.notifyInFly.CompareAndSwap(false, true) {
		return
	}
	newFailures := s.newFailures.Swap(0)
	abandoned := s.abandoned.Swap(0)
	go func() {
		defer s.notifyInFly.Store(false)
		if err := s.deliverNotify(newFailures, abandoned); err != nil {
			logger.LegacyPrintf(usageBillingRetryLogComponent, "[BillingRetry] serverchan push failed: %v", err)
		}
	}()
}

func (s *UsageBillingRetryService) loadNotifyConfig(ctx context.Context) (billingFailureNotifyConfig, error) {
	cfg := billingFailureNotifyConfig{SiteName: usageBillingRetryDefaultSite}
	if s == nil || s.settingRepo == nil {
		return cfg, errors.New("setting repository is not configured")
	}
	values, err := s.settingRepo.GetMultiple(ctx, []string{
		SettingKeyBillingFailureNotifyServerChanEnabled,
		SettingKeyGroupStatusNotifyServerChanUID,
		SettingKeyGroupStatusNotifyServerChanSendKey,
		SettingKeySiteName,
		SettingKeyFrontendURL,
	})
	if err != nil {
		return cfg, err
	}
	cfg.Enabled = values[SettingKeyBillingFailureNotifyServerChanEnabled] == "true"
	cfg.UID = strings.TrimSpace(values[SettingKeyGroupStatusNotifyServerChanUID])
	cfg.SendKey = strings.TrimSpace(values[SettingKeyGroupStatusNotifyServerChanSendKey])
	cfg.FrontendURL = strings.TrimSpace(values[SettingKeyFrontendURL])
	if site := strings.TrimSpace(values[SettingKeySiteName]); site != "" {
		cfg.SiteName = site
	}
	return cfg, nil
}

// deliverNotify 同步投递一条摘要推送；开关关闭 / 未配置时静默返回 nil。测试可直接调用。
func (s *UsageBillingRetryService) deliverNotify(newFailures, abandoned int64) error {
	ctx, cancel := context.WithTimeout(context.Background(), usageBillingRetryNotifyBudget)
	defer cancel()

	cfg, err := s.loadNotifyConfig(ctx)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	if !cfg.Enabled {
		return nil
	}
	if cfg.UID == "" || cfg.SendKey == "" {
		logger.LegacyPrintf(usageBillingRetryLogComponent, "[BillingRetry] push skipped: serverchan uid/sendkey not configured")
		return nil
	}
	if err := ValidateServerChanUID(cfg.UID); err != nil {
		logger.LegacyPrintf(usageBillingRetryLogComponent, "[BillingRetry] push skipped: invalid serverchan uid")
		return nil
	}
	summary, err := s.Summary(ctx)
	if err != nil {
		logger.LegacyPrintf(usageBillingRetryLogComponent, "[BillingRetry] load summary for push failed: %v", err)
		summary = &UsageBillingRetrySummary{}
	}
	title, desp := buildBillingFailureNotifyMessage(cfg.SiteName, cfg.FrontendURL, newFailures, abandoned, summary, s.now())
	if s.notifyDeliver == nil {
		return nil
	}
	if err := s.notifyDeliver(ctx, cfg, title, desp); err != nil {
		return redactServerChanSecret(err, cfg.SendKey)
	}
	logger.LegacyPrintf(usageBillingRetryLogComponent, "[BillingRetry] pushed via serverchan new=%d abandoned=%d pending=%d failed=%d", newFailures, abandoned, summary.Pending, summary.Failed)
	return nil
}

func (s *UsageBillingRetryService) deliverServerChan(ctx context.Context, cfg billingFailureNotifyConfig, title, desp string) error {
	if s == nil || s.client == nil {
		return errors.New("serverchan client not configured")
	}
	return s.client.sendWithRetry(ctx, cfg.UID, cfg.SendKey, title, desp)
}

func buildBillingFailureNotifyMessage(siteName, frontendURL string, newFailures, abandoned int64, summary *UsageBillingRetrySummary, at time.Time) (string, string) {
	if summary == nil {
		summary = &UsageBillingRetrySummary{}
	}
	title := fmt.Sprintf("【%s】扣费失败 %d 笔，待补扣 %d 笔", siteName, newFailures, summary.Pending)
	if abandoned > 0 {
		title = fmt.Sprintf("【%s】扣费重试放弃 %d 笔，待人工处理", siteName, abandoned)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "- 新增扣费失败：%d 笔（自上次推送起）\n", newFailures)
	fmt.Fprintf(&b, "- 本次放弃重试：%d 笔\n", abandoned)
	fmt.Fprintf(&b, "- 当前待补扣：%d 笔，合计 $%s\n", summary.Pending, strconv.FormatFloat(summary.PendingCost, 'f', 4, 64))
	fmt.Fprintf(&b, "- 已放弃（需人工处理）：%d 笔，合计 $%s\n", summary.Failed, strconv.FormatFloat(summary.FailedCost, 'f', 4, 64))
	fmt.Fprintf(&b, "- 时间：%s\n", at.In(timezone.Location()).Format("2006-01-02 15:04:05 MST"))
	if base := strings.TrimRight(strings.TrimSpace(frontendURL), "/"); base != "" {
		fmt.Fprintf(&b, "- 使用记录：%s/admin/usage\n", base)
	}
	_, _ = b.WriteString("\n失败原因见系统日志 gateway.record_usage_failed / openai.record_usage_failed；后台每 30 秒按退避自动重放，放弃的笔数需人工核对补扣。")
	return title, b.String()
}

// usageBillingRetryBackoff 第 attempt 次失败后的等待：30s × 2^(attempt-1)，封顶 1h。
func usageBillingRetryBackoff(attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	backoff := usageBillingRetryBaseBackoff
	for i := 1; i < attempt && backoff < usageBillingRetryMaxBackoff; i++ {
		backoff *= 2
	}
	if backoff > usageBillingRetryMaxBackoff {
		backoff = usageBillingRetryMaxBackoff
	}
	return backoff
}

// isPermanentUsageBillingError 重放也不可能成功的错误。
func isPermanentUsageBillingError(err error) bool {
	return errors.Is(err, ErrUsageBillingRequestConflict) ||
		errors.Is(err, ErrUsageBillingRequestIDRequired) ||
		errors.Is(err, ErrUserNotFound) ||
		errors.Is(err, ErrSubscriptionNotFound) ||
		errors.Is(err, ErrAccountNotFound)
}

func cloneUsageBillingCommand(cmd *UsageBillingCommand) *UsageBillingCommand {
	if cmd == nil {
		return nil
	}
	clone := *cmd
	if cmd.SubscriptionID != nil {
		id := *cmd.SubscriptionID
		clone.SubscriptionID = &id
	}
	return &clone
}

func errorText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// SetBillingRetryEnqueuer 挂上扣费失败入队钩子（fork）。
func (s *GatewayService) SetBillingRetryEnqueuer(enqueuer BillingRetryEnqueuer) {
	if s == nil {
		return
	}
	s.billingRetry = enqueuer
}

// SetBillingRetryEnqueuer 挂上扣费失败入队钩子（fork）。
func (s *OpenAIGatewayService) SetBillingRetryEnqueuer(enqueuer BillingRetryEnqueuer) {
	if s == nil {
		return
	}
	s.billingRetry = enqueuer
}

// enqueueBillingRetry 后扣费失败时的统一入口：钩子未挂时只保留日志。
func enqueueBillingRetry(ctx context.Context, enqueuer BillingRetryEnqueuer, requestID string, usageLog *UsageLog, p *postUsageBillingParams, cause error) {
	if enqueuer == nil {
		return
	}
	enqueuer.EnqueueFailedBilling(ctx, requestID, usageLog, p, cause)
}

// SetBillingFailureSource 让运维告警评估器读到扣费失败指标（fork）。
func (s *OpsAlertEvaluatorService) SetBillingFailureSource(src BillingFailureSource) {
	if s == nil {
		return
	}
	s.billingFailureSource = src
}
