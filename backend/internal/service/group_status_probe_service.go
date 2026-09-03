package service

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/pkg/openai"
	"github.com/Wei-Shaw/sub2api/internal/pkg/xai"
)

type GroupStatusProbeService struct {
	repo             GroupStatusRepository
	groupRepo        GroupRepository
	scheduler        *SchedulerSnapshotService
	accountTestSvc   *AccountTestService
	gatewaySvc       *GatewayService
	openAIGatewaySvc *OpenAIGatewayService
	// notifier 在稳定状态切换（down / up）时收到通知；可为空
	notifier groupStatusTransitionNotifier
}

// groupStatusTransitionNotifier 消费探测落库后产生的稳定状态切换事件（如 Server酱³ 推送）。
type groupStatusTransitionNotifier interface {
	NotifyTransition(group *Group, cfg *GroupStatusConfig, event *GroupStatusEvent)
}

// SetTransitionNotifier 挂上状态切换通知器；传 nil 表示不通知。
func (s *GroupStatusProbeService) SetTransitionNotifier(n groupStatusTransitionNotifier) {
	if s == nil {
		return
	}
	s.notifier = n
}

func NewGroupStatusProbeService(
	repo GroupStatusRepository,
	groupRepo GroupRepository,
	scheduler *SchedulerSnapshotService,
	accountTestSvc *AccountTestService,
	gatewaySvc *GatewayService,
	openAIGatewaySvc *OpenAIGatewayService,
) *GroupStatusProbeService {
	return &GroupStatusProbeService{
		repo:             repo,
		groupRepo:        groupRepo,
		scheduler:        scheduler,
		accountTestSvc:   accountTestSvc,
		gatewaySvc:       gatewaySvc,
		openAIGatewaySvc: openAIGatewaySvc,
	}
}

type groupStatusProbeAttempt struct {
	Account  *Account
	WaitPlan *AccountWaitPlan
	Acquired bool
	Reason   string
}

func (s *GroupStatusProbeService) ProbeGroupNow(ctx context.Context, groupID int64) (*GroupStatusProbeExecution, error) {
	group, cfg, err := s.ensureProbeTarget(ctx, groupID)
	if err != nil {
		return nil, err
	}
	return s.executeProbe(ctx, group, cfg)
}

func (s *GroupStatusProbeService) ProbeWithConfig(ctx context.Context, cfg *GroupStatusConfig) (*GroupStatusProbeExecution, error) {
	if cfg == nil {
		return nil, ErrGroupStatusInvalidConfig
	}
	group, err := s.groupRepo.GetByID(ctx, cfg.GroupID)
	if err != nil {
		return nil, err
	}
	return s.executeProbe(ctx, group, cfg)
}

func (s *GroupStatusProbeService) ensureProbeTarget(ctx context.Context, groupID int64) (*Group, *GroupStatusConfig, error) {
	group, err := s.groupRepo.GetByID(ctx, groupID)
	if err != nil {
		return nil, nil, err
	}
	cfg, err := s.repo.GetConfig(ctx, groupID)
	if err != nil {
		if !errors.Is(err, ErrGroupStatusConfigNotFound) {
			return nil, nil, err
		}
		defaultCfg := DefaultGroupStatusConfig(group)
		defaultCfg.GroupID = groupID
		cfg, err = s.repo.UpsertConfig(ctx, defaultCfg)
		if err != nil {
			return nil, nil, err
		}
	}
	return group, cfg, nil
}

func (s *GroupStatusProbeService) executeProbe(ctx context.Context, group *Group, cfg *GroupStatusConfig) (*GroupStatusProbeExecution, error) {
	if group == nil || cfg == nil {
		return nil, ErrGroupStatusInvalidConfig
	}
	if err := ValidateGroupStatusConfig(cfg); err != nil {
		return nil, err
	}

	excludedIDs := make(map[int64]struct{})
	maxAttempts := s.maxProbeAttempts(group)
	var (
		firstFailureDetail string
		lastFailureResult  *GroupStatusProbeResult
		lastAccount        *Account
	)

	for attemptNo := 0; attemptNo < maxAttempts; attemptNo++ {
		attempt, selectErr := s.selectProbeAttempt(ctx, group, cfg, excludedIDs)
		if selectErr != nil {
			if lastFailureResult != nil {
				lastFailureResult.SubStatus = "failover_exhausted"
				lastFailureResult.ErrorDetail = mergeProbeErrorDetails(firstFailureDetail, selectErr.Error())
				return s.saveProbeExecution(ctx, group, cfg, lastAccount, lastFailureResult)
			}
			result := s.newProbeSelectionFailureResult(group, cfg, selectErr, firstFailureDetail)
			return s.saveProbeExecution(ctx, group, cfg, nil, result)
		}
		if attempt == nil || attempt.Account == nil {
			result := s.newProbeSelectionFailureResult(group, cfg, errors.New("no schedulable account available"), firstFailureDetail)
			return s.saveProbeExecution(ctx, group, cfg, nil, result)
		}

		account := attempt.Account
		if attempt.WaitPlan != nil && !attempt.Acquired {
			result := s.newProbeQueueingResult(group, cfg, account, attempt)
			return s.saveProbeExecution(ctx, group, cfg, account, result)
		}

		timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(cfg.TimeoutSeconds)*time.Second)
		rawResult, err := s.executeAccountProbe(timeoutCtx, account, cfg)
		cancel()
		if err != nil {
			logger.LegacyPrintf("service.group_status_probe", "[GroupStatusProbe] execute group=%d account=%d err=%v", group.ID, account.ID, err)
		}
		rawResult.GroupID = group.ID
		rawResult.ConfigID = cfg.ID
		if rawResult.ObservedAt.IsZero() {
			rawResult.ObservedAt = time.Now()
		}
		finalizeProbeResult(rawResult, cfg)

		if rawResult.Status != GroupRuntimeStatusDown {
			if firstFailureDetail != "" {
				rawResult.Status = GroupRuntimeStatusDegraded
				rawResult.SubStatus = "failover_recovered"
				rawResult.ErrorDetail = mergeProbeErrorDetails(firstFailureDetail, rawResult.ErrorDetail)
			}
			return s.saveProbeExecution(ctx, group, cfg, account, rawResult)
		}

		lastFailureResult = rawResult
		lastAccount = account
		if firstFailureDetail == "" {
			firstFailureDetail = probeAttemptFailureSummary(account, rawResult)
		}

		if !s.shouldProbeFailover(account, rawResult, err) {
			return s.saveProbeExecution(ctx, group, cfg, account, rawResult)
		}
		excludedIDs[account.ID] = struct{}{}
	}

	if lastFailureResult == nil {
		lastFailureResult = s.newProbeSelectionFailureResult(group, cfg, errors.New("no schedulable account available"), firstFailureDetail)
	} else if firstFailureDetail != "" {
		lastFailureResult.SubStatus = "failover_exhausted"
		lastFailureResult.ErrorDetail = mergeProbeErrorDetails(firstFailureDetail, lastFailureResult.ErrorDetail)
	}
	return s.saveProbeExecution(ctx, group, cfg, lastAccount, lastFailureResult)
}

func (s *GroupStatusProbeService) selectProbeAttempt(ctx context.Context, group *Group, cfg *GroupStatusConfig, excludedIDs map[int64]struct{}) (*groupStatusProbeAttempt, error) {
	if group == nil || cfg == nil {
		return nil, ErrGroupStatusInvalidConfig
	}
	if excludedIDs == nil {
		excludedIDs = make(map[int64]struct{})
	}

	maxSkips := s.maxProbeAttempts(group) + len(excludedIDs) + 8
	for i := 0; i < maxSkips; i++ {
		selection, reason, err := s.selectProbeAttemptWithRealScheduler(ctx, group, cfg, excludedIDs)
		if err != nil {
			return nil, err
		}
		if selection == nil || selection.Account == nil {
			return nil, ErrNoAvailableAccounts
		}

		if selection.Acquired && selection.ReleaseFunc != nil {
			selection.ReleaseFunc()
		}

		if s.probeAccountBlockedByGroupRules(group, selection.Account) {
			excludedIDs[selection.Account.ID] = struct{}{}
			continue
		}

		return &groupStatusProbeAttempt{
			Account:  selection.Account,
			WaitPlan: selection.WaitPlan,
			Acquired: false,
			Reason:   reason,
		}, nil
	}
	return nil, ErrNoAvailableAccounts
}

func (s *GroupStatusProbeService) selectProbeAttemptWithRealScheduler(ctx context.Context, group *Group, cfg *GroupStatusConfig, excludedIDs map[int64]struct{}) (*AccountSelectionResult, string, error) {
	groupID := group.ID
	// Grok 走的是 OpenAI 网关（handler/openai_gateway_handler.go），调度规则、
	// 账号切换上限都在 OpenAIGatewayService 上。探测必须用同一个调度器，否则
	// 选出的账号与真实请求命中的账号可能不是同一批。
	if isOpenAIGatewayPlatform(group.Platform) && s.openAIGatewaySvc != nil {
		selection, err := s.openAIGatewaySvc.SelectAccountWithLoadAwarenessForPlatform(ctx, &groupID, group.Platform, "", cfg.ProbeModel, excludedIDs)
		return selection, "openai_gateway", err
	}
	if s.gatewaySvc != nil {
		if group != nil {
			ctx = s.gatewaySvc.withGroupContext(ctx, group)
		}
		selection, err := s.gatewaySvc.SelectAccountWithLoadAwareness(ctx, &groupID, "", cfg.ProbeModel, excludedIDs, "", 0)
		return selection, "gateway", err
	}
	selection, err := s.selectProbeAttemptFromSnapshot(ctx, group, cfg, excludedIDs)
	return selection, "scheduler_snapshot", err
}

func (s *GroupStatusProbeService) selectProbeAttemptFromSnapshot(ctx context.Context, group *Group, cfg *GroupStatusConfig, excludedIDs map[int64]struct{}) (*AccountSelectionResult, error) {
	if s.scheduler == nil {
		return nil, errors.New("scheduler snapshot service is not configured")
	}
	groupID := group.ID
	accounts, _, err := s.scheduler.ListSchedulableAccounts(ctx, &groupID, group.Platform, false)
	if err != nil {
		return nil, err
	}
	for i := range accounts {
		account := accounts[i]
		if _, excluded := excludedIDs[account.ID]; excluded {
			continue
		}
		if !account.IsSchedulable() {
			continue
		}
		if s.probeAccountBlockedByGroupRules(group, &account) {
			continue
		}
		if account.Platform == PlatformAntigravity && group.Platform != PlatformAntigravity && !account.IsMixedSchedulingEnabled() {
			continue
		}
		if cfg.ProbeModel != "" && !account.IsModelSupported(cfg.ProbeModel) {
			continue
		}
		if cfg.ProbeModel != "" && account.GetRateLimitRemainingTimeWithContext(ctx, cfg.ProbeModel) > 0 {
			continue
		}
		return &AccountSelectionResult{Account: &account}, nil
	}
	return nil, ErrNoAvailableAccounts
}

func (s *GroupStatusProbeService) probeAccountBlockedByGroupRules(group *Group, account *Account) bool {
	if group == nil || account == nil {
		return true
	}
	if group.RequireOAuthOnly && account.Type == AccountTypeAPIKey {
		return true
	}
	if group.RequirePrivacySet && !account.IsPrivacySet() {
		return true
	}
	return false
}

func (s *GroupStatusProbeService) maxProbeAttempts(group *Group) int {
	switches := 10
	if s.gatewaySvc != nil && s.gatewaySvc.cfg != nil && s.gatewaySvc.cfg.Gateway.MaxAccountSwitches > 0 {
		switches = s.gatewaySvc.cfg.Gateway.MaxAccountSwitches
	}
	if group != nil && isOpenAIGatewayPlatform(group.Platform) && s.openAIGatewaySvc != nil && s.openAIGatewaySvc.cfg != nil && s.openAIGatewaySvc.cfg.Gateway.MaxAccountSwitches > 0 {
		switches = s.openAIGatewaySvc.cfg.Gateway.MaxAccountSwitches
	}
	if group != nil && group.Platform == PlatformGemini && s.gatewaySvc != nil && s.gatewaySvc.cfg != nil && s.gatewaySvc.cfg.Gateway.MaxAccountSwitchesGemini > 0 {
		switches = s.gatewaySvc.cfg.Gateway.MaxAccountSwitchesGemini
	}
	if switches < 0 {
		switches = 0
	}
	return switches + 1
}

func (s *GroupStatusProbeService) newProbeSelectionFailureResult(group *Group, cfg *GroupStatusConfig, selectErr error, firstFailureDetail string) *GroupStatusProbeResult {
	detail := ""
	if selectErr != nil {
		detail = selectErr.Error()
	}
	if firstFailureDetail != "" {
		detail = mergeProbeErrorDetails(firstFailureDetail, detail)
	}
	return &GroupStatusProbeResult{
		GroupID:     group.ID,
		ConfigID:    cfg.ID,
		Status:      GroupRuntimeStatusDown,
		SubStatus:   "no_schedulable_account",
		ErrorDetail: detail,
		ObservedAt:  time.Now(),
	}
}

func (s *GroupStatusProbeService) newProbeQueueingResult(group *Group, cfg *GroupStatusConfig, account *Account, attempt *groupStatusProbeAttempt) *GroupStatusProbeResult {
	detail := fmt.Sprintf("selected account %d would wait for an account concurrency slot", account.ID)
	if attempt != nil && attempt.WaitPlan != nil {
		detail = fmt.Sprintf("%s (max_concurrency=%d max_waiting=%d timeout=%s)",
			detail,
			attempt.WaitPlan.MaxConcurrency,
			attempt.WaitPlan.MaxWaiting,
			attempt.WaitPlan.Timeout,
		)
	}
	return &GroupStatusProbeResult{
		GroupID:     group.ID,
		ConfigID:    cfg.ID,
		Status:      GroupRuntimeStatusDegraded,
		SubStatus:   "queueing",
		ErrorDetail: detail,
		ObservedAt:  time.Now(),
	}
}

func (s *GroupStatusProbeService) saveProbeExecution(ctx context.Context, group *Group, cfg *GroupStatusConfig, account *Account, result *GroupStatusProbeResult) (*GroupStatusProbeExecution, error) {
	if result == nil {
		result = s.newProbeSelectionFailureResult(group, cfg, errors.New("empty probe result"), "")
	}
	result.GroupID = group.ID
	result.ConfigID = cfg.ID
	if result.ObservedAt.IsZero() {
		result.ObservedAt = time.Now()
	}
	finalizeProbeResult(result, cfg)

	state, event, err := s.repo.SaveProbeResult(ctx, result)
	if err != nil {
		return nil, err
	}
	// 稳定状态切换（变红 / 恢复）→ 异步推送；notifier 内部自行判断是否发送
	if event != nil && s.notifier != nil {
		s.notifier.NotifyTransition(group, cfg, event)
	}
	return &GroupStatusProbeExecution{
		Group:   group,
		Config:  cfg,
		Account: account,
		Result:  result,
		State:   state,
		Event:   event,
	}, nil
}

func (s *GroupStatusProbeService) shouldProbeFailover(account *Account, result *GroupStatusProbeResult, probeErr error) bool {
	if result == nil || result.HTTPCode == nil {
		return false
	}
	statusCode := *result.HTTPCode
	if account != nil && account.Platform == PlatformGrok {
		// Grok 的 failover 判定是看响应体的：内容策略拒绝必须留在当前账号（换号也会
		// 被拒，只会白白抽干账号池），而 free-usage 耗尽的 400 反过来必须换号。仅凭
		// 状态码判断会把这两种情况都判反。
		if s.openAIGatewaySvc != nil {
			return s.openAIGatewaySvc.shouldFailoverGrokUpstreamError(statusCode, probeUpstreamErrorBody(probeErr))
		}
		return shouldOpenAIProbeFailoverStatus(statusCode)
	}
	if account != nil && account.Platform == PlatformOpenAI {
		if s.openAIGatewaySvc != nil {
			return s.openAIGatewaySvc.shouldFailoverUpstreamError(statusCode)
		}
		return shouldOpenAIProbeFailoverStatus(statusCode)
	}
	if s.gatewaySvc != nil {
		return s.gatewaySvc.shouldFailoverUpstreamError(statusCode)
	}
	return shouldGatewayProbeFailoverStatus(statusCode)
}

// isOpenAIGatewayPlatform 报告该平台的真实请求是否由 OpenAIGatewayService 承载。
func isOpenAIGatewayPlatform(platform string) bool {
	return platform == PlatformOpenAI || platform == PlatformGrok
}

func shouldOpenAIProbeFailoverStatus(statusCode int) bool {
	switch statusCode {
	case http.StatusUnauthorized, http.StatusPaymentRequired, http.StatusForbidden, http.StatusTooManyRequests, 529:
		return true
	default:
		return statusCode >= 500
	}
}

func shouldGatewayProbeFailoverStatus(statusCode int) bool {
	switch statusCode {
	case http.StatusUnauthorized, http.StatusForbidden, http.StatusTooManyRequests, 529:
		return true
	default:
		return statusCode >= 500
	}
}

func probeAttemptFailureSummary(account *Account, result *GroupStatusProbeResult) string {
	parts := make([]string, 0, 4)
	if account != nil {
		parts = append(parts, fmt.Sprintf("account %d", account.ID))
	}
	if result != nil {
		if result.HTTPCode != nil {
			parts = append(parts, fmt.Sprintf("http %d", *result.HTTPCode))
		}
		if result.SubStatus != "" {
			parts = append(parts, result.SubStatus)
		}
		if result.ErrorDetail != "" {
			parts = append(parts, result.ErrorDetail)
		}
	}
	return truncateProbeText(strings.Join(parts, ": "))
}

func mergeProbeErrorDetails(firstFailureDetail, lastDetail string) string {
	firstFailureDetail = strings.TrimSpace(firstFailureDetail)
	lastDetail = strings.TrimSpace(lastDetail)
	switch {
	case firstFailureDetail == "":
		return truncateProbeText(lastDetail)
	case lastDetail == "":
		return truncateProbeText("first failure: " + firstFailureDetail)
	case strings.Contains(lastDetail, firstFailureDetail):
		return truncateProbeText(lastDetail)
	default:
		return truncateProbeText("first failure: " + firstFailureDetail + "; last detail: " + lastDetail)
	}
}

func (s *GroupStatusProbeService) executeAccountProbe(ctx context.Context, account *Account, cfg *GroupStatusConfig) (*GroupStatusProbeResult, error) {
	startedAt := time.Now()
	result := &GroupStatusProbeResult{}
	if account == nil {
		return result, errors.New("nil account")
	}

	var (
		responseText string
		httpCode     *int
		err          error
	)

	switch account.Platform {
	case PlatformOpenAI:
		responseText, httpCode, err = s.probeOpenAI(ctx, account, cfg)
	case PlatformGemini:
		responseText, httpCode, err = s.probeGemini(ctx, account, cfg)
	case PlatformAntigravity:
		responseText, httpCode, err = s.probeAntigravity(ctx, account, cfg)
	case PlatformGrok:
		responseText, httpCode, err = s.probeGrok(ctx, account, cfg)
	default:
		responseText, httpCode, err = s.probeAnthropic(ctx, account, cfg)
	}

	latency := time.Since(startedAt).Milliseconds()
	result.ObservedAt = time.Now()
	result.ResponseExcerpt = truncateProbeText(responseText)
	result.LatencyMS = &latency
	result.HTTPCode = httpCode

	if err != nil {
		result.Status = GroupRuntimeStatusDown
		result.SubStatus = inferProbeSubStatus(httpCode, err)
		result.ErrorDetail = truncateProbeText(sanitizeProbeErrorDetail(err))
		return result, err
	}

	if httpCode != nil && (*httpCode < 200 || *httpCode >= 300) {
		result.Status = GroupRuntimeStatusDown
		result.SubStatus = inferProbeSubStatus(httpCode, nil)
		result.ErrorDetail = fmt.Sprintf("unexpected http status: %d", *httpCode)
		return result, nil
	}

	if !EvaluateGroupStatusValidation(cfg.ValidationMode, cfg.ExpectedKeywords, responseText) {
		result.Status = GroupRuntimeStatusDown
		if strings.TrimSpace(responseText) == "" {
			result.SubStatus = "empty_response"
		} else {
			result.SubStatus = "keyword_mismatch"
		}
		result.ErrorDetail = "probe validation failed"
		return result, nil
	}

	if result.LatencyMS != nil && *result.LatencyMS > cfg.SlowLatencyMS {
		result.Status = GroupRuntimeStatusDegraded
		result.SubStatus = "slow"
		return result, nil
	}

	result.Status = GroupRuntimeStatusUp
	result.SubStatus = "ok"
	return result, nil
}

func finalizeProbeResult(result *GroupStatusProbeResult, cfg *GroupStatusConfig) {
	if result == nil {
		return
	}
	result.ResponseExcerpt = truncateProbeText(result.ResponseExcerpt)
	result.ErrorDetail = truncateProbeText(redactProbeUpstreamAddresses(result.ErrorDetail))
	if result.Status == "" {
		result.Status = GroupRuntimeStatusDown
	}
	if result.SubStatus == "" {
		if result.Status == GroupRuntimeStatusUp {
			result.SubStatus = "ok"
		} else {
			result.SubStatus = "failed"
		}
	}
	if result.Status == GroupRuntimeStatusUp && result.LatencyMS != nil && cfg != nil && *result.LatencyMS > cfg.SlowLatencyMS {
		result.Status = GroupRuntimeStatusDegraded
		result.SubStatus = "slow"
	}
}

func inferProbeSubStatus(httpCode *int, err error) string {
	if httpCode != nil {
		switch {
		case *httpCode == http.StatusTooManyRequests:
			return "http_429"
		case *httpCode >= 500:
			return "http_5xx"
		case *httpCode >= 400:
			return "http_error"
		}
	}
	if err == nil {
		return "failed"
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "timeout"
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return "timeout"
	}
	return "network_error"
}

var probeUpstreamURLPattern = regexp.MustCompile(`(?i)\bhttps?://[^\s"'<>]+`)

// redactProbeUpstreamAddresses 把文本中的 URL 全部替换成占位符。error_detail 会展示在
// 用户可见的模型状态页，上游 base_url 属于内部配置，不能出现在里面；中转服务的错误
// 报文里也经常回显自家地址，一并覆盖。
func redactProbeUpstreamAddresses(text string) string {
	if text == "" {
		return text
	}
	return strings.TrimSpace(probeUpstreamURLPattern.ReplaceAllString(text, "[upstream]"))
}

// sanitizeProbeErrorDetail 生成不含上游地址的探测失败文案。http.Client 返回的
// *url.Error 会把完整上游 URL 拼进 Error()（如 `Post "https://host/path": context
// deadline exceeded`），DNS/dial 错误还会带上主机名或 IP，这里逐层剥掉，只保留真正的
// 失败原因；原始错误仍会完整打到日志里（见 executeProbe 的 LegacyPrintf）。
func sanitizeProbeErrorDetail(err error) string {
	if err == nil {
		return ""
	}
	msg := err.Error()
	var urlErr *url.Error
	if errors.As(err, &urlErr) && urlErr.Err != nil {
		msg = strings.Replace(msg, urlErr.Error(), sanitizeProbeNetErrorMessage(urlErr.Err), 1)
	}
	return redactProbeUpstreamAddresses(msg)
}

func sanitizeProbeNetErrorMessage(err error) string {
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		// 不带 dnsErr.Name，避免泄露上游主机名。
		reason := strings.TrimSpace(dnsErr.Err)
		if reason == "" {
			reason = "lookup failed"
		}
		return "dns lookup failed: " + reason
	}
	var opErr *net.OpError
	if errors.As(err, &opErr) && opErr.Err != nil {
		// net.OpError.Error() 会拼上 dial 目标地址，只保留操作名和底层原因。
		return strings.TrimSpace(opErr.Op + " failed: " + opErr.Err.Error())
	}
	return err.Error()
}

func truncateProbeText(text string) string {
	trimmed := strings.TrimSpace(text)
	if len(trimmed) <= 500 {
		return trimmed
	}
	return trimmed[:500]
}

func (s *GroupStatusProbeService) probeAnthropic(ctx context.Context, account *Account, cfg *GroupStatusConfig) (string, *int, error) {
	if s.accountTestSvc == nil {
		return "", nil, errors.New("account test service is not configured")
	}
	if account.IsBedrock() {
		return s.probeBedrock(ctx, account, cfg)
	}

	testModelID := cfg.ProbeModel
	if account.Type == AccountTypeAPIKey {
		testModelID = account.GetMappedModel(testModelID)
	}

	var authToken string
	var useBearer bool
	var apiURL string

	if account.IsOAuth() {
		useBearer = true
		apiURL = testClaudeAPIURL
		authToken = account.GetCredential("access_token")
		if authToken == "" {
			return "", nil, errors.New("no access token available")
		}
	} else if account.Type == AccountTypeAPIKey {
		authToken = account.GetCredential("api_key")
		if authToken == "" {
			return "", nil, errors.New("no API key available")
		}
		baseURL := account.GetBaseURL()
		normalizedBaseURL, err := s.accountTestSvc.validateUpstreamBaseURL(baseURL)
		if err != nil {
			return "", nil, fmt.Errorf("invalid base URL: %w", err)
		}
		apiURL = strings.TrimSuffix(normalizedBaseURL, "/") + "/v1/messages?beta=true"
	} else {
		return "", nil, fmt.Errorf("unsupported account type: %s", account.Type)
	}

	payload, err := createAnthropicProbePayload(testModelID, cfg.ProbePrompt)
	if err != nil {
		return "", nil, err
	}
	payloadBytes, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(payloadBytes))
	if err != nil {
		return "", nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("accept", "text/event-stream")
	if useBearer {
		req.Header.Set("Authorization", "Bearer "+authToken)
	} else {
		req.Header.Set("x-api-key", authToken)
	}

	return s.executeStreamingProbe(req, account, parseClaudeProbeStream)
}

func (s *GroupStatusProbeService) probeBedrock(ctx context.Context, account *Account, cfg *GroupStatusConfig) (string, *int, error) {
	if s.accountTestSvc == nil {
		return "", nil, errors.New("account test service is not configured")
	}
	region := bedrockRuntimeRegion(account)
	resolvedModelID, ok := ResolveBedrockModelID(account, cfg.ProbeModel)
	if !ok {
		return "", nil, fmt.Errorf("unsupported Bedrock model: %s", cfg.ProbeModel)
	}
	bodyBytes, _ := json.Marshal(map[string]any{
		"anthropic_version": "bedrock-2023-05-31",
		"messages": []map[string]any{
			{
				"role": "user",
				"content": []map[string]any{
					{"type": "text", "text": strings.TrimSpace(cfg.ProbePrompt)},
				},
			},
		},
		"max_tokens":  64,
		"temperature": 0,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, BuildBedrockURL(region, resolvedModelID, false), bytes.NewReader(bodyBytes))
	if err != nil {
		return "", nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if account.IsBedrockAPIKey() {
		apiKey := account.GetCredential("api_key")
		if apiKey == "" {
			return "", nil, errors.New("no API key available")
		}
		req.Header.Set("Authorization", "Bearer "+apiKey)
	} else {
		signer, err := NewBedrockSignerFromAccount(account)
		if err != nil {
			return "", nil, err
		}
		if err := signer.SignRequest(ctx, req, bodyBytes); err != nil {
			return "", nil, err
		}
	}
	text, code, err := s.executeJSONProbe(req, account, func(body []byte) (string, error) {
		var result struct {
			Content []struct {
				Text string `json:"text"`
			} `json:"content"`
		}
		if err := json.Unmarshal(body, &result); err != nil {
			return "", err
		}
		if len(result.Content) == 0 {
			return "", nil
		}
		return result.Content[0].Text, nil
	})
	return text, code, err
}

func (s *GroupStatusProbeService) probeOpenAI(ctx context.Context, account *Account, cfg *GroupStatusConfig) (string, *int, error) {
	if s.accountTestSvc == nil {
		return "", nil, errors.New("account test service is not configured")
	}

	testModelID := cfg.ProbeModel
	if account.Type == AccountTypeAPIKey {
		if mapping := account.GetModelMapping(); len(mapping) > 0 {
			if mapped, ok := mapping[testModelID]; ok {
				testModelID = mapped
			}
		}
	}

	var authToken string
	var apiURL string
	var isOAuth bool
	var chatgptAccountID string
	if account.IsOAuth() {
		isOAuth = true
		authToken = account.GetOpenAIAccessToken()
		if authToken == "" {
			return "", nil, errors.New("no access token available")
		}
		apiURL = chatgptCodexAPIURL
		chatgptAccountID = account.GetChatGPTAccountID()
	} else if account.Type == AccountTypeAPIKey {
		authToken = account.GetOpenAIApiKey()
		if authToken == "" {
			return "", nil, errors.New("no API key available")
		}
		normalizedBaseURL, err := s.accountTestSvc.validateUpstreamBaseURL(account.GetOpenAIBaseURL())
		if err != nil {
			return "", nil, fmt.Errorf("invalid base URL: %w", err)
		}
		apiURL = strings.TrimSuffix(normalizedBaseURL, "/") + "/responses"
	} else {
		return "", nil, fmt.Errorf("unsupported account type: %s", account.Type)
	}

	payloadBytes, _ := json.Marshal(createOpenAIProbePayload(testModelID, strings.TrimSpace(cfg.ProbePrompt), isOAuth))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(payloadBytes))
	if err != nil {
		return "", nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+authToken)
	req.Header.Set("accept", "text/event-stream")
	if isOAuth {
		req.Host = "chatgpt.com"
		if chatgptAccountID != "" {
			req.Header.Set("chatgpt-account-id", chatgptAccountID)
		}
	}

	return s.executeStreamingProbe(req, account, parseOpenAIProbeStream)
}

func (s *GroupStatusProbeService) probeGemini(ctx context.Context, account *Account, cfg *GroupStatusConfig) (string, *int, error) {
	if s.accountTestSvc == nil {
		return "", nil, errors.New("account test service is not configured")
	}
	testModelID := cfg.ProbeModel
	if account.Type == AccountTypeAPIKey {
		if mapping := account.GetModelMapping(); len(mapping) > 0 {
			if mapped, ok := mapping[testModelID]; ok {
				testModelID = mapped
			}
		}
	}
	payload := createGeminiTestPayload(testModelID, cfg.ProbePrompt)
	var req *http.Request
	var err error
	switch account.Type {
	case AccountTypeAPIKey:
		req, err = s.accountTestSvc.buildGeminiAPIKeyRequest(ctx, account, testModelID, payload)
	case AccountTypeOAuth:
		req, err = s.accountTestSvc.buildGeminiOAuthRequest(ctx, account, testModelID, payload)
	default:
		return "", nil, fmt.Errorf("unsupported account type: %s", account.Type)
	}
	if err != nil {
		return "", nil, err
	}
	return s.executeStreamingProbe(req, account, parseGeminiProbeStream)
}

// probeGrok 复用网关转发 Grok Responses 的请求构造（base_url 解析、CLI 身份头、
// 模型别名归一），而不是落到 probeAnthropic —— 后者会把 Grok 的 OAuth token 发到
// api.anthropic.com，或者用 x-api-key 打到错误的 /v1/messages 路径，探测必然失败。
func (s *GroupStatusProbeService) probeGrok(ctx context.Context, account *Account, cfg *GroupStatusConfig) (string, *int, error) {
	if s.accountTestSvc == nil {
		return "", nil, errors.New("account test service is not configured")
	}

	upstreamModel := strings.TrimSpace(account.GetMappedModel(cfg.ProbeModel))
	if upstreamModel == "" {
		upstreamModel = grokDefaultResponsesModel
	}
	upstreamModel = xai.ResolveGrokTextResponsesModelID(upstreamModel, grokDefaultResponsesModel)
	if isGrokImageGenerationModel(upstreamModel) || isGrokVideoGenerationModel(upstreamModel) {
		return "", nil, fmt.Errorf("probe model %s is a media model and is not available on the Responses endpoint", upstreamModel)
	}

	token, err := s.grokProbeAccessToken(ctx, account)
	if err != nil {
		return "", nil, err
	}

	payloadBytes, err := createGrokProbePayload(upstreamModel, cfg.ProbePrompt)
	if err != nil {
		return "", nil, err
	}
	// cacheIdentity 留空：探测不参与 prompt cache 分桶，也就不该污染真实会话的缓存键。
	req, err := buildGrokResponsesRequest(ctx, nil, account, payloadBytes, token, "", s.accountTestSvc.cfg, s.accountTestSvc.settingService)
	if err != nil {
		return "", nil, err
	}
	return s.executeStreamingProbe(req, account, parseOpenAIProbeStream)
}

// grokProbeAccessToken 取生产调度使用的凭据（不是管理员手测那条会跳过可调度性检查的
// 路径），这样探测结果才代表用户真实调用时的可用性。
func (s *GroupStatusProbeService) grokProbeAccessToken(ctx context.Context, account *Account) (string, error) {
	switch account.Type {
	case AccountTypeOAuth:
		if s.accountTestSvc.grokTokenProvider == nil {
			return "", errors.New("grok token provider is not configured")
		}
		token, err := s.accountTestSvc.grokTokenProvider.GetAccessToken(ctx, account)
		if err != nil {
			return "", fmt.Errorf("failed to get grok access token: %w", err)
		}
		return token, nil
	case AccountTypeAPIKey:
		token := strings.TrimSpace(account.GetCredential("api_key"))
		if token == "" {
			return "", errors.New("no API key available")
		}
		return token, nil
	default:
		return "", fmt.Errorf("unsupported account type: %s", account.Type)
	}
}

func createGrokProbePayload(modelID, prompt string) ([]byte, error) {
	textPrompt := strings.TrimSpace(prompt)
	if textPrompt == "" {
		textPrompt = "Please reply with ONLINE."
	}
	return json.Marshal(map[string]any{
		"model":  modelID,
		"input":  textPrompt,
		"stream": true,
	})
}

func (s *GroupStatusProbeService) probeAntigravity(ctx context.Context, account *Account, cfg *GroupStatusConfig) (string, *int, error) {
	if account.Type == AccountTypeAPIKey {
		if strings.HasPrefix(cfg.ProbeModel, "gemini-") {
			return s.probeGemini(ctx, account, cfg)
		}
		return s.probeAnthropic(ctx, account, cfg)
	}
	if s.accountTestSvc == nil || s.accountTestSvc.antigravityGatewayService == nil {
		return "", nil, errors.New("antigravity gateway service not configured")
	}
	res, err := s.accountTestSvc.antigravityGatewayService.TestConnection(ctx, account, cfg.ProbeModel)
	if err != nil {
		return "", nil, err
	}
	code := http.StatusOK
	return res.Text, &code, nil
}

// probeUpstreamError 是探测遇到上游 HTTP 错误时的错误类型。
//
// Error() 返回的是调用方真实会收到的文案（见 probeUpstreamClientMessage），因为它
// 会被写进 error_detail 并展示在用户可见的模型状态页；原始响应体单独留在 Body 上，
// 只用于与请求路径一致的 failover 分类，不外泄给用户。
type probeUpstreamError struct {
	StatusCode int
	Body       []byte
	Message    string
}

func (e *probeUpstreamError) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

func newProbeUpstreamError(account *Account, statusCode int, body []byte) *probeUpstreamError {
	return &probeUpstreamError{
		StatusCode: statusCode,
		Body:       append([]byte(nil), body...),
		Message:    fmt.Sprintf("http %d: %s", statusCode, probeUpstreamClientMessage(account, statusCode, body)),
	}
}

func probeUpstreamErrorBody(err error) []byte {
	var upstreamErr *probeUpstreamError
	if errors.As(err, &upstreamErr) && upstreamErr != nil {
		return upstreamErr.Body
	}
	return nil
}

// probeUpstreamClientMessage 复用网关的下游错误文案，让探测记录与用户实际调用收到的
// 提示一致：确定性的请求错误（400）保留上游 message，其余状态码使用网关那张固定文案
// 表，而不是把上游原始报文糊到状态页上。
func probeUpstreamClientMessage(account *Account, statusCode int, body []byte) string {
	platform := ""
	if account != nil {
		platform = account.Platform
	}
	if platform == PlatformGrok && isGrokContentPolicyRejection(statusCode, body) {
		return grokContentPolicyClientMessage(body)
	}
	if isOpenAIDeterministicClientError(statusCode) {
		if msg := sanitizeUpstreamErrorMessage(strings.TrimSpace(extractUpstreamErrorMessage(body))); msg != "" {
			return msg
		}
		if isOpenAIGatewayPlatform(platform) {
			return openAIUpstreamClientErrorFallbackMessage
		}
	}
	if isOpenAIGatewayPlatform(platform) {
		_, _, msg := openAIUpstreamClientError(statusCode)
		return msg
	}
	_, _, msg := anthropicUpstreamClientError(statusCode)
	return msg
}

func (s *GroupStatusProbeService) executeStreamingProbe(req *http.Request, account *Account, parser func(io.Reader) (string, error)) (string, *int, error) {
	resp, err := s.doHTTPRequest(req, account)
	if err != nil {
		return "", nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	code := resp.StatusCode
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return string(body), &code, newProbeUpstreamError(account, code, body)
	}
	text, err := parser(resp.Body)
	return text, &code, err
}

func (s *GroupStatusProbeService) executeJSONProbe(req *http.Request, account *Account, parser func([]byte) (string, error)) (string, *int, error) {
	resp, err := s.doHTTPRequest(req, account)
	if err != nil {
		return "", nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	code := resp.StatusCode
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return string(body), &code, newProbeUpstreamError(account, code, body)
	}
	text, err := parser(body)
	return text, &code, err
}

func (s *GroupStatusProbeService) doHTTPRequest(req *http.Request, account *Account) (*http.Response, error) {
	if s.accountTestSvc == nil || s.accountTestSvc.httpUpstream == nil {
		return nil, errors.New("http upstream is not configured")
	}
	proxyURL := ""
	if account.ProxyID != nil && account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}
	return s.accountTestSvc.httpUpstream.DoWithTLS(req, proxyURL, account.ID, account.Concurrency, s.accountTestSvc.tlsFPProfileService.ResolveTLSProfile(account))
}

func createAnthropicProbePayload(modelID, prompt string) (map[string]any, error) {
	sessionID, err := generateSessionString()
	if err != nil {
		return nil, err
	}
	textPrompt := strings.TrimSpace(prompt)
	if textPrompt == "" {
		textPrompt = "Please reply with ONLINE."
	}
	return map[string]any{
		"model": modelID,
		"messages": []map[string]any{
			{
				"role": "user",
				"content": []map[string]any{
					{
						"type": "text",
						"text": textPrompt,
						"cache_control": map[string]string{
							"type": "ephemeral",
						},
					},
				},
			},
		},
		"system": []map[string]any{
			{
				"type": "text",
				"text": claudeCodeSystemPrompt,
				"cache_control": map[string]string{
					"type": "ephemeral",
				},
			},
		},
		"metadata": map[string]string{
			"user_id": sessionID,
		},
		"max_tokens":  64,
		"temperature": 0,
		"stream":      true,
	}, nil
}

func createOpenAIProbePayload(modelID, prompt string, isOAuth bool) map[string]any {
	textPrompt := strings.TrimSpace(prompt)
	if textPrompt == "" {
		textPrompt = "Please reply with ONLINE."
	}
	payload := map[string]any{
		"model": modelID,
		"input": []map[string]any{
			{
				"role": "user",
				"content": []map[string]any{
					{
						"type": "input_text",
						"text": textPrompt,
					},
				},
			},
		},
		"stream":       true,
		"instructions": openai.DefaultInstructions,
	}
	if isOAuth {
		payload["store"] = false
	}
	return payload
}

func parseClaudeProbeStream(body io.Reader) (string, error) {
	reader := bufio.NewReader(body)
	var parts []string
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				return strings.Join(parts, ""), nil
			}
			return "", err
		}
		line = strings.TrimSpace(line)
		if line == "" || !sseDataPrefix.MatchString(line) {
			continue
		}
		jsonStr := sseDataPrefix.ReplaceAllString(line, "")
		if jsonStr == "[DONE]" {
			return strings.Join(parts, ""), nil
		}
		var data map[string]any
		if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
			continue
		}
		eventType, _ := data["type"].(string)
		switch eventType {
		case "content_block_delta":
			if delta, ok := data["delta"].(map[string]any); ok {
				if text, ok := delta["text"].(string); ok && text != "" {
					parts = append(parts, text)
				}
			}
		case "message_stop":
			return strings.Join(parts, ""), nil
		case "error":
			if errData, ok := data["error"].(map[string]any); ok {
				if msg, ok := errData["message"].(string); ok {
					return strings.Join(parts, ""), errors.New(msg)
				}
			}
			return strings.Join(parts, ""), errors.New("anthropic probe failed")
		}
	}
}

func parseOpenAIProbeStream(body io.Reader) (string, error) {
	reader := bufio.NewReader(body)
	var parts []string
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				return strings.Join(parts, ""), nil
			}
			return "", err
		}
		line = strings.TrimSpace(line)
		if line == "" || !sseDataPrefix.MatchString(line) {
			continue
		}
		jsonStr := sseDataPrefix.ReplaceAllString(line, "")
		if jsonStr == "[DONE]" {
			return strings.Join(parts, ""), nil
		}
		var data map[string]any
		if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
			continue
		}
		switch data["type"] {
		case "response.output_text.delta":
			if delta, ok := data["delta"].(string); ok && delta != "" {
				parts = append(parts, delta)
			}
		case "response.completed":
			return strings.Join(parts, ""), nil
		case "error":
			if errData, ok := data["error"].(map[string]any); ok {
				if msg, ok := errData["message"].(string); ok {
					return strings.Join(parts, ""), errors.New(msg)
				}
			}
			return strings.Join(parts, ""), errors.New("openai probe failed")
		}
	}
}

func parseGeminiProbeStream(body io.Reader) (string, error) {
	reader := bufio.NewReader(body)
	var parts []string
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				return strings.Join(parts, ""), nil
			}
			return "", err
		}
		line = strings.TrimSpace(line)
		if line == "" || !strings.HasPrefix(line, "data: ") {
			continue
		}
		jsonStr := strings.TrimPrefix(line, "data: ")
		if jsonStr == "[DONE]" {
			return strings.Join(parts, ""), nil
		}
		var data map[string]any
		if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
			continue
		}
		if resp, ok := data["response"].(map[string]any); ok && resp != nil {
			data = resp
		}
		if candidates, ok := data["candidates"].([]any); ok && len(candidates) > 0 {
			if candidate, ok := candidates[0].(map[string]any); ok {
				if content, ok := candidate["content"].(map[string]any); ok {
					if partsAny, ok := content["parts"].([]any); ok {
						for _, part := range partsAny {
							if partMap, ok := part.(map[string]any); ok {
								if text, ok := partMap["text"].(string); ok && text != "" {
									parts = append(parts, text)
								}
							}
						}
					}
				}
				if finishReason, ok := candidate["finishReason"].(string); ok && finishReason != "" {
					return strings.Join(parts, ""), nil
				}
			}
		}
		if errData, ok := data["error"].(map[string]any); ok {
			if msg, ok := errData["message"].(string); ok {
				return strings.Join(parts, ""), errors.New(msg)
			}
			return strings.Join(parts, ""), errors.New("gemini probe failed")
		}
	}
}
