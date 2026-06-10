package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"

	fusaops "github.com/SoundMatt/FuSaOps"
	"github.com/SoundMatt/FuSaOps/orchestrator"
	"github.com/SoundMatt/FuSaOps/trace"
)

// runTrace rolls every applicable tool's requirement traceability and
// qualification up into one cross-language matrix. With --strict it exits 1 when
// any requirement is untraced or untested, making it a polyglot coverage gate.
//
//fusa:req REQ-FO-CLI011
func runTrace(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("fusaops trace", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dir := fs.String("dir", ".", "project root directory")
	only := fs.String("only", "", "comma-separated tool names to run (default: all applicable)")
	format := fs.String("format", "text", "output format: text|json|html")
	output := fs.String("output", "", "output file (default: stdout)")
	strict := fs.Bool("strict", false, "exit non-zero when any requirement is untraced or untested")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	root, opts, _, err := loadOptions(*dir, *only, stderr)
	if err != nil {
		fmt.Fprintf(stderr, "fusaops trace: %v\n", err)
		return 1
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

	if *output == "" {
		if err := trace.Render(stdout, agg, *format); err != nil {
			fmt.Fprintf(stderr, "fusaops trace: %v\n", err)
			return 1
		}
	} else {
		if err := trace.RenderToFile(agg, *format, *output); err != nil {
			fmt.Fprintf(stderr, "fusaops trace: %v\n", err)
			return 1
		}
		fmt.Fprintf(stdout, "Wrote %s traceability matrix to %s\n", *format, *output)
	}
	if *strict && agg.HasGaps() {
		return 1
	}
	return 0
}
