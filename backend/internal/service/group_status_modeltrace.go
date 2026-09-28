package service

import (
	crand "crypto/rand"
	"encoding/hex"
	"fmt"
	"math/rand/v2"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// ModelTrace 指纹验证（数字分布指纹），本 fork 自有功能，替换原来的纯 Sol 验证（Juice）。
//
// 每次运行向分组的一个账号发 3 条（最多 6 条）「凭第一反应逐项输出 292–332 个 1..355 整数」的
// 挑战，用 ModelTrace（MIT）统一指纹库对库内全部模型做闭集归因，再按 ModelTrace Guard 的
// classifySample 规则判定是否与管理员选定的预期模型一致。挑战措辞原样沿用 ModelTrace，
// 评分、判定、状态机与执行是本仓库的实现。

const (
	GroupStatusEventModelTraceMismatch  = "modeltrace_mismatch"
	GroupStatusEventModelTraceRecovered = "modeltrace_recovered"

	ModelTraceVerdictMatch        = "match"
	ModelTraceVerdictMismatch     = "mismatch"
	ModelTraceVerdictInconclusive = "inconclusive"

	ModelTraceStatusPass     = "pass"
	ModelTraceStatusMismatch = "mismatch"

	// Outcome 沿用 ModelTrace Guard 的命名
	ModelTraceOutcomeCompatible       = "compatible"
	ModelTraceOutcomeDifferenceSignal = "difference_signal"
	ModelTraceOutcomeInconclusive     = "inconclusive"

	ModelTraceReasonUnknownExpected  = "unknown_expected_model"
	ModelTraceReasonNoValidOutputs   = "no_valid_outputs"
	ModelTraceReasonInsufficient     = "insufficient_outputs"
	ModelTraceReasonAmbiguous        = "ambiguous"
	ModelTraceReasonTargetNotAllowed = "target_not_allowed"
	ModelTraceReasonBankInvalid      = "bank_invalid"
	ModelTraceReasonNoAccount        = "no_account"

	groupStatusModelTraceDefaultIntervalSeconds = 3600
	groupStatusModelTraceMinIntervalSeconds     = 900
	groupStatusModelTraceTargetOutputs          = 3
	groupStatusModelTraceMaxChallenges          = 6
	groupStatusModelTraceMinDecisiveOutputs     = 2
	groupStatusModelTraceMismatchThreshold      = 2
	groupStatusModelTraceDefaultConcurrency     = 3
	groupStatusModelTraceMaxHTTPAttempts        = 3
	groupStatusModelTraceRequestTimeout         = 240 * time.Second
	groupStatusModelTraceRunBudget              = 20 * time.Minute
	groupStatusModelTraceAnthropicMaxTokens     = 4096
	groupStatusModelTraceRankingSize            = 5

	// ModelTrace Guard classifySample 的阈值
	modelTraceMatchMinProbability       = 0.5
	modelTraceMismatchMinTopProbability = 0.8
	modelTraceMismatchMaxExpected       = 0.15
	modelTraceMismatchMinGap            = 0.65

	modelTraceChallengeMinLength = 292
	modelTraceChallengeMaxLength = 332
)

var (
	ErrGroupStatusModelTraceUnsupported   = infraerrors.BadRequest("GROUP_STATUS_MODELTRACE_UNSUPPORTED", "ModelTrace fingerprint check is only available for OpenAI and Anthropic groups")
	ErrGroupStatusModelTraceRunning       = infraerrors.Conflict("GROUP_STATUS_MODELTRACE_RUNNING", "ModelTrace fingerprint check is already running for this group")
	ErrGroupStatusModelTraceTargetInvalid = infraerrors.BadRequest("GROUP_STATUS_MODELTRACE_TARGET_INVALID", "unsupported ModelTrace expected model for this group")

	// modelTraceTargetsByPlatform 是各平台可选的预期模型；每个目标都必须在内置指纹库中（测试钉住）。
	modelTraceTargetsByPlatform = map[string][]string{
		PlatformOpenAI:    {"gpt-5.6-sol", "gpt-6-sol", "gpt-6-astra"},
		PlatformAnthropic: {"claude-opus-5-5"},
	}

	modelTraceModelLabels = map[string]string{
		"gpt-5.4":                   "GPT-5.4",
		"gpt-5.5":                   "GPT-5.5",
		"gpt-5.6-sol":               "GPT-5.6 Sol",
		"gpt-5.6-terra":             "GPT-5.6 Terra",
		"gpt-5.6-luna":              "GPT-5.6 Luna",
		"gpt-6-astra":               "GPT-6 Astra",
		"gpt-6-sol":                 "GPT-6 Sol",
		"gpt-6-luna":                "GPT-6 Luna",
		"claude-haiku-4-5-20251001": "Claude Haiku 4.5",
		"claude-sonnet-4-6":         "Claude Sonnet 4.6",
		"claude-sonnet-5":           "Claude Sonnet 5",
		"claude-opus-4-6":           "Claude Opus 4.6",
		"claude-opus-4-7":           "Claude Opus 4.7",
		"claude-opus-4-8":           "Claude Opus 4.8",
		"claude-opus-5":             "Claude Opus 5",
		"claude-opus-5-5":           "Claude Opus 5.5",
	}

	// 挑战素材池原样取自 ModelTrace fingerprint.py generate_challenges（MIT）：
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

// ModelTraceTarget 是某个平台可选的预期模型。
type ModelTraceTarget struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
}

// ModelTraceChallenge 是一条数字挑战。
type ModelTraceChallenge struct {
	ID            string `json:"id"`
	ExpectedCount int    `json:"expected_count"`
	Prompt        string `json:"prompt"`
}

// ModelTraceOutputRecord 是一次挑战的诊断记录（只落在管理端可见的运行表里）。
type ModelTraceOutputRecord struct {
	Seq             int     `json:"seq"`
	ChallengeID     string  `json:"challenge_id"`
	Prompt          string  `json:"prompt"`
	ExpectedCount   int     `json:"expected_count"`
	MinimumNumbers  int     `json:"minimum_numbers"`
	ParsedNumbers   int     `json:"parsed_numbers"`
	Accepted        bool    `json:"accepted"`
	Rejection       string  `json:"rejection,omitempty"`
	StopReason      string  `json:"stop_reason,omitempty"`
	ThinkingPresent bool    `json:"thinking_present"`
	HTTPCode        *int    `json:"http_code,omitempty"`
	Attempts        int     `json:"attempts"`
	LatencyMS       int64   `json:"latency_ms"`
	InputTokens     int64   `json:"input_tokens"`
	OutputTokens    int64   `json:"output_tokens"`
	ReasoningTokens int64   `json:"reasoning_tokens"`
	Error           string  `json:"error,omitempty"`
	Excerpt         string  `json:"excerpt,omitempty"`
	Numbers         []int   `json:"numbers,omitempty"`
	TopModel        string  `json:"top_model,omitempty"`
	TopProbability  float64 `json:"top_probability,omitempty"`

	// transportFailed：最后一次尝试仍是传输 / 非 2xx 错误（决定是否换号），不落库
	transportFailed bool
}

// GroupStatusModelTraceResult 是落库前的一次运行结果。
type GroupStatusModelTraceResult struct {
	GroupID             int64
	ConfigID            int64
	Platform            string
	BankSHA256          string
	BankBuiltAt         string
	ExpectedModel       string
	RequestModel        string
	AccountID           *int64
	AccountType         string
	Round               int
	Verdict             string
	Outcome             string
	TopModel            string
	TopProbability      float64
	ExpectedProbability *float64
	CalibrationQueries  int
	Beta                float64
	Ranking             []ModelTraceRankEntry
	FamilyProbabilities []ModelTraceFamilyProbability
	Reasons             []string
	Outputs             []ModelTraceOutputRecord
	AttemptsPlanned     int
	AttemptsMade        int
	ValidOutputs        int
	InputTokens         int64
	OutputTokens        int64
	ReasoningTokens     int64
	CostUSD             float64
	LatencyMS           *int64
	HTTPCode            *int
	ErrorDetail         string
	StartedAt           time.Time
	FinishedAt          time.Time
}

// GroupStatusModelTraceRun 是落库后的运行记录。
type GroupStatusModelTraceRun struct {
	ID                  int64                         `json:"id"`
	GroupID             int64                         `json:"group_id"`
	ConfigID            int64                         `json:"config_id"`
	Platform            string                        `json:"platform"`
	BankSHA256          string                        `json:"bank_sha256"`
	BankBuiltAt         string                        `json:"bank_built_at"`
	ExpectedModel       string                        `json:"expected_model"`
	RequestModel        string                        `json:"request_model"`
	AccountID           *int64                        `json:"account_id"`
	AccountType         string                        `json:"account_type"`
	Round               int                           `json:"round"`
	Verdict             string                        `json:"verdict"`
	Outcome             string                        `json:"outcome"`
	TopModel            string                        `json:"top_model"`
	TopProbability      float64                       `json:"top_probability"`
	ExpectedProbability *float64                      `json:"expected_probability"`
	CalibrationQueries  int                           `json:"calibration_queries"`
	Beta                float64                       `json:"beta"`
	Ranking             []ModelTraceRankEntry         `json:"ranking"`
	FamilyProbabilities []ModelTraceFamilyProbability `json:"family_probabilities"`
	Reasons             []string                      `json:"reasons"`
	Outputs             []ModelTraceOutputRecord      `json:"outputs"`
	AttemptsPlanned     int                           `json:"attempts_planned"`
	AttemptsMade        int                           `json:"attempts_made"`
	ValidOutputs        int                           `json:"valid_outputs"`
	InputTokens         int64                         `json:"input_tokens"`
	OutputTokens        int64                         `json:"output_tokens"`
	ReasoningTokens     int64                         `json:"reasoning_tokens"`
	CostUSD             float64                       `json:"cost_usd"`
	LatencyMS           *int64                        `json:"latency_ms"`
	HTTPCode            *int                          `json:"http_code"`
	ErrorDetail         string                        `json:"error_detail"`
	StartedAt           time.Time                     `json:"started_at"`
	FinishedAt          time.Time                     `json:"finished_at"`
	CreatedAt           time.Time                     `json:"created_at"`
}

type GroupStatusModelTraceExecution struct {
	Group   *Group                       `json:"group,omitempty"`
	Config  *GroupStatusConfig           `json:"config,omitempty"`
	Account *Account                     `json:"account,omitempty"`
	Result  *GroupStatusModelTraceResult `json:"result,omitempty"`
	Run     *GroupStatusModelTraceRun    `json:"run,omitempty"`
	State   *GroupStatusState            `json:"state,omitempty"`
	Event   *GroupStatusEvent            `json:"event,omitempty"`
	// Confirmed 表示本次是首次 mismatch 后在同一账号上的立即复测
	Confirmed bool `json:"confirmed"`
}

// ModelTraceTargetsForPlatform 返回某平台可选的预期模型（无则为空）。
func ModelTraceTargetsForPlatform(platform string) []ModelTraceTarget {
	ids := modelTraceTargetsByPlatform[platform]
	out := make([]ModelTraceTarget, 0, len(ids))
	for _, id := range ids {
		out = append(out, ModelTraceTarget{ID: id, DisplayName: ModelTraceModelLabel(id)})
	}
	return out
}

// modelTraceSupportsPlatform 报告该平台的分组能否开启 ModelTrace 指纹验证。
func modelTraceSupportsPlatform(platform string) bool {
	return len(modelTraceTargetsByPlatform[platform]) > 0
}

// modelTraceDefaultExpected 是某平台的默认预期模型；不支持的平台为空。
func modelTraceDefaultExpected(platform string) string {
	if ids := modelTraceTargetsByPlatform[platform]; len(ids) > 0 {
		return ids[0]
	}
	return ""
}

// modelTraceTargetAllowed 报告预期模型是否在该平台的目标列表里。
func modelTraceTargetAllowed(platform, model string) bool {
	for _, id := range modelTraceTargetsByPlatform[platform] {
		if id == model {
			return true
		}
	}
	return false
}

// modelTraceKnownTarget 报告模型是否是任一平台的目标（平台未知时的宽松校验）。
func modelTraceKnownTarget(model string) bool {
	for _, ids := range modelTraceTargetsByPlatform {
		for _, id := range ids {
			if id == model {
				return true
			}
		}
	}
	return false
}

// ModelTraceModelLabel 把模型 id 转成可读名称；未知 id 原样返回。
func ModelTraceModelLabel(model string) string {
	model = strings.TrimSpace(model)
	if model == "" {
		return "?"
	}
	if label, ok := modelTraceModelLabels[model]; ok {
		return label
	}
	return model
}

// GenerateModelTraceChallenges 生成 n 条挑战：长度从 292..332 中不重复抽取，措辞按参考实现拼接。
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

// ClassifyModelTrace 按 ModelTrace Guard classifySample 的阈值判定；有效回答少于 2 条时不下结论。
func ClassifyModelTrace(analysis *ModelTraceAnalysis, bank *ModelTraceBank, expected string) (verdict, outcome string, reasons []string) {
	expected = strings.TrimSpace(expected)
	if bank == nil || !bank.HasModel(expected) {
		return ModelTraceVerdictInconclusive, ModelTraceOutcomeInconclusive, []string{ModelTraceReasonUnknownExpected}
	}
	if analysis == nil || analysis.UsedOutputs == 0 {
		return ModelTraceVerdictInconclusive, ModelTraceOutcomeInconclusive, []string{ModelTraceReasonNoValidOutputs}
	}
	if analysis.UsedOutputs < groupStatusModelTraceMinDecisiveOutputs {
		return ModelTraceVerdictInconclusive, ModelTraceOutcomeInconclusive, []string{ModelTraceReasonInsufficient}
	}
	top := analysis.Top()
	candidate := analysis.Find(expected)
	if top == nil || candidate == nil {
		return ModelTraceVerdictInconclusive, ModelTraceOutcomeInconclusive, []string{ModelTraceReasonUnknownExpected}
	}
	if top.Model == expected && top.Probability >= modelTraceMatchMinProbability {
		return ModelTraceVerdictMatch, ModelTraceOutcomeCompatible, nil
	}
	if top.Model != expected &&
		top.Probability >= modelTraceMismatchMinTopProbability &&
		candidate.Probability <= modelTraceMismatchMaxExpected &&
		top.Probability-candidate.Probability >= modelTraceMismatchMinGap {
		return ModelTraceVerdictMismatch, ModelTraceOutcomeDifferenceSignal, nil
	}
	return ModelTraceVerdictInconclusive, ModelTraceOutcomeInconclusive, []string{ModelTraceReasonAmbiguous}
}

// ComputeModelTraceTransition 是纯函数：把一次运行结果并进状态，并在稳定结论切换时产出事件。
//
//   - 预期模型与上次稳定结论对应的不同：先静默清零稳定结论与计数（不发事件）
//   - match：清零计数，稳定置 pass；原为 mismatch 则发 modeltrace_recovered
//   - mismatch：计数 +1；原来不是 mismatch 且计数达到阈值（2）时置 mismatch 并发 modeltrace_mismatch
//   - inconclusive：只更新最近一次结果，不动稳定结论与计数
//
// 存活探测与 Astra 的字段在这里不会被改动。
func ComputeModelTraceTransition(prev *GroupStatusState, result *GroupStatusModelTraceResult, runID int64) (*GroupStatusState, *GroupStatusEvent) {
	next := &GroupStatusState{}
	if prev != nil {
		*next = *prev
	}
	next.GroupID = result.GroupID
	if next.ConfigID == 0 {
		next.ConfigID = result.ConfigID
	}

	observedAt := result.FinishedAt
	if observedAt.IsZero() {
		observedAt = time.Now()
	}
	detail := modelTraceDetailText(result)
	next.ModelTraceVerdict = result.Verdict
	next.ModelTraceTopModel = result.TopModel
	next.ModelTraceTopProbability = result.TopProbability
	if result.ExpectedProbability != nil {
		p := *result.ExpectedProbability
		next.ModelTraceExpectedProbability = &p
	} else {
		next.ModelTraceExpectedProbability = nil
	}
	next.ModelTraceRanking = modelTraceTopRanking(result.Ranking, result.ExpectedModel)
	next.ModelTraceReasons = append([]string(nil), result.Reasons...)
	next.ModelTraceDetail = detail
	next.ModelTraceCheckedAt = &observedAt
	next.ModelTraceValidOutputs = result.ValidOutputs
	next.ModelTraceInputTokens = result.InputTokens
	next.ModelTraceOutputTokens = result.OutputTokens
	next.ModelTraceReasoningTokens = result.ReasoningTokens
	next.ModelTraceLastCostUSD = result.CostUSD
	if runID > 0 {
		id := runID
		next.ModelTraceLastRunID = &id
	}

	if prevExpected := strings.TrimSpace(next.ModelTraceRunExpectedModel); prevExpected != "" && prevExpected != result.ExpectedModel {
		next.ModelTraceStableStatus = ""
		next.ModelTraceConsecutiveMismatch = 0
	}
	next.ModelTraceRunExpectedModel = result.ExpectedModel

	prevStable := strings.TrimSpace(next.ModelTraceStableStatus)
	newEvent := func(eventType, from, to string) *GroupStatusEvent {
		subStatus := "top_unknown"
		if result.TopModel != "" {
			subStatus = "top_" + result.TopModel
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
			ErrorDetail: detail,
			ObservedAt:  observedAt,
		}
	}

	switch result.Verdict {
	case ModelTraceVerdictMatch:
		next.ModelTraceConsecutiveMismatch = 0
		next.ModelTraceStableStatus = ModelTraceStatusPass
		if prevStable == ModelTraceStatusMismatch {
			return next, newEvent(GroupStatusEventModelTraceRecovered, prevStable, ModelTraceStatusPass)
		}
		return next, nil
	case ModelTraceVerdictMismatch:
		next.ModelTraceConsecutiveMismatch++
		if prevStable == ModelTraceStatusMismatch {
			return next, nil
		}
		if next.ModelTraceConsecutiveMismatch >= groupStatusModelTraceMismatchThreshold {
			next.ModelTraceStableStatus = ModelTraceStatusMismatch
			return next, newEvent(GroupStatusEventModelTraceMismatch, prevStable, ModelTraceStatusMismatch)
		}
		return next, nil
	default:
		return next, nil
	}
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

// modelTraceDetailText 生成一行可读摘要；summary 会公开给用户，所以不含账号 id 与提示词。
func modelTraceDetailText(result *GroupStatusModelTraceResult) string {
	if result == nil {
		return ""
	}
	parts := make([]string, 0, 6)
	expected := ModelTraceModelLabel(result.ExpectedModel)
	if result.ExpectedProbability != nil {
		parts = append(parts, fmt.Sprintf("expected %s %.1f%%", expected, *result.ExpectedProbability*100))
	} else {
		parts = append(parts, "expected "+expected)
	}
	if result.TopModel != "" {
		parts = append(parts, fmt.Sprintf("top %s %.1f%%", ModelTraceModelLabel(result.TopModel), result.TopProbability*100))
	}
	parts = append(parts, fmt.Sprintf("outputs %d/%d (attempts %d/%d)", result.ValidOutputs, groupStatusModelTraceTargetOutputs, result.AttemptsMade, result.AttemptsPlanned))
	if sha := strings.TrimSpace(result.BankSHA256); sha != "" {
		if len(sha) > 8 {
			sha = sha[:8]
		}
		parts = append(parts, "bank "+sha)
	}
	if len(result.Reasons) > 0 {
		parts = append(parts, "reasons "+strings.Join(result.Reasons, ","))
	}
	if detail := strings.TrimSpace(result.ErrorDetail); detail != "" {
		parts = append(parts, detail)
	}
	return truncateProbeText(strings.Join(parts, " · "))
}

// decorateModelTraceSummary 填充派生字段：内置指纹库元数据与空切片（不落库）。
func decorateModelTraceSummary(summary *GroupStatusSummary) {
	if summary == nil {
		return
	}
	// 没有状态行时这些切片是 nil，会序列化成 null；前端直接读 .length，必须给空数组
	if summary.ModelTraceRanking == nil {
		summary.ModelTraceRanking = []ModelTraceRankEntry{}
	}
	if summary.ModelTraceReasons == nil {
		summary.ModelTraceReasons = []string{}
	}
	if _, meta, err := LoadEmbeddedModelTraceBank(); err == nil && meta != nil {
		summary.ModelTraceBankSHA256 = modelTraceShortSHA(meta.SHA256)
		summary.ModelTraceBankBuiltAt = meta.BuiltAt
	}
}

func modelTraceShortSHA(sha string) string {
	if len(sha) > 12 {
		return sha[:12]
	}
	return sha
}

// modelTraceTopFromEvent 从事件 sub_status（top_<model>）里取出第一候选的可读名。
func modelTraceTopFromEvent(event *GroupStatusEvent) string {
	if event == nil {
		return "?"
	}
	subStatus := strings.TrimSpace(event.SubStatus)
	top := strings.TrimPrefix(subStatus, "top_")
	if top == "" || top == "unknown" || top == subStatus {
		return "?"
	}
	return ModelTraceModelLabel(top)
}

// isModelTraceEvent 报告事件是否来自 ModelTrace 指纹验证。
func isModelTraceEvent(eventType string) bool {
	return eventType == GroupStatusEventModelTraceMismatch || eventType == GroupStatusEventModelTraceRecovered
}

// modelTraceStatusLabel 是 ModelTrace 事件里 from/to 的中文标签。
func modelTraceStatusLabel(status string) string {
	switch strings.TrimSpace(status) {
	case ModelTraceStatusPass:
		return "指纹一致"
	case ModelTraceStatusMismatch:
		return "指纹不符"
	default:
		return "未知"
	}
}
