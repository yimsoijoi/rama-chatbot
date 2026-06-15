package sqlite

import (
	"database/sql"
	"fmt"
	"strings"

	"obgynrama-chatbot/internal/domain/entity"

	_ "modernc.org/sqlite"
)

// UserDiagnosisRepo stores LINE user → diagnosis mapping in SQLite.
type UserDiagnosisRepo struct {
	db *sql.DB
}

// New opens (or creates) a SQLite database at dbPath and runs schema migration.
func New(dbPath string) (*UserDiagnosisRepo, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("sqlite.New: open %q: %w", dbPath, err)
	}
	db.SetMaxOpenConns(1) // SQLite is single-writer
	if err := migrate(db); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("sqlite.New: migrate: %w", err)
	}
	return &UserDiagnosisRepo{db: db}, nil
}

// Close closes the underlying database connection.
func (r *UserDiagnosisRepo) Close() error {
	return r.db.Close()
}

func migrate(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS user_diagnosis (
			line_user_id TEXT PRIMARY KEY,
			diagnosis    TEXT NOT NULL,
			updated_at   DATETIME DEFAULT (datetime('now'))
		);

		CREATE TABLE IF NOT EXISTS faq_reply (
			diagnosis TEXT NOT NULL,
			faq_key   TEXT NOT NULL,
			question  TEXT NOT NULL DEFAULT '',
			category  TEXT NOT NULL DEFAULT '',
			answer    TEXT NOT NULL,
			PRIMARY KEY (diagnosis, faq_key)
		);

		CREATE TABLE IF NOT EXISTS faq_match_phrase (
			diagnosis TEXT NOT NULL,
			faq_key   TEXT NOT NULL,
			phrase    TEXT NOT NULL,
			PRIMARY KEY (diagnosis, faq_key, phrase)
		);

		CREATE TABLE IF NOT EXISTS faq_quick_reply (
			diagnosis   TEXT NOT NULL,
			faq_key     TEXT NOT NULL,
			quick_reply TEXT NOT NULL,
			PRIMARY KEY (diagnosis, faq_key, quick_reply)
		);
	`)
	if err != nil {
		return err
	}

	_, _ = db.Exec(`ALTER TABLE faq_reply ADD COLUMN question TEXT NOT NULL DEFAULT ''`)
	_, _ = db.Exec(`ALTER TABLE faq_reply ADD COLUMN category TEXT NOT NULL DEFAULT ''`)
	return nil
}

// GetDiagnosisByLineUserID returns the stored diagnosis for a LINE user.
func (r *UserDiagnosisRepo) GetDiagnosisByLineUserID(lineUserID string) (string, bool, error) {
	var dx string
	err := r.db.QueryRow(
		`SELECT diagnosis FROM user_diagnosis WHERE line_user_id = ?`, lineUserID,
	).Scan(&dx)
	if err == sql.ErrNoRows {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("GetDiagnosisByLineUserID: %w", err)
	}
	return dx, true, nil
}

// SetDiagnosisByLineUserID upserts the diagnosis for a LINE user.
func (r *UserDiagnosisRepo) SetDiagnosisByLineUserID(lineUserID, diagnosis string) error {
	_, err := r.db.Exec(`
		INSERT INTO user_diagnosis (line_user_id, diagnosis, updated_at)
		VALUES (?, ?, datetime('now'))
		ON CONFLICT(line_user_id) DO UPDATE SET
			diagnosis  = excluded.diagnosis,
			updated_at = excluded.updated_at
	`, lineUserID, diagnosis)
	if err != nil {
		return fmt.Errorf("SetDiagnosisByLineUserID: %w", err)
	}
	return nil
}

// FindFAQByDiagnosisAndText looks up reply templates from DB first.
// It checks both diagnosis-specific and shared entries.
func (r *UserDiagnosisRepo) FindFAQByDiagnosisAndText(diagnosis, userText string) (entity.FAQ, bool, error) {
	rows, err := r.db.Query(`
		SELECT fr.diagnosis, fr.faq_key, fr.question, fr.answer,
			COALESCE((
				SELECT group_concat(quick_reply, char(10))
				FROM faq_quick_reply qr
				WHERE qr.diagnosis = fr.diagnosis AND qr.faq_key = fr.faq_key
			), '') AS quick_replies,
			COALESCE((
				SELECT group_concat(phrase, char(10))
				FROM faq_match_phrase mp
				WHERE mp.diagnosis = fr.diagnosis AND mp.faq_key = fr.faq_key
			), '') AS match_phrases
		FROM faq_reply fr
		WHERE fr.diagnosis = ? OR fr.diagnosis = 'shared'
	`, diagnosis)
	if err != nil {
		return entity.FAQ{}, false, fmt.Errorf("FindFAQByDiagnosisAndText query: %w", err)
	}
	defer rows.Close()

	needle := normalize(userText)
	for rows.Next() {
		var rowDiagnosis, faqKey, question, answer, quickReplies, matchPhrases string
		if err := rows.Scan(&rowDiagnosis, &faqKey, &question, &answer, &quickReplies, &matchPhrases); err != nil {
			return entity.FAQ{}, false, fmt.Errorf("FindFAQByDiagnosisAndText scan: %w", err)
		}

		if matchFAQRow(faqKey, question, matchPhrases, needle) {
			return entity.FAQ{
				Answer:       answer,
				QuickReply:   splitLines(quickReplies),
				MatchPhrases: splitLines(matchPhrases),
			}, true, nil
		}
	}
	if err := rows.Err(); err != nil {
		return entity.FAQ{}, false, fmt.Errorf("FindFAQByDiagnosisAndText rows: %w", err)
	}

	return entity.FAQ{}, false, nil
}

func (r *UserDiagnosisRepo) ListCategories(diagnosis string) ([]string, error) {
	rows, err := r.db.Query(`
		SELECT DISTINCT category
		FROM faq_reply
		WHERE diagnosis = ? AND category <> ''
		ORDER BY category
	`, diagnosis)
	if err != nil {
		return nil, fmt.Errorf("ListCategories query: %w", err)
	}
	defer rows.Close()

	var out []string
	for rows.Next() {
		var c string
		if err := rows.Scan(&c); err != nil {
			return nil, fmt.Errorf("ListCategories scan: %w", err)
		}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("ListCategories rows: %w", err)
	}
	return out, nil
}

func (r *UserDiagnosisRepo) ListQuestionsByCategory(diagnosis, category string) ([]string, error) {
	rows, err := r.db.Query(`
		SELECT question
		FROM faq_reply
		WHERE diagnosis = ? AND category = ?
		ORDER BY faq_key
	`, diagnosis, strings.TrimSpace(category))
	if err != nil {
		return nil, fmt.Errorf("ListQuestionsByCategory query: %w", err)
	}
	defer rows.Close()

	var out []string
	for rows.Next() {
		var q string
		if err := rows.Scan(&q); err != nil {
			return nil, fmt.Errorf("ListQuestionsByCategory scan: %w", err)
		}
		q = strings.TrimSpace(q)
		if q != "" {
			out = append(out, q)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("ListQuestionsByCategory rows: %w", err)
	}
	return out, nil
}

func matchFAQRow(faqKey, question, phrases, needle string) bool {
	if strings.Contains(needle, normalize(faqKey)) {
		return true
	}
	if strings.Contains(needle, normalize(question)) {
		return true
	}
	for _, p := range splitLines(phrases) {
		if strings.Contains(needle, normalize(p)) {
			return true
		}
	}
	return false
}

func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, "\n")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func normalize(s string) string {
	return strings.TrimSpace(strings.ToLower(s))
}
