package service

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"sync"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	astrabenchmark "github.com/Wei-Shaw/sub2api/resources/astra-benchmark"
)

// meow 指纹验证的基准包（meow LLM detector 的 *.meow.json），本 fork 自有功能。
//
// 历史上只检测 GPT-6 Astra，所以代码与表名沿用 astra_check；现在内置多个 v3 基准包
// （GPT-6 + Claude），一个分组可以同时检测多个预期模型（GPT-5.6 Sol 改用 Juice 读数，见 group_status_sol_juice.go）。包内已经给出每道题的类别词表、
// 各来源的 Dirichlet 参数和各档强指向线；这里只做解析、严格校验与内存缓存，判定见 group_status_astra_check.go。

const (
	astraBenchmarkScoringVersion = "meow-fingerprint-v3-predictive"
	astraBenchmarkAggregation    = "nearest_source"
	astraBenchmarkUnseenCategory = "__UNSEEN_IN_TRAINING__"
	astraBenchmarkModeGPT        = "gpt"
	astraBenchmarkModeClaude     = "claude"

	// AstraCheckOtherModel 是包里代表「其他已知外部模型」的虚拟候选，证据取最接近的单一参考源。
	AstraCheckOtherModel = "other_known_external"

	AstraCheckTierLow    = "low"
	AstraCheckTierMedium = "medium"
	AstraCheckTierHigh   = "high"
)

var (
	ErrGroupStatusAstraBenchmarkInvalid = infraerrors.ServiceUnavailable("GROUP_STATUS_ASTRA_BENCHMARK_INVALID", "embedded meow benchmark package is invalid")

	astraCheckTiers = []string{AstraCheckTierLow, AstraCheckTierMedium, AstraCheckTierHigh}
)

type AstraBenchmarkModel struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	RequestModel  string `json:"request_model"`
	ReferenceOnly bool   `json:"reference_only,omitempty"`
}

// AstraNormalizer 描述某道题答案的归一化规则（与基准采集时一致）。
type AstraNormalizer struct {
	ID        string
	MaxLength int
}

type AstraBenchmarkCell struct {
	ID              string
	ProbeID         string
	System          string
	Prompt          string
	Effort          string
	Profile         string
	MaxOutputTokens int
	Normalizer      AstraNormalizer
}

type AstraBenchmarkTier struct {
	Counts        map[string]int
	Thresholds    map[string]float64
	Calibrated    bool
	TotalRequests int
}

// AstraFittedCell 是某道题的拟合结果：类别词表（含一个「训练中未见」类别）与各来源的 Dirichlet 参数。
type AstraFittedCell struct {
	Categories    []string
	CategoryIndex map[string]int
	UnseenIndex   int
	Alpha         map[string][]float64
}

// AstraBenchmark 是解析并校验后的 v3 基准包。
type AstraBenchmark struct {
	PackageID        string
	Version          string
	Mode             string
	ScoringVersion   string
	ContentSHA256    string
	BodySHA256       string
	CompletionRatio  float64
	Models           []AstraBenchmarkModel
	ModelIDs         []string
	RealSources      map[string]struct{}
	ReferenceSources []string
	Cells            []AstraBenchmarkCell
	CellIndex        map[string]*AstraBenchmarkCell
	Tiers            map[string]AstraBenchmarkTier
	Fitted           map[string]AstraFittedCell
}

type AstraBenchmarkTierMeta struct {
	Tier       string `json:"tier"`
	Requests   int    `json:"requests"`
	Calibrated bool   `json:"calibrated"`
}

// AstraBenchmarkMeta 是给管理端展示的基准包元数据（不含题面与参数）。
type AstraBenchmarkMeta struct {
	PackageID     string                   `json:"package_id"`
	Version       string                   `json:"version"`
	Mode          string                   `json:"mode"`
	ContentSHA256 string                   `json:"content_sha256"`
	BodySHA256    string                   `json:"body_sha256"`
	Models        []AstraBenchmarkModel    `json:"models"`
	Tiers         []AstraBenchmarkTierMeta `json:"tiers"`
}

type astraRawPackage struct {
	Mode          string `json:"mode"`
	ID            string `json:"id"`
	Version       string `json:"version"`
	ContentSHA256 string `json:"content_sha256"`
	Engine        struct {
		ScoringVersion  string  `json:"scoring_version"`
		CompletionRatio float64 `json:"completion_ratio"`
	} `json:"engine"`
	Models []AstraBenchmarkModel `json:"models"`
	Probes []struct {
		ID         string `json:"id"`
		Normalizer struct {
			ID         string `json:"id"`
			Parameters struct {
				MaxLength int `json:"max_length"`
			} `json:"parameters"`
		} `json:"normalizer"`
		Cells []struct {
			ID         string            `json:"id"`
			System     string            `json:"system"`
			Prompt     string            `json:"prompt"`
			History    []json.RawMessage `json:"history"`
			Effort     string            `json:"effort"`
			Profile    string            `json:"profile"`
			Parameters struct {
				MaxOutputTokens int `json:"max_output_tokens"`
			} `json:"parameters"`
		} `json:"cells"`
	} `json:"probes"`
	Tiers map[string]struct {
		Counts     map[string]int     `json:"counts"`
		Thresholds map[string]float64 `json:"thresholds"`
	} `json:"tiers"`
	Fitted struct {
		ScoringVersion   string   `json:"scoring_version"`
		Models           []string `json:"models"`
		Sources          []string `json:"sources"`
		ReferenceSources []string `json:"reference_sources"`
		Aggregation      string   `json:"aggregation"`
		Cells            map[string]struct {
			Categories []string             `json:"categories"`
			Alpha      map[string][]float64 `json:"alpha"`
		} `json:"cells"`
	} `json:"fitted"`
	Calibration struct {
		Tiers map[string]struct {
			Result struct {
				Status string `json:"status"`
			} `json:"result"`
		} `json:"tiers"`
	} `json:"calibration"`
}

// ParseAstraBenchmark 解析并严格校验一个 v3 meow 基准包；任何结构问题都拒绝整个包。
func ParseAstraBenchmark(raw []byte) (*AstraBenchmark, *AstraBenchmarkMeta, error) {
	if len(raw) == 0 {
		return nil, nil, errors.New("meow benchmark: empty package")
	}
	var pkg astraRawPackage
	if err := json.Unmarshal(raw, &pkg); err != nil {
		return nil, nil, fmt.Errorf("meow benchmark: decode: %w", err)
	}
	if pkg.Mode != astraBenchmarkModeGPT && pkg.Mode != astraBenchmarkModeClaude {
		return nil, nil, fmt.Errorf("meow benchmark: unsupported mode %q", pkg.Mode)
	}
	if pkg.Engine.ScoringVersion != astraBenchmarkScoringVersion || pkg.Fitted.ScoringVersion != astraBenchmarkScoringVersion {
		return nil, nil, fmt.Errorf("meow benchmark: unsupported scoring_version %q", pkg.Engine.ScoringVersion)
	}
	if pkg.Fitted.Aggregation != astraBenchmarkAggregation {
		return nil, nil, fmt.Errorf("meow benchmark: unsupported aggregation %q", pkg.Fitted.Aggregation)
	}
	ratio := pkg.Engine.CompletionRatio
	if math.IsNaN(ratio) || ratio <= 0 || ratio > 1 {
		return nil, nil, fmt.Errorf("meow benchmark: completion_ratio %v out of (0,1]", ratio)
	}
	if len(pkg.Models) < 2 {
		return nil, nil, errors.New("meow benchmark: needs at least two models")
	}

	bench := &AstraBenchmark{
		PackageID:       strings.TrimSpace(pkg.ID),
		Version:         strings.TrimSpace(pkg.Version),
		Mode:            pkg.Mode,
		ScoringVersion:  pkg.Engine.ScoringVersion,
		ContentSHA256:   strings.TrimSpace(pkg.ContentSHA256),
		CompletionRatio: ratio,
		RealSources:     make(map[string]struct{}),
		CellIndex:       make(map[string]*AstraBenchmarkCell),
		Tiers:           make(map[string]AstraBenchmarkTier, len(astraCheckTiers)),
		Fitted:          make(map[string]AstraFittedCell, len(pkg.Fitted.Cells)),
	}
	sum := sha256.Sum256(raw)
	bench.BodySHA256 = hex.EncodeToString(sum[:])
	if bench.PackageID == "" || bench.Version == "" {
		return nil, nil, errors.New("meow benchmark: id and version are required")
	}

	sources := make(map[string]struct{}, len(pkg.Fitted.Sources))
	for _, source := range pkg.Fitted.Sources {
		sources[source] = struct{}{}
	}
	referenceOnly := 0
	modelSet := make(map[string]struct{}, len(pkg.Models))
	for i, model := range pkg.Models {
		id := strings.TrimSpace(model.ID)
		if id == "" {
			return nil, nil, errors.New("meow benchmark: model id is empty")
		}
		if _, dup := modelSet[id]; dup {
			return nil, nil, fmt.Errorf("meow benchmark: duplicate model %q", id)
		}
		if i >= len(pkg.Fitted.Models) || pkg.Fitted.Models[i] != id {
			return nil, nil, fmt.Errorf("meow benchmark: fitted.models does not match models at %q", id)
		}
		modelSet[id] = struct{}{}
		if model.ReferenceOnly {
			referenceOnly++
		} else {
			if _, ok := sources[id]; !ok {
				return nil, nil, fmt.Errorf("meow benchmark: model %q has no fitted source", id)
			}
			bench.RealSources[id] = struct{}{}
		}
		name := strings.TrimSpace(model.Name)
		if name == "" {
			name = id
		}
		bench.Models = append(bench.Models, AstraBenchmarkModel{ID: id, Name: name, RequestModel: strings.TrimSpace(model.RequestModel), ReferenceOnly: model.ReferenceOnly})
		bench.ModelIDs = append(bench.ModelIDs, id)
	}
	if len(pkg.Fitted.Models) != len(pkg.Models) {
		return nil, nil, errors.New("meow benchmark: fitted.models length does not match models")
	}
	if referenceOnly > 1 {
		return nil, nil, errors.New("meow benchmark: at most one reference-only model is supported")
	}
	if referenceOnly == 1 {
		for _, source := range pkg.Fitted.ReferenceSources {
			if _, ok := sources[source]; !ok {
				return nil, nil, fmt.Errorf("meow benchmark: reference source %q is not a fitted source", source)
			}
			bench.ReferenceSources = append(bench.ReferenceSources, source)
		}
		if len(bench.ReferenceSources) == 0 {
			return nil, nil, errors.New("meow benchmark: reference-only model without reference sources")
		}
	}

	for _, probe := range pkg.Probes {
		probeID := strings.TrimSpace(probe.ID)
		if probeID == "" || len(probe.Cells) == 0 {
			return nil, nil, errors.New("meow benchmark: probe without id or cells")
		}
		normalizer := AstraNormalizer{ID: strings.TrimSpace(probe.Normalizer.ID), MaxLength: probe.Normalizer.Parameters.MaxLength}
		switch normalizer.ID {
		case "":
			normalizer.ID = "exact_trimmed_casefold"
		case "exact_trimmed_casefold", "exact_trimmed":
		default:
			return nil, nil, fmt.Errorf("meow benchmark: probe %s uses unsupported normalizer %q", probeID, normalizer.ID)
		}
		for _, cell := range probe.Cells {
			cellID := strings.TrimSpace(cell.ID)
			if cellID == "" || strings.TrimSpace(cell.Prompt) == "" {
				return nil, nil, fmt.Errorf("meow benchmark: probe %s has a cell without id or prompt", probeID)
			}
			if len(cell.History) > 0 {
				return nil, nil, fmt.Errorf("meow benchmark: cell %s uses history, which is not supported", cellID)
			}
			if _, dup := bench.CellIndex[cellID]; dup {
				return nil, nil, fmt.Errorf("meow benchmark: duplicate cell %q", cellID)
			}
			effort := strings.TrimSpace(cell.Effort)
			if effort == "" {
				effort = "low"
			}
			maxOutput := cell.Parameters.MaxOutputTokens
			if maxOutput <= 0 {
				maxOutput = 128
			}
			bench.Cells = append(bench.Cells, AstraBenchmarkCell{
				ID:              cellID,
				ProbeID:         probeID,
				System:          cell.System,
				Prompt:          cell.Prompt,
				Effort:          effort,
				Profile:         strings.TrimSpace(cell.Profile),
				MaxOutputTokens: maxOutput,
				Normalizer:      normalizer,
			})
		}
	}
	if len(bench.Cells) == 0 {
		return nil, nil, errors.New("meow benchmark: no cells")
	}
	for i := range bench.Cells {
		bench.CellIndex[bench.Cells[i].ID] = &bench.Cells[i]
	}

	for _, cell := range bench.Cells {
		raw, ok := pkg.Fitted.Cells[cell.ID]
		if !ok {
			return nil, nil, fmt.Errorf("meow benchmark: cell %s has no fitted parameters", cell.ID)
		}
		fitted := AstraFittedCell{
			Categories:    append([]string(nil), raw.Categories...),
			CategoryIndex: make(map[string]int, len(raw.Categories)),
			UnseenIndex:   -1,
			Alpha:         make(map[string][]float64, len(pkg.Fitted.Sources)),
		}
		for i, category := range raw.Categories {
			if _, dup := fitted.CategoryIndex[category]; dup {
				return nil, nil, fmt.Errorf("meow benchmark: cell %s has duplicate category %q", cell.ID, category)
			}
			fitted.CategoryIndex[category] = i
			if category == astraBenchmarkUnseenCategory {
				fitted.UnseenIndex = i
			}
		}
		if fitted.UnseenIndex < 0 {
			return nil, nil, fmt.Errorf("meow benchmark: cell %s lacks the unseen category", cell.ID)
		}
		for source := range sources {
			alpha, ok := raw.Alpha[source]
			if !ok || len(alpha) != len(raw.Categories) {
				return nil, nil, fmt.Errorf("meow benchmark: cell %s lacks alpha for %s", cell.ID, source)
			}
			for _, a := range alpha {
				if math.IsNaN(a) || math.IsInf(a, 0) || a <= 0 {
					return nil, nil, fmt.Errorf("meow benchmark: cell %s has a non-positive alpha for %s", cell.ID, source)
				}
			}
			fitted.Alpha[source] = alpha
		}
		bench.Fitted[cell.ID] = fitted
	}

	for _, tier := range astraCheckTiers {
		raw, ok := pkg.Tiers[tier]
		if !ok || len(raw.Counts) == 0 {
			return nil, nil, fmt.Errorf("meow benchmark: tier %s is missing", tier)
		}
		counts := make(map[string]int, len(bench.Cells))
		total := 0
		for cellID, n := range raw.Counts {
			if _, ok := bench.CellIndex[cellID]; !ok {
				return nil, nil, fmt.Errorf("meow benchmark: tier %s counts unknown cell %s", tier, cellID)
			}
			if n < 0 {
				n = 0
			}
			counts[cellID] = n
			total += n
		}
		if total == 0 {
			return nil, nil, fmt.Errorf("meow benchmark: tier %s plans zero requests", tier)
		}
		thresholds := make(map[string]float64, len(bench.ModelIDs))
		for _, modelID := range bench.ModelIDs {
			value, ok := raw.Thresholds[modelID]
			if !ok || math.IsNaN(value) || value < 0 || value >= 1 {
				return nil, nil, fmt.Errorf("meow benchmark: tier %s lacks a valid threshold for %s", tier, modelID)
			}
			thresholds[modelID] = value
		}
		bench.Tiers[tier] = AstraBenchmarkTier{
			Counts:        counts,
			Thresholds:    thresholds,
			Calibrated:    strings.EqualFold(strings.TrimSpace(pkg.Calibration.Tiers[tier].Result.Status), "target_met"),
			TotalRequests: total,
		}
	}

	return bench, bench.Meta(), nil
}

// Meta 返回基准包的展示元数据。
func (b *AstraBenchmark) Meta() *AstraBenchmarkMeta {
	if b == nil {
		return nil
	}
	meta := &AstraBenchmarkMeta{
		PackageID:     b.PackageID,
		Version:       b.Version,
		Mode:          b.Mode,
		ContentSHA256: b.ContentSHA256,
		BodySHA256:    b.BodySHA256,
		Models:        append([]AstraBenchmarkModel(nil), b.Models...),
	}
	for _, tier := range astraCheckTiers {
		meta.Tiers = append(meta.Tiers, AstraBenchmarkTierMeta{
			Tier:       tier,
			Requests:   b.Tiers[tier].TotalRequests,
			Calibrated: b.Tiers[tier].Calibrated,
		})
	}
	return meta
}

// HasModel 报告包内是否有该候选（含 other）。
func (b *AstraBenchmark) HasModel(id string) bool {
	if b == nil {
		return false
	}
	for _, model := range b.ModelIDs {
		if model == id {
			return true
		}
	}
	return false
}

// TierRequests 返回某档计划的请求数。
func (b *AstraBenchmark) TierRequests(tier string) int {
	if b == nil {
		return 0
	}
	return b.Tiers[tier].TotalRequests
}

// AstraBenchmarkRegistry 是按包 id 索引的一组基准包。
type AstraBenchmarkRegistry struct {
	Packages []*AstraBenchmark
	byID     map[string]*AstraBenchmark
}

// NewAstraBenchmarkRegistry 按给定顺序组装注册表；包 id 不能重复。
func NewAstraBenchmarkRegistry(packages ...*AstraBenchmark) (*AstraBenchmarkRegistry, error) {
	reg := &AstraBenchmarkRegistry{byID: make(map[string]*AstraBenchmark, len(packages))}
	for _, pkg := range packages {
		if pkg == nil {
			continue
		}
		if _, dup := reg.byID[pkg.PackageID]; dup {
			return nil, fmt.Errorf("meow benchmark: duplicate package %s", pkg.PackageID)
		}
		reg.byID[pkg.PackageID] = pkg
		reg.Packages = append(reg.Packages, pkg)
	}
	return reg, nil
}

// Package 返回指定 id 的基准包。
func (r *AstraBenchmarkRegistry) Package(id string) *AstraBenchmark {
	if r == nil {
		return nil
	}
	return r.byID[id]
}

// Metas 返回全部基准包的展示元数据。
func (r *AstraBenchmarkRegistry) Metas() []AstraBenchmarkMeta {
	if r == nil {
		return []AstraBenchmarkMeta{}
	}
	out := make([]AstraBenchmarkMeta, 0, len(r.Packages))
	for _, pkg := range r.Packages {
		out = append(out, *pkg.Meta())
	}
	return out
}

var (
	embeddedAstraBenchmarksOnce sync.Once
	embeddedAstraBenchmarks     *AstraBenchmarkRegistry
	embeddedAstraBenchmarksErr  error
)

// LoadEmbeddedAstraBenchmarks 解析随二进制内置的全部基准包；只解析一次，失败则每次返回同一错误。
func LoadEmbeddedAstraBenchmarks() (*AstraBenchmarkRegistry, error) {
	embeddedAstraBenchmarksOnce.Do(func() {
		packages := make([]*AstraBenchmark, 0, len(astrabenchmark.Files))
		for _, file := range astrabenchmark.Files {
			raw, err := astrabenchmark.FS.ReadFile(file)
			if err != nil {
				embeddedAstraBenchmarksErr = fmt.Errorf("%w: %s: %v", ErrGroupStatusAstraBenchmarkInvalid, file, err)
				return
			}
			bench, _, err := ParseAstraBenchmark(raw)
			if err != nil {
				embeddedAstraBenchmarksErr = fmt.Errorf("%w: %s: %v", ErrGroupStatusAstraBenchmarkInvalid, file, err)
				return
			}
			packages = append(packages, bench)
		}
		embeddedAstraBenchmarks, embeddedAstraBenchmarksErr = NewAstraBenchmarkRegistry(packages...)
	})
	return embeddedAstraBenchmarks, embeddedAstraBenchmarksErr
}

// astraBenchmarkProvider 抽象基准来源；默认用内置包，测试注入合成包。
type astraBenchmarkProvider interface {
	Registry() (*AstraBenchmarkRegistry, error)
}

// ---------- 可检测的目标模型 ----------

// 目标的检测方法：meow 基准（一批短答题 + v3 判定）或 Juice 读数（一条 high 推理请求）。
const (
	AstraCheckMethodMeow     = "meow"
	AstraCheckMethodSolJuice = "sol_juice"
)

// AstraCheckTarget 是某个平台可选的预期模型、判定它的方法，以及 meow 方法所用的基准包。
type AstraCheckTarget struct {
	ID                  string `json:"id"`
	DisplayName         string `json:"display_name"`
	Platform            string `json:"platform"`
	DefaultRequestModel string `json:"default_request_model"`
	Method              string `json:"method"`
	PackageID           string `json:"package_id"`
}

// astraCheckTargets 是固定的目标列表；meow 方法的目标必须在对应基准包里（测试钉住）。
// GPT-5.6 Sol 用 Juice 读数（Sol 回 40）：官方渠道下它在 meow 基准里的答案分布会漂移、被判成其他模型，
// 而 Juice 一条请求就能把 Sol / Terra / Luna 区分开。
var astraCheckTargets = []AstraCheckTarget{
	{ID: "gpt-5.6-sol", DisplayName: "GPT-5.6 Sol", Platform: PlatformOpenAI, DefaultRequestModel: "gpt-5.6-sol", Method: AstraCheckMethodSolJuice},
	{ID: "gpt-6-sol", DisplayName: "GPT-6 Sol", Platform: PlatformOpenAI, DefaultRequestModel: "gpt-6-sol", Method: AstraCheckMethodMeow, PackageID: "meow-gpt-other-cap98-efficient"},
	{ID: "gpt-6-astra", DisplayName: "GPT-6 Astra", Platform: PlatformOpenAI, DefaultRequestModel: "gpt-6-astra", Method: AstraCheckMethodMeow, PackageID: "meow-gpt-other-cap98-efficient"},
	{ID: "claude-opus-5.5", DisplayName: "Claude Opus 5.5", Platform: PlatformAnthropic, DefaultRequestModel: "claude-opus-5-5", Method: AstraCheckMethodMeow, PackageID: "meow-claude-other-cap98-efficient"},
	{ID: "claude-fable-5.1", DisplayName: "Claude Fable 5.1", Platform: PlatformAnthropic, DefaultRequestModel: "claude-fable-5-1", Method: AstraCheckMethodMeow, PackageID: "meow-claude-other-cap98-efficient"},
}

// astraModelLabels 是包内候选的可读名（含非目标的候选，用于「强指向 X」）。
var astraModelLabels = map[string]string{
	"gpt-6-astra":        "GPT-6 Astra",
	"gpt-6-sol":          "GPT-6 Sol",
	"gpt-6-luna":         "GPT-6 Luna",
	"gpt-5.6-sol":        "GPT-5.6 Sol",
	"gpt-5.6-terra":      "GPT-5.6 Terra",
	"gpt-5.6-luna":       "GPT-5.6 Luna",
	"gpt-5.5":            "GPT-5.5 / GPT-5.4",
	"gpt-5.4-mini":       "GPT-5.4 mini",
	"claude-opus-5.5":    "Claude Opus 5.5",
	"claude-fable-5.1":   "Claude Fable 5.1",
	"claude-sonnet-5":    "Claude Sonnet 5",
	"claude-haiku-4.5":   "Claude Haiku 4.5",
	AstraCheckOtherModel: "其他模型",
}

// AstraModelLabel 把候选 id 转成可读名；未知 id 原样返回。
func AstraModelLabel(model string) string {
	model = strings.TrimSpace(model)
	if model == "" {
		return "?"
	}
	if label, ok := astraModelLabels[model]; ok {
		return label
	}
	return model
}

// AstraCheckTargetsForPlatform 返回某平台可选的预期模型（无则为空）。
func AstraCheckTargetsForPlatform(platform string) []AstraCheckTarget {
	out := make([]AstraCheckTarget, 0, len(astraCheckTargets))
	for _, target := range astraCheckTargets {
		if target.Platform == platform {
			out = append(out, target)
		}
	}
	return out
}

// astraCheckTarget 返回指定 id 的目标。
func astraCheckTarget(id string) (AstraCheckTarget, bool) {
	for _, target := range astraCheckTargets {
		if target.ID == id {
			return target, true
		}
	}
	return AstraCheckTarget{}, false
}

// astraCheckSupportsPlatform 报告该平台的分组能否开启指纹验证。
func astraCheckSupportsPlatform(platform string) bool {
	return len(AstraCheckTargetsForPlatform(platform)) > 0
}

// astraCheckTargetAllowed 报告预期模型是否是该平台的目标。
func astraCheckTargetAllowed(platform, model string) bool {
	target, ok := astraCheckTarget(model)
	return ok && target.Platform == platform
}
