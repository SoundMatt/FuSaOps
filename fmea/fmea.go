// Package fmea generates a Design Failure Mode and Effects Analysis (dFMEA)
// for the FuSaOps multi-language safety-analysis toolchain, following the
// IEC 61508 / ISO 26262 Part 8 FMEA methodology.
//
// Build produces a fixed set of failure modes covering the key architectural
// components of the FuSaOps orchestration pipeline. Each failure mode carries
// Severity, Occurrence, and Detection ratings (1–10) from which a Risk
// Priority Number (RPN = S × O × D) is computed. The assembled FMEA can be
// persisted to ReportFile and rendered as human-readable text or JSON.
package fmea

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	fusaops "github.com/SoundMatt/FuSaOps"
)

// ReportFile is the default filename for the persisted FMEA.
//
//fusa:req REQ-FO-FMEA001
const ReportFile = ".fusaops-fmea.json"

// HighRPNThreshold is the RPN value above which a failure mode is classified
// as high-priority and requires immediate corrective action.
//
//fusa:req REQ-FO-FMEA001
const HighRPNThreshold = 100

// FailureMode is one row in the FMEA worksheet.
//
//fusa:req REQ-FO-FMEA001
type FailureMode struct {
	ID         string   `json:"id"`
	Component  string   `json:"component"`
	Function   string   `json:"function"`
	Mode       string   `json:"mode"`
	Effect     string   `json:"effect"`
	Cause      string   `json:"cause"`
	Severity   int      `json:"severity"`   // 1 (negligible) – 10 (catastrophic)
	Occurrence int      `json:"occurrence"` // 1 (improbable) – 10 (frequent)
	Detection  int      `json:"detection"`  // 1 (certain)    – 10 (undetectable)
	RPN        int      `json:"rpn"`        // Severity × Occurrence × Detection
	Controls   []string `json:"controls"`
	Action     string   `json:"action"`
}

// FMEA is the top-level dFMEA document.
//
//fusa:req REQ-FO-FMEA001
type FMEA struct {
	GeneratedAt  time.Time     `json:"generatedAt"`
	ProjectRoot  string        `json:"projectRoot"`
	Tool         string        `json:"tool"`
	ToolVersion  string        `json:"toolVersion"`
	Standard     string        `json:"standard"`
	FailureModes []FailureMode `json:"failureModes"`
	TotalItems   int           `json:"totalItems"`
	HighRPNItems int           `json:"highRpnItems"`
	Hash         string        `json:"hash"`
}

// HasHighRPN returns true when any failure mode exceeds HighRPNThreshold.
//
//fusa:req REQ-FO-FMEA005
func (f *FMEA) HasHighRPN() bool { return f.HighRPNItems > 0 }

// modeSpec holds the static definition of one failure mode; RPN is computed.
type modeSpec struct {
	id         string
	component  string
	function   string
	mode       string
	effect     string
	cause      string
	severity   int
	occurrence int
	detection  int
	controls   []string
	action     string
}

// standardModes covers failure modes in the FuSaOps orchestration pipeline
// per IEC 61508 / ISO 26262 Part 8 FMEA methodology.
var standardModes = []modeSpec{
	{
		id:         "FM-001",
		component:  "Adapter Registry",
		function:   "Execute language-specific safety analysis",
		mode:       "Adapter binary not found on PATH",
		effect:     "Language component analysis silently skipped; safety defects in that language undetected.",
		cause:      "Missing tool installation or PATH misconfiguration in CI environment.",
		severity:   9,
		occurrence: 4,
		detection:  3,
		controls: []string{
			"fusaops adapters — lists adapter availability before analysis",
			"CI gate fails when a required adapter is absent",
			"fusaops sci — inventories tool presence with version and hash",
		},
		action: "Add adapter availability check to the CI gate; fail the build on missing required adapters.",
	},
	{
		id:         "FM-002",
		component:  "cmdAdapter",
		function:   "Run x-FuSa tool and collect findings",
		mode:       "Adapter process exits with non-zero code",
		effect:     "Findings from that language component dropped from the aggregate report.",
		cause:      "Tool bug, memory exhaustion, or malformed source input.",
		severity:   8,
		occurrence: 3,
		detection:  5,
		controls: []string{
			"Adapter exit code propagated as orchestrator error",
			"Orchestrator records skipped and failed adapters in the aggregate report",
			"fusaops qualify — validates adapter health against each subcommand",
		},
		action: "Surface adapter error prominently in aggregate report; treat adapter failure as a gate failure.",
	},
	{
		id:         "FM-003",
		component:  "Report decoder",
		function:   "Parse adapter JSON output per x-FuSa spec",
		mode:       "Adapter output schema does not conform to x-FuSa spec",
		effect:     "Findings silently dropped or aggregate report mis-renders.",
		cause:      "Tool version skew between adapter binary and FuSaOps decoder expectations.",
		severity:   9,
		occurrence: 3,
		detection:  4,
		controls: []string{
			"fusaops conform — schema conformance checks against the x-FuSa spec",
			"fusaops qualify — validates adapter output against each spec section",
			"SpecVersion pinning in capabilities response",
		},
		action: "Run fusaops conform in CI after every tool version update; pin tool versions in Dockerfile.",
	},
	{
		id:         "FM-004",
		component:  "trace package",
		function:   "Cross-language requirement traceability",
		mode:       "Source annotation removed or misspelled",
		effect:     "Requirement appears untraceable; false coverage gap reported.",
		cause:      "Developer removes or typos a //fusa:req annotation during refactoring.",
		severity:   7,
		occurrence: 5,
		detection:  5,
		controls: []string{
			"go-FuSa selfcheck catches annotation errors in Go source",
			"CI gate fails on untraced requirements",
			"fusaops trace report flags coverage gaps",
		},
		action: "Install fusaops hooks pre-commit hook to validate annotation format before every commit.",
	},
	{
		id:         "FM-005",
		component:  "sbom package",
		function:   "Merge per-language SBOMs into cross-language SBOM",
		mode:       "Language SBOM file missing from merge input",
		effect:     "Component not included in final SBOM; unknown dependency risk reaches certification.",
		cause:      "Adapter SBOM generation failed silently or output file not written.",
		severity:   6,
		occurrence: 3,
		detection:  4,
		controls: []string{
			"fusaops sbom warns on missing per-language SBOM inputs",
			"fusaops sci checks SBOM artefact presence with file hash",
			"Audit pack includes all SBOM artefacts for reviewer verification",
		},
		action: "Add SBOM completeness check to the release gate; fail release if any language SBOM is absent.",
	},
	{
		id:         "FM-006",
		component:  "auditpack / sign packages",
		function:   "Verify integrity of evidence artefacts",
		mode:       "Artefact file modified after signing",
		effect:     "Tampered evidence passes a manual inspection that does not re-verify the HMAC.",
		cause:      "Post-sign file modification (intentional or accidental) or disk corruption.",
		severity:   9,
		occurrence: 2,
		detection:  3,
		controls: []string{
			"fusaops sign — HMAC-SHA256 verification step re-verifies before submission",
			"fusaops audit-pack — includes hash manifest with all artefact digests",
			"Release gate re-verifies all signatures before generating provenance",
		},
		action: "Mandate fusaops sign --verify as the final step of the release pipeline before submission.",
	},
	{
		id:         "FM-007",
		component:  "cmdAdapter / orchestrator",
		function:   "Execute adapter within configured timeout",
		mode:       "Adapter process hangs indefinitely",
		effect:     "Analysis pipeline blocked; no findings produced; release delayed or silently skipped.",
		cause:      "Deadlock in adapter tool, infinite loop, or unavailable external resource.",
		severity:   7,
		occurrence: 3,
		detection:  6,
		controls: []string{
			"Adapter process killed after configurable per-adapter timeout",
			"Orchestrator records the skipped component with reason in the aggregate report",
			"CI wall-clock timeout surfaces the failure as a build error",
		},
		action: "Set per-adapter timeout in .fusaops.json; raise alert in report when timeout fires.",
	},
	{
		id:         "FM-008",
		component:  "suppression package",
		function:   "Filter findings against the suppression list",
		mode:       "Overly broad suppression rule hides new violations",
		effect:     "New safety violations match an existing suppression entry and are never reviewed.",
		cause:      "Wildcard or unlimited-duration suppression entry in .fusaops-suppress.json.",
		severity:   8,
		occurrence: 4,
		detection:  4,
		controls: []string{
			"fusaops suppress requires explicit rule IDs; no glob wildcards",
			"PR review required for all changes to .fusaops-suppress.json",
			"fusaops qualify audits active suppressions for expiry compliance",
		},
		action: "Enforce suppression expiry dates; alert in CI when expired suppressions are still active.",
	},
}

// Build assembles the FMEA for the given project root.
//
//fusa:req REQ-FO-FMEA002
func Build(root string) (*FMEA, error) {
	f := &FMEA{
		GeneratedAt: time.Now().UTC(),
		ProjectRoot: root,
		Tool:        "fusaops",
		ToolVersion: fusaops.Version,
		Standard:    "IEC 61508:2010 / ISO 26262:2018 Part 8-7",
	}

	for _, spec := range standardModes {
		rpn := spec.severity * spec.occurrence * spec.detection
		fm := FailureMode{
			ID:         spec.id,
			Component:  spec.component,
			Function:   spec.function,
			Mode:       spec.mode,
			Effect:     spec.effect,
			Cause:      spec.cause,
			Severity:   spec.severity,
			Occurrence: spec.occurrence,
			Detection:  spec.detection,
			RPN:        rpn,
			Controls:   spec.controls,
			Action:     spec.action,
		}
		f.FailureModes = append(f.FailureModes, fm)
		f.TotalItems++
		if rpn > HighRPNThreshold {
			f.HighRPNItems++
		}
	}

	f.Hash = computeHash(f)
	return f, nil
}

func computeHash(f *FMEA) string {
	tmp := *f
	tmp.Hash = ""
	data, err := json.Marshal(tmp)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// Save writes the FMEA to path as indented JSON.
//
//fusa:req REQ-FO-FMEA003
func Save(path string, f *FMEA) error {
	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return fmt.Errorf("fmea: marshal: %w", err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o640); err != nil {
		return fmt.Errorf("fmea: write %s: %w", path, err)
	}
	return nil
}

// Load reads a persisted FMEA from path.
//
//fusa:req REQ-FO-FMEA003
func Load(path string) (*FMEA, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fusaops.ErrNoConfig
		}
		return nil, fmt.Errorf("fmea: read %s: %w", path, err)
	}
	var f FMEA
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("fmea: unmarshal %s: %w", path, err)
	}
	return &f, nil
}

// Render writes a representation of the FMEA to w in the given format.
// Supported formats: "text", "json".
//
//fusa:req REQ-FO-FMEA004
func Render(w io.Writer, f *FMEA, format string) error {
	switch format {
	case "text", "":
		return renderText(w, f)
	case "json":
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(f)
	default:
		return fmt.Errorf("fmea: unsupported format %q", format)
	}
}

func renderText(w io.Writer, f *FMEA) error {
	fmt.Fprintf(w, "FuSaOps Design Failure Mode and Effects Analysis (dFMEA)\n")
	fmt.Fprintf(w, "=========================================================\n")
	fmt.Fprintf(w, "Standard:  %s\n", f.Standard)
	fmt.Fprintf(w, "Project:   %s\n", f.ProjectRoot)
	fmt.Fprintf(w, "Tool:      %s v%s\n", f.Tool, f.ToolVersion)
	fmt.Fprintf(w, "Generated: %s\n", f.GeneratedAt.Format("2006-01-02T15:04:05Z"))
	fmt.Fprintf(w, "Items:     %d total, %d high-RPN (>%d)\n\n",
		f.TotalItems, f.HighRPNItems, HighRPNThreshold)

	for _, fm := range f.FailureModes {
		priority := "  "
		if fm.RPN > HighRPNThreshold {
			priority = "!"
		}
		fmt.Fprintf(w, "[%s] %s — %s\n", priorityLabel(fm.RPN), fm.ID, fm.Component)
		fmt.Fprintf(w, "  Function:   %s\n", fm.Function)
		fmt.Fprintf(w, "  Mode:       %s\n", fm.Mode)
		fmt.Fprintf(w, "  Effect:     %s\n", fm.Effect)
		fmt.Fprintf(w, "  Cause:      %s\n", fm.Cause)
		fmt.Fprintf(w, "  S=%d O=%d D=%d  RPN=%d%s\n",
			fm.Severity, fm.Occurrence, fm.Detection, fm.RPN,
			highMark(fm.RPN))
		for _, c := range fm.Controls {
			fmt.Fprintf(w, "  • %s\n", c)
		}
		fmt.Fprintf(w, "  Action: %s\n", fm.Action)
		fmt.Fprintln(w)
		_ = priority // used via priorityLabel
	}

	if f.Hash != "" {
		fmt.Fprintf(w, "Integrity: %s\n", f.Hash)
	}
	if f.HasHighRPN() {
		fmt.Fprintf(w, "\nHIGH-RPN ITEMS: %d failure mode(s) require corrective action (RPN > %d).\n",
			f.HighRPNItems, HighRPNThreshold)
	}
	return nil
}

func priorityLabel(rpn int) string {
	switch {
	case rpn > 200:
		return "CRITICAL"
	case rpn > HighRPNThreshold:
		return "HIGH    "
	case rpn > 50:
		return "MEDIUM  "
	default:
		return "LOW     "
	}
}

func highMark(rpn int) string {
	if rpn > HighRPNThreshold {
		return " ← HIGH"
	}
	return ""
}
