// Package safetycase assembles a structured safety argument from FuSaOps
// evidence artefacts across all language components.
//
// Build discovers known evidence files in the project root and maps them to
// top-level safety claims. Each claim passes when all its required evidence is
// present. The assembled case can be persisted to ReportFile and rendered as
// human-readable text or machine-readable JSON.
package safetycase

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	fusaops "github.com/SoundMatt/FuSaOps"
)

// ReportFile is the default filename for the persisted safety case.
//
//fusa:req REQ-FO-SC001
const ReportFile = ".fusaops-safety-case.json"

// Standard identifies the target functional-safety or cybersecurity standard.
//
//fusa:req REQ-FO-SC001
type Standard string

const (
	StandardISO26262 Standard = "ISO 26262"
	StandardDO178C   Standard = "DO-178C"
	StandardIEC61508 Standard = "IEC 61508"
	StandardISO21434 Standard = "ISO 21434"
)

// ValidStandards lists the accepted standard names.
var ValidStandards = []Standard{StandardISO26262, StandardDO178C, StandardIEC61508, StandardISO21434}

// EvidenceStatus indicates whether an evidence file was found on disk.
//
//fusa:req REQ-FO-SC001
type EvidenceStatus string

const (
	StatusPresent EvidenceStatus = "present"
	StatusMissing EvidenceStatus = "missing"
)

// EvidenceRef points to one evidence artefact that substantiates a claim.
//
//fusa:req REQ-FO-SC001
type EvidenceRef struct {
	Title  string         `json:"title"`
	Path   string         `json:"path"`
	Status EvidenceStatus `json:"status"`
	SHA256 string         `json:"sha256,omitempty"`
	Size   int64          `json:"size,omitempty"`
}

// Claim is one node in the safety argument hierarchy.
//
//fusa:req REQ-FO-SC001
type Claim struct {
	ID       string        `json:"id"`
	Title    string        `json:"title"`
	Strategy string        `json:"strategy"`
	Evidence []EvidenceRef `json:"evidence"`
	Passed   bool          `json:"passed"`
}

// SafetyCase is the top-level safety argument document.
//
//fusa:req REQ-FO-SC001
type SafetyCase struct {
	GeneratedAt  time.Time `json:"generatedAt"`
	ProjectRoot  string    `json:"projectRoot"`
	Tool         string    `json:"tool"`
	ToolVersion  string    `json:"toolVersion"`
	Standard     Standard  `json:"standard"`
	Claims       []Claim   `json:"claims"`
	TotalClaims  int       `json:"totalClaims"`
	PassedClaims int       `json:"passedClaims"`
	Hash         string    `json:"hash"`
}

// HasGaps returns true when at least one claim failed due to missing evidence.
//
//fusa:req REQ-FO-SC005
func (s *SafetyCase) HasGaps() bool { return s.PassedClaims < s.TotalClaims }

// evidenceSpec defines one expected evidence file.
type evidenceSpec struct {
	title    string
	filename string // relative to project root
}

// claimSpec defines one top-level claim with its required evidence.
type claimSpec struct {
	id       string
	title    string
	strategy string
	evidence []evidenceSpec
}

// commonClaims applies to all supported standards.
var commonClaims = []claimSpec{
	{
		id:       "C-001",
		title:    "Tool qualification evidence is available",
		strategy: "Argument by inspection of tool qualification report",
		evidence: []evidenceSpec{
			{title: "Tool qualification report", filename: ".fusaops-qualify-report.json"},
		},
	},
	{
		id:       "C-002",
		title:    "Requirements are traced to tests across all language components",
		strategy: "Argument by inspection of cross-language traceability matrix",
		evidence: []evidenceSpec{
			{title: "Traceability matrix (JSON)", filename: ".fusaops-trace.json"},
		},
	},
	{
		id:       "C-003",
		title:    "Test execution evidence is captured",
		strategy: "Argument by inspection of structured test evidence bundle",
		evidence: []evidenceSpec{
			{title: "Test evidence bundle", filename: ".fusaops-evidence.json"},
		},
	},
	{
		id:       "C-004",
		title:    "Software Bill of Materials is complete",
		strategy: "Argument by inspection of cross-language SBOM",
		evidence: []evidenceSpec{
			{title: "Cross-language SBOM", filename: "sbom.json"},
		},
	},
	{
		id:       "C-005",
		title:    "Build integrity is established",
		strategy: "Argument by inspection of build provenance and artifact manifest",
		evidence: []evidenceSpec{
			{title: "Build provenance", filename: "provenance.json"},
			{title: "Artifact manifest (SHA-256)", filename: "artifact-manifest.json"},
		},
	},
	{
		id:       "C-006",
		title:    "Problem reports are tracked and resolved",
		strategy: "Argument by inspection of problem report log",
		evidence: []evidenceSpec{
			{title: "Problem report log", filename: ".fusaops-problems.json"},
		},
	},
	{
		id:       "C-007",
		title:    "Audit evidence pack is available for review",
		strategy: "Argument by inspection of bundled audit evidence",
		evidence: []evidenceSpec{
			{title: "Audit pack archive", filename: "audit-pack.zip"},
		},
	},
}

// resolveEvidence checks whether each evidence file exists and records its
// SHA-256 hash and size when present.
func resolveEvidence(root string, specs []evidenceSpec) []EvidenceRef {
	refs := make([]EvidenceRef, len(specs))
	for i, s := range specs {
		path := root + "/" + s.filename
		info, err := os.Stat(path)
		if err != nil {
			refs[i] = EvidenceRef{Title: s.title, Path: path, Status: StatusMissing}
			continue
		}
		sum := hashFile(path)
		refs[i] = EvidenceRef{Title: s.title, Path: path, Status: StatusPresent, SHA256: sum, Size: info.Size()}
	}
	return refs
}

// claimPassed returns true when every evidence item has status StatusPresent.
func claimPassed(refs []EvidenceRef) bool {
	for _, r := range refs {
		if r.Status != StatusPresent {
			return false
		}
	}
	return true
}

func hashFile(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return ""
	}
	return hex.EncodeToString(h.Sum(nil))
}

// Build assembles a SafetyCase for the given project root and standard.
//
//fusa:req REQ-FO-SC002
func Build(root string, std Standard) (*SafetyCase, error) {
	sc := &SafetyCase{
		GeneratedAt: time.Now().UTC(),
		ProjectRoot: root,
		Tool:        "fusaops",
		ToolVersion: fusaops.Version,
		Standard:    std,
	}

	for _, cs := range commonClaims {
		refs := resolveEvidence(root, cs.evidence)
		passed := claimPassed(refs)
		sc.Claims = append(sc.Claims, Claim{
			ID:       cs.id,
			Title:    cs.title,
			Strategy: cs.strategy,
			Evidence: refs,
			Passed:   passed,
		})
		sc.TotalClaims++
		if passed {
			sc.PassedClaims++
		}
	}

	// Compute integrity hash over JSON representation (without Hash field).
	sc.Hash = computeHash(sc)
	return sc, nil
}

func computeHash(sc *SafetyCase) string {
	tmp := *sc
	tmp.Hash = ""
	data, err := json.Marshal(tmp)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// Save writes the safety case to path as indented JSON.
//
//fusa:req REQ-FO-SC003
func Save(path string, s *SafetyCase) error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("safetycase: marshal: %w", err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o640); err != nil {
		return fmt.Errorf("safetycase: write %s: %w", path, err)
	}
	return nil
}

// Load reads a persisted safety case from path.
//
//fusa:req REQ-FO-SC003
func Load(path string) (*SafetyCase, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fusaops.ErrNoConfig
		}
		return nil, fmt.Errorf("safetycase: read %s: %w", path, err)
	}
	var sc SafetyCase
	if err := json.Unmarshal(data, &sc); err != nil {
		return nil, fmt.Errorf("safetycase: unmarshal %s: %w", path, err)
	}
	return &sc, nil
}

// Render writes a representation of the safety case to w in the given format.
// Supported formats: "text", "json".
//
//fusa:req REQ-FO-SC004
func Render(w io.Writer, s *SafetyCase, format string) error {
	switch format {
	case "text", "":
		return renderText(w, s)
	case "json":
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(s)
	default:
		return fmt.Errorf("safetycase: unsupported format %q", format)
	}
}

func renderText(w io.Writer, s *SafetyCase) error {
	fmt.Fprintf(w, "FuSaOps Safety Case\n")
	fmt.Fprintf(w, "===================\n")
	fmt.Fprintf(w, "Standard:  %s\n", s.Standard)
	fmt.Fprintf(w, "Project:   %s\n", s.ProjectRoot)
	fmt.Fprintf(w, "Tool:      %s v%s\n", s.Tool, s.ToolVersion)
	fmt.Fprintf(w, "Generated: %s\n", s.GeneratedAt.Format("2006-01-02T15:04:05Z"))
	fmt.Fprintf(w, "Claims:    %d/%d passed\n\n", s.PassedClaims, s.TotalClaims)

	for _, c := range s.Claims {
		status := "PASS"
		if !c.Passed {
			status = "FAIL"
		}
		fmt.Fprintf(w, "[%s] %s — %s\n", status, c.ID, c.Title)
		fmt.Fprintf(w, "     Strategy: %s\n", c.Strategy)
		for _, e := range c.Evidence {
			mark := "✓"
			if e.Status == StatusMissing {
				mark = "✗"
			}
			fmt.Fprintf(w, "     %s  %s (%s)\n", mark, e.Title, shortPath(e.Path, s.ProjectRoot))
		}
	}

	if s.Hash != "" {
		fmt.Fprintf(w, "\nIntegrity: %s\n", s.Hash)
	}
	if s.HasGaps() {
		fmt.Fprintf(w, "\nGAPS DETECTED: %d claim(s) have missing evidence.\n",
			s.TotalClaims-s.PassedClaims)
		fmt.Fprintf(w, "Run the relevant fusaops commands to generate missing artefacts.\n")
	}
	return nil
}

func shortPath(path, root string) string {
	if strings.HasPrefix(path, root+"/") {
		return path[len(root)+1:]
	}
	return path
}
