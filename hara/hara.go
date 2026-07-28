// Package hara manages Hazard Analysis and Risk Assessment (HARA) data for
// FuSaOps projects per ISO 26262-3:2018.
//
// A HARA captures operational situations, hazards, ASIL-rated risk assessments,
// and safety goals in .fusa-hara.json. ASIL is derived automatically from
// Severity × Exposure × Controllability per ISO 26262-3:2018 Table 4.
package hara

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	fusaops "github.com/SoundMatt/FuSaOps"
)

// HARAFile is the standard filename for the HARA data store.
//
//fusa:req REQ-FO-HARA001
const HARAFile = ".fusa-hara.json"

// Severity is the harm severity class (ISO 26262-3:2018 §6.4.3).
type Severity string

const (
	SeverityS0 Severity = "S0"
	SeverityS1 Severity = "S1"
	SeverityS2 Severity = "S2"
	SeverityS3 Severity = "S3"
)

// Exposure is the probability of the operational situation (ISO 26262-3:2018 §6.4.4).
type Exposure string

const (
	ExposureE0 Exposure = "E0"
	ExposureE1 Exposure = "E1"
	ExposureE2 Exposure = "E2"
	ExposureE3 Exposure = "E3"
	ExposureE4 Exposure = "E4"
)

// Controllability is the ability to avoid harm (ISO 26262-3:2018 §6.4.5).
type Controllability string

const (
	ControllabilityC0 Controllability = "C0"
	ControllabilityC1 Controllability = "C1"
	ControllabilityC2 Controllability = "C2"
	ControllabilityC3 Controllability = "C3"
)

// ASIL is the Automotive Safety Integrity Level (ISO 26262-1:2018 §3.6).
type ASIL string

const (
	ASILQM ASIL = "QM"
	ASILA  ASIL = "ASIL-A"
	ASILB  ASIL = "ASIL-B"
	ASILC  ASIL = "ASIL-C"
	ASILD  ASIL = "ASIL-D"
)

// RiskRating holds the three ISO 26262-3 classification parameters and the derived ASIL.
//
//fusa:req REQ-FO-HARA001
type RiskRating struct {
	Severity        Severity        `json:"severity"`
	Exposure        Exposure        `json:"exposure"`
	Controllability Controllability `json:"controllability"`
	ASIL            ASIL            `json:"asil"`
}

// OperationalSituation describes a scenario in which a hazard can manifest.
//
//fusa:req REQ-FO-HARA001
type OperationalSituation struct {
	ID          string `json:"id"`
	Description string `json:"description"`
}

// Hazard describes a potential source of harm.
//
//fusa:req REQ-FO-HARA001
type Hazard struct {
	ID          string     `json:"id"`
	Description string     `json:"description"`
	Source      string     `json:"source,omitempty"`
	Situations  []string   `json:"situations"`
	Risk        RiskRating `json:"risk"`
	SafetyGoals []string   `json:"safetyGoals"`
}

// SafetyGoal is a top-level safety requirement derived from one or more hazards.
//
//fusa:req REQ-FO-HARA001
type SafetyGoal struct {
	ID          string   `json:"id"`
	Description string   `json:"description"`
	HazardIDs   []string `json:"hazards"`
	ASIL        ASIL     `json:"asil"`
	SafeState   string   `json:"safeState,omitempty"`
	// FssrRefs lists the Functional Safety Requirement id(s), from
	// .fusa-reqs.json, that decompose this safety goal. Per x-FuSa spec
	// §1.2.5, a safety goal MUST have at least one — ISO 26262-8 Clause 6
	// requires a safety goal to decompose into a traceable requirement.
	//
	//fusa:req REQ-FO-HARA005
	FssrRefs []string `json:"fssrRefs"`
}

// HARA is the full hazard analysis and risk assessment for a project.
//
//fusa:req REQ-FO-HARA002
type HARA struct {
	Project     string                 `json:"project"`
	Standard    string                 `json:"standard"`
	CreatedAt   time.Time              `json:"createdAt"`
	Situations  []OperationalSituation `json:"operationalSituations"`
	Hazards     []Hazard               `json:"hazards"`
	SafetyGoals []SafetyGoal           `json:"safetyGoals"`
	Attestation *fusaops.Attestation   `json:"attestation,omitempty"`
}

// AttestationContentHash computes the hash a §1.6.2 attestation must match
// to be considered non-stale: h's substantive content, excluding Attestation
// itself (CreatedAt is included — unlike the generated-evidence packages,
// .fusa-hara.json is a human-authored input file whose CreatedAt is stable
// content, not a volatile regeneration timestamp).
//
//fusa:req REQ-FO-HARA006
func AttestationContentHash(h *HARA) string {
	tmp := *h
	tmp.Attestation = nil
	data, err := json.Marshal(tmp)
	if err != nil {
		return ""
	}
	return fusaops.ContentHash(data)
}

// ValidationFinding is a gap identified by Validate.
type ValidationFinding struct {
	HazardID     string
	SafetyGoalID string
	Message      string
}

// DetermineASIL derives the ASIL from S, E, C per ISO 26262-3:2018 Table 4.
//
//fusa:req REQ-FO-HARA004
func DetermineASIL(s Severity, e Exposure, c Controllability) ASIL {
	if s == SeverityS0 || s == "" {
		return ASILQM
	}
	if e == ExposureE0 || e == "" {
		return ASILQM
	}
	type key struct {
		s Severity
		e Exposure
		c Controllability
	}
	table := map[key]ASIL{
		// S1
		{SeverityS1, ExposureE1, ControllabilityC0}: ASILQM,
		{SeverityS1, ExposureE1, ControllabilityC1}: ASILQM,
		{SeverityS1, ExposureE1, ControllabilityC2}: ASILQM,
		{SeverityS1, ExposureE1, ControllabilityC3}: ASILQM,
		{SeverityS1, ExposureE2, ControllabilityC0}: ASILQM,
		{SeverityS1, ExposureE2, ControllabilityC1}: ASILQM,
		{SeverityS1, ExposureE2, ControllabilityC2}: ASILQM,
		{SeverityS1, ExposureE2, ControllabilityC3}: ASILQM,
		{SeverityS1, ExposureE3, ControllabilityC0}: ASILQM,
		{SeverityS1, ExposureE3, ControllabilityC1}: ASILQM,
		{SeverityS1, ExposureE3, ControllabilityC2}: ASILQM,
		{SeverityS1, ExposureE3, ControllabilityC3}: ASILA,
		{SeverityS1, ExposureE4, ControllabilityC0}: ASILQM,
		{SeverityS1, ExposureE4, ControllabilityC1}: ASILQM,
		{SeverityS1, ExposureE4, ControllabilityC2}: ASILA,
		{SeverityS1, ExposureE4, ControllabilityC3}: ASILB,
		// S2
		{SeverityS2, ExposureE1, ControllabilityC0}: ASILQM,
		{SeverityS2, ExposureE1, ControllabilityC1}: ASILQM,
		{SeverityS2, ExposureE1, ControllabilityC2}: ASILQM,
		{SeverityS2, ExposureE1, ControllabilityC3}: ASILQM,
		{SeverityS2, ExposureE2, ControllabilityC0}: ASILQM,
		{SeverityS2, ExposureE2, ControllabilityC1}: ASILQM,
		{SeverityS2, ExposureE2, ControllabilityC2}: ASILA,
		{SeverityS2, ExposureE2, ControllabilityC3}: ASILB,
		{SeverityS2, ExposureE3, ControllabilityC0}: ASILQM,
		{SeverityS2, ExposureE3, ControllabilityC1}: ASILA,
		{SeverityS2, ExposureE3, ControllabilityC2}: ASILB,
		{SeverityS2, ExposureE3, ControllabilityC3}: ASILC,
		{SeverityS2, ExposureE4, ControllabilityC0}: ASILA,
		{SeverityS2, ExposureE4, ControllabilityC1}: ASILB,
		{SeverityS2, ExposureE4, ControllabilityC2}: ASILC,
		{SeverityS2, ExposureE4, ControllabilityC3}: ASILD,
		// S3
		{SeverityS3, ExposureE1, ControllabilityC0}: ASILQM,
		{SeverityS3, ExposureE1, ControllabilityC1}: ASILA,
		{SeverityS3, ExposureE1, ControllabilityC2}: ASILB,
		{SeverityS3, ExposureE1, ControllabilityC3}: ASILC,
		{SeverityS3, ExposureE2, ControllabilityC0}: ASILA,
		{SeverityS3, ExposureE2, ControllabilityC1}: ASILB,
		{SeverityS3, ExposureE2, ControllabilityC2}: ASILC,
		{SeverityS3, ExposureE2, ControllabilityC3}: ASILD,
		{SeverityS3, ExposureE3, ControllabilityC0}: ASILB,
		{SeverityS3, ExposureE3, ControllabilityC1}: ASILC,
		{SeverityS3, ExposureE3, ControllabilityC2}: ASILD,
		{SeverityS3, ExposureE3, ControllabilityC3}: ASILD,
		{SeverityS3, ExposureE4, ControllabilityC0}: ASILC,
		{SeverityS3, ExposureE4, ControllabilityC1}: ASILD,
		{SeverityS3, ExposureE4, ControllabilityC2}: ASILD,
		{SeverityS3, ExposureE4, ControllabilityC3}: ASILD,
	}
	if a, ok := table[key{s, e, c}]; ok {
		return a
	}
	return ASILQM
}

// asilRank maps ASIL to a comparable integer (QM=0, A=1, B=2, C=3, D=4).
func asilRank(a ASIL) int {
	switch a {
	case ASILQM:
		return 0
	case ASILA:
		return 1
	case ASILB:
		return 2
	case ASILC:
		return 3
	case ASILD:
		return 4
	}
	return -1
}

// MaxASIL returns the highest ASIL across all hazards.
//
//fusa:req REQ-FO-HARA002
func MaxASIL(hazards []Hazard) ASIL {
	best := ASILQM
	for _, hz := range hazards {
		a := hz.Risk.ASIL
		if a == "" {
			a = DetermineASIL(hz.Risk.Severity, hz.Risk.Exposure, hz.Risk.Controllability)
		}
		if asilRank(a) > asilRank(best) {
			best = a
		}
	}
	return best
}

// Load reads the HARA from projectRoot/.fusa-hara.json. Returns an empty HARA
// if the file does not exist.
//
//fusa:req REQ-FO-HARA003
func Load(projectRoot string) (*HARA, error) {
	path := filepath.Join(projectRoot, HARAFile)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &HARA{}, nil
		}
		return nil, fmt.Errorf("hara: read %s: %w", HARAFile, err)
	}
	var h HARA
	if err := json.Unmarshal(data, &h); err != nil {
		return nil, fmt.Errorf("hara: parse %s: %w", HARAFile, err)
	}
	return &h, nil
}

// Save writes the HARA to path as indented JSON.
//
//fusa:req REQ-FO-HARA003
func Save(path string, h *HARA) error {
	data, err := json.MarshalIndent(h, "", "  ")
	if err != nil {
		return fmt.Errorf("hara: marshal: %w", err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o640); err != nil {
		return fmt.Errorf("hara: write %s: %w", path, err)
	}
	return nil
}

// Validate checks the HARA for completeness gaps per HARA002–HARA004.
//
//fusa:req REQ-FO-HARA002
func Validate(h *HARA) []ValidationFinding {
	goalIDs := make(map[string]bool)
	for _, g := range h.SafetyGoals {
		goalIDs[g.ID] = true
	}
	var out []ValidationFinding
	for _, hz := range h.Hazards {
		if hz.Risk.Severity == "" || hz.Risk.Exposure == "" || hz.Risk.Controllability == "" {
			out = append(out, ValidationFinding{
				HazardID: hz.ID,
				Message:  fmt.Sprintf("hazard %s has incomplete risk rating — S, E, and C must all be set", hz.ID),
			})
		}
		if len(hz.SafetyGoals) == 0 {
			out = append(out, ValidationFinding{
				HazardID: hz.ID,
				Message:  fmt.Sprintf("hazard %s has no linked safety goal", hz.ID),
			})
		}
		for _, gid := range hz.SafetyGoals {
			if !goalIDs[gid] {
				out = append(out, ValidationFinding{
					HazardID: hz.ID,
					Message:  fmt.Sprintf("hazard %s references unknown safety goal %s", hz.ID, gid),
				})
			}
		}
	}
	for _, g := range h.SafetyGoals {
		if g.ASIL == "" {
			out = append(out, ValidationFinding{
				SafetyGoalID: g.ID,
				Message:      fmt.Sprintf("safety goal %s has no ASIL assigned", g.ID),
			})
		}
		if len(g.FssrRefs) == 0 {
			out = append(out, ValidationFinding{
				SafetyGoalID: g.ID,
				Message:      fmt.Sprintf("safety goal %s has no fssrRefs — a safety goal MUST decompose into at least one functional safety requirement (x-FuSa spec §1.2.5, ISO 26262-8 Clause 6)", g.ID),
			})
		}
	}
	return out
}

// Render writes the HARA to w in text, json, or markdown format.
//
//fusa:req REQ-FO-HARA003
func Render(w io.Writer, h *HARA, format string) error {
	switch format {
	case "json":
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(h)
	case "text", "markdown", "":
		return renderText(w, h)
	default:
		return fmt.Errorf("hara: unsupported format %q", format)
	}
}

func renderText(w io.Writer, h *HARA) error {
	fmt.Fprintf(w, "# Hazard Analysis and Risk Assessment (HARA)\n\n")
	fmt.Fprintf(w, "Project: %s  Standard: %s\n\n", h.Project, h.Standard)

	fmt.Fprintf(w, "## Operational Situations (%d)\n\n", len(h.Situations))
	fmt.Fprintf(w, "| ID | Description |\n|---|---|\n")
	for _, s := range h.Situations {
		fmt.Fprintf(w, "| %s | %s |\n", s.ID, s.Description)
	}
	fmt.Fprintln(w)

	fmt.Fprintf(w, "## Hazards (%d)\n\n", len(h.Hazards))
	fmt.Fprintf(w, "| ID | Description | S | E | C | ASIL | Safety Goals |\n|---|---|---|---|---|---|---|\n")
	for _, hz := range h.Hazards {
		asil := hz.Risk.ASIL
		if asil == "" {
			asil = DetermineASIL(hz.Risk.Severity, hz.Risk.Exposure, hz.Risk.Controllability)
		}
		goals := ""
		for i, g := range hz.SafetyGoals {
			if i > 0 {
				goals += ", "
			}
			goals += g
		}
		fmt.Fprintf(w, "| %s | %s | %s | %s | %s | **%s** | %s |\n",
			hz.ID, hz.Description,
			hz.Risk.Severity, hz.Risk.Exposure, hz.Risk.Controllability,
			asil, goals)
	}
	fmt.Fprintln(w)

	fmt.Fprintf(w, "## Safety Goals (%d)\n\n", len(h.SafetyGoals))
	fmt.Fprintf(w, "| ID | Description | ASIL | Safe State |\n|---|---|---|---|\n")
	for _, g := range h.SafetyGoals {
		fmt.Fprintf(w, "| %s | %s | **%s** | %s |\n", g.ID, g.Description, g.ASIL, g.SafeState)
	}
	fmt.Fprintln(w)

	findings := Validate(h)
	if len(findings) > 0 {
		fmt.Fprintf(w, "## Gaps (%d)\n\n", len(findings))
		for _, f := range findings {
			fmt.Fprintf(w, "- %s\n", f.Message)
		}
		fmt.Fprintln(w)
	}
	return nil
}
