package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	fusaops "github.com/SoundMatt/FuSaOps"
	"github.com/SoundMatt/FuSaOps/qualitybar"
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
		dir                = fs.String("dir", "", "project root directory (default: current directory)")
		output             = fs.String("output", "", "path for the safety case report (default: <dir>/.fusaops-safety-case.json)")
		format             = fs.String("format", "text", "output format: text, json")
		standard           = fs.String("standard", "ISO 26262", "target standard: ISO 26262, DO-178C, IEC 61508, ISO 21434")
		strict             = fs.Bool("strict", false, "implies --require-attestation")
		requireAttestation = fs.Bool("require-attestation", false, "gate exit code on an unsuppressed FUSA-STUB002 finding")
	)
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *strict {
		*requireAttestation = true
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

	outPath := *output
	if outPath == "" {
		outPath = filepath.Join(projectRoot, safetycase.ReportFile)
	}

	var priorAttestation *fusaops.Attestation
	if prior, loadErr := safetycase.Load(outPath); loadErr == nil {
		priorAttestation = prior.Attestation
	}

	sc, err := safetycase.Build(projectRoot, std)
	if err != nil {
		fmt.Fprintf(stderr, "fusaops safety-case: build: %v\n", err)
		return 1
	}
	sc.Attestation = priorAttestation

	if renderErr := safetycase.Render(stdout, sc, *format); renderErr != nil {
		fmt.Fprintf(stderr, "fusaops safety-case: render: %v\n", renderErr)
		return 2
	}

	if saveErr := safetycase.Save(outPath, sc); saveErr != nil {
		fmt.Fprintf(stderr, "fusaops safety-case: save: %v\n", saveErr)
		return 1
	}
	fmt.Fprintf(stdout, "\nSafety case written to %s\n", outPath)
	fmt.Fprintf(stdout, "Integrity hash: %s\n", sc.Hash)

	exit := 0
	if sc.HasGaps() {
		exit = 1
	}

	attestOK := attestationValid(sc.Attestation, safetycase.AttestationContentHash(sc))
	if code := runQualityGate(stderr, safetycase.ReportFile, safetyCaseQualFields(sc), attestOK, *requireAttestation); code != 0 {
		exit = code
	}

	return exit
}

// safetyCaseQualFields extracts sc's qualitative text fields (each GSN
// node's Text) for §1.6.1 detection.
//
//fusa:req REQ-FO-CLI084
func safetyCaseQualFields(sc *safetycase.SafetyCase) []qualitybar.QualField {
	var out []qualitybar.QualField
	for _, n := range sc.Nodes {
		out = append(out, qualitybar.QualField{EntryID: n.ID, Field: string(n.Type), Value: n.Text})
	}
	return out
}
