// Package fusaops is the root package of FuSaOps, the multi-language
// functional safety orchestration layer.
//
// FuSaOps does not implement language-specific safety rules itself. Instead it
// orchestrates the per-language x-FuSa toolchain (go-FuSa, c-FuSa, cpp-FuSa,
// rust-FuSa, py-FuSa, java-FuSa, and any future tools), aggregates their
// machine-readable reports into a single multi-language evidence view, and
// serves an intuitive web reporting UI.
//
// This root package exports the value types and sentinel errors shared across
// all sub-packages (config, adapter, scan, orchestrator, report, server).
package fusaops

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"regexp"
	"strings"
)

// Version is the current release of FuSaOps.
const Version = "1.147.0"

// SpecVersion is the x-FuSa specification version this release targets.
//
//fusa:req REQ-FO-CORE007
const SpecVersion = "1.15.2"

// Sentinel errors. Callers should use errors.Is for comparison.
//
//fusa:req REQ-FO-ERR001
var (
	// ErrNoConfig is returned when no .fusaops.json is present.
	//fusa:req REQ-FO-ERR002
	ErrNoConfig = errors.New("fusaops: no configuration file found")

	// ErrInvalidConfig is returned when the configuration is malformed.
	//fusa:req REQ-FO-ERR003
	ErrInvalidConfig = errors.New("fusaops: invalid configuration")

	// ErrNoAdapters is returned when a scan detects no languages that any
	// registered adapter can handle.
	//fusa:req REQ-FO-ERR004
	ErrNoAdapters = errors.New("fusaops: no applicable adapters for project")

	// ErrCheckFailed is returned when one or more ERROR-severity findings exist
	// across the aggregated multi-language report.
	//fusa:req REQ-FO-ERR005
	ErrCheckFailed = errors.New("fusaops: one or more safety checks failed")
)

// Severity ranks the importance of a Finding. It serialises as a string in
// JSON output and is value-compatible with every x-FuSa tool's severity scheme.
//
//fusa:req REQ-FO-CORE001
type Severity string

const (
	SeverityInfo    Severity = "INFO"    // Informational observation.
	SeverityWarning Severity = "WARNING" // Should be addressed before release.
	SeverityError   Severity = "ERROR"   // Must be addressed; fails the check.
)

// String implements fmt.Stringer.
func (s Severity) String() string { return string(s) }

// Rank returns an ordinal for sorting/comparison (higher is more severe).
//
//fusa:req REQ-FO-CORE002
func (s Severity) Rank() int {
	switch s {
	case SeverityError:
		return 3
	case SeverityWarning:
		return 2
	case SeverityInfo:
		return 1
	default:
		return 0
	}
}

// Language identifies a programming language handled by an adapter.
//
//fusa:req REQ-FO-CORE003
type Language string

const (
	LangGo     Language = "go"
	LangC      Language = "c"
	LangCpp    Language = "cpp"
	LangRust   Language = "rust"
	LangPython Language = "python"
	LangJava   Language = "java"
	LangAda    Language = "ada"
)

// String implements fmt.Stringer.
func (l Language) String() string { return string(l) }

// Finding is a single observation produced by an x-FuSa tool, normalised into
// the FuSaOps schema. It augments the common per-tool finding shape with the
// Language and Tool that produced it so findings remain attributable after
// they are merged across languages.
//
//fusa:req REQ-FO-CORE004
type Finding struct {
	Language Language `json:"language"`
	Tool     string   `json:"tool"`
	RuleID   string   `json:"ruleId"`
	Severity Severity `json:"severity"`
	Message  string   `json:"message"`
	Location Location `json:"location"`
	Category string   `json:"category,omitempty"`
	// Disposition carries an upstream tool's own §4.1 waiver ("accepted" or
	// "deferred") through the aggregate unchanged. A dispositioned finding
	// MUST remain in the JSON but MUST NOT by itself gate the aggregate exit
	// code — see Severity-independent handling in report.Summary.
	Disposition string `json:"disposition,omitempty"`
	Standard    string `json:"standard,omitempty"`
	Clause      string `json:"clause,omitempty"`
	Remediation string `json:"remediation,omitempty"`
	Fingerprint string `json:"fingerprint,omitempty"`
}

// Gates reports whether f should count toward a gate-failure exit code. A
// finding already dispositioned "accepted" or "deferred" by its originating
// tool MUST NOT by itself cause exit 1 (x-FuSa spec §4.1).
//
//fusa:req REQ-FO-CORE011
func (f Finding) Gates() bool {
	return f.Disposition != "accepted" && f.Disposition != "deferred"
}

// QualifiedRuleID returns the cross-language identity "<language>/<ruleId>"
// (e.g. "go/LINT001"). Any reference that can span languages — the FuSaOps
// aggregate by name — MUST use this qualified form rather than the bare,
// only-tool-local RuleID, since two unrelated tools' rules can share a
// literal id (x-FuSa spec §1.5.1).
//
//fusa:req REQ-FO-CORE012
func (f Finding) QualifiedRuleID() string {
	return string(f.Language) + "/" + f.RuleID
}

var digitRunRE = regexp.MustCompile(`[0-9]+`)

// canonicalStandardIDs maps a human-readable standard display string to its
// canonical lowercase id per x-FuSa spec §2.4.1. Only standards FuSaOps
// itself emits are covered; an unmapped input is returned unchanged by
// CanonicalStandardID rather than guessed at.
var canonicalStandardIDs = map[string]string{
	"ISO 26262": "iso26262",
	"DO-178C":   "do178c",
	"IEC 61508": "iec61508",
	"ISO 21434": "iso21434",
	"iso26262":  "iso26262",
	"do178c":    "do178c",
	"iec61508":  "iec61508",
	"iso21434":  "iso21434",
}

// CanonicalStandardID returns display's canonical lowercase standard id per
// x-FuSa spec §2.4.1 (e.g. "ISO 26262" -> "iso26262"). Per §2.9, a "standard"
// field MUST carry the same canonical value in every output format — display
// remains distinct from the format string, unlike some strings. When display
// is not a known standard, it is returned unchanged.
//
//fusa:req REQ-FO-CORE010
func CanonicalStandardID(display string) string {
	if id, ok := canonicalStandardIDs[display]; ok {
		return id
	}
	return display
}

// ComputeFingerprint returns the canonical sha256:<64 hex> fingerprint for f
// per §4.2 of the x-FuSa spec. Digit runs in Message are normalised to "#"
// and whitespace is collapsed so cosmetic differences do not produce distinct
// fingerprints.
//
//fusa:req REQ-FO-CORE006
func ComputeFingerprint(f Finding) string {
	normalized := strings.Join(strings.Fields(digitRunRE.ReplaceAllString(f.Message, "#")), " ")
	canonical := f.RuleID + "\x1f" + f.Location.File + "\x1f" + normalized
	h := sha256.Sum256([]byte(canonical))
	return "sha256:" + hex.EncodeToString(h[:])
}

// Location identifies the origin of a Finding.
//
//fusa:req REQ-FO-CORE005
type Location struct {
	File   string `json:"file"`
	Line   int    `json:"line,omitempty"`
	Column int    `json:"column,omitempty"`
}
