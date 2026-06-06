package clean

import (
	"encoding/json"
	"strings"

	"github.com/hedgeg0d/tg-finetune-tools/internal/telegram"
)

func Flatten(m telegram.RawMessage, stripLinks bool) string {
	if len(m.TextEntities) > 0 {
		var b strings.Builder
		for _, e := range m.TextEntities {
			if e.Type == "link" && stripLinks {
				continue
			}
			b.WriteString(e.Text)
		}
		return collapse(b.String())
	}
	return collapse(fromRaw(m.Text, stripLinks))
}

func fromRaw(raw json.RawMessage, stripLinks bool) string {
	if len(raw) == 0 {
		return ""
	}

	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s
	}

	var parts []json.RawMessage
	if json.Unmarshal(raw, &parts) != nil {
		return ""
	}

	var b strings.Builder
	for _, p := range parts {
		var str string
		if json.Unmarshal(p, &str) == nil {
			b.WriteString(str)
			continue
		}
		var ent telegram.Entity
		if json.Unmarshal(p, &ent) == nil {
			if ent.Type == "link" && stripLinks {
				continue
			}
			b.WriteString(ent.Text)
		}
	}
	return b.String()
}

func collapse(s string) string {
	return strings.Join(strings.Fields(s), " ")
}
