package dataset

import (
	"testing"

	"github.com/hedgeg0d/tg-finetune-tools/internal/config"
	"github.com/hedgeg0d/tg-finetune-tools/internal/model"
)

func testCfg() config.Config {
	c := config.Default()
	c.Roles = config.Roles{AssistantID: "a", UserID: "u"}
	c.Build.MaxChars = 0
	return c
}

func msg(id, date int64, from, text string) model.Message {
	return model.Message{ID: id, Date: date, FromID: from, Text: text}
}

func TestBuildMergesBurstsAndRoles(t *testing.T) {
	in := []model.Message{
		msg(1, 0, "u", "hi"),
		msg(2, 1, "u", "there"),
		msg(3, 2, "a", "hey"),
	}
	convs := Build(in, testCfg())
	if len(convs) != 1 {
		t.Fatalf("convs=%d", len(convs))
	}
	tr := convs[0].Turns
	if len(tr) != 2 || tr[0].Role != "user" || tr[0].Content != "hi\nthere" || tr[1].Role != "assistant" {
		t.Fatalf("turns=%+v", tr)
	}
}

func TestBuildSplitsBySessionGap(t *testing.T) {
	cfg := testCfg()
	cfg.Build.SessionGapMinutes = 1
	in := []model.Message{
		msg(1, 0, "u", "a"),
		msg(2, 10, "a", "b"),
		msg(3, 10000, "u", "c"),
		msg(4, 10010, "a", "d"),
	}
	if got := len(Build(in, cfg)); got != 2 {
		t.Fatalf("sessions=%d", got)
	}
}

func TestBuildWindowsByMaxTurns(t *testing.T) {
	cfg := testCfg()
	cfg.Build.MaxTurns = 2
	var in []model.Message
	for i := int64(0); i < 8; i++ {
		from := "u"
		if i%2 == 1 {
			from = "a"
		}
		in = append(in, msg(i, i, from, "x"))
	}
	convs := Build(in, cfg)
	if len(convs) != 4 {
		t.Fatalf("windows=%d", len(convs))
	}
}

func TestBuildDropsLowAssistantContent(t *testing.T) {
	cfg := testCfg()
	cfg.Build.MinAssistantRunes = 5
	in := []model.Message{
		msg(1, 0, "u", "how are you doing today"),
		msg(2, 1, "a", "ok"),
	}
	if got := len(Build(in, cfg)); got != 0 {
		t.Fatalf("convs=%d", got)
	}
	cfg.Build.MinAssistantRunes = 1
	if got := len(Build(in, cfg)); got != 1 {
		t.Fatalf("convs=%d", got)
	}
}

func TestDedupRemovesIdenticalConversations(t *testing.T) {
	a := Conversation{Turns: []Turn{{Role: "user", Content: "hi"}, {Role: "assistant", Content: "yo"}}}
	b := Conversation{Turns: []Turn{{Role: "user", Content: "hi"}, {Role: "assistant", Content: "yo"}}}
	c := Conversation{Turns: []Turn{{Role: "user", Content: "bye"}, {Role: "assistant", Content: "ok"}}}
	got := Dedup([]Conversation{a, b, c})
	if len(got) != 2 {
		t.Fatalf("got %d", len(got))
	}
}

func TestBuildDropsConversationWithoutAssistant(t *testing.T) {
	in := []model.Message{msg(1, 0, "u", "a"), msg(2, 1, "u", "b")}
	if got := len(Build(in, testCfg())); got != 0 {
		t.Fatalf("convs=%d", got)
	}
}
