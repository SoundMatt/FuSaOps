package main

import (
	"fmt"
	"io"

	fusaops "github.com/SoundMatt/FuSaOps"
	"github.com/SoundMatt/FuSaOps/qualitybar"
)

// runQualityGate runs both x-FuSa spec §1.6.1 detection rules over fields and
// prints any findings to stderr. It returns 1 (the caller should propagate
// this into the command's exit code) when:
//   - any FUSA-STUB001 (placeholder text) finding exists — always gating,
//     never attestation-suppressible; or
//   - any FUSA-STUB002 (blanket qualitative fallback) finding exists,
//     attestationValid is false, and requireAttestation is set.
//
// A valid attestation suppresses FUSA-STUB002 entirely (with a note printed
// instead of the findings) regardless of requireAttestation.
//
//fusa:req REQ-FO-CLI084
func runQualityGate(stderr io.Writer, artifactFile string, fields []qualitybar.QualField, attestationValid bool, requireAttestation bool) int {
	exit := 0

	placeholderFindings := qualitybar.DetectPlaceholder(artifactFile, fields)
	for _, f := range placeholderFindings {
		fmt.Fprintf(stderr, "[%s] %s: %s\n", f.Severity, f.RuleID, f.Message)
	}
	if len(placeholderFindings) > 0 {
		exit = 1
	}

	fallbackFindings := qualitybar.DetectBlankFallback(artifactFile, fields)
	if len(fallbackFindings) > 0 {
		if attestationValid {
			fmt.Fprintf(stderr, "note: %d FUSA-STUB002 finding(s) suppressed by a valid attestation\n", len(fallbackFindings))
		} else {
			for _, f := range fallbackFindings {
				fmt.Fprintf(stderr, "[%s] %s: %s\n", f.Severity, f.RuleID, f.Message)
			}
			if requireAttestation {
				exit = 1
			}
		}
	}

	return exit
}

// attestationValid reports whether a is a non-stale, genuinely independent
// "reviewed" attestation for content hashed to currentHash. A nil a (no
// attestation present) is always invalid.
//
//fusa:req REQ-FO-CLI084
func attestationValid(a *fusaops.Attestation, currentHash string) bool {
	if a == nil {
		return false
	}
	return fusaops.AttestationValid(*a, currentHash)
}
