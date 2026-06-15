package config

import (
	"sort"
	"strings"

	"github.com/yimsoijoi/rama-chatbot/internal/domain/entity"
)

type KnowledgeRepo struct {
	cfg *entity.BotConfig
}

func NewKnowledgeRepo(cfg *entity.BotConfig) *KnowledgeRepo {
	return &KnowledgeRepo{cfg: cfg}
}

func (r *KnowledgeRepo) ResolveDiagnosis(userID string) string {
	if dx, ok := r.cfg.UserDiagnosis[userID]; ok {
		if _, exists := r.cfg.Diagnoses[dx]; exists {
			return dx
		}
	}
	return r.cfg.DefaultDiagnosis
}

func (r *KnowledgeRepo) FindFAQ(diagnosis, userText string) (entity.FAQ, bool) {
	text := normalize(userText)

	if dx, ok := r.cfg.Diagnoses[diagnosis]; ok {
		if faq, found := matchFAQ(dx.FAQ, text); found {
			return faq, true
		}
	}

	if faq, found := matchFAQ(r.cfg.SharedFAQ, text); found {
		return faq, true
	}

	return entity.FAQ{}, false
}

func (r *KnowledgeRepo) EscalationReply(userText string) (string, bool) {
	text := normalize(userText)
	for _, kw := range r.cfg.Escalation.Keywords {
		if strings.Contains(text, normalize(kw)) {
			return r.cfg.Escalation.Reply, true
		}
	}
	return "", false
}

func (r *KnowledgeRepo) FallbackReply() string {
	return r.cfg.FallbackReply
}

func (r *KnowledgeRepo) IsDiagnosis(diagnosis string) bool {
	_, ok := r.cfg.Diagnoses[diagnosis]
	return ok
}

func (r *KnowledgeRepo) DiagnosisName(diagnosis string) string {
	dx, ok := r.cfg.Diagnoses[diagnosis]
	if !ok {
		return ""
	}
	return dx.Name
}

func (r *KnowledgeRepo) RichMenuID(diagnosis string) string {
	if dx, ok := r.cfg.Diagnoses[diagnosis]; ok {
		return dx.RichMenuID
	}
	return ""
}

func (r *KnowledgeRepo) DiagnosisCodes() []string {
	codes := make([]string, 0, len(r.cfg.Diagnoses))
	for code := range r.cfg.Diagnoses {
		codes = append(codes, code)
	}
	sort.Strings(codes)
	return codes
}

func matchFAQ(items map[string]entity.FAQ, text string) (entity.FAQ, bool) {
	for key, faq := range items {
		if strings.Contains(text, normalize(key)) {
			return faq, true
		}
		for _, phrase := range faq.MatchPhrases {
			if strings.Contains(text, normalize(phrase)) {
				return faq, true
			}
		}
	}
	return entity.FAQ{}, false
}

func normalize(s string) string {
	return strings.TrimSpace(strings.ToLower(s))
}
