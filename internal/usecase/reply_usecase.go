package usecase

import (
	"strings"
	"sync"

	"github.com/yimsoijoi/rama-chatbot/internal/domain/entity"
	"github.com/yimsoijoi/rama-chatbot/internal/domain/repository"
)

type ReplyUsecase struct {
	repo     repository.KnowledgeRepository
	userRepo repository.UserDiagnosisRepository
	faqRepo  repository.FAQRepository
	mu       sync.RWMutex
	cache    map[string]string
}

type ReplyResult struct {
	Message    string
	RichMenuID string // non-empty = link this rich menu to the user
	QuickReply []string
}

func NewReplyUsecase(repo repository.KnowledgeRepository) *ReplyUsecase {
	return NewReplyUsecaseWithRepos(repo, nil, nil)
}

func NewReplyUsecaseWithUserRepo(repo repository.KnowledgeRepository, userRepo repository.UserDiagnosisRepository) *ReplyUsecase {
	return NewReplyUsecaseWithRepos(repo, userRepo, nil)
}

func NewReplyUsecaseWithRepos(repo repository.KnowledgeRepository, userRepo repository.UserDiagnosisRepository, faqRepo repository.FAQRepository) *ReplyUsecase {
	return &ReplyUsecase{repo: repo, userRepo: userRepo, faqRepo: faqRepo, cache: make(map[string]string)}
}

func (u *ReplyUsecase) BuildReply(userID, userText string) ReplyResult {
	if dx, ok := parseDiagnosisSelection(userText); ok {
		if !u.repo.IsDiagnosis(dx) {
			return ReplyResult{Message: "ไม่พบกลุ่มผลตรวจที่เลือกค่ะ กรุณาเลือกใหม่"}
		}
		if userID == "" {
			return ReplyResult{Message: "ยังไม่พบ LINE user ID สำหรับบันทึกผลตรวจ กรุณาลองใหม่อีกครั้งค่ะ"}
		}
		if u.userRepo == nil {
			return ReplyResult{Message: "ระบบยังไม่พร้อมบันทึกผลตรวจ กรุณาลองใหม่อีกครั้งค่ะ"}
		}
		if err := u.userRepo.SetDiagnosisByLineUserID(userID, dx); err != nil {
			return ReplyResult{Message: "บันทึกผลตรวจไม่สำเร็จ กรุณาลองใหม่อีกครั้งค่ะ"}
		}
		u.rememberDiagnosis(userID, dx)

		name := u.repo.DiagnosisName(dx)
		if name == "" {
			name = dx
		}
		return ReplyResult{
			Message:    "บันทึกกลุ่มผลตรวจเรียบร้อยค่ะ\nDX: " + dx + " (" + name + ")\n\nพิมพ์คำถามที่ต้องการได้เลย เช่น ตรวจซ้ำเมื่อไหร่",
			RichMenuID: u.repo.RichMenuID(dx),
		}
	}

	if msg, ok := u.repo.EscalationReply(userText); ok {
		return ReplyResult{Message: msg}
	}

	dx := u.resolveDiagnosis(userID)
	if u.faqRepo != nil {
		if isSubmenuCommand(userText) {
			cats, err := u.faqRepo.ListCategories(dx)
			if err == nil && len(cats) > 0 {
				return ReplyResult{
					Message:    "เลือกหมวดคำถามที่ต้องการได้เลยค่ะ",
					QuickReply: cats,
				}
			}
		}

		qs, err := u.faqRepo.ListQuestionsByCategory(dx, userText)
		if err == nil && len(qs) > 0 {
			return ReplyResult{
				Message:    "เลือกคำถามย่อยในหมวดนี้ได้เลยค่ะ",
				QuickReply: qs,
			}
		}
	}

	if u.faqRepo != nil {
		faq, ok, err := u.faqRepo.FindFAQByDiagnosisAndText(dx, userText)
		if err == nil && ok {
			return buildReplyFromFAQ(faq)
		}
	}

	faq, ok := u.repo.FindFAQ(dx, userText)
	if !ok {
		return ReplyResult{Message: u.repo.FallbackReply() + "\n\nหากต้องการเลือกกลุ่มผลตรวจใหม่ พิมพ์: เลือก DX1, DX2, DX3, DX4 หรือ DX5"}
	}

	return buildReplyFromFAQ(faq)
}

func buildReplyFromFAQ(faq entity.FAQ) ReplyResult {
	var builder strings.Builder
	builder.WriteString(faq.Answer)
	if len(faq.QuickReply) > 0 {
		builder.WriteString("\n\nคำถามต่อไปที่อาจสนใจ:\n")
		for i, q := range faq.QuickReply {
			builder.WriteString("- ")
			builder.WriteString(q)
			if i < len(faq.QuickReply)-1 {
				builder.WriteString("\n")
			}
		}
	}

	return ReplyResult{Message: builder.String(), QuickReply: faq.QuickReply}
}

func (u *ReplyUsecase) resolveDiagnosis(userID string) string {
	if dx, ok := u.readCachedDiagnosis(userID); ok && u.repo.IsDiagnosis(dx) {
		return dx
	}

	if userID != "" && u.userRepo != nil {
		dx, ok, err := u.userRepo.GetDiagnosisByLineUserID(userID)
		if err == nil && ok && u.repo.IsDiagnosis(dx) {
			u.rememberDiagnosis(userID, dx)
			return dx
		}
	}

	fallbackDX := u.repo.ResolveDiagnosis(userID)
	if userID != "" && u.repo.IsDiagnosis(fallbackDX) {
		u.rememberDiagnosis(userID, fallbackDX)
	}
	return fallbackDX
}

func (u *ReplyUsecase) readCachedDiagnosis(userID string) (string, bool) {
	if userID == "" {
		return "", false
	}
	u.mu.RLock()
	dx, ok := u.cache[userID]
	u.mu.RUnlock()
	return dx, ok
}

func (u *ReplyUsecase) rememberDiagnosis(userID, dx string) {
	if userID == "" || dx == "" {
		return
	}
	u.mu.Lock()
	u.cache[userID] = dx
	u.mu.Unlock()
}

func parseDiagnosisSelection(text string) (string, bool) {
	normalized := strings.ToLower(strings.TrimSpace(text))
	replacer := strings.NewReplacer(" ", "", "-", "", "_", "")
	compact := replacer.Replace(normalized)

	switch compact {
	case "dx1", "เลือกdx1", "เลือกd1", "d1":
		return "d1", true
	case "dx2", "เลือกdx2", "เลือกd2", "d2":
		return "d2", true
	case "dx3", "เลือกdx3", "เลือกd3", "d3":
		return "d3", true
	case "dx4", "เลือกdx4", "เลือกd4", "d4":
		return "d4", true
	case "dx5", "เลือกdx5", "เลือกd5", "d5":
		return "d5", true
	default:
		return "", false
	}
}

func isSubmenuCommand(text string) bool {
	normalized := strings.ToLower(strings.TrimSpace(text))
	return normalized == "เมนูย่อย" || normalized == "หมวดคำถาม" || normalized == "ดูหมวดคำถาม"
}
