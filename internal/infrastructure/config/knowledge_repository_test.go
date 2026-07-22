package config

import (
	"os"
	"strings"
	"testing"

	"github.com/yimsoijoi/rama-chatbot/internal/domain/entity"
	"gopkg.in/yaml.v3"
)

// seedPath is the real, checked-in FAQ content the bot ships with. Testing
// against it (rather than an inline fixture) means every diagnosis, FAQ
// item, and escalation keyword actually in configs/faq_seed.yaml is covered,
// and new items added there are picked up automatically.
const seedPath = "../../../configs/faq_seed.yaml"

func loadSeedConfig(t *testing.T) *entity.BotConfig {
	t.Helper()
	cfg, err := LoadBotConfig(seedPath)
	if err != nil {
		t.Fatalf("LoadBotConfig(%s): %v", seedPath, err)
	}
	return cfg
}

func TestKnowledgeRepo_EveryDiagnosisFAQItemIsMatchable(t *testing.T) {
	cfg := loadSeedConfig(t)
	repo := NewKnowledgeRepo(cfg)

	for dx, diag := range cfg.Diagnoses {
		for code, faq := range diag.FAQ {
			if len(faq.MatchPhrases) == 0 {
				t.Errorf("%s/%s has no match phrases", dx, code)
				continue
			}
			for _, phrase := range faq.MatchPhrases {
				t.Run(dx+"/"+code+"/"+phrase, func(t *testing.T) {
					got, ok := repo.FindFAQ(dx, phrase)
					if !ok {
						t.Fatalf("FindFAQ(%q, %q) = not found, want a match for %s", dx, phrase, code)
					}
					if got.Answer != faq.Answer {
						t.Fatalf("FindFAQ(%q, %q) answer mismatch:\ngot  %q\nwant %q (%s)", dx, phrase, got.Answer, faq.Answer, code)
					}
				})
			}
		}
	}
}

// TestKnowledgeRepo_ExactCodeMatchIsNotShadowedByPrefixCollision guards a
// real bug found while building this suite: matchFAQ used to pick a match by
// plain substring containment with no exact-match priority and no word
// boundary. A code that is a textual prefix of another code in the same
// bucket (real data has "D3-Q1" as a prefix of "D3-Q10"/"D3-Q11"/"D3-Q12" —
// any diagnosis with 10+ items hits this) could resolve to either FAQ
// depending on Go's randomized map iteration order. matchFAQ now checks for
// an exact key/phrase match before falling back to substring search, which
// this test locks in with an isolated two-item config (not faq_seed.yaml) so
// the check doesn't depend on which codes happen to exist in the seed file
// today.
func TestKnowledgeRepo_ExactCodeMatchIsNotShadowedByPrefixCollision(t *testing.T) {
	cfg := &entity.BotConfig{
		DefaultDiagnosis: "d1",
		Diagnoses: map[string]entity.Diagnosis{
			"d1": {
				Name: "test",
				FAQ: map[string]entity.FAQ{
					"Q1":  {Answer: "answer-for-q1", MatchPhrases: []string{"Q1"}},
					"Q10": {Answer: "answer-for-q10", MatchPhrases: []string{"Q10"}},
				},
			},
		},
	}
	repo := NewKnowledgeRepo(cfg)

	const attempts = 200
	for range attempts {
		got, ok := repo.FindFAQ("d1", "Q10")
		if !ok {
			t.Fatal(`FindFAQ("d1", "Q10") = not found, want a match`)
		}
		if got.Answer != "answer-for-q10" {
			t.Fatalf(`FindFAQ("d1", "Q10") = %q, want "answer-for-q10" (must not be shadowed by "Q1", which is a prefix of "Q10")`, got.Answer)
		}
	}
}

func TestKnowledgeRepo_SharedFAQItemsReachableFromEveryDiagnosis(t *testing.T) {
	cfg := loadSeedConfig(t)
	repo := NewKnowledgeRepo(cfg)

	if len(cfg.SharedFAQ) == 0 {
		t.Fatal("no shared FAQ items loaded; expected at least one dx: shared item")
	}

	for code, faq := range cfg.SharedFAQ {
		for _, dx := range repo.DiagnosisCodes() {
			for _, phrase := range faq.MatchPhrases {
				t.Run(dx+"/shared/"+code+"/"+phrase, func(t *testing.T) {
					got, ok := repo.FindFAQ(dx, phrase)
					if !ok {
						t.Fatalf("shared FAQ %s not reachable from dx=%s via phrase %q", code, dx, phrase)
					}
					if got.Answer != faq.Answer {
						t.Fatalf("shared FAQ %s under dx=%s answer mismatch:\ngot  %q\nwant %q", code, dx, got.Answer, faq.Answer)
					}
				})
			}
		}
	}
}

func TestKnowledgeRepo_EscalationKeywordsTriggerEscalationReply(t *testing.T) {
	cfg := loadSeedConfig(t)
	repo := NewKnowledgeRepo(cfg)

	if len(cfg.Escalation.Keywords) == 0 {
		t.Fatal("no escalation keywords loaded")
	}

	for _, kw := range cfg.Escalation.Keywords {
		t.Run("bare/"+kw, func(t *testing.T) {
			msg, ok := repo.EscalationReply(kw)
			if !ok {
				t.Fatalf("EscalationReply(%q) = not triggered, want escalation", kw)
			}
			if msg != cfg.Escalation.Reply {
				t.Fatalf("EscalationReply(%q) message mismatch:\ngot  %q\nwant %q", kw, msg, cfg.Escalation.Reply)
			}
		})
		t.Run("in_sentence/"+kw, func(t *testing.T) {
			text := "หนู" + kw + "เลยค่ะตอนนี้"
			msg, ok := repo.EscalationReply(text)
			if !ok {
				t.Fatalf("EscalationReply(%q) = not triggered, want escalation", text)
			}
			if msg != cfg.Escalation.Reply {
				t.Fatalf("EscalationReply(%q) message mismatch:\ngot  %q\nwant %q", text, msg, cfg.Escalation.Reply)
			}
		})
	}
}

func TestKnowledgeRepo_BenignTextDoesNotEscalate(t *testing.T) {
	cfg := loadSeedConfig(t)
	repo := NewKnowledgeRepo(cfg)

	benign := []string{
		"ตรวจซ้ำเมื่อไหร่",
		"สวัสดีค่ะ",
		"ขอบคุณค่ะ",
		"HPV หายเองได้ไหม",
	}

	for _, text := range benign {
		for _, kw := range cfg.Escalation.Keywords {
			if strings.Contains(strings.ToLower(text), strings.ToLower(kw)) {
				t.Fatalf("test fixture %q unexpectedly contains escalation keyword %q; pick a different benign phrase", text, kw)
			}
		}
		t.Run(text, func(t *testing.T) {
			if _, ok := repo.EscalationReply(text); ok {
				t.Fatalf("EscalationReply(%q) = triggered, want no escalation", text)
			}
		})
	}
}

func TestKnowledgeRepo_ResolveDiagnosis(t *testing.T) {
	cfg := loadSeedConfig(t)
	repo := NewKnowledgeRepo(cfg)

	if len(cfg.UserDiagnosis) == 0 {
		t.Fatal("no user_diagnosis entries loaded")
	}

	for userID, wantDX := range cfg.UserDiagnosis {
		t.Run("known/"+userID, func(t *testing.T) {
			if got := repo.ResolveDiagnosis(userID); got != wantDX {
				t.Fatalf("ResolveDiagnosis(%q) = %q, want %q", userID, got, wantDX)
			}
		})
	}

	t.Run("unknown_user_falls_back_to_default", func(t *testing.T) {
		got := repo.ResolveDiagnosis("Uunknownuserwhoneverpickedadiagnosis")
		if got != cfg.DefaultDiagnosis {
			t.Fatalf("ResolveDiagnosis(unknown) = %q, want default %q", got, cfg.DefaultDiagnosis)
		}
	})

	t.Run("empty_user_id_falls_back_to_default", func(t *testing.T) {
		got := repo.ResolveDiagnosis("")
		if got != cfg.DefaultDiagnosis {
			t.Fatalf("ResolveDiagnosis(\"\") = %q, want default %q", got, cfg.DefaultDiagnosis)
		}
	})
}

func TestKnowledgeRepo_DiagnosisMetadata(t *testing.T) {
	cfg := loadSeedConfig(t)
	repo := NewKnowledgeRepo(cfg)

	codes := repo.DiagnosisCodes()
	if len(codes) != len(cfg.Diagnoses) {
		t.Fatalf("DiagnosisCodes() returned %d codes, want %d", len(codes), len(cfg.Diagnoses))
	}
	if !isSorted(codes) {
		t.Errorf("DiagnosisCodes() = %v, want sorted order", codes)
	}

	for dx, diag := range cfg.Diagnoses {
		t.Run(dx, func(t *testing.T) {
			if !repo.IsDiagnosis(dx) {
				t.Errorf("IsDiagnosis(%q) = false, want true", dx)
			}
			if got := repo.DiagnosisName(dx); got != diag.Name {
				t.Errorf("DiagnosisName(%q) = %q, want %q", dx, got, diag.Name)
			}
			if got := repo.RichMenuID(dx); got != diag.RichMenuID {
				t.Errorf("RichMenuID(%q) = %q, want %q", dx, got, diag.RichMenuID)
			}
			if diag.RichMenuID == "" {
				t.Errorf("diagnosis %q has no rich_menu_id configured", dx)
			}
		})
	}

	t.Run("unknown_diagnosis", func(t *testing.T) {
		if repo.IsDiagnosis("does-not-exist") {
			t.Error("IsDiagnosis(\"does-not-exist\") = true, want false")
		}
		if got := repo.DiagnosisName("does-not-exist"); got != "" {
			t.Errorf("DiagnosisName(\"does-not-exist\") = %q, want empty", got)
		}
		if got := repo.RichMenuID("does-not-exist"); got != "" {
			t.Errorf("RichMenuID(\"does-not-exist\") = %q, want empty", got)
		}
	})
}

func TestKnowledgeRepo_UnmatchedTextReturnsNotFound(t *testing.T) {
	cfg := loadSeedConfig(t)
	repo := NewKnowledgeRepo(cfg)

	for _, dx := range repo.DiagnosisCodes() {
		t.Run(dx, func(t *testing.T) {
			if _, ok := repo.FindFAQ(dx, "asdkjhasdkjhqwertyzxcvbn0987654321"); ok {
				t.Errorf("FindFAQ(%q, gibberish) = found, want not found", dx)
			}
		})
	}
}

func TestKnowledgeRepo_FallbackReplyNonEmpty(t *testing.T) {
	cfg := loadSeedConfig(t)
	repo := NewKnowledgeRepo(cfg)

	if got := repo.FallbackReply(); got != cfg.FallbackReply {
		t.Errorf("FallbackReply() = %q, want %q", got, cfg.FallbackReply)
	}
	if repo.FallbackReply() == "" {
		t.Error("FallbackReply() is empty")
	}
}

// TestLoadBotConfig_NoItemsDroppedByDuplicateCode parses the raw YAML
// independently of the loader and checks every item's code survived into the
// built config exactly once. A repeated `code` across two items would
// silently overwrite one of them in buildBotConfig's per-dx map.
func TestLoadBotConfig_NoItemsDroppedByDuplicateCode(t *testing.T) {
	raw, err := os.ReadFile(seedPath)
	if err != nil {
		t.Fatalf("read %s: %v", seedPath, err)
	}

	var seed struct {
		Items []struct {
			Code string `yaml:"code"`
			DX   string `yaml:"dx"`
		} `yaml:"items"`
	}
	if err := yaml.Unmarshal(raw, &seed); err != nil {
		t.Fatalf("parse %s: %v", seedPath, err)
	}
	if len(seed.Items) == 0 {
		t.Fatal("no items found in raw yaml")
	}

	wantCodesPerDX := map[string]map[string]bool{}
	for _, it := range seed.Items {
		dx := strings.TrimSpace(it.DX)
		code := strings.TrimSpace(it.Code)
		if dx == "" || code == "" {
			continue
		}
		if wantCodesPerDX[dx] == nil {
			wantCodesPerDX[dx] = map[string]bool{}
		}
		if wantCodesPerDX[dx][code] {
			t.Errorf("duplicate code %q declared twice under dx=%s in %s", code, dx, seedPath)
		}
		wantCodesPerDX[dx][code] = true
	}

	cfg := loadSeedConfig(t)
	for dx, codes := range wantCodesPerDX {
		var faqMap map[string]entity.FAQ
		if dx == sharedDX {
			faqMap = cfg.SharedFAQ
		} else if diag, ok := cfg.Diagnoses[dx]; ok {
			faqMap = diag.FAQ
		} else {
			t.Errorf("dx=%s referenced by yaml items but not present in built config", dx)
			continue
		}
		for code := range codes {
			if _, ok := faqMap[code]; !ok {
				t.Errorf("item %s/%s present in yaml but missing from built config", dx, code)
			}
		}
	}
}

func isSorted(ss []string) bool {
	for i := 1; i < len(ss); i++ {
		if ss[i-1] > ss[i] {
			return false
		}
	}
	return true
}
