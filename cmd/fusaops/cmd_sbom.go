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
	"github.com/SoundMatt/FuSaOps/sbom"
)

// runSBOM merges every applicable tool's SBOM into one cross-language bill of
// materials, rendered as native JSON, plain text, SPDX 2.3, HTML, or markdown.
//
//fusa:req REQ-FO-CLI012
//fusa:req REQ-FO-SBM010
//fusa:req REQ-FO-SBM011
//fusa:req REQ-FO-CLI049
func runSBOM(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("fusaops sbom", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dir := fs.String("dir", ".", "project root directory")
	only := fs.String("only", "", "comma-separated tool names to run (default: all applicable)")
	format := fs.String("format", "json", "output format: json|text|spdx|html|markdown")
	output := fs.String("output", "", "output file (default: stdout)")
	workers := fs.Int("workers", 0, "max parallel adapters (0 = unlimited; overrides config)")
	timeout := fs.String("timeout", "", "per-adapter deadline e.g. 30s, 5m (overrides config)")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	root, opts, _, err := loadOptions(*dir, *only, stderr)
	if err != nil {
		fmt.Fprintf(stderr, "fusaops sbom: %v\n", err)
		return 1
	}
	if *workers > 0 {
		opts.Workers = *workers
	}
	if *timeout != "" {
		d, perr := time.ParseDuration(*timeout)
		if perr != nil {
			fmt.Fprintf(stderr, "fusaops sbom: --timeout %q: %v\n", *timeout, perr)
			return 2
		}
		opts.Timeout = d
	}

	agg, err := orchestrator.New(nil).RunSBOM(context.Background(), root, opts)
	if err != nil {
		if errors.Is(err, fusaops.ErrNoAdapters) {
			fmt.Fprintln(stderr, "fusaops sbom: no supported languages detected")
			return 1
		}
		fmt.Fprintf(stderr, "fusaops sbom: %v\n", err)
		return 1
	}

	if *output == "" {
		if err := sbom.Render(stdout, agg, *format); err != nil {
			fmt.Fprintf(stderr, "fusaops sbom: %v\n", err)
			return 1
		}
	} else {
		if err := sbom.RenderToFile(agg, *format, *output); err != nil {
			fmt.Fprintf(stderr, "fusaops sbom: %v\n", err)
			return 1
		}
		fmt.Fprintf(stderr, "Wrote %s SBOM (%d packages) to %s\n", *format, agg.TotalPackages, *output)
	}
	return 0
}
