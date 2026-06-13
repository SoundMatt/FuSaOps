package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"

	fusaops "github.com/SoundMatt/FuSaOps"
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
func runCheck(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("fusaops check", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dir := fs.String("dir", ".", "project root directory")
	only := fs.String("only", "", "comma-separated tool names to run (default: all applicable)")
	format := fs.String("format", "text", "output format: text|json|html|sarif|junit|csv|markdown")
	output := fs.String("output", "", "write report to file instead of stdout")
	strict := fs.Bool("strict", false, "exit non-zero on WARNING findings too")
	suppressFile := fs.String("suppress-file", "", "path to .fusaops-suppress.json")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	root, opts, _, err := loadOptions(*dir, *only, stderr)
	if err != nil {
		fmt.Fprintf(stderr, "fusaops check: %v\n", err)
		return 1
	}
	opts.SuppressFile = *suppressFile

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

	if *output != "" {
		if err := report.RenderToFile(rep, *format, *output); err != nil {
			fmt.Fprintf(stderr, "fusaops check: %v\n", err)
			return 1
		}
		fmt.Fprintf(stdout, "Wrote %s report to %s\n", *format, *output)
	} else {
		if err := report.Render(stdout, rep, *format); err != nil {
			fmt.Fprintf(stderr, "fusaops check: %v\n", err)
			return 1
		}
	}

	if rep.HasErrors() || (*strict && rep.Summary.Warnings > 0) {
		return 1
	}
	return 0
}
