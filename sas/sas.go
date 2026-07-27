// Package sas generates the Software Accomplishment Summary (SAS) per DO-178C §11.20.
//
// The SAS is the top-level evidence document for a software product — it
// attests that the software lifecycle activities described in the plans have
// been carried out and their outputs verified. At the FuSaOps multi-language
// level it aggregates evidence across all x-FuSa adapters into one summary.
package sas

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

// ReportFile is the default filename for the persisted SAS.
//
//fusa:req REQ-FO-SAS001
const ReportFile = ".fusaops-sas.json"

// ActivityStatus indicates whether a lifecycle activity is complete.
//
//fusa:req REQ-FO-SAS001
type ActivityStatus string

const (
	StatusComplete   ActivityStatus = "complete"
	StatusIncomplete ActivityStatus = "incomplete"
	StatusNA         ActivityStatus = "N/A"
)

// Activity represents one software lifecycle activity from DO-178C Table A-1 to A-10.
//
//fusa:req REQ-FO-SAS001
type Activity struct {
	ID       string         `json:"id"`
	Title    string         `json:"title"`
	Status   ActivityStatus `json:"status"`
	Evidence string         `json:"evidence,omitempty"` // path of supporting evidence file
}

// SAS is the Software Accomplishment Summary document.
//
//fusa:req REQ-FO-SAS001
type SAS struct {
	GeneratedAt        time.Time  `json:"generatedAt"`
	ProjectRoot        string     `json:"projectRoot"`
	Tool               string     `json:"tool"`
	ToolVersion        string     `json:"toolVersion"`
	SoftwareLevel      string     `json:"softwareLevel"` // DAL-A through DAL-E
	Activities         []Activity `json:"activities"`
	TotalActivities    int        `json:"totalActivities"`
	CompleteActivities int        `json:"completeActivities"`
	Hash               string     `json:"hash"`
}

// HasGaps returns true when at least one required activity is incomplete.
//
//fusa:req REQ-FO-SAS005
func (s *SAS) HasGaps() bool { return s.CompleteActivities < s.TotalActivities }

// activitySpec defines one lifecycle activity and the evidence file that
// demonstrates its completion.
type activitySpec struct {
	id       string
	title    string
	evidence string // relative filename that signals completion
}

// lifecycleActivities maps DO-178C lifecycle activities to FuSaOps evidence files.
var lifecycleActivities = []activitySpec{
	{"A-001", "Software planning process", ""},
	{"A-002", "Software development process — requirements", ".fusaops-trace.json"},
	{"A-003", "Software development process — design", ".fusaops-trace.json"},
	{"A-004", "Software development process — coding", ".fusaops-evidence.json"},
	{"A-005", "Software verification process — reviews and analyses", ".fusaops-qualify-report.json"},
	{"A-006", "Software verification process — testing", ".fusaops-evidence.json"},
	{"A-007", "Software configuration management", ".fusaops-sci.json"},
	{"A-008", "Software quality assurance", ".fusaops-qualify-report.json"},
	{"A-009", "Certification liaison process", ".fusaops-safety-case.json"},
	{"A-010", "Software problem reporting", ".fusaops-problems.json"},
	{"A-011", "SBOM and supply-chain evidence", "sbom.json"},
	{"A-012", "Release and build integrity", "artifact-manifest.json"},
}

// statusForEvidence checks whether an evidence file exists and returns
// the corresponding ActivityStatus.
func statusForEvidence(root, file string) ActivityStatus {
	if file == "" {
		return StatusNA
	}
	if _, err := os.Stat(root + "/" + file); err == nil {
		return StatusComplete
	}
	return StatusIncomplete
}

// Build assembles the SAS for the given project root.
//
//fusa:req REQ-FO-SAS002
func Build(root, softwareLevel string) (*SAS, error) {
	if softwareLevel == "" {
		softwareLevel = "DAL-C"
	}

	s := &SAS{
		GeneratedAt:   time.Now().UTC(),
		ProjectRoot:   root,
		Tool:          "fusaops",
		ToolVersion:   fusaops.Version,
		SoftwareLevel: softwareLevel,
	}

	for _, spec := range lifecycleActivities {
		status := statusForEvidence(root, spec.evidence)
		act := Activity{
			ID:       spec.id,
			Title:    spec.title,
			Status:   status,
			Evidence: spec.evidence,
		}
		s.Activities = append(s.Activities, act)
		s.TotalActivities++
		if status == StatusComplete || status == StatusNA {
			s.CompleteActivities++
		}
	}

	s.Hash = computeHash(s)
	return s, nil
}

func computeHash(s *SAS) string {
	tmp := *s
	tmp.Hash = ""
	data, err := json.Marshal(tmp)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// Save writes the SAS to path as indented JSON.
//
//fusa:req REQ-FO-SAS003
func Save(path string, s *SAS) error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("sas: marshal: %w", err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o640); err != nil {
		return fmt.Errorf("sas: write %s: %w", path, err)
	}
	return nil
}

// Load reads a persisted SAS from path.
//
//fusa:req REQ-FO-SAS003
func Load(path string) (*SAS, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fusaops.ErrNoConfig
		}
		return nil, fmt.Errorf("sas: read %s: %w", path, err)
	}
	var s SAS
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("sas: unmarshal %s: %w", path, err)
	}
	return &s, nil
}

// Render writes a representation of the SAS to w in the given format.
// Supported formats: "text", "json".
//
//fusa:req REQ-FO-SAS004
func Render(w io.Writer, s *SAS, format string) error {
	switch format {
	case "text", "":
		return renderText(w, s)
	case "json":
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(s)
	default:
		return fmt.Errorf("sas: unsupported format %q", format)
	}
}

func renderText(w io.Writer, s *SAS) error {
	fmt.Fprintf(w, "Software Accomplishment Summary (DO-178C §11.20)\n")
	fmt.Fprintf(w, "================================================\n")
	fmt.Fprintf(w, "Project:        %s\n", s.ProjectRoot)
	fmt.Fprintf(w, "Tool:           %s v%s\n", s.Tool, s.ToolVersion)
	fmt.Fprintf(w, "Software Level: %s\n", s.SoftwareLevel)
	fmt.Fprintf(w, "Generated:      %s\n", s.GeneratedAt.Format("2006-01-02T15:04:05Z"))
	fmt.Fprintf(w, "Activities:     %d/%d complete\n\n", s.CompleteActivities, s.TotalActivities)

	for _, a := range s.Activities {
		mark := statusMark(a.Status)
		fmt.Fprintf(w, "  %s  [%s] %s — %s\n", mark, a.ID, a.Status, a.Title)
	}

	if s.Hash != "" {
		fmt.Fprintf(w, "\nIntegrity: %s\n", s.Hash)
	}
	if s.HasGaps() {
		fmt.Fprintf(w, "\nINCOMPLETE: %d activity/activities require evidence.\n",
			s.TotalActivities-s.CompleteActivities)
		fmt.Fprintf(w, "Run fusaops verify/qualify/trace/sci/release to generate missing artefacts.\n")
	}
	return nil
}

func statusMark(st ActivityStatus) string {
	switch st {
	case StatusComplete:
		return "✓"
	case StatusNA:
		return "—"
	default:
		return "✗"
	}
}
