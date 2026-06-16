// Package vuln discovers dependency manifests across all languages and
// integrates with osv-scanner (https://github.com/google/osv-scanner) to
// produce a cross-language vulnerability report.
//
// Scan walks the project root for go.mod, Cargo.toml, requirements.txt,
// package.json, and pom.xml files. When osv-scanner is present on PATH it is
// invoked and its JSON output is parsed into a structured VulnReport. When
// osv-scanner is absent each discovered manifest is recorded with status
// "skipped" and the report is still written — callers can use it to see what
// manifests exist even without the scanner binary.
//
// The runner parameter enables testable injection of a fake osv-scanner.
package vuln

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	fusaops "github.com/SoundMatt/FuSaOps"
)

// ReportFile is the default filename for the persisted vulnerability report.
//
//fusa:req REQ-FO-VULN001
const ReportFile = ".fusaops-vuln.json"

// ScannerBinary is the expected binary name for the OSV vulnerability scanner.
//
//fusa:req REQ-FO-VULN001
const ScannerBinary = "osv-scanner"

// ManifestKind identifies the type of a dependency manifest file.
//
//fusa:req REQ-FO-VULN001
type ManifestKind string

const (
	ManifestGoMod           ManifestKind = "go_mod"
	ManifestCargoToml       ManifestKind = "cargo_toml"
	ManifestRequirementsTxt ManifestKind = "requirements_txt"
	ManifestPackageJSON     ManifestKind = "package_json"
	ManifestPomXML          ManifestKind = "pom_xml"
)

// ScanStatus describes whether a manifest was scanned or skipped.
//
//fusa:req REQ-FO-VULN001
type ScanStatus string

const (
	StatusScanned ScanStatus = "scanned"
	StatusSkipped ScanStatus = "skipped" // scanner binary not available
)

// Manifest records one discovered dependency manifest file.
//
//fusa:req REQ-FO-VULN001
type Manifest struct {
	Kind   ManifestKind `json:"kind"`
	Path   string       `json:"path"`
	Status ScanStatus   `json:"status"`
}

// VulnFinding is one vulnerability found in a dependency.
//
//fusa:req REQ-FO-VULN001
type VulnFinding struct {
	ManifestPath string `json:"manifestPath"`
	Package      string `json:"package"`
	Version      string `json:"version"`
	Ecosystem    string `json:"ecosystem"`
	VulnID       string `json:"vulnId"`
	Severity     string `json:"severity"`
	Summary      string `json:"summary"`
}

// VulnReport is the top-level cross-language vulnerability report.
//
//fusa:req REQ-FO-VULN001
type VulnReport struct {
	GeneratedAt    time.Time     `json:"generatedAt"`
	ProjectRoot    string        `json:"projectRoot"`
	Tool           string        `json:"tool"`
	ToolVersion    string        `json:"toolVersion"`
	ScannerTool    string        `json:"scannerTool"`
	ScannerPresent bool          `json:"scannerPresent"`
	Manifests      []Manifest    `json:"manifests"`
	Findings       []VulnFinding `json:"findings"`
	TotalManifests int           `json:"totalManifests"`
	TotalFindings  int           `json:"totalFindings"`
	CriticalCount  int           `json:"criticalCount"`
	HighCount      int           `json:"highCount"`
	Hash           string        `json:"hash"`
}

// HasFindings returns true when at least one vulnerability was found.
func (r *VulnReport) HasFindings() bool { return r.TotalFindings > 0 }

// RunnerFunc is a function signature for invoking an external binary.
// args[0] is the binary name; the rest are arguments.
// Returns stdout bytes and any error.
type RunnerFunc func(args ...string) ([]byte, error)

// manifestNames maps a base filename to its ManifestKind.
var manifestNames = map[string]ManifestKind{
	"go.mod":           ManifestGoMod,
	"Cargo.toml":       ManifestCargoToml,
	"requirements.txt": ManifestRequirementsTxt,
	"package.json":     ManifestPackageJSON,
	"pom.xml":          ManifestPomXML,
}

// Scan discovers dependency manifests under root and, if runner is non-nil,
// invokes osv-scanner to find vulnerabilities.
//
//fusa:req REQ-FO-VULN002
func Scan(root string, runner RunnerFunc) (*VulnReport, error) {
	r := &VulnReport{
		GeneratedAt: time.Now().UTC(),
		ProjectRoot: root,
		Tool:        "fusaops",
		ToolVersion: fusaops.Version,
		ScannerTool: ScannerBinary,
	}

	manifests, err := discoverManifests(root)
	if err != nil {
		return nil, fmt.Errorf("vuln: discover manifests: %w", err)
	}
	r.TotalManifests = len(manifests)

	if runner == nil {
		runner = defaultRunner
	}

	scannerPresent := isScannerPresent(runner)
	r.ScannerPresent = scannerPresent

	for _, m := range manifests {
		status := StatusSkipped
		if scannerPresent {
			status = StatusScanned
		}
		r.Manifests = append(r.Manifests, Manifest{Kind: m.kind, Path: m.path, Status: status})
	}

	if scannerPresent && len(manifests) > 0 {
		paths := make([]string, len(manifests))
		for i, m := range manifests {
			paths[i] = m.path
		}
		findings, err := runOSVScanner(runner, paths)
		if err == nil {
			r.Findings = findings
		}
		// scanner errors are non-fatal — report what we found
	}

	r.TotalFindings = len(r.Findings)
	for _, f := range r.Findings {
		switch strings.ToUpper(f.Severity) {
		case "CRITICAL":
			r.CriticalCount++
		case "HIGH":
			r.HighCount++
		}
	}

	r.Hash = computeHash(r)
	return r, nil
}

type manifestEntry struct {
	kind ManifestKind
	path string
}

func discoverManifests(root string) ([]manifestEntry, error) {
	var out []manifestEntry
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // skip unreadable entries
		}
		if d.IsDir() {
			name := d.Name()
			if name == "vendor" || name == "node_modules" || name == ".git" || strings.HasPrefix(name, ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if kind, ok := manifestNames[d.Name()]; ok {
			out = append(out, manifestEntry{kind: kind, path: path})
		}
		return nil
	})
	return out, err
}

// defaultRunner invokes a real system binary via os/exec.
func defaultRunner(args ...string) ([]byte, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("vuln: empty command")
	}
	// Use os/exec via the standard library only.
	// We import os/exec indirectly through the shell.
	return runCommand(args[0], args[1:]...)
}

// isScannerPresent checks whether the scanner binary responds to --version.
func isScannerPresent(runner RunnerFunc) bool {
	_, err := runner(ScannerBinary, "--version")
	return err == nil
}

// osvOutput is a minimal subset of the osv-scanner JSON output schema.
type osvOutput struct {
	Results []struct {
		Source struct {
			Path string `json:"path"`
		} `json:"source"`
		Packages []struct {
			Package struct {
				Name      string `json:"name"`
				Version   string `json:"version"`
				Ecosystem string `json:"ecosystem"`
			} `json:"package"`
			Vulnerabilities []struct {
				ID               string `json:"id"`
				Summary          string `json:"summary"`
				DatabaseSpecific struct {
					Severity string `json:"severity"`
				} `json:"database_specific"`
			} `json:"vulnerabilities"`
		} `json:"packages"`
	} `json:"results"`
}

func runOSVScanner(runner RunnerFunc, manifestPaths []string) ([]VulnFinding, error) {
	args := []string{ScannerBinary, "--format", "json"}
	for _, p := range manifestPaths {
		args = append(args, "--lockfile", p)
	}
	out, err := runner(args...)
	if err != nil && len(out) == 0 {
		return nil, err
	}

	// osv-scanner exits non-zero when vulnerabilities are found; that is not an
	// error for us — we still parse the output.
	var parsed osvOutput
	dec := json.NewDecoder(bytes.NewReader(out))
	if err := dec.Decode(&parsed); err != nil {
		return nil, fmt.Errorf("vuln: parse osv-scanner output: %w", err)
	}

	var findings []VulnFinding
	for _, result := range parsed.Results {
		for _, pkg := range result.Packages {
			for _, v := range pkg.Vulnerabilities {
				findings = append(findings, VulnFinding{
					ManifestPath: result.Source.Path,
					Package:      pkg.Package.Name,
					Version:      pkg.Package.Version,
					Ecosystem:    pkg.Package.Ecosystem,
					VulnID:       v.ID,
					Severity:     v.DatabaseSpecific.Severity,
					Summary:      v.Summary,
				})
			}
		}
	}
	return findings, nil
}

func computeHash(r *VulnReport) string {
	tmp := *r
	tmp.Hash = ""
	data, err := json.Marshal(tmp)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// Save writes the vulnerability report to path as indented JSON.
//
//fusa:req REQ-FO-VULN003
func Save(path string, r *VulnReport) error {
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return fmt.Errorf("vuln: marshal: %w", err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o640); err != nil {
		return fmt.Errorf("vuln: write %s: %w", path, err)
	}
	return nil
}

// Load reads a persisted vulnerability report from path.
//
//fusa:req REQ-FO-VULN003
func Load(path string) (*VulnReport, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fusaops.ErrNoConfig
		}
		return nil, fmt.Errorf("vuln: read %s: %w", path, err)
	}
	var r VulnReport
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, fmt.Errorf("vuln: unmarshal %s: %w", path, err)
	}
	return &r, nil
}

// Render writes a representation of the vulnerability report to w.
// Supported formats: "text", "json".
//
//fusa:req REQ-FO-VULN004
func Render(w io.Writer, r *VulnReport, format string) error {
	switch format {
	case "text", "":
		return renderText(w, r)
	case "json":
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(r)
	default:
		return fmt.Errorf("vuln: unsupported format %q", format)
	}
}

func renderText(w io.Writer, r *VulnReport) error {
	fmt.Fprintf(w, "FuSaOps Vulnerability Scan\n")
	fmt.Fprintf(w, "==========================\n")
	fmt.Fprintf(w, "Project:   %s\n", r.ProjectRoot)
	fmt.Fprintf(w, "Tool:      %s v%s\n", r.Tool, r.ToolVersion)
	fmt.Fprintf(w, "Scanner:   %s", r.ScannerTool)
	if r.ScannerPresent {
		fmt.Fprintf(w, " (present)\n")
	} else {
		fmt.Fprintf(w, " (not found — install to enable scanning)\n")
	}
	fmt.Fprintf(w, "Generated: %s\n", r.GeneratedAt.Format("2006-01-02T15:04:05Z"))
	fmt.Fprintf(w, "Manifests: %d found\n", r.TotalManifests)
	fmt.Fprintf(w, "Findings:  %d (%d critical, %d high)\n\n",
		r.TotalFindings, r.CriticalCount, r.HighCount)

	if len(r.Manifests) > 0 {
		fmt.Fprintf(w, "Manifests:\n")
		for _, m := range r.Manifests {
			fmt.Fprintf(w, "  [%s] %s (%s)\n", m.Status, m.Path, m.Kind)
		}
		fmt.Fprintln(w)
	}

	if len(r.Findings) > 0 {
		fmt.Fprintf(w, "Vulnerabilities:\n")
		for _, f := range r.Findings {
			fmt.Fprintf(w, "  [%s] %s — %s@%s (%s)\n",
				f.Severity, f.VulnID, f.Package, f.Version, f.Ecosystem)
			if f.Summary != "" {
				fmt.Fprintf(w, "    %s\n", f.Summary)
			}
		}
		fmt.Fprintln(w)
	} else if r.ScannerPresent {
		fmt.Fprintf(w, "No vulnerabilities found.\n\n")
	}

	if r.Hash != "" {
		fmt.Fprintf(w, "Integrity: %s\n", r.Hash)
	}
	if r.TotalFindings > 0 {
		fmt.Fprintf(w, "\nVULNERABILITIES FOUND: %d (%d critical, %d high). Review and remediate.\n",
			r.TotalFindings, r.CriticalCount, r.HighCount)
	}
	return nil
}
