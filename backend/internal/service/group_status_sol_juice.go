package service

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// 纯 Sol 验证（Juice 指纹探测），本 fork 自有功能。
//
// GPT-5.6 在 reasoning.effort=high 时被要求读出内部 "Juice" 预算数字：Sol 回 40、Terra 回 32、
// Luna 回 48，旧的 gpt-5.5 / gpt-5.4 回 96，gpt-5.4-mini 回 64。只有 high 档三个型号互不重合，
// 所以一条 high 请求就能判断自称 Sol 的分组到底是不是纯 Sol。这里只保存指纹数值这一事实，
// 提示词、分类和状态机都是本仓库自写的实现。

const (
	GroupStatusEventSolJuiceMismatch  = "sol_juice_mismatch"
	GroupStatusEventSolJuiceRecovered = "sol_juice_recovered"

	SolJuiceStatusPass         = "pass"
	SolJuiceStatusMismatch     = "mismatch"
	SolJuiceStatusInconclusive = "inconclusive"

	groupStatusSolJuiceDefaultIntervalSeconds = 900
	groupStatusSolJuiceMinIntervalSeconds     = 300
	groupStatusSolJuiceDefaultModel           = "gpt-5.6-sol"
	groupStatusSolJuiceEffort                 = "high"
	groupStatusSolJuiceTimeout                = 120 * time.Second
	groupStatusSolJuiceMismatchThreshold      = 2

	// 与 billing_service.go 里 fallbackPrices["gpt-5.6-sol"] 保持一致（USD/token）；
	// OpenAI 的 output_tokens 已包含 reasoning token，不再重复计价。
	solJuiceInputPricePerToken  = 5e-6
	solJuiceOutputPricePerToken = 30e-6
)

var (
	ErrGroupStatusSolJuiceUnsupported = infraerrors.BadRequest("GROUP_STATUS_SOL_JUICE_UNSUPPORTED", "Sol Juice probe is only available for OpenAI groups")

	// Sol 在 high 档偶尔会输出 40 后面跟小数或更多位数字，按 detector 的经验一并视为 Sol。
	solJuiceSolPattern     = regexp.MustCompile(`^40(?:\.\d+|\d{2,})$`)
	solJuiceNumericPattern = regexp.MustCompile(`^[+-]?\d+(?:\.\d+)?$`)
	solJuiceLanguageTag    = regexp.MustCompile(`^[A-Za-z0-9_-]*$`)

	// 明确属于其他型号的指纹；命中即 mismatch。
	solJuiceKnownOtherFingerprints = map[string]string{
		"32": "gpt-5.6-terra",
		"48": "gpt-5.6-luna",
		"96": "gpt-5.5 / gpt-5.4",
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

// ClassifySolJuiceAnswer 把一次回复分类为 pass / mismatch / inconclusive，并返回规范值与说明。
func ClassifySolJuiceAnswer(raw string) (classification, value, detail string) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return SolJuiceStatusInconclusive, "", "empty answer"
	}
	normalized, ok := NormalizeSolJuiceAnswer(trimmed)
	if !ok {
		return SolJuiceStatusInconclusive, "", "non-numeric answer: " + truncateProbeText(trimmed)
	}
	if normalized == "40" || solJuiceSolPattern.MatchString(normalized) {
		return SolJuiceStatusPass, normalized, ""
	}
	if model, known := solJuiceKnownOtherFingerprints[normalized]; known {
		return SolJuiceStatusMismatch, normalized, fmt.Sprintf("juice %s matches %s fingerprint, not gpt-5.6-sol", normalized, model)
	}
	return SolJuiceStatusInconclusive, normalized, "unknown juice value " + normalized
}

// ComputeSolJuiceTransition 是纯函数：把一次 Juice 结果并进状态，并在稳定结论切换时产出事件。
//
//   - pass：清零计数，稳定结论置为 pass；若原来是 mismatch 则发 sol_juice_recovered
//   - mismatch：计数 +1；原来不是 mismatch 且计数达到阈值（2）时置为 mismatch 并发 sol_juice_mismatch
//   - inconclusive：只更新最近一次结果，不动稳定结论和计数
//
// 存活探测的 latest_status / stable_status 在这里不会被改动。
func ComputeSolJuiceTransition(prev *GroupStatusState, result *GroupStatusSolJuiceResult) (*GroupStatusState, *GroupStatusEvent) {
	next := &GroupStatusState{}
	if prev != nil {
		*next = *prev
	}
	next.GroupID = result.GroupID
	if next.ConfigID == 0 {
		next.ConfigID = result.ConfigID
	}

	observedAt := result.ObservedAt
	if observedAt.IsZero() {
		observedAt = time.Now()
	}
	detail := strings.TrimSpace(result.ErrorDetail)
	if detail == "" {
		detail = strings.TrimSpace(result.AnswerExcerpt)
	}
	next.SolJuiceStatus = result.Classification
	next.SolJuiceValue = result.NormalizedValue
	next.SolJuiceDetail = detail
	next.SolJuiceCheckedAt = &observedAt
	next.SolJuiceInputTokens = result.InputTokens
	next.SolJuiceOutputTokens = result.OutputTokens
	next.SolJuiceReasoningTokens = result.ReasoningTokens

	prevStable := strings.TrimSpace(next.SolJuiceStableStatus)
	newEvent := func(eventType, from, to string) *GroupStatusEvent {
		subStatus := "juice_unknown"
		if result.NormalizedValue != "" {
			subStatus = "juice_" + result.NormalizedValue
		}
		return &GroupStatusEvent{
			GroupID:     result.GroupID,
			ConfigID:    result.ConfigID,
			EventType:   eventType,
			FromStatus:  from,
			ToStatus:    to,
			LatencyMS:   result.LatencyMS,
			HTTPCode:    result.HTTPCode,
			SubStatus:   subStatus,
			ErrorDetail: result.ErrorDetail,
			ObservedAt:  observedAt,
		}
	}

	switch result.Classification {
	case SolJuiceStatusPass:
		next.SolJuiceConsecutiveMismatch = 0
		next.SolJuiceStableStatus = SolJuiceStatusPass
		if prevStable == SolJuiceStatusMismatch {
			return next, newEvent(GroupStatusEventSolJuiceRecovered, prevStable, SolJuiceStatusPass)
		}
		return next, nil
	case SolJuiceStatusMismatch:
		next.SolJuiceConsecutiveMismatch++
		if prevStable == SolJuiceStatusMismatch {
			return next, nil
		}
		if next.SolJuiceConsecutiveMismatch >= groupStatusSolJuiceMismatchThreshold {
			next.SolJuiceStableStatus = SolJuiceStatusMismatch
			return next, newEvent(GroupStatusEventSolJuiceMismatch, prevStable, SolJuiceStatusMismatch)
		}
		return next, nil
	default:
		return next, nil
	}
}

// EstimateSolJuiceCostUSD 按 gpt-5.6-sol 标准价估算一次请求的费用；output_tokens 已含 reasoning。
func EstimateSolJuiceCostUSD(inputTokens, outputTokens int64) float64 {
	if inputTokens <= 0 && outputTokens <= 0 {
		return 0
	}
	cost := 0.0
	if inputTokens > 0 {
		cost += float64(inputTokens) * solJuiceInputPricePerToken
	}
	if outputTokens > 0 {
		cost += float64(outputTokens) * solJuiceOutputPricePerToken
	}
	return cost
}

// decorateSolJuiceSummary 填充只读的派生字段（不落库）。
func decorateSolJuiceSummary(summary *GroupStatusSummary) {
	if summary == nil {
		return
	}
	summary.SolJuiceLastCostUSD = EstimateSolJuiceCostUSD(summary.SolJuiceInputTokens, summary.SolJuiceOutputTokens)
}

// isGroupStatusNotifyEvent 报告事件类型是否需要推送提醒。
func isGroupStatusNotifyEvent(eventType string) bool {
	switch eventType {
	case GroupStatusEventDown, GroupStatusEventUp, GroupStatusEventSolJuiceMismatch, GroupStatusEventSolJuiceRecovered:
		return true
	default:
		return false
	}
}

// isSolJuiceEvent 报告事件是否来自 Juice 探测。
func isSolJuiceEvent(eventType string) bool {
	return eventType == GroupStatusEventSolJuiceMismatch || eventType == GroupStatusEventSolJuiceRecovered
}
