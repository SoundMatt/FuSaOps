package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"time"

	fusaops "github.com/SoundMatt/FuSaOps"
	"github.com/SoundMatt/FuSaOps/diff"
	"github.com/SoundMatt/FuSaOps/orchestrator"
	"github.com/SoundMatt/FuSaOps/report"
)

// runCheck runs every applicable x-FuSa tool and prints the aggregate report.
// It exits 1 when any component produces an ERROR-severity finding, making it
// a CI gate for mixed-language repositories.
//
//fusa:req REQ-FO-CLI008
//fusa:req REQ-FO-CLI033
//fusa:req REQ-FO-CLI034
//fusa:req REQ-FO-CLI035
//fusa:req REQ-FO-CLI036
//fusa:req REQ-FO-CLI037
//fusa:req REQ-FO-CLI040
//fusa:req REQ-FO-CLI041
//fusa:req REQ-FO-CLI042
//fusa:req REQ-FO-CLI047
//fusa:req REQ-FO-CLI049
func runCheck(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("fusaops check", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dir := fs.String("dir", ".", "project root directory")
	only := fs.String("only", "", "comma-separated tool names to run (default: all applicable)")
	format := fs.String("format", "text", "output format: text|json|html|sarif|junit|csv|markdown")
	output := fs.String("output", "", "write report to file instead of stdout")
	strict := fs.Bool("strict", false, "exit non-zero on WARNING findings too")
	suppressFile := fs.String("suppress-file", "", "path to .fusaops-suppress.json")
	showSuppressed := fs.Bool("show-suppressed", false, "include suppressed findings in output")
	showFingerprints := fs.Bool("show-fingerprints", false, "show fingerprint and suppress scaffold per finding")
	saveBaseline := fs.String("save-baseline", "", "save current findings as a diff baseline to this file after check")
	minSeverity := fs.String("min-severity", "", "hide findings below this severity: INFO, WARNING, or ERROR")
	workers := fs.Int("workers", 0, "max parallel adapters (0 = unlimited; overrides config)")
	timeout := fs.String("timeout", "", "per-adapter deadline e.g. 30s, 5m (overrides config)")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	root, opts, cfg, err := loadOptions(*dir, *only, stderr)
	if err != nil {
		fmt.Fprintf(stderr, "fusaops check: %v\n", err)
		return 1
	}
	opts.SuppressFile = *suppressFile
	if *workers > 0 {
		opts.Workers = *workers
	}
	if *timeout != "" {
		d, perr := time.ParseDuration(*timeout)
		if perr != nil {
			fmt.Fprintf(stderr, "fusaops check: --timeout %q: %v\n", *timeout, perr)
			return 2
		}
		opts.Timeout = d
	}
	if *minSeverity != "" {
		sev := fusaops.Severity(*minSeverity)
		if sev.Rank() == 0 {
			fmt.Fprintf(stderr, "fusaops check: --min-severity must be INFO, WARNING, or ERROR\n")
			return 2
		}
		opts.MinSeverity = sev
	}

	rn := orchestrator.New(nil)
	rep, err := rn.Run(context.Background(), root, opts)
	if err != nil {
		if errors.Is(err, fusaops.ErrNoAdapters) {
			fmt.Fprintln(stderr, "fusaops check: no supported languages detected")
			return 1
		}
		fmt.Fprintf(stderr, "fusaops check: %v\n", err)
		return 1
	}

	applyIntegrityLevel(rep, cfg)
	renderOpts := report.RenderOptions{ShowSuppressed: *showSuppressed, ShowFingerprints: *showFingerprints}
	if *output != "" {
		if err := report.RenderToFileWithOptions(rep, *format, *output, renderOpts); err != nil {
			fmt.Fprintf(stderr, "fusaops check: %v\n", err)
			return 1
		}
		fmt.Fprintf(stdout, "Wrote %s report to %s\n", *format, *output)
	} else {
		if err := report.RenderWithOptions(stdout, rep, *format, renderOpts); err != nil {
			fmt.Fprintf(stderr, "fusaops check: %v\n", err)
			return 1
		}
	}

	if *saveBaseline != "" {
		var all []fusaops.Finding
		for _, c := range rep.Components {
			all = append(all, c.Findings...)
		}
		if err := diff.SaveBaseline(*saveBaseline, all); err != nil {
			fmt.Fprintf(stderr, "fusaops check: save-baseline: %v\n", err)
			return 1
		}
		fmt.Fprintf(stdout, "Saved baseline to %s (%d findings)\n", *saveBaseline, len(all))
	}

	if rep.HasErrors() || (*strict && rep.Summary.Warnings > 0) {
		return 1
	}
	return 0
}
