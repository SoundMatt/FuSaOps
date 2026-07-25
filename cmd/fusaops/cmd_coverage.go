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
// coverage report, or runs the LLVM source-based MC/DC gate when --mcdc is set.
//
//fusa:req REQ-FO-CLI051
//fusa:req REQ-FO-CLI080
func runCoverage(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("fusaops coverage", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() {
		fmt.Fprintf(stderr, "Usage: fusaops coverage [flags] [coverage.out]\n\n")
		fmt.Fprintf(stderr, "Produce a DO-178C structural coverage report from a Go coverage profile.\n")
		fmt.Fprintf(stderr, "Generate a profile with: go test -coverprofile=coverage.out ./...\n\n")
		fmt.Fprintf(stderr, "Flags:\n")
		fs.PrintDefaults()
		fmt.Fprintf(stderr, "\nMC/DC flags (LLVM source-based, DAL-A prerequisite):\n")
		fmt.Fprintf(stderr, "  --mcdc              enable LLVM source-based MC/DC coverage gate\n")
		fmt.Fprintf(stderr, "  --mcdc-file         path to LLVM coverage JSON from llvm-cov export --format=json (default: mcdc.json)\n")
		fmt.Fprintf(stderr, "  --mcdc-threshold    minimum condition coverage percentage for gate pass (default: 100.0)\n")
		fmt.Fprintf(stderr, "  --req-dir           directory to scan for //fusa:req annotated Go functions (default: --dir or cwd)\n")
	}

	var (
		dalFlag       = fs.String("dal", "DAL-B", "design assurance level: DAL-A, DAL-B, DAL-C, DAL-D")
		format        = fs.String("format", "text", "output format: text, json, or markdown")
		output        = fs.String("output", "", "write report to file (default: stdout)")
		dir           = fs.String("dir", "", "search for coverage.out in this directory")
		mcdcFlag      = fs.Bool("mcdc", false, "enable LLVM source-based MC/DC gate (DAL-A prerequisite)")
		mcdcFile      = fs.String("mcdc-file", "mcdc.json", "path to llvm-cov export --format=json output")
		mcdcThreshold = fs.Float64("mcdc-threshold", coverage.DefaultMcdcThreshold, "minimum condition coverage % for gate pass")
		reqDir        = fs.String("req-dir", "", "directory to scan for //fusa:req annotated functions (default: --dir or cwd)")
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

	// Determine output writer (shared by both paths).
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

	// MC/DC path — when --mcdc is set the command produces only the McdcReport.
	if *mcdcFlag {
		return runMCDC(stderr, w, *mcdcFile, *format, *reqDir, *dir, dal, *mcdcThreshold)
	}

	// Standard Go profile path.
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

	if err := coverage.Render(w, rep, *format); err != nil {
		fmt.Fprintf(stderr, "fusaops coverage: render: %v\n", err)
		return 1
	}
	return 0
}

// runMCDC implements the --mcdc branch of fusaops coverage.
//
//fusa:req REQ-FO-CLI080
func runMCDC(stderr, w io.Writer, mcdcFilePath, format, reqDir, dirFlag string, dal coverage.DAL, threshold float64) int {
	// 1. Open the LLVM coverage JSON.
	f, err := os.Open(mcdcFilePath)
	if err != nil {
		fmt.Fprintf(stderr, "fusaops coverage: open mcdc-file %q: %v\n", mcdcFilePath, err)
		return 1
	}
	defer func() { _ = f.Close() }()

	// 2. Parse MC/DC data.
	funcs, err := coverage.ParseMCDC(f)
	if err != nil {
		fmt.Fprintf(stderr, "fusaops coverage: parse mcdc: %v\n", err)
		return 1
	}

	// 3. Determine req-dir.
	rDir := reqDir
	if rDir == "" {
		rDir = dirFlag
	}
	if rDir == "" {
		cwd, _ := os.Getwd()
		rDir = cwd
	}

	// 4. Find annotated functions (non-fatal if scan fails).
	annotated, scanErr := coverage.FindAnnotatedFunctions(rDir)
	if scanErr != nil {
		fmt.Fprintf(stderr, "fusaops coverage: warning: cannot scan req-dir %q: %v\n", rDir, scanErr)
		annotated = map[string]bool{}
	}

	// 5. Analyse.
	mcdcRep := coverage.AnalyseMCDC(funcs, annotated, dal, threshold)

	// 6. Render.
	if err := coverage.RenderMCDC(w, mcdcRep, format); err != nil {
		fmt.Fprintf(stderr, "fusaops coverage: render mcdc: %v\n", err)
		return 2
	}

	// 7. Gate.
	if !coverage.GateMCDC(mcdcRep) {
		uncovCount := len(mcdcRep.UncoveredReqs)
		fmt.Fprintf(stderr, "fusaops coverage: MC/DC gate FAILED: %d uncovered req-annotated condition(s)\n", uncovCount)
		return 1
	}
	return 0
}
