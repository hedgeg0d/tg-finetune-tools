package main

import (
	"flag"

	"github.com/hedgeg0d/tg-finetune-tools/internal/config"
)

type overrides struct {
	assistantID      string
	userID           string
	stripLinks       bool
	stickersToEmoji  bool
	dropForwarded    bool
	dropViaBot       bool
	redactPII        bool
	minRunes         int
	gap              int
	maxTurns         int
	maxChars         int
	minTurns         int
	format           string
	system           string
	startWithUser    bool
	requireAssistant bool
	burstSep         string
	valRatio         float64
	seed             int64
}

func registerOverrides(fs *flag.FlagSet) *overrides {
	o := &overrides{}
	fs.StringVar(&o.assistantID, "assistant-id", "", "override roles.assistant_id")
	fs.StringVar(&o.userID, "user-id", "", "override roles.user_id")
	fs.BoolVar(&o.stripLinks, "strip-links", false, "override clean.strip_links")
	fs.BoolVar(&o.stickersToEmoji, "stickers-emoji", false, "override clean.stickers_to_emoji")
	fs.BoolVar(&o.dropForwarded, "drop-forwarded", false, "override clean.drop_forwarded")
	fs.BoolVar(&o.dropViaBot, "drop-via-bot", false, "override clean.drop_via_bot")
	fs.BoolVar(&o.redactPII, "redact-pii", false, "override clean.redact_pii (drop phone/email)")
	fs.IntVar(&o.minRunes, "min-runes", 0, "override clean.min_runes")
	fs.IntVar(&o.gap, "gap", 0, "override build.session_gap_minutes")
	fs.IntVar(&o.maxTurns, "max-turns", 0, "override build.max_turns")
	fs.IntVar(&o.maxChars, "max-chars", 0, "override build.max_chars")
	fs.IntVar(&o.minTurns, "min-turns", 0, "override build.min_turns")
	fs.StringVar(&o.format, "format", "", "override build.format (openai|sharegpt)")
	fs.StringVar(&o.system, "system", "", "override build.system prompt")
	fs.BoolVar(&o.startWithUser, "start-with-user", false, "override build.start_with_user")
	fs.BoolVar(&o.requireAssistant, "require-assistant", false, "override build.require_assistant")
	fs.StringVar(&o.burstSep, "burst-sep", "", "override build.burst_separator")
	fs.Float64Var(&o.valRatio, "val-ratio", 0, "override build.val_ratio (0 = no split)")
	fs.Int64Var(&o.seed, "seed", 0, "override build.seed (shuffle for val split)")
	return o
}

func (o *overrides) apply(fs *flag.FlagSet, cfg *config.Config) {
	fs.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "assistant-id":
			cfg.Roles.AssistantID = o.assistantID
		case "user-id":
			cfg.Roles.UserID = o.userID
		case "strip-links":
			cfg.Clean.StripLinks = o.stripLinks
		case "stickers-emoji":
			cfg.Clean.StickersToEmoji = o.stickersToEmoji
		case "drop-forwarded":
			cfg.Clean.DropForwarded = o.dropForwarded
		case "drop-via-bot":
			cfg.Clean.DropViaBot = o.dropViaBot
		case "redact-pii":
			cfg.Clean.RedactPII = o.redactPII
		case "min-runes":
			cfg.Clean.MinRunes = o.minRunes
		case "gap":
			cfg.Build.SessionGapMinutes = o.gap
		case "max-turns":
			cfg.Build.MaxTurns = o.maxTurns
		case "max-chars":
			cfg.Build.MaxChars = o.maxChars
		case "min-turns":
			cfg.Build.MinTurns = o.minTurns
		case "format":
			cfg.Build.Format = o.format
		case "system":
			cfg.Build.System = o.system
		case "start-with-user":
			cfg.Build.StartWithUser = o.startWithUser
		case "require-assistant":
			cfg.Build.RequireAssistant = o.requireAssistant
		case "burst-sep":
			cfg.Build.BurstSeparator = o.burstSep
		case "val-ratio":
			cfg.Build.ValRatio = o.valRatio
		case "seed":
			cfg.Build.Seed = o.seed
		}
	})
}
