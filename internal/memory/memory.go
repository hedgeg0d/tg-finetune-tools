package memory

import (
	"fmt"
	"os"
	"sort"

	"github.com/hedgeg0d/tg-finetune-tools/internal/config"
	"github.com/hedgeg0d/tg-finetune-tools/internal/dataset"
	"github.com/hedgeg0d/tg-finetune-tools/internal/model"
	"github.com/hedgeg0d/tg-finetune-tools/internal/tokenize"
)

type Stats struct {
	Windows      int
	RawFacts     int
	Consolidated int
	Embedded     int
}

type Record struct {
	Text      string    `json:"text"`
	Embedding []float64 `json:"embedding,omitempty"`
}

type fact struct {
	Text string `json:"text"`
	Date int64  `json:"date"`
}

func Run(inPath, outPath string, mcfg config.Memory, roles config.Roles) (Stats, error) {
	in, err := os.Open(inPath)
	if err != nil {
		return Stats{}, err
	}
	msgs, err := model.ReadAll(in)
	in.Close()
	if err != nil {
		return Stats{}, err
	}

	msgs = filterRoles(msgs, roles)
	sort.SliceStable(msgs, func(i, j int) bool { return msgs[i].Date < msgs[j].Date })

	counter, err := tokenize.New(mcfg.TokenEncoding)
	if err != nil {
		return Stats{}, err
	}
	wins := windows(msgs, roles, counter.Count, mcfg.WindowTokens)

	facts, err := extractAll(wins, roles, mcfg)
	if err != nil {
		return Stats{}, err
	}
	stats := Stats{Windows: len(wins), RawFacts: len(facts)}

	if mcfg.DoConsolidate {
		facts, err = consolidate(facts, mcfg)
		if err != nil {
			return stats, err
		}
	}
	stats.Consolidated = len(facts)

	records := make([]Record, len(facts))
	for i, f := range facts {
		records[i] = Record{Text: f.Text}
	}

	if mcfg.DoEmbeddings {
		if err := embedAll(records, mcfg); err != nil {
			return stats, err
		}
		stats.Embedded = len(records)
	}

	if err := write(outPath, records); err != nil {
		return stats, err
	}
	return stats, nil
}

func filterRoles(msgs []model.Message, roles config.Roles) []model.Message {
	known := map[string]bool{roles.AssistantID: true, roles.UserID: true}
	out := msgs[:0]
	for _, m := range msgs {
		if known[m.FromID] {
			out = append(out, m)
		}
	}
	return out
}

func windows(msgs []model.Message, roles config.Roles, count func(string) int, maxTokens int) [][]model.Message {
	var out [][]model.Message
	var cur []model.Message
	size := 0
	for _, m := range msgs {
		c := count(line(m, roles))
		if size+c > maxTokens && len(cur) > 0 {
			out = append(out, cur)
			cur = nil
			size = 0
		}
		cur = append(cur, m)
		size += c
	}
	if len(cur) > 0 {
		out = append(out, cur)
	}
	return out
}

func line(m model.Message, roles config.Roles) string {
	role := "user"
	if m.FromID == roles.AssistantID {
		role = "assistant"
	}
	return role + ": " + m.Text
}

func write(path string, records []Record) error {
	out, err := os.Create(path)
	if err != nil {
		return err
	}
	defer out.Close()
	w := dataset.NewWriter(out)
	for _, r := range records {
		if err := w.Write(r); err != nil {
			return err
		}
	}
	return w.Flush()
}

func progress(label string, done, total int) {
	fmt.Fprintf(os.Stderr, "\r  %s: %d/%d   ", label, done, total)
}

func progressDone() {
	fmt.Fprintln(os.Stderr)
}
