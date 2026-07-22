package sqlite

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// seedYAMLPath is the real, checked-in FAQ content the bot ships with.
// Seeding a temp DB from it (mirroring scripts/cmd/seed-faq-from-config)
// means every diagnosis, FAQ item, and category in configs/faq_seed.yaml is
// exercised against the actual DB-backed matcher used in production.
const seedYAMLPath = "../../../configs/faq_seed.yaml"

type seedItem struct {
	Code         string   `yaml:"code"`
	DX           string   `yaml:"dx"`
	Category     string   `yaml:"category"`
	Question     string   `yaml:"question"`
	Answer       string   `yaml:"answer"`
	QuickReplies []string `yaml:"quick_replies"`
	MatchPhrases []string `yaml:"match_phrases"`
}

type seedFile struct {
	Items []seedItem `yaml:"items"`
}

func loadSeedItems(t *testing.T) []seedItem {
	t.Helper()
	raw, err := os.ReadFile(seedYAMLPath)
	if err != nil {
		t.Fatalf("read %s: %v", seedYAMLPath, err)
	}
	var seed seedFile
	if err := yaml.Unmarshal(raw, &seed); err != nil {
		t.Fatalf("parse %s: %v", seedYAMLPath, err)
	}
	if len(seed.Items) == 0 {
		t.Fatalf("no items loaded from %s", seedYAMLPath)
	}
	return seed.Items
}

// newSeededRepo opens a temp sqlite DB and loads it exactly the way
// scripts/cmd/seed-faq-from-config does, so the DB-backed matcher under test
// is exercised against real faq_seed.yaml content, not a hand-picked fixture.
func newSeededRepo(t *testing.T, items []seedItem) *UserDiagnosisRepo {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "test.db")
	repo, err := New(dbPath)
	if err != nil {
		t.Fatalf("sqlite.New: %v", err)
	}
	t.Cleanup(func() { _ = repo.Close() })

	for _, it := range items {
		code := strings.TrimSpace(it.Code)
		dx := strings.TrimSpace(it.DX)
		question := strings.TrimSpace(it.Question)
		answer := strings.TrimSpace(it.Answer)
		category := strings.TrimSpace(it.Category)
		if code == "" || dx == "" || question == "" || answer == "" {
			continue
		}

		if _, err := repo.db.Exec(
			`INSERT INTO faq_reply (diagnosis, faq_key, question, category, answer) VALUES (?, ?, ?, ?, ?)`,
			dx, code, question, category, answer,
		); err != nil {
			t.Fatalf("insert faq_reply %s/%s: %v", dx, code, err)
		}

		for _, p := range dedupeNonEmpty(append([]string{code, question}, it.MatchPhrases...)) {
			if _, err := repo.db.Exec(
				`INSERT INTO faq_match_phrase (diagnosis, faq_key, phrase) VALUES (?, ?, ?)`,
				dx, code, p,
			); err != nil {
				t.Fatalf("insert faq_match_phrase %s/%s/%s: %v", dx, code, p, err)
			}
		}

		for _, q := range dedupeNonEmpty(it.QuickReplies) {
			if _, err := repo.db.Exec(
				`INSERT INTO faq_quick_reply (diagnosis, faq_key, quick_reply) VALUES (?, ?, ?)`,
				dx, code, q,
			); err != nil {
				t.Fatalf("insert faq_quick_reply %s/%s/%s: %v", dx, code, q, err)
			}
		}
	}

	return repo
}

func dedupeNonEmpty(in []string) []string {
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

func TestFindFAQByDiagnosisAndText_AllSeedItemsAreMatchable(t *testing.T) {
	items := loadSeedItems(t)
	repo := newSeededRepo(t, items)

	for _, it := range items {
		code := strings.TrimSpace(it.Code)
		dx := strings.TrimSpace(it.DX)
		question := strings.TrimSpace(it.Question)
		answer := strings.TrimSpace(it.Answer)
		if code == "" || dx == "" || question == "" || answer == "" {
			continue
		}

		phrases := dedupeNonEmpty(append([]string{code, question}, it.MatchPhrases...))

		for _, phrase := range phrases {
			t.Run(dx+"/"+code+"/"+phrase, func(t *testing.T) {
				faq, ok, err := repo.FindFAQByDiagnosisAndText(dx, phrase)
				if err != nil {
					t.Fatalf("FindFAQByDiagnosisAndText(%q, %q): %v", dx, phrase, err)
				}
				if !ok {
					t.Fatalf("FindFAQByDiagnosisAndText(%q, %q) = not found, want a match for %s", dx, phrase, code)
				}
				if faq.Answer != answer {
					t.Fatalf("FindFAQByDiagnosisAndText(%q, %q) answer mismatch:\ngot  %q\nwant %q (%s)", dx, phrase, faq.Answer, answer, code)
				}
			})
		}
	}
}

func TestFindFAQByDiagnosisAndText_SharedItemsReachableFromEveryDiagnosis(t *testing.T) {
	items := loadSeedItems(t)
	repo := newSeededRepo(t, items)

	diagnoses := []string{"d1", "d2", "d3", "d4", "d5"}
	hasSharedItems := false
	for _, it := range items {
		if strings.TrimSpace(it.DX) == "shared" {
			hasSharedItems = true
			break
		}
	}
	if !hasSharedItems {
		t.Fatal("no shared items found in seed data; expected at least one dx: shared item")
	}

	for _, it := range items {
		if strings.TrimSpace(it.DX) != "shared" {
			continue
		}
		code := strings.TrimSpace(it.Code)
		question := strings.TrimSpace(it.Question)
		answer := strings.TrimSpace(it.Answer)
		if code == "" || question == "" || answer == "" {
			continue
		}

		phrases := dedupeNonEmpty(append([]string{code, question}, it.MatchPhrases...))

		for _, dx := range diagnoses {
			for _, phrase := range phrases {
				t.Run(dx+"/shared/"+code+"/"+phrase, func(t *testing.T) {
					faq, ok, err := repo.FindFAQByDiagnosisAndText(dx, phrase)
					if err != nil {
						t.Fatalf("FindFAQByDiagnosisAndText(%q, %q): %v", dx, phrase, err)
					}
					if !ok {
						t.Fatalf("shared FAQ %s not reachable from dx=%s via phrase %q", code, dx, phrase)
					}
					if faq.Answer != answer {
						t.Fatalf("shared FAQ %s under dx=%s answer mismatch:\ngot  %q\nwant %q", code, dx, faq.Answer, answer)
					}
				})
			}
		}
	}
}

func TestFindFAQByDiagnosisAndText_NoMatchReturnsFalse(t *testing.T) {
	items := loadSeedItems(t)
	repo := newSeededRepo(t, items)

	faq, ok, err := repo.FindFAQByDiagnosisAndText("d1", "qwertyuiopasdfghjkl1234567890")
	if err != nil {
		t.Fatalf("FindFAQByDiagnosisAndText: %v", err)
	}
	if ok {
		t.Fatalf("expected no match, got %+v", faq)
	}
}

// TestFindFAQByDiagnosisAndText_ExactCodeMatchIsNotShadowedByPrefixCollision
// guards a real bug found while building this suite: matchFAQRow used to
// pick a match by plain substring containment with no exact-match priority
// and no word boundary. A code that is a textual prefix of another code in
// the same diagnosis bucket (real data: "D3-Q1" is a prefix of "D3-Q10",
// "D3-Q11", "D3-Q12" — every diagnosis with 10+ items hits this) could
// resolve to either FAQ depending on the order SQLite returned rows in.
// matchFAQRow (via a two-pass exact-then-substring scan in
// FindFAQByDiagnosisAndText) now checks for an exact match before falling
// back to substring search, which this test locks in with an isolated
// two-item DB (not faq_seed.yaml) so the check doesn't depend on what codes
// happen to exist in the seed file today.
func TestFindFAQByDiagnosisAndText_ExactCodeMatchIsNotShadowedByPrefixCollision(t *testing.T) {
	items := []seedItem{
		{Code: "Q1", DX: "d1", Question: "คำถามที่หนึ่ง", Answer: "answer-for-q1"},
		{Code: "Q10", DX: "d1", Question: "คำถามที่สิบ", Answer: "answer-for-q10"},
	}
	repo := newSeededRepo(t, items)

	const attempts = 50
	for range attempts {
		faq, ok, err := repo.FindFAQByDiagnosisAndText("d1", "Q10")
		if err != nil {
			t.Fatalf("FindFAQByDiagnosisAndText: %v", err)
		}
		if !ok {
			t.Fatal(`FindFAQByDiagnosisAndText("d1", "Q10") = not found, want a match`)
		}
		if faq.Answer != "answer-for-q10" {
			t.Fatalf(`FindFAQByDiagnosisAndText("d1", "Q10") = %q, want "answer-for-q10" (must not be shadowed by "Q1", which is a prefix of "Q10")`, faq.Answer)
		}
	}
}

func TestListCategoriesAndQuestions_MatchSeedData(t *testing.T) {
	items := loadSeedItems(t)
	repo := newSeededRepo(t, items)

	wantCategories := map[string]map[string]bool{}
	wantQuestions := map[string]map[string]map[string]bool{}

	for _, it := range items {
		dx := strings.TrimSpace(it.DX)
		cat := strings.TrimSpace(it.Category)
		q := strings.TrimSpace(it.Question)
		if dx == "" || dx == "shared" || cat == "" || q == "" {
			continue
		}
		if wantCategories[dx] == nil {
			wantCategories[dx] = map[string]bool{}
		}
		wantCategories[dx][cat] = true

		if wantQuestions[dx] == nil {
			wantQuestions[dx] = map[string]map[string]bool{}
		}
		if wantQuestions[dx][cat] == nil {
			wantQuestions[dx][cat] = map[string]bool{}
		}
		wantQuestions[dx][cat][q] = true
	}

	if len(wantCategories) == 0 {
		t.Fatal("no per-diagnosis categories found in seed data")
	}

	for dx, cats := range wantCategories {
		t.Run(dx, func(t *testing.T) {
			got, err := repo.ListCategories(dx)
			if err != nil {
				t.Fatalf("ListCategories(%q): %v", dx, err)
			}
			gotSet := map[string]bool{}
			for _, c := range got {
				gotSet[c] = true
			}
			for c := range cats {
				if !gotSet[c] {
					t.Errorf("ListCategories(%q) missing category %q, got %v", dx, c, got)
				}
			}

			for cat, questions := range wantQuestions[dx] {
				gotQs, err := repo.ListQuestionsByCategory(dx, cat)
				if err != nil {
					t.Fatalf("ListQuestionsByCategory(%q, %q): %v", dx, cat, err)
				}
				gotQSet := map[string]bool{}
				for _, q := range gotQs {
					gotQSet[q] = true
				}
				for q := range questions {
					if !gotQSet[q] {
						t.Errorf("ListQuestionsByCategory(%q, %q) missing question %q, got %v", dx, cat, q, gotQs)
					}
				}
			}
		})
	}
}

func TestUserDiagnosisRoundTrip(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	repo, err := New(dbPath)
	if err != nil {
		t.Fatalf("sqlite.New: %v", err)
	}
	defer repo.Close()

	const userID = "Utestuser0000000000000000000000000"

	if _, ok, err := repo.GetDiagnosisByLineUserID(userID); err != nil || ok {
		t.Fatalf("expected not found before set, got ok=%v err=%v", ok, err)
	}

	if err := repo.SetDiagnosisByLineUserID(userID, "d3"); err != nil {
		t.Fatalf("SetDiagnosisByLineUserID: %v", err)
	}

	dx, ok, err := repo.GetDiagnosisByLineUserID(userID)
	if err != nil || !ok || dx != "d3" {
		t.Fatalf("GetDiagnosisByLineUserID = (%q, %v, %v), want (d3, true, nil)", dx, ok, err)
	}

	// Upsert should overwrite, not duplicate.
	if err := repo.SetDiagnosisByLineUserID(userID, "d5"); err != nil {
		t.Fatalf("SetDiagnosisByLineUserID (update): %v", err)
	}
	dx, ok, err = repo.GetDiagnosisByLineUserID(userID)
	if err != nil || !ok || dx != "d5" {
		t.Fatalf("GetDiagnosisByLineUserID after update = (%q, %v, %v), want (d5, true, nil)", dx, ok, err)
	}
}
