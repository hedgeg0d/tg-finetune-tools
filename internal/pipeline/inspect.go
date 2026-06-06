package pipeline

import (
	"io"
	"os"
	"sort"

	"github.com/hedgeg0d/tg-finetune-tools/internal/progress"
	"github.com/hedgeg0d/tg-finetune-tools/internal/telegram"
)

type Participant struct {
	FromID string
	Name   string
	Count  int
}

type Report struct {
	Total        int
	Participants []Participant
	Media        map[string]int
}

func Inspect(inPath string, prog bool) (Report, error) {
	in, err := os.Open(inPath)
	if err != nil {
		return Report{}, err
	}
	defer in.Close()

	var src io.Reader = in
	if prog {
		if fi, err := in.Stat(); err == nil && fi.Size() > 0 {
			pr := progress.NewReader(in)
			stop := progress.Track("reading", fi.Size(), pr.Bytes)
			defer stop()
			src = pr
		}
	}

	byID := map[string]*Participant{}
	media := map[string]int{}
	total := 0

	err = telegram.Stream(src, func(raw telegram.RawMessage) error {
		total++
		if raw.FromID != "" {
			p := byID[raw.FromID]
			if p == nil {
				p = &Participant{FromID: raw.FromID, Name: raw.From}
				byID[raw.FromID] = p
			}
			p.Count++
		}
		if raw.MediaType != "" {
			media[raw.MediaType]++
		}
		return nil
	})
	if err != nil {
		return Report{}, err
	}

	parts := make([]Participant, 0, len(byID))
	for _, p := range byID {
		parts = append(parts, *p)
	}
	sort.Slice(parts, func(i, j int) bool { return parts[i].Count > parts[j].Count })

	return Report{Total: total, Participants: parts, Media: media}, nil
}
