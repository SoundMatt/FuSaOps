// Package qualitybar implements the x-FuSa spec §1.6.1 detection heuristics
// for evidence-artifact content quality: placeholder/template text (Rule A,
// FUSA-STUB001, always an ERROR) and blanket qualitative fallback (Rule B,
// FUSA-STUB002, an advisory WARNING suppressible by a valid §1.6.2
// attestation). Both reuse fusaops.Finding and fusaops.ComputeFingerprint so
// they compose with disposition/suppression like any other check finding.
package qualitybar

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	fusaops "github.com/SoundMatt/FuSaOps"
)

// QualField is one qualitative text field to scan, tagged with which entry
// and logical field it came from so a Finding can point back to it and Rule
// B can group same-field values across entries.
//
//fusa:req REQ-QB001
type QualField struct {
	EntryID string // e.g. "FM-001", "H-001", "TS-001" — the entry this belongs to
	Field   string // e.g. "failureMode", "hazard", "threat" — semantic field name
	Value   string
}

var placeholderBracketRE = regexp.MustCompile(`\[[A-Za-z ][^\]]*\]`)

// placeholderSubstrings mirrors the x-FuSa spec §1.6.1 Rule A deny-list.
var placeholderSubstrings = []string{
	"replace with",
	"example hazard",
	"tbd",
	"lorem ipsum",
	"fill in",
}

// DetectPlaceholder implements Rule A: a deny-list scan for literal
// template/scaffold text. A match is always an ERROR finding — never
// suppressible via attestation, only via a per-finding disposition, because
// no attestation can make placeholder text real.
//
//fusa:req REQ-QB001
func DetectPlaceholder(artifactFile string, fields []QualField) []fusaops.Finding {
	var out []fusaops.Finding
	for _, f := range fields {
		if !looksLikePlaceholder(f.Value) {
			continue
		}
		finding := fusaops.Finding{
			Tool:        "fusaops",
			RuleID:      "FUSA-STUB001",
			Severity:    fusaops.SeverityError,
			Message:     fmt.Sprintf("%s.%s contains placeholder/template text: %q", f.EntryID, f.Field, f.Value),
			Location:    fusaops.Location{File: artifactFile},
			Category:    "safety",
			Remediation: "replace the placeholder text with real, item-specific analysis",
		}
		finding.Fingerprint = fusaops.ComputeFingerprint(finding)
		out = append(out, finding)
	}
	return out
}

func looksLikePlaceholder(v string) bool {
	lower := strings.ToLower(v)
	for _, s := range placeholderSubstrings {
		if strings.Contains(lower, s) {
			return true
		}
	}
	return placeholderBracketRE.MatchString(v)
}

// DetectBlankFallback implements Rule B: for a group of ≥10 entries sharing
// the same semantic field (grouped by QualField.Field), a distinct-value
// ratio below 0.1 (fewer than 1 distinct value per 10 entries) is flagged as
// a WARNING — advisory only, never gating on its own, because a similarity
// heuristic can misread genuinely repetitive-but-real content as templated.
// Suppress via a valid attestation (fusaops.AttestationValid) rather than a
// disposition, since the concern is about the artifact as a whole, not one
// entry.
//
//fusa:req REQ-QB002
func DetectBlankFallback(artifactFile string, fields []QualField) []fusaops.Finding {
	byField := map[string][]QualField{}
	for _, f := range fields {
		byField[f.Field] = append(byField[f.Field], f)
	}

	fieldNames := make([]string, 0, len(byField))
	for name := range byField {
		fieldNames = append(fieldNames, name)
	}
	sort.Strings(fieldNames) // deterministic finding order

	var out []fusaops.Finding
	for _, fieldName := range fieldNames {
		group := byField[fieldName]
		if len(group) < 10 {
			continue
		}
		distinct := map[string]bool{}
		for _, f := range group {
			distinct[f.Value] = true
		}
		ratio := float64(len(distinct)) / float64(len(group))
		if ratio >= 0.1 {
			continue
		}
		finding := fusaops.Finding{
			Tool:     "fusaops",
			RuleID:   "FUSA-STUB002",
			Severity: fusaops.SeverityWarning,
			Message: fmt.Sprintf(
				"field %q shows only %d distinct value(s) across %d entries (ratio %.2f < 0.10) — looks templated, not per-item analysis",
				fieldName, len(distinct), len(group), ratio),
			Location:    fusaops.Location{File: artifactFile},
			Category:    "safety",
			Remediation: "vary this field per entry's actual signature/behaviour, or attest (x-FuSa spec §1.6.2) that the repetition is genuine",
		}
		finding.Fingerprint = fusaops.ComputeFingerprint(finding)
		out = append(out, finding)
	}
	return out
}
