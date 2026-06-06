package dataset

import "github.com/hedgeg0d/tg-finetune-tools/internal/config"

type oaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type oaRecord struct {
	Messages []oaMessage `json:"messages"`
}

type sgTurn struct {
	From  string `json:"from"`
	Value string `json:"value"`
}

type sgRecord struct {
	Conversations []sgTurn `json:"conversations"`
}

func Encode(c Conversation, cfg config.Config) any {
	switch cfg.Build.Format {
	case "sharegpt":
		return shareGPT(c, cfg)
	default:
		return openAI(c, cfg)
	}
}

func openAI(c Conversation, cfg config.Config) oaRecord {
	rec := oaRecord{}
	if cfg.Build.System != "" {
		rec.Messages = append(rec.Messages, oaMessage{Role: "system", Content: cfg.Build.System})
	}
	for _, t := range c.Turns {
		rec.Messages = append(rec.Messages, oaMessage{Role: t.Role, Content: t.Content})
	}
	return rec
}

func shareGPT(c Conversation, cfg config.Config) sgRecord {
	rec := sgRecord{}
	if cfg.Build.System != "" {
		rec.Conversations = append(rec.Conversations, sgTurn{From: "system", Value: cfg.Build.System})
	}
	for _, t := range c.Turns {
		rec.Conversations = append(rec.Conversations, sgTurn{From: sgRole(t.Role), Value: t.Content})
	}
	return rec
}

func sgRole(role string) string {
	if role == "assistant" {
		return "gpt"
	}
	return "human"
}
