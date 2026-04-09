package service

import (
	"encoding/json"
	"fmt"
	"strings"
)

const (
	ModelMirrorProbePassModeAny = "any"
	ModelMirrorProbePassModeAll = "all"
)

type ModelMirrorKnowledgeProbe struct {
	ID               string   `json:"id"`
	Prompt           string   `json:"prompt"`
	ExpectedKeywords []string `json:"expected_keywords"`
	PassMode         string   `json:"pass_mode"`
	Weight           int      `json:"weight"`
	Enabled          bool     `json:"enabled"`
}

type ModelMirrorVerifyRequest struct {
	APIEndpoint string `json:"api_endpoint"`
	APIKey      string `json:"api_key"`
	APIModel    string `json:"api_model"`
}

type ModelMirrorCheckResult struct {
	ID     string `json:"id"`
	Label  string `json:"label"`
	Weight int    `json:"weight"`
	Pass   bool   `json:"pass"`
	Detail string `json:"detail"`
	Status string `json:"status,omitempty"`
}

type ModelMirrorVerdict string

const (
	ModelMirrorVerdictPending         ModelMirrorVerdict = "pending"
	ModelMirrorVerdictMaxPure         ModelMirrorVerdict = "max_pure"
	ModelMirrorVerdictOfficialAPI     ModelMirrorVerdict = "official_api"
	ModelMirrorVerdictReverseProxy    ModelMirrorVerdict = "reverse_proxy"
	ModelMirrorVerdictLikelyNotClaude ModelMirrorVerdict = "likely_not_claude"
)

func DefaultModelMirrorKnowledgeProbes() []ModelMirrorKnowledgeProbe {
	return []ModelMirrorKnowledgeProbe{
		{
			ID:     "papal-succession-2025",
			Prompt: "2025 年 5 月接替教皇方济各的新教皇是谁？他是哪国人？请直接回答，不允许联网。",
			ExpectedKeywords: []string{
				"利奥十四",
				"leo xiv",
				"robert prevost",
				"美国",
				"american",
			},
			PassMode: ModelMirrorProbePassModeAny,
			Weight:   10,
			Enabled:  true,
		}
	}
}

func ValidateModelMirrorKnowledgeProbes(input []ModelMirrorKnowledgeProbe) ([]ModelMirrorKnowledgeProbe, error) {
	if len(input) == 0 {
		return nil, nil
	}

	normalized := make([]ModelMirrorKnowledgeProbe, 0, len(input))
	seenIDs := make(map[string]struct{}, len(input))

	for index, probe := range input {
		prompt := strings.TrimSpace(probe.Prompt)
		if prompt == "" {
			return nil, fmt.Errorf("probe %d prompt is required", index+1)
		}

		keywords := normalizeModelMirrorKeywords(probe.ExpectedKeywords)
		if len(keywords) == 0 {
			return nil, fmt.Errorf("probe %d expected_keywords is required", index+1)
		}

		id := strings.TrimSpace(probe.ID)
		if id == "" {
			id = fmt.Sprintf("probe-%d", index+1)
		}
		if _, exists := seenIDs[id]; exists {
			return nil, fmt.Errorf("duplicate probe id: %s", id)
		}
		seenIDs[id] = struct{}{}

		passMode := strings.ToLower(strings.TrimSpace(probe.PassMode))
		switch passMode {
		case "", ModelMirrorProbePassModeAny:
			passMode = ModelMirrorProbePassModeAny
		case ModelMirrorProbePassModeAll:
		default:
			return nil, fmt.Errorf("probe %s pass_mode must be 'any' or 'all'", id)
		}

		weight := probe.Weight
		if weight <= 0 {
			weight = 10
		}
		if weight > 100 {
			return nil, fmt.Errorf("probe %s weight must be between 1 and 100", id)
		}

		normalized = append(normalized, ModelMirrorKnowledgeProbe{
			ID:               id,
			Prompt:           prompt,
			ExpectedKeywords: keywords,
			PassMode:         passMode,
			Weight:           weight,
			Enabled:          probe.Enabled,
		})
	}

	return normalized, nil
}

func parseModelMirrorKnowledgeProbes(raw string) []ModelMirrorKnowledgeProbe {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "[]" {
		return DefaultModelMirrorKnowledgeProbes()
	}

	var probes []ModelMirrorKnowledgeProbe
	if err := json.Unmarshal([]byte(raw), &probes); err != nil {
		return DefaultModelMirrorKnowledgeProbes()
	}

	normalized, err := ValidateModelMirrorKnowledgeProbes(probes)
	if err != nil || len(normalized) == 0 {
		return DefaultModelMirrorKnowledgeProbes()
	}

	return normalized
}

func normalizeModelMirrorKeywords(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	normalized := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		keyword := strings.ToLower(strings.TrimSpace(value))
		if keyword == "" {
			continue
		}
		if _, exists := seen[keyword]; exists {
			continue
		}
		seen[keyword] = struct{}{}
		normalized = append(normalized, keyword)
	}
	return normalized
}

func copyModelMirrorKnowledgeProbes(input []ModelMirrorKnowledgeProbe) []ModelMirrorKnowledgeProbe {
	if len(input) == 0 {
		return nil
	}
	output := make([]ModelMirrorKnowledgeProbe, 0, len(input))
	for _, probe := range input {
		output = append(output, ModelMirrorKnowledgeProbe{
			ID:               probe.ID,
			Prompt:           probe.Prompt,
			ExpectedKeywords: append([]string(nil), probe.ExpectedKeywords...),
			PassMode:         probe.PassMode,
			Weight:           probe.Weight,
			Enabled:          probe.Enabled,
		})
	}
	return output
}
