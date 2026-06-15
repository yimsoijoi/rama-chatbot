package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/yimsoijoi/rama-chatbot/internal/domain/entity"
	"github.com/yimsoijoi/rama-chatbot/internal/observability"

	"gopkg.in/yaml.v3"
)

// sharedDX is the pseudo-diagnosis used by FAQ items that apply to every
// diagnosis. Items with this dx are loaded into BotConfig.SharedFAQ.
const sharedDX = "shared"

// seedFile is the on-disk shape of configs/faq_seed.yaml: a flat list of FAQ
// items plus the operational config the bot needs at runtime.
type seedFile struct {
	DefaultDiagnosis string                       `yaml:"default_diagnosis"`
	UserDiagnosis    map[string]string            `yaml:"user_diagnosis"`
	Diagnoses        map[string]seedDiagnosisMeta `yaml:"diagnoses"`
	Escalation       entity.Escalation            `yaml:"escalation"`
	FallbackReply    string                       `yaml:"fallback_reply"`
	Items            []seedItem                   `yaml:"items"`
}

type seedDiagnosisMeta struct {
	Name       string `yaml:"name"`
	RichMenuID string `yaml:"rich_menu_id"`
}

type seedItem struct {
	Code         string   `yaml:"code"`
	DX           string   `yaml:"dx"`
	Category     string   `yaml:"category"`
	Question     string   `yaml:"question"`
	Answer       string   `yaml:"answer"`
	QuickReplies []string `yaml:"quick_replies"`
	MatchPhrases []string `yaml:"match_phrases"`
}

// LoadBotConfig reads the seed config (configs/faq_seed.yaml) and builds the
// runtime BotConfig. FAQ items are grouped by their dx; items with dx "shared"
// become shared FAQ available to every diagnosis.
func LoadBotConfig(path string) (*entity.BotConfig, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, observability.NewAppError(
			"CFG_READ_FAILED",
			"config.LoadBotConfig.read",
			fmt.Sprintf("failed to read bot config from path=%s", path),
			err,
		)
	}

	var seed seedFile
	if err := yaml.Unmarshal(b, &seed); err != nil {
		return nil, observability.NewAppError(
			"CFG_PARSE_FAILED",
			"config.LoadBotConfig.unmarshal",
			"failed to parse bot YAML config",
			err,
		)
	}

	cfg := buildBotConfig(seed)

	if len(cfg.Diagnoses) == 0 {
		return nil, observability.NewAppError(
			"CFG_INVALID",
			"config.LoadBotConfig.validate",
			"config must contain at least one diagnosis",
			nil,
		)
	}

	if _, ok := cfg.Diagnoses[cfg.DefaultDiagnosis]; !ok {
		return nil, observability.NewAppError(
			"CFG_INVALID",
			"config.LoadBotConfig.validate",
			fmt.Sprintf("default diagnosis %q not found in diagnoses", cfg.DefaultDiagnosis),
			nil,
		)
	}

	if cfg.FallbackReply == "" {
		cfg.FallbackReply = "ขออภัยค่ะ ตอนนี้บอตยังไม่เข้าใจคำถามนี้ กรุณากดเมนูหรือพิมพ์ใหม่อีกครั้ง"
	}

	return cfg, nil
}

// buildBotConfig converts the flat seed shape into the nested BotConfig the
// rest of the application consumes.
func buildBotConfig(seed seedFile) *entity.BotConfig {
	cfg := &entity.BotConfig{
		DefaultDiagnosis: seed.DefaultDiagnosis,
		UserDiagnosis:    seed.UserDiagnosis,
		Diagnoses:        make(map[string]entity.Diagnosis, len(seed.Diagnoses)),
		SharedFAQ:        make(map[string]entity.FAQ),
		Escalation:       seed.Escalation,
		FallbackReply:    strings.TrimSpace(seed.FallbackReply),
	}

	// Seed declared diagnoses (name + rich_menu_id) with empty FAQ maps.
	for code, meta := range seed.Diagnoses {
		cfg.Diagnoses[code] = entity.Diagnosis{
			Name:       meta.Name,
			RichMenuID: meta.RichMenuID,
			FAQ:        make(map[string]entity.FAQ),
		}
	}

	for _, it := range seed.Items {
		code := strings.TrimSpace(it.Code)
		dx := strings.TrimSpace(it.DX)
		question := strings.TrimSpace(it.Question)
		answer := strings.TrimSpace(it.Answer)
		if code == "" || dx == "" || question == "" || answer == "" {
			continue
		}

		faq := entity.FAQ{
			Answer:       answer,
			QuickReply:   trimUnique(it.QuickReplies),
			MatchPhrases: trimUnique(append([]string{code, question}, it.MatchPhrases...)),
		}

		if dx == sharedDX {
			cfg.SharedFAQ[code] = faq
			continue
		}

		diag, ok := cfg.Diagnoses[dx]
		if !ok {
			// FAQ references a dx not declared in the diagnoses block; create
			// a minimal entry so the answer is still reachable.
			diag = entity.Diagnosis{Name: dx, FAQ: make(map[string]entity.FAQ)}
		}
		if diag.FAQ == nil {
			diag.FAQ = make(map[string]entity.FAQ)
		}
		diag.FAQ[code] = faq
		cfg.Diagnoses[dx] = diag
	}

	return cfg
}

func trimUnique(in []string) []string {
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, v := range in {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}
