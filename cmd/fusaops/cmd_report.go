package main

import (
	"context"
	"flag"
	"fmt"
	"io"

	"github.com/SoundMatt/FuSaOps/orchestrator"
	"github.com/SoundMatt/FuSaOps/report"
)

// runReport generates an aggregate report and writes it to a file (or stdout).
// Unlike check it never fails on findings; it is for evidence generation.
//
//fusa:req REQ-FO-CLI009
//fusa:req REQ-FO-CLI033
//fusa:req REQ-FO-CLI034
//fusa:req REQ-FO-CLI035
//fusa:req REQ-FO-CLI036
func runReport(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("fusaops report", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dir := fs.String("dir", ".", "project root directory")
	only := fs.String("only", "", "comma-separated tool names to run (default: all applicable)")
	format := fs.String("format", "json", "output format: text|json|html|sarif|junit|csv|markdown")
	output := fs.String("output", "", "output file (default: stdout)")
	suppressFile := fs.String("suppress-file", "", "path to .fusaops-suppress.json")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	root, opts, _, err := loadOptions(*dir, *only, stderr)
	if err != nil {
		fmt.Fprintf(stderr, "fusaops report: %v\n", err)
		return 1
	}
	opts.SuppressFile = *suppressFile

	rn := orchestrator.New(nil)
	rep, err := rn.Run(context.Background(), root, opts)
	if err != nil {
		fmt.Fprintf(stderr, "fusaops report: %v\n", err)
		return 1
	}

	if err := report.RenderToFile(rep, *format, *output); err != nil {
		fmt.Fprintf(stderr, "fusaops report: %v\n", err)
		return 1
	}
	if *output != "" {
		fmt.Fprintf(stdout, "Wrote %s report to %s\n", *format, *output)
	}
	return 0
}
