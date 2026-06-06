package clean

import (
	"strconv"
	"unicode/utf8"

	"github.com/hedgeg0d/tg-finetune-tools/internal/config"
	"github.com/hedgeg0d/tg-finetune-tools/internal/model"
	"github.com/hedgeg0d/tg-finetune-tools/internal/telegram"
)

func Normalize(raw telegram.RawMessage, cfg config.Config) (model.Message, Reason) {
	if raw.Type != "message" {
		return model.Message{}, DropService
	}
	if cfg.RoleOf(raw.FromID) == "" {
		return model.Message{}, DropUnknownSender
	}
	if cfg.Clean.DropForwarded && raw.ForwardedFrom != nil {
		return model.Message{}, DropForwarded
	}
	if cfg.Clean.DropViaBot && raw.ViaBot != "" {
		return model.Message{}, DropViaBot
	}

	text := Flatten(raw, cfg.Clean.StripLinks, cfg.Clean.RedactPII)

	if raw.MediaType == "sticker" && text == "" {
		if cfg.Clean.StickersToEmoji && raw.StickerEmoji != "" {
			text = raw.StickerEmoji
		}
	}

	if raw.MediaType != "" && text == "" {
		return model.Message{}, DropMedia
	}
	if text == "" {
		return model.Message{}, DropEmpty
	}
	if utf8.RuneCountInString(text) < cfg.Clean.MinRunes {
		return model.Message{}, DropShort
	}

	date, _ := strconv.ParseInt(raw.DateUnixtime, 10, 64)

	return model.Message{
		ID:     raw.ID,
		Date:   date,
		FromID: raw.FromID,
		From:   raw.From,
		Text:   text,
	}, Kept
}
