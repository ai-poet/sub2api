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
	modeltracebank "github.com/Wei-Shaw/sub2api/resources/modeltrace-bank"
)

// ModelTrace 指纹验证的指纹库（ModelTrace 的 unified_bank.json，MIT），本 fork 自有功能。
//
// 库里已经给出各模型的 Hellinger / 有序分块特征中心、干扰方向基与校准温度；这里只做解析、
// 严格校验与内存缓存，评分见 group_status_modeltrace_scoring.go。

const (
	modelTraceBankSchema        = "robust-number-fingerprint-bank"
	modelTraceValueMin          = 1
	modelTraceValueMax          = 355
	modelTraceHellingerDim      = modelTraceValueMax - modelTraceValueMin + 1 // 355
	modelTraceOrderedBlockDim   = 4*16 + 10                                   // 74
	modelTraceMaxCalibrationKey = 3
)

var ErrGroupStatusModelTraceBankInvalid = infraerrors.ServiceUnavailable("GROUP_STATUS_MODELTRACE_BANK_INVALID", "embedded ModelTrace fingerprint bank is invalid")

// ModelTraceBankModel 是指纹库里的一个候选模型。
type ModelTraceBankModel struct {
	ID            string `json:"id"`
	DisplayName   string `json:"display_name"`
	Family        string `json:"family"`
	FamilyName    string `json:"family_name"`
	ResponseCount int    `json:"response_count"`
}

// ModelTraceCalibration 是某个回答条数下的 softmax 温度。
type ModelTraceCalibration struct {
	Beta       float64 `json:"beta"`
	CVAccuracy float64 `json:"cv_accuracy"`
}

// ModelTraceBank 是解析并校验后的指纹库；各矩阵的行顺序与 Models 一致。
type ModelTraceBank struct {
	Models []ModelTraceBankModel
	index  map[string]int

	HellMean      []float64
	HellScale     []float64
	HellBasis     [][]float64
	HellCentroids [][]float64

	OrdWeight       float64
	OrdMean         []float64
	OrdScale        []float64
	OrdBasis        [][]float64
	OrdCentroids    [][]float64
	OrdEnvCentroids [][][]float64

	// Calibration 以回答条数 1..3 为键
	Calibration map[int]ModelTraceCalibration

	FamilyOrder []string
	FamilyNames map[string]string
}

// ModelTraceBankMeta 是指纹库的展示元数据。
type ModelTraceBankMeta struct {
	SHA256  string                `json:"sha256"`
	BuiltAt string                `json:"built_at"`
	Models  []ModelTraceBankModel `json:"models"`
}

// HasModel 报告指纹库是否收录该模型。
func (b *ModelTraceBank) HasModel(id string) bool {
	if b == nil {
		return false
	}
	_, ok := b.index[strings.TrimSpace(id)]
	return ok
}

// ModelIndex 返回模型在库中的下标。
func (b *ModelTraceBank) ModelIndex(id string) (int, bool) {
	if b == nil {
		return 0, false
	}
	i, ok := b.index[strings.TrimSpace(id)]
	return i, ok
}

type modelTraceBankFile struct {
	Schema  string `json:"schema"`
	BuiltAt string `json:"built_at"`
	Method  struct {
		Range []int `json:"range"`
	} `json:"method"`
	Models []ModelTraceBankModel `json:"models"`
	Robust struct {
		ModelOrder []string `json:"model_order"`
		Hellinger  struct {
			FeatureMean  []float64   `json:"feature_mean"`
			FeatureScale []float64   `json:"feature_scale"`
			Basis        [][]float64 `json:"nuisance_basis"`
			Centroids    [][]float64 `json:"centroids"`
		} `json:"hellinger"`
		OrderedBlocks *struct {
			Weight               float64       `json:"weight"`
			FeatureMean          []float64     `json:"feature_mean"`
			FeatureScale         []float64     `json:"feature_scale"`
			Basis                [][]float64   `json:"nuisance_basis"`
			Centroids            [][]float64   `json:"centroids"`
			EnvironmentCentroids [][][]float64 `json:"environment_centroids"`
		} `json:"ordered_blocks"`
	} `json:"robust"`
	Calibration map[string]ModelTraceCalibration `json:"calibration"`
}

// ParseModelTraceBank 解析并严格校验指纹库；任何维度或数值问题都拒绝整个库。
func ParseModelTraceBank(raw []byte) (*ModelTraceBank, *ModelTraceBankMeta, error) {
	var file modelTraceBankFile
	if err := json.Unmarshal(raw, &file); err != nil {
		return nil, nil, fmt.Errorf("decode bank: %w", err)
	}
	if file.Schema != modelTraceBankSchema {
		return nil, nil, fmt.Errorf("unsupported schema %q", file.Schema)
	}
	if len(file.Method.Range) != 2 || file.Method.Range[0] != modelTraceValueMin || file.Method.Range[1] != modelTraceValueMax {
		return nil, nil, fmt.Errorf("unsupported value range %v", file.Method.Range)
	}
	n := len(file.Models)
	if n < 2 {
		return nil, nil, errors.New("bank needs at least two models")
	}
	if len(file.Robust.ModelOrder) != n {
		return nil, nil, errors.New("model_order length does not match models")
	}

	bank := &ModelTraceBank{
		Models:      make([]ModelTraceBankModel, n),
		index:       make(map[string]int, n),
		Calibration: make(map[int]ModelTraceCalibration, modelTraceMaxCalibrationKey),
		FamilyNames: make(map[string]string),
	}
	for i, model := range file.Models {
		id := strings.TrimSpace(model.ID)
		if id == "" {
			return nil, nil, fmt.Errorf("model %d has empty id", i)
		}
		if _, dup := bank.index[id]; dup {
			return nil, nil, fmt.Errorf("duplicate model id %q", id)
		}
		if file.Robust.ModelOrder[i] != id {
			return nil, nil, fmt.Errorf("model_order[%d]=%q does not match models[%d]=%q", i, file.Robust.ModelOrder[i], i, id)
		}
		model.ID = id
		if strings.TrimSpace(model.DisplayName) == "" {
			model.DisplayName = id
		}
		// 与参考实现一致：缺失家族归入 "models"，家族名取该家族第一个非空 family_name
		if strings.TrimSpace(model.Family) == "" {
			model.Family = "models"
		}
		bank.Models[i] = model
		bank.index[id] = i
		if _, seen := bank.FamilyNames[model.Family]; !seen {
			bank.FamilyOrder = append(bank.FamilyOrder, model.Family)
			bank.FamilyNames[model.Family] = ""
		}
		if bank.FamilyNames[model.Family] == "" && strings.TrimSpace(model.FamilyName) != "" {
			bank.FamilyNames[model.Family] = model.FamilyName
		}
	}
	for _, family := range bank.FamilyOrder {
		if bank.FamilyNames[family] == "" {
			bank.FamilyNames[family] = family
		}
	}

	hell := file.Robust.Hellinger
	if err := checkModelTraceVector("hellinger.feature_mean", hell.FeatureMean, modelTraceHellingerDim, false); err != nil {
		return nil, nil, err
	}
	if err := checkModelTraceVector("hellinger.feature_scale", hell.FeatureScale, modelTraceHellingerDim, true); err != nil {
		return nil, nil, err
	}
	if err := checkModelTraceMatrix("hellinger.nuisance_basis", hell.Basis, -1, modelTraceHellingerDim); err != nil {
		return nil, nil, err
	}
	if err := checkModelTraceMatrix("hellinger.centroids", hell.Centroids, n, modelTraceHellingerDim); err != nil {
		return nil, nil, err
	}
	bank.HellMean, bank.HellScale, bank.HellBasis, bank.HellCentroids = hell.FeatureMean, hell.FeatureScale, hell.Basis, hell.Centroids

	if ob := file.Robust.OrderedBlocks; ob != nil {
		if math.IsNaN(ob.Weight) || ob.Weight < 0 || ob.Weight > 1 {
			return nil, nil, fmt.Errorf("ordered_blocks.weight %v out of [0,1]", ob.Weight)
		}
		bank.OrdWeight = ob.Weight
		if ob.Weight > 0 {
			if err := checkModelTraceVector("ordered_blocks.feature_mean", ob.FeatureMean, modelTraceOrderedBlockDim, false); err != nil {
				return nil, nil, err
			}
			if err := checkModelTraceVector("ordered_blocks.feature_scale", ob.FeatureScale, modelTraceOrderedBlockDim, true); err != nil {
				return nil, nil, err
			}
			if err := checkModelTraceMatrix("ordered_blocks.nuisance_basis", ob.Basis, -1, modelTraceOrderedBlockDim); err != nil {
				return nil, nil, err
			}
			if err := checkModelTraceMatrix("ordered_blocks.centroids", ob.Centroids, n, modelTraceOrderedBlockDim); err != nil {
				return nil, nil, err
			}
			if len(ob.EnvironmentCentroids) == 0 {
				return nil, nil, errors.New("ordered_blocks.environment_centroids is empty")
			}
			for e, env := range ob.EnvironmentCentroids {
				if err := checkModelTraceMatrix(fmt.Sprintf("ordered_blocks.environment_centroids[%d]", e), env, n, modelTraceOrderedBlockDim); err != nil {
					return nil, nil, err
				}
			}
			bank.OrdMean, bank.OrdScale, bank.OrdBasis = ob.FeatureMean, ob.FeatureScale, ob.Basis
			bank.OrdCentroids, bank.OrdEnvCentroids = ob.Centroids, ob.EnvironmentCentroids
		}
	}

	for key := 1; key <= modelTraceMaxCalibrationKey; key++ {
		cal, ok := file.Calibration[fmt.Sprint(key)]
		if !ok {
			return nil, nil, fmt.Errorf("calibration %d is missing", key)
		}
		if math.IsNaN(cal.Beta) || math.IsInf(cal.Beta, 0) || cal.Beta <= 0 {
			return nil, nil, fmt.Errorf("calibration %d beta %v is invalid", key, cal.Beta)
		}
		bank.Calibration[key] = cal
	}

	sum := sha256.Sum256(raw)
	meta := &ModelTraceBankMeta{
		SHA256:  hex.EncodeToString(sum[:]),
		BuiltAt: file.BuiltAt,
		Models:  append([]ModelTraceBankModel(nil), bank.Models...),
	}
	return bank, meta, nil
}

func checkModelTraceVector(name string, values []float64, dim int, nonZero bool) error {
	if len(values) != dim {
		return fmt.Errorf("%s has %d values, want %d", name, len(values), dim)
	}
	for i, v := range values {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return fmt.Errorf("%s[%d] is not finite", name, i)
		}
		if nonZero && v == 0 {
			return fmt.Errorf("%s[%d] is zero", name, i)
		}
	}
	return nil
}

// checkModelTraceMatrix 校验 rows×dim 矩阵；rows<0 表示行数不限（干扰方向基可以为空）。
func checkModelTraceMatrix(name string, rows [][]float64, wantRows, dim int) error {
	if wantRows >= 0 && len(rows) != wantRows {
		return fmt.Errorf("%s has %d rows, want %d", name, len(rows), wantRows)
	}
	for i, row := range rows {
		if err := checkModelTraceVector(fmt.Sprintf("%s[%d]", name, i), row, dim, false); err != nil {
			return err
		}
	}
	return nil
}

var (
	embeddedModelTraceBankOnce sync.Once
	embeddedModelTraceBank     *ModelTraceBank
	embeddedModelTraceBankMeta *ModelTraceBankMeta
	embeddedModelTraceBankErr  error
)

// LoadEmbeddedModelTraceBank 解析一次内置指纹库并缓存结果。
func LoadEmbeddedModelTraceBank() (*ModelTraceBank, *ModelTraceBankMeta, error) {
	embeddedModelTraceBankOnce.Do(func() {
		bank, meta, err := ParseModelTraceBank(modeltracebank.Bank)
		if err != nil {
			embeddedModelTraceBankErr = fmt.Errorf("%w: %s: %v", ErrGroupStatusModelTraceBankInvalid, modeltracebank.FileName, err)
			return
		}
		embeddedModelTraceBank, embeddedModelTraceBankMeta = bank, meta
	})
	if embeddedModelTraceBankErr != nil {
		return nil, nil, embeddedModelTraceBankErr
	}
	return embeddedModelTraceBank, embeddedModelTraceBankMeta, nil
}

// modelTraceBankProvider 抽象指纹库来源；默认用内置库，测试可注入。
type modelTraceBankProvider interface {
	Active() (*ModelTraceBank, *ModelTraceBankMeta, error)
}
