package sci_test

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	fusaops "github.com/SoundMatt/FuSaOps"
	"github.com/SoundMatt/FuSaOps/adapter"
	"github.com/SoundMatt/FuSaOps/sci"
)

// fakeAdapter satisfies adapter.Adapter for testing without requiring any
// x-FuSa binary on PATH.
type fakeAdapter struct {
	name      string
	lang      fusaops.Language
	tool      string
	available bool
}

func (f *fakeAdapter) Name() string                                                 { return f.name }
func (f *fakeAdapter) Language() fusaops.Language                                   { return f.lang }
func (f *fakeAdapter) Tool() string                                                 { return f.tool }
func (f *fakeAdapter) Detect(_ string) (bool, error)                                { return true, nil }
func (f *fakeAdapter) Available() bool                                              { return f.available }
func (f *fakeAdapter) Check(_ context.Context, _ string) ([]fusaops.Finding, error) { return nil, nil }

//fusa:test REQ-FO-SCI001
func TestConstants(t *testing.T) {
	if sci.ReportFile != ".fusaops-sci.json" {
		t.Errorf("ReportFile = %q", sci.ReportFile)
	}
}

//fusa:test REQ-FO-SCI001
func TestKinds(t *testing.T) {
	kinds := []sci.ItemKind{sci.KindTool, sci.KindArtefact, sci.KindComponent}
	for _, k := range kinds {
		if string(k) == "" {
			t.Errorf("kind is empty")
		}
	}
}

//fusa:test REQ-FO-SCI001
func TestTypes(t *testing.T) {
	item := sci.ConfigItem{
		ID:      "TOOL-001",
		Name:    "fusaops",
		Kind:    sci.KindTool,
		Version: "1.52.0",
		Present: true,
	}
	if item.ID != "TOOL-001" || !item.Present || item.Kind != sci.KindTool {
		t.Errorf("ConfigItem fields: %+v", item)
	}
	s := &sci.SCI{Tool: "fusaops", TotalItems: 5}
	if s.Tool != "fusaops" || s.TotalItems != 5 {
		t.Errorf("SCI fields: %+v", s)
	}
}

//fusa:test REQ-FO-SCI002
func TestBuildNoAdapters(t *testing.T) {
	dir := t.TempDir()
	s, err := sci.Build(dir, nil)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if s.Tool != "fusaops" {
		t.Errorf("Tool = %q, want fusaops", s.Tool)
	}
	if s.ProjectRoot != dir {
		t.Errorf("ProjectRoot = %q, want %q", s.ProjectRoot, dir)
	}
	if s.GoVersion == "" {
		t.Error("GoVersion should be set")
	}
	if s.TotalItems == 0 {
		t.Error("TotalItems should be > 0 (at least fusaops tool + artefact list)")
	}
	if s.Hash == "" {
		t.Error("Hash should be set")
	}
	if s.GeneratedAt.IsZero() {
		t.Error("GeneratedAt should be set")
	}
	// fusaops itself should be item TOOL-001.
	if len(s.Items) == 0 || s.Items[0].ID != "TOOL-001" {
		t.Errorf("first item should be TOOL-001 (fusaops), got %+v", s.Items[0])
	}
}

//fusa:test REQ-FO-SCI002
func TestBuildWithAdapters(t *testing.T) {
	dir := t.TempDir()
	fakes := []adapter.Adapter{
		&fakeAdapter{name: "go-FuSa", lang: fusaops.LangGo, tool: "gofusa", available: true},
		&fakeAdapter{name: "c-FuSa", lang: fusaops.LangC, tool: "cfusa", available: false},
	}

	s, buildErr := sci.Build(dir, fakes)
	if buildErr != nil {
		t.Fatalf("Build: %v", buildErr)
	}
	// Should include TOOL-001 (fusaops) + TOOL-002 (gofusa) + TOOL-003 (cfusa).
	toolCount := 0
	for _, item := range s.Items {
		if item.Kind == sci.KindTool {
			toolCount++
		}
	}
	if toolCount < 3 {
		t.Errorf("expected at least 3 tool items, got %d", toolCount)
	}
	// go-FuSa should be present; c-FuSa should not.
	for _, item := range s.Items {
		if item.Name == "gofusa" && !item.Present {
			t.Error("gofusa adapter should be marked present")
		}
		if item.Name == "cfusa" && item.Present {
			t.Error("cfusa adapter should be marked not present")
		}
	}
}

//fusa:test REQ-FO-SCI002
func TestBuildWithArtefacts(t *testing.T) {
	dir := t.TempDir()
	// Write some evidence files.
	if err := os.WriteFile(filepath.Join(dir, ".fusaops-evidence.json"),
		[]byte(`{"results":[]}`), 0o640); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "sbom.json"),
		[]byte(`{"packages":[]}`), 0o640); err != nil {
		t.Fatal(err)
	}

	s, err := sci.Build(dir, nil)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	presentCount := 0
	for _, item := range s.Items {
		if item.Kind == sci.KindArtefact && item.Present {
			presentCount++
			if item.SHA256 == "" || len(item.SHA256) != 64 {
				t.Errorf("artefact %s: expected 64-char SHA256, got %q", item.Name, item.SHA256)
			}
			if item.Size == 0 {
				t.Errorf("artefact %s: size should be > 0", item.Name)
			}
		}
	}
	if presentCount < 2 {
		t.Errorf("expected at least 2 present artefacts, got %d", presentCount)
	}
}

//fusa:test REQ-FO-SCI002
func TestBuildHashChanges(t *testing.T) {
	dir := t.TempDir()
	s1, _ := sci.Build(dir, nil)
	// Write an artefact and rebuild — hash must change.
	if err := os.WriteFile(filepath.Join(dir, ".fusaops-evidence.json"),
		[]byte(`{}`), 0o640); err != nil {
		t.Fatal(err)
	}
	s2, _ := sci.Build(dir, nil)
	if s1.Hash == s2.Hash {
		t.Error("Hash should differ after adding an artefact")
	}
}

//fusa:test REQ-FO-SCI003
func TestSaveLoad(t *testing.T) {
	dir := t.TempDir()
	s, err := sci.Build(dir, nil)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	path := filepath.Join(dir, sci.ReportFile)
	if saveErr := sci.Save(path, s); saveErr != nil {
		t.Fatalf("Save: %v", saveErr)
	}
	got, loadErr := sci.Load(path)
	if loadErr != nil {
		t.Fatalf("Load: %v", loadErr)
	}
	if got.Tool != s.Tool {
		t.Errorf("Tool = %q, want %q", got.Tool, s.Tool)
	}
	if got.TotalItems != s.TotalItems {
		t.Errorf("TotalItems = %d, want %d", got.TotalItems, s.TotalItems)
	}
	if got.Hash != s.Hash {
		t.Errorf("Hash mismatch: %q vs %q", got.Hash, s.Hash)
	}
}

//fusa:test REQ-FO-SCI003
func TestLoadMissing(t *testing.T) {
	_, err := sci.Load("/nonexistent/.fusaops-sci.json")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

// TestLoadReadError verifies Load returns an error for a non-IsNotExist read
// failure (e.g. the path is a directory rather than a file).
//
//fusa:test REQ-FO-SCI003
func TestLoadReadError(t *testing.T) {
	_, err := sci.Load(t.TempDir())
	if err == nil {
		t.Fatal("expected error when path is a directory")
	}
}

//fusa:test REQ-FO-SCI003
func TestLoadBadJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(path, []byte("not json"), 0o640); err != nil {
		t.Fatal(err)
	}
	_, err := sci.Load(path)
	if err == nil {
		t.Fatal("expected error for bad JSON, got nil")
	}
}

//fusa:test REQ-FO-SCI004
func TestRenderText(t *testing.T) {
	dir := t.TempDir()
	s, _ := sci.Build(dir, nil)
	var buf bytes.Buffer
	if err := sci.Render(&buf, s, "text"); err != nil {
		t.Fatalf("Render text: %v", err)
	}
	out := buf.String()
	for _, want := range []string{"Software Configuration Index", "DO-178C", "Tool Items", "TOOL-001", "fusaops"} {
		if !strings.Contains(out, want) {
			t.Errorf("text output missing %q:\n%s", want, out)
		}
	}
}

//fusa:test REQ-FO-SCI004
func TestRenderJSON(t *testing.T) {
	dir := t.TempDir()
	s, _ := sci.Build(dir, nil)
	var buf bytes.Buffer
	if err := sci.Render(&buf, s, "json"); err != nil {
		t.Fatalf("Render json: %v", err)
	}
	var check sci.SCI
	if err := json.Unmarshal(buf.Bytes(), &check); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if check.Tool != "fusaops" {
		t.Errorf("Tool = %q, want fusaops", check.Tool)
	}
	if check.TotalItems != s.TotalItems {
		t.Errorf("TotalItems = %d, want %d", check.TotalItems, s.TotalItems)
	}
}

//fusa:test REQ-FO-SCI004
func TestRenderDefault(t *testing.T) {
	dir := t.TempDir()
	s, _ := sci.Build(dir, nil)
	var buf bytes.Buffer
	if err := sci.Render(&buf, s, ""); err != nil {
		t.Fatalf("Render default: %v", err)
	}
	if !strings.Contains(buf.String(), "Software Configuration Index") {
		t.Error("default format should produce text output")
	}
}

//fusa:test REQ-FO-SCI004
func TestRenderUnknownFormat(t *testing.T) {
	dir := t.TempDir()
	s, _ := sci.Build(dir, nil)
	var buf bytes.Buffer
	if err := sci.Render(&buf, s, "xml"); err == nil {
		t.Fatal("expected error for unknown format, got nil")
	}
}

// TestRenderTextWithComponentItem verifies kindLabel(KindComponent) is exercised
// when an SCI contains a Component-kind item.
//
//fusa:test REQ-FO-SCI004
func TestRenderTextWithComponentItem(t *testing.T) {
	s := &sci.SCI{
		ProjectRoot: "/root",
		Tool:        "fusaops",
		ToolVersion: "1.0.0",
		GoVersion:   "go1.22",
		Items: []sci.ConfigItem{
			{ID: "COMP-001", Name: "mycomponent", Kind: sci.KindComponent, Present: true},
		},
	}
	var buf bytes.Buffer
	if err := sci.Render(&buf, s, "text"); err != nil {
		t.Fatalf("Render text: %v", err)
	}
	if !strings.Contains(buf.String(), "Component Items:") {
		t.Errorf("expected Component Items section in output:\n%s", buf.String())
	}
}

// TestSaveWriteError verifies Save returns an error when the parent directory
// does not exist.
//
//fusa:test REQ-FO-SCI003
func TestSaveWriteError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing", "out.json")
	if err := sci.Save(path, &sci.SCI{}); err == nil {
		t.Error("Save: expected error for non-existent parent directory")
	}
}
