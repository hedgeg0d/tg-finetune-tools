package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"runtime"
)

type Roles struct {
	AssistantID string `json:"assistant_id"`
	UserID      string `json:"user_id"`
}

type Clean struct {
	StripLinks      bool `json:"strip_links"`
	StickersToEmoji bool `json:"stickers_to_emoji"`
	DropForwarded   bool `json:"drop_forwarded"`
	DropViaBot      bool `json:"drop_via_bot"`
	RedactPII       bool `json:"redact_pii"`
	MinRunes        int  `json:"min_runes"`
}

type Generated struct {
	APIBase        string  `json:"api_base"`
	APIStyle       string  `json:"api_style"`
	APIKeyEnv      string  `json:"api_key_env"`
	Model          string  `json:"model"`
	BatchSize      int     `json:"batch_size"`
	ContextSamples int     `json:"context_samples"`
	Temperature    float64 `json:"temperature"`
	Instruction    string  `json:"instruction"`
	CacheFile      string  `json:"cache_file"`
}

type System struct {
	Mode      string    `json:"mode"`
	Fixed     string    `json:"fixed"`
	Pool      []string  `json:"pool"`
	Generated Generated `json:"generated"`
}

type Build struct {
	SessionGapMinutes int     `json:"session_gap_minutes"`
	MaxTurns          int     `json:"max_turns"`
	MaxChars          int     `json:"max_chars"`
	MaxTokens         int     `json:"max_tokens"`
	TokenEncoding     string  `json:"token_encoding"`
	MinTurns          int     `json:"min_turns"`
	Format            string  `json:"format"`
	System            System  `json:"system"`
	StartWithUser     bool    `json:"start_with_user"`
	RequireAssistant  bool    `json:"require_assistant"`
	BurstSeparator    string  `json:"burst_separator"`
	ValRatio          float64 `json:"val_ratio"`
	Seed              int64   `json:"seed"`
	Dedup             bool    `json:"dedup"`
	MinAssistantRunes int     `json:"min_assistant_runes"`
}

type Memory struct {
	WindowTokens      int       `json:"window_tokens"`
	TokenEncoding     string    `json:"token_encoding"`
	Workers           int       `json:"workers"`
	CacheFile         string    `json:"cache_file"`
	Extract           Generated `json:"extract"`
	DoConsolidate     bool      `json:"do_consolidate"`
	ConsolidatePasses int       `json:"consolidate_passes"`
	Consolidate       Generated `json:"consolidate"`
	DoEmbeddings      bool      `json:"do_embeddings"`
	Embeddings        Generated `json:"embeddings"`
}

type Config struct {
	Workers int    `json:"workers"`
	Roles   Roles  `json:"roles"`
	Clean   Clean  `json:"clean"`
	Build   Build  `json:"build"`
	Memory  Memory `json:"memory"`
}

func Default() Config {
	return Config{
		Workers: runtime.NumCPU(),
		Clean: Clean{
			StripLinks:      true,
			StickersToEmoji: true,
			DropForwarded:   true,
			DropViaBot:      true,
			RedactPII:       false,
			MinRunes:        1,
		},
		Build: Build{
			SessionGapMinutes: 180,
			MaxTurns:          20,
			MaxChars:          6000,
			MaxTokens:         0,
			TokenEncoding:     "cl100k_base",
			MinTurns:          2,
			Format:            "openai",
			StartWithUser:     true,
			RequireAssistant:  true,
			BurstSeparator:    "\n",
			ValRatio:          0,
			Seed:              42,
			Dedup:             true,
			System: System{
				Mode: "empty",
				Generated: Generated{
					APIBase:        "https://api.openai.com/v1",
					APIKeyEnv:      "OPENAI_API_KEY",
					Model:          "gpt-4o-mini",
					BatchSize:      100,
					ContextSamples: 20,
					Temperature:    0.8,
					CacheFile:      "system_prompts.json",
				},
			},
		},
		Memory: Memory{
			WindowTokens:      1500,
			TokenEncoding:     "cl100k_base",
			Workers:           4,
			CacheFile:         "memory_facts.json",
			DoConsolidate:     true,
			ConsolidatePasses: 2,
			DoEmbeddings:      true,
			Extract: Generated{
				APIBase:     "http://localhost:11434",
				APIStyle:    "ollama",
				Model:       "qwen3.5:9b",
				Temperature: 0.2,
			},
			Consolidate: Generated{
				APIBase:     "http://localhost:11434",
				APIStyle:    "ollama",
				Model:       "qwen3.5:9b",
				Temperature: 0.3,
				BatchSize:   40,
			},
			Embeddings: Generated{
				APIBase:  "http://localhost:11434",
				APIStyle: "ollama",
				Model:    "nomic-embed-text",
			},
		},
	}
}

func Load(path string) (Config, error) {
	cfg := Default()
	if path == "" {
		return cfg, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, fmt.Errorf("read config: %w", err)
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("parse config: %w", err)
	}
	if cfg.Workers < 1 {
		cfg.Workers = runtime.NumCPU()
	}
	return cfg, nil
}

func (c Config) Validate() error {
	if c.Roles.AssistantID == "" || c.Roles.UserID == "" {
		return errors.New("roles.assistant_id and roles.user_id must be set (run `tgprep inspect` to find them)")
	}
	switch c.Build.Format {
	case "openai", "sharegpt":
	default:
		return fmt.Errorf("unknown build.format %q (want openai|sharegpt)", c.Build.Format)
	}
	if c.Build.ValRatio < 0 || c.Build.ValRatio >= 1 {
		return fmt.Errorf("build.val_ratio must be in [0, 1), got %v", c.Build.ValRatio)
	}
	switch c.Build.System.Mode {
	case "", "empty", "fixed":
	case "pool":
		if len(c.Build.System.Pool) == 0 {
			return errors.New("build.system.pool must be non-empty for pool mode")
		}
	case "generated":
		g := c.Build.System.Generated
		if g.APIBase == "" || g.Model == "" {
			return errors.New("build.system.generated requires api_base and model")
		}
		if g.BatchSize < 1 {
			return errors.New("build.system.generated.batch_size must be >= 1")
		}
		switch g.APIStyle {
		case "", "openai", "ollama":
		default:
			return fmt.Errorf("unknown build.system.generated.api_style %q (want openai|ollama)", g.APIStyle)
		}
	default:
		return fmt.Errorf("unknown build.system.mode %q (want empty|fixed|pool|generated)", c.Build.System.Mode)
	}
	return nil
}

func (c Config) RoleOf(fromID string) string {
	switch fromID {
	case c.Roles.AssistantID:
		return "assistant"
	case c.Roles.UserID:
		return "user"
	default:
		return ""
	}
}
