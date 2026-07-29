package usecase

import (
	"log/slog"
	"os"
	"regexp"
	"strings"
	"sync"

	"github.com/yimsoijoi/rama-chatbot/internal/domain/entity"
	"github.com/yimsoijoi/rama-chatbot/internal/domain/repository"
)

type ReplyUsecase struct {
	repo     repository.KnowledgeRepository
	userRepo repository.UserDiagnosisRepository
	faqRepo  repository.FAQRepository
	mu           sync.RWMutex
	cache        map[string]string
	logger       *slog.Logger
	debugEnabled bool
	debugText    bool
}

// SetLogger wires the logger and reads the debug env flags once. Reply-
// resolution logging is OFF unless DEBUG_REPLY=true (keeps prod logs clean and
// avoids any per-message logging). DEBUG_LOG_TEXT=true additionally includes the
// raw user text (may contain PII) — for short-lived debugging only.
func (u *ReplyUsecase) SetLogger(l *slog.Logger) {
	u.logger = l
	u.debugEnabled = os.Getenv("DEBUG_REPLY") == "true"
	u.debugText = os.Getenv("DEBUG_LOG_TEXT") == "true"
}

func (u *ReplyUsecase) debug(msg, userText string, args ...any) {
	if u.logger == nil || !u.debugEnabled {
		return
	}
	base := []any{slog.Int("text_len", len(userText))}
	if u.debugText {
		base = append(base, slog.String("text", userText))
	}
	u.logger.Info(msg, append(base, args...)...)
}

type ReplyResult struct {
	Message    string
	RichMenuID string // non-empty = (re)link this rich menu to the user
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
	// "กลับหน้าหลัก" re-shows the user's current diagnosis rich menu (re-links
	// the same menu; does not change or clear their diagnosis). If they have not
	// chosen a diagnosis yet, ask them to pick one first (they already see the
	// default selector menu).
	if isHomeCommand(userText) {
		dx, ok := u.explicitDiagnosis(userID)
		if !ok {
			return ReplyResult{Message: "กรุณาเลือกกลุ่มผลตรวจของคุณก่อนนะคะ เลือกได้จากเมนูด้านล่าง หรือพิมพ์: เลือก DX1, DX2, DX3, DX4 หรือ DX5"}
		}
		return ReplyResult{
			Message:    "เลือกคำถามที่ต้องการจากเมนูด้านล่างได้เลยค่ะ",
			RichMenuID: u.repo.RichMenuID(dx),
		}
	}

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
	u.debug("reply_resolved", userText, slog.String("dx", dx), slog.Bool("faq_repo_active", u.faqRepo != nil))
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
	u.debug("reply_config_lookup", userText, slog.String("dx", dx), slog.Bool("matched", ok))
	if !ok {
		return ReplyResult{Message: u.repo.FallbackReply()}
	}

	return buildReplyFromFAQ(faq)
}

var (
	reBR      = regexp.MustCompile(`(?i)<br\s*/?>`)
	reHTMLTag = regexp.MustCompile(`<[^>]+>`)
	reBlank   = regexp.MustCompile(`\n{3,}`)
)

// stripHTML turns the HTML-flavored answer text into plain text for LINE:
// <br> becomes a newline, all other tags are removed.
func stripHTML(s string) string {
	s = reBR.ReplaceAllString(s, "\n")
	s = reHTMLTag.ReplaceAllString(s, "")
	s = reBlank.ReplaceAllString(s, "\n\n")
	return strings.TrimSpace(s)
}

func buildReplyFromFAQ(faq entity.FAQ) ReplyResult {
	var builder strings.Builder
	builder.WriteString(stripHTML(faq.Answer))
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

// explicitDiagnosis returns the diagnosis the user has actually chosen (from
// the in-memory cache or the DB), NOT the default fallback. ok is false when
// the user has not selected a diagnosis yet.
func (u *ReplyUsecase) explicitDiagnosis(userID string) (string, bool) {
	if dx, ok := u.readCachedDiagnosis(userID); ok && u.repo.IsDiagnosis(dx) {
		return dx, true
	}
	if userID != "" && u.userRepo != nil {
		dx, ok, err := u.userRepo.GetDiagnosisByLineUserID(userID)
		if err == nil && ok && u.repo.IsDiagnosis(dx) {
			u.rememberDiagnosis(userID, dx)
			return dx, true
		}
	}
	return "", false
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

func isHomeCommand(text string) bool {
	t := strings.ToLower(strings.TrimSpace(text))
	return t == "กลับหน้าหลัก" || t == "หน้าหลัก"
}

func isSubmenuCommand(text string) bool {
	normalized := strings.ToLower(strings.TrimSpace(text))
	return normalized == "เมนูย่อย" || normalized == "หมวดคำถาม" || normalized == "ดูหมวดคำถาม"
}
