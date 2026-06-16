package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/SoundMatt/FuSaOps/sas"
)

// runSAS generates the Software Accomplishment Summary per DO-178C §11.20.
//
//fusa:req REQ-FO-CLI068
func runSAS(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("fusaops sas", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() {
		fmt.Fprintf(stderr, "Usage: fusaops sas [flags]\n\n")
		fmt.Fprintf(stderr, "Generate the Software Accomplishment Summary (SAS) per DO-178C §11.20.\n\n")
		fmt.Fprintf(stderr, "The SAS attests that all software lifecycle activities have been completed\n")
		fmt.Fprintf(stderr, "and their outputs verified. Each activity is mapped to a FuSaOps evidence\n")
		fmt.Fprintf(stderr, "artefact; a missing artefact marks the activity as incomplete.\n\n")
		fmt.Fprintf(stderr, "Flags:\n")
		fs.PrintDefaults()
	}

	var (
		dir    = fs.String("dir", "", "project root directory (default: current directory)")
		output = fs.String("output", "", "path for the SAS report (default: <dir>/.fusaops-sas.json)")
		format = fs.String("format", "text", "output format: text, json")
		level  = fs.String("level", "DAL-C", "software level: DAL-A through DAL-E")
	)
	if err := fs.Parse(args); err != nil {
		return 2
	}

	projectRoot := *dir
	if projectRoot == "" {
		var err error
		projectRoot, err = os.Getwd()
		if err != nil {
			fmt.Fprintf(stderr, "fusaops sas: get working directory: %v\n", err)
			return 1
		}
	}

	s, err := sas.Build(projectRoot, *level)
	if err != nil {
		fmt.Fprintf(stderr, "fusaops sas: build: %v\n", err)
		return 1
	}

	if renderErr := sas.Render(stdout, s, *format); renderErr != nil {
		fmt.Fprintf(stderr, "fusaops sas: render: %v\n", renderErr)
		return 2
	}

	outPath := *output
	if outPath == "" {
		outPath = filepath.Join(projectRoot, sas.ReportFile)
	}
	if saveErr := sas.Save(outPath, s); saveErr != nil {
		fmt.Fprintf(stderr, "fusaops sas: save: %v\n", saveErr)
		return 1
	}
	fmt.Fprintf(stdout, "\nSAS written to %s\n", outPath)
	fmt.Fprintf(stdout, "Integrity hash: %s\n", s.Hash)

	if s.HasGaps() {
		return 1
	}
	return 0
}
