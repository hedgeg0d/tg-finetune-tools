package pipeline

import (
	"io"
	"math/rand"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"unicode/utf8"

	"github.com/hedgeg0d/tg-finetune-tools/internal/clean"
	"github.com/hedgeg0d/tg-finetune-tools/internal/config"
	"github.com/hedgeg0d/tg-finetune-tools/internal/dataset"
	"github.com/hedgeg0d/tg-finetune-tools/internal/model"
	"github.com/hedgeg0d/tg-finetune-tools/internal/progress"
	"github.com/hedgeg0d/tg-finetune-tools/internal/telegram"
	"github.com/hedgeg0d/tg-finetune-tools/internal/tokenize"
)

type CleanStats struct {
	Read    int64
	Reasons [clean.NumReasons]int64
}

func (s CleanStats) Kept() int64 {
	return s.Reasons[clean.Kept]
}

func (s CleanStats) Dropped() int64 {
	return s.Read - s.Kept()
}

type BuildStats struct {
	Messages      int
	Conversations int
	Duplicates    int
	Train         int
	Val           int
	Samples       []dataset.Conversation
}

type Options struct {
	Progress bool
	DryRun   bool
	Sample   int
}

func Clean(inPath, outPath string, cfg config.Config, opts Options) (CleanStats, error) {
	in, err := os.Open(inPath)
	if err != nil {
		return CleanStats{}, err
	}
	defer in.Close()

	var w *dataset.Writer
	if !opts.DryRun {
		out, err := os.Create(outPath)
		if err != nil {
			return CleanStats{}, err
		}
		defer out.Close()
		w = dataset.NewWriter(out)
	}

	var mu sync.Mutex
	var stats CleanStats

	sink := func(m model.Message) error {
		if w == nil {
			return nil
		}
		mu.Lock()
		defer mu.Unlock()
		return w.Write(m)
	}

	if err := normalizeStream(in, cfg, &stats, sink, opts); err != nil {
		return stats, err
	}
	if w == nil {
		return stats, nil
	}
	return stats, w.Flush()
}

func Build(inPath, outPath string, cfg config.Config, opts Options) (BuildStats, error) {
	in, err := os.Open(inPath)
	if err != nil {
		return BuildStats{}, err
	}
	defer in.Close()

	msgs, err := model.ReadAll(in)
	if err != nil {
		return BuildStats{}, err
	}

	return writeConversations(outPath, msgs, cfg, opts)
}

func All(inPath, outPath string, cfg config.Config, opts Options) (CleanStats, BuildStats, error) {
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
		return nil
	}

	if err := normalizeStream(in, cfg, &stats, sink, opts); err != nil {
		return stats, BuildStats{}, err
	}

	bs, err := writeConversations(outPath, msgs, cfg, opts)
	return stats, bs, err
}

func normalizeStream(in *os.File, cfg config.Config, stats *CleanStats, sink func(model.Message) error, opts Options) error {
	jobs := make(chan telegram.RawMessage, cfg.Workers*64)
	errc := make(chan error, cfg.Workers+1)
	var wg sync.WaitGroup

	for range cfg.Workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for raw := range jobs {
				m, reason := clean.Normalize(raw, cfg)
				atomic.AddInt64(&stats.Reasons[reason], 1)
				if reason != clean.Kept {
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

	var src io.Reader = in
	if opts.Progress {
		if fi, err := in.Stat(); err == nil && fi.Size() > 0 {
			pr := progress.NewReader(in)
			stop := progress.Track("reading", fi.Size(), pr.Bytes)
			defer stop()
			src = pr
		}
	}

	produceErr := telegram.Stream(src, func(raw telegram.RawMessage) error {
		atomic.AddInt64(&stats.Read, 1)
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

func writeConversations(outPath string, msgs []model.Message, cfg config.Config, opts Options) (BuildStats, error) {
	measure, err := measureFor(cfg)
	if err != nil {
		return BuildStats{}, err
	}

	convs := dataset.Build(msgs, cfg, measure)
	duplicates := 0
	if cfg.Build.Dedup {
		before := len(convs)
		convs = dataset.Dedup(convs)
		duplicates = before - len(convs)
	}
	stats := BuildStats{Messages: len(msgs), Conversations: len(convs), Duplicates: duplicates}

	if opts.DryRun {
		stats.Samples = sample(convs, opts.Sample)
		return stats, nil
	}

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

func sample(convs []dataset.Conversation, n int) []dataset.Conversation {
	if n <= 0 || n > len(convs) {
		n = len(convs)
	}
	out := make([]dataset.Conversation, n)
	copy(out, convs[:n])
	return out
}

func measureFor(cfg config.Config) (dataset.Measure, error) {
	if cfg.Build.MaxTokens <= 0 {
		return utf8.RuneCountInString, nil
	}
	counter, err := tokenize.New(cfg.Build.TokenEncoding)
	if err != nil {
		return nil, err
	}
	return counter.Count, nil
}

func SplitPaths(out string) (train, val string) {
	ext := ""
	if i := strings.LastIndex(out, "."); i >= 0 {
		ext = out[i:]
		out = out[:i]
	}
	return out + ".train" + ext, out + ".val" + ext
}
