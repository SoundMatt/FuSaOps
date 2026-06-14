package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	fusaops "github.com/SoundMatt/FuSaOps"
	"github.com/SoundMatt/FuSaOps/badge"
	"github.com/SoundMatt/FuSaOps/report"
)

// runBadge generates an SVG status badge from an aggregate check report.
//
//fusa:req REQ-FO-CLI056
func runBadge(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("fusaops badge", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() {
		fmt.Fprintf(stderr, "Usage: fusaops badge [flags] [report.json]\n\n")
		fmt.Fprintf(stderr, "Generate an SVG status badge from a fusaops check --format json report.\n")
		fmt.Fprintf(stderr, "Reads from a file argument, or from stdin if no file is given.\n\n")
		fmt.Fprintf(stderr, "Flags:\n")
		fs.PrintDefaults()
	}
	output := fs.String("output", "", "write SVG to file (default: stdout)")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	var rep report.AggregateReport
	switch fs.NArg() {
	case 0:
		if err := json.NewDecoder(os.Stdin).Decode(&rep); err != nil {
			fmt.Fprintf(stderr, "fusaops badge: read stdin: %v\n", err)
			return 1
		}
	case 1:
		data, err := os.ReadFile(fs.Arg(0))
		if err != nil {
			fmt.Fprintf(stderr, "fusaops badge: %v\n", err)
			return 1
		}
		if err := json.Unmarshal(data, &rep); err != nil {
			fmt.Fprintf(stderr, "fusaops badge: parse report: %v\n", err)
			return 1
		}
	default:
		fs.Usage()
		return 2
	}

	b := badge.New(rep.Summary.Errors, rep.Summary.Warnings, fusaops.Version)

	w := stdout
	if *output != "" {
		f, err := os.Create(*output)
		if err != nil {
			fmt.Fprintf(stderr, "fusaops badge: create output: %v\n", err)
			return 1
		}
		defer func() { _ = f.Close() }()
		w = f
	}
	if err := badge.Render(w, b); err != nil {
		fmt.Fprintf(stderr, "fusaops badge: render: %v\n", err)
		return 1
	}
	return 0
}
