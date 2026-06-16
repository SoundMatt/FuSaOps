package release_test

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	fusaops "github.com/SoundMatt/FuSaOps"
	"github.com/SoundMatt/FuSaOps/release"
)

//fusa:test REQ-FO-REL001
func TestConstants(t *testing.T) {
	if release.SBOMFile != "sbom.json" {
		t.Errorf("SBOMFile = %q, want sbom.json", release.SBOMFile)
	}
	if release.ProvenanceFile != "provenance.json" {
		t.Errorf("ProvenanceFile = %q, want provenance.json", release.ProvenanceFile)
	}
	if release.ManifestFile != "artifact-manifest.json" {
		t.Errorf("ManifestFile = %q, want artifact-manifest.json", release.ManifestFile)
	}
}

//fusa:test REQ-FO-REL001
func TestTypes(t *testing.T) {
	a := release.Artifact{Path: "/out/sbom.json", SHA256: "abc123", Size: 1024}
	if a.Path != "/out/sbom.json" || a.SHA256 != "abc123" || a.Size != 1024 {
		t.Errorf("Artifact fields not set: %+v", a)
	}
	p := release.Provenance{Tool: "fusaops", ToolVersion: fusaops.Version, GoVersion: fusaops.Version}
	if p.Tool != "fusaops" || p.ToolVersion != fusaops.Version || p.GoVersion != fusaops.Version {
		t.Errorf("Provenance fields not set: %+v", p)
	}
	m := &release.Manifest{Artifacts: []release.Artifact{a}}
	if len(m.Artifacts) != 1 {
		t.Errorf("Manifest.Artifacts length = %d, want 1", len(m.Artifacts))
	}
}

//fusa:test REQ-FO-REL002
func TestBuildProvenance(t *testing.T) {
	dir := t.TempDir()
	p, err := release.BuildProvenance(context.Background(), dir)
	if err != nil {
		t.Fatalf("BuildProvenance: %v", err)
	}
	if p.Tool != "fusaops" {
		t.Errorf("Tool = %q, want fusaops", p.Tool)
	}
	if p.ToolVersion == "" {
		t.Error("ToolVersion should be set")
	}
	if p.GoVersion == "" {
		t.Error("GoVersion should be set")
	}
	if p.GOOS == "" || p.GOARCH == "" {
		t.Errorf("GOOS/GOARCH not set: %q/%q", p.GOOS, p.GOARCH)
	}
	if p.ProjectRoot != dir {
		t.Errorf("ProjectRoot = %q, want %q", p.ProjectRoot, dir)
	}
	if p.GeneratedAt.IsZero() {
		t.Error("GeneratedAt should be set")
	}
}

//fusa:test REQ-FO-REL002
func TestBuildProvenanceGitRepo(t *testing.T) {
	// Run in FuSaOps repo dir — git should return a commit hash.
	p, err := release.BuildProvenance(context.Background(), ".")
	if err != nil {
		t.Fatalf("BuildProvenance: %v", err)
	}
	// VCSRevision may or may not be set depending on git availability,
	// but it should be a 40-char hex string when present.
	if p.VCSRevision != "" && len(p.VCSRevision) != 40 {
		t.Errorf("VCSRevision = %q, want 40-char hex or empty", p.VCSRevision)
	}
}

//fusa:test REQ-FO-REL003
func TestBuildManifest(t *testing.T) {
	dir := t.TempDir()
	paths := make([]string, 3)
	for i, name := range []string{"a.json", "b.json", "c.json"} {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte(`{"key":"value"}`), 0o640); err != nil {
			t.Fatalf("WriteFile %s: %v", name, err)
		}
		paths[i] = p
	}

	m, err := release.BuildManifest(paths)
	if err != nil {
		t.Fatalf("BuildManifest: %v", err)
	}
	if len(m.Artifacts) != 3 {
		t.Fatalf("artifacts = %d, want 3", len(m.Artifacts))
	}
	for _, a := range m.Artifacts {
		if len(a.SHA256) != 64 {
			t.Errorf("SHA256 = %q (len %d), want 64 hex chars", a.SHA256, len(a.SHA256))
		}
		if a.Size == 0 {
			t.Errorf("Size = 0 for %s", a.Path)
		}
	}
	if m.GeneratedAt.IsZero() {
		t.Error("GeneratedAt should be set")
	}
}

//fusa:test REQ-FO-REL003
func TestBuildManifestMissingFile(t *testing.T) {
	_, err := release.BuildManifest([]string{"/nonexistent/file.json"})
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

//fusa:test REQ-FO-REL003
func TestBuildManifestEmpty(t *testing.T) {
	m, err := release.BuildManifest(nil)
	if err != nil {
		t.Fatalf("BuildManifest(nil): %v", err)
	}
	if len(m.Artifacts) != 0 {
		t.Errorf("expected 0 artifacts, got %d", len(m.Artifacts))
	}
}

//fusa:test REQ-FO-REL004
func TestSaveJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.json")
	data := map[string]string{"key": "value"}
	if err := release.SaveJSON(path, data); err != nil {
		t.Fatalf("SaveJSON: %v", err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	var check map[string]string
	if unmarshalErr := json.Unmarshal(raw, &check); unmarshalErr != nil {
		t.Fatalf("Unmarshal: %v", unmarshalErr)
	}
	if check["key"] != "value" {
		t.Errorf("key = %q, want value", check["key"])
	}
}

//fusa:test REQ-FO-REL004
func TestRenderProvenanceText(t *testing.T) {
	p := &release.Provenance{
		Tool:        "fusaops",
		ToolVersion: "1.50.0",
		GoVersion:   "go1.22.0",
		GOOS:        "linux",
		GOARCH:      "amd64",
		VCSRevision: strings.Repeat("a", 40),
	}
	var buf bytes.Buffer
	if err := release.RenderProvenance(&buf, p, "text"); err != nil {
		t.Fatalf("RenderProvenance text: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "fusaops") {
		t.Errorf("expected 'fusaops' in output:\n%s", out)
	}
	if !strings.Contains(out, "go1.22.0") {
		t.Errorf("expected go version in output:\n%s", out)
	}
	if !strings.Contains(out, strings.Repeat("a", 40)) {
		t.Errorf("expected commit hash in output:\n%s", out)
	}
}

//fusa:test REQ-FO-REL004
func TestRenderProvenanceNoCommit(t *testing.T) {
	p := &release.Provenance{Tool: "fusaops", ToolVersion: "1.50.0", GoVersion: "go1.22.0", GOOS: "linux", GOARCH: "amd64"}
	var buf bytes.Buffer
	if err := release.RenderProvenance(&buf, p, "text"); err != nil {
		t.Fatalf("RenderProvenance text: %v", err)
	}
	if strings.Contains(buf.String(), "Commit:") {
		t.Error("should not show 'Commit:' when VCSRevision is empty")
	}
}

//fusa:test REQ-FO-REL004
func TestRenderProvenanceJSON(t *testing.T) {
	p := &release.Provenance{Tool: "fusaops", ToolVersion: "1.50.0"}
	var buf bytes.Buffer
	if err := release.RenderProvenance(&buf, p, "json"); err != nil {
		t.Fatalf("RenderProvenance json: %v", err)
	}
	var check release.Provenance
	if err := json.Unmarshal(buf.Bytes(), &check); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if check.Tool != "fusaops" {
		t.Errorf("Tool = %q, want fusaops", check.Tool)
	}
}

//fusa:test REQ-FO-REL004
func TestRenderProvenanceUnknownFormat(t *testing.T) {
	p := &release.Provenance{}
	var buf bytes.Buffer
	if err := release.RenderProvenance(&buf, p, "xml"); err == nil {
		t.Fatal("expected error for unknown format, got nil")
	}
}

//fusa:test REQ-FO-REL004
func TestRenderManifestText(t *testing.T) {
	m := &release.Manifest{
		Artifacts: []release.Artifact{
			{Path: "sbom.json", SHA256: strings.Repeat("b", 64), Size: 512},
			{Path: "provenance.json", SHA256: strings.Repeat("c", 64), Size: 256},
		},
	}
	var buf bytes.Buffer
	if err := release.RenderManifest(&buf, m, "text"); err != nil {
		t.Fatalf("RenderManifest text: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "sbom.json") {
		t.Errorf("expected sbom.json in output:\n%s", out)
	}
	if !strings.Contains(out, "2") {
		t.Errorf("expected artifact count in output:\n%s", out)
	}
}

//fusa:test REQ-FO-REL004
func TestRenderManifestJSON(t *testing.T) {
	m := &release.Manifest{Artifacts: []release.Artifact{{Path: "sbom.json", SHA256: "abc", Size: 10}}}
	var buf bytes.Buffer
	if err := release.RenderManifest(&buf, m, "json"); err != nil {
		t.Fatalf("RenderManifest json: %v", err)
	}
	var check release.Manifest
	if err := json.Unmarshal(buf.Bytes(), &check); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if len(check.Artifacts) != 1 {
		t.Errorf("artifacts = %d, want 1", len(check.Artifacts))
	}
}

//fusa:test REQ-FO-REL004
func TestRenderManifestUnknownFormat(t *testing.T) {
	m := &release.Manifest{}
	var buf bytes.Buffer
	if err := release.RenderManifest(&buf, m, "xml"); err == nil {
		t.Fatal("expected error for unknown format, got nil")
	}
}
