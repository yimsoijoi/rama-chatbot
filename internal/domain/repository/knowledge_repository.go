package repository

import "obgynrama-chatbot/internal/domain/entity"

type KnowledgeRepository interface {
	ResolveDiagnosis(userID string) string
	FindFAQ(diagnosis, userText string) (entity.FAQ, bool)
	EscalationReply(userText string) (string, bool)
	FallbackReply() string
	IsDiagnosis(diagnosis string) bool
	DiagnosisName(diagnosis string) string
	DiagnosisCodes() []string
	RichMenuID(diagnosis string) string
}
