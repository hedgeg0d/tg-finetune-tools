package clean

import (
	"encoding/json"
	"testing"

	"github.com/hedgeg0d/tg-finetune-tools/internal/telegram"
)

func TestFlattenStripsBareLinks(t *testing.T) {
	m := telegram.RawMessage{
		TextEntities: []telegram.Entity{
			{Type: "plain", Text: "see "},
			{Type: "link", Text: "https://example.com"},
			{Type: "plain", Text: " now"},
		},
	}
	if got := Flatten(m, true, false); got != "see now" {
		t.Fatalf("strip on: got %q", got)
	}
	if got := Flatten(m, false, false); got != "see https://example.com now" {
		t.Fatalf("strip off: got %q", got)
	}
}

func TestFlattenTextLinkKeepsAnchor(t *testing.T) {
	m := telegram.RawMessage{
		TextEntities: []telegram.Entity{
			{Type: "text_link", Text: "click", Href: "https://x.y"},
		},
	}
	if got := Flatten(m, true, false); got != "click" {
		t.Fatalf("got %q", got)
	}
}

func TestFlattenArrayFallback(t *testing.T) {
	m := telegram.RawMessage{Text: json.RawMessage(`[{"type":"spoiler","text":"hi"},""]`)}
	if got := Flatten(m, true, false); got != "hi" {
		t.Fatalf("got %q", got)
	}
}
