// Package fusaops is the root package of FuSaOps, the multi-language
// functional safety orchestration layer.
//
// FuSaOps does not implement language-specific safety rules itself. Instead it
// orchestrates the per-language x-FuSa toolchain (go-FuSa, c-FuSa, cpp-FuSa and
// future tools), aggregates their machine-readable reports into a single
// multi-language evidence view, and serves an intuitive web reporting UI.
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
const Version = "0.8.0"

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
	Language    Language `json:"language"`
	Tool        string   `json:"tool"`
	RuleID      string   `json:"ruleId"`
	Severity    Severity `json:"severity"`
	Message     string   `json:"message"`
	Location    Location `json:"location"`
	Category    string   `json:"category,omitempty"`
	Remediation string   `json:"remediation,omitempty"`
	Fingerprint string   `json:"fingerprint,omitempty"`
}

var digitRunRE = regexp.MustCompile(`[0-9]+`)

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
