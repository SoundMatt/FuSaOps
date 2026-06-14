// Package slsa provides a SLSA (Supply-chain Levels for Software Artifacts)
// supply-chain gap report for multi-language FuSaOps projects.
//
// It evaluates the project directory against SLSA v1.0 levels L1–L4 and
// produces a gap report showing which objectives are satisfied and which need
// remediation.
package slsa

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// Level is a SLSA supply-chain level (v1.0).
//
//fusa:req REQ-FO-SLSA001
type Level string

const (
	LevelL1 Level = "L1"
	LevelL2 Level = "L2"
	LevelL3 Level = "L3"
	LevelL4 Level = "L4" // treated as L3; SLSA v1.0 has three build levels
)

// Objective is one assessed SLSA objective.
//
//fusa:req REQ-FO-SLSA001
type Objective struct {
	ID       string `json:"id"`
	Clause   string `json:"clause"`
	Title    string `json:"title"`
	MinLevel Level  `json:"minLevel"`
	Status   string `json:"status"` // PASS / GAP / N/A
	Note     string `json:"note,omitempty"`
}

// Report is the SLSA gap assessment result.
//
//fusa:req REQ-FO-SLSA001
type Report struct {
	Project    string      `json:"project"`
	Level      Level       `json:"level"`
	Generated  time.Time   `json:"generated"`
	Pass       int         `json:"pass"`
	Gap        int         `json:"gap"`
	NA         int         `json:"na"`
	Objectives []Objective `json:"objectives"`
}

type objectiveSpec struct {
	id       string
	clause   string
	title    string
	minLevel Level
	file     string // evidence file; empty = checked via code
}

var objectives = []objectiveSpec{
	{"SLSA-L1.1", "§3.1", "Source is version controlled (.git)", LevelL1, ".git"},
	{"SLSA-L1.2", "§3.2", "Build is scripted/automated (module file present)", LevelL1, ""},
	{"SLSA-L1.3", "§4.1", "Build provenance generated (provenance.json)", LevelL1, "provenance.json"},
	{"SLSA-L2.1", "§4.2", "Build system identified in provenance (builder field)", LevelL2, ""},
	{"SLSA-L2.2", "§4.3", "VCS revision recorded in provenance (vcsRevision field)", LevelL2, ""},
	{"SLSA-L2.3", "§4.4", "Software Bill of Materials generated (sbom.json)", LevelL2, "sbom.json"},
	{"SLSA-L3.1", "§5.1", "Two-party review policy (CODEOWNERS or branch-protection)", LevelL3, ""},
	{"SLSA-L3.2", "§5.2", "Dependency integrity tracked (sbom.json with packages)", LevelL3, ""},
	{"SLSA-L3.3", "§5.3", "Artifact integrity recorded (SHA256SUMS or .sha256 files)", LevelL3, ""},
	{"SLSA-L3.4", "§5.4", "Evidence bundle produced (audit-pack.zip)", LevelL3, "audit-pack.zip"},
}

func levelNum(l Level) int {
	switch l {
	case LevelL1:
		return 1
	case LevelL2:
		return 2
	case LevelL3, LevelL4:
		return 3
	}
	return 0
}

// Assess scans projectRoot and returns a SLSA gap report for the given level.
//
//fusa:req REQ-FO-SLSA002
func Assess(projectRoot, project string, level Level) (*Report, error) {
	rep := &Report{
		Project:   project,
		Level:     level,
		Generated: time.Now().UTC(),
	}
	targetNum := levelNum(level)
	for _, spec := range objectives {
		obj := Objective{
			ID:       spec.id,
			Clause:   spec.clause,
			Title:    spec.title,
			MinLevel: spec.minLevel,
		}
		if levelNum(spec.minLevel) > targetNum {
			obj.Status = "N/A"
			rep.NA++
			rep.Objectives = append(rep.Objectives, obj)
			continue
		}
		status, note := assessObjective(projectRoot, spec)
		obj.Status = status
		obj.Note = note
		if status == "PASS" {
			rep.Pass++
		} else {
			rep.Gap++
		}
		rep.Objectives = append(rep.Objectives, obj)
	}
	return rep, nil
}

var moduleFiles = []string{
	"go.mod", "Cargo.toml", "pyproject.toml", "setup.py",
	"pom.xml", "build.gradle", "package.json",
}

var codeownersLocations = []string{
	"CODEOWNERS", ".github/CODEOWNERS", "docs/CODEOWNERS",
	".github/branch-protection.json", ".github/rulesets.json",
}

func assessObjective(root string, spec objectiveSpec) (status, note string) {
	switch spec.id {
	case "SLSA-L1.2":
		for _, f := range moduleFiles {
			if _, err := os.Stat(filepath.Join(root, f)); err == nil {
				return "PASS", ""
			}
		}
		return "GAP", "no build module file found (go.mod, Cargo.toml, pyproject.toml, pom.xml, etc.)"
	case "SLSA-L2.1":
		return assessProvenanceField(root, "builder", "provenance.json missing builder field — add via CI environment (e.g. GITHUB_ACTIONS_URL)")
	case "SLSA-L2.2":
		return assessProvenanceField(root, "vcsRevision", "provenance.json missing vcsRevision — run 'fusaops audit-pack' from a git repo")
	case "SLSA-L3.1":
		for _, name := range codeownersLocations {
			if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(name))); err == nil {
				return "PASS", ""
			}
		}
		return "GAP", "no CODEOWNERS or branch-protection policy — create .github/CODEOWNERS and enable branch protection"
	case "SLSA-L3.2":
		return assessSBOMHashes(root)
	case "SLSA-L3.3":
		return assessArtifactIntegrity(root)
	default:
		if spec.file == "" {
			return "PASS", ""
		}
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(spec.file))); err == nil {
			return "PASS", ""
		}
		return "GAP", fmt.Sprintf("%s not found", spec.file)
	}
}

func assessProvenanceField(root, field, gapNote string) (string, string) {
	data, err := os.ReadFile(filepath.Join(root, "provenance.json"))
	if err != nil {
		return "GAP", "provenance.json not found — run 'fusaops audit-pack' or add a CI release step"
	}
	var prov map[string]interface{}
	if err := json.Unmarshal(data, &prov); err != nil {
		return "GAP", "provenance.json is not valid JSON"
	}
	if v, _ := prov[field].(string); v != "" {
		return "PASS", ""
	}
	return "GAP", gapNote
}

func assessSBOMHashes(root string) (string, string) {
	data, err := os.ReadFile(filepath.Join(root, "sbom.json"))
	if err != nil {
		return "GAP", "sbom.json not found — run 'fusaops sbom'"
	}
	var sbom map[string]interface{}
	if err := json.Unmarshal(data, &sbom); err != nil {
		return "GAP", "sbom.json is not valid JSON"
	}
	for _, key := range []string{"packages", "components", "dependencies"} {
		if arr, ok := sbom[key].([]interface{}); ok && len(arr) > 0 {
			return "PASS", ""
		}
	}
	return "GAP", "sbom.json has no packages/components — regenerate with 'fusaops sbom'"
}

var sha256Candidates = []string{"SHA256SUMS", "sha256sums.txt"}

func assessArtifactIntegrity(root string) (string, string) {
	for _, name := range sha256Candidates {
		if _, err := os.Stat(filepath.Join(root, name)); err == nil {
			return "PASS", ""
		}
	}
	entries, err := os.ReadDir(root)
	if err == nil {
		for _, e := range entries {
			if !e.IsDir() && len(e.Name()) > 7 && e.Name()[len(e.Name())-7:] == ".sha256" {
				return "PASS", ""
			}
		}
	}
	return "GAP", "no SHA256SUMS or .sha256 files found — add artifact integrity hashes to your release process"
}

// Render writes the SLSA gap report to w in the requested format (text or json).
//
//fusa:req REQ-FO-SLSA003
func Render(w io.Writer, rep *Report, format string) error {
	switch format {
	case "json", "":
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(rep)
	case "text":
		return renderText(w, rep)
	default:
		return fmt.Errorf("slsa: unsupported format %q", format)
	}
}

func renderText(w io.Writer, rep *Report) error {
	total := rep.Pass + rep.Gap + rep.NA
	fmt.Fprintf(w, "FuSaOps SLSA Supply-Chain Gap Report\n")
	fmt.Fprintf(w, "Project: %s   Level: %s   Generated: %s\n\n",
		rep.Project, rep.Level, rep.Generated.Format("2006-01-02"))
	fmt.Fprintf(w, "Summary: %d objectives  %d PASS  %d GAP  %d N/A\n\n",
		total, rep.Pass, rep.Gap, rep.NA)
	for _, obj := range rep.Objectives {
		icon := statusIcon(obj.Status)
		suffix := ""
		if obj.Status == "N/A" {
			suffix = " (above target level)"
		}
		fmt.Fprintf(w, "  %s [%s] %s  %s%s\n", icon, obj.ID, obj.Status, obj.Title, suffix)
		if obj.Note != "" {
			fmt.Fprintf(w, "     NOTE: %s\n", obj.Note)
		}
	}
	fmt.Fprintln(w)
	if rep.Gap > 0 {
		fmt.Fprintf(w, "Action items (%d gap(s)):\n", rep.Gap)
		for _, obj := range rep.Objectives {
			if obj.Status == "GAP" {
				fmt.Fprintf(w, "  %s — %s\n", obj.ID, obj.Note)
			}
		}
	}
	return nil
}

func statusIcon(s string) string {
	switch s {
	case "PASS":
		return "✓"
	case "GAP":
		return "✗"
	case "N/A":
		return "–"
	default:
		return "!"
	}
}
