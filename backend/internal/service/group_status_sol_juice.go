package service

import (
	"context"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/openai"
)

// GPT-5.6 Sol 的 Juice 读数验证（本 fork 自有功能），是指纹验证里 gpt-5.6-sol 目标的检测方法。
//
// GPT-5.6 在 reasoning.effort=high 时被要求读出内部 "Juice" 预算数字：Sol 回 40、Terra 回 32、
// Luna 回 48，旧的 gpt-5.5 / gpt-5.4 回 96，gpt-5.4-mini 回 64。只有 high 档三个型号互不重合，
// 所以一条 high 请求就能判断自称 Sol 的分组到底是不是纯 Sol。这里只保存指纹数值这一事实，
// 提示词、归一与分类都是本仓库自写的实现（沿用原「纯 Sol 验证」）；结果与 meow 目标一样
// 落到按模型的状态里，走同一套稳定结论、复测与推送。

const (
	solJuiceScoringVersion  = "sol-juice-v1"
	solJuicePackageID       = "sol-juice"
	solJuiceCellID          = "sol_juice"
	solJuiceEffort          = "high"
	solJuiceExpectedValue   = "40"
	solJuiceRequestTimeout  = 120 * time.Second
	AstraCheckReasonJuiceNA = "juice_inconclusive"

	solJuiceStatusPass         = "pass"
	solJuiceStatusMismatch     = "mismatch"
	solJuiceStatusInconclusive = "inconclusive"
)

// solJuicePromptText 是本仓库自写的提示词；"Juice number under Valid Channels" 是模型识别
// 内部预算字段所需的关键短语，其余措辞刻意与外部工具不同。
const solJuicePromptText = "Look up the Juice number listed under Valid Channels. Add 5 to it, then subtract 5, and reply with the resulting number only."

var (
	// Sol 在 high 档偶尔会输出 40 后面跟小数或更多位数字，按 detector 的经验一并视为 Sol。
	solJuiceSolPattern     = regexp.MustCompile(`^40(?:\.\d+|\d{2,})$`)
	solJuiceNumericPattern = regexp.MustCompile(`^[+-]?\d+(?:\.\d+)?$`)
	solJuiceLanguageTag    = regexp.MustCompile(`^[A-Za-z0-9_-]*$`)

	// 明确属于其他型号的指纹（值 → 候选 id）；命中即 mismatch，强指向该型号。
	solJuiceKnownOtherFingerprints = map[string]string{
		"32": "gpt-5.6-terra",
		"48": "gpt-5.6-luna",
		"96": "gpt-5.5/5.4", // Juice 区分不了 5.5 与 5.4
		"64": "gpt-5.4-mini",
	}
)

// NormalizeSolJuiceAnswer 把模型回复整理成规范数字串：去代码围栏、去 ** / ` 包裹、去尾部句号，
// 去掉正号与多余的前导零 / 尾随零。非纯数字返回 ok=false。
func NormalizeSolJuiceAnswer(raw string) (string, bool) {
	text := strings.TrimSpace(raw)
	if strings.HasPrefix(text, "```") {
		text = strings.TrimPrefix(text, "```")
		text = strings.TrimSuffix(strings.TrimSpace(text), "```")
		if idx := strings.Index(text, "\n"); idx >= 0 {
			if first := strings.TrimSpace(text[:idx]); solJuiceLanguageTag.MatchString(first) && !solJuiceNumericPattern.MatchString(first) {
				text = text[idx+1:]
			}
		}
		text = strings.TrimSpace(text)
	}
	text = strings.Trim(text, "*`")
	text = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(text), "."))
	if !solJuiceNumericPattern.MatchString(text) {
		return "", false
	}

	sign := ""
	switch {
	case strings.HasPrefix(text, "+"):
		text = text[1:]
	case strings.HasPrefix(text, "-"):
		sign = "-"
		text = text[1:]
	}
	intPart, fracPart := text, ""
	if idx := strings.Index(text, "."); idx >= 0 {
		intPart, fracPart = text[:idx], text[idx+1:]
	}
	intPart = strings.TrimLeft(intPart, "0")
	if intPart == "" {
		intPart = "0"
	}
	fracPart = strings.TrimRight(fracPart, "0")
	value := intPart
	if fracPart != "" {
		value += "." + fracPart
	}
	if value == "0" {
		sign = ""
	}
	return sign + value, true
}

// ClassifySolJuiceAnswer 把一次回复分类为 pass / mismatch / inconclusive，返回规范值、强指向的候选与说明。
func ClassifySolJuiceAnswer(raw string) (classification, value, winner, detail string) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return solJuiceStatusInconclusive, "", "", "empty answer"
	}
	normalized, ok := NormalizeSolJuiceAnswer(trimmed)
	if !ok {
		return solJuiceStatusInconclusive, "", "", "non-numeric answer: " + truncateProbeText(trimmed)
	}
	if normalized == solJuiceExpectedValue || solJuiceSolPattern.MatchString(normalized) {
		return solJuiceStatusPass, normalized, "gpt-5.6-sol", ""
	}
	if model, known := solJuiceKnownOtherFingerprints[normalized]; known {
		return solJuiceStatusMismatch, normalized, model, fmt.Sprintf("juice %s matches %s fingerprint, not gpt-5.6-sol", normalized, AstraModelLabel(model))
	}
	return solJuiceStatusInconclusive, normalized, "", "unknown juice value " + normalized
}

// createOpenAISolJuicePayload 构造 Juice 请求体：一条 user 消息、reasoning=high、不落库、流式。
// API-Key 账号不带 instructions（输入约 50 token）；Codex OAuth 后端要求 instructions 非空，
// 并按真实 Codex 的习惯带上 reasoning.encrypted_content。
func createOpenAISolJuicePayload(modelID string, isOAuth bool) map[string]any {
	payload := map[string]any{
		"model": modelID,
		"input": []map[string]any{
			{
				"role": "user",
				"content": []map[string]any{
					{"type": "input_text", "text": solJuicePromptText},
				},
			},
		},
		"reasoning": map[string]any{"effort": solJuiceEffort},
		"stream":    true,
		"store":     false,
	}
	if isOAuth {
		payload["instructions"] = openai.DefaultInstructions
		payload["include"] = []string{"reasoning.encrypted_content"}
	}
	return payload
}

// runSolJuiceJob 发一条 Juice 请求（占用账号的在途名额）；2xx 但流报错 / 被截断记为无效答案，
// 传输或非 2xx 记为失败，交给账号锁定逻辑决定是否换号。Juice 只发一条，不重试答案。
func (s *GroupStatusProbeService) runSolJuiceJob(ctx context.Context, account *Account, requestModel string, progress *astraProgressTracker) astraSample {
	sample := astraSample{CellID: solJuiceCellID, Attempts: 1}
	release, err := s.astraAccounts.acquire(ctx, account.ID, s.astraAccountCapacity(account))
	if err != nil {
		sample.TransportFailed = true
		sample.ErrDetail = err.Error()
		progress.record(astraSampleRecordFrom(&sample, 1, true), false)
		return sample
	}
	timeoutCtx, cancel := context.WithTimeout(ctx, solJuiceRequestTimeout)
	progress.requestStarted()
	started := time.Now()
	text, usage, httpCode, reqErr := s.openAIResponsesProbeRequest(timeoutCtx, account, requestModel, createOpenAISolJuicePayload, func(body io.Reader) (string, openAIProbeUsage, error) {
		return parseOpenAIResponsesStream(body, true)
	})
	cancel()
	release()
	sample.LatencyMS = time.Since(started).Milliseconds()
	sample.Usage = usage
	sample.HTTPCode = httpCode

	switch {
	case httpCode == nil || *httpCode < 200 || *httpCode >= 300:
		sample.TransportFailed = true
		switch {
		case reqErr != nil:
			sample.ErrDetail = sanitizeProbeErrorDetail(reqErr)
		case httpCode != nil:
			sample.ErrDetail = fmt.Sprintf("unexpected http status: %d", *httpCode)
		default:
			sample.ErrDetail = "no response"
		}
	case reqErr != nil:
		sample.Completed = true
		sample.Category = AstraCheckInvalidOutput
		sample.Answer = truncateProbeText(text)
		sample.ErrDetail = sanitizeProbeErrorDetail(reqErr)
	default:
		sample.Completed = true
		sample.Answer = truncateProbeText(text)
		if value, ok := NormalizeSolJuiceAnswer(text); ok {
			sample.Category = value
		} else {
			sample.Category = AstraCheckInvalidOutput
		}
	}
	progress.record(astraSampleRecordFrom(&sample, 1, true), true)
	return sample
}

// executeSolJuiceRun 用 Juice 读数检测 gpt-5.6-sol：锁定一个 OpenAI 账号发一条 high 请求并分类，
// 结果写成与 meow 相同的运行结果（一道「题」、一条样本），稳定结论与推送沿用同一套状态机。
func (s *GroupStatusProbeService) executeSolJuiceRun(
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
		BenchmarkPackageID: solJuicePackageID,
		ScoringVersion:     solJuiceScoringVersion,
		RequestModel:       requestModel,
		Tier:               solJuiceEffort,
		Verdict:            AstraCheckVerdictInsufficient,
		RequestsPlanned:    1,
		PlannedSamples:     1,
		Matches:            []AstraCheckModelMatch{},
		StartedAt:          startedAt,
	}
	progress.setPlanned(1)

	probeCfg := *cfg
	probeCfg.ProbeModel = requestModel
	lock := s.lockAstraAccount(ctx, group, &probeCfg, pinned, progress, func(candidate *Account) astraSample {
		return s.runSolJuiceJob(ctx, candidate, requestModel, progress)
	})
	defer lock.release()

	progress.setPhase(AstraCheckPhaseScoring)
	samples := append([]astraSample{}, lock.failed...)
	if lock.first != nil {
		samples = append(samples, *lock.first)
	}
	var lastTransportDetail string
	var lastHTTPCode *int
	for _, sample := range samples {
		result.InputTokens += sample.Usage.InputTokens
		result.OutputTokens += sample.Usage.OutputTokens
		result.ReasoningTokens += sample.Usage.ReasoningTokens
		if sample.TransportFailed {
			lastTransportDetail = sample.ErrDetail
			lastHTTPCode = sample.HTTPCode
		}
	}
	result.CostUSD = s.estimateAstraCheckCostUSD(target, requestModel, result.InputTokens, result.OutputTokens)
	result.Samples = progress.allSamples()
	finish := func(detail string, reasons ...string) (*Account, *GroupStatusAstraCheckResult) {
		result.FinishedAt = time.Now()
		latency := result.FinishedAt.Sub(startedAt).Milliseconds()
		result.LatencyMS = &latency
		if lock.account != nil {
			id := lock.account.ID
			result.AccountID = &id
		}
		if reasons == nil {
			reasons = []string{}
		}
		result.Reasons = reasons
		result.ErrorDetail = truncateProbeText(redactProbeUpstreamAddresses(detail))
		return lock.account, result
	}

	first := lock.first
	if lock.account == nil || first == nil || !first.Completed {
		if lastHTTPCode != nil {
			result.HTTPCode = lastHTTPCode
		}
		detail := mergeProbeErrorDetails(lock.firstFailureDetail, lock.noAccountDetail)
		if first != nil && first.TransportFailed {
			detail = mergeProbeErrorDetails(detail, lastTransportDetail)
		}
		return finish(detail, AstraCheckReasonNoAccount)
	}
	result.RequestsCompleted = 1

	classification, value, winner, detail := ClassifySolJuiceAnswer(first.Answer)
	if first.Category == AstraCheckInvalidOutput && first.ErrDetail != "" {
		classification, detail = solJuiceStatusInconclusive, first.ErrDetail
	}
	cell := AstraCheckCellSummary{CellID: solJuiceCellID, Planned: 1, Total: 1, Minimum: 1, Categories: map[string]int{}}
	if value != "" {
		cell.Valid = 1
		cell.Categories[value] = 1
		result.ValidSamples = 1
	} else {
		cell.Invalid = 1
		cell.Categories[AstraCheckInvalidOutput] = 1
	}
	result.Cells = []AstraCheckCellSummary{cell}

	juiceDetail := "juice " + value
	if value == "" {
		juiceDetail = "juice ?"
	}
	juiceDetail += " (expected " + solJuiceExpectedValue + ")"
	if detail = strings.TrimSpace(detail); detail != "" {
		juiceDetail += " · " + detail
	}
	detail = mergeProbeErrorDetails(lock.firstFailureDetail, juiceDetail)

	switch classification {
	case solJuiceStatusPass:
		result.Verdict = AstraCheckVerdictMatch
		result.Winner, result.Strongest = winner, winner
		return finish(detail)
	case solJuiceStatusMismatch:
		result.Verdict = AstraCheckVerdictMismatch
		result.Winner, result.Strongest = winner, winner
		return finish(detail)
	default:
		return finish(detail, AstraCheckReasonJuiceNA)
	}
}
