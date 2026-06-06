package memory

import (
	"encoding/json"
	"regexp"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/hedgeg0d/tg-finetune-tools/internal/config"
	"github.com/hedgeg0d/tg-finetune-tools/internal/llm"
	"github.com/hedgeg0d/tg-finetune-tools/internal/model"
)

const defaultExtractInstruction = "Дан фрагмент переписки двух людей; реплики помечены `assistant:` и `user:`. " +
	"Извлеки устойчивые факты ТОЛЬКО про человека `assistant` (её жизнь, работа, учёба, здоровье, " +
	"отношения, вкусы, привычки, взгляды). НЕ включай факты про `user`, не пересказывай отдельные " +
	"реплики и эмоции момента, не выдумывай. Каждый факт — короткое утверждение в третьем лице. " +
	"Ответь СТРОГО валидным JSON-массивом строк в двойных кавычках, например: " +
	"[\"любит кошек\",\"учится на юриста\"]. Если устойчивых фактов нет — верни []."

type extractCache struct {
	Processed int    `json:"processed"`
	Facts     []fact `json:"facts"`
}

func extractAll(wins [][]model.Message, roles config.Roles, mcfg config.Memory) ([]fact, error) {
	cache, _ := loadJSON[extractCache](mcfg.CacheFile)
	start := cache.Processed

	seen := map[string]bool{}
	var facts []fact
	for _, f := range cache.Facts {
		add(&facts, seen, f)
	}

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
			for _, f := range results[i] {
				add(&facts, seen, f)
			}
		}
		progress("extracting facts", end, len(wins))
		saveJSON(mcfg.CacheFile, extractCache{Processed: end, Facts: facts})
	}
	progressDone()
	return facts, nil
}

func add(facts *[]fact, seen map[string]bool, f fact) bool {
	key := strings.ToLower(strings.TrimSpace(f.Text))
	if key == "" || seen[key] {
		return false
	}
	seen[key] = true
	*facts = append(*facts, f)
	return true
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

var quotedRe = regexp.MustCompile(`"([^"\\]*(?:\\.[^"\\]*)*)"`)
var letterRe = regexp.MustCompile(`\p{L}`)

func parseList(s string) []string {
	s = stripFences(s)
	s = normalizeQuotes(s)

	if i := strings.IndexByte(s, '['); i >= 0 {
		if j := strings.LastIndexByte(s, ']'); j > i {
			var arr []string
			if json.Unmarshal([]byte(s[i:j+1]), &arr) == nil {
				return cleanFacts(arr)
			}
		}
	}

	var found []string
	for _, m := range quotedRe.FindAllStringSubmatch(s, -1) {
		found = append(found, m[1])
	}
	if len(found) > 0 {
		return cleanFacts(found)
	}

	var lines []string
	for _, ln := range strings.Split(s, "\n") {
		lines = append(lines, ln)
	}
	return cleanFacts(lines)
}

func stripFences(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```") {
		if i := strings.IndexByte(s, '\n'); i >= 0 {
			s = s[i+1:]
		}
		s = strings.TrimSuffix(strings.TrimSpace(s), "```")
	}
	return s
}

func normalizeQuotes(s string) string {
	r := strings.NewReplacer("«", "\"", "»", "\"", "„", "\"", "“", "\"", "”", "\"")
	return r.Replace(s)
}

func cleanFacts(in []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, t := range in {
		t = strings.TrimSpace(t)
		t = strings.TrimLeft(t, "-*•0123456789. \t")
		t = strings.Trim(t, "\"',.;:[]() \t")
		t = strings.TrimSpace(t)
		if utf8.RuneCountInString(t) < 8 || !letterRe.MatchString(t) {
			continue
		}
		key := strings.ToLower(t)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, t)
	}
	return out
}
