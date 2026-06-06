package persona

import (
	"fmt"
	"math/rand"
	"os"
	"strings"

	"github.com/hedgeg0d/tg-finetune-tools/internal/config"
	"github.com/hedgeg0d/tg-finetune-tools/internal/dataset"
	"github.com/hedgeg0d/tg-finetune-tools/internal/llm"
)

const defaultInstruction = "Below are real messages written by one person in a private chat. " +
	"Write a short system prompt (1-2 sentences) instructing an assistant to reply exactly like this " +
	"person: their tone, manner, and speech style. Return only the prompt, with no explanations."

func Assign(convs []dataset.Conversation, cfg config.Config, offline bool) ([]string, error) {
	sys := cfg.Build.System
	out := make([]string, len(convs))

	switch sys.Mode {
	case "", "empty":
		return out, nil
	case "fixed":
		for i := range out {
			out[i] = sys.Fixed
		}
		return out, nil
	case "pool":
		rng := rand.New(rand.NewSource(cfg.Build.Seed))
		for i := range out {
			out[i] = sys.Pool[rng.Intn(len(sys.Pool))]
		}
		return out, nil
	case "generated":
		return generated(convs, cfg, offline)
	default:
		return out, nil
	}
}

func generated(convs []dataset.Conversation, cfg config.Config, offline bool) ([]string, error) {
	g := cfg.Build.System.Generated
	batches := (len(convs) + g.BatchSize - 1) / g.BatchSize

	prompts, err := loadCache(g.CacheFile)
	if err != nil {
		return nil, err
	}

	if len(prompts) < batches {
		if offline {
			for len(prompts) < batches {
				prompts = append(prompts, "<generated at build time>")
			}
		} else {
			c := llm.New(g)
			for b := len(prompts); b < batches; b++ {
				fmt.Fprintf(os.Stderr, "\r  system prompts: %d/%d   ", b+1, batches)
				p, err := c.Generate(instruction(g), batchContext(convs, b, g))
				if err != nil {
					return nil, fmt.Errorf("generate system prompt for batch %d: %w", b, err)
				}
				prompts = append(prompts, strings.TrimSpace(p))
				if err := saveCache(g.CacheFile, prompts); err != nil {
					return nil, err
				}
			}
			fmt.Fprintln(os.Stderr)
		}
	}

	out := make([]string, len(convs))
	for i := range convs {
		out[i] = prompts[i/g.BatchSize]
	}
	return out, nil
}

func batchContext(convs []dataset.Conversation, batch int, g config.Generated) string {
	start := batch * g.BatchSize
	end := min(start+g.BatchSize, len(convs))

	var msgs []string
	for _, c := range convs[start:end] {
		for _, t := range c.Turns {
			if t.Role == "assistant" {
				msgs = append(msgs, t.Content)
			}
		}
	}

	rng := rand.New(rand.NewSource(int64(batch) + 1))
	rng.Shuffle(len(msgs), func(i, j int) { msgs[i], msgs[j] = msgs[j], msgs[i] })
	if g.ContextSamples > 0 && len(msgs) > g.ContextSamples {
		msgs = msgs[:g.ContextSamples]
	}
	return strings.Join(msgs, "\n")
}

func instruction(g config.Generated) string {
	if strings.TrimSpace(g.Instruction) != "" {
		return g.Instruction
	}
	return defaultInstruction
}
