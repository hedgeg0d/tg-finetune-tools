package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/hedgeg0d/tg-finetune-tools/internal/clean"
	"github.com/hedgeg0d/tg-finetune-tools/internal/config"
	"github.com/hedgeg0d/tg-finetune-tools/internal/dataset"
	"github.com/hedgeg0d/tg-finetune-tools/internal/pipeline"
	"github.com/hedgeg0d/tg-finetune-tools/internal/progress"
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
	dryRun := fs.Bool("dry-run", false, "report stats without writing output")
	noProgress := fs.Bool("no-progress", false, "disable the progress bar")
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

	opts := pipeline.Options{Progress: progressEnabled(*noProgress), DryRun: *dryRun}
	stats, err := pipeline.Clean(*in, *out, cfg, opts)
	if err != nil {
		return err
	}
	printCleanStats(stats)
	if *dryRun {
		fmt.Fprintln(os.Stderr, "  -> dry run, nothing written")
	} else {
		fmt.Fprintf(os.Stderr, "  -> %s\n", *out)
	}
	return nil
}

func runBuild(args []string) error {
	fs := flag.NewFlagSet("build", flag.ExitOnError)
	in := fs.String("in", "clean.jsonl", "normalized jsonl input")
	out := fs.String("out", "dataset.jsonl", "finetune dataset output")
	cfgPath := fs.String("config", "", "config file (json)")
	dryRun := fs.Bool("dry-run", false, "preview sample conversations without writing")
	sampleN := fs.Int("sample", 3, "number of conversations to preview with --dry-run")
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

	warnGenerated(cfg, *dryRun)
	opts := pipeline.Options{DryRun: *dryRun, Sample: *sampleN}
	stats, err := pipeline.Build(*in, *out, cfg, opts)
	if err != nil {
		return err
	}
	if *dryRun {
		printSamples(stats.Samples)
		fmt.Fprintf(os.Stderr, "build: %d messages -> %d conversations%s (dry run)\n", stats.Messages, stats.Conversations, dupNote(stats))
		return nil
	}
	fmt.Fprintf(os.Stderr, "build: %d messages -> %d conversations%s%s\n", stats.Messages, stats.Conversations, dupNote(stats), splitNote(stats, *out, cfg))
	return nil
}

func runAll(args []string) error {
	fs := flag.NewFlagSet("all", flag.ExitOnError)
	in := fs.String("in", "", "telegram export result.json")
	out := fs.String("out", "dataset.jsonl", "finetune dataset output")
	cfgPath := fs.String("config", "", "config file (json)")
	workers := fs.Int("workers", 0, "override worker count")
	dryRun := fs.Bool("dry-run", false, "preview sample conversations without writing")
	sampleN := fs.Int("sample", 3, "number of conversations to preview with --dry-run")
	noProgress := fs.Bool("no-progress", false, "disable the progress bar")
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

	warnGenerated(cfg, *dryRun)
	opts := pipeline.Options{Progress: progressEnabled(*noProgress), DryRun: *dryRun, Sample: *sampleN}
	cs, bs, err := pipeline.All(*in, *out, cfg, opts)
	if err != nil {
		return err
	}
	printCleanStats(cs)
	if *dryRun {
		printSamples(bs.Samples)
		fmt.Fprintf(os.Stderr, "build: %d conversations%s (dry run)\n", bs.Conversations, dupNote(bs))
		return nil
	}
	fmt.Fprintf(os.Stderr, "build: %d conversations%s%s\n", bs.Conversations, dupNote(bs), splitNote(bs, *out, cfg))
	return nil
}

func dupNote(bs pipeline.BuildStats) string {
	if bs.Duplicates == 0 {
		return ""
	}
	return fmt.Sprintf(" (%d duplicates removed)", bs.Duplicates)
}

func printCleanStats(s pipeline.CleanStats) {
	fmt.Fprintf(os.Stderr, "clean:\n  read    %8d\n  kept    %8d\n  dropped %8d\n", s.Read, s.Kept(), s.Dropped())

	var active []clean.ReasonInfo
	for _, r := range clean.DropReasons {
		if s.Reasons[r.Reason] > 0 {
			active = append(active, r)
		}
	}
	for i, r := range active {
		branch := "├─"
		if i == len(active)-1 {
			branch = "└─"
		}
		fmt.Fprintf(os.Stderr, "    %s %-20s %8d\n", branch, r.Label, s.Reasons[r.Reason])
	}
}

func progressEnabled(noProgress bool) bool {
	return !noProgress && progress.IsTerminal(os.Stderr)
}

func printSamples(convs []dataset.Conversation) {
	for i, c := range convs {
		fmt.Fprintf(os.Stderr, "\n─── sample %d ───\n", i+1)
		if c.System != "" {
			fmt.Fprintf(os.Stderr, "\033[33m[system]\033[0m %s\n", truncate(c.System, 240))
		}
		for _, t := range c.Turns {
			fmt.Fprintf(os.Stderr, "%s %s\n", roleTag(t.Role), truncate(t.Content, 240))
		}
	}
	if len(convs) > 0 {
		fmt.Fprintln(os.Stderr)
	}
}

func warnGenerated(cfg config.Config, dryRun bool) {
	if dryRun || cfg.Build.System.Mode != "generated" {
		return
	}
	g := cfg.Build.System.Generated
	fmt.Fprintf(os.Stderr, "\033[33mwarning:\033[0m system.mode=generated will send sampled messages to %s (model %s).\n", g.APIBase, g.Model)
	if !cfg.Clean.RedactPII {
		fmt.Fprintln(os.Stderr, "         consider clean.redact_pii=true; data leaving your machine may be logged by the provider.")
	}
}

func roleTag(role string) string {
	if role == "assistant" {
		return "\033[32m[assistant]\033[0m"
	}
	return "\033[36m[user]\033[0m"
}

func truncate(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max]) + "…"
}

func splitNote(bs pipeline.BuildStats, out string, cfg config.Config) string {
	if cfg.Build.ValRatio <= 0 {
		return fmt.Sprintf(" -> %s", out)
	}
	train, val := pipeline.SplitPaths(out)
	return fmt.Sprintf(" -> train %d (%s), val %d (%s)", bs.Train, train, bs.Val, val)
}

func runInspect(args []string) error {
	fs := flag.NewFlagSet("inspect", flag.ExitOnError)
	in := fs.String("in", "", "telegram export result.json")
	noProgress := fs.Bool("no-progress", false, "disable the progress bar")
	fs.Parse(args)

	if *in == "" {
		return fmt.Errorf("--in is required")
	}

	rep, err := pipeline.Inspect(*in, progressEnabled(*noProgress))
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
