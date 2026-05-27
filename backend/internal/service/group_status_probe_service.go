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
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/pkg/openai"
)

type GroupStatusProbeService struct {
	repo             GroupStatusRepository
	groupRepo        GroupRepository
	scheduler        *SchedulerSnapshotService
	accountTestSvc   *AccountTestService
	gatewaySvc       *GatewayService
	openAIGatewaySvc *OpenAIGatewayService
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

		if !s.shouldProbeFailover(account, rawResult) {
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
	if group.Platform == PlatformOpenAI && s.openAIGatewaySvc != nil {
		selection, err := s.openAIGatewaySvc.SelectAccountWithLoadAwareness(ctx, &groupID, "", cfg.ProbeModel, excludedIDs)
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
	if group != nil && group.Platform == PlatformOpenAI && s.openAIGatewaySvc != nil && s.openAIGatewaySvc.cfg != nil && s.openAIGatewaySvc.cfg.Gateway.MaxAccountSwitches > 0 {
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
	return &GroupStatusProbeExecution{
		Group:   group,
		Config:  cfg,
		Account: account,
		Result:  result,
		State:   state,
		Event:   event,
	}, nil
}

func (s *GroupStatusProbeService) shouldProbeFailover(account *Account, result *GroupStatusProbeResult) bool {
	if result == nil || result.HTTPCode == nil {
		return false
	}
	statusCode := *result.HTTPCode
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

	switch {
	case account.Platform == PlatformOpenAI:
		responseText, httpCode, err = s.probeOpenAI(ctx, account, cfg)
	case account.Platform == PlatformGemini:
		responseText, httpCode, err = s.probeGemini(ctx, account, cfg)
	case account.Platform == PlatformAntigravity:
		responseText, httpCode, err = s.probeAntigravity(ctx, account, cfg)
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
		result.ErrorDetail = truncateProbeText(err.Error())
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
	result.ErrorDetail = truncateProbeText(result.ErrorDetail)
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

func (s *GroupStatusProbeService) executeStreamingProbe(req *http.Request, account *Account, parser func(io.Reader) (string, error)) (string, *int, error) {
	resp, err := s.doHTTPRequest(req, account)
	if err != nil {
		return "", nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	code := resp.StatusCode
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return string(body), &code, fmt.Errorf("http %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
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
		return string(body), &code, fmt.Errorf("http %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
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
