package vuln_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SoundMatt/FuSaOps/vuln"
)

// fakeRunner returns a no-error response to --version and JSON output for scan.
func fakeRunnerNoVulns(args ...string) ([]byte, error) {
	if len(args) >= 2 && args[1] == "--version" {
		return []byte("osv-scanner version 1.0.0"), nil
	}
	return []byte(`{"results":[]}`), nil
}

// fakeRunnerWithVulns simulates osv-scanner finding two vulnerabilities.
func fakeRunnerWithVulns(args ...string) ([]byte, error) {
	if len(args) >= 2 && args[1] == "--version" {
		return []byte("osv-scanner version 1.0.0"), nil
	}
	out := `{
  "results": [
    {
      "source": {"path": "/tmp/go.mod"},
      "packages": [
        {
          "package": {"name": "github.com/example/pkg", "version": "1.2.3", "ecosystem": "Go"},
          "vulnerabilities": [
            {
              "id": "GHSA-abcd-1234-5678",
              "summary": "Example critical vulnerability",
              "database_specific": {"severity": "CRITICAL"}
            },
            {
              "id": "GHSA-efgh-5678-9012",
              "summary": "Example high vulnerability",
              "database_specific": {"severity": "HIGH"}
            }
          ]
        }
      ]
    }
  ]
}`
	return []byte(out), nil
}

// fakeRunnerAbsent simulates osv-scanner not being installed.
func fakeRunnerAbsent(args ...string) ([]byte, error) {
	return nil, fmt.Errorf("exec: %q: executable file not found in $PATH", args[0])
}

//fusa:test REQ-FO-VULN001
func TestConstants(t *testing.T) {
	if vuln.ReportFile == "" {
		t.Fatal("ReportFile must not be empty")
	}
	if !strings.HasSuffix(vuln.ReportFile, ".json") {
		t.Errorf("ReportFile should end in .json, got %q", vuln.ReportFile)
	}
	if vuln.ScannerBinary == "" {
		t.Fatal("ScannerBinary must not be empty")
	}
}

//fusa:test REQ-FO-VULN001
func TestManifestKinds(t *testing.T) {
	for _, k := range []vuln.ManifestKind{
		vuln.ManifestGoMod, vuln.ManifestCargoToml,
		vuln.ManifestRequirementsTxt, vuln.ManifestPackageJSON, vuln.ManifestPomXML,
	} {
		if string(k) == "" {
			t.Errorf("ManifestKind constant must not be empty")
		}
	}
}

//fusa:test REQ-FO-VULN002
func TestScanEmptyDirNoScanner(t *testing.T) {
	r, err := vuln.Scan(t.TempDir(), fakeRunnerAbsent)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if r.TotalManifests != 0 {
		t.Errorf("TotalManifests=%d, want 0", r.TotalManifests)
	}
	if r.ScannerPresent {
		t.Error("ScannerPresent must be false when runner returns error")
	}
}

//fusa:test REQ-FO-VULN002
func TestScanEmptyDirWithScanner(t *testing.T) {
	r, err := vuln.Scan(t.TempDir(), fakeRunnerNoVulns)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if !r.ScannerPresent {
		t.Error("ScannerPresent must be true when runner succeeds")
	}
	if r.TotalFindings != 0 {
		t.Errorf("TotalFindings=%d, want 0", r.TotalFindings)
	}
}

//fusa:test REQ-FO-VULN002
func TestScanDiscoversGoMod(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	r, err := vuln.Scan(dir, fakeRunnerNoVulns)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if r.TotalManifests != 1 {
		t.Errorf("TotalManifests=%d, want 1", r.TotalManifests)
	}
	if r.Manifests[0].Kind != vuln.ManifestGoMod {
		t.Errorf("Kind=%q, want %q", r.Manifests[0].Kind, vuln.ManifestGoMod)
	}
	if r.Manifests[0].Status != vuln.StatusScanned {
		t.Errorf("Status=%q, want %q", r.Manifests[0].Status, vuln.StatusScanned)
	}
}

//fusa:test REQ-FO-VULN002
func TestScanDiscoversMultipleManifests(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"go.mod", "Cargo.toml", "requirements.txt"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(""), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	r, err := vuln.Scan(dir, fakeRunnerAbsent)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if r.TotalManifests != 3 {
		t.Errorf("TotalManifests=%d, want 3", r.TotalManifests)
	}
	for _, m := range r.Manifests {
		if m.Status != vuln.StatusSkipped {
			t.Errorf("Status=%q, want %q (scanner absent)", m.Status, vuln.StatusSkipped)
		}
	}
}

//fusa:test REQ-FO-VULN002
func TestScanFindsVulnerabilities(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	r, err := vuln.Scan(dir, fakeRunnerWithVulns)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if r.TotalFindings != 2 {
		t.Errorf("TotalFindings=%d, want 2", r.TotalFindings)
	}
	if r.CriticalCount != 1 {
		t.Errorf("CriticalCount=%d, want 1", r.CriticalCount)
	}
	if r.HighCount != 1 {
		t.Errorf("HighCount=%d, want 1", r.HighCount)
	}
	if !r.HasFindings() {
		t.Error("HasFindings() must return true when TotalFindings > 0")
	}
}

//fusa:test REQ-FO-VULN002
func TestScanMetadata(t *testing.T) {
	root := t.TempDir()
	r, err := vuln.Scan(root, fakeRunnerAbsent)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if r.ProjectRoot != root {
		t.Errorf("ProjectRoot=%q, want %q", r.ProjectRoot, root)
	}
	if r.Tool != "fusaops" {
		t.Errorf("Tool=%q, want fusaops", r.Tool)
	}
	if r.ToolVersion == "" {
		t.Error("ToolVersion must not be empty")
	}
	if r.ScannerTool != vuln.ScannerBinary {
		t.Errorf("ScannerTool=%q, want %q", r.ScannerTool, vuln.ScannerBinary)
	}
	if r.Hash == "" {
		t.Error("Hash must not be empty after Scan")
	}
	if r.GeneratedAt.IsZero() {
		t.Error("GeneratedAt must not be zero")
	}
}

//fusa:test REQ-FO-VULN002
func TestHasFindings(t *testing.T) {
	r, _ := vuln.Scan(t.TempDir(), fakeRunnerAbsent)
	if r.HasFindings() {
		t.Error("HasFindings() must return false when no findings")
	}
}

//fusa:test REQ-FO-VULN003
func TestSaveLoad(t *testing.T) {
	dir := t.TempDir()
	r, err := vuln.Scan(dir, fakeRunnerAbsent)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	path := filepath.Join(dir, vuln.ReportFile)
	if saveErr := vuln.Save(path, r); saveErr != nil {
		t.Fatalf("Save: %v", saveErr)
	}
	loaded, err := vuln.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.Hash != r.Hash {
		t.Errorf("Hash mismatch after round-trip: got %q, want %q", loaded.Hash, r.Hash)
	}
}

//fusa:test REQ-FO-VULN003
func TestLoadMissing(t *testing.T) {
	_, err := vuln.Load(filepath.Join(t.TempDir(), "missing.json"))
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

// TestLoadReadError verifies Load returns an error for a non-IsNotExist read
// failure (e.g. the path is a directory rather than a file).
//
//fusa:test REQ-FO-VULN003
func TestLoadReadError(t *testing.T) {
	_, err := vuln.Load(t.TempDir())
	if err == nil {
		t.Fatal("expected error when path is a directory")
	}
}

//fusa:test REQ-FO-VULN003
func TestLoadBadJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.json")
	if err := os.WriteFile(path, []byte("{bad json"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := vuln.Load(path)
	if err == nil {
		t.Fatal("expected error for malformed JSON")
	}
}

//fusa:test REQ-FO-VULN004
func TestRenderTextNoScanner(t *testing.T) {
	r, _ := vuln.Scan(t.TempDir(), fakeRunnerAbsent)
	var buf bytes.Buffer
	if err := vuln.Render(&buf, r, "text"); err != nil {
		t.Fatalf("Render text: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "Vulnerability Scan") {
		t.Error("text output should contain 'Vulnerability Scan'")
	}
	if !strings.Contains(out, "not found") {
		t.Error("text output should note that scanner is not found")
	}
}

//fusa:test REQ-FO-VULN004
func TestRenderTextWithFindings(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	r, _ := vuln.Scan(dir, fakeRunnerWithVulns)
	var buf bytes.Buffer
	if err := vuln.Render(&buf, r, "text"); err != nil {
		t.Fatalf("Render text with findings: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "GHSA-abcd-1234-5678") {
		t.Error("text output should contain vuln ID")
	}
	if !strings.Contains(out, "VULNERABILITIES FOUND") {
		t.Error("text output should warn about findings")
	}
}

//fusa:test REQ-FO-VULN004
func TestRenderJSON(t *testing.T) {
	r, _ := vuln.Scan(t.TempDir(), fakeRunnerAbsent)
	var buf bytes.Buffer
	if err := vuln.Render(&buf, r, "json"); err != nil {
		t.Fatalf("Render json: %v", err)
	}
	var got vuln.VulnReport
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("JSON unmarshal: %v", err)
	}
	if got.Tool != "fusaops" {
		t.Errorf("Tool=%q, want fusaops", got.Tool)
	}
}

//fusa:test REQ-FO-VULN004
func TestRenderDefault(t *testing.T) {
	r, _ := vuln.Scan(t.TempDir(), fakeRunnerAbsent)
	var buf bytes.Buffer
	if err := vuln.Render(&buf, r, ""); err != nil {
		t.Fatalf("Render default: %v", err)
	}
	if buf.Len() == 0 {
		t.Error("default render must produce output")
	}
}

//fusa:test REQ-FO-VULN004
func TestRenderUnknownFormat(t *testing.T) {
	r, _ := vuln.Scan(t.TempDir(), fakeRunnerAbsent)
	err := vuln.Render(io.Discard, r, "xml")
	if err == nil {
		t.Fatal("expected error for unknown format")
	}
}

// TestSaveWriteError verifies Save returns an error when the parent directory
// does not exist.
//
//fusa:test REQ-FO-VULN003
func TestSaveWriteError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing", "out.json")
	if err := vuln.Save(path, &vuln.VulnReport{}); err == nil {
		t.Error("Save: expected error for non-existent parent directory")
	}
}

// TestScanSkipsVendorDir verifies that Scan (and discoverManifests underneath)
// skips vendor/ directories, covering the filepath.SkipDir return branch.
//
//fusa:test REQ-FO-VULN001
func TestScanSkipsVendorDir(t *testing.T) {
	dir := t.TempDir()
	// Root-level manifest — should be found.
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Manifest inside vendor/ — must be skipped.
	vendorDir := filepath.Join(dir, "vendor")
	if err := os.Mkdir(vendorDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(vendorDir, "go.mod"), []byte("module nested\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	rep, err := vuln.Scan(dir, fakeRunnerAbsent)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if rep.TotalManifests != 1 {
		t.Errorf("TotalManifests: want 1 (vendor skipped), got %d", rep.TotalManifests)
	}
}
