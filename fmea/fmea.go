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
	"encoding/json"
	"fmt"
	"io"
	"math"
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

// ComponentsInProject is the total count of FuSaOps packages (the core
// orchestration-pipeline table plus the layered compliance/evidence/workflow
// table in CLAUDE.md) — the denominator behind Summary.CoveragePct. Update
// this alongside CLAUDE.md's package tables if either grows.
//
//fusa:req REQ-FO-FMEA008
const ComponentsInProject = 41

// ComponentInventoryMethod documents how ComponentsInProject was counted, so
// Summary.CoveragePct is auditable rather than an unexplained number (x-FuSa
// spec §9.2's coveragePct rationale).
//
//fusa:req REQ-FO-FMEA008
const ComponentInventoryMethod = "count of all packages in CLAUDE.md's core orchestration-pipeline table (11) and layered compliance/evidence/workflow table (30), 41 total; componentsAnalyzed counts distinct FMEA entries, not distinct packages named (some entries span multiple packages, e.g. \"auditpack / sign packages\")"

// RatingScale identifies the severity/occurrence/detection rating table in
// use. FuSaOps predates AIAG-VDA 2019 adoption and uses its own 1-10 scale,
// so it is named rather than claiming a standard it does not implement.
//
//fusa:req REQ-FO-FMEA006
const RatingScale = "custom-1-10"

// FailureMode is one row in the FMEA worksheet.
//
//fusa:req REQ-FO-FMEA001
type FailureMode struct {
	ID         string `json:"id"`
	Component  string `json:"component"`
	Function   string `json:"function"`
	Item       string `json:"item"` // x-FuSa spec §9.2 single-field identity: "Component.Function"
	File       string `json:"file"` // x-FuSa spec §9.2 MUST: project-relative path to the component's source
	Mode       string `json:"failureMode"`
	Effect     string `json:"effect"`
	Cause      string `json:"cause"`
	Severity   int    `json:"severity"`   // 1 (negligible) – 10 (catastrophic)
	Occurrence int    `json:"occurrence"` // 1 (improbable) – 10 (frequent)
	Detection  int    `json:"detection"`  // 1 (certain)    – 10 (undetectable)
	RPN        int    `json:"rpn"`        // Severity × Occurrence × Detection
	// ActionPriority approximates the AIAG-VDA Handbook's Action Priority
	// method (high|medium|low) — see actionPriority() for the (deliberately
	// coarse, severity-dominant) bucketing used in place of the full S×O×D
	// lookup table that method defines.
	ActionPriority string   `json:"actionPriority"`
	Controls       []string `json:"mitigations"`
	Action         string   `json:"action"`
	// RequirementIDs links this failure mode back to the FuSaOps requirement(s)
	// (in .fusa-reqs.json) whose behaviour the corrective action lives in.
	RequirementIDs []string `json:"requirementIds"`
}

// Summary rolls up the FMEA's totals and analysis-coverage metrics.
//
//fusa:req REQ-FO-FMEA006
//fusa:req REQ-FO-FMEA008
type Summary struct {
	Total        int `json:"total"`
	HighPriority int `json:"highPriority"`
	// ComponentsInProject is the stated denominator behind CoveragePct — see
	// ComponentInventoryMethod for how it's counted. This is what stops an
	// FMEA from covering only a few convenient functions while looking
	// thorough by entry count alone (x-FuSa spec §9.2).
	ComponentsAnalyzed       int     `json:"componentsAnalyzed"`
	ComponentsInProject      int     `json:"componentsInProject"`
	CoveragePct              float64 `json:"coveragePct"`
	ComponentInventoryMethod string  `json:"componentInventoryMethod"`
}

// FMEA is the top-level dFMEA document.
//
//fusa:req REQ-FO-FMEA001
type FMEA struct {
	// Common header, x-FuSa spec §3.1.
	SchemaVersion string               `json:"schemaVersion"`
	Kind          string               `json:"kind"`
	Language      string               `json:"language"`
	GeneratedAt   time.Time            `json:"generatedAt"`
	ProjectRoot   string               `json:"projectRoot"`
	Tool          string               `json:"tool"`
	ToolVersion   string               `json:"toolVersion"`
	Standard      string               `json:"standard"`
	RatingScale   string               `json:"ratingScale"`
	Entries       []FailureMode        `json:"entries"`
	Summary       Summary              `json:"summary"`
	Attestation   *fusaops.Attestation `json:"attestation,omitempty"`
	Hash          string               `json:"hash"`
}

// HasHighRPN returns true when any failure mode exceeds HighRPNThreshold.
//
//fusa:req REQ-FO-FMEA005
func (f *FMEA) HasHighRPN() bool { return f.Summary.HighPriority > 0 }

// actionPriority approximates the AIAG-VDA Handbook's Action Priority method
// (high|medium|low), which in the real handbook derives from a full S×O×D
// lookup table. FuSaOps uses a deliberately coarser, severity-dominant
// bucketing — severity ≥8 is always "high" regardless of occurrence/
// detection, matching AIAG-VDA's own philosophy that a severe failure
// warrants urgent attention even when rare or well-detected.
//
//fusa:req REQ-FO-FMEA006
func actionPriority(severity int) string {
	switch {
	case severity >= 8:
		return "high"
	case severity >= 5:
		return "medium"
	default:
		return "low"
	}
}

// modeSpec holds the static definition of one failure mode; RPN is computed.
type modeSpec struct {
	id             string
	component      string
	function       string
	file           string
	mode           string
	effect         string
	cause          string
	severity       int
	occurrence     int
	detection      int
	controls       []string
	action         string
	requirementIDs []string
}

// standardModes covers failure modes in the FuSaOps orchestration pipeline
// per IEC 61508 / ISO 26262 Part 8 FMEA methodology.
var standardModes = []modeSpec{
	{
		id:         "FM-001",
		component:  "Adapter Registry",
		function:   "Execute language-specific safety analysis",
		file:       "adapter/adapter.go",
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
		action:         "Add adapter availability check to the CI gate; fail the build on missing required adapters.",
		requirementIDs: []string{"REQ-FO-ADP007"},
	},
	{
		id:         "FM-002",
		component:  "cmdAdapter",
		function:   "Run x-FuSa tool and collect findings",
		file:       "adapter/adapter.go",
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
		action:         "Surface adapter error prominently in aggregate report; treat adapter failure as a gate failure.",
		requirementIDs: []string{"REQ-FO-ORC002"},
	},
	{
		id:         "FM-003",
		component:  "Report decoder",
		function:   "Parse adapter JSON output per x-FuSa spec",
		file:       "report/report.go",
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
		action:         "Run fusaops conform in CI after every tool version update; pin tool versions in Dockerfile.",
		requirementIDs: []string{"REQ-FO-CNF003"},
	},
	{
		id:         "FM-004",
		component:  "trace package",
		function:   "Cross-language requirement traceability",
		file:       "trace/trace.go",
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
		action:         "Install fusaops hooks pre-commit hook to validate annotation format before every commit.",
		requirementIDs: []string{"REQ-FO-TRC002"},
	},
	{
		id:         "FM-005",
		component:  "sbom package",
		function:   "Merge per-language SBOMs into cross-language SBOM",
		file:       "sbom/sbom.go",
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
		action:         "Add SBOM completeness check to the release gate; fail release if any language SBOM is absent.",
		requirementIDs: []string{"REQ-FO-SBM004"},
	},
	{
		id:         "FM-006",
		component:  "auditpack / sign packages",
		function:   "Verify integrity of evidence artefacts",
		file:       "sign/sign.go",
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
		action:         "Mandate fusaops sign --verify as the final step of the release pipeline before submission.",
		requirementIDs: []string{"REQ-FO-SIGN004"},
	},
	{
		id:         "FM-007",
		component:  "cmdAdapter / orchestrator",
		function:   "Execute adapter within configured timeout",
		file:       "orchestrator/orchestrator.go",
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
		action:         "Set per-adapter timeout in .fusaops.json; raise alert in report when timeout fires.",
		requirementIDs: []string{"REQ-FO-ORC009"},
	},
	{
		id:         "FM-008",
		component:  "suppression package",
		function:   "Filter findings against the suppression list",
		file:       "suppression/suppression.go",
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
		action:         "Enforce suppression expiry dates; alert in CI when expired suppressions are still active.",
		requirementIDs: []string{"REQ-FO-SUP001"},
	},
}

// Build assembles the FMEA for the given project root.
//
//fusa:req REQ-FO-FMEA002
func Build(root string) (*FMEA, error) {
	f := &FMEA{
		SchemaVersion: fusaops.SpecVersion,
		Kind:          "fmea-report",
		Language:      "go",
		GeneratedAt:   time.Now().UTC(),
		ProjectRoot:   root,
		Tool:          "fusaops",
		ToolVersion:   fusaops.Version,
		Standard:      fusaops.CanonicalStandardID("ISO 26262"), // IEC 61508:2010 / ISO 26262:2018 Part 8-7
		RatingScale:   RatingScale,
	}

	for _, spec := range standardModes {
		rpn := spec.severity * spec.occurrence * spec.detection
		fm := FailureMode{
			ID:             spec.id,
			Component:      spec.component,
			Function:       spec.function,
			Item:           spec.component + "." + spec.function,
			File:           spec.file,
			Mode:           spec.mode,
			Effect:         spec.effect,
			Cause:          spec.cause,
			Severity:       spec.severity,
			Occurrence:     spec.occurrence,
			Detection:      spec.detection,
			RPN:            rpn,
			ActionPriority: actionPriority(spec.severity),
			Controls:       spec.controls,
			Action:         spec.action,
			RequirementIDs: spec.requirementIDs,
		}
		f.Entries = append(f.Entries, fm)
		f.Summary.Total++
		if rpn > HighRPNThreshold {
			f.Summary.HighPriority++
		}
	}

	f.Summary.ComponentsAnalyzed = f.Summary.Total
	f.Summary.ComponentsInProject = ComponentsInProject
	f.Summary.CoveragePct = coveragePct(f.Summary.ComponentsAnalyzed, f.Summary.ComponentsInProject)
	f.Summary.ComponentInventoryMethod = ComponentInventoryMethod

	f.Hash = computeHash(f)
	return f, nil
}

// coveragePct returns 100*analyzed/total rounded to one decimal, or 100 when
// total is 0 (no denominator means nothing is uncovered). x-FuSa spec §9.2:
// coveragePct MUST NOT exceed 100 — clamped defensively even though analyzed
// cannot currently exceed total by construction.
func coveragePct(analyzed, total int) float64 {
	if total == 0 {
		return 100
	}
	pct := 100 * float64(analyzed) / float64(total)
	pct = math.Round(pct*10) / 10
	if pct > 100 {
		return 100
	}
	return pct
}

func computeHash(f *FMEA) string {
	tmp := *f
	tmp.Hash = ""
	data, err := json.Marshal(tmp)
	if err != nil {
		return ""
	}
	canon, err := fusaops.Canonicalize(data)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(canon)
	return fmt.Sprintf("sha256:%x", sum)
}

// AttestationContentHash computes the hash a §1.6.2 attestation must match
// to be considered non-stale: f's substantive content, excluding Hash,
// Attestation, and GeneratedAt — the fields an attestation is not about.
//
//fusa:req REQ-FO-FMEA007
func AttestationContentHash(f *FMEA) string {
	tmp := *f
	tmp.Hash = ""
	tmp.Attestation = nil
	tmp.GeneratedAt = time.Time{}
	data, err := json.Marshal(tmp)
	if err != nil {
		return ""
	}
	return fusaops.ContentHash(data)
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
		f.Summary.Total, f.Summary.HighPriority, HighRPNThreshold)

	for _, fm := range f.Entries {
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
			f.Summary.HighPriority, HighRPNThreshold)
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
