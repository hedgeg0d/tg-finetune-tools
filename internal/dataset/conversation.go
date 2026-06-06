package dataset

import (
	"sort"
	"unicode/utf8"

	"github.com/hedgeg0d/tg-finetune-tools/internal/config"
	"github.com/hedgeg0d/tg-finetune-tools/internal/model"
)

type Turn struct {
	Role    string
	Content string
}

type Conversation struct {
	Turns []Turn
}

type Measure func(string) int

func Build(msgs []model.Message, cfg config.Config, measure Measure) []Conversation {
	sort.SliceStable(msgs, func(i, j int) bool {
		if msgs[i].Date != msgs[j].Date {
			return msgs[i].Date < msgs[j].Date
		}
		return msgs[i].ID < msgs[j].ID
	})

	gap := int64(cfg.Build.SessionGapMinutes) * 60
	var out []Conversation
	var session []model.Message

	flush := func() {
		out = append(out, assembleSession(session, cfg, measure)...)
		session = session[:0]
	}

	var prev int64
	for _, m := range msgs {
		if len(session) > 0 && m.Date-prev > gap {
			flush()
		}
		session = append(session, m)
		prev = m.Date
	}
	flush()

	return out
}

func assembleSession(msgs []model.Message, cfg config.Config, measure Measure) []Conversation {
	if len(msgs) == 0 {
		return nil
	}

	var turns []Turn
	for _, m := range msgs {
		role := cfg.RoleOf(m.FromID)
		if n := len(turns); n > 0 && turns[n-1].Role == role {
			turns[n-1].Content += cfg.Build.BurstSeparator + m.Text
			continue
		}
		turns = append(turns, Turn{Role: role, Content: m.Text})
	}

	var out []Conversation
	for _, window := range chunk(turns, cfg, measure) {
		if conv, ok := finalize(window, cfg); ok {
			out = append(out, conv)
		}
	}
	return out
}

func chunk(turns []Turn, cfg config.Config, measure Measure) [][]Turn {
	maxTurns := cfg.Build.MaxTurns
	maxSize := cfg.Build.MaxChars
	if cfg.Build.MaxTokens > 0 {
		maxSize = cfg.Build.MaxTokens
	}
	if maxTurns <= 0 && maxSize <= 0 {
		return [][]Turn{turns}
	}

	var windows [][]Turn
	var cur []Turn
	size := 0

	for _, t := range turns {
		c := measure(t.Content)
		overTurns := maxTurns > 0 && len(cur) >= maxTurns
		overSize := maxSize > 0 && size+c > maxSize && len(cur) > 0
		if overTurns || overSize {
			windows = append(windows, cur)
			cur = nil
			size = 0
		}
		cur = append(cur, t)
		size += c
	}
	if len(cur) > 0 {
		windows = append(windows, cur)
	}
	return windows
}

func finalize(turns []Turn, cfg config.Config) (Conversation, bool) {
	if cfg.Build.StartWithUser {
		for len(turns) > 0 && turns[0].Role != "user" {
			turns = turns[1:]
		}
	}
	if cfg.Build.RequireAssistant {
		for len(turns) > 0 && turns[len(turns)-1].Role != "assistant" {
			turns = turns[:len(turns)-1]
		}
		if !hasAssistant(turns) {
			return Conversation{}, false
		}
	}
	if len(turns) < cfg.Build.MinTurns {
		return Conversation{}, false
	}
	if assistantRunes(turns) < cfg.Build.MinAssistantRunes {
		return Conversation{}, false
	}
	return Conversation{Turns: turns}, true
}

func assistantRunes(turns []Turn) int {
	total := 0
	for _, t := range turns {
		if t.Role == "assistant" {
			total += utf8.RuneCountInString(t.Content)
		}
	}
	return total
}

func hasAssistant(turns []Turn) bool {
	for _, t := range turns {
		if t.Role == "assistant" {
			return true
		}
	}
	return false
}
