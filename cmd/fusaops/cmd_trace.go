package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"time"

	fusaops "github.com/SoundMatt/FuSaOps"
	"github.com/SoundMatt/FuSaOps/orchestrator"
	"github.com/SoundMatt/FuSaOps/trace"
)

// runTrace rolls every applicable tool's requirement traceability and
// qualification up into one cross-language matrix. With --strict it exits 1 when
// any requirement is untraced or untested, making it a polyglot coverage gate.
//
//fusa:req REQ-FO-CLI011
//fusa:req REQ-FO-CLI021
//fusa:req REQ-FO-CLI049
//fusa:req REQ-FO-CLI077
func runTrace(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("fusaops trace", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dir := fs.String("dir", ".", "project root directory")
	only := fs.String("only", "", "comma-separated tool names to run (default: all applicable)")
	format := fs.String("format", "text", "output format: text|json|html|markdown")
	output := fs.String("output", "", "output file (default: stdout)")
	strict := fs.Bool("strict", false, "exit non-zero when any requirement is untraced or untested")
	gaps := fs.Bool("gaps", false, "show only untraced/untested requirements")
	reqCoverage := fs.Int("req-coverage", -1, "fail when traced% < N (0–100)")
	secTested := fs.Int("sec-tested", -1, "fail when sec-tested% < N (0–100)")
	workers := fs.Int("workers", 0, "max parallel adapters (0 = unlimited; overrides config)")
	timeout := fs.String("timeout", "", "per-adapter deadline e.g. 30s, 5m (overrides config)")
	decomp := fs.Bool("decomp", false, "run HLR/LLR requirement decomposition gate")
	decompEnforce := fs.String("decomp-enforce", "", "decomposition enforcement: warn|error|off|auto (overrides config)")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	root, opts, cfg, err := loadOptions(*dir, *only, stderr)
	if err != nil {
		fmt.Fprintf(stderr, "fusaops trace: %v\n", err)
		return 1
	}
	if *workers > 0 {
		opts.Workers = *workers
	}
	if *timeout != "" {
		d, perr := time.ParseDuration(*timeout)
		if perr != nil {
			fmt.Fprintf(stderr, "fusaops trace: --timeout %q: %v\n", *timeout, perr)
			return 2
		}
		opts.Timeout = d
	}

	agg, err := orchestrator.New(nil).RunTrace(context.Background(), root, opts)
	if err != nil {
		if errors.Is(err, fusaops.ErrNoAdapters) {
			fmt.Fprintln(stderr, "fusaops trace: no supported languages detected")
			return 1
		}
		fmt.Fprintf(stderr, "fusaops trace: %v\n", err)
		return 1
	}

	// Merge config-driven decomp settings with flag overrides.
	decompEnabled := *decomp
	enforce := *decompEnforce
	if cfg != nil && cfg.Trace.ReqDecomposition.Enforce != "" {
		if cfg.Trace.ReqDecomposition.Enforce != "off" {
			decompEnabled = true
		}
		if enforce == "" {
			enforce = cfg.Trace.ReqDecomposition.Enforce
		}
	}

	// Run the gate and attach results to the aggregate so renderers can include them.
	if decompEnabled {
		agg.Decomposition = trace.CheckDecomposition(agg)
	}

	renderAgg := agg
	if *gaps {
		renderAgg = trace.FilterGaps(agg)
	}

	if *output == "" {
		if err := trace.Render(stdout, renderAgg, *format); err != nil {
			fmt.Fprintf(stderr, "fusaops trace: %v\n", err)
			return 1
		}
	} else {
		if err := trace.RenderToFile(renderAgg, *format, *output); err != nil {
			fmt.Fprintf(stderr, "fusaops trace: %v\n", err)
			return 1
		}
		fmt.Fprintf(stderr, "Wrote %s traceability matrix to %s\n", *format, *output)
	}

	if *strict && agg.HasGaps() {
		return 1
	}
	if *reqCoverage >= 0 && agg.Coverage.TracedPct < *reqCoverage {
		fmt.Fprintf(stderr, "fusaops trace: traced coverage %d%% < required %d%%\n",
			agg.Coverage.TracedPct, *reqCoverage)
		return 1
	}
	if *secTested >= 0 && agg.Coverage.SecTestedPct < *secTested {
		fmt.Fprintf(stderr, "fusaops trace: sec-tested coverage %d%% < required %d%%\n",
			agg.Coverage.SecTestedPct, *secTested)
		return 1
	}
	if agg.Decomposition != nil && !agg.Decomposition.Valid() {
		dal, asil := "", ""
		if cfg != nil {
			dal = cfg.Project.DAL
			asil = cfg.Project.ASIL
		}
		sev := trace.SeverityForDecomposition(enforce, dal, asil)
		for _, v := range agg.Decomposition.Violations {
			fmt.Fprintf(stderr, "fusaops trace: decomp: %s\n", v)
		}
		if sev == "error" {
			return 1
		}
	}
	return 0
}
