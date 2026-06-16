package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/SoundMatt/FuSaOps/safetycase"
)

// runSafetyCase assembles a structured safety argument from evidence artefacts.
//
//fusa:req REQ-FO-CLI066
func runSafetyCase(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("fusaops safety-case", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() {
		fmt.Fprintf(stderr, "Usage: fusaops safety-case [flags]\n\n")
		fmt.Fprintf(stderr, "Assemble a structured safety case from FuSaOps evidence artefacts.\n\n")
		fmt.Fprintf(stderr, "Each claim in the safety case maps to a class of evidence (test bundle,\n")
		fmt.Fprintf(stderr, "qualification report, SBOM, build provenance, etc.). A claim passes when\n")
		fmt.Fprintf(stderr, "all required artefacts are present in the project root.\n\n")
		fmt.Fprintf(stderr, "Supported standards: ISO 26262, DO-178C, IEC 61508, ISO 21434\n\n")
		fmt.Fprintf(stderr, "Flags:\n")
		fs.PrintDefaults()
	}

	var (
		dir      = fs.String("dir", "", "project root directory (default: current directory)")
		output   = fs.String("output", "", "path for the safety case report (default: <dir>/.fusaops-safety-case.json)")
		format   = fs.String("format", "text", "output format: text, json")
		standard = fs.String("standard", "ISO 26262", "target standard: ISO 26262, DO-178C, IEC 61508, ISO 21434")
	)
	if err := fs.Parse(args); err != nil {
		return 2
	}

	std := safetycase.Standard(*standard)
	valid := false
	for _, s := range safetycase.ValidStandards {
		if s == std {
			valid = true
			break
		}
	}
	if !valid {
		fmt.Fprintf(stderr, "fusaops safety-case: unknown standard %q\n", *standard)
		fmt.Fprintf(stderr, "Supported: ISO 26262, DO-178C, IEC 61508, ISO 21434\n")
		return 2
	}

	projectRoot := *dir
	if projectRoot == "" {
		var err error
		projectRoot, err = os.Getwd()
		if err != nil {
			fmt.Fprintf(stderr, "fusaops safety-case: get working directory: %v\n", err)
			return 1
		}
	}

	sc, err := safetycase.Build(projectRoot, std)
	if err != nil {
		fmt.Fprintf(stderr, "fusaops safety-case: build: %v\n", err)
		return 1
	}

	if renderErr := safetycase.Render(stdout, sc, *format); renderErr != nil {
		fmt.Fprintf(stderr, "fusaops safety-case: render: %v\n", renderErr)
		return 2
	}

	outPath := *output
	if outPath == "" {
		outPath = filepath.Join(projectRoot, safetycase.ReportFile)
	}
	if saveErr := safetycase.Save(outPath, sc); saveErr != nil {
		fmt.Fprintf(stderr, "fusaops safety-case: save: %v\n", saveErr)
		return 1
	}
	fmt.Fprintf(stdout, "\nSafety case written to %s\n", outPath)
	fmt.Fprintf(stdout, "Integrity hash: %s\n", sc.Hash)

	if sc.HasGaps() {
		return 1
	}
	return 0
}
