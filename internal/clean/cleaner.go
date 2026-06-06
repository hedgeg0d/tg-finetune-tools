package clean

import (
	"strconv"
	"unicode/utf8"

	"github.com/hedgeg0d/tg-finetune-tools/internal/config"
	"github.com/hedgeg0d/tg-finetune-tools/internal/model"
	"github.com/hedgeg0d/tg-finetune-tools/internal/telegram"
)

func Normalize(raw telegram.RawMessage, cfg config.Config) (model.Message, bool) {
	if raw.Type != "message" {
		return model.Message{}, false
	}
	if cfg.RoleOf(raw.FromID) == "" {
		return model.Message{}, false
	}
	if cfg.Clean.DropForwarded && raw.ForwardedFrom != nil {
		return model.Message{}, false
	}
	if cfg.Clean.DropViaBot && raw.ViaBot != "" {
		return model.Message{}, false
	}

	text := Flatten(raw, cfg.Clean.StripLinks)

	if raw.MediaType == "sticker" && text == "" {
		if cfg.Clean.StickersToEmoji && raw.StickerEmoji != "" {
			text = raw.StickerEmoji
		}
	}

	if raw.MediaType != "" && text == "" {
		return model.Message{}, false
	}
	if text == "" {
		return model.Message{}, false
	}
	if utf8.RuneCountInString(text) < cfg.Clean.MinRunes {
		return model.Message{}, false
	}

	date, _ := strconv.ParseInt(raw.DateUnixtime, 10, 64)

	return model.Message{
		ID:     raw.ID,
		Date:   date,
		FromID: raw.FromID,
		From:   raw.From,
		Text:   text,
	}, true
}
