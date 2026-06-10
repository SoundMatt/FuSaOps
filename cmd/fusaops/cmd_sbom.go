package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"

	fusaops "github.com/SoundMatt/FuSaOps"
	"github.com/SoundMatt/FuSaOps/orchestrator"
	"github.com/SoundMatt/FuSaOps/sbom"
)

// runSBOM merges every applicable tool's SBOM into one cross-language bill of
// materials, rendered as native JSON, plain text, or an SPDX 2.3 document.
//
//fusa:req REQ-FO-CLI012
func runSBOM(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("fusaops sbom", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dir := fs.String("dir", ".", "project root directory")
	only := fs.String("only", "", "comma-separated tool names to run (default: all applicable)")
	format := fs.String("format", "json", "output format: json|text|spdx")
	output := fs.String("output", "", "output file (default: stdout)")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	root, opts, _, err := loadOptions(*dir, *only, stderr)
	if err != nil {
		fmt.Fprintf(stderr, "fusaops sbom: %v\n", err)
		return 1
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
		fmt.Fprintf(stdout, "Wrote %s SBOM (%d packages) to %s\n", *format, agg.TotalPackages, *output)
	}
	return 0
}
