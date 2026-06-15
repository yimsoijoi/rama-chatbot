package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
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
	QuickReplies []string `yaml:"quick_replies,omitempty"`
	MatchPhrases []string `yaml:"match_phrases,omitempty"`
}

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "usage: go run ./scripts/cmd/export-faq-config <html_path> <output_yaml>\n")
		os.Exit(1)
	}
	htmlPath := os.Args[1]
	outPath := os.Args[2]

	raw, err := os.ReadFile(htmlPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read html: %v\n", err)
		os.Exit(1)
	}

	segment, err := extractDataArray(string(raw))
	if err != nil {
		fmt.Fprintf(os.Stderr, "extract data array: %v\n", err)
		os.Exit(1)
	}

	blocks := splitObjectBlocks(segment)
	items := make([]faqItem, 0, len(blocks))
	for _, b := range blocks {
		it, ok := parseBlock(b)
		if !ok {
			continue
		}
		items = append(items, it)
	}

	seed := faqSeed{Items: items}
	out, err := yaml.Marshal(seed)
	if err != nil {
		fmt.Fprintf(os.Stderr, "marshal yaml: %v\n", err)
		os.Exit(1)
	}
	if err := os.WriteFile(outPath, out, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "write yaml: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Exported %d FAQ entries to %s\n", len(items), outPath)
}

func extractDataArray(html string) (string, error) {
	const start = "const data = ["
	si := strings.Index(html, start)
	if si < 0 {
		return "", fmt.Errorf("data array start not found")
	}
	si += len(start)
	ei := strings.Index(html[si:], "];\n\n//")
	if ei < 0 {
		return "", fmt.Errorf("data array end not found")
	}
	return html[si : si+ei], nil
}

func splitObjectBlocks(segment string) []string {
	var blocks []string
	depth := 0
	start := -1
	inDouble := false
	inBacktick := false
	escaped := false

	for i := 0; i < len(segment); i++ {
		ch := segment[i]
		if escaped {
			escaped = false
			continue
		}
		if ch == '\\' {
			escaped = true
			continue
		}
		if !inBacktick && ch == '"' {
			inDouble = !inDouble
			continue
		}
		if !inDouble && ch == '`' {
			inBacktick = !inBacktick
			continue
		}
		if inDouble || inBacktick {
			continue
		}
		if ch == '{' {
			if depth == 0 {
				start = i
			}
			depth++
			continue
		}
		if ch == '}' {
			depth--
			if depth == 0 && start >= 0 {
				blocks = append(blocks, segment[start:i+1])
				start = -1
			}
		}
	}
	return blocks
}

var (
	reCode     = regexp.MustCompile(`code:"([^"]+)"`)
	reDX       = regexp.MustCompile(`dx:"([^"]+)"`)
	reQ        = regexp.MustCompile(`q:"([^"]+)"`)
	reBot      = regexp.MustCompile("bot:`(?s)(.*?)`")
	reReplies  = regexp.MustCompile(`replies:\[(.*?)\]`)
	reCategory = regexp.MustCompile(`category:"([^"]+)"`)
	reQuoted   = regexp.MustCompile(`"([^"]+)"`)
)

func parseBlock(block string) (faqItem, bool) {
	it := faqItem{
		Code:     findOne(reCode, block),
		DX:       findOne(reDX, block),
		Question: findOne(reQ, block),
		Answer:   findOne(reBot, block),
		Category: findOne(reCategory, block),
	}
	if it.Code == "" || it.DX == "" || it.Question == "" || it.Answer == "" {
		return faqItem{}, false
	}

	rawReplies := findOne(reReplies, block)
	for _, m := range reQuoted.FindAllStringSubmatch(rawReplies, -1) {
		if len(m) > 1 {
			it.QuickReplies = append(it.QuickReplies, m[1])
		}
	}

	it.MatchPhrases = []string{it.Code, it.Question}
	return it, true
}

func findOne(re *regexp.Regexp, s string) string {
	m := re.FindStringSubmatch(s)
	if len(m) < 2 {
		return ""
	}
	return strings.TrimSpace(m[1])
}
