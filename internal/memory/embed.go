package memory

import (
	"github.com/hedgeg0d/tg-finetune-tools/internal/config"
	"github.com/hedgeg0d/tg-finetune-tools/internal/llm"
)

func embedAll(records []Record, mcfg config.Memory) error {
	client := llm.New(mcfg.Embeddings)
	for i := range records {
		vec, err := client.Embed(records[i].Text)
		if err != nil {
			return err
		}
		records[i].Embedding = vec
		progress("embeddings", i+1, len(records))
	}
	progressDone()
	return nil
}
