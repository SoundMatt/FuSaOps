package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"time"

	fusaops "github.com/SoundMatt/FuSaOps"
	"github.com/SoundMatt/FuSaOps/comp"
	"github.com/SoundMatt/FuSaOps/orchestrator"
)

// runComp rolls every applicable tool's cyclomatic complexity (V(G)) analysis up
// into one cross-language complexity report.
//
//fusa:req REQ-FO-CLI082
func runComp(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("fusaops comp", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() {
		fmt.Fprintf(stderr, "Usage: fusaops comp [flags]\n\n")
		fmt.Fprintf(stderr, "Compute McCabe cyclomatic complexity (V(G)) across all languages.\n")
		fmt.Fprintf(stderr, "Delegates to each tool's comp command and rolls up the results.\n")
		fmt.Fprintf(stderr, "Exits 1 when any function exceeds the threshold.\n\n")
		fmt.Fprintf(stderr, "DAL-level thresholds (DO-178C §6.3.4): A≤4  B≤10  C≤15  D≤20\n\n")
		fmt.Fprintf(stderr, "Flags:\n")
		fs.PrintDefaults()
	}

	var (
		dir       = fs.String("dir", ".", "project root directory")
		only      = fs.String("only", "", "comma-separated tool names to run (default: all applicable)")
		format    = fs.String("format", "text", "output format: text|json")
		output    = fs.String("output", "", "output file (default: stdout)")
		threshold = fs.Int("threshold", 0, "complexity threshold override (0 = use tool default, DAL-B = 10)")
		dal       = fs.String("dal", "", "DAL level override: DAL-A|DAL-B|DAL-C|DAL-D (sets threshold)")
		workers   = fs.Int("workers", 0, "max parallel adapters (0 = unlimited)")
		timeout   = fs.String("timeout", "", "per-adapter deadline e.g. 30s, 5m")
	)
	if err := fs.Parse(args); err != nil {
		return 2
	}

	if err := comp.ValidateDAL(*dal); err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}

	t := *threshold
	if *dal != "" && t == 0 {
		t = comp.DALThreshold(*dal)
	}

	root, opts, _, err := loadOptions(*dir, *only, stderr)
	if err != nil {
		fmt.Fprintf(stderr, "fusaops comp: %v\n", err)
		return 1
	}
	if *workers > 0 {
		opts.Workers = *workers
	}
	if *timeout != "" {
		d, perr := time.ParseDuration(*timeout)
		if perr != nil {
			fmt.Fprintf(stderr, "fusaops comp: --timeout %q: %v\n", *timeout, perr)
			return 2
		}
		opts.Timeout = d
	}

	agg, err := orchestrator.New(nil).RunComp(context.Background(), root, opts, t, *dal)
	if err != nil {
		if errors.Is(err, fusaops.ErrNoAdapters) {
			fmt.Fprintln(stderr, "fusaops comp: no supported languages detected")
			return 1
		}
		fmt.Fprintf(stderr, "fusaops comp: %v\n", err)
		return 1
	}

	if *output == "" {
		if err := comp.Render(stdout, agg, *format); err != nil {
			fmt.Fprintf(stderr, "fusaops comp: %v\n", err)
			return 1
		}
	} else {
		if err := comp.RenderToFile(agg, *format, *output); err != nil {
			fmt.Fprintf(stderr, "fusaops comp: %v\n", err)
			return 1
		}
		fmt.Fprintf(stderr, "Wrote %s complexity report to %s\n", *format, *output)
	}

	if agg.HasViolations() {
		return 1
	}
	return 0
}
