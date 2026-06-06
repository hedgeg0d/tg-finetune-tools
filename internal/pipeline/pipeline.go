package pipeline

import (
	"math/rand"
	"os"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/hedgeg0d/tg-finetune-tools/internal/clean"
	"github.com/hedgeg0d/tg-finetune-tools/internal/config"
	"github.com/hedgeg0d/tg-finetune-tools/internal/dataset"
	"github.com/hedgeg0d/tg-finetune-tools/internal/model"
	"github.com/hedgeg0d/tg-finetune-tools/internal/telegram"
)

type CleanStats struct {
	Read int64
	Kept int64
}

type BuildStats struct {
	Messages      int
	Conversations int
	Train         int
	Val           int
}

func Clean(inPath, outPath string, cfg config.Config) (CleanStats, error) {
	in, err := os.Open(inPath)
	if err != nil {
		return CleanStats{}, err
	}
	defer in.Close()

	out, err := os.Create(outPath)
	if err != nil {
		return CleanStats{}, err
	}
	defer out.Close()

	w := dataset.NewWriter(out)
	var mu sync.Mutex
	var stats CleanStats

	sink := func(m model.Message) error {
		mu.Lock()
		defer mu.Unlock()
		atomic.AddInt64(&stats.Kept, 1)
		return w.Write(m)
	}

	if err := normalizeStream(in, cfg, &stats.Read, sink); err != nil {
		return stats, err
	}
	return stats, w.Flush()
}

func Build(inPath, outPath string, cfg config.Config) (BuildStats, error) {
	in, err := os.Open(inPath)
	if err != nil {
		return BuildStats{}, err
	}
	defer in.Close()

	msgs, err := model.ReadAll(in)
	if err != nil {
		return BuildStats{}, err
	}

	return writeConversations(outPath, msgs, cfg)
}

func All(inPath, outPath string, cfg config.Config) (CleanStats, BuildStats, error) {
	in, err := os.Open(inPath)
	if err != nil {
		return CleanStats{}, BuildStats{}, err
	}
	defer in.Close()

	var stats CleanStats
	var mu sync.Mutex
	var msgs []model.Message

	sink := func(m model.Message) error {
		mu.Lock()
		msgs = append(msgs, m)
		mu.Unlock()
		atomic.AddInt64(&stats.Kept, 1)
		return nil
	}

	if err := normalizeStream(in, cfg, &stats.Read, sink); err != nil {
		return stats, BuildStats{}, err
	}

	bs, err := writeConversations(outPath, msgs, cfg)
	return stats, bs, err
}

func normalizeStream(in *os.File, cfg config.Config, read *int64, sink func(model.Message) error) error {
	jobs := make(chan telegram.RawMessage, cfg.Workers*64)
	errc := make(chan error, cfg.Workers+1)
	var wg sync.WaitGroup

	for range cfg.Workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for raw := range jobs {
				m, ok := clean.Normalize(raw, cfg)
				if !ok {
					continue
				}
				if err := sink(m); err != nil {
					select {
					case errc <- err:
					default:
					}
					return
				}
			}
		}()
	}

	produceErr := telegram.Stream(in, func(raw telegram.RawMessage) error {
		atomic.AddInt64(read, 1)
		jobs <- raw
		return nil
	})

	close(jobs)
	wg.Wait()

	if produceErr != nil {
		return produceErr
	}
	select {
	case err := <-errc:
		return err
	default:
		return nil
	}
}

func writeConversations(outPath string, msgs []model.Message, cfg config.Config) (BuildStats, error) {
	convs := dataset.Build(msgs, cfg)
	stats := BuildStats{Messages: len(msgs), Conversations: len(convs)}

	if cfg.Build.ValRatio <= 0 {
		if err := writeSplit(outPath, convs, cfg); err != nil {
			return stats, err
		}
		stats.Train = len(convs)
		return stats, nil
	}

	rng := rand.New(rand.NewSource(cfg.Build.Seed))
	rng.Shuffle(len(convs), func(i, j int) { convs[i], convs[j] = convs[j], convs[i] })

	valN := int(float64(len(convs)) * cfg.Build.ValRatio)
	val, train := convs[:valN], convs[valN:]

	trainPath, valPath := SplitPaths(outPath)
	if err := writeSplit(trainPath, train, cfg); err != nil {
		return stats, err
	}
	if err := writeSplit(valPath, val, cfg); err != nil {
		return stats, err
	}

	stats.Train, stats.Val = len(train), len(val)
	return stats, nil
}

func writeSplit(path string, convs []dataset.Conversation, cfg config.Config) error {
	out, err := os.Create(path)
	if err != nil {
		return err
	}
	defer out.Close()

	w := dataset.NewWriter(out)
	for _, c := range convs {
		if err := w.Write(dataset.Encode(c, cfg)); err != nil {
			return err
		}
	}
	return w.Flush()
}

func SplitPaths(out string) (train, val string) {
	ext := ""
	if i := strings.LastIndex(out, "."); i >= 0 {
		ext = out[i:]
		out = out[:i]
	}
	return out + ".train" + ext, out + ".val" + ext
}
