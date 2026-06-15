package entity

type FAQ struct {
	Answer       string   `yaml:"answer"`
	QuickReply   []string `yaml:"quick_reply"`
	MatchPhrases []string `yaml:"match_phrases"`
}

type Diagnosis struct {
	Name       string         `yaml:"name"`
	RichMenuID string         `yaml:"rich_menu_id"`
	FAQ        map[string]FAQ `yaml:"faq"`
}

type Escalation struct {
	Keywords []string `yaml:"keywords"`
	Reply    string   `yaml:"reply"`
}

type BotConfig struct {
	DefaultDiagnosis string               `yaml:"default_diagnosis"`
	UserDiagnosis    map[string]string    `yaml:"user_diagnosis"`
	Diagnoses        map[string]Diagnosis `yaml:"diagnoses"`
	SharedFAQ        map[string]FAQ       `yaml:"shared_faq"`
	Escalation       Escalation           `yaml:"escalation"`
	FallbackReply    string               `yaml:"fallback_reply"`
}
