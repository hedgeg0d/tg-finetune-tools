package memory

import (
	"encoding/json"
	"strings"
	"sync"

	"github.com/hedgeg0d/tg-finetune-tools/internal/config"
	"github.com/hedgeg0d/tg-finetune-tools/internal/llm"
	"github.com/hedgeg0d/tg-finetune-tools/internal/model"
)

const defaultExtractInstruction = "Ниже фрагмент личной переписки, реплики помечены ролями. " +
	"Извлеки устойчивые факты о человеке с ролью «assistant»: его жизнь, работа, учёба, отношения, " +
	"предпочтения, привычки, взгляды — только то, что явно следует из текста, без домыслов. " +
	"Верни ТОЛЬКО JSON-массив строк, каждая — один краткий факт на русском. Если фактов нет, верни []."

type extractCache struct {
	Processed int    `json:"processed"`
	Facts     []fact `json:"facts"`
}

func extractAll(wins [][]model.Message, roles config.Roles, mcfg config.Memory) ([]fact, error) {
	cache, _ := loadJSON[extractCache](mcfg.CacheFile)
	facts := cache.Facts
	start := cache.Processed

	client := llm.New(mcfg.Extract)
	instr := mcfg.Extract.Instruction
	if strings.TrimSpace(instr) == "" {
		instr = defaultExtractInstruction
	}

	workers := mcfg.Workers
	if workers < 1 {
		workers = 1
	}

	for base := start; base < len(wins); base += workers {
		end := min(base+workers, len(wins))
		batch := wins[base:end]

		results := make([][]fact, len(batch))
		errs := make([]error, len(batch))
		var wg sync.WaitGroup
		for i := range batch {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				results[i], errs[i] = extractWindow(client, instr, batch[i], roles)
			}(i)
		}
		wg.Wait()

		for i := range batch {
			if errs[i] != nil {
				saveJSON(mcfg.CacheFile, extractCache{Processed: base + i, Facts: facts})
				return facts, errs[i]
			}
			facts = append(facts, results[i]...)
		}
		progress("extracting facts", end, len(wins))
		saveJSON(mcfg.CacheFile, extractCache{Processed: end, Facts: facts})
	}
	progressDone()
	return facts, nil
}

func extractWindow(client *llm.Client, instr string, win []model.Message, roles config.Roles) ([]fact, error) {
	var b strings.Builder
	for _, m := range win {
		b.WriteString(line(m, roles))
		b.WriteByte('\n')
	}
	out, err := client.Generate(instr, b.String())
	if err != nil {
		return nil, err
	}
	date := win[len(win)-1].Date
	texts := parseList(out)
	facts := make([]fact, 0, len(texts))
	for _, t := range texts {
		t = strings.TrimSpace(t)
		if t != "" {
			facts = append(facts, fact{Text: t, Date: date})
		}
	}
	return facts, nil
}

func parseList(s string) []string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '['); i >= 0 {
		if j := strings.LastIndexByte(s, ']'); j > i {
			var arr []string
			if json.Unmarshal([]byte(s[i:j+1]), &arr) == nil {
				return arr
			}
		}
	}
	var lines []string
	for _, ln := range strings.Split(s, "\n") {
		ln = strings.TrimSpace(ln)
		ln = strings.TrimLeft(ln, "-*0123456789. \t")
		ln = strings.Trim(ln, "\"")
		if ln != "" {
			lines = append(lines, ln)
		}
	}
	return lines
}
