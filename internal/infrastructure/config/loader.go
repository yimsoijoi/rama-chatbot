package config

import (
	"fmt"
	"os"

	"obgynrama-chatbot/internal/domain/entity"
	"obgynrama-chatbot/internal/observability"

	"gopkg.in/yaml.v3"
)

func LoadBotConfig(path string) (*entity.BotConfig, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, observability.NewAppError(
			"CFG_READ_FAILED",
			"config.LoadBotConfig.read",
			fmt.Sprintf("failed to read bot config from path=%s", path),
			err,
		)
	}

	var cfg entity.BotConfig
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		return nil, observability.NewAppError(
			"CFG_PARSE_FAILED",
			"config.LoadBotConfig.unmarshal",
			"failed to parse bot YAML config",
			err,
		)
	}

	if len(cfg.Diagnoses) == 0 {
		return nil, observability.NewAppError(
			"CFG_INVALID",
			"config.LoadBotConfig.validate",
			"config must contain at least one diagnosis",
			nil,
		)
	}

	if _, ok := cfg.Diagnoses[cfg.DefaultDiagnosis]; !ok {
		return nil, observability.NewAppError(
			"CFG_INVALID",
			"config.LoadBotConfig.validate",
			fmt.Sprintf("default diagnosis %q not found in diagnoses", cfg.DefaultDiagnosis),
			nil,
		)
	}

	if cfg.FallbackReply == "" {
		cfg.FallbackReply = "ขออภัยค่ะ ตอนนี้บอตยังไม่เข้าใจคำถามนี้ กรุณากดเมนูหรือพิมพ์ใหม่อีกครั้ง"
	}

	return &cfg, nil
}
