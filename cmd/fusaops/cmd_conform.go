package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/SoundMatt/FuSaOps/conform"
)

// runConform runs the x-FuSa spec conformance checker against a tool binary.
//
//fusa:req REQ-FO-CLI014
//fusa:req REQ-FO-CNF019
func runConform(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("conform", flag.ContinueOnError)
	fs.SetOutput(stderr)
	format := fs.String("format", "text", "Output format: text|json|html|markdown")
	output := fs.String("output", "", "Write report to file (default: stdout)")
	fs.Usage = func() {
		fmt.Fprint(stderr, `Usage: fusaops conform <binary> [flags]

Run x-FuSa spec v1.8 conformance checks against <binary>.

Flags:
  --format text|json|html|markdown   Output format (default: text)
  --output <file>                    Write report to file (default: stdout)

Exit codes:
  0  all MUST checks passed
  1  one or more MUST checks failed
  2  usage error

`)
	}
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() < 1 {
		fmt.Fprintln(stderr, "fusaops conform: <binary> argument required")
		fs.Usage()
		return 2
	}
	if *format != "text" && *format != "json" && *format != "html" && *format != "markdown" && *format != "md" {
		fmt.Fprintf(stderr, "fusaops conform: unsupported format %q\n", *format)
		return 2
	}

	binary := fs.Arg(0)
	rep, err := conform.Run(binary, conform.Options{})
	if err != nil {
		fmt.Fprintf(stderr, "fusaops conform: %v\n", err)
		return 1
	}

	w := stdout
	if *output != "" {
		f, err := os.Create(*output)
		if err != nil {
			fmt.Fprintf(stderr, "fusaops conform: %v\n", err)
			return 1
		}
		defer f.Close()
		w = f
	}

	if err := conform.Render(w, rep, *format); err != nil {
		fmt.Fprintf(stderr, "fusaops conform: render: %v\n", err)
		return 1
	}

	if *output != "" {
		fmt.Fprintf(stdout, "Wrote %s\n", *output)
	}

	if rep.HasFailures() {
		return 1
	}
	return 0
}
