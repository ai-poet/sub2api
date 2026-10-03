package service

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// meow 指纹验证（行为指纹，多模型），本 fork 自有功能；历史名称为 Astra 指纹验证，表与代码沿用 astra_check。
//
// 每个分组可以同时检测多个预期模型：对每个模型，向分组里一个与平台一致的账号发一批固定短答题，
// 把答案计数交给该模型所在的 v3 基准包判定——每个候选的证据是各题 Dirichlet-multinomial 边际似然之和，
// other 取整轮最接近的单一外部参考源；显示分数是「每条有效答案相对最强对手的对数优势」的 sigmoid，
// 只有唯一最高者严格越过自身强指向线、且样本满足资格时才算强指向。公式按 meow 技术报告自行实现，不引用其代码。

const (
	GroupStatusEventAstraMismatch  = "astra_mismatch"
	GroupStatusEventAstraRecovered = "astra_recovered"

	AstraCheckVerdictMatch        = "match"
	AstraCheckVerdictMismatch     = "mismatch"
	AstraCheckVerdictInsufficient = "insufficient"

	AstraCheckStatusPass     = "pass"
	AstraCheckStatusMismatch = "mismatch"

	AstraCheckInvalidOutput = "__INVALID_OUTPUT__"

	AstraCheckReasonSamplesIncomplete     = "samples_incomplete"
	AstraCheckReasonNoValidSamples        = "no_valid_samples"
	AstraCheckReasonNoStrongDirection     = "no_strong_direction"
	AstraCheckReasonUncalibrated          = "uncalibrated"
	AstraCheckReasonTargetNotAllowed      = "target_not_allowed"
	AstraCheckReasonTargetNotInBenchmark  = "target_not_in_benchmark"
	AstraCheckReasonBenchmarkInvalid      = "benchmark_invalid"
	AstraCheckReasonNoAccount             = "no_account"
	AstraCheckReasonScoringFailed         = "scoring_failed"
	astraCheckEventSubStatusWinnerPrefix  = ":winner_"
	astraCheckEventSubStatusUnknownWinner = "unknown"

	groupStatusAstraCheckDefaultTier           = AstraCheckTierMedium
	groupStatusAstraCheckDefaultIntervalSecond = 3600
	groupStatusAstraCheckMinIntervalSeconds    = 900
	groupStatusAstraCheckDefaultConcurrency    = 16
	groupStatusAstraCheckRequestTimeout        = 120 * time.Second
	groupStatusAstraCheckMaxAttempts           = 3
	groupStatusAstraCheckMismatchThreshold     = 2
	groupStatusAstraCheckRunBudget             = 20 * time.Minute
	groupStatusAstraCheckMaxModels             = 8
	// 一次运行里同时检测的模型数；各模型优先摊到不同账号上
	groupStatusAstraCheckModelParallelism = 4
)

var (
	ErrGroupStatusAstraCheckUnsupported   = infraerrors.BadRequest("GROUP_STATUS_ASTRA_CHECK_UNSUPPORTED", "meow fingerprint check is only available for OpenAI and Anthropic groups")
	ErrGroupStatusAstraCheckRunning       = infraerrors.Conflict("GROUP_STATUS_ASTRA_CHECK_RUNNING", "meow fingerprint check is already running for this group")
	ErrGroupStatusAstraCheckTargetInvalid = infraerrors.BadRequest("GROUP_STATUS_ASTRA_CHECK_TARGET_INVALID", "unsupported expected model for this group")
)

// AstraCheckModelConfig 是分组要检测的一个预期模型；RequestModel 为空表示用目标的默认请求模型名。
type AstraCheckModelConfig struct {
	ExpectedModel string `json:"expected_model"`
	RequestModel  string `json:"request_model"`
}

// requestModelFor 返回实际发往上游的模型名。
func (m AstraCheckModelConfig) requestModelFor(target AstraCheckTarget) string {
	if model := strings.TrimSpace(m.RequestModel); model != "" {
		return model
	}
	if target.DefaultRequestModel != "" {
		return target.DefaultRequestModel
	}
	return m.ExpectedModel
}

// AstraCheckModelMatch 是某个候选在一次运行里的得分。
type AstraCheckModelMatch struct {
	Model string `json:"model"`
	Name  string `json:"name"`
	// Score 是该候选的对数证据（other 取最接近参考源的证据）
	Score float64 `json:"score"`
	// Match 是每条有效答案相对最强对手对数优势的 sigmoid，不是身份概率
	Match     float64 `json:"match"`
	Threshold float64 `json:"threshold"`
	Passed    bool    `json:"passed"`
}

// AstraCheckCellSummary 是某道题在一次运行里的样本情况。
type AstraCheckCellSummary struct {
	CellID     string         `json:"cell_id"`
	Planned    int            `json:"planned"`
	Total      int            `json:"total"`
	Valid      int            `json:"valid"`
	Invalid    int            `json:"invalid"`
	Minimum    int            `json:"minimum"`
	Categories map[string]int `json:"categories"`
}

// AstraCheckScore 是一次运行的判定结果。
type AstraCheckScore struct {
	Verdict string `json:"verdict"`
	// Winner 是强指向的候选（只有强指向时才有）；Strongest 是证据最高的候选（不论是否越线）
	Winner           string                  `json:"winner"`
	Strongest        string                  `json:"strongest"`
	NearestReference string                  `json:"nearest_reference"`
	Matches          []AstraCheckModelMatch  `json:"matches"`
	Reasons          []string                `json:"reasons"`
	Cells            []AstraCheckCellSummary `json:"cells"`
	ValidSamples     int                     `json:"valid_samples"`
	PlannedSamples   int                     `json:"planned_samples"`
}

// AstraCellObservation 是某道题的原始归一化计数（含无效输出）。
type AstraCellObservation struct {
	CellID string
	Counts map[string]int
}

// AstraJob 是一次运行里的一个请求任务。
type AstraJob struct {
	CellID string
	Index  int
}

// GroupStatusAstraCheckResult 是落库前的一次运行结果（一个预期模型）。
type GroupStatusAstraCheckResult struct {
	GroupID            int64
	ConfigID           int64
	Platform           string
	ExpectedModel      string
	Round              int
	BenchmarkPackageID string
	BenchmarkVersion   string
	BenchmarkSHA256    string
	ScoringVersion     string
	RequestModel       string
	Tier               string
	AccountID          *int64
	Verdict            string
	Winner             string
	Strongest          string
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
	CostUSD            float64
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
	Platform           string                   `json:"platform"`
	ExpectedModel      string                   `json:"expected_model"`
	Round              int                      `json:"round"`
	BenchmarkPackageID string                   `json:"benchmark_package_id"`
	BenchmarkVersion   string                   `json:"benchmark_version"`
	BenchmarkSHA256    string                   `json:"benchmark_sha256"`
	ScoringVersion     string                   `json:"scoring_version"`
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
	CostUSD            float64                  `json:"cost_usd"`
	LatencyMS          *int64                   `json:"latency_ms"`
	HTTPCode           *int                     `json:"http_code"`
	ErrorDetail        string                   `json:"error_detail"`
	StartedAt          time.Time                `json:"started_at"`
	FinishedAt         time.Time                `json:"finished_at"`
	CreatedAt          time.Time                `json:"created_at"`
}

// GroupStatusAstraCheckState 是某个（分组, 预期模型）的最近结果与稳定结论。
type GroupStatusAstraCheckState struct {
	ID            int64  `json:"id"`
	GroupID       int64  `json:"group_id"`
	ConfigID      int64  `json:"config_id"`
	ExpectedModel string `json:"expected_model"`
	DisplayName   string `json:"display_name"`
	// Method 是该目标的检测方法（meow / sol_juice / modeltrace），派生字段，不落库
	Method string `json:"method"`
	// TraceProxyModel 是代表该目标做 ModelTrace 归因的库内模型（目标本身不在指纹库里时，如 GPT-6.1 Sol → gpt-6-astra），派生字段
	TraceProxyModel     string                 `json:"trace_proxy_model,omitempty"`
	Verdict             string                 `json:"verdict"`
	StableStatus        string                 `json:"stable_status"`
	Winner              string                 `json:"winner"`
	Matches             []AstraCheckModelMatch `json:"matches"`
	Reasons             []string               `json:"reasons"`
	Detail              string                 `json:"detail"`
	CheckedAt           *time.Time             `json:"checked_at"`
	ConsecutiveMismatch int                    `json:"consecutive_mismatch"`
	ValidSamples        int                    `json:"valid_samples"`
	PlannedSamples      int                    `json:"planned_samples"`
	InputTokens         int64                  `json:"input_tokens"`
	OutputTokens        int64                  `json:"output_tokens"`
	ReasoningTokens     int64                  `json:"reasoning_tokens"`
	LastCostUSD         float64                `json:"last_cost_usd"`
	LastRunID           *int64                 `json:"last_run_id"`
	BenchmarkPackageID  string                 `json:"benchmark_package_id"`
	BenchmarkVersion    string                 `json:"benchmark_version"`
	CreatedAt           time.Time              `json:"created_at"`
	UpdatedAt           time.Time              `json:"updated_at"`
}

type GroupStatusAstraCheckExecution struct {
	Group   *Group                       `json:"group,omitempty"`
	Config  *GroupStatusConfig           `json:"config,omitempty"`
	Account *Account                     `json:"account,omitempty"`
	Result  *GroupStatusAstraCheckResult `json:"result,omitempty"`
	Run     *GroupStatusAstraCheckRun    `json:"run,omitempty"`
	State   *GroupStatusAstraCheckState  `json:"state,omitempty"`
	Event   *GroupStatusEvent            `json:"event,omitempty"`
	// Confirmed 表示本次是首次 mismatch 后在同一账号上的立即复测
	Confirmed bool `json:"confirmed"`
}

// ---------- 模型列表 ----------

// astraCheckDefaultModels 是某平台分组的默认检测列表（第一个目标）；不支持的平台为空。
func astraCheckDefaultModels(platform string) []AstraCheckModelConfig {
	if targets := AstraCheckTargetsForPlatform(platform); len(targets) > 0 {
		return []AstraCheckModelConfig{{ExpectedModel: targets[0].ID}}
	}
	return []AstraCheckModelConfig{}
}

// normalizeAstraCheckModels 去空白、去掉空的预期模型并按预期模型去重（保留首次出现的顺序与请求模型）。
func normalizeAstraCheckModels(models []AstraCheckModelConfig) []AstraCheckModelConfig {
	out := make([]AstraCheckModelConfig, 0, len(models))
	seen := make(map[string]struct{}, len(models))
	for _, m := range models {
		expected := strings.TrimSpace(m.ExpectedModel)
		if expected == "" {
			continue
		}
		if _, dup := seen[expected]; dup {
			continue
		}
		seen[expected] = struct{}{}
		out = append(out, AstraCheckModelConfig{ExpectedModel: expected, RequestModel: strings.TrimSpace(m.RequestModel)})
	}
	return out
}

// astraCheckExpectedModels 返回配置里的预期模型 id 列表。
func astraCheckExpectedModels(models []AstraCheckModelConfig) []string {
	out := make([]string, 0, len(models))
	for _, m := range models {
		out = append(out, m.ExpectedModel)
	}
	return out
}

// ---------- 归一化与任务规划 ----------

// NormalizeAstraAnswer 按基准包的归一器整理回复：去首尾空白（casefold 版再转小写）。
// 空回复或超过长度上限的回复不参与判定（记为无效）；不合规的措辞仍是一次观测，由计分时归入「未见」类别。
func NormalizeAstraAnswer(n AstraNormalizer, raw string) string {
	text := strings.TrimSpace(raw)
	if text == "" {
		return AstraCheckInvalidOutput
	}
	if n.ID != "exact_trimmed" {
		text = strings.ToLower(text)
	}
	maxLength := n.MaxLength
	if maxLength <= 0 {
		maxLength = 4096
	}
	if len([]rune(text)) > maxLength {
		return AstraCheckInvalidOutput
	}
	return text
}

// PlanAstraJobs 按档位生成请求任务，题目轮转派发（cell1, cell2, …, cell1, …）。
func PlanAstraJobs(bench *AstraBenchmark, tier string) ([]AstraJob, error) {
	if bench == nil {
		return nil, fmt.Errorf("meow benchmark is nil")
	}
	tierCfg, ok := bench.Tiers[tier]
	if !ok {
		return nil, fmt.Errorf("unsupported tier %q", tier)
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

// ---------- v3 判定 ----------

// astraDirichletLogEvidence 是一题的 Dirichlet-multinomial 对数边际似然（公共多项式系数在来源之间抵消，省略）：
// lgamma(Σa) − lgamma(Σa+Σn) + Σ_k [lgamma(a_k+n_k) − lgamma(a_k)]。
func astraDirichletLogEvidence(alpha []float64, counts []int) float64 {
	sumAlpha, sumCounts := 0.0, 0
	for i, a := range alpha {
		sumAlpha += a
		sumCounts += counts[i]
	}
	if sumCounts == 0 {
		return 0
	}
	lgamma := func(x float64) float64 {
		v, _ := math.Lgamma(x)
		return v
	}
	value := lgamma(sumAlpha) - lgamma(sumAlpha+float64(sumCounts))
	for i, a := range alpha {
		if counts[i] > 0 {
			value += lgamma(a+float64(counts[i])) - lgamma(a)
		}
	}
	return value
}

func astraSigmoid(x float64) float64 {
	if x >= 0 {
		return 1 / (1 + math.Exp(-x))
	}
	e := math.Exp(x)
	return e / (1 + e)
}

// ScoreAstraCheck 用基准包对一次运行的计数做判定（纯函数）：expected 是本次要核对的预期模型。
func ScoreAstraCheck(bench *AstraBenchmark, tier, expected string, observations []AstraCellObservation) (*AstraCheckScore, error) {
	if bench == nil {
		return nil, fmt.Errorf("meow benchmark is nil")
	}
	tierCfg, ok := bench.Tiers[tier]
	if !ok {
		return nil, fmt.Errorf("unsupported tier %q", tier)
	}
	reasons := make(map[string]struct{})
	if !tierCfg.Calibrated {
		reasons[AstraCheckReasonUncalibrated] = struct{}{}
	}
	observed := make(map[string]map[string]int, len(observations))
	for _, obs := range observations {
		observed[obs.CellID] = obs.Counts
	}

	sourceEvidence := make(map[string]float64)
	score := &AstraCheckScore{}
	for _, cell := range bench.Cells {
		planned := tierCfg.Counts[cell.ID]
		if planned == 0 {
			continue
		}
		fitted := bench.Fitted[cell.ID]
		vector := make([]int, len(fitted.Categories))
		categories := make(map[string]int)
		invalid, valid := 0, 0
		for answer, count := range observed[cell.ID] {
			if count <= 0 {
				continue
			}
			if answer == AstraCheckInvalidOutput {
				invalid += count
				categories[AstraCheckInvalidOutput] += count
				continue
			}
			idx, known := fitted.CategoryIndex[answer]
			if !known {
				idx = fitted.UnseenIndex
			}
			vector[idx] += count
			categories[fitted.Categories[idx]] += count
			valid += count
		}
		minimum := int(math.Ceil(float64(planned) * bench.CompletionRatio))
		if valid < minimum {
			reasons[AstraCheckReasonSamplesIncomplete] = struct{}{}
		}
		score.Cells = append(score.Cells, AstraCheckCellSummary{
			CellID:     cell.ID,
			Planned:    planned,
			Total:      valid + invalid,
			Valid:      valid,
			Invalid:    invalid,
			Minimum:    minimum,
			Categories: categories,
		})
		score.ValidSamples += valid
		score.PlannedSamples += planned
		if valid == 0 {
			continue
		}
		for source, alpha := range fitted.Alpha {
			sourceEvidence[source] += astraDirichletLogEvidence(alpha, vector)
		}
	}
	if score.ValidSamples < int(math.Ceil(float64(score.PlannedSamples)*bench.CompletionRatio)) {
		reasons[AstraCheckReasonSamplesIncomplete] = struct{}{}
	}

	// 候选证据：真实候选用自己的来源；other 取整轮最接近的单一参考源（不是逐题混合）
	evidence := make(map[string]float64, len(bench.ModelIDs))
	for _, model := range bench.Models {
		if !model.ReferenceOnly {
			evidence[model.ID] = sourceEvidence[model.ID]
			continue
		}
		best := math.Inf(-1)
		for _, source := range bench.ReferenceSources {
			if v := sourceEvidence[source]; v > best {
				best = v
				score.NearestReference = source
			}
		}
		evidence[model.ID] = best
	}

	strongest, unique := "", false
	best, second := math.Inf(-1), math.Inf(-1)
	for _, model := range bench.ModelIDs {
		v := evidence[model]
		switch {
		case v > best:
			second = best
			best = v
			strongest = model
		case v > second:
			second = v
		}
	}
	unique = best-second > 1e-9
	score.Strongest = strongest
	eligible := len(reasons) == 0 && score.ValidSamples > 0

	for _, model := range bench.ModelIDs {
		rival := math.Inf(-1)
		for _, other := range bench.ModelIDs {
			if other != model && evidence[other] > rival {
				rival = evidence[other]
			}
		}
		match := 0.5
		if score.ValidSamples > 0 {
			match = astraSigmoid((evidence[model] - rival) / float64(score.ValidSamples))
		}
		threshold := tierCfg.Thresholds[model]
		passed := eligible && unique && model == strongest && match > threshold
		if passed {
			score.Winner = model
		}
		score.Matches = append(score.Matches, AstraCheckModelMatch{
			Model:     model,
			Name:      AstraModelLabel(model),
			Score:     evidence[model],
			Match:     match,
			Threshold: threshold,
			Passed:    passed,
		})
	}

	switch {
	case score.ValidSamples == 0:
		reasons[AstraCheckReasonNoValidSamples] = struct{}{}
		score.Verdict = AstraCheckVerdictInsufficient
	case score.Winner == "":
		if eligible {
			reasons[AstraCheckReasonNoStrongDirection] = struct{}{}
		}
		score.Verdict = AstraCheckVerdictInsufficient
	case score.Winner == expected:
		score.Verdict = AstraCheckVerdictMatch
	default:
		score.Verdict = AstraCheckVerdictMismatch
	}
	score.Reasons = make([]string, 0, len(reasons))
	for reason := range reasons {
		score.Reasons = append(score.Reasons, reason)
	}
	sort.Strings(score.Reasons)
	return score, nil
}

// ---------- 状态机 ----------

// ComputeAstraCheckTransition 是纯函数：把某个预期模型的一次运行并进它自己的状态，稳定结论切换时产出事件。
//
//   - match：清零计数，稳定置 pass；原为 mismatch 则发 astra_recovered
//   - mismatch：计数 +1；原来不是 mismatch 且计数达到阈值（2）时置 mismatch 并发 astra_mismatch
//   - insufficient：只更新最近一次结果，不动稳定结论与计数
func ComputeAstraCheckTransition(prev *GroupStatusAstraCheckState, result *GroupStatusAstraCheckResult, runID int64) (*GroupStatusAstraCheckState, *GroupStatusEvent) {
	next := &GroupStatusAstraCheckState{}
	if prev != nil {
		*next = *prev
		// 结果来源换了（如 GPT-5.6 Sol 从 meow 基准改为 Juice、Claude Opus 5.5 改为 ModelTrace）：旧来源的
		// 稳定结论与连续计数不再有意义，静默清零，不发事件，避免旧来源的误判拖着新来源一起变红
		if prev.BenchmarkPackageID != "" && result.BenchmarkPackageID != "" && prev.BenchmarkPackageID != result.BenchmarkPackageID {
			next.StableStatus = ""
			next.ConsecutiveMismatch = 0
		}
	}
	next.GroupID = result.GroupID
	next.ConfigID = result.ConfigID
	next.ExpectedModel = result.ExpectedModel

	observedAt := result.FinishedAt
	if observedAt.IsZero() {
		observedAt = time.Now()
	}
	detail := astraCheckDetailText(result)
	next.Verdict = result.Verdict
	next.Winner = result.Winner
	next.Matches = append([]AstraCheckModelMatch(nil), result.Matches...)
	next.Reasons = append([]string(nil), result.Reasons...)
	next.Detail = detail
	next.CheckedAt = &observedAt
	next.ValidSamples = result.ValidSamples
	next.PlannedSamples = result.PlannedSamples
	next.InputTokens = result.InputTokens
	next.OutputTokens = result.OutputTokens
	next.ReasoningTokens = result.ReasoningTokens
	next.LastCostUSD = result.CostUSD
	next.BenchmarkPackageID = result.BenchmarkPackageID
	next.BenchmarkVersion = result.BenchmarkVersion
	if runID > 0 {
		id := runID
		next.LastRunID = &id
	}

	prevStable := strings.TrimSpace(next.StableStatus)
	newEvent := func(eventType, from, to string) *GroupStatusEvent {
		return &GroupStatusEvent{
			GroupID:     result.GroupID,
			ConfigID:    result.ConfigID,
			EventType:   eventType,
			FromStatus:  from,
			ToStatus:    to,
			LatencyMS:   result.LatencyMS,
			HTTPCode:    result.HTTPCode,
			SubStatus:   astraCheckEventSubStatus(result.ExpectedModel, result.Winner),
			ErrorDetail: detail,
			ObservedAt:  observedAt,
		}
	}

	switch result.Verdict {
	case AstraCheckVerdictMatch:
		next.ConsecutiveMismatch = 0
		next.StableStatus = AstraCheckStatusPass
		if prevStable == AstraCheckStatusMismatch {
			return next, newEvent(GroupStatusEventAstraRecovered, prevStable, AstraCheckStatusPass)
		}
		return next, nil
	case AstraCheckVerdictMismatch:
		next.ConsecutiveMismatch++
		if prevStable == AstraCheckStatusMismatch {
			return next, nil
		}
		if next.ConsecutiveMismatch >= groupStatusAstraCheckMismatchThreshold {
			next.StableStatus = AstraCheckStatusMismatch
			return next, newEvent(GroupStatusEventAstraMismatch, prevStable, AstraCheckStatusMismatch)
		}
		return next, nil
	default:
		return next, nil
	}
}

// astraCheckDetailText 生成一行可读摘要；state 会公开给用户，所以不含账号 id 与答案原文。
func astraCheckDetailText(result *GroupStatusAstraCheckResult) string {
	if result == nil {
		return ""
	}
	parts := make([]string, 0, 6)
	parts = append(parts, "expected "+AstraModelLabel(result.ExpectedModel))
	if result.Strongest != "" {
		for _, match := range result.Matches {
			if match.Model == result.Strongest {
				parts = append(parts, fmt.Sprintf("strongest %s %.1f%%/%.1f%%", AstraModelLabel(match.Model), match.Match*100, match.Threshold*100))
				break
			}
		}
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

// decorateAstraCheckSummary 填充派生字段（不落库）：按配置顺序整理各预期模型的状态——只保留仍在配置里的模型，
// 还没检测过的给一个空占位，前端据此为每个模型画一个徽章；另附内置基准包元数据。
func decorateAstraCheckSummary(summary *GroupStatusSummary) {
	if summary == nil {
		return
	}
	if summary.AstraCheckModels == nil {
		summary.AstraCheckModels = []AstraCheckModelConfig{}
	}
	byModel := make(map[string]GroupStatusAstraCheckState, len(summary.AstraCheckStates))
	for _, state := range summary.AstraCheckStates {
		byModel[state.ExpectedModel] = state
	}
	states := make([]GroupStatusAstraCheckState, 0, len(summary.AstraCheckModels))
	for _, m := range summary.AstraCheckModels {
		state, ok := byModel[m.ExpectedModel]
		if !ok {
			state = GroupStatusAstraCheckState{GroupID: summary.GroupID, ConfigID: summary.ConfigID, ExpectedModel: m.ExpectedModel}
		}
		state.DisplayName = AstraModelLabel(m.ExpectedModel)
		if target, ok := astraCheckTarget(m.ExpectedModel); ok {
			state.Method = target.Method
			if target.TraceProxy {
				state.TraceProxyModel = target.TraceModelID
			}
		}
		// 前端直接读 .length，切片必须是 []
		if state.Matches == nil {
			state.Matches = []AstraCheckModelMatch{}
		}
		if state.Reasons == nil {
			state.Reasons = []string{}
		}
		states = append(states, state)
	}
	summary.AstraCheckStates = states
	summary.AstraCheckBenchmarks = []AstraBenchmarkMeta{}
	if reg, err := LoadEmbeddedAstraBenchmarks(); err == nil {
		summary.AstraCheckBenchmarks = reg.Metas()
	}
}

// ---------- 事件 ----------

// astraCheckEventSubStatus 把预期模型与强指向的候选编进事件 sub_status：<expected>:winner_<winner>。
func astraCheckEventSubStatus(expected, winner string) string {
	if strings.TrimSpace(winner) == "" {
		winner = astraCheckEventSubStatusUnknownWinner
	}
	return expected + astraCheckEventSubStatusWinnerPrefix + winner
}

// astraCheckEventModels 从事件 sub_status 里取出预期模型与强指向的候选（缺失时为空串）。
func astraCheckEventModels(event *GroupStatusEvent) (expected, winner string) {
	if event == nil {
		return "", ""
	}
	subStatus := strings.TrimSpace(event.SubStatus)
	idx := strings.LastIndex(subStatus, astraCheckEventSubStatusWinnerPrefix)
	if idx < 0 {
		return "", ""
	}
	expected, winner = subStatus[:idx], subStatus[idx+len(astraCheckEventSubStatusWinnerPrefix):]
	if winner == astraCheckEventSubStatusUnknownWinner {
		winner = ""
	}
	return expected, winner
}

// astraWinnerFromEvent 取出事件里强指向候选的可读名。
func astraWinnerFromEvent(event *GroupStatusEvent) string {
	_, winner := astraCheckEventModels(event)
	if winner == "" {
		return "?"
	}
	return AstraModelLabel(winner)
}

// astraExpectedFromEvent 取出事件对应的预期模型的可读名。
func astraExpectedFromEvent(event *GroupStatusEvent) string {
	expected, _ := astraCheckEventModels(event)
	if expected == "" {
		return "?"
	}
	return AstraModelLabel(expected)
}

// isAstraCheckEvent 报告事件是否来自 meow 指纹验证。
func isAstraCheckEvent(eventType string) bool {
	return eventType == GroupStatusEventAstraMismatch || eventType == GroupStatusEventAstraRecovered
}

// astraCheckStatusLabel 是指纹事件里 from/to 的中文标签。
func astraCheckStatusLabel(status string) string {
	switch strings.TrimSpace(status) {
	case AstraCheckStatusPass:
		return "指纹一致"
	case AstraCheckStatusMismatch:
		return "指纹不符"
	default:
		return "未知"
	}
}
