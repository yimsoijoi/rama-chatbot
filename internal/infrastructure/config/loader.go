package config

import (
	"fmt"
	"os"
	"strings"
	"unicode"

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

	addQuickReplyAliases(cfg, seed.Items)
	return cfg
}

// addQuickReplyAliases makes suggested quick-reply chips answerable. A chip's
// text is a short paraphrase (e.g. "หายเองได้มากแค่ไหน?") that isn't itself a
// match phrase. For each chip we find the FAQ (in the same dx or shared) whose
// question contains that text and register the chip as one of its match
// phrases — but only when the target is unambiguous, to avoid wrong routing.
func addQuickReplyAliases(cfg *entity.BotConfig, items []seedItem) {
	type ref struct{ dx, code, question string }
	var all []ref
	for _, it := range items {
		code, dx, q := strings.TrimSpace(it.Code), strings.TrimSpace(it.DX), strings.TrimSpace(it.Question)
		if code != "" && dx != "" && q != "" {
			all = append(all, ref{dx, code, q})
		}
	}

	for _, it := range items {
		srcDX := strings.TrimSpace(it.DX)
		for _, qr := range it.QuickReplies {
			qr = strings.TrimSpace(qr)
			if qr == "" {
				continue
			}
			sqr := squash(qr)
			if sqr == "" {
				continue
			}
			// Prefer an exact question match; otherwise a unique containing
			// question. Compare on a space/punctuation-insensitive form so a
			// trailing "?" or spacing difference doesn't block the mapping.
			var exact, contains []ref
			for _, r := range all {
				if r.dx != srcDX && r.dx != sharedDX {
					continue
				}
				sq := squash(r.question)
				if sq == sqr {
					exact = append(exact, r)
				} else if strings.Contains(sq, sqr) {
					contains = append(contains, r)
				}
			}
			var target *ref
			switch {
			case len(exact) == 1:
				target = &exact[0]
			case len(exact) == 0 && len(contains) == 1:
				target = &contains[0]
			default:
				continue // none or ambiguous → leave unmapped (chip falls back gracefully)
			}
			addMatchPhrase(cfg, target.dx, target.code, qr)
		}
	}
}

func addMatchPhrase(cfg *entity.BotConfig, dx, code, phrase string) {
	get := func() (entity.FAQ, bool) {
		if dx == sharedDX {
			f, ok := cfg.SharedFAQ[code]
			return f, ok
		}
		if d, ok := cfg.Diagnoses[dx]; ok {
			f, ok2 := d.FAQ[code]
			return f, ok2
		}
		return entity.FAQ{}, false
	}
	faq, ok := get()
	if !ok {
		return
	}
	for _, p := range faq.MatchPhrases {
		if strings.EqualFold(strings.TrimSpace(p), phrase) {
			return // already present
		}
	}
	faq.MatchPhrases = append(faq.MatchPhrases, phrase)
	if dx == sharedDX {
		cfg.SharedFAQ[code] = faq
	} else {
		cfg.Diagnoses[dx].FAQ[code] = faq
	}
}

// squash lower-cases s and drops whitespace and common punctuation so that
// paraphrase chips match their source question despite "?" or spacing diffs.
func squash(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if unicode.IsSpace(r) {
			continue
		}
		switch r {
		case '?', '!', '.', ',', '(', ')', ':', ';', '"', '\'', 'ๆ': // incl. ๆ
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
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
