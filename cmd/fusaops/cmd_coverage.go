package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/SoundMatt/FuSaOps/coverage"
)

// runCoverage reads a Go coverage profile and emits a DO-178C structural
// coverage report.
//
//fusa:req REQ-FO-CLI051
func runCoverage(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("fusaops coverage", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() {
		fmt.Fprintf(stderr, "Usage: fusaops coverage [flags] [coverage.out]\n\n")
		fmt.Fprintf(stderr, "Produce a DO-178C structural coverage report from a Go coverage profile.\n")
		fmt.Fprintf(stderr, "Generate a profile with: go test -coverprofile=coverage.out ./...\n\n")
		fmt.Fprintf(stderr, "Flags:\n")
		fs.PrintDefaults()
	}

	var (
		dalFlag = fs.String("dal", "DAL-B", "design assurance level: DAL-A, DAL-B, DAL-C, DAL-D")
		format  = fs.String("format", "text", "output format: text, json, or markdown")
		output  = fs.String("output", "", "write report to file (default: stdout)")
		dir     = fs.String("dir", "", "search for coverage.out in this directory")
	)

	if err := fs.Parse(args); err != nil {
		return 2
	}

	dal := coverage.DAL(*dalFlag)
	switch dal {
	case coverage.DALA, coverage.DALB, coverage.DALC, coverage.DALD:
	default:
		fmt.Fprintf(stderr, "fusaops coverage: invalid --dal %q (want DAL-A, DAL-B, DAL-C, or DAL-D)\n", *dalFlag)
		return 2
	}

	profilePath := coverage.CoverageFile
	if fs.NArg() > 0 {
		profilePath = fs.Arg(0)
	} else if *dir != "" {
		profilePath = filepath.Join(*dir, coverage.CoverageFile)
	} else if _, err := os.Stat(profilePath); err != nil {
		cwd, _ := os.Getwd()
		profilePath = filepath.Join(cwd, coverage.CoverageFile)
	}

	rep, err := coverage.BuildFromFile(profilePath, dal)
	if err != nil {
		fmt.Fprintf(stderr, "fusaops coverage: %v\n", err)
		fmt.Fprintf(stderr, "Tip: generate a profile with: go test -coverprofile=%s ./...\n", coverage.CoverageFile)
		return 1
	}

	w := stdout
	if *output != "" {
		f, err := os.Create(*output)
		if err != nil {
			fmt.Fprintf(stderr, "fusaops coverage: create output: %v\n", err)
			return 1
		}
		defer func() { _ = f.Close() }()
		w = f
	}

	if err := coverage.Render(w, rep, *format); err != nil {
		fmt.Fprintf(stderr, "fusaops coverage: render: %v\n", err)
		return 1
	}
	return 0
}
