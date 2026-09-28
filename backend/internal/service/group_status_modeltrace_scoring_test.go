package service

import (
	"encoding/json"
	"math"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

// 与 ModelTrace 参考 JS 评分器逐位对齐的 golden 测试；数据由 testdata/modeltrace/gen_golden.mjs 生成。

const modelTraceParityTolerance = 1e-9

type modelTraceGoldenFile struct {
	BankSHA256 string `json:"bank_sha256"`
	Cases      []struct {
		Name            string `json:"name"`
		ExpectedModel   string `json:"expected_model"`
		ExpectedVerdict string `json:"expected_verdict"`
		Outputs         []struct {
			Text          string `json:"text"`
			ExpectedCount int    `json:"expected_count"`
		} `json:"outputs"`
		Prediction         string  `json:"prediction"`
		UsedOutputs        int     `json:"used_outputs"`
		CalibrationQueries int     `json:"calibration_queries"`
		Beta               float64 `json:"beta"`
		Diagnostics        []struct {
			ParsedNumbers  int  `json:"parsed_numbers"`
			MinimumNumbers int  `json:"minimum_numbers"`
			Accepted       bool `json:"accepted"`
		} `json:"diagnostics"`
		Results []struct {
			Model                  string  `json:"model"`
			Probability            float64 `json:"probability"`
			Score                  float64 `json:"score"`
			ConditionalProbability float64 `json:"conditional_probability"`
		} `json:"results"`
		FamilyProbabilities []struct {
			Family      string  `json:"family"`
			Probability float64 `json:"probability"`
		} `json:"family_probabilities"`
	} `json:"cases"`
	ParserCases []struct {
		Text    string `json:"text"`
		Numbers []int  `json:"numbers"`
	} `json:"parser_cases"`
}

func loadModelTraceGolden(t *testing.T) *modelTraceGoldenFile {
	t.Helper()
	raw, err := os.ReadFile("testdata/modeltrace/reference_cases.json")
	require.NoError(t, err)
	var golden modelTraceGoldenFile
	require.NoError(t, json.Unmarshal(raw, &golden))
	return &golden
}

func TestModelTraceGolden_MatchesEmbeddedBank(t *testing.T) {
	golden := loadModelTraceGolden(t)
	_, meta, err := LoadEmbeddedModelTraceBank()
	require.NoError(t, err)
	if meta.SHA256 != golden.BankSHA256 {
		t.Fatalf("embedded bank sha256 %s != golden %s; regenerate with: node backend/internal/service/testdata/modeltrace/gen_golden.mjs <ModelTrace dir> > backend/internal/service/testdata/modeltrace/reference_cases.json", meta.SHA256, golden.BankSHA256)
	}
}

func TestModelTraceGolden_ScoringParity(t *testing.T) {
	golden := loadModelTraceGolden(t)
	bank, _, err := LoadEmbeddedModelTraceBank()
	require.NoError(t, err)

	for _, tc := range golden.Cases {
		t.Run(tc.Name, func(t *testing.T) {
			outputs := make([]ModelTraceOutput, 0, len(tc.Outputs))
			for _, out := range tc.Outputs {
				outputs = append(outputs, ModelTraceOutput{Text: out.Text, ExpectedCount: out.ExpectedCount})
			}
			analysis, err := bank.Analyze(outputs)
			require.NoError(t, err)

			require.Equal(t, tc.Prediction, analysis.Top().Model)
			require.Equal(t, tc.UsedOutputs, analysis.UsedOutputs)
			require.Equal(t, tc.CalibrationQueries, analysis.CalibrationKey)
			require.InDelta(t, tc.Beta, analysis.Beta, modelTraceParityTolerance)

			require.Len(t, analysis.Diagnostics, len(tc.Diagnostics))
			for i, diag := range tc.Diagnostics {
				require.Equal(t, diag.ParsedNumbers, analysis.Diagnostics[i].ParsedNumbers, "diagnostic %d parsed", i)
				require.Equal(t, diag.MinimumNumbers, analysis.Diagnostics[i].MinimumNumbers, "diagnostic %d minimum", i)
				require.Equal(t, diag.Accepted, analysis.Diagnostics[i].Accepted, "diagnostic %d accepted", i)
			}

			require.Len(t, analysis.Results, len(tc.Results))
			for _, want := range tc.Results {
				got := analysis.Find(want.Model)
				require.NotNil(t, got, want.Model)
				require.InDelta(t, want.Probability, got.Probability, modelTraceParityTolerance, "%s probability", want.Model)
				require.InDelta(t, want.Score, got.Score, modelTraceParityTolerance, "%s score", want.Model)
				require.InDelta(t, want.ConditionalProbability, got.ConditionalProbability, modelTraceParityTolerance, "%s conditional", want.Model)
			}
			for _, want := range tc.FamilyProbabilities {
				found := false
				for _, got := range analysis.Families {
					if got.Family == want.Family {
						found = true
						require.InDelta(t, want.Probability, got.Probability, modelTraceParityTolerance, "family %s", want.Family)
					}
				}
				require.True(t, found, "family %s", want.Family)
			}

			if tc.ExpectedVerdict != "" {
				verdict, _ := ClassifyModelTrace(analysis, bank, tc.ExpectedModel)
				require.Equal(t, tc.ExpectedVerdict, verdict)
			}
		})
	}
}

func TestModelTraceGolden_EveryBankModelHasASelfCase(t *testing.T) {
	golden := loadModelTraceGolden(t)
	bank, _, err := LoadEmbeddedModelTraceBank()
	require.NoError(t, err)
	covered := map[string]bool{}
	for _, tc := range golden.Cases {
		if tc.Name == "self-"+tc.ExpectedModel {
			covered[tc.ExpectedModel] = true
		}
	}
	for _, model := range bank.Models {
		require.True(t, covered[model.ID], "golden fixtures lack a self case for %s; regenerate them after updating the bank", model.ID)
	}
}

func TestModelTraceGolden_ParserParity(t *testing.T) {
	golden := loadModelTraceGolden(t)
	require.NotEmpty(t, golden.ParserCases)
	for _, tc := range golden.ParserCases {
		got := ParseModelTraceNumbers(tc.Text)
		if len(tc.Numbers) == 0 {
			require.Empty(t, got, tc.Text)
			continue
		}
		require.Equal(t, tc.Numbers, got, tc.Text)
	}
}

func TestModelTraceAnalyzeNumbers_MatchesTextAnalysis(t *testing.T) {
	golden := loadModelTraceGolden(t)
	bank, _, err := LoadEmbeddedModelTraceBank()
	require.NoError(t, err)
	tc := golden.Cases[0]
	var outputs []ModelTraceOutput
	var sequences [][]int
	for _, out := range tc.Outputs {
		outputs = append(outputs, ModelTraceOutput{Text: out.Text, ExpectedCount: out.ExpectedCount})
		sequences = append(sequences, ParseModelTraceNumbers(out.Text))
	}
	fromText, err := bank.Analyze(outputs)
	require.NoError(t, err)
	fromNumbers, err := bank.AnalyzeNumbers(sequences)
	require.NoError(t, err)
	for i := range fromText.Results {
		require.Equal(t, fromText.Results[i].Model, fromNumbers.Results[i].Model)
		require.False(t, math.Abs(fromText.Results[i].Probability-fromNumbers.Results[i].Probability) > 0)
	}
}

func TestModelTraceAnalyze_NoValidOutputs(t *testing.T) {
	bank, _, err := LoadEmbeddedModelTraceBank()
	require.NoError(t, err)
	analysis, err := bank.Analyze([]ModelTraceOutput{{Text: "I can't help with that.", ExpectedCount: 300}})
	require.ErrorIs(t, err, ErrModelTraceNoValidOutputs)
	require.Len(t, analysis.Diagnostics, 1)
	require.False(t, analysis.Diagnostics[0].Accepted)
	require.Equal(t, 165, analysis.Diagnostics[0].MinimumNumbers)
}
