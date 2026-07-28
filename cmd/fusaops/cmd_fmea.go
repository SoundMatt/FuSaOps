package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	fusaops "github.com/SoundMatt/FuSaOps"
	"github.com/SoundMatt/FuSaOps/fmea"
	"github.com/SoundMatt/FuSaOps/qualitybar"
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
		dir                = fs.String("dir", "", "project root directory (default: current directory)")
		output             = fs.String("output", "", "path for the FMEA report (default: <dir>/.fusaops-fmea.json)")
		format             = fs.String("format", "text", "output format: text, json")
		minCoverage        = fs.Int("min-coverage", -1, "fail when coveragePct < N (0-100)")
		strict             = fs.Bool("strict", false, "implies --require-attestation")
		requireAttestation = fs.Bool("require-attestation", false, "gate exit code on an unsuppressed FUSA-STUB002 finding")
	)
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *strict {
		*requireAttestation = true
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

	outPath := *output
	if outPath == "" {
		outPath = filepath.Join(projectRoot, fmea.ReportFile)
	}

	// Carry forward any prior attestation (from a previously-saved report at
	// outPath) — Build always assembles a fresh document, so without this a
	// hand-added attestation would be silently discarded on the next run.
	var priorAttestation *fusaops.Attestation
	if prior, loadErr := fmea.Load(outPath); loadErr == nil {
		priorAttestation = prior.Attestation
	}

	f, err := fmea.Build(projectRoot)
	if err != nil {
		fmt.Fprintf(stderr, "fusaops fmea: build: %v\n", err)
		return 1
	}
	f.Attestation = priorAttestation

	if renderErr := fmea.Render(stdout, f, *format); renderErr != nil {
		fmt.Fprintf(stderr, "fusaops fmea: render: %v\n", renderErr)
		return 2
	}

	if saveErr := fmea.Save(outPath, f); saveErr != nil {
		fmt.Fprintf(stderr, "fusaops fmea: save: %v\n", saveErr)
		return 1
	}
	fmt.Fprintf(stdout, "\nFMEA written to %s\n", outPath)
	fmt.Fprintf(stdout, "Integrity hash: %s\n", f.Hash)

	exit := 0
	if f.HasHighRPN() {
		exit = 1
	}

	valid := attestationValid(f.Attestation, fmea.AttestationContentHash(f))
	qualFields := fmeaQualFields(f)
	if code := runQualityGate(stderr, fmea.ReportFile, qualFields, valid, *requireAttestation); code != 0 {
		exit = code
	}

	if *minCoverage >= 0 && f.Summary.CoveragePct < float64(*minCoverage) {
		fmt.Fprintf(stderr, "fusaops fmea: coverage %.1f%% < required %d%%\n", f.Summary.CoveragePct, *minCoverage)
		exit = 1
	}

	return exit
}

// fmeaQualFields extracts f's qualitative text fields for §1.6.1 detection.
//
//fusa:req REQ-FO-CLI084
func fmeaQualFields(f *fmea.FMEA) []qualitybar.QualField {
	var out []qualitybar.QualField
	for _, e := range f.Entries {
		out = append(out,
			qualitybar.QualField{EntryID: e.ID, Field: "failureMode", Value: e.Mode},
			qualitybar.QualField{EntryID: e.ID, Field: "effect", Value: e.Effect},
			qualitybar.QualField{EntryID: e.ID, Field: "cause", Value: e.Cause},
		)
	}
	return out
}
