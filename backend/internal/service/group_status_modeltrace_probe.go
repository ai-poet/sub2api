package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/pkg/openai"
)

// ModelTrace 指纹验证的执行（本 fork 自有功能）。
//
// 一次运行 = 选一个与分组平台一致的账号 → 发 3 条（不够时补到最多 6 条）数字挑战 → 本地归因 →
// 按 Guard 规则判定 → 落库 → 稳定结论切换时推送。与存活探测、Astra 互不影响。
//
// 请求刻意贴近指纹库的采集环境：不带 temperature / reasoning / thinking / max_output_tokens，
// Claude 的 max_tokens 为 4096；OAuth 账号只带各自通道硬性要求的 instructions / system。

const (
	groupStatusModelTraceLogComponent = "service.group_status_modeltrace"
	modelTraceExcerptMaxRunes         = 160

	ModelTraceRejectionTooFewNumbers    = "too_few_numbers"
	ModelTraceRejectionMaxTokens        = "max_tokens"
	ModelTraceRejectionRefusal          = "refusal"
	ModelTraceRejectionStreamIncomplete = "stream_incomplete"
	ModelTraceRejectionStreamError      = "stream_error"
)

// modelTraceResponse 是一次挑战请求的解析结果（两种通道统一成同一形状）。
type modelTraceResponse struct {
	Text       string
	Usage      openAIProbeUsage
	StopReason string
	Thinking   bool
	Completed  bool
}

func (s *GroupStatusProbeService) modelTraceBank() (*ModelTraceBank, *ModelTraceBankMeta, error) {
	if s != nil && s.modelTraceBanks != nil {
		return s.modelTraceBanks.Active()
	}
	return LoadEmbeddedModelTraceBank()
}

// SetModelTraceBankProvider 替换指纹库来源（测试用）。
func (s *GroupStatusProbeService) SetModelTraceBankProvider(p modelTraceBankProvider) {
	if s == nil {
		return
	}
	s.modelTraceBanks = p
}

func (s *GroupStatusProbeService) modelTraceWorkers() int {
	if s != nil && s.modelTraceConcurrency > 0 {
		return s.modelTraceConcurrency
	}
	return groupStatusModelTraceDefaultConcurrency
}

func (s *GroupStatusProbeService) newModelTraceChallenges(n int) []ModelTraceChallenge {
	if s != nil && s.modelTraceChallenges != nil {
		return s.modelTraceChallenges(n)
	}
	return GenerateModelTraceChallenges(newModelTraceRNG(), n)
}

// modelTraceSleepFor 重试退避；测试可通过 modelTraceSleep 字段替换为不等待。
func (s *GroupStatusProbeService) modelTraceSleepFor(ctx context.Context, d time.Duration) error {
	if s != nil && s.modelTraceSleep != nil {
		return s.modelTraceSleep(ctx, d)
	}
	return serverChanSleep(ctx, d)
}

// IsModelTraceRunning 报告该分组是否有正在进行的 ModelTrace 指纹验证。
func (s *GroupStatusProbeService) IsModelTraceRunning(groupID int64) bool {
	if s == nil {
		return false
	}
	_, ok := s.modelTraceRunning.Load(groupID)
	return ok
}

func (s *GroupStatusProbeService) markModelTraceRunning(groupID int64) bool {
	_, loaded := s.modelTraceRunning.LoadOrStore(groupID, time.Now())
	return !loaded
}

func (s *GroupStatusProbeService) clearModelTraceRunning(groupID int64) {
	s.modelTraceRunning.Delete(groupID)
}

// StartModelTraceAsync 在后台启动一次验证（管理端「立即检测」用）；同一分组运行中则报错。
func (s *GroupStatusProbeService) StartModelTraceAsync(groupID int64) error {
	if s == nil {
		return errors.New("group status probe service is not configured")
	}
	if s.IsModelTraceRunning(groupID) {
		return ErrGroupStatusModelTraceRunning
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), groupStatusModelTraceRunBudget)
		defer cancel()
		if _, err := s.ProbeModelTraceGroupNow(ctx, groupID); err != nil {
			logger.LegacyPrintf(groupStatusModelTraceLogComponent, "[ModelTrace] group=%d manual run failed: %v", groupID, err)
		}
	}()
	return nil
}

// ProbeModelTraceGroupNow 立即验证；不要求分组已开启 modeltrace_enabled。
func (s *GroupStatusProbeService) ProbeModelTraceGroupNow(ctx context.Context, groupID int64) (*GroupStatusModelTraceExecution, error) {
	group, cfg, err := s.ensureProbeTarget(ctx, groupID)
	if err != nil {
		return nil, err
	}
	return s.probeModelTrace(ctx, group, cfg)
}

// ProbeModelTraceWithConfig 供定时 runner 调用。
func (s *GroupStatusProbeService) ProbeModelTraceWithConfig(ctx context.Context, cfg *GroupStatusConfig) (*GroupStatusModelTraceExecution, error) {
	if cfg == nil {
		return nil, ErrGroupStatusInvalidConfig
	}
	group, err := s.groupRepo.GetByID(ctx, cfg.GroupID)
	if err != nil {
		return nil, err
	}
	return s.probeModelTrace(ctx, group, cfg)
}

func (s *GroupStatusProbeService) probeModelTrace(ctx context.Context, group *Group, cfg *GroupStatusConfig) (*GroupStatusModelTraceExecution, error) {
	if group == nil || cfg == nil {
		return nil, ErrGroupStatusInvalidConfig
	}
	if !modelTraceSupportsPlatform(group.Platform) {
		return nil, ErrGroupStatusModelTraceUnsupported
	}
	if err := ValidateGroupStatusConfig(cfg); err != nil {
		return nil, err
	}
	if !s.markModelTraceRunning(group.ID) {
		return nil, ErrGroupStatusModelTraceRunning
	}
	defer s.clearModelTraceRunning(group.ID)
	defer s.clearModelTraceProgress(group.ID)

	bank, meta, bankErr := s.modelTraceBank()
	progress := s.beginModelTraceProgress(group.ID, 1)
	account, result := s.executeModelTraceRun(ctx, group, cfg, bank, meta, bankErr, progress, 1, nil)
	logModelTraceRun(group, account, result)
	execution, err := s.saveModelTraceExecution(ctx, group, cfg, account, result)
	if err != nil {
		return nil, err
	}

	// 首次出现「强指向其他模型」时，在同一账号上立即复测一轮确认：问的是「这个上游是不是冒充的」，
	// 换号复测会让 N 个账号里只有 1 个假的分组永远凑不满连续两次。
	if execution.State != nil &&
		result.Verdict == ModelTraceVerdictMismatch &&
		execution.State.ModelTraceConsecutiveMismatch == 1 &&
		execution.State.ModelTraceStableStatus != ModelTraceStatusMismatch &&
		ctx.Err() == nil {
		confirmProgress := s.beginModelTraceProgress(group.ID, 2)
		confirmAccount, confirmResult := s.executeModelTraceRun(ctx, group, cfg, bank, meta, bankErr, confirmProgress, 2, account)
		logModelTraceRun(group, confirmAccount, confirmResult)
		confirmed, err := s.saveModelTraceExecution(ctx, group, cfg, confirmAccount, confirmResult)
		if err != nil {
			return nil, err
		}
		confirmed.Confirmed = true
		return confirmed, nil
	}
	return execution, nil
}

// logModelTraceRun 每轮结束记一行摘要，方便在服务端日志里定位慢、失败和判定。
func logModelTraceRun(group *Group, account *Account, result *GroupStatusModelTraceResult) {
	if group == nil || result == nil {
		return
	}
	accountID := int64(0)
	if account != nil {
		accountID = account.ID
	}
	latency := int64(0)
	if result.LatencyMS != nil {
		latency = *result.LatencyMS
	}
	logger.LegacyPrintf(groupStatusModelTraceLogComponent,
		"[ModelTrace] group=%d round=%d expected=%s verdict=%s top=%s(%.3f) valid=%d attempts=%d/%d account=%d latency=%dms reasons=%s detail=%q",
		group.ID, result.Round, result.ExpectedModel, result.Verdict, result.TopModel, result.TopProbability,
		result.ValidOutputs, result.AttemptsMade, result.AttemptsPlanned, accountID, latency,
		strings.Join(result.Reasons, ","), result.ErrorDetail)
}

// modelTraceAccountCompatible 报告账号能否承载本平台的 ModelTrace 请求。
func modelTraceAccountCompatible(group *Group, account *Account) bool {
	if group == nil || account == nil {
		return false
	}
	if !account.IsOAuth() && account.Type != AccountTypeAPIKey {
		return false
	}
	switch group.Platform {
	case PlatformOpenAI:
		return account.Platform == PlatformOpenAI
	case PlatformAnthropic:
		return account.Platform == PlatformAnthropic && !account.IsBedrock()
	default:
		return false
	}
}

// executeModelTraceRun 选账号、发挑战、归因判定；永远返回一个结果，不返回 error。
// pinned 非空时先用该账号（复测确认），不可用再回到正常调度。
func (s *GroupStatusProbeService) executeModelTraceRun(
	ctx context.Context,
	group *Group,
	cfg *GroupStatusConfig,
	bank *ModelTraceBank,
	meta *ModelTraceBankMeta,
	bankErr error,
	progress *modelTraceProgressTracker,
	round int,
	pinned *Account,
) (*Account, *GroupStatusModelTraceResult) {
	startedAt := time.Now()
	expected := strings.TrimSpace(cfg.ModelTraceExpectedModel)
	if expected == "" {
		expected = modelTraceDefaultExpected(group.Platform)
	}
	requestModel := strings.TrimSpace(cfg.ModelTraceRequestModel)
	if requestModel == "" {
		requestModel = expected
	}
	result := &GroupStatusModelTraceResult{
		GroupID:       group.ID,
		ConfigID:      cfg.ID,
		Platform:      group.Platform,
		ExpectedModel: expected,
		RequestModel:  requestModel,
		Round:         round,
		Verdict:       ModelTraceVerdictInconclusive,
		Outcome:       ModelTraceOutcomeInconclusive,
		StartedAt:     startedAt,
	}
	if meta != nil {
		result.BankSHA256 = meta.SHA256
		result.BankBuiltAt = meta.BuiltAt
	}

	finish := func(account *Account, records []ModelTraceOutputRecord, extraDetail string, reasons ...string) (*Account, *GroupStatusModelTraceResult) {
		progress.setPhase(ModelTracePhaseScoring)
		result.FinishedAt = time.Now()
		latency := result.FinishedAt.Sub(startedAt).Milliseconds()
		result.LatencyMS = &latency
		if account != nil {
			id := account.ID
			result.AccountID = &id
			result.AccountType = account.Type
		}
		sort.SliceStable(records, func(i, j int) bool { return records[i].Seq < records[j].Seq })
		result.Outputs = records
		var (
			valid         [][]int
			lastHTTPCode  *int
			lastFailure   string
			usedSequences = map[int]struct{}{}
		)
		for _, rec := range records {
			usedSequences[rec.Seq] = struct{}{}
			result.InputTokens += rec.InputTokens
			result.OutputTokens += rec.OutputTokens
			result.ReasoningTokens += rec.ReasoningTokens
			if rec.Accepted {
				valid = append(valid, rec.Numbers)
				continue
			}
			if rec.transportFailed {
				lastHTTPCode = rec.HTTPCode
				lastFailure = rec.Error
			}
		}
		result.AttemptsMade = len(usedSequences)
		result.ValidOutputs = len(valid)
		if len(valid) == 0 && lastHTTPCode != nil {
			result.HTTPCode = lastHTTPCode
		}

		switch {
		case len(reasons) > 0:
			result.Reasons = reasons
		case len(valid) == 0:
			result.Reasons = []string{ModelTraceReasonNoValidOutputs}
		default:
			analysis, err := bank.AnalyzeNumbers(valid)
			if err != nil {
				result.Reasons = []string{ModelTraceReasonNoValidOutputs}
				break
			}
			result.Ranking = analysis.Results
			result.FamilyProbabilities = analysis.Families
			result.CalibrationQueries = analysis.CalibrationKey
			result.Beta = analysis.Beta
			if top := analysis.Top(); top != nil {
				result.TopModel = top.Model
				result.TopProbability = top.Probability
			}
			if candidate := analysis.Find(expected); candidate != nil {
				p := candidate.Probability
				result.ExpectedProbability = &p
			}
			result.Verdict, result.Outcome, result.Reasons = ClassifyModelTrace(analysis, bank, expected)
		}
		result.CostUSD = s.estimateModelTraceCostUSD(expected, requestModel, result.InputTokens, result.OutputTokens)
		detail := mergeProbeErrorDetails(extraDetail, lastFailure)
		result.ErrorDetail = truncateProbeText(redactProbeUpstreamAddresses(detail))
		return account, result
	}

	switch {
	case bankErr != nil || bank == nil:
		detail := "fingerprint bank unavailable"
		if bankErr != nil {
			detail = bankErr.Error()
		}
		return finish(nil, nil, detail, ModelTraceReasonBankInvalid)
	case !modelTraceTargetAllowed(group.Platform, expected):
		return finish(nil, nil, "expected model "+expected+" is not a target for "+group.Platform+" groups", ModelTraceReasonTargetNotAllowed)
	case !bank.HasModel(expected):
		return finish(nil, nil, "expected model "+expected+" is not in the fingerprint bank", ModelTraceReasonUnknownExpected)
	}

	challenges := s.newModelTraceChallenges(groupStatusModelTraceMaxChallenges)
	if len(challenges) == 0 {
		return finish(nil, nil, "no challenges generated", ModelTraceReasonNoValidOutputs)
	}
	result.AttemptsPlanned = len(challenges)

	// 让调度器按请求模型过滤账号（模型支持 / 模型级限流），其余配置照抄
	probeCfg := *cfg
	probeCfg.ProbeModel = requestModel

	// 账号锁定：同步跑第一条挑战，拿到 2xx 后整轮固定在该账号
	excludedIDs := make(map[int64]struct{})
	maxAttempts := s.maxProbeAttempts(group)
	tryPinned := pinned != nil && modelTraceAccountCompatible(group, pinned)
	var (
		account            *Account
		records            []ModelTraceOutputRecord
		firstFailureDetail string
	)
	for attemptNo := 0; attemptNo < maxAttempts; attemptNo++ {
		var candidate *Account
		if tryPinned {
			tryPinned = false
			candidate = pinned
		} else {
			attempt, selectErr := s.selectProbeAttempt(ctx, group, &probeCfg, excludedIDs)
			if selectErr != nil {
				return finish(nil, records, mergeProbeErrorDetails(firstFailureDetail, "no schedulable account: "+selectErr.Error()), ModelTraceReasonNoAccount)
			}
			if attempt == nil || attempt.Account == nil {
				return finish(nil, records, mergeProbeErrorDetails(firstFailureDetail, "no schedulable account available"), ModelTraceReasonNoAccount)
			}
			candidate = attempt.Account
			if _, excluded := excludedIDs[candidate.ID]; excluded || !modelTraceAccountCompatible(group, candidate) || attempt.WaitPlan != nil {
				excludedIDs[candidate.ID] = struct{}{}
				continue
			}
		}
		progress.setAccount(candidate.ID)
		rec := s.runModelTraceChallenge(ctx, bank, candidate, requestModel, challenges[0], 1, progress)
		if rec.transportFailed {
			failure := &GroupStatusProbeResult{HTTPCode: rec.HTTPCode}
			if s.shouldProbeFailover(candidate, failure, errors.New(rec.Error)) && attemptNo < maxAttempts-1 {
				if firstFailureDetail == "" {
					firstFailureDetail = fmt.Sprintf("account %d: %s", candidate.ID, truncateProbeText(rec.Error))
				}
				// 首条挑战会在下一个账号上重跑；保留记录以便汇总 token，但不算进度
				records = append(records, rec)
				excludedIDs[candidate.ID] = struct{}{}
				progress.rollbackFailover()
				continue
			}
		}
		account = candidate
		records = append(records, rec)
		break
	}
	if account == nil {
		return finish(nil, records, mergeProbeErrorDetails(firstFailureDetail, "failover_exhausted"), ModelTraceReasonNoAccount)
	}
	progress.setPhase(ModelTracePhaseRunning)

	accepted := 0
	if records[len(records)-1].Accepted {
		accepted = 1
	}
	used := 1
	workers := s.modelTraceWorkers()
	// 上游连接数受账号并发限制，超出的请求只会在传输层排队并耗光超时
	if account.Concurrency > 0 && workers > account.Concurrency {
		workers = account.Concurrency
	}
	if workers < 1 {
		workers = 1
	}
	for accepted < groupStatusModelTraceTargetOutputs && used < len(challenges) && ctx.Err() == nil {
		wave := groupStatusModelTraceTargetOutputs - accepted
		if remaining := len(challenges) - used; wave > remaining {
			wave = remaining
		}
		if wave > workers {
			wave = workers
		}
		waveRecords := make([]ModelTraceOutputRecord, wave)
		var wg sync.WaitGroup
		for i := 0; i < wave; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				seq := used + i + 1
				waveRecords[i] = s.runModelTraceChallenge(ctx, bank, account, requestModel, challenges[used+i], seq, progress)
			}(i)
		}
		wg.Wait()
		used += wave
		for _, rec := range waveRecords {
			if rec.Accepted {
				accepted++
			}
			records = append(records, rec)
		}
	}

	extra := firstFailureDetail
	if ctx.Err() != nil {
		extra = mergeProbeErrorDetails(extra, "run interrupted: "+ctx.Err().Error())
	}
	return finish(account, records, extra)
}

// runModelTraceChallenge 执行一条挑战：传输错误、408、429、5xx 最多试 3 次；拿到 2xx 后不论
// 回答是否可用都结束（内容问题换一条新挑战，与 ModelTrace 的 API 测试一致）。
func (s *GroupStatusProbeService) runModelTraceChallenge(
	ctx context.Context,
	bank *ModelTraceBank,
	account *Account,
	requestModel string,
	challenge ModelTraceChallenge,
	seq int,
	progress *modelTraceProgressTracker,
) ModelTraceOutputRecord {
	rec := ModelTraceOutputRecord{
		Seq:            seq,
		ChallengeID:    challenge.ID,
		Prompt:         challenge.Prompt,
		ExpectedCount:  challenge.ExpectedCount,
		MinimumNumbers: ModelTraceMinimumNumbers(challenge.ExpectedCount),
	}
	defer func() { progress.record(rec) }()
	for attempt := 1; attempt <= groupStatusModelTraceMaxHTTPAttempts; attempt++ {
		if ctx.Err() != nil {
			rec.transportFailed = true
			rec.Error = ctx.Err().Error()
			return rec
		}
		rec.Attempts = attempt
		timeoutCtx, cancel := context.WithTimeout(ctx, groupStatusModelTraceRequestTimeout)
		progress.requestStarted()
		started := time.Now()
		resp, httpCode, err := s.modelTraceRequest(timeoutCtx, account, requestModel, challenge.Prompt)
		cancel()
		progress.requestFinished()
		rec.LatencyMS = time.Since(started).Milliseconds()
		rec.InputTokens += resp.Usage.InputTokens
		rec.OutputTokens += resp.Usage.OutputTokens
		rec.ReasoningTokens += resp.Usage.ReasoningTokens
		rec.HTTPCode = httpCode

		non2xx := httpCode != nil && (*httpCode < 200 || *httpCode >= 300)
		if httpCode == nil || non2xx {
			rec.transportFailed = true
			switch {
			case err != nil:
				rec.Error = sanitizeProbeErrorDetail(err)
			case httpCode != nil:
				rec.Error = fmt.Sprintf("unexpected http status: %d", *httpCode)
			default:
				rec.Error = "no response"
			}
			if attempt < groupStatusModelTraceMaxHTTPAttempts && modelTraceRetryable(httpCode) {
				if sleepErr := s.modelTraceSleepFor(ctx, time.Duration(attempt)*time.Second); sleepErr != nil {
					rec.Error = mergeProbeErrorDetails(rec.Error, sleepErr.Error())
					return rec
				}
				continue
			}
			return rec
		}

		rec.transportFailed = false
		rec.Error = ""
		rec.StopReason = resp.StopReason
		rec.ThinkingPresent = resp.Thinking
		rec.Excerpt = modelTraceExcerpt(resp.Text)
		numbers := ParseModelTraceNumbers(resp.Text)
		rec.ParsedNumbers = len(numbers)
		switch {
		case err != nil:
			rec.Rejection = ModelTraceRejectionStreamError
			rec.Error = sanitizeProbeErrorDetail(err)
		case resp.StopReason == "max_tokens":
			rec.Rejection = ModelTraceRejectionMaxTokens
		case resp.StopReason == "refusal":
			rec.Rejection = ModelTraceRejectionRefusal
		case !resp.Completed:
			rec.Rejection = ModelTraceRejectionStreamIncomplete
		case len(numbers) < rec.MinimumNumbers:
			rec.Rejection = ModelTraceRejectionTooFewNumbers
		default:
			rec.Accepted = true
			rec.Numbers = numbers
			if bank != nil {
				rec.TopModel, rec.TopProbability = bank.SingleOutputTop(bank.scoreNumbers(numbers))
			}
		}
		return rec
	}
	return rec
}

// modelTraceRetryable：传输错误（无状态码）、408、429 与 5xx 值得在同一账号上重试。
func modelTraceRetryable(httpCode *int) bool {
	if httpCode == nil {
		return true
	}
	code := *httpCode
	return code == 408 || code == 429 || code >= 500
}

func modelTraceExcerpt(text string) string {
	text = strings.TrimSpace(redactProbeUpstreamAddresses(text))
	runes := []rune(text)
	if len(runes) <= modelTraceExcerptMaxRunes {
		return text
	}
	return string(runes[:modelTraceExcerptMaxRunes]) + "…"
}

// modelTraceRequest 按账号平台发一条挑战，统一成同一种结果。
func (s *GroupStatusProbeService) modelTraceRequest(ctx context.Context, account *Account, requestModel, prompt string) (modelTraceResponse, *int, error) {
	if account != nil && account.Platform == PlatformAnthropic {
		req, err := s.buildAnthropicMessagesProbeRequest(ctx, account, requestModel, anthropicProbeHeadersClaudeCode, func(modelID string, isOAuth bool) (map[string]any, error) {
			return createAnthropicModelTracePayload(modelID, prompt, isOAuth)
		})
		if err != nil {
			return modelTraceResponse{}, nil, err
		}
		res, code, err := s.executeAnthropicStreamingProbe(req, account, parseAnthropicMessagesStream)
		return modelTraceResponse{
			Text: res.Text,
			Usage: openAIProbeUsage{
				InputTokens:  res.Usage.InputTokens + res.Usage.CacheCreationInputTokens + res.Usage.CacheReadInputTokens,
				OutputTokens: res.Usage.OutputTokens,
			},
			StopReason: res.StopReason,
			Thinking:   res.SawThinking,
			Completed:  res.Completed,
		}, code, err
	}

	var completed bool
	text, usage, code, err := s.openAIResponsesProbeRequest(ctx, account, requestModel, func(modelID string, isOAuth bool) map[string]any {
		return createOpenAIModelTracePayload(modelID, prompt, isOAuth)
	}, func(body io.Reader) (string, openAIProbeUsage, error) {
		t, u, c, e := parseOpenAIResponsesStreamDetailed(body, true)
		completed = c
		return t, u, e
	})
	return modelTraceResponse{
		Text:      text,
		Usage:     usage,
		Thinking:  usage.ReasoningTokens > 0,
		Completed: completed,
	}, code, err
}

// createOpenAIModelTracePayload：一条 user 消息，不带 reasoning / temperature / max_output_tokens。
// Codex OAuth 后端要求 instructions 非空，用与真实转发相同的 Codex 基础提示词（GPT 指纹库也采集自 Codex）。
func createOpenAIModelTracePayload(modelID, prompt string, isOAuth bool) map[string]any {
	payload := map[string]any{
		"model": modelID,
		"input": []map[string]any{
			{
				"role": "user",
				"content": []map[string]any{
					{"type": "input_text", "text": prompt},
				},
			},
		},
		"stream": true,
		"store":  false,
	}
	if isOAuth {
		payload["instructions"] = openai.CodexBaseInstructionsForModel(modelID)
	}
	return payload
}

// createAnthropicModelTracePayload：一条 user 消息、max_tokens 4096，不带 temperature / thinking。
// OAuth 凭证只允许 Claude Code 客户端使用，必须带一行 Claude Code system 与 metadata.user_id；API-Key 不带 system。
func createAnthropicModelTracePayload(modelID, prompt string, isOAuth bool) (map[string]any, error) {
	payload := map[string]any{
		"model": modelID,
		"messages": []map[string]any{
			{
				"role": "user",
				"content": []map[string]any{
					{"type": "text", "text": prompt},
				},
			},
		},
		"max_tokens": groupStatusModelTraceAnthropicMaxTokens,
		"stream":     true,
	}
	if isOAuth {
		sessionID, err := generateSessionString()
		if err != nil {
			return nil, err
		}
		payload["system"] = []map[string]any{
			{"type": "text", "text": claudeCodeSystemPrompt},
		}
		payload["metadata"] = map[string]string{"user_id": sessionID}
	}
	return payload, nil
}

// estimateModelTraceCostUSD 按网关计价估算一次运行的费用（预期模型优先，其次请求模型）；无价格时为 0。
func (s *GroupStatusProbeService) estimateModelTraceCostUSD(expected, requestModel string, inputTokens, outputTokens int64) float64 {
	if s == nil || s.gatewaySvc == nil || s.gatewaySvc.billingService == nil {
		return 0
	}
	if inputTokens <= 0 && outputTokens <= 0 {
		return 0
	}
	tokens := UsageTokens{InputTokens: int(inputTokens), OutputTokens: int(outputTokens)}
	for _, model := range []string{expected, requestModel} {
		if strings.TrimSpace(model) == "" {
			continue
		}
		if cost, err := s.gatewaySvc.billingService.CalculateCost(model, tokens, 1); err == nil && cost != nil {
			return cost.TotalCost
		}
	}
	return 0
}

func (s *GroupStatusProbeService) saveModelTraceExecution(ctx context.Context, group *Group, cfg *GroupStatusConfig, account *Account, result *GroupStatusModelTraceResult) (*GroupStatusModelTraceExecution, error) {
	if result == nil {
		result = &GroupStatusModelTraceResult{
			Verdict:     ModelTraceVerdictInconclusive,
			Outcome:     ModelTraceOutcomeInconclusive,
			Reasons:     []string{"empty_result"},
			ErrorDetail: "empty run result",
			StartedAt:   time.Now(),
			FinishedAt:  time.Now(),
		}
	}
	result.GroupID = group.ID
	result.ConfigID = cfg.ID
	if result.Platform == "" {
		result.Platform = group.Platform
	}
	if result.FinishedAt.IsZero() {
		result.FinishedAt = time.Now()
	}
	if result.StartedAt.IsZero() {
		result.StartedAt = result.FinishedAt
	}
	result.ErrorDetail = truncateProbeText(redactProbeUpstreamAddresses(result.ErrorDetail))

	run, state, event, err := s.repo.SaveModelTraceRun(ctx, result)
	if err != nil {
		return nil, err
	}
	if event != nil && s.notifier != nil {
		s.notifier.NotifyTransition(group, cfg, event)
	}
	return &GroupStatusModelTraceExecution{
		Group:   group,
		Config:  cfg,
		Account: account,
		Result:  result,
		Run:     run,
		State:   state,
		Event:   event,
	}, nil
}
