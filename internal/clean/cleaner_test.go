package clean

import (
	"testing"

	"github.com/hedgeg0d/tg-finetune-tools/internal/config"
	"github.com/hedgeg0d/tg-finetune-tools/internal/telegram"
)

func testCfg() config.Config {
	c := config.Default()
	c.Roles = config.Roles{AssistantID: "a", UserID: "u"}
	return c
}

func TestNormalizeDropsService(t *testing.T) {
	if _, ok := Normalize(telegram.RawMessage{Type: "service", FromID: "u"}, testCfg()); ok {
		t.Fatal("service kept")
	}
}

func TestNormalizeDropsUnknownSender(t *testing.T) {
	m := telegram.RawMessage{Type: "message", FromID: "x", TextEntities: []telegram.Entity{{Type: "plain", Text: "hi"}}}
	if _, ok := Normalize(m, testCfg()); ok {
		t.Fatal("unknown sender kept")
	}
}

func TestNormalizeStickerToEmoji(t *testing.T) {
	m := telegram.RawMessage{Type: "message", FromID: "u", MediaType: "sticker", StickerEmoji: "🔥"}
	got, ok := Normalize(m, testCfg())
	if !ok || got.Text != "🔥" {
		t.Fatalf("got %q ok=%v", got.Text, ok)
	}
}

func TestNormalizeDropsMediaWithoutText(t *testing.T) {
	m := telegram.RawMessage{Type: "message", FromID: "u", MediaType: "video_file"}
	if _, ok := Normalize(m, testCfg()); ok {
		t.Fatal("empty media kept")
	}
}
