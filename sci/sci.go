// Package sci generates the Software Configuration Index (SCI) per DO-178C §11.16.
//
// The SCI provides an auditable inventory of software configuration items —
// tools, evidence artefacts, and detected language components — with SHA-256
// hashes and availability status. At the FuSaOps multi-language level it
// aggregates items from all x-FuSa adapters into one document.
package sci

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"runtime"
	"time"

	fusaops "github.com/SoundMatt/FuSaOps"
	"github.com/SoundMatt/FuSaOps/adapter"
)

// ReportFile is the default filename for the persisted SCI.
//
//fusa:req REQ-FO-SCI001
const ReportFile = ".fusaops-sci.json"

// ItemKind classifies a configuration item.
//
//fusa:req REQ-FO-SCI001
type ItemKind string

const (
	KindTool      ItemKind = "tool"
	KindArtefact  ItemKind = "artefact"
	KindComponent ItemKind = "component"
)

// ConfigItem is one entry in the Software Configuration Index.
//
//fusa:req REQ-FO-SCI001
type ConfigItem struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Kind     ItemKind `json:"kind"`
	Version  string   `json:"version,omitempty"`
	Language string   `json:"language,omitempty"`
	Path     string   `json:"path,omitempty"`
	SHA256   string   `json:"sha256,omitempty"`
	Size     int64    `json:"size,omitempty"`
	Present  bool     `json:"present"`
}

// ArtifactRef is x-FuSa spec §9.3's promoted-schema view of an evidence
// artefact: `file`/`hash`/`version` — a project-relative subset of the
// richer ConfigItem/Items model, covering only KindArtefact entries that
// are actually present (a missing artefact has no hash to report).
//
//fusa:req REQ-FO-SCI005
type ArtifactRef struct {
	File    string `json:"file"`
	Hash    string `json:"hash"`
	Version string `json:"version,omitempty"`
}

// SCI is the Software Configuration Index document. Items/ConfigItem remain
// the primary tool+artefact+component inventory; Artifacts is the
// project-relative file/hash projection required by x-FuSa spec §9.3,
// derived from the same artefact scan at Build time.
//
//fusa:req REQ-FO-SCI001
type SCI struct {
	// Common header, x-FuSa spec §3.1.
	SchemaVersion string        `json:"schemaVersion"`
	Kind          string        `json:"kind"`
	Language      string        `json:"language"`
	GeneratedAt   time.Time     `json:"generatedAt"`
	ProjectRoot   string        `json:"projectRoot"`
	Tool          string        `json:"tool"`
	ToolVersion   string        `json:"toolVersion"`
	GoVersion     string        `json:"goVersion"`
	Items         []ConfigItem  `json:"items"`
	TotalItems    int           `json:"totalItems"`
	Artifacts     []ArtifactRef `json:"artifacts"`
	Hash          string        `json:"hash"`
}

// knownArtefacts lists the standard FuSaOps evidence artefacts.
var knownArtefacts = []struct {
	id   string
	name string
	file string
}{
	{"DOC-001", "Test evidence bundle", ".fusaops-evidence.json"},
	{"DOC-002", "Tool qualification report", ".fusaops-qualify-report.json"},
	{"DOC-003", "Cross-language traceability matrix", ".fusaops-trace.json"},
	{"DOC-004", "Cross-language SBOM", "sbom.json"},
	{"DOC-005", "Build provenance", "provenance.json"},
	{"DOC-006", "Artifact manifest", "artifact-manifest.json"},
	{"DOC-007", "Problem report log", ".fusaops-problems.json"},
	{"DOC-008", "Safety case", ".fusaops-safety-case.json"},
	{"DOC-009", "Project metrics", ".fusaops-metrics.json"},
	{"DOC-010", "Audit evidence pack", "audit-pack.zip"},
}

// hashFile returns the SHA-256 hex digest of a file, or empty string on error.
func hashFile(path string) (string, int64) {
	f, err := os.Open(path)
	if err != nil {
		return "", 0
	}
	defer f.Close()
	h := sha256.New()
	n, err := io.Copy(h, f)
	if err != nil {
		return "", 0
	}
	return hex.EncodeToString(h.Sum(nil)), n
}

// Build assembles the SCI for the given project root.
//
//fusa:req REQ-FO-SCI002
func Build(root string, adapters []adapter.Adapter) (*SCI, error) {
	s := &SCI{
		SchemaVersion: fusaops.SpecVersion,
		Kind:          "sci",
		Language:      "go",
		GeneratedAt:   time.Now().UTC(),
		ProjectRoot:   root,
		Tool:          "fusaops",
		ToolVersion:   fusaops.Version,
		GoVersion:     runtime.Version(),
	}

	// Orchestration tool itself.
	s.Items = append(s.Items, ConfigItem{
		ID:      "TOOL-001",
		Name:    "fusaops",
		Kind:    KindTool,
		Version: fusaops.Version,
		Present: true,
	})

	// Each detected x-FuSa adapter.
	for i, a := range adapters {
		item := ConfigItem{
			ID:       fmt.Sprintf("TOOL-%03d", i+2),
			Name:     a.Tool(),
			Kind:     KindTool,
			Language: string(a.Language()),
			Present:  a.Available(),
		}
		s.Items = append(s.Items, item)
	}

	// Known evidence artefacts. Path is project-relative (x-FuSa spec §4 MUST)
	// even though hashFile stats the full filesystem path.
	for _, art := range knownArtefacts {
		fullPath := root + "/" + art.file
		sum, size := hashFile(fullPath)
		item := ConfigItem{
			ID:      art.id,
			Name:    art.name,
			Kind:    KindArtefact,
			Path:    art.file,
			Present: sum != "",
			SHA256:  sum,
			Size:    size,
		}
		s.Items = append(s.Items, item)
		if sum != "" {
			s.Artifacts = append(s.Artifacts, ArtifactRef{File: art.file, Hash: "sha256:" + sum})
		}
	}

	s.TotalItems = len(s.Items)
	s.Hash = computeHash(s)
	return s, nil
}

func computeHash(s *SCI) string {
	tmp := *s
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

// Save writes the SCI to path as indented JSON.
//
//fusa:req REQ-FO-SCI003
func Save(path string, s *SCI) error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("sci: marshal: %w", err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o640); err != nil {
		return fmt.Errorf("sci: write %s: %w", path, err)
	}
	return nil
}

// Load reads a persisted SCI from path.
//
//fusa:req REQ-FO-SCI003
func Load(path string) (*SCI, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fusaops.ErrNoConfig
		}
		return nil, fmt.Errorf("sci: read %s: %w", path, err)
	}
	var s SCI
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("sci: unmarshal %s: %w", path, err)
	}
	return &s, nil
}

// Render writes a representation of the SCI to w in the given format.
// Supported formats: "text", "json".
//
//fusa:req REQ-FO-SCI004
func Render(w io.Writer, s *SCI, format string) error {
	switch format {
	case "text", "":
		return renderText(w, s)
	case "json":
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(s)
	default:
		return fmt.Errorf("sci: unsupported format %q", format)
	}
}

func renderText(w io.Writer, s *SCI) error {
	fmt.Fprintf(w, "Software Configuration Index (DO-178C §11.16)\n")
	fmt.Fprintf(w, "==============================================\n")
	fmt.Fprintf(w, "Project:   %s\n", s.ProjectRoot)
	fmt.Fprintf(w, "Tool:      %s v%s (%s)\n", s.Tool, s.ToolVersion, s.GoVersion)
	fmt.Fprintf(w, "Generated: %s\n", s.GeneratedAt.Format("2006-01-02T15:04:05Z"))
	fmt.Fprintf(w, "Items:     %d\n\n", s.TotalItems)

	// Group by kind.
	for _, kind := range []ItemKind{KindTool, KindComponent, KindArtefact} {
		printed := false
		for _, item := range s.Items {
			if item.Kind != kind {
				continue
			}
			if !printed {
				fmt.Fprintf(w, "%s Items:\n", kindLabel(kind))
				printed = true
			}
			status := "PRESENT"
			if !item.Present {
				status = "MISSING"
			}
			ver := ""
			if item.Version != "" {
				ver = " v" + item.Version
			}
			lang := ""
			if item.Language != "" {
				lang = " [" + item.Language + "]"
			}
			fmt.Fprintf(w, "  [%s] %s  %s%s%s\n", status, item.ID, item.Name, ver, lang)
			if item.SHA256 != "" {
				fmt.Fprintf(w, "         SHA256: %s (%d bytes)\n", item.SHA256, item.Size)
			}
		}
		if printed {
			fmt.Fprintln(w)
		}
	}

	if s.Hash != "" {
		fmt.Fprintf(w, "Integrity: %s\n", s.Hash)
	}
	return nil
}

func kindLabel(k ItemKind) string {
	switch k {
	case KindTool:
		return "Tool"
	case KindArtefact:
		return "Artefact"
	case KindComponent:
		return "Component"
	default:
		return string(k)
	}
}
