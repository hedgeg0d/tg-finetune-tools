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
	if _, r := Normalize(telegram.RawMessage{Type: "service", FromID: "u"}, testCfg()); r != DropService {
		t.Fatalf("reason=%v", r)
	}
}

func TestNormalizeDropsUnknownSender(t *testing.T) {
	m := telegram.RawMessage{Type: "message", FromID: "x", TextEntities: []telegram.Entity{{Type: "plain", Text: "hi"}}}
	if _, r := Normalize(m, testCfg()); r != DropUnknownSender {
		t.Fatalf("reason=%v", r)
	}
}

func TestNormalizeStickerToEmoji(t *testing.T) {
	m := telegram.RawMessage{Type: "message", FromID: "u", MediaType: "sticker", StickerEmoji: "🔥"}
	got, r := Normalize(m, testCfg())
	if r != Kept || got.Text != "🔥" {
		t.Fatalf("got %q reason=%v", got.Text, r)
	}
}

func TestNormalizeDropsMediaWithoutText(t *testing.T) {
	m := telegram.RawMessage{Type: "message", FromID: "u", MediaType: "video_file"}
	if _, r := Normalize(m, testCfg()); r != DropMedia {
		t.Fatalf("reason=%v", r)
	}
}

func TestNormalizeRedactsPII(t *testing.T) {
	cfg := testCfg()
	cfg.Clean.RedactPII = true
	m := telegram.RawMessage{
		Type:   "message",
		FromID: "u",
		TextEntities: []telegram.Entity{
			{Type: "plain", Text: "call me "},
			{Type: "phone", Text: "+1234567890"},
		},
	}
	got, r := Normalize(m, cfg)
	if r != Kept || got.Text != "call me" {
		t.Fatalf("got %q reason=%v", got.Text, r)
	}
}
