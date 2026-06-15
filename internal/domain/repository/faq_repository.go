package repository

import "github.com/yimsoijoi/rama-chatbot/internal/domain/entity"

// FAQRepository provides FAQ lookup from persistent storage.
type FAQRepository interface {
	FindFAQByDiagnosisAndText(diagnosis, userText string) (entity.FAQ, bool, error)
	ListCategories(diagnosis string) ([]string, error)
	ListQuestionsByCategory(diagnosis, category string) ([]string, error)
}
