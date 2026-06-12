package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	fusaops "github.com/SoundMatt/FuSaOps"
	"github.com/SoundMatt/FuSaOps/diff"
	"github.com/SoundMatt/FuSaOps/orchestrator"
)

// runDiff implements the fusaops diff command.
//
// fusaops diff compares a baseline check-report with the findings from a fresh
// scan of the project directory. It exits 0 when no new error-severity findings
// are introduced; it exits 1 when new errors appear (or any new finding under
// --strict). This makes it a CI gate: store a passing baseline, then fail builds
// that introduce regressions.
//
//fusa:req REQ-FO-CLI018
func runDiff(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("diff", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dir := fs.String("dir", ".", "project root to scan")
	baseline := fs.String("baseline", "check-report.json", "path to baseline check-report.json (relative to --dir if not absolute)")
	format := fs.String("format", "text", "output format: text | json")
	output := fs.String("output", "", "write output to this file (default: stdout)")
	strict := fs.Bool("strict", false, "fail on any new finding, not just new errors")
	only := fs.String("only", "", "comma-separated list of adapters to run")
	updateBaseline := fs.Bool("update-baseline", false, "overwrite --baseline with the current run's findings after comparing")

	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 1
	}

	root, opts, _, err := loadOptions(*dir, *only, stderr)
	if err != nil {
		fmt.Fprintf(stderr, "fusaops diff: %v\n", err)
		return 1
	}

	baselinePath := *baseline
	if !filepath.IsAbs(baselinePath) {
		baselinePath = filepath.Join(root, baselinePath)
	}

	bl, err := diff.LoadBaseline(baselinePath)
	if err != nil {
		fmt.Fprintf(stderr, "fusaops diff: %v\n", err)
		return 1
	}

	rn := orchestrator.New(nil)
	rep, err := rn.Run(context.Background(), root, opts)
	if err != nil {
		if errors.Is(err, fusaops.ErrNoAdapters) {
			fmt.Fprintln(stderr, "fusaops diff: no supported languages detected")
			return 1
		}
		fmt.Fprintf(stderr, "fusaops diff: scan failed: %v\n", err)
		return 1
	}

	var current []fusaops.Finding
	for _, c := range rep.Components {
		current = append(current, c.Findings...)
	}

	result := diff.Compare(bl, current)

	if *updateBaseline {
		if err := diff.SaveBaseline(baselinePath, current); err != nil {
			fmt.Fprintf(stderr, "fusaops diff: update-baseline: %v\n", err)
			return 1
		}
	}

	w := io.Writer(stdout)
	if *output != "" {
		f, ferr := os.Create(*output)
		if ferr != nil {
			fmt.Fprintf(stderr, "fusaops diff: create output: %v\n", ferr)
			return 1
		}
		defer func() { _ = f.Close() }()
		w = f
	}

	if err := diff.Render(w, result, *format, *strict); err != nil {
		fmt.Fprintf(stderr, "fusaops diff: render: %v\n", err)
		return 1
	}

	if *strict && result.HasNewFindings() {
		return 1
	}
	if result.HasNewErrors() {
		return 1
	}
	return 0
}
