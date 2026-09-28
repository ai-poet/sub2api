package service

import (
	crand "crypto/rand"
	"encoding/hex"
	"fmt"
	"math/rand/v2"
	"strings"
	"time"
)

// ModelTrace 数字分布指纹（本 fork 自有功能），是指纹验证里 Claude Opus 5.5 目标的检测方法。
//
// 每次运行向分组的一个账号发 3 条（不够时补到最多 6 条）「凭第一反应逐项输出 292–332 个 1..355 整数」的
// 挑战，用 ModelTrace（MIT）统一指纹库对库内全部 16 个模型做闭集归因，再按 ModelTrace Guard 的
// classifySample 阈值判定是否与预期模型一致。挑战措辞与 ModelTrace 网页版（challenge-browser.js）逐字一致，
// 评分器是 fingerprint-core.js 的 Go 移植（数值一致性由 testdata/modeltrace 钉住）；结果与 meow / Juice
// 目标一样落到按模型的状态里，走同一套稳定结论、复测与推送。

const (
	modelTraceScoringVersion = "modeltrace-v1"
	modelTracePackageID      = "modeltrace-bank"

	ModelTraceVerdictMatch        = "match"
	ModelTraceVerdictMismatch     = "mismatch"
	ModelTraceVerdictInconclusive = "inconclusive"

	ModelTraceReasonUnknownExpected = "unknown_expected_model"
	ModelTraceReasonNoValidOutputs  = "no_valid_outputs"
	ModelTraceReasonInsufficient    = "insufficient_outputs"
	ModelTraceReasonAmbiguous       = "ambiguous"
	ModelTraceReasonBankInvalid     = "bank_invalid"

	ModelTraceRejectionTooFewNumbers    = "too_few_numbers"
	ModelTraceRejectionMaxTokens        = "max_tokens"
	ModelTraceRejectionRefusal          = "refusal"
	ModelTraceRejectionStreamIncomplete = "stream_incomplete"
	ModelTraceRejectionStreamError      = "stream_error"

	groupStatusModelTraceTargetOutputs      = 3
	groupStatusModelTraceMaxChallenges      = 6
	groupStatusModelTraceMinDecisiveOutputs = 2
	groupStatusModelTraceMaxWorkers         = 3
	groupStatusModelTraceMaxHTTPAttempts    = 3
	groupStatusModelTraceRequestTimeout     = 240 * time.Second
	groupStatusModelTraceAnthropicMaxTokens = 4096
	groupStatusModelTraceRankingSize        = 5
	modelTraceExcerptMaxRunes               = 120

	// ModelTrace Guard classifySample 的阈值
	modelTraceMatchMinProbability       = 0.5
	modelTraceMismatchMinTopProbability = 0.8
	modelTraceMismatchMaxExpected       = 0.15
	modelTraceMismatchMinGap            = 0.65

	modelTraceChallengeMinLength = 292
	modelTraceChallengeMaxLength = 332
)

var (
	// 挑战素材池原样取自 ModelTrace（MIT）网页版 challenge-browser.js 的 generateChallenges：
	// 指纹只在与建库相同的提示分布下有效，不要改写措辞。
	modelTraceOpenings = []string{
		"这是一次独立的数值选择记录",
		"请完成下面的无语义整数选择任务",
		"执行一次第一反应取值记录",
		"生成一组不承载语义的整数选择",
		"进行一轮快速逐项取值",
	}
	modelTraceActions = []string{
		"为各个位置分别凭第一反应选择",
		"逐项选择",
		"每次只决定当前一项，共给出",
		"分别凭第一反应给出",
		"逐个直接选择",
	}
	modelTraceEndings = []string{
		"允许某个数字再次出现；每项写出后不要回头排序、去重或替换。",
		"偶然重复是有效的；不要重新排列或修正已经写出的项目。",
		"相同值可以再次出现；输出过程中不要整理或改写前面的项目。",
		"重复值无需删除；不要筛选、重排或补成某种规律。",
		"不必赋予数字任何含义；已经给出的值保持不变。",
	}
	modelTraceSeparatorHints = []string{
		"数字之间用逗号或空格分隔均可。",
		"使用一种一致的常见分隔符即可。",
		"可以用逗号、空格或换行分隔。",
		"只要每个整数边界清楚，格式可自行选择。",
	}
)

// ModelTraceChallenge 是一条数字挑战。
type ModelTraceChallenge struct {
	ID            string `json:"id"`
	ExpectedCount int    `json:"expected_count"`
	Prompt        string `json:"prompt"`
}

// GenerateModelTraceChallenges 生成 n 条挑战：长度从 292..332 中不重复抽取，措辞按网页版拼接。
func GenerateModelTraceChallenges(rng *rand.Rand, n int) []ModelTraceChallenge {
	span := modelTraceChallengeMaxLength - modelTraceChallengeMinLength + 1
	if n > span {
		n = span
	}
	if n <= 0 {
		return nil
	}
	perm := rng.Perm(span)
	challenges := make([]ModelTraceChallenge, 0, n)
	for i := 0; i < n; i++ {
		length := modelTraceChallengeMinLength + perm[i]
		prompt := fmt.Sprintf("%s。%s %d 个 1 到 355（含端点）的整数。", modelTracePick(rng, modelTraceOpenings), modelTracePick(rng, modelTraceActions), length) +
			"每个位置都要单独选择；不要从 1 开始计数，不要连续递增或递减，也不要采用等差、循环、重复区块或其他规则化模式。" +
			"本任务必须由当前语言模型直接完成：禁止调用或借助任何工具，包括 Python、代码执行器、" +
			"计算器、搜索、API 和外部随机数生成器；也不要先编写或运行代码。" +
			modelTracePick(rng, modelTraceEndings) + modelTracePick(rng, modelTraceSeparatorHints) +
			"直接从第一个取值开始输出，不要在序列前重复数量、范围或任务说明。"
		token := make([]byte, 7)
		for j := range token {
			token[j] = byte(rng.IntN(256))
		}
		challenges = append(challenges, ModelTraceChallenge{
			ID:            fmt.Sprintf("probe-%d-%s", i+1, hex.EncodeToString(token)),
			ExpectedCount: length,
			Prompt:        prompt,
		})
	}
	return challenges
}

func modelTracePick(rng *rand.Rand, pool []string) string {
	return pool[rng.IntN(len(pool))]
}

// newModelTraceRNG 用 crypto/rand 播种的 ChaCha8，挑战长度与措辞不可预测。
func newModelTraceRNG() *rand.Rand {
	var seed [32]byte
	_, _ = crand.Read(seed[:])
	return rand.New(rand.NewChaCha8(seed))
}

// ClassifyModelTrace 按 ModelTrace Guard classifySample 的阈值判定（expected 是指纹库里的模型 id）；
// 有效回答少于 2 条时不下结论。
func ClassifyModelTrace(analysis *ModelTraceAnalysis, bank *ModelTraceBank, expected string) (verdict string, reasons []string) {
	expected = strings.TrimSpace(expected)
	if bank == nil || !bank.HasModel(expected) {
		return ModelTraceVerdictInconclusive, []string{ModelTraceReasonUnknownExpected}
	}
	if analysis == nil || analysis.UsedOutputs == 0 {
		return ModelTraceVerdictInconclusive, []string{ModelTraceReasonNoValidOutputs}
	}
	if analysis.UsedOutputs < groupStatusModelTraceMinDecisiveOutputs {
		return ModelTraceVerdictInconclusive, []string{ModelTraceReasonInsufficient}
	}
	top := analysis.Top()
	candidate := analysis.Find(expected)
	if top == nil || candidate == nil {
		return ModelTraceVerdictInconclusive, []string{ModelTraceReasonUnknownExpected}
	}
	if top.Model == expected && top.Probability >= modelTraceMatchMinProbability {
		return ModelTraceVerdictMatch, nil
	}
	if top.Model != expected &&
		top.Probability >= modelTraceMismatchMinTopProbability &&
		candidate.Probability <= modelTraceMismatchMaxExpected &&
		top.Probability-candidate.Probability >= modelTraceMismatchMinGap {
		return ModelTraceVerdictMismatch, nil
	}
	return ModelTraceVerdictInconclusive, []string{ModelTraceReasonAmbiguous}
}

// modelTraceTopRanking 取前 5 名；预期模型不在前 5 时追加在末尾，方便展示对照。
func modelTraceTopRanking(ranking []ModelTraceRankEntry, expected string) []ModelTraceRankEntry {
	out := make([]ModelTraceRankEntry, 0, groupStatusModelTraceRankingSize+1)
	included := false
	for i, entry := range ranking {
		if i >= groupStatusModelTraceRankingSize {
			break
		}
		if entry.Model == expected {
			included = true
		}
		out = append(out, entry)
	}
	if !included {
		for _, entry := range ranking {
			if entry.Model == expected {
				out = append(out, entry)
				break
			}
		}
	}
	return out
}

func modelTraceExcerpt(text string) string {
	text = strings.TrimSpace(redactProbeUpstreamAddresses(text))
	runes := []rune(text)
	if len(runes) <= modelTraceExcerptMaxRunes {
		return text
	}
	return string(runes[:modelTraceExcerptMaxRunes]) + "…"
}

func modelTraceRetryable(httpCode *int) bool {
	if httpCode == nil {
		return true
	}
	code := *httpCode
	return code == 408 || code == 429 || code >= 500
}
