package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/SoundMatt/FuSaOps/adapter"
	"github.com/SoundMatt/FuSaOps/sci"
)

// runSCI generates the Software Configuration Index per DO-178C §11.16.
//
//fusa:req REQ-FO-CLI067
func runSCI(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("fusaops sci", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() {
		fmt.Fprintf(stderr, "Usage: fusaops sci [flags]\n\n")
		fmt.Fprintf(stderr, "Generate the Software Configuration Index (SCI) per DO-178C §11.16.\n\n")
		fmt.Fprintf(stderr, "The SCI lists all software configuration items — tools, evidence artefacts,\n")
		fmt.Fprintf(stderr, "and language components — with SHA-256 hashes and availability status.\n\n")
		fmt.Fprintf(stderr, "Flags:\n")
		fs.PrintDefaults()
	}

	var (
		dir    = fs.String("dir", "", "project root directory (default: current directory)")
		output = fs.String("output", "", "path for the SCI report (default: <dir>/.fusaops-sci.json)")
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
			fmt.Fprintf(stderr, "fusaops sci: get working directory: %v\n", err)
			return 1
		}
	}

	// Detect applicable adapters (availability is best-effort).
	adapters, err := adapter.Default.Applicable(projectRoot)
	if err != nil {
		fmt.Fprintf(stderr, "fusaops sci: detect adapters: %v\n", err)
		return 1
	}

	s, buildErr := sci.Build(projectRoot, adapters)
	if buildErr != nil {
		fmt.Fprintf(stderr, "fusaops sci: build: %v\n", buildErr)
		return 1
	}

	if renderErr := sci.Render(stdout, s, *format); renderErr != nil {
		fmt.Fprintf(stderr, "fusaops sci: render: %v\n", renderErr)
		return 2
	}

	outPath := *output
	if outPath == "" {
		outPath = filepath.Join(projectRoot, sci.ReportFile)
	}
	if saveErr := sci.Save(outPath, s); saveErr != nil {
		fmt.Fprintf(stderr, "fusaops sci: save: %v\n", saveErr)
		return 1
	}
	fmt.Fprintf(stdout, "\nSCI written to %s (%d items)\n", outPath, s.TotalItems)
	fmt.Fprintf(stdout, "Integrity hash: %s\n", s.Hash)
	return 0
}
