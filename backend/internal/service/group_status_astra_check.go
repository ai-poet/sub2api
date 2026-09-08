package service

import (
	"fmt"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// Astra 指纹验证（meow 基准，行为指纹），本 fork 自有功能。
//
// 与 Sol Juice 并列的第三条探针：每次向分组的一个 OpenAI 账号发一批固定短答题，
// 把答案分布与基准包里 Astra / Sol / Terra / Luna 的分布比较，唯一越过阈值的模型
// 才算强指向。判定逻辑按 meow 技术报告的公式自行实现，不引用其代码。

const (
	GroupStatusEventAstraMismatch  = "astra_mismatch"
	GroupStatusEventAstraRecovered = "astra_recovered"

	AstraCheckVerdictMatch        = "match"
	AstraCheckVerdictMismatch     = "mismatch"
	AstraCheckVerdictInsufficient = "insufficient"

	AstraCheckStatusPass     = "pass"
	AstraCheckStatusMismatch = "mismatch"

	AstraCheckInvalidOutput = "__INVALID_OUTPUT__"
	AstraCheckOtherCategory = "__OTHER__"

	groupStatusAstraCheckDefaultRequestModel   = "gpt-6-astra"
	groupStatusAstraCheckDefaultTier           = AstraCheckTierLow
	groupStatusAstraCheckDefaultIntervalSecond = 3600
	groupStatusAstraCheckMinIntervalSeconds    = 900
	groupStatusAstraCheckDefaultConcurrency    = 8
	groupStatusAstraCheckRequestTimeout        = 120 * time.Second
	groupStatusAstraCheckMaxAttempts           = 3
	groupStatusAstraCheckSampleRatio           = 0.6
	groupStatusAstraCheckMismatchThreshold     = 2
	groupStatusAstraCheckRunBudget             = 20 * time.Minute

	// gpt-6-astra 官方价（USD/token），与 billing_service.go 兜底价一致；output 已含 reasoning。
	astraCheckInputPricePerToken  = 10e-6
	astraCheckOutputPricePerToken = 50e-6
)

var (
	ErrGroupStatusAstraCheckUnsupported = infraerrors.BadRequest("GROUP_STATUS_ASTRA_CHECK_UNSUPPORTED", "Astra fingerprint check is only available for OpenAI groups")
	ErrGroupStatusAstraCheckRunning     = infraerrors.Conflict("GROUP_STATUS_ASTRA_CHECK_RUNNING", "Astra fingerprint check is already running for this group")

	astraBehaviorLabelPattern = regexp.MustCompile(`^[a-z][a-z .'-]*$`)
	astraIntegerPattern       = regexp.MustCompile(`^[+-]?\d+$`)
)

// AstraCheckModelMatch 是某个候选模型在一次运行里的得分。
type AstraCheckModelMatch struct {
	Model     string  `json:"model"`
	Name      string  `json:"name"`
	Score     float64 `json:"score"`
	Match     float64 `json:"match"`
	Threshold float64 `json:"threshold"`
	Passed    bool    `json:"passed"`
}

// AstraCheckCellSummary 是某道题在一次运行里的样本情况。
type AstraCheckCellSummary struct {
	CellID     string         `json:"cell_id"`
	FamilyID   string         `json:"family_id"`
	Planned    int            `json:"planned"`
	Total      int            `json:"total"`
	Valid      int            `json:"valid"`
	Invalid    int            `json:"invalid"`
	Minimum    int            `json:"minimum"`
	Weight     float64        `json:"weight"`
	Categories map[string]int `json:"categories"`
}

// AstraCheckScore 是一次运行的判定结果。
type AstraCheckScore struct {
	Verdict        string                  `json:"verdict"`
	Winner         string                  `json:"winner"`
	Matches        []AstraCheckModelMatch  `json:"matches"`
	Reasons        []string                `json:"reasons"`
	Cells          []AstraCheckCellSummary `json:"cells"`
	ValidSamples   int                     `json:"valid_samples"`
	PlannedSamples int                     `json:"planned_samples"`
}

// AstraCellObservation 是某道题的原始归一化计数（含无效输出）。
type AstraCellObservation struct {
	CellID  string
	Planned int
	Counts  map[string]int
}

// AstraJob 是一次运行里的一个请求任务。
type AstraJob struct {
	CellID string
	Index  int
}

// GroupStatusAstraCheckResult 是落库前的一次运行结果。
type GroupStatusAstraCheckResult struct {
	GroupID            int64
	ConfigID           int64
	BenchmarkPackageID string
	BenchmarkVersion   string
	BenchmarkSHA256    string
	RequestModel       string
	Tier               string
	AccountID          *int64
	Verdict            string
	Winner             string
	Matches            []AstraCheckModelMatch
	Cells              []AstraCheckCellSummary
	Reasons            []string
	Samples            []AstraCheckSampleRecord
	RequestsPlanned    int
	RequestsCompleted  int
	ValidSamples       int
	PlannedSamples     int
	InputTokens        int64
	OutputTokens       int64
	ReasoningTokens    int64
	LatencyMS          *int64
	HTTPCode           *int
	ErrorDetail        string
	StartedAt          time.Time
	FinishedAt         time.Time
}

// GroupStatusAstraCheckRun 是落库后的运行记录。
type GroupStatusAstraCheckRun struct {
	ID                 int64                    `json:"id"`
	GroupID            int64                    `json:"group_id"`
	ConfigID           int64                    `json:"config_id"`
	BenchmarkPackageID string                   `json:"benchmark_package_id"`
	BenchmarkVersion   string                   `json:"benchmark_version"`
	BenchmarkSHA256    string                   `json:"benchmark_sha256"`
	RequestModel       string                   `json:"request_model"`
	Tier               string                   `json:"tier"`
	AccountID          *int64                   `json:"account_id"`
	Verdict            string                   `json:"verdict"`
	Winner             string                   `json:"winner_model"`
	Matches            []AstraCheckModelMatch   `json:"matches"`
	Cells              []AstraCheckCellSummary  `json:"cells"`
	Reasons            []string                 `json:"reasons"`
	Samples            []AstraCheckSampleRecord `json:"samples"`
	RequestsPlanned    int                      `json:"requests_planned"`
	RequestsCompleted  int                      `json:"requests_completed"`
	ValidSamples       int                      `json:"valid_samples"`
	InputTokens        int64                    `json:"input_tokens"`
	OutputTokens       int64                    `json:"output_tokens"`
	ReasoningTokens    int64                    `json:"reasoning_tokens"`
	LatencyMS          *int64                   `json:"latency_ms"`
	HTTPCode           *int                     `json:"http_code"`
	ErrorDetail        string                   `json:"error_detail"`
	StartedAt          time.Time                `json:"started_at"`
	FinishedAt         time.Time                `json:"finished_at"`
	CreatedAt          time.Time                `json:"created_at"`
}

type GroupStatusAstraCheckExecution struct {
	Group   *Group                       `json:"group,omitempty"`
	Config  *GroupStatusConfig           `json:"config,omitempty"`
	Account *Account                     `json:"account,omitempty"`
	Result  *GroupStatusAstraCheckResult `json:"result,omitempty"`
	Run     *GroupStatusAstraCheckRun    `json:"run,omitempty"`
	State   *GroupStatusState            `json:"state,omitempty"`
	Event   *GroupStatusEvent            `json:"event,omitempty"`
	// Confirmed 表示本次是首次 mismatch 后的立即复测
	Confirmed bool `json:"confirmed"`
}

// NormalizeAstraAnswer 按基准包指定的归一器把模型回复整理成类别；不合规则返回 __INVALID_OUTPUT__。
func NormalizeAstraAnswer(n AstraNormalizer, raw string) string {
	text := strings.TrimSpace(raw)
	if text == "" {
		return AstraCheckInvalidOutput
	}
	maxLength := n.MaxLength
	if maxLength <= 0 {
		maxLength = 128
	}
	switch n.ID {
	case "exact_trimmed":
		// 只去首尾空白
	case "exact_trimmed_casefold", "":
		text = strings.ToLower(text)
	case "whitespace_collapse":
		text = strings.Join(strings.Fields(text), " ")
	case "integer", "b80_exact_3":
		if utf8Len(text) > 128 || !astraIntegerPattern.MatchString(text) {
			return AstraCheckInvalidOutput
		}
		value, err := strconv.ParseInt(text, 10, 64)
		if err != nil {
			return AstraCheckInvalidOutput
		}
		text = strconv.FormatInt(value, 10)
		if n.ID == "b80_exact_3" {
			if text == "3" {
				text = "exact_3"
			} else {
				text = "other_integer"
			}
		}
	case "behavior_label":
		text = strings.Trim(text, "`\"'.,:;!?()[]{} ")
		text = strings.ToLower(text)
		text = strings.Join(strings.Fields(text), " ")
		if !astraBehaviorLabelPattern.MatchString(text) {
			return AstraCheckInvalidOutput
		}
	case "fixed_enum":
		key := strings.ToLower(text)
		mapped := ""
		for candidate, value := range n.Values {
			if strings.ToLower(strings.TrimSpace(candidate)) == key {
				mapped = value
				break
			}
		}
		if mapped == "" {
			return AstraCheckOtherCategory
		}
		text = mapped
	default:
		text = strings.ToLower(text)
	}
	if text == "" || utf8Len(text) > maxLength {
		return AstraCheckInvalidOutput
	}
	return text
}

func utf8Len(s string) int {
	return len([]rune(s))
}

// PlanAstraJobs 按档位生成请求任务，题目轮转派发（cell1, cell2, …, cell1, …）。
func PlanAstraJobs(bench *AstraBenchmark, tier string) ([]AstraJob, error) {
	if bench == nil {
		return nil, fmt.Errorf("astra benchmark is nil")
	}
	tierCfg, ok := bench.Tiers[tier]
	if !ok {
		return nil, fmt.Errorf("unsupported astra tier %q", tier)
	}
	maxCount := 0
	for _, cell := range bench.Cells {
		if n := tierCfg.Counts[cell.ID]; n > maxCount {
			maxCount = n
		}
	}
	jobs := make([]AstraJob, 0, tierCfg.TotalRequests)
	for round := 0; round < maxCount; round++ {
		for _, cell := range bench.Cells {
			if round < tierCfg.Counts[cell.ID] {
				jobs = append(jobs, AstraJob{CellID: cell.ID, Index: round})
			}
		}
	}
	return jobs, nil
}

// ScoreAstraCheck 用基准包对一次运行的计数做判定（纯函数）。
func ScoreAstraCheck(bench *AstraBenchmark, tier string, observations []AstraCellObservation) (*AstraCheckScore, error) {
	if bench == nil {
		return nil, fmt.Errorf("astra benchmark is nil")
	}
	tierCfg, ok := bench.Tiers[tier]
	if !ok {
		return nil, fmt.Errorf("unsupported astra tier %q", tier)
	}
	models := bench.ModelIDs
	reasons := make(map[string]struct{})
	if !tierCfg.Calibrated {
		reasons["uncalibrated"] = struct{}{}
	}

	observed := make(map[string]AstraCellObservation, len(observations))
	for _, obs := range observations {
		observed[obs.CellID] = obs
	}

	type cellLikelihood struct {
		family string
		weight float64
		values map[string]float64
	}
	var likelihoods []cellLikelihood
	score := &AstraCheckScore{}
	for _, cell := range bench.Cells {
		planned := tierCfg.Counts[cell.ID]
		if planned == 0 {
			continue
		}
		fitted, ok := bench.Fitted[cell.ID]
		if !ok || !fitted.ReferenceReady {
			reasons["baseline_cell_missing"] = struct{}{}
			continue
		}
		allowed := make(map[string]struct{}, len(fitted.Categories))
		for _, category := range fitted.Categories {
			allowed[category] = struct{}{}
		}
		counts := make(map[string]int)
		if obs, ok := observed[cell.ID]; ok {
			for category, count := range obs.Counts {
				if count <= 0 {
					continue
				}
				if _, ok := allowed[category]; ok || category == AstraCheckInvalidOutput {
					counts[category] += count
				} else {
					counts[AstraCheckOtherCategory] += count
				}
			}
		}
		total := 0
		for _, count := range counts {
			total += count
		}
		invalid := counts[AstraCheckInvalidOutput]
		valid := total - invalid
		minimum := int(math.Ceil(float64(planned) * groupStatusAstraCheckSampleRatio))
		if valid < minimum {
			reasons["samples_incomplete"] = struct{}{}
		}
		if total > planned {
			reasons["samples_exceed_plan"] = struct{}{}
		}
		score.Cells = append(score.Cells, AstraCheckCellSummary{
			CellID:     cell.ID,
			FamilyID:   fitted.FamilyID,
			Planned:    planned,
			Total:      total,
			Valid:      valid,
			Invalid:    invalid,
			Minimum:    minimum,
			Weight:     fitted.Weight,
			Categories: counts,
		})
		score.ValidSamples += valid
		score.PlannedSamples += planned
		if valid == 0 {
			continue
		}
		values := make(map[string]float64, len(models))
		for _, model := range models {
			dist := fitted.Distributions[model]
			sum := 0.0
			for category, count := range counts {
				if category == AstraCheckInvalidOutput || count == 0 {
					continue
				}
				p := dist[category]
				if p <= 0 {
					p = 1e-12
				}
				sum += float64(count) * math.Log(p)
			}
			values[model] = sum / float64(valid)
		}
		if fitted.Weight > 0 {
			likelihoods = append(likelihoods, cellLikelihood{family: fitted.FamilyID, weight: fitted.Weight, values: values})
		}
	}

	// 题族聚合：F(f,m) = max(w) × Σ w·L / Σ w；score(m) = Σ_f F
	families := make(map[string][]cellLikelihood)
	for _, item := range likelihoods {
		families[item.family] = append(families[item.family], item)
	}
	scores := make(map[string]float64, len(models))
	if len(families) == 0 {
		reasons["no_weighted_family"] = struct{}{}
	}
	familyIDs := make([]string, 0, len(families))
	for family := range families {
		familyIDs = append(familyIDs, family)
	}
	sort.Strings(familyIDs)
	for _, family := range familyIDs {
		entries := families[family]
		weightSum, weightMax := 0.0, 0.0
		for _, entry := range entries {
			weightSum += entry.weight
			if entry.weight > weightMax {
				weightMax = entry.weight
			}
		}
		for _, model := range models {
			acc := 0.0
			for _, entry := range entries {
				acc += entry.weight * entry.values[model]
			}
			scores[model] += weightMax * acc / weightSum
		}
	}

	// softmax（减最大值保证数值稳定）
	maxScore := math.Inf(-1)
	for _, model := range models {
		if scores[model] > maxScore {
			maxScore = scores[model]
		}
	}
	if math.IsInf(maxScore, -1) {
		maxScore = 0
	}
	expSum := 0.0
	exps := make(map[string]float64, len(models))
	for _, model := range models {
		exps[model] = math.Exp(scores[model] - maxScore)
		expSum += exps[model]
	}
	reasonList := make([]string, 0, len(reasons))
	for reason := range reasons {
		reasonList = append(reasonList, reason)
	}
	sort.Strings(reasonList)

	var winners []string
	for _, model := range models {
		match := 0.0
		if expSum > 0 {
			match = exps[model] / expSum
		}
		threshold := tierCfg.Thresholds[model]
		passed := len(reasonList) == 0 && match > threshold
		if passed {
			winners = append(winners, model)
		}
		score.Matches = append(score.Matches, AstraCheckModelMatch{
			Model:     model,
			Name:      bench.modelName(model),
			Score:     scores[model],
			Match:     match,
			Threshold: threshold,
			Passed:    passed,
		})
	}
	if len(reasonList) == 0 && len(winners) != 1 {
		if len(winners) == 0 {
			reasonList = append(reasonList, "no_threshold")
		} else {
			reasonList = append(reasonList, "multiple_thresholds")
		}
	}
	score.Reasons = reasonList
	switch {
	case len(winners) == 1 && winners[0] == astraCheckClaimedModel:
		score.Verdict = AstraCheckVerdictMatch
		score.Winner = winners[0]
	case len(winners) == 1:
		score.Verdict = AstraCheckVerdictMismatch
		score.Winner = winners[0]
	default:
		score.Verdict = AstraCheckVerdictInsufficient
	}
	return score, nil
}

// ComputeAstraCheckTransition 是纯函数：把一次运行并进状态，稳定结论切换时产出事件。
//
//   - match：清零计数，稳定置 pass；原为 mismatch 则发 astra_recovered
//   - mismatch：计数 +1；原来不是 mismatch 且计数达到阈值（2）时置 mismatch 并发 astra_mismatch
//   - insufficient：只更新最近一次结果，不动稳定结论与计数
//
// 存活探测与 Sol Juice 的字段在这里不会被改动。
func ComputeAstraCheckTransition(prev *GroupStatusState, result *GroupStatusAstraCheckResult, runID int64) (*GroupStatusState, *GroupStatusEvent) {
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
	next.AstraCheckVerdict = result.Verdict
	next.AstraCheckWinner = result.Winner
	next.AstraCheckMatches = append([]AstraCheckModelMatch(nil), result.Matches...)
	next.AstraCheckReasons = append([]string(nil), result.Reasons...)
	next.AstraCheckDetail = astraCheckDetailText(result)
	next.AstraCheckCheckedAt = &observedAt
	next.AstraCheckValidSamples = result.ValidSamples
	next.AstraCheckPlannedSamples = result.PlannedSamples
	next.AstraCheckInputTokens = result.InputTokens
	next.AstraCheckOutputTokens = result.OutputTokens
	next.AstraCheckReasoningTokens = result.ReasoningTokens
	if runID > 0 {
		id := runID
		next.AstraCheckLastRunID = &id
	}

	prevStable := strings.TrimSpace(next.AstraCheckStableStatus)
	newEvent := func(eventType, from, to string) *GroupStatusEvent {
		subStatus := "winner_unknown"
		if result.Winner != "" {
			subStatus = "winner_" + result.Winner
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
			ErrorDetail: astraCheckDetailText(result),
			ObservedAt:  observedAt,
		}
	}

	switch result.Verdict {
	case AstraCheckVerdictMatch:
		next.AstraCheckConsecutiveMismatch = 0
		next.AstraCheckStableStatus = AstraCheckStatusPass
		if prevStable == AstraCheckStatusMismatch {
			return next, newEvent(GroupStatusEventAstraRecovered, prevStable, AstraCheckStatusPass)
		}
		return next, nil
	case AstraCheckVerdictMismatch:
		next.AstraCheckConsecutiveMismatch++
		if prevStable == AstraCheckStatusMismatch {
			return next, nil
		}
		if next.AstraCheckConsecutiveMismatch >= groupStatusAstraCheckMismatchThreshold {
			next.AstraCheckStableStatus = AstraCheckStatusMismatch
			return next, newEvent(GroupStatusEventAstraMismatch, prevStable, AstraCheckStatusMismatch)
		}
		return next, nil
	default:
		return next, nil
	}
}

// astraCheckDetailText 生成一行可读摘要：各模型匹配度/阈值、样本数、基准版本、错误。
func astraCheckDetailText(result *GroupStatusAstraCheckResult) string {
	if result == nil {
		return ""
	}
	parts := make([]string, 0, len(result.Matches)+3)
	for _, match := range result.Matches {
		parts = append(parts, fmt.Sprintf("%s %.3f/%.3f", astraModelShortName(match.Model), match.Match, match.Threshold))
	}
	parts = append(parts, fmt.Sprintf("valid %d/%d", result.ValidSamples, result.PlannedSamples))
	if result.BenchmarkVersion != "" {
		parts = append(parts, "benchmark "+result.BenchmarkVersion)
	}
	if len(result.Reasons) > 0 {
		parts = append(parts, "reasons "+strings.Join(result.Reasons, ","))
	}
	if detail := strings.TrimSpace(result.ErrorDetail); detail != "" {
		parts = append(parts, detail)
	}
	return truncateProbeText(strings.Join(parts, " · "))
}

// EstimateAstraCheckCostUSD 按 gpt-6-astra 标准价估算一次运行的费用。
func EstimateAstraCheckCostUSD(inputTokens, outputTokens int64) float64 {
	if inputTokens <= 0 && outputTokens <= 0 {
		return 0
	}
	cost := 0.0
	if inputTokens > 0 {
		cost += float64(inputTokens) * astraCheckInputPricePerToken
	}
	if outputTokens > 0 {
		cost += float64(outputTokens) * astraCheckOutputPricePerToken
	}
	return cost
}

// decorateAstraCheckSummary 填充派生字段：成本估算与内置基准元数据（不落库）。
func decorateAstraCheckSummary(summary *GroupStatusSummary) {
	if summary == nil {
		return
	}
	summary.AstraCheckLastCostUSD = EstimateAstraCheckCostUSD(summary.AstraCheckInputTokens, summary.AstraCheckOutputTokens)
	// 没有状态行时这些切片是 nil，会序列化成 null；前端直接读 .length，必须给空数组
	if summary.AstraCheckMatches == nil {
		summary.AstraCheckMatches = []AstraCheckModelMatch{}
	}
	if summary.AstraCheckReasons == nil {
		summary.AstraCheckReasons = []string{}
	}
	summary.AstraCheckBenchmarkModels = []AstraBenchmarkModel{}
	summary.AstraCheckBenchmarkTiers = []AstraBenchmarkTierMeta{}
	if _, meta, err := LoadEmbeddedAstraBenchmark(); err == nil && meta != nil {
		summary.AstraCheckBenchmarkVersion = meta.Version
		summary.AstraCheckBenchmarkModels = append([]AstraBenchmarkModel(nil), meta.Models...)
		summary.AstraCheckBenchmarkTiers = append([]AstraBenchmarkTierMeta(nil), meta.Tiers...)
	}
}

// astraModelShortName 把候选模型 id 缩成可读短名。
func astraModelShortName(model string) string {
	switch strings.TrimSpace(model) {
	case "gpt-6-astra", "gpt-6":
		return "Astra"
	case "gpt-5.6-sol":
		return "Sol"
	case "gpt-5.6-terra":
		return "Terra"
	case "gpt-5.6-luna":
		return "Luna"
	case "":
		return "?"
	default:
		return strings.TrimSpace(model)
	}
}

// astraWinnerFromEvent 从事件 sub_status（winner_<model>）里取出强指向模型的短名。
func astraWinnerFromEvent(event *GroupStatusEvent) string {
	if event == nil {
		return "?"
	}
	winner := strings.TrimPrefix(strings.TrimSpace(event.SubStatus), "winner_")
	if winner == "" || winner == "unknown" {
		return "?"
	}
	return astraModelShortName(winner)
}

// isAstraCheckEvent 报告事件是否来自 Astra 指纹验证。
func isAstraCheckEvent(eventType string) bool {
	return eventType == GroupStatusEventAstraMismatch || eventType == GroupStatusEventAstraRecovered
}

// astraCheckStatusLabel 是 Astra 事件里 from/to 的中文标签。
func astraCheckStatusLabel(status string) string {
	switch strings.TrimSpace(status) {
	case AstraCheckStatusPass:
		return "Astra 指纹正常"
	case AstraCheckStatusMismatch:
		return "Astra 指纹不符"
	default:
		return "未知"
	}
}
