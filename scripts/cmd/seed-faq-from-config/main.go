package main

import (
	"database/sql"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
	_ "modernc.org/sqlite"
)

type faqSeed struct {
	Items []faqItem `yaml:"items"`
}

type faqItem struct {
	Code         string   `yaml:"code"`
	DX           string   `yaml:"dx"`
	Category     string   `yaml:"category"`
	Question     string   `yaml:"question"`
	Answer       string   `yaml:"answer"`
	QuickReplies []string `yaml:"quick_replies"`
	MatchPhrases []string `yaml:"match_phrases"`
}

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "usage: go run ./scripts/cmd/seed-faq-from-config <db_path> <seed_yaml>\n")
		os.Exit(1)
	}
	dbPath := os.Args[1]
	configPath := os.Args[2]

	raw, err := os.ReadFile(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read config: %v\n", err)
		os.Exit(1)
	}

	var seed faqSeed
	if err := yaml.Unmarshal(raw, &seed); err != nil {
		fmt.Fprintf(os.Stderr, "parse yaml: %v\n", err)
		os.Exit(1)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open db: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		fmt.Fprintf(os.Stderr, "begin tx: %v\n", err)
		os.Exit(1)
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM faq_quick_reply; DELETE FROM faq_match_phrase; DELETE FROM faq_reply;`); err != nil {
		fmt.Fprintf(os.Stderr, "clear old faq data: %v\n", err)
		os.Exit(1)
	}

	for _, it := range seed.Items {
		it.Code = strings.TrimSpace(it.Code)
		it.DX = strings.TrimSpace(it.DX)
		it.Question = strings.TrimSpace(it.Question)
		it.Answer = strings.TrimSpace(it.Answer)
		it.Category = strings.TrimSpace(it.Category)
		if it.Code == "" || it.DX == "" || it.Question == "" || it.Answer == "" {
			continue
		}

		if _, err := tx.Exec(`
			INSERT INTO faq_reply (diagnosis, faq_key, question, category, answer)
			VALUES (?, ?, ?, ?, ?)
			ON CONFLICT(diagnosis, faq_key) DO UPDATE SET
				question = excluded.question,
				category = excluded.category,
				answer = excluded.answer
		`, it.DX, it.Code, it.Question, it.Category, it.Answer); err != nil {
			fmt.Fprintf(os.Stderr, "insert faq_reply %s: %v\n", it.Code, err)
			os.Exit(1)
		}

		phrases := unique(append([]string{it.Code, it.Question}, it.MatchPhrases...))
		for _, p := range phrases {
			p = strings.TrimSpace(p)
			if p == "" {
				continue
			}
			if _, err := tx.Exec(`
				INSERT INTO faq_match_phrase (diagnosis, faq_key, phrase)
				VALUES (?, ?, ?)
				ON CONFLICT(diagnosis, faq_key, phrase) DO NOTHING
			`, it.DX, it.Code, p); err != nil {
				fmt.Fprintf(os.Stderr, "insert faq_match_phrase %s: %v\n", it.Code, err)
				os.Exit(1)
			}
		}

		for _, r := range unique(it.QuickReplies) {
			r = strings.TrimSpace(r)
			if r == "" {
				continue
			}
			if _, err := tx.Exec(`
				INSERT INTO faq_quick_reply (diagnosis, faq_key, quick_reply)
				VALUES (?, ?, ?)
				ON CONFLICT(diagnosis, faq_key, quick_reply) DO NOTHING
			`, it.DX, it.Code, r); err != nil {
				fmt.Fprintf(os.Stderr, "insert faq_quick_reply %s: %v\n", it.Code, err)
				os.Exit(1)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		fmt.Fprintf(os.Stderr, "commit: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Seeded %d FAQ entries from %s into %s\n", len(seed.Items), configPath, dbPath)
}

func unique(in []string) []string {
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
