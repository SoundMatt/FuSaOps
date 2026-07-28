package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	fusaops "github.com/SoundMatt/FuSaOps"
	"github.com/SoundMatt/FuSaOps/qualitybar"
	"github.com/SoundMatt/FuSaOps/tara"
)

// runTARA generates a Threat Analysis and Risk Assessment per ISO 21434:2021 Ch. 9.
//
//fusa:req REQ-FO-CLI069
func runTARA(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("fusaops tara", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() {
		fmt.Fprintf(stderr, "Usage: fusaops tara [flags]\n\n")
		fmt.Fprintf(stderr, "Generate a Threat Analysis and Risk Assessment (TARA) per ISO 21434:2021 Ch. 9.\n\n")
		fmt.Fprintf(stderr, "Produces a structured report of cybersecurity threat scenarios for the\n")
		fmt.Fprintf(stderr, "multi-language safety-analysis toolchain, each with impact, feasibility,\n")
		fmt.Fprintf(stderr, "computed risk level, and recommended treatment controls.\n\n")
		fmt.Fprintf(stderr, "Exits 1 if any scenario carries a critical risk level.\n\n")
		fmt.Fprintf(stderr, "Flags:\n")
		fs.PrintDefaults()
	}

	var (
		dir                = fs.String("dir", "", "project root directory (default: current directory)")
		output             = fs.String("output", "", "path for the TARA report (default: <dir>/.fusaops-tara.json)")
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
			fmt.Fprintf(stderr, "fusaops tara: get working directory: %v\n", err)
			return 1
		}
	}

	outPath := *output
	if outPath == "" {
		outPath = filepath.Join(projectRoot, tara.ReportFile)
	}

	var priorAttestation *fusaops.Attestation
	if prior, loadErr := tara.Load(outPath); loadErr == nil {
		priorAttestation = prior.Attestation
	}

	t, err := tara.Build(projectRoot)
	if err != nil {
		fmt.Fprintf(stderr, "fusaops tara: build: %v\n", err)
		return 1
	}
	t.Attestation = priorAttestation

	if renderErr := tara.Render(stdout, t, *format); renderErr != nil {
		fmt.Fprintf(stderr, "fusaops tara: render: %v\n", renderErr)
		return 2
	}

	if saveErr := tara.Save(outPath, t); saveErr != nil {
		fmt.Fprintf(stderr, "fusaops tara: save: %v\n", saveErr)
		return 1
	}
	fmt.Fprintf(stdout, "\nTARA written to %s\n", outPath)
	fmt.Fprintf(stdout, "Integrity hash: %s\n", t.Hash)

	exit := 0
	if t.HasCritical() {
		exit = 1
	}

	valid := attestationValid(t.Attestation, tara.AttestationContentHash(t))
	qualFields := taraQualFields(t)
	if code := runQualityGate(stderr, tara.ReportFile, qualFields, valid, *requireAttestation); code != 0 {
		exit = code
	}

	if *minCoverage >= 0 && t.Summary.CoveragePct < float64(*minCoverage) {
		fmt.Fprintf(stderr, "fusaops tara: coverage %.1f%% < required %d%%\n", t.Summary.CoveragePct, *minCoverage)
		exit = 1
	}

	return exit
}

// taraQualFields extracts t's qualitative text fields for §1.6.1 detection.
//
//fusa:req REQ-FO-CLI084
func taraQualFields(t *tara.TARA) []qualitybar.QualField {
	var out []qualitybar.QualField
	for _, s := range t.Threats {
		out = append(out,
			qualitybar.QualField{EntryID: s.ID, Field: "threat", Value: s.Threat},
			qualitybar.QualField{EntryID: s.ID, Field: "damageScenario", Value: s.DamageScenario},
		)
	}
	return out
}
