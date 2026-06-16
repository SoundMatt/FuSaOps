package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/SoundMatt/FuSaOps/fmea"
)

// runFMEA generates a Design Failure Mode and Effects Analysis per IEC 61508 / ISO 26262 Part 8.
//
//fusa:req REQ-FO-CLI070
func runFMEA(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("fusaops fmea", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() {
		fmt.Fprintf(stderr, "Usage: fusaops fmea [flags]\n\n")
		fmt.Fprintf(stderr, "Generate a Design Failure Mode and Effects Analysis (dFMEA) per\n")
		fmt.Fprintf(stderr, "IEC 61508:2010 / ISO 26262:2018 Part 8-7.\n\n")
		fmt.Fprintf(stderr, "Analyses 8 failure modes in the FuSaOps orchestration pipeline.\n")
		fmt.Fprintf(stderr, "Each mode has Severity, Occurrence, and Detection ratings (1–10);\n")
		fmt.Fprintf(stderr, "RPN = S × O × D. Items with RPN > %d are flagged as high-priority.\n\n",
			fmea.HighRPNThreshold)
		fmt.Fprintf(stderr, "Exits 1 if any failure mode has a high RPN.\n\n")
		fmt.Fprintf(stderr, "Flags:\n")
		fs.PrintDefaults()
	}

	var (
		dir    = fs.String("dir", "", "project root directory (default: current directory)")
		output = fs.String("output", "", "path for the FMEA report (default: <dir>/.fusaops-fmea.json)")
		format = fs.String("format", "text", "output format: text, json")
	)
	if err := fs.Parse(args); err != nil {
		return 2
	}

	projectRoot := *dir
	if projectRoot == "" {
		var err error
		projectRoot, err = os.Getwd()
		if err != nil {
			fmt.Fprintf(stderr, "fusaops fmea: get working directory: %v\n", err)
			return 1
		}
	}

	f, err := fmea.Build(projectRoot)
	if err != nil {
		fmt.Fprintf(stderr, "fusaops fmea: build: %v\n", err)
		return 1
	}

	if renderErr := fmea.Render(stdout, f, *format); renderErr != nil {
		fmt.Fprintf(stderr, "fusaops fmea: render: %v\n", renderErr)
		return 2
	}

	outPath := *output
	if outPath == "" {
		outPath = filepath.Join(projectRoot, fmea.ReportFile)
	}
	if saveErr := fmea.Save(outPath, f); saveErr != nil {
		fmt.Fprintf(stderr, "fusaops fmea: save: %v\n", saveErr)
		return 1
	}
	fmt.Fprintf(stdout, "\nFMEA written to %s\n", outPath)
	fmt.Fprintf(stdout, "Integrity hash: %s\n", f.Hash)

	if f.HasHighRPN() {
		return 1
	}
	return 0
}
