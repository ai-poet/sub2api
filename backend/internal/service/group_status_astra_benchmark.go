package service

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	astrabenchmark "github.com/Wei-Shaw/sub2api/resources/astra-benchmark"
)

// Astra 指纹验证的基准包（meow LLM detector 的 *.meow.json），本 fork 自有功能。
//
// 包内已经给出四个候选模型在每道题上的平滑分布、题族权重和各档阈值；这里只做解析、校验
// 与内存缓存，判定逻辑见 group_status_astra_check.go。

const (
	astraBenchmarkScoringVersion = "meow-fingerprint-v2"
	astraBenchmarkMode           = "gpt"
	astraCheckClaimedModel       = "gpt-6-astra"

	AstraCheckTierLow    = "low"
	AstraCheckTierMedium = "medium"
	AstraCheckTierHigh   = "high"
)

var (
	ErrGroupStatusAstraBenchmarkInvalid = infraerrors.ServiceUnavailable("GROUP_STATUS_ASTRA_BENCHMARK_INVALID", "embedded Astra benchmark package is invalid")

	astraCheckTiers = []string{AstraCheckTierLow, AstraCheckTierMedium, AstraCheckTierHigh}
)

type AstraBenchmarkModel struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	RequestModel string `json:"request_model"`
}

// AstraNormalizer 描述某道题答案的归一化规则（与基准采集时一致）。
type AstraNormalizer struct {
	ID        string
	MaxLength int
	Values    map[string]string
}

type AstraBenchmarkCell struct {
	ID              string
	ProbeID         string
	FamilyID        string
	System          string
	Prompt          string
	Effort          string
	MaxOutputTokens int
	Normalizer      AstraNormalizer
}

type AstraBenchmarkTier struct {
	Counts        map[string]int
	Thresholds    map[string]float64
	Calibrated    bool
	TotalRequests int
}

type AstraFittedCell struct {
	Categories     []string
	Distributions  map[string]map[string]float64
	Weight         float64
	FamilyID       string
	ReferenceReady bool
}

// AstraBenchmark 是解析并校验后的基准包。
type AstraBenchmark struct {
	PackageID      string
	Version        string
	Mode           string
	ScoringVersion string
	ContentSHA256  string
	BodySHA256     string
	Models         []AstraBenchmarkModel
	ModelIDs       []string
	Cells          []AstraBenchmarkCell
	CellIndex      map[string]*AstraBenchmarkCell
	Tiers          map[string]AstraBenchmarkTier
	Fitted         map[string]AstraFittedCell
}

type AstraBenchmarkTierMeta struct {
	Tier       string `json:"tier"`
	Requests   int    `json:"requests"`
	Calibrated bool   `json:"calibrated"`
}

// AstraBenchmarkMeta 是给管理端展示的基准包元数据（不含题面与分布）。
type AstraBenchmarkMeta struct {
	PackageID     string                   `json:"package_id"`
	Version       string                   `json:"version"`
	ContentSHA256 string                   `json:"content_sha256"`
	BodySHA256    string                   `json:"body_sha256"`
	Models        []AstraBenchmarkModel    `json:"models"`
	Tiers         []AstraBenchmarkTierMeta `json:"tiers"`
}

// astraTierCounts 兼容「每格次数 map」和「所有格同一次数」两种写法。
type astraTierCounts struct {
	perCell map[string]int
	uniform int
	set     bool
}

func (c *astraTierCounts) UnmarshalJSON(data []byte) error {
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" || trimmed == "null" {
		return nil
	}
	if strings.HasPrefix(trimmed, "{") {
		var m map[string]int
		if err := json.Unmarshal(data, &m); err != nil {
			return err
		}
		c.perCell = m
		c.set = true
		return nil
	}
	var n int
	if err := json.Unmarshal(data, &n); err != nil {
		return err
	}
	c.uniform = n
	c.set = true
	return nil
}

type astraRawPackage struct {
	Mode          string `json:"mode"`
	ID            string `json:"id"`
	Version       string `json:"version"`
	ContentSHA256 string `json:"content_sha256"`
	Engine        struct {
		ScoringVersion string `json:"scoring_version"`
	} `json:"engine"`
	Models []AstraBenchmarkModel `json:"models"`
	Probes []struct {
		ID         string `json:"id"`
		FamilyID   string `json:"family_id"`
		Normalizer struct {
			ID         string `json:"id"`
			Parameters struct {
				MaxLength int             `json:"max_length"`
				Values    json.RawMessage `json:"values"`
			} `json:"parameters"`
		} `json:"normalizer"`
		Cells []struct {
			ID         string `json:"id"`
			System     string `json:"system"`
			Prompt     string `json:"prompt"`
			Effort     string `json:"effort"`
			Parameters struct {
				MaxOutputTokens int `json:"max_output_tokens"`
			} `json:"parameters"`
		} `json:"cells"`
	} `json:"probes"`
	Tiers map[string]struct {
		Counts     astraTierCounts    `json:"counts"`
		Thresholds map[string]float64 `json:"thresholds"`
	} `json:"tiers"`
	Fitted struct {
		Models []string `json:"models"`
		Cells  map[string]struct {
			Categories         []string                      `json:"categories"`
			ModelDistributions map[string]map[string]float64 `json:"model_distributions"`
			Weight             float64                       `json:"weight"`
			FamilyID           string                        `json:"family_id"`
			ReferenceReady     bool                          `json:"reference_ready"`
		} `json:"cells"`
	} `json:"fitted"`
	Calibration struct {
		Tiers map[string]struct {
			Status string `json:"status"`
		} `json:"tiers"`
	} `json:"calibration"`
}

// ParseAstraBenchmark 解析并校验一个 meow 基准包。
func ParseAstraBenchmark(raw []byte) (*AstraBenchmark, *AstraBenchmarkMeta, error) {
	if len(raw) == 0 {
		return nil, nil, errors.New("astra benchmark: empty package")
	}
	var pkg astraRawPackage
	if err := json.Unmarshal(raw, &pkg); err != nil {
		return nil, nil, fmt.Errorf("astra benchmark: decode: %w", err)
	}
	if pkg.Mode != astraBenchmarkMode {
		return nil, nil, fmt.Errorf("astra benchmark: mode must be %q, got %q", astraBenchmarkMode, pkg.Mode)
	}
	if pkg.Engine.ScoringVersion != astraBenchmarkScoringVersion {
		return nil, nil, fmt.Errorf("astra benchmark: unsupported scoring_version %q", pkg.Engine.ScoringVersion)
	}
	if len(pkg.Models) == 0 {
		return nil, nil, errors.New("astra benchmark: models is empty")
	}

	bench := &AstraBenchmark{
		PackageID:      strings.TrimSpace(pkg.ID),
		Version:        strings.TrimSpace(pkg.Version),
		Mode:           pkg.Mode,
		ScoringVersion: pkg.Engine.ScoringVersion,
		ContentSHA256:  strings.TrimSpace(pkg.ContentSHA256),
		CellIndex:      make(map[string]*AstraBenchmarkCell),
		Tiers:          make(map[string]AstraBenchmarkTier, len(astraCheckTiers)),
		Fitted:         make(map[string]AstraFittedCell, len(pkg.Fitted.Cells)),
	}
	sum := sha256.Sum256(raw)
	bench.BodySHA256 = hex.EncodeToString(sum[:])
	if bench.PackageID == "" {
		bench.PackageID = "astra-benchmark-" + bench.BodySHA256[:12]
	}
	if bench.Version == "" {
		bench.Version = bench.BodySHA256[:12]
	}

	modelSet := make(map[string]struct{}, len(pkg.Models))
	for _, model := range pkg.Models {
		id := strings.TrimSpace(model.ID)
		if id == "" {
			return nil, nil, errors.New("astra benchmark: model id is empty")
		}
		if _, dup := modelSet[id]; dup {
			return nil, nil, fmt.Errorf("astra benchmark: duplicate model %q", id)
		}
		modelSet[id] = struct{}{}
		name := strings.TrimSpace(model.Name)
		if name == "" {
			name = id
		}
		bench.Models = append(bench.Models, AstraBenchmarkModel{ID: id, Name: name, RequestModel: strings.TrimSpace(model.RequestModel)})
		bench.ModelIDs = append(bench.ModelIDs, id)
	}
	if _, ok := modelSet[astraCheckClaimedModel]; !ok {
		return nil, nil, fmt.Errorf("astra benchmark: package does not contain %s", astraCheckClaimedModel)
	}

	for _, probe := range pkg.Probes {
		probeID := strings.TrimSpace(probe.ID)
		if probeID == "" || len(probe.Cells) == 0 {
			return nil, nil, errors.New("astra benchmark: probe without id or cells")
		}
		normalizer := AstraNormalizer{
			ID:        strings.TrimSpace(probe.Normalizer.ID),
			MaxLength: probe.Normalizer.Parameters.MaxLength,
		}
		if normalizer.ID == "" {
			normalizer.ID = "exact_trimmed_casefold"
		}
		if len(probe.Normalizer.Parameters.Values) > 0 {
			values, err := parseAstraEnumValues(probe.Normalizer.Parameters.Values)
			if err != nil {
				return nil, nil, fmt.Errorf("astra benchmark: probe %s normalizer values: %w", probeID, err)
			}
			normalizer.Values = values
		}
		familyID := strings.TrimSpace(probe.FamilyID)
		if familyID == "" {
			familyID = probeID
		}
		for _, cell := range probe.Cells {
			cellID := strings.TrimSpace(cell.ID)
			if cellID == "" || strings.TrimSpace(cell.Prompt) == "" {
				return nil, nil, fmt.Errorf("astra benchmark: probe %s has a cell without id or prompt", probeID)
			}
			if _, dup := bench.CellIndex[cellID]; dup {
				return nil, nil, fmt.Errorf("astra benchmark: duplicate cell %q", cellID)
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
				FamilyID:        familyID,
				System:          cell.System,
				Prompt:          cell.Prompt,
				Effort:          effort,
				MaxOutputTokens: maxOutput,
				Normalizer:      normalizer,
			})
		}
	}
	if len(bench.Cells) == 0 {
		return nil, nil, errors.New("astra benchmark: no cells")
	}
	for i := range bench.Cells {
		bench.CellIndex[bench.Cells[i].ID] = &bench.Cells[i]
	}

	weighted := 0
	for cellID, raw := range pkg.Fitted.Cells {
		if len(raw.Categories) == 0 {
			return nil, nil, fmt.Errorf("astra benchmark: fitted cell %s has no categories", cellID)
		}
		dists := make(map[string]map[string]float64, len(bench.ModelIDs))
		for _, modelID := range bench.ModelIDs {
			dist, ok := raw.ModelDistributions[modelID]
			if !ok || len(dist) == 0 {
				return nil, nil, fmt.Errorf("astra benchmark: fitted cell %s lacks distribution for %s", cellID, modelID)
			}
			dists[modelID] = dist
		}
		familyID := strings.TrimSpace(raw.FamilyID)
		if familyID == "" {
			if cell, ok := bench.CellIndex[cellID]; ok {
				familyID = cell.FamilyID
			} else {
				familyID = cellID
			}
		}
		if raw.Weight > 0 {
			weighted++
		}
		bench.Fitted[cellID] = AstraFittedCell{
			Categories:     append([]string(nil), raw.Categories...),
			Distributions:  dists,
			Weight:         raw.Weight,
			FamilyID:       familyID,
			ReferenceReady: raw.ReferenceReady,
		}
	}
	for _, cell := range bench.Cells {
		if _, ok := bench.Fitted[cell.ID]; !ok {
			return nil, nil, fmt.Errorf("astra benchmark: cell %s has no fitted distribution", cell.ID)
		}
	}
	if weighted == 0 {
		return nil, nil, errors.New("astra benchmark: no fitted cell has a positive weight")
	}

	for _, tier := range astraCheckTiers {
		raw, ok := pkg.Tiers[tier]
		if !ok || !raw.Counts.set {
			return nil, nil, fmt.Errorf("astra benchmark: tier %s is missing", tier)
		}
		counts := make(map[string]int, len(bench.Cells))
		total := 0
		for _, cell := range bench.Cells {
			n := raw.Counts.uniform
			if raw.Counts.perCell != nil {
				n = raw.Counts.perCell[cell.ID]
			}
			if n < 0 {
				n = 0
			}
			counts[cell.ID] = n
			total += n
		}
		if total == 0 {
			return nil, nil, fmt.Errorf("astra benchmark: tier %s plans zero requests", tier)
		}
		thresholds := make(map[string]float64, len(bench.ModelIDs))
		calibrated := strings.EqualFold(strings.TrimSpace(pkg.Calibration.Tiers[tier].Status), "target_met")
		for _, modelID := range bench.ModelIDs {
			value, ok := raw.Thresholds[modelID]
			if !ok {
				calibrated = false
				continue
			}
			thresholds[modelID] = value
		}
		if len(raw.Thresholds) != len(bench.ModelIDs) {
			calibrated = false
		}
		bench.Tiers[tier] = AstraBenchmarkTier{
			Counts:        counts,
			Thresholds:    thresholds,
			Calibrated:    calibrated,
			TotalRequests: total,
		}
	}

	meta := &AstraBenchmarkMeta{
		PackageID:     bench.PackageID,
		Version:       bench.Version,
		ContentSHA256: bench.ContentSHA256,
		BodySHA256:    bench.BodySHA256,
		Models:        append([]AstraBenchmarkModel(nil), bench.Models...),
	}
	for _, tier := range astraCheckTiers {
		meta.Tiers = append(meta.Tiers, AstraBenchmarkTierMeta{
			Tier:       tier,
			Requests:   bench.Tiers[tier].TotalRequests,
			Calibrated: bench.Tiers[tier].Calibrated,
		})
	}
	return bench, meta, nil
}

func parseAstraEnumValues(raw json.RawMessage) (map[string]string, error) {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" {
		return nil, nil
	}
	if strings.HasPrefix(trimmed, "{") {
		var m map[string]string
		if err := json.Unmarshal(raw, &m); err != nil {
			return nil, err
		}
		return m, nil
	}
	var list []string
	if err := json.Unmarshal(raw, &list); err != nil {
		return nil, err
	}
	m := make(map[string]string, len(list))
	for _, item := range list {
		m[item] = item
	}
	return m, nil
}

// SortedModelIDs 返回稳定顺序的模型 ID（包内顺序）。
func (b *AstraBenchmark) TierRequests(tier string) int {
	if b == nil {
		return 0
	}
	return b.Tiers[tier].TotalRequests
}

// modelName 返回模型显示名，未知时原样返回 id。
func (b *AstraBenchmark) modelName(modelID string) string {
	if b != nil {
		for _, model := range b.Models {
			if model.ID == modelID {
				return model.Name
			}
		}
	}
	return modelID
}

var (
	embeddedAstraBenchmarkOnce sync.Once
	embeddedAstraBenchmark     *AstraBenchmark
	embeddedAstraBenchmarkMeta *AstraBenchmarkMeta
	embeddedAstraBenchmarkErr  error
)

// LoadEmbeddedAstraBenchmark 解析随二进制内置的基准包；只解析一次，失败则每次返回同一错误。
func LoadEmbeddedAstraBenchmark() (*AstraBenchmark, *AstraBenchmarkMeta, error) {
	embeddedAstraBenchmarkOnce.Do(func() {
		bench, meta, err := ParseAstraBenchmark(astrabenchmark.Package)
		if err != nil {
			embeddedAstraBenchmarkErr = fmt.Errorf("%w: %s: %v", ErrGroupStatusAstraBenchmarkInvalid, astrabenchmark.FileName, err)
			return
		}
		embeddedAstraBenchmark, embeddedAstraBenchmarkMeta = bench, meta
	})
	if embeddedAstraBenchmarkErr != nil {
		return nil, nil, embeddedAstraBenchmarkErr
	}
	return embeddedAstraBenchmark, embeddedAstraBenchmarkMeta, nil
}

// astraBenchmarkProvider 抽象基准来源；默认用内置包，测试注入合成包。
type astraBenchmarkProvider interface {
	Active() (*AstraBenchmark, *AstraBenchmarkMeta, error)
}

type embeddedAstraBenchmarkProvider struct{}

func (embeddedAstraBenchmarkProvider) Active() (*AstraBenchmark, *AstraBenchmarkMeta, error) {
	return LoadEmbeddedAstraBenchmark()
}

// astraBenchmarkFamilyOrder 返回题族的稳定顺序，供聚合与展示使用。
func (b *AstraBenchmark) familyOrder() []string {
	seen := make(map[string]struct{})
	var out []string
	for _, cell := range b.Cells {
		fitted, ok := b.Fitted[cell.ID]
		if !ok {
			continue
		}
		if _, dup := seen[fitted.FamilyID]; dup {
			continue
		}
		seen[fitted.FamilyID] = struct{}{}
		out = append(out, fitted.FamilyID)
	}
	sort.Strings(out)
	return out
}
