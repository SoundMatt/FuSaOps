// Package tara generates a Threat Analysis and Risk Assessment (TARA) per
// ISO 21434:2021 Chapter 9 for the FuSaOps software development toolchain.
//
// Build produces a fixed set of cybersecurity threat scenarios relevant to a
// multi-language safety-analysis pipeline. Each scenario carries an impact
// rating, attack feasibility, computed risk level, and recommended treatment.
// The assembled TARA can be persisted to ReportFile and rendered as
// human-readable text or machine-readable JSON.
package tara

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

// ReportFile is the default filename for the persisted TARA.
//
//fusa:req REQ-FO-TARA001
const ReportFile = ".fusaops-tara.json"

// Impact rates the damage potential of a threat scenario.
//
//fusa:req REQ-FO-TARA001
type Impact string

const (
	ImpactCritical   Impact = "critical"   // Catastrophic safety or certification consequence.
	ImpactMajor      Impact = "major"      // Significant harm to safety evidence or process.
	ImpactModerate   Impact = "moderate"   // Limited impact; recoverable with effort.
	ImpactNegligible Impact = "negligible" // No meaningful harm.
)

// Feasibility rates how achievable a threat scenario is for an attacker.
//
//fusa:req REQ-FO-TARA001
type Feasibility string

const (
	FeasibilityHigh    Feasibility = "high"     // Well-resourced attacker with easy access.
	FeasibilityMedium  Feasibility = "medium"   // Moderate skill and access required.
	FeasibilityLow     Feasibility = "low"      // Significant barriers; specialised access needed.
	FeasibilityVeryLow Feasibility = "very_low" // Exceptional attacker capability required.
)

// RiskLevel is the computed outcome of the Impact × Feasibility risk matrix.
//
//fusa:req REQ-FO-TARA001
type RiskLevel string

const (
	RiskCritical RiskLevel = "critical"
	RiskHigh     RiskLevel = "high"
	RiskMedium   RiskLevel = "medium"
	RiskLow      RiskLevel = "low"
)

// TreatmentDecision is the recommended risk treatment per ISO 21434 §9.4.
//
//fusa:req REQ-FO-TARA001
type TreatmentDecision string

const (
	TreatmentMitigate TreatmentDecision = "mitigate"
	TreatmentTransfer TreatmentDecision = "transfer"
	TreatmentAvoid    TreatmentDecision = "avoid"
	TreatmentAccept   TreatmentDecision = "accept"
)

// ThreatScenario is one row in the TARA, covering a single threat/asset pair.
//
//fusa:req REQ-FO-TARA001
type ThreatScenario struct {
	ID             string            `json:"id"`
	Asset          string            `json:"asset"`
	ThreatProperty string            `json:"threatProperty"`
	Description    string            `json:"description"`
	DamageScenario string            `json:"damageScenario"`
	Impact         Impact            `json:"impact"`
	AttackPath     string            `json:"attackPath"`
	Feasibility    Feasibility       `json:"feasibility"`
	RiskLevel      RiskLevel         `json:"riskLevel"`
	Treatment      TreatmentDecision `json:"treatment"`
	Controls       []string          `json:"controls"`
}

// TARA is the top-level threat analysis document.
//
//fusa:req REQ-FO-TARA001
type TARA struct {
	GeneratedAt       time.Time        `json:"generatedAt"`
	ProjectRoot       string           `json:"projectRoot"`
	Tool              string           `json:"tool"`
	ToolVersion       string           `json:"toolVersion"`
	Standard          string           `json:"standard"`
	Scenarios         []ThreatScenario `json:"scenarios"`
	TotalScenarios    int              `json:"totalScenarios"`
	CriticalScenarios int              `json:"criticalScenarios"`
	HighScenarios     int              `json:"highScenarios"`
	Hash              string           `json:"hash"`
}

// HasCritical returns true when any scenario carries a critical risk level.
func (t *TARA) HasCritical() bool { return t.CriticalScenarios > 0 }

// riskMatrix maps Impact × Feasibility to a RiskLevel per ISO 21434 Table 1.
func riskMatrix(imp Impact, feas Feasibility) RiskLevel {
	switch imp {
	case ImpactCritical:
		switch feas {
		case FeasibilityHigh, FeasibilityMedium:
			return RiskCritical
		case FeasibilityLow:
			return RiskHigh
		default:
			return RiskMedium
		}
	case ImpactMajor:
		switch feas {
		case FeasibilityHigh, FeasibilityMedium:
			return RiskHigh
		default:
			return RiskMedium
		}
	case ImpactModerate:
		switch feas {
		case FeasibilityHigh, FeasibilityMedium:
			return RiskMedium
		default:
			return RiskLow
		}
	default:
		return RiskLow
	}
}

// scenarioSpec defines the static fields of a threat scenario; RiskLevel is
// computed from Impact × Feasibility at Build time.
type scenarioSpec struct {
	id             string
	asset          string
	property       string
	description    string
	damageScenario string
	impact         Impact
	attackPath     string
	feasibility    Feasibility
	treatment      TreatmentDecision
	controls       []string
}

// standardScenarios covers cybersecurity threats to a multi-language
// safety-analysis toolchain per ISO 21434:2021 Chapter 9.
var standardScenarios = []scenarioSpec{
	{
		id:             "TS-001",
		asset:          "Software Bill of Materials (SBOM)",
		property:       "Integrity",
		description:    "Attacker modifies the SBOM to hide malicious components or misrepresent software composition.",
		damageScenario: "False software composition evidence submitted to a certification authority.",
		impact:         ImpactCritical,
		attackPath:     "Unauthorized write access to sbom.json or sbom-spdx.json in the release directory.",
		feasibility:    FeasibilityLow,
		treatment:      TreatmentMitigate,
		controls:       []string{"fusaops sign — HMAC artifact signing", "Read-only CI output directories", "Hash verification via artifact-manifest"},
	},
	{
		id:             "TS-002",
		asset:          "Test Evidence Bundle",
		property:       "Authenticity",
		description:    "Insider substitutes passing test results to conceal known defects.",
		damageScenario: "Defective software approved for use in a safety-critical system.",
		impact:         ImpactCritical,
		attackPath:     "Modification of .fusaops-evidence.json before certification review by a developer with repository write access.",
		feasibility:    FeasibilityMedium,
		treatment:      TreatmentMitigate,
		controls:       []string{"HMAC signing (fusaops sign)", "CI-only write access to evidence artefacts", "Audit trail via fusaops audit-pack"},
	},
	{
		id:             "TS-003",
		asset:          "Language Adapter Tools (go-FuSa, c-FuSa, cpp-FuSa, etc.)",
		property:       "Integrity",
		description:    "Supply-chain attacker replaces a language adapter binary with a trojaned version that suppresses findings.",
		damageScenario: "Safety defects not detected by the analysis pipeline; defective software ships.",
		impact:         ImpactCritical,
		attackPath:     "Compromise of upstream tool registry, tampered container image, or PATH manipulation in CI.",
		feasibility:    FeasibilityVeryLow,
		treatment:      TreatmentMitigate,
		controls:       []string{"SLSA provenance verification (fusaops slsa)", "Tool version pinning in Dockerfile", "Hash verification in Software Configuration Index (fusaops sci)"},
	},
	{
		id:             "TS-004",
		asset:          "Aggregate Safety Report",
		property:       "Integrity",
		description:    "Insider or attacker modifies the aggregate report to downgrade or remove findings.",
		damageScenario: "Safety defects concealed from engineering and certification review.",
		impact:         ImpactMajor,
		attackPath:     "Direct file modification of report artefacts before or during review by a developer with local file access.",
		feasibility:    FeasibilityHigh,
		treatment:      TreatmentMitigate,
		controls:       []string{"Artifact signing (fusaops sign)", "Immutable CI artefact storage", "Hash manifest (artifact-manifest.json)"},
	},
	{
		id:             "TS-005",
		asset:          "Signed Release Artifacts",
		property:       "Authenticity",
		description:    "Attacker gains access to the HMAC signing key and creates fraudulent signed artefacts.",
		damageScenario: "Fraudulent signed artefacts accepted as legitimate evidence by a certification body.",
		impact:         ImpactCritical,
		attackPath:     "Compromise of signing key stored in CI/CD secrets, developer workstation, or key management system.",
		feasibility:    FeasibilityLow,
		treatment:      TreatmentMitigate,
		controls:       []string{"Hardware security modules for key storage", "Periodic key rotation", "Least-privilege access to signing secrets"},
	},
	{
		id:             "TS-006",
		asset:          "Requirement Traceability Matrix",
		property:       "Integrity",
		description:    "Attacker modifies or deletes requirement trace links to conceal coverage gaps.",
		damageScenario: "Incomplete requirements coverage not detected; defective design reaches production.",
		impact:         ImpactMajor,
		attackPath:     "Unauthorized commit to the repository or direct modification of .fusaops-trace.json artefact.",
		feasibility:    FeasibilityMedium,
		treatment:      TreatmentMitigate,
		controls:       []string{"PR-gated branch protection with mandatory review", "Signed commits (DCO)", "Artefact hash in artifact-manifest.json"},
	},
	{
		id:             "TS-007",
		asset:          "FuSaOps Orchestration Pipeline",
		property:       "Availability",
		description:    "Attacker disrupts the CI/CD pipeline preventing safety analysis from running before release.",
		damageScenario: "Release proceeds without required safety analysis; unreviewed defects ship.",
		impact:         ImpactMajor,
		attackPath:     "DoS against CI infrastructure, sabotage of build scripts, or removal of adapter tool binaries.",
		feasibility:    FeasibilityMedium,
		treatment:      TreatmentMitigate,
		controls:       []string{"Required status checks for all pull requests", "Redundant CI runner configuration", "Pipeline integrity monitoring"},
	},
	{
		id:             "TS-008",
		asset:          "FuSaOps Configuration (.fusaops.json)",
		property:       "Integrity",
		description:    "Attacker modifies configuration to exclude safety-critical components from analysis.",
		damageScenario: "Known-dangerous code paths excluded from safety analysis; findings suppressed globally.",
		impact:         ImpactMajor,
		attackPath:     "Unauthorized commit modifying .fusaops.json to add scan exclusions or disable adapters.",
		feasibility:    FeasibilityMedium,
		treatment:      TreatmentMitigate,
		controls:       []string{"PR review gate with code-owner approval", "Git commit signing", "Configuration change alerting in CI"},
	},
}

// Build assembles a TARA for the given project root.
//
//fusa:req REQ-FO-TARA002
func Build(root string) (*TARA, error) {
	t := &TARA{
		GeneratedAt: time.Now().UTC(),
		ProjectRoot: root,
		Tool:        "fusaops",
		ToolVersion: fusaops.Version,
		Standard:    "ISO 21434:2021 Chapter 9",
	}

	for _, spec := range standardScenarios {
		risk := riskMatrix(spec.impact, spec.feasibility)
		ts := ThreatScenario{
			ID:             spec.id,
			Asset:          spec.asset,
			ThreatProperty: spec.property,
			Description:    spec.description,
			DamageScenario: spec.damageScenario,
			Impact:         spec.impact,
			AttackPath:     spec.attackPath,
			Feasibility:    spec.feasibility,
			RiskLevel:      risk,
			Treatment:      spec.treatment,
			Controls:       spec.controls,
		}
		t.Scenarios = append(t.Scenarios, ts)
		t.TotalScenarios++
		switch risk {
		case RiskCritical:
			t.CriticalScenarios++
		case RiskHigh:
			t.HighScenarios++
		}
	}

	t.Hash = computeHash(t)
	return t, nil
}

func computeHash(t *TARA) string {
	tmp := *t
	tmp.Hash = ""
	data, err := json.Marshal(tmp)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// Save writes the TARA to path as indented JSON.
//
//fusa:req REQ-FO-TARA003
func Save(path string, t *TARA) error {
	data, err := json.MarshalIndent(t, "", "  ")
	if err != nil {
		return fmt.Errorf("tara: marshal: %w", err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o640); err != nil {
		return fmt.Errorf("tara: write %s: %w", path, err)
	}
	return nil
}

// Load reads a persisted TARA from path.
//
//fusa:req REQ-FO-TARA003
func Load(path string) (*TARA, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fusaops.ErrNoConfig
		}
		return nil, fmt.Errorf("tara: read %s: %w", path, err)
	}
	var t TARA
	if err := json.Unmarshal(data, &t); err != nil {
		return nil, fmt.Errorf("tara: unmarshal %s: %w", path, err)
	}
	return &t, nil
}

// Render writes a representation of the TARA to w in the given format.
// Supported formats: "text", "json".
//
//fusa:req REQ-FO-TARA004
func Render(w io.Writer, t *TARA, format string) error {
	switch format {
	case "text", "":
		return renderText(w, t)
	case "json":
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(t)
	default:
		return fmt.Errorf("tara: unsupported format %q", format)
	}
}

func renderText(w io.Writer, t *TARA) error {
	fmt.Fprintf(w, "FuSaOps Threat Analysis and Risk Assessment (TARA)\n")
	fmt.Fprintf(w, "====================================================\n")
	fmt.Fprintf(w, "Standard:  %s\n", t.Standard)
	fmt.Fprintf(w, "Project:   %s\n", t.ProjectRoot)
	fmt.Fprintf(w, "Tool:      %s v%s\n", t.Tool, t.ToolVersion)
	fmt.Fprintf(w, "Generated: %s\n", t.GeneratedAt.Format("2006-01-02T15:04:05Z"))
	fmt.Fprintf(w, "Scenarios: %d total, %d critical, %d high\n\n",
		t.TotalScenarios, t.CriticalScenarios, t.HighScenarios)

	for _, s := range t.Scenarios {
		fmt.Fprintf(w, "[%s] %s — %s (%s)\n", s.RiskLevel, s.ID, s.Asset, s.ThreatProperty)
		fmt.Fprintf(w, "  Impact: %-12s  Feasibility: %s\n", s.Impact, s.Feasibility)
		fmt.Fprintf(w, "  Threat: %s\n", s.Description)
		fmt.Fprintf(w, "  Damage: %s\n", s.DamageScenario)
		fmt.Fprintf(w, "  Treatment: %s\n", s.Treatment)
		for _, c := range s.Controls {
			fmt.Fprintf(w, "    • %s\n", c)
		}
		fmt.Fprintln(w)
	}

	if t.Hash != "" {
		fmt.Fprintf(w, "Integrity: %s\n", t.Hash)
	}
	if t.HasCritical() {
		fmt.Fprintf(w, "\nCRITICAL RISK: %d scenario(s) require immediate treatment.\n", t.CriticalScenarios)
	}
	return nil
}
