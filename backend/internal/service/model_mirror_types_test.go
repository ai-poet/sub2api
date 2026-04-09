package service

import "testing"

func TestValidateModelMirrorKnowledgeProbesNormalizesInput(t *testing.T) {
	input := []ModelMirrorKnowledgeProbe{
		{
			Prompt:           "  latest fact?  ",
			ExpectedKeywords: []string{" Anthropic ", "anthropic", " Claude "},
			PassMode:         "",
			Weight:           0,
			Enabled:          true,
		},
	}

	normalized, err := ValidateModelMirrorKnowledgeProbes(input)
	if err != nil {
		t.Fatalf("ValidateModelMirrorKnowledgeProbes returned error: %v", err)
	}
	if len(normalized) != 1 {
		t.Fatalf("expected 1 probe, got %d", len(normalized))
	}

	probe := normalized[0]
	if probe.ID != "probe-1" {
		t.Fatalf("expected generated id probe-1, got %q", probe.ID)
	}
	if probe.Prompt != "latest fact?" {
		t.Fatalf("expected trimmed prompt, got %q", probe.Prompt)
	}
	if probe.PassMode != ModelMirrorProbePassModeAny {
		t.Fatalf("expected default pass mode %q, got %q", ModelMirrorProbePassModeAny, probe.PassMode)
	}
	if probe.Weight != 10 {
		t.Fatalf("expected default weight 10, got %d", probe.Weight)
	}
	if len(probe.ExpectedKeywords) != 2 {
		t.Fatalf("expected deduplicated keywords, got %v", probe.ExpectedKeywords)
	}
	if probe.ExpectedKeywords[0] != "anthropic" || probe.ExpectedKeywords[1] != "claude" {
		t.Fatalf("unexpected normalized keywords: %v", probe.ExpectedKeywords)
	}
}

func TestEvaluateKnowledgeProbeUsesProvidedProbe(t *testing.T) {
	probe := &ModelMirrorKnowledgeProbe{
		ID:               "custom-probe",
		ExpectedKeywords: []string{"custom-signal"},
		PassMode:         ModelMirrorProbePassModeAny,
		Weight:           17,
		Enabled:          true,
	}

	passResult := evaluateKnowledgeProbe("the relay returned CUSTOM-SIGNAL correctly", probe)
	if !passResult.Pass {
		t.Fatalf("expected provided probe to pass, got %+v", passResult)
	}
	if passResult.Weight != 17 {
		t.Fatalf("expected provided probe weight to be used, got %d", passResult.Weight)
	}

	failResult := evaluateKnowledgeProbe("leo xiv anthropic", probe)
	if failResult.Pass {
		t.Fatalf("expected provided probe to fail when only default keywords match, got %+v", failResult)
	}
	if failResult.Weight != 17 {
		t.Fatalf("expected provided probe weight on failure, got %d", failResult.Weight)
	}
}
