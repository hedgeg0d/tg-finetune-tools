package memory

import (
	"fmt"
	"sort"
	"strings"

	"github.com/hedgeg0d/tg-finetune-tools/internal/config"
	"github.com/hedgeg0d/tg-finetune-tools/internal/llm"
)

const defaultConsolidateInstruction = "Вот список фактов об одном человеке в формате «[дата] факт». " +
	"Объедини дубликаты и близкие по смыслу, удали незначительные бытовые мелочи, " +
	"при противоречии оставь более поздний по дате. " +
	"Верни ТОЛЬКО JSON-массив строк — итоговые уникальные факты на русском, без дат."

func consolidate(facts []fact, mcfg config.Memory) ([]fact, error) {
	client := llm.New(mcfg.Consolidate)
	instr := mcfg.Consolidate.Instruction
	if strings.TrimSpace(instr) == "" {
		instr = defaultConsolidateInstruction
	}
	batchSize := mcfg.Consolidate.BatchSize
	if batchSize < 2 {
		batchSize = 40
	}
	passes := mcfg.ConsolidatePasses
	if passes < 1 {
		passes = 1
	}

	for p := 0; p < passes; p++ {
		if p%2 == 0 {
			sort.SliceStable(facts, func(i, j int) bool { return facts[i].Text < facts[j].Text })
		} else {
			sort.SliceStable(facts, func(i, j int) bool { return facts[i].Date < facts[j].Date })
		}

		var next []fact
		batches := (len(facts) + batchSize - 1) / batchSize
		for b := 0; b < batches; b++ {
			start := b * batchSize
			end := min(start+batchSize, len(facts))
			merged, err := consolidateBatch(client, instr, facts[start:end])
			if err != nil {
				return facts, err
			}
			next = append(next, merged...)
			progress(fmt.Sprintf("consolidating pass %d/%d", p+1, passes), end, len(facts))
		}
		progressDone()

		if len(next) >= len(facts) {
			facts = next
			break
		}
		facts = next
	}
	return facts, nil
}

func consolidateBatch(client *llm.Client, instr string, batch []fact) ([]fact, error) {
	var b strings.Builder
	for _, f := range batch {
		fmt.Fprintf(&b, "[%s] %s\n", dateStr(f.Date), f.Text)
	}
	out, err := client.Generate(instr, b.String())
	if err != nil {
		return nil, err
	}
	latest := batch[0].Date
	for _, f := range batch {
		if f.Date > latest {
			latest = f.Date
		}
	}
	var facts []fact
	for _, t := range parseList(out) {
		t = strings.TrimSpace(t)
		if t != "" {
			facts = append(facts, fact{Text: t, Date: latest})
		}
	}
	return facts, nil
}
