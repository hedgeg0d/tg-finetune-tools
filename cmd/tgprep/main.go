package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/hedgeg0d/tg-finetune-tools/internal/config"
	"github.com/hedgeg0d/tg-finetune-tools/internal/pipeline"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	var err error
	switch cmd {
	case "clean":
		err = runClean(args)
	case "build":
		err = runBuild(args)
	case "all":
		err = runAll(args)
	case "inspect":
		err = runInspect(args)
	case "-h", "--help", "help":
		usage()
		return
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n", cmd)
		usage()
		os.Exit(2)
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func runClean(args []string) error {
	fs := flag.NewFlagSet("clean", flag.ExitOnError)
	in := fs.String("in", "", "telegram export result.json")
	out := fs.String("out", "clean.jsonl", "output normalized jsonl")
	cfgPath := fs.String("config", "", "config file (json)")
	workers := fs.Int("workers", 0, "override worker count")
	ov := registerOverrides(fs)
	fs.Parse(args)

	cfg, err := loadConfig(*cfgPath, *workers)
	if err != nil {
		return err
	}
	ov.apply(fs, &cfg)
	if err := requireRoles(cfg); err != nil {
		return err
	}
	if *in == "" {
		return fmt.Errorf("--in is required")
	}

	stats, err := pipeline.Clean(*in, *out, cfg)
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "clean: read %d, kept %d -> %s\n", stats.Read, stats.Kept, *out)
	return nil
}

func runBuild(args []string) error {
	fs := flag.NewFlagSet("build", flag.ExitOnError)
	in := fs.String("in", "clean.jsonl", "normalized jsonl input")
	out := fs.String("out", "dataset.jsonl", "finetune dataset output")
	cfgPath := fs.String("config", "", "config file (json)")
	ov := registerOverrides(fs)
	fs.Parse(args)

	cfg, err := loadConfig(*cfgPath, 0)
	if err != nil {
		return err
	}
	ov.apply(fs, &cfg)
	if err := cfg.Validate(); err != nil {
		return err
	}

	stats, err := pipeline.Build(*in, *out, cfg)
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "build: %d messages -> %d conversations -> %s\n", stats.Messages, stats.Conversations, *out)
	return nil
}

func runAll(args []string) error {
	fs := flag.NewFlagSet("all", flag.ExitOnError)
	in := fs.String("in", "", "telegram export result.json")
	out := fs.String("out", "dataset.jsonl", "finetune dataset output")
	cfgPath := fs.String("config", "", "config file (json)")
	workers := fs.Int("workers", 0, "override worker count")
	ov := registerOverrides(fs)
	fs.Parse(args)

	cfg, err := loadConfig(*cfgPath, *workers)
	if err != nil {
		return err
	}
	ov.apply(fs, &cfg)
	if err := cfg.Validate(); err != nil {
		return err
	}
	if *in == "" {
		return fmt.Errorf("--in is required")
	}

	cs, bs, err := pipeline.All(*in, *out, cfg)
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "all: read %d, kept %d -> %d conversations -> %s\n", cs.Read, cs.Kept, bs.Conversations, *out)
	return nil
}

func runInspect(args []string) error {
	fs := flag.NewFlagSet("inspect", flag.ExitOnError)
	in := fs.String("in", "", "telegram export result.json")
	fs.Parse(args)

	if *in == "" {
		return fmt.Errorf("--in is required")
	}

	rep, err := pipeline.Inspect(*in)
	if err != nil {
		return err
	}

	fmt.Printf("total messages: %d\n\nparticipants (use from_id for config roles):\n", rep.Total)
	for _, p := range rep.Participants {
		fmt.Printf("  %-22s %-24s %d\n", p.FromID, p.Name, p.Count)
	}
	fmt.Println("\nmedia types (dropped unless captioned / sticker->emoji):")
	for k, v := range rep.Media {
		fmt.Printf("  %-16s %d\n", k, v)
	}
	return nil
}

func loadConfig(path string, workers int) (config.Config, error) {
	cfg, err := config.Load(path)
	if err != nil {
		return cfg, err
	}
	if workers > 0 {
		cfg.Workers = workers
	}
	return cfg, nil
}

func requireRoles(cfg config.Config) error {
	if cfg.Roles.AssistantID == "" || cfg.Roles.UserID == "" {
		return fmt.Errorf("roles.assistant_id and roles.user_id must be set (run `tgprep inspect` to find them)")
	}
	return nil
}

func usage() {
	fmt.Fprint(os.Stderr, `tgprep — prepare Telegram exports for LLM finetuning

usage:
  tgprep inspect --in result.json
  tgprep clean   --in result.json --out clean.jsonl   [--config c.json] [--workers N]
  tgprep build   --in clean.jsonl --out dataset.jsonl [--config c.json]
  tgprep all     --in result.json --out dataset.jsonl [--config c.json] [--workers N]
`)
}
