package config

import "testing"

// TestLoadBotConfigFromSeed verifies the seed-format loader builds a usable
// BotConfig from the real configs/faq_seed.yaml.
func TestLoadBotConfigFromSeed(t *testing.T) {
	cfg, err := LoadBotConfig("../../../configs/faq_seed.yaml")
	if err != nil {
		t.Fatalf("LoadBotConfig: %v", err)
	}

	if cfg.DefaultDiagnosis != "d1" {
		t.Errorf("default_diagnosis = %q, want d1", cfg.DefaultDiagnosis)
	}
	for _, dx := range []string{"d1", "d2", "d3", "d4", "d5"} {
		d, ok := cfg.Diagnoses[dx]
		if !ok {
			t.Errorf("missing diagnosis %q", dx)
			continue
		}
		if d.Name == "" {
			t.Errorf("diagnosis %q has empty name", dx)
		}
		if len(d.FAQ) == 0 {
			t.Errorf("diagnosis %q has no FAQ items", dx)
		}
	}
	if len(cfg.SharedFAQ) == 0 {
		t.Error("shared FAQ is empty (expected dx: shared items)")
	}
	if len(cfg.Escalation.Keywords) == 0 || cfg.Escalation.Reply == "" {
		t.Error("escalation config not loaded")
	}
	if cfg.FallbackReply == "" {
		t.Error("fallback_reply not loaded")
	}

	// A known item must carry code + question as match phrases.
	d1q1, ok := cfg.Diagnoses["d1"].FAQ["D1-Q1"]
	if !ok {
		t.Fatal("D1-Q1 not loaded under d1")
	}
	if len(d1q1.MatchPhrases) < 2 {
		t.Errorf("D1-Q1 match phrases = %v, want >= 2 (code + question)", d1q1.MatchPhrases)
	}
}
