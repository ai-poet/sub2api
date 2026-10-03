package service

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/openai"
)

// ModelTrace 方法的执行：锁定一个账号 → 发 3 条（不够时补到最多 6 条）数字挑战 → 本地归因 → Guard 规则判定，
// 写成与 meow / Juice 相同形状的运行结果。请求刻意贴近指纹库的采集环境：Claude（库内数据经 API 直接采集）
// 只带一条 user 消息、max_tokens 4096，不带 temperature / thinking，OAuth 账号只带通道硬性要求的一行
// Claude Code system；GPT（库内数据在官方 Codex 里采集）走 Responses，带该模型真实的 Codex base
// instructions、Codex 默认的 low 推理与 low verbosity，不带 temperature。

// modelTraceAttempt 是一条挑战的最终结果。
type modelTraceAttempt struct {
	seq      int
	numbers  []int
	accepted bool
	sample   astraSample
}

// modelTraceResponse 是一次挑战请求的解析结果。
type modelTraceResponse struct {
	Text       string
	Usage      openAIProbeUsage
	StopReason string
	Completed  bool
}

func (s *GroupStatusProbeService) modelTraceBank() (*ModelTraceBank, *ModelTraceBankMeta, error) {
	if s != nil && s.modelTraceBankSource != nil {
		return s.modelTraceBankSource.Active()
	}
	return LoadEmbeddedModelTraceBank()
}

func (s *GroupStatusProbeService) newModelTraceChallenges(n int) []ModelTraceChallenge {
	if s != nil && s.modelTraceChallengeGen != nil {
		return s.modelTraceChallengeGen(n)
	}
	return GenerateModelTraceChallenges(newModelTraceRNG(), n)
}

// modelTraceCandidateID 把指纹库里的模型 id 换成目标 id（如 claude-opus-5-5 → claude-opus-5.5），
// 让状态、事件与界面用同一套 id；不是目标的模型原样返回。代表模型（TraceProxy）不换：
// gpt-6-astra 始终是 GPT-6 Astra，不会因为它代表 GPT-6.1 Sol 就在别的结果里显示成 6.1 Sol。
func modelTraceCandidateID(bankModel string) string {
	for _, target := range astraCheckTargets {
		if target.TraceModelID != "" && !target.TraceProxy && target.TraceModelID == bankModel {
			return target.ID
		}
	}
	return bankModel
}

// modelTraceBankVersion 是指纹库的展示版本：建库日期 + sha 前 8 位。
func modelTraceBankVersion(meta *ModelTraceBankMeta) string {
	if meta == nil {
		return ""
	}
	version := meta.BuiltAt
	if len(version) >= 10 {
		version = version[:10]
	}
	sha := meta.SHA256
	if len(sha) > 8 {
		sha = sha[:8]
	}
	return strings.TrimSpace(version + " " + sha)
}

// modelTraceMatches 把归因排名（前 5 名 + 预期模型）转成候选匹配：Match 是归因概率，
// Threshold 是 Guard 的线（预期模型 50% 判一致，其他模型 80% 才判强指向）。
func modelTraceMatches(analysis *ModelTraceAnalysis, expectedBank, verdict string) []AstraCheckModelMatch {
	out := []AstraCheckModelMatch{}
	if analysis == nil {
		return out
	}
	top := analysis.Top()
	for _, entry := range modelTraceTopRanking(analysis.Results, expectedBank) {
		threshold := modelTraceMismatchMinTopProbability
		if entry.Model == expectedBank {
			threshold = modelTraceMatchMinProbability
		}
		passed := (verdict == ModelTraceVerdictMatch && entry.Model == expectedBank) ||
			(verdict == ModelTraceVerdictMismatch && top != nil && entry.Model == top.Model)
		out = append(out, AstraCheckModelMatch{
			Model:     modelTraceCandidateID(entry.Model),
			Name:      AstraModelLabel(entry.Model),
			Score:     entry.Score,
			Match:     entry.Probability,
			Threshold: threshold,
			Passed:    passed,
		})
	}
	return out
}

// modelTraceRequest 按账号平台发一条挑战：Claude 走 Messages，GPT 走 Responses。
func (s *GroupStatusProbeService) modelTraceRequest(ctx context.Context, account *Account, requestModel, prompt string) (modelTraceResponse, *int, error) {
	if account == nil {
		return modelTraceResponse{}, nil, errors.New("nil account")
	}
	switch account.Platform {
	case PlatformAnthropic:
		return s.modelTraceAnthropicRequest(ctx, account, requestModel, prompt)
	case PlatformOpenAI:
		return s.modelTraceOpenAIRequest(ctx, account, requestModel, prompt)
	default:
		return modelTraceResponse{}, nil, fmt.Errorf("modeltrace probe does not support %s accounts", account.Platform)
	}
}

func (s *GroupStatusProbeService) modelTraceAnthropicRequest(ctx context.Context, account *Account, requestModel, prompt string) (modelTraceResponse, *int, error) {
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
		Completed:  res.Completed,
	}, code, err
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

func (s *GroupStatusProbeService) modelTraceOpenAIRequest(ctx context.Context, account *Account, requestModel, prompt string) (modelTraceResponse, *int, error) {
	var parsed modelTraceResponse
	text, usage, code, err := s.openAIResponsesProbeRequest(ctx, account, requestModel, func(modelID string, isOAuth bool) map[string]any {
		return createOpenAIModelTracePayload(modelID, prompt, isOAuth)
	}, func(body io.Reader) (string, openAIProbeUsage, error) {
		res, parseErr := parseOpenAIModelTraceStream(body)
		parsed = res
		return res.Text, res.Usage, parseErr
	})
	parsed.Text = text
	parsed.Usage = usage
	return parsed, code, err
}

// createOpenAIModelTracePayload：Codex 为该模型使用的 base instructions + 一条 user 消息，low 推理、low verbosity，
// 不带 temperature、不落库。API-Key 账号也带同一份 instructions，让两类账号与库内的 Codex 采集环境一致。
func createOpenAIModelTracePayload(modelID, prompt string, isOAuth bool) map[string]any {
	payload := map[string]any{
		"model":        modelID,
		"instructions": openai.CodexBaseInstructionsForModel(modelID),
		"input": []map[string]any{
			{
				"role": "user",
				"content": []map[string]any{
					{"type": "input_text", "text": prompt},
				},
			},
		},
		"reasoning":         map[string]any{"effort": groupStatusModelTraceOpenAIEffort},
		"text":              map[string]any{"verbosity": groupStatusModelTraceOpenAIVerbosity},
		"max_output_tokens": groupStatusModelTraceOpenAIMaxOutputTokens,
		"stream":            true,
		"store":             false,
	}
	if isOAuth {
		payload["include"] = []string{"reasoning.encrypted_content"}
	}
	return payload
}

// parseOpenAIModelTraceStream 解析一条挑战的 Responses 流，把收尾方式折算成与 Claude 相同的判据：
// 撞到 max_output_tokens 记为 max_tokens，拒答或内容过滤记为 refusal，流没有正常收尾时 Completed 为 false。
func parseOpenAIModelTraceStream(body io.Reader) (modelTraceResponse, error) {
	var (
		res     modelTraceResponse
		parts   []string
		refused bool
	)
	reader := bufio.NewReader(body)
	for {
		line, readErr := reader.ReadString('\n')
		if readErr != nil && line == "" {
			res.Text = strings.Join(parts, "")
			if readErr == io.EOF {
				return res, nil
			}
			return res, readErr
		}
		line = strings.TrimSpace(line)
		if line == "" || !sseDataPrefix.MatchString(line) {
			continue
		}
		jsonStr := sseDataPrefix.ReplaceAllString(line, "")
		if jsonStr == "[DONE]" {
			res.Text = strings.Join(parts, "")
			return res, nil
		}
		var data map[string]any
		if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
			continue
		}
		switch data["type"] {
		case "response.output_text.delta":
			if delta, ok := data["delta"].(string); ok {
				parts = append(parts, delta)
			}
		case "response.refusal.delta", "response.refusal.done":
			refused = true
		case "response.completed", "response.incomplete":
			resp, _ := data["response"].(map[string]any)
			res.Usage = parseOpenAIResponseUsage(resp)
			res.Text = strings.Join(parts, "")
			if strings.TrimSpace(res.Text) == "" {
				res.Text = extractOpenAIResponseOutputText(resp)
			}
			res.Completed = true
			if data["type"] == "response.incomplete" {
				details, _ := resp["incomplete_details"].(map[string]any)
				switch reason, _ := details["reason"].(string); reason {
				case "max_output_tokens":
					res.StopReason = "max_tokens"
				case "content_filter":
					res.StopReason = "refusal"
				default:
					res.Completed = false
				}
			}
			if refused || openAIResponseHasRefusal(resp) {
				res.StopReason = "refusal"
			}
			return res, nil
		case "response.failed":
			resp, _ := data["response"].(map[string]any)
			res.Usage = parseOpenAIResponseUsage(resp)
			res.Text = strings.Join(parts, "")
			return res, errors.New(openAIResponseFailureMessage(resp, "openai probe response.failed"))
		case "error":
			res.Text = strings.Join(parts, "")
			if errData, ok := data["error"].(map[string]any); ok {
				if msg, ok := errData["message"].(string); ok && msg != "" {
					return res, errors.New(msg)
				}
			}
			return res, errors.New("openai probe failed")
		}
	}
}

// openAIResponseHasRefusal 报告最终 output 里是否有 refusal 内容块。
func openAIResponseHasRefusal(resp map[string]any) bool {
	output, _ := resp["output"].([]any)
	for _, item := range output {
		itemMap, _ := item.(map[string]any)
		content, _ := itemMap["content"].([]any)
		for _, block := range content {
			if blockMap, ok := block.(map[string]any); ok && blockMap["type"] == "refusal" {
				return true
			}
		}
	}
	return false
}

// runModelTraceChallenge 执行一条挑战：传输错误 / 408 / 429 / 5xx 最多试 3 次；内容不合格（数字不够、
// 截断、拒答）不重发同一提示，直接消耗这条挑战，由后续挑战补足。
func (s *GroupStatusProbeService) runModelTraceChallenge(ctx context.Context, account *Account, requestModel string, challenge ModelTraceChallenge, seq int, progress *astraProgressTracker) modelTraceAttempt {
	att := modelTraceAttempt{seq: seq}
	sample := astraSample{CellID: fmt.Sprintf("challenge-%d", seq)}
	minimum := ModelTraceMinimumNumbers(challenge.ExpectedCount)
	for attempt := 1; attempt <= groupStatusModelTraceMaxHTTPAttempts; attempt++ {
		sample.Attempts = attempt
		release, acquireErr := s.astraAccounts.acquire(ctx, account.ID, s.astraAccountCapacity(account))
		if acquireErr != nil {
			sample.TransportFailed = true
			sample.Completed = false
			sample.ErrDetail = acquireErr.Error()
			progress.record(astraSampleRecordFrom(&sample, attempt, true), false)
			att.sample = sample
			return att
		}
		timeoutCtx, cancel := context.WithTimeout(ctx, groupStatusModelTraceRequestTimeout)
		progress.requestStarted()
		started := time.Now()
		resp, httpCode, err := s.modelTraceRequest(timeoutCtx, account, requestModel, challenge.Prompt)
		cancel()
		release()
		sample.LatencyMS = time.Since(started).Milliseconds()
		sample.Usage.InputTokens += resp.Usage.InputTokens
		sample.Usage.OutputTokens += resp.Usage.OutputTokens
		sample.Usage.ReasoningTokens += resp.Usage.ReasoningTokens
		sample.HTTPCode = httpCode

		if httpCode == nil || *httpCode < 200 || *httpCode >= 300 {
			sample.TransportFailed = true
			sample.Completed = false
			switch {
			case err != nil:
				sample.ErrDetail = sanitizeProbeErrorDetail(err)
			case httpCode != nil:
				sample.ErrDetail = fmt.Sprintf("unexpected http status: %d", *httpCode)
			default:
				sample.ErrDetail = "no response"
			}
			final := attempt == groupStatusModelTraceMaxHTTPAttempts || !modelTraceRetryable(httpCode)
			progress.record(astraSampleRecordFrom(&sample, attempt, final), true)
			if final {
				break
			}
			if sleepErr := s.astraSleepFor(ctx, time.Duration(attempt)*time.Second); sleepErr != nil {
				sample.ErrDetail = mergeProbeErrorDetails(sample.ErrDetail, sleepErr.Error())
				break
			}
			continue
		}

		sample.TransportFailed = false
		sample.Completed = true
		sample.ErrDetail = ""
		sample.Answer = modelTraceExcerpt(resp.Text)
		numbers := ParseModelTraceNumbers(resp.Text)
		counts := fmt.Sprintf("%d/%d", len(numbers), challenge.ExpectedCount)
		rejection := ""
		switch {
		case err != nil:
			rejection = ModelTraceRejectionStreamError + ": " + sanitizeProbeErrorDetail(err)
		case resp.StopReason == "max_tokens":
			rejection = ModelTraceRejectionMaxTokens
		case resp.StopReason == "refusal":
			rejection = ModelTraceRejectionRefusal
		case !resp.Completed:
			rejection = ModelTraceRejectionStreamIncomplete
		case len(numbers) < minimum:
			rejection = fmt.Sprintf("%s (%d < %d)", ModelTraceRejectionTooFewNumbers, len(numbers), minimum)
		}
		if rejection != "" {
			sample.Category = AstraCheckInvalidOutput
			sample.ErrDetail = rejection + " · " + counts
		} else {
			sample.Category = counts
			att.accepted = true
			att.numbers = numbers
		}
		progress.record(astraSampleRecordFrom(&sample, attempt, true), true)
		break
	}
	att.sample = sample
	return att
}

// executeModelTraceRun 用 ModelTrace 检测一个目标（target.TraceModelID 是它在指纹库里的 id，或代表它的库内模型）。
func (s *GroupStatusProbeService) executeModelTraceRun(
	ctx context.Context,
	group *Group,
	cfg *GroupStatusConfig,
	model AstraCheckModelConfig,
	target AstraCheckTarget,
	requestModel string,
	progress *astraProgressTracker,
	round int,
	pinned *Account,
) (*Account, *GroupStatusAstraCheckResult) {
	startedAt := time.Now()
	result := &GroupStatusAstraCheckResult{
		GroupID:            group.ID,
		ConfigID:           cfg.ID,
		Platform:           group.Platform,
		ExpectedModel:      model.ExpectedModel,
		Round:              round,
		BenchmarkPackageID: modelTracePackageID,
		ScoringVersion:     modelTraceScoringVersion,
		RequestModel:       requestModel,
		Verdict:            AstraCheckVerdictInsufficient,
		PlannedSamples:     groupStatusModelTraceTargetOutputs,
		Matches:            []AstraCheckModelMatch{},
		StartedAt:          startedAt,
	}
	bank, meta, bankErr := s.modelTraceBank()
	if meta != nil {
		result.BenchmarkSHA256 = meta.SHA256
		result.BenchmarkVersion = modelTraceBankVersion(meta)
	}
	progress.setPlanned(groupStatusModelTraceTargetOutputs)

	var (
		mu       sync.Mutex
		attempts []modelTraceAttempt
	)
	finish := func(account *Account, extra string, reasons ...string) (*Account, *GroupStatusAstraCheckResult) {
		progress.setPhase(AstraCheckPhaseScoring)
		result.FinishedAt = time.Now()
		latency := result.FinishedAt.Sub(startedAt).Milliseconds()
		result.LatencyMS = &latency
		if account != nil {
			id := account.ID
			result.AccountID = &id
		}
		result.Samples = progress.allSamples()

		var (
			valid       [][]int
			lastHTTP    *int
			lastFailure string
			used        = map[int]struct{}{}
		)
		for _, att := range attempts {
			result.InputTokens += att.sample.Usage.InputTokens
			result.OutputTokens += att.sample.Usage.OutputTokens
			result.ReasoningTokens += att.sample.Usage.ReasoningTokens
			if att.sample.Completed {
				used[att.seq] = struct{}{}
			}
			if att.accepted {
				valid = append(valid, att.numbers)
				continue
			}
			if att.sample.TransportFailed {
				lastHTTP = att.sample.HTTPCode
				lastFailure = att.sample.ErrDetail
			}
		}
		result.RequestsCompleted = len(used)
		result.ValidSamples = len(valid)
		if len(valid) == 0 && lastHTTP != nil {
			result.HTTPCode = lastHTTP
		}
		result.CostUSD = s.estimateAstraCheckCostUSD(target, requestModel, result.InputTokens, result.OutputTokens)

		detail := mergeProbeErrorDetails(extra, lastFailure)
		switch {
		case len(reasons) > 0:
			result.Reasons = reasons
		case len(valid) == 0:
			result.Reasons = []string{ModelTraceReasonNoValidOutputs}
		default:
			analysis, err := bank.AnalyzeNumbers(valid)
			if err != nil {
				result.Reasons = []string{ModelTraceReasonNoValidOutputs}
				detail = mergeProbeErrorDetails(detail, err.Error())
				break
			}
			verdict, classifyReasons := ClassifyModelTrace(analysis, bank, target.TraceModelID)
			result.Matches = modelTraceMatches(analysis, target.TraceModelID, verdict)
			result.Reasons = classifyReasons
			if top := analysis.Top(); top != nil {
				result.Strongest = modelTraceCandidateID(top.Model)
			}
			switch verdict {
			case ModelTraceVerdictMatch:
				result.Verdict = AstraCheckVerdictMatch
				result.Winner = target.ID
			case ModelTraceVerdictMismatch:
				result.Verdict = AstraCheckVerdictMismatch
				result.Winner = result.Strongest
			}
		}
		if result.Reasons == nil {
			result.Reasons = []string{}
		}
		summary := fmt.Sprintf("outputs %d/%d · challenges %d/%d", result.ValidSamples, groupStatusModelTraceTargetOutputs, result.RequestsCompleted, result.RequestsPlanned)
		if detail = strings.TrimSpace(detail); detail != "" {
			summary += " · " + detail
		}
		result.ErrorDetail = truncateProbeText(redactProbeUpstreamAddresses(summary))
		return account, result
	}

	switch {
	case bankErr != nil || bank == nil:
		detail := "fingerprint bank unavailable"
		if bankErr != nil {
			detail = bankErr.Error()
		}
		return finish(nil, detail, ModelTraceReasonBankInvalid)
	case !bank.HasModel(target.TraceModelID):
		return finish(nil, "expected model "+target.TraceModelID+" is not in the fingerprint bank", ModelTraceReasonUnknownExpected)
	}

	challenges := s.newModelTraceChallenges(groupStatusModelTraceMaxChallenges)
	if len(challenges) == 0 {
		return finish(nil, "no challenges generated", ModelTraceReasonNoValidOutputs)
	}
	result.RequestsPlanned = len(challenges)

	// 让调度器按请求模型过滤账号，其余配置照抄；首条挑战同步发出以锁定账号
	probeCfg := *cfg
	probeCfg.ProbeModel = requestModel
	lock := s.lockAstraAccount(ctx, group, &probeCfg, pinned, true, progress, func(candidate *Account) astraSample {
		att := s.runModelTraceChallenge(ctx, candidate, requestModel, challenges[0], 1, progress)
		attempts = append(attempts, att)
		return att.sample
	})
	defer lock.release()
	if lock.account == nil {
		return finish(nil, mergeProbeErrorDetails(lock.firstFailureDetail, lock.noAccountDetail), AstraCheckReasonNoAccount)
	}
	account := lock.account
	progress.setPhase(AstraCheckPhaseRunning)

	accepted := 0
	if last := attempts[len(attempts)-1]; last.accepted {
		accepted = 1
	}
	used := 1
	workers := groupStatusModelTraceMaxWorkers
	if capacity := s.astraAccountCapacity(account); workers > capacity {
		workers = capacity
	}
	for accepted < groupStatusModelTraceTargetOutputs && used < len(challenges) && ctx.Err() == nil {
		wave := groupStatusModelTraceTargetOutputs - accepted
		if remaining := len(challenges) - used; wave > remaining {
			wave = remaining
		}
		if wave > workers {
			wave = workers
		}
		var wg sync.WaitGroup
		for i := 0; i < wave; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				seq := used + i + 1
				att := s.runModelTraceChallenge(ctx, account, requestModel, challenges[used+i], seq, progress)
				mu.Lock()
				attempts = append(attempts, att)
				if att.accepted {
					accepted++
				}
				mu.Unlock()
			}(i)
		}
		wg.Wait()
		used += wave
	}

	extra := lock.firstFailureDetail
	if ctx.Err() != nil {
		extra = mergeProbeErrorDetails(extra, "run interrupted: "+ctx.Err().Error())
	}
	return finish(account, extra)
}
