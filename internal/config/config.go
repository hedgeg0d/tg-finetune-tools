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

type Build struct {
	SessionGapMinutes int     `json:"session_gap_minutes"`
	MaxTurns          int     `json:"max_turns"`
	MaxChars          int     `json:"max_chars"`
	MinTurns          int     `json:"min_turns"`
	Format            string  `json:"format"`
	System            string  `json:"system"`
	StartWithUser     bool    `json:"start_with_user"`
	RequireAssistant  bool    `json:"require_assistant"`
	BurstSeparator    string  `json:"burst_separator"`
	ValRatio          float64 `json:"val_ratio"`
	Seed              int64   `json:"seed"`
	Dedup             bool    `json:"dedup"`
}

type Config struct {
	Workers int   `json:"workers"`
	Roles   Roles `json:"roles"`
	Clean   Clean `json:"clean"`
	Build   Build `json:"build"`
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
			MinTurns:          2,
			Format:            "openai",
			StartWithUser:     true,
			RequireAssistant:  true,
			BurstSeparator:    "\n",
			ValRatio:          0,
			Seed:              42,
			Dedup:             true,
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
