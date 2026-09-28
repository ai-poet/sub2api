package service

import (
	"errors"
	"math"
	"regexp"
	"sort"
	"strconv"
	"unicode"
)

// ModelTrace 数字指纹的解析与评分（本 fork 自有功能）。
//
// 以 ModelTrace（MIT）的 static/fingerprint-core.js 为规格逐步实现，数值与参考实现逐位对齐：
// 每条回答分别算 Hellinger 边缘分布分与有序分块分，按 0.75 / 0.25 融合成各模型的 z 分，
// 多条回答取均值后按回答条数对应的校准温度做 softmax，得到库内闭集概率。
// 与参考实现的一致性由 testdata/modeltrace 的 golden 测试钉住。

var (
	ErrModelTraceNoValidOutputs = errors.New("no valid ModelTrace outputs")

	// 与参考实现的 JS 版本一致：只认 ASCII 数字（Python 版的 \d 会额外接受全角数字）
	modelTraceDigitsPattern = regexp.MustCompile(`[0-9]+`)
)

// ModelTraceOutput 是一条待评分的回答。
type ModelTraceOutput struct {
	Text          string
	ExpectedCount int
}

// ModelTraceOutputDiagnostic 是一条回答的解析情况。
type ModelTraceOutputDiagnostic struct {
	Index          int  `json:"index"`
	ParsedNumbers  int  `json:"parsed_numbers"`
	MinimumNumbers int  `json:"minimum_numbers"`
	Accepted       bool `json:"accepted"`
}

// ModelTraceRankEntry 是某个候选模型的归因结果。
type ModelTraceRankEntry struct {
	Model                  string  `json:"model"`
	DisplayName            string  `json:"display_name"`
	Family                 string  `json:"family"`
	FamilyName             string  `json:"family_name"`
	Probability            float64 `json:"probability"`
	ConditionalProbability float64 `json:"conditional_probability"`
	Score                  float64 `json:"score"`
}

// ModelTraceFamilyProbability 是某个模型家族的概率（家族内各模型概率之和）。
type ModelTraceFamilyProbability struct {
	Family      string  `json:"family"`
	DisplayName string  `json:"display_name"`
	Probability float64 `json:"probability"`
}

// ModelTraceAnalysis 是多条回答的归因结果；Results 按概率降序。
type ModelTraceAnalysis struct {
	Results        []ModelTraceRankEntry         `json:"results"`
	Families       []ModelTraceFamilyProbability `json:"families"`
	UsedOutputs    int                           `json:"used_outputs"`
	Diagnostics    []ModelTraceOutputDiagnostic  `json:"diagnostics"`
	CalibrationKey int                           `json:"calibration_queries"`
	Beta           float64                       `json:"beta"`
	CVAccuracy     float64                       `json:"cv_accuracy"`
	// OutputScores 是每条有效回答的融合分（按库内模型顺序），与 Diagnostics 中 Accepted 的条目依次对应
	OutputScores [][]float64 `json:"-"`
}

// Top 返回概率最高的候选；没有结果时返回 nil。
func (a *ModelTraceAnalysis) Top() *ModelTraceRankEntry {
	if a == nil || len(a.Results) == 0 {
		return nil
	}
	return &a.Results[0]
}

// Find 返回指定模型的归因结果。
func (a *ModelTraceAnalysis) Find(model string) *ModelTraceRankEntry {
	if a == nil {
		return nil
	}
	for i := range a.Results {
		if a.Results[i].Model == model {
			return &a.Results[i]
		}
	}
	return nil
}

// ParseModelTraceNumbers 取回答里最长的一段数字序列：分隔符中出现字母时切段，只保留 1..355。
func ParseModelTraceNumbers(text string) []int {
	var runs [][]int
	var current []int
	previousEnd := 0
	for _, loc := range modelTraceDigitsPattern.FindAllStringIndex(text, -1) {
		separator := text[previousEnd:loc[0]]
		if len(current) > 0 && modelTraceContainsLetter(separator) {
			runs = append(runs, current)
			current = nil
		}
		if value, err := strconv.Atoi(text[loc[0]:loc[1]]); err == nil && value >= modelTraceValueMin && value <= modelTraceValueMax {
			current = append(current, value)
		}
		previousEnd = loc[1]
	}
	if len(current) > 0 {
		runs = append(runs, current)
	}
	var best []int
	for _, run := range runs {
		if len(run) > len(best) {
			best = run
		}
	}
	return best
}

func modelTraceContainsLetter(s string) bool {
	for _, r := range s {
		if unicode.IsLetter(r) {
			return true
		}
	}
	return false
}

// ModelTraceMinimumNumbers 是一条回答计入评分所需的最少数字个数：max(80, ⌈expected×0.55⌉)。
func ModelTraceMinimumNumbers(expected int) int {
	if expected <= 0 {
		return 80
	}
	minimum := int(math.Ceil(float64(expected) * 0.55))
	if minimum < 80 {
		return 80
	}
	return minimum
}

// Analyze 对多条回答做闭集归因；没有一条回答达到最少数字个数时返回 ErrModelTraceNoValidOutputs。
func (b *ModelTraceBank) Analyze(outputs []ModelTraceOutput) (*ModelTraceAnalysis, error) {
	if b == nil {
		return nil, ErrGroupStatusModelTraceBankInvalid
	}
	analysis := &ModelTraceAnalysis{Diagnostics: make([]ModelTraceOutputDiagnostic, 0, len(outputs))}
	for i, output := range outputs {
		numbers := ParseModelTraceNumbers(output.Text)
		minimum := ModelTraceMinimumNumbers(output.ExpectedCount)
		accepted := len(numbers) >= minimum
		analysis.Diagnostics = append(analysis.Diagnostics, ModelTraceOutputDiagnostic{
			Index:          i,
			ParsedNumbers:  len(numbers),
			MinimumNumbers: minimum,
			Accepted:       accepted,
		})
		if accepted {
			analysis.OutputScores = append(analysis.OutputScores, b.scoreNumbers(numbers))
		}
	}
	if len(analysis.OutputScores) == 0 {
		return analysis, ErrModelTraceNoValidOutputs
	}
	b.finishAnalysis(analysis)
	return analysis, nil
}

// AnalyzeNumbers 与 Analyze 相同，但输入是已解析的数字序列（已计入的回答）。
func (b *ModelTraceBank) AnalyzeNumbers(sequences [][]int) (*ModelTraceAnalysis, error) {
	if b == nil {
		return nil, ErrGroupStatusModelTraceBankInvalid
	}
	analysis := &ModelTraceAnalysis{}
	for i, numbers := range sequences {
		analysis.Diagnostics = append(analysis.Diagnostics, ModelTraceOutputDiagnostic{
			Index:          i,
			ParsedNumbers:  len(numbers),
			MinimumNumbers: 0,
			Accepted:       len(numbers) > 0,
		})
		if len(numbers) > 0 {
			analysis.OutputScores = append(analysis.OutputScores, b.scoreNumbers(numbers))
		}
	}
	if len(analysis.OutputScores) == 0 {
		return analysis, ErrModelTraceNoValidOutputs
	}
	b.finishAnalysis(analysis)
	return analysis, nil
}

// SingleOutputTop 返回单条回答（按 1 条回答的校准温度）的第一候选与其概率。
func (b *ModelTraceBank) SingleOutputTop(scores []float64) (string, float64) {
	if b == nil || len(scores) != len(b.Models) {
		return "", 0
	}
	beta := b.Calibration[1].Beta
	scaled := make([]float64, len(scores))
	for i, v := range scores {
		scaled[i] = beta * v
	}
	probs := modelTraceSoftmax(scaled)
	best := 0
	for i := range probs {
		if probs[i] > probs[best] {
			best = i
		}
	}
	return b.Models[best].ID, probs[best]
}

func (b *ModelTraceBank) finishAnalysis(analysis *ModelTraceAnalysis) {
	valid := len(analysis.OutputScores)
	n := len(b.Models)
	combined := make([]float64, n)
	for m := 0; m < n; m++ {
		sum := 0.0
		for _, scores := range analysis.OutputScores {
			sum += scores[m]
		}
		combined[m] = sum / float64(valid)
	}
	key := valid
	if key > modelTraceMaxCalibrationKey {
		key = modelTraceMaxCalibrationKey
	}
	cal := b.Calibration[key]
	analysis.UsedOutputs = valid
	analysis.CalibrationKey = key
	analysis.Beta = cal.Beta
	analysis.CVAccuracy = cal.CVAccuracy

	scaled := make([]float64, n)
	for i, v := range combined {
		scaled[i] = cal.Beta * v
	}
	probs := modelTraceSoftmax(scaled)
	results := make([]ModelTraceRankEntry, n)
	for i, model := range b.Models {
		results[i] = ModelTraceRankEntry{
			Model:       model.ID,
			DisplayName: model.DisplayName,
			Family:      model.Family,
			FamilyName:  b.FamilyNames[model.Family],
			Probability: probs[i],
			Score:       combined[i],
		}
	}
	sort.SliceStable(results, func(i, j int) bool { return results[i].Probability > results[j].Probability })

	familyProb := make(map[string]float64, len(b.FamilyOrder))
	for _, item := range results {
		familyProb[item.Family] += item.Probability
	}
	for i := range results {
		if total := familyProb[results[i].Family]; total > 0 {
			results[i].ConditionalProbability = results[i].Probability / total
		}
	}
	families := make([]ModelTraceFamilyProbability, 0, len(b.FamilyOrder))
	for _, family := range b.FamilyOrder {
		families = append(families, ModelTraceFamilyProbability{
			Family:      family,
			DisplayName: b.FamilyNames[family],
			Probability: familyProb[family],
		})
	}
	analysis.Results = results
	analysis.Families = families
}

// scoreNumbers 返回一条回答对库内各模型的融合 z 分（按库内模型顺序）。
func (b *ModelTraceBank) scoreNumbers(numbers []int) []float64 {
	marginal := b.hellingerScores(numbers)
	if b.OrdWeight == 0 || len(b.OrdCentroids) == 0 {
		return marginal
	}
	ordered := b.orderedBlockScores(numbers)
	fused := make([]float64, len(marginal))
	for i := range marginal {
		fused[i] = (1-b.OrdWeight)*marginal[i] + b.OrdWeight*ordered[i]
	}
	return fused
}

func (b *ModelTraceBank) hellingerScores(numbers []int) []float64 {
	counts := make([]float64, modelTraceHellingerDim)
	for _, v := range numbers {
		counts[v-modelTraceValueMin]++
	}
	total := float64(len(numbers)) + 0.5*float64(modelTraceHellingerDim)
	projected := make([]float64, modelTraceHellingerDim)
	for i, c := range counts {
		feature := math.Sqrt((c + 0.5) / total)
		projected[i] = (feature - b.HellMean[i]) / b.HellScale[i]
	}
	projected = modelTraceNormalized(modelTraceSubtractBasis(projected, b.HellBasis))
	scores := make([]float64, len(b.HellCentroids))
	for m, centroid := range b.HellCentroids {
		scores[m] = modelTraceDot(projected, centroid)
	}
	return modelTraceStandardize(scores)
}

func (b *ModelTraceBank) orderedBlockScores(numbers []int) []float64 {
	feature := modelTraceOrderedBlockFeature(numbers)
	standardized := make([]float64, len(feature))
	for i, v := range feature {
		standardized[i] = (v - b.OrdMean[i]) / b.OrdScale[i]
	}
	unit := modelTraceNormalized(standardized)
	n := len(b.OrdCentroids)
	template := make([]float64, n)
	for m := 0; m < n; m++ {
		best := math.Inf(-1)
		for _, env := range b.OrdEnvCentroids {
			if v := modelTraceDot(unit, env[m]); v > best {
				best = v
			}
		}
		template[m] = best
	}
	template = modelTraceStandardize(template)

	projected := modelTraceNormalized(modelTraceSubtractBasis(standardized, b.OrdBasis))
	nuisance := make([]float64, n)
	for m, centroid := range b.OrdCentroids {
		nuisance[m] = modelTraceDot(projected, centroid)
	}
	nuisance = modelTraceStandardize(nuisance)

	mixed := make([]float64, n)
	for m := range mixed {
		mixed[m] = 0.5*template[m] + 0.5*nuisance[m]
	}
	return modelTraceStandardize(mixed)
}

// modelTraceOrderedBlockFeature：按位置分 4 块（余数分给靠前的块），每块 16 个取值区间，
// 再加 10 个末位数字区间；各区间计数加 0.5 平滑后取比例的平方根。
func modelTraceOrderedBlockFeature(numbers []int) []float64 {
	pieces := make([]float64, 0, modelTraceOrderedBlockDim)
	base := len(numbers) / 4
	remainder := len(numbers) % 4
	start := 0
	for i := 0; i < 4; i++ {
		size := base
		if i < remainder {
			size++
		}
		bins := make([]float64, 16)
		for j := range bins {
			bins[j] = 0.5
		}
		for _, v := range numbers[start : start+size] {
			index := int(math.Floor((float64(v-1) / 355) * 16))
			if index > 15 {
				index = 15
			}
			bins[index]++
		}
		start += size
		pieces = append(pieces, modelTraceSqrtProportions(bins)...)
	}
	lastDigits := make([]float64, 10)
	for j := range lastDigits {
		lastDigits[j] = 0.5
	}
	for _, v := range numbers {
		lastDigits[v%10]++
	}
	return append(pieces, modelTraceSqrtProportions(lastDigits)...)
}

func modelTraceSqrtProportions(bins []float64) []float64 {
	total := 0.0
	for _, v := range bins {
		total += v
	}
	out := make([]float64, len(bins))
	for i, v := range bins {
		out[i] = math.Sqrt(v / total)
	}
	return out
}

func modelTraceDot(left, right []float64) float64 {
	value := 0.0
	for i := range left {
		value += left[i] * right[i]
	}
	return value
}

func modelTraceNormalized(values []float64) []float64 {
	scale := math.Max(math.Sqrt(modelTraceDot(values, values)), 1e-12)
	out := make([]float64, len(values))
	for i, v := range values {
		out[i] = v / scale
	}
	return out
}

// modelTraceSubtractBasis 依次扣除在每个基向量上的投影（与参考 JS 实现相同的顺序）。
func modelTraceSubtractBasis(values []float64, basis [][]float64) []float64 {
	out := append([]float64(nil), values...)
	for _, vector := range basis {
		projection := modelTraceDot(out, vector)
		for i := range out {
			out[i] -= projection * vector[i]
		}
	}
	return out
}

// modelTraceStandardize 按总体标准差做 z 化，标准差下限 1e-12。
func modelTraceStandardize(values []float64) []float64 {
	if len(values) == 0 {
		return values
	}
	mean := 0.0
	for _, v := range values {
		mean += v
	}
	mean /= float64(len(values))
	variance := 0.0
	for _, v := range values {
		variance += (v - mean) * (v - mean)
	}
	variance /= float64(len(values))
	scale := math.Max(math.Sqrt(variance), 1e-12)
	out := make([]float64, len(values))
	for i, v := range values {
		out[i] = (v - mean) / scale
	}
	return out
}

func modelTraceSoftmax(values []float64) []float64 {
	maximum := math.Inf(-1)
	for _, v := range values {
		if v > maximum {
			maximum = v
		}
	}
	weights := make([]float64, len(values))
	total := 0.0
	for i, v := range values {
		weights[i] = math.Exp(v - maximum)
		total += weights[i]
	}
	for i := range weights {
		weights[i] /= total
	}
	return weights
}
