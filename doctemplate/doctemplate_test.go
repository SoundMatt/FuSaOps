package doctemplate_test

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SoundMatt/FuSaOps/doctemplate"
)

//fusa:test REQ-FO-TMPL001
func TestConstants(t *testing.T) {
	if doctemplate.ReportFile == "" {
		t.Fatal("ReportFile must not be empty")
	}
	if !strings.HasSuffix(doctemplate.ReportFile, ".json") {
		t.Errorf("ReportFile should end in .json, got %q", doctemplate.ReportFile)
	}
	if doctemplate.DefaultOutputDir == "" {
		t.Fatal("DefaultOutputDir must not be empty")
	}
}

//fusa:test REQ-FO-TMPL001
func TestDocKinds(t *testing.T) {
	for _, k := range []doctemplate.DocKind{
		doctemplate.DocKindPlan, doctemplate.DocKindRequirements,
		doctemplate.DocKindReport, doctemplate.DocKindChecklist,
	} {
		if string(k) == "" {
			t.Errorf("DocKind constant must not be empty")
		}
	}
}

//fusa:test REQ-FO-TMPL002
func TestGenerateAllTemplates(t *testing.T) {
	dir := t.TempDir()
	outDir := filepath.Join(dir, "safety-docs")
	r, err := doctemplate.Generate(dir, outDir, nil)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if r.TotalGenerated == 0 {
		t.Error("Generate must produce at least one template")
	}
	if len(r.Generated) != r.TotalGenerated {
		t.Errorf("len(Generated)=%d, TotalGenerated=%d", len(r.Generated), r.TotalGenerated)
	}
	// verify files exist on disk
	for _, d := range r.Generated {
		if _, err := os.Stat(d.Path); err != nil {
			t.Errorf("generated file missing: %v", err)
		}
		if d.Size == 0 {
			t.Errorf("%s: generated file must not be empty", d.TemplateID)
		}
	}
}

//fusa:test REQ-FO-TMPL002
func TestGenerateWithStandardFilter(t *testing.T) {
	dir := t.TempDir()
	outDir := filepath.Join(dir, "docs")
	r, err := doctemplate.Generate(dir, outDir, []string{"ISO 21434"})
	if err != nil {
		t.Fatalf("Generate ISO 21434: %v", err)
	}
	if r.TotalGenerated == 0 {
		t.Error("Generate with ISO 21434 must produce at least one template")
	}
	// TARA template should be included
	var hasTARA bool
	for _, d := range r.Generated {
		if strings.Contains(strings.ToLower(d.Name), "tara") {
			hasTARA = true
		}
	}
	if !hasTARA {
		t.Error("ISO 21434 filter must include the TARA template")
	}
	// HARA (ISO 26262 / IEC 61508 specific) should not be included
	for _, d := range r.Generated {
		if strings.Contains(strings.ToLower(d.Name), "hara") {
			t.Errorf("ISO 21434 filter must not include HARA template, got %q", d.Name)
		}
	}
}

//fusa:test REQ-FO-TMPL002
func TestGenerateCreatesOutputDir(t *testing.T) {
	dir := t.TempDir()
	outDir := filepath.Join(dir, "new", "nested", "dir")
	_, err := doctemplate.Generate(dir, outDir, nil)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if _, err := os.Stat(outDir); err != nil {
		t.Errorf("output dir not created: %v", err)
	}
}

//fusa:test REQ-FO-TMPL002
func TestGenerateMetadata(t *testing.T) {
	root := t.TempDir()
	outDir := filepath.Join(root, "docs")
	r, err := doctemplate.Generate(root, outDir, nil)
	if err != nil {
		t.Fatalf("Generate: %v", err)
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
	if r.OutputDir != outDir {
		t.Errorf("OutputDir=%q, want %q", r.OutputDir, outDir)
	}
	if r.Hash == "" {
		t.Error("Hash must not be empty after Generate")
	}
	if r.GeneratedAt.IsZero() {
		t.Error("GeneratedAt must not be zero")
	}
}

//fusa:test REQ-FO-TMPL002
func TestGenerateDO178C(t *testing.T) {
	dir := t.TempDir()
	outDir := filepath.Join(dir, "docs")
	r, err := doctemplate.Generate(dir, outDir, []string{"DO-178C"})
	if err != nil {
		t.Fatalf("Generate DO-178C: %v", err)
	}
	// SCI and SAS templates should be included
	names := make([]string, 0, len(r.Generated))
	for _, d := range r.Generated {
		names = append(names, strings.ToLower(d.Name))
	}
	nameset := strings.Join(names, "|")
	if !strings.Contains(nameset, "configuration index") {
		t.Error("DO-178C filter must include Software Configuration Index template")
	}
	if !strings.Contains(nameset, "accomplishment summary") {
		t.Error("DO-178C filter must include Software Accomplishment Summary template")
	}
}

//fusa:test REQ-FO-TMPL003
func TestSaveLoad(t *testing.T) {
	dir := t.TempDir()
	outDir := filepath.Join(dir, "docs")
	r, err := doctemplate.Generate(dir, outDir, nil)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	path := filepath.Join(dir, doctemplate.ReportFile)
	if saveErr := doctemplate.Save(path, r); saveErr != nil {
		t.Fatalf("Save: %v", saveErr)
	}
	loaded, err := doctemplate.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.Hash != r.Hash {
		t.Errorf("Hash mismatch after round-trip: got %q, want %q", loaded.Hash, r.Hash)
	}
	if loaded.TotalGenerated != r.TotalGenerated {
		t.Errorf("TotalGenerated mismatch: got %d, want %d", loaded.TotalGenerated, r.TotalGenerated)
	}
}

//fusa:test REQ-FO-TMPL003
func TestLoadMissing(t *testing.T) {
	_, err := doctemplate.Load(filepath.Join(t.TempDir(), "missing.json"))
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

// TestLoadReadError verifies Load returns an error for a non-IsNotExist read
// failure (e.g. the path is a directory rather than a file).
//
//fusa:test REQ-FO-TMPL003
func TestLoadReadError(t *testing.T) {
	_, err := doctemplate.Load(t.TempDir())
	if err == nil {
		t.Fatal("expected error when path is a directory")
	}
}

//fusa:test REQ-FO-TMPL003
func TestLoadBadJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.json")
	if err := os.WriteFile(path, []byte("{bad json"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := doctemplate.Load(path)
	if err == nil {
		t.Fatal("expected error for malformed JSON")
	}
}

//fusa:test REQ-FO-TMPL004
func TestRenderText(t *testing.T) {
	dir := t.TempDir()
	r, _ := doctemplate.Generate(dir, filepath.Join(dir, "docs"), nil)
	var buf bytes.Buffer
	if err := doctemplate.Render(&buf, r, "text"); err != nil {
		t.Fatalf("Render text: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "Safety Documentation") {
		t.Error("text output should contain 'Safety Documentation'")
	}
	if !strings.Contains(out, "SP-001") {
		t.Error("text output should contain SP-001")
	}
}

//fusa:test REQ-FO-TMPL004
func TestRenderJSON(t *testing.T) {
	dir := t.TempDir()
	r, _ := doctemplate.Generate(dir, filepath.Join(dir, "docs"), nil)
	var buf bytes.Buffer
	if err := doctemplate.Render(&buf, r, "json"); err != nil {
		t.Fatalf("Render json: %v", err)
	}
	var got doctemplate.Report
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("JSON unmarshal: %v", err)
	}
	if got.TotalGenerated != r.TotalGenerated {
		t.Errorf("TotalGenerated=%d, want %d", got.TotalGenerated, r.TotalGenerated)
	}
}

//fusa:test REQ-FO-TMPL004
func TestRenderDefault(t *testing.T) {
	dir := t.TempDir()
	r, _ := doctemplate.Generate(dir, filepath.Join(dir, "docs"), nil)
	var buf bytes.Buffer
	if err := doctemplate.Render(&buf, r, ""); err != nil {
		t.Fatalf("Render default: %v", err)
	}
	if buf.Len() == 0 {
		t.Error("default render must produce output")
	}
}

//fusa:test REQ-FO-TMPL004
func TestRenderUnknownFormat(t *testing.T) {
	dir := t.TempDir()
	r, _ := doctemplate.Generate(dir, filepath.Join(dir, "docs"), nil)
	err := doctemplate.Render(io.Discard, r, "xml")
	if err == nil {
		t.Fatal("expected error for unknown format")
	}
}

// TestSaveWriteError verifies Save returns an error when the parent directory
// does not exist.
//
//fusa:test REQ-FO-TMPL003
func TestSaveWriteError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing", "out.json")
	if err := doctemplate.Save(path, &doctemplate.Report{}); err == nil {
		t.Error("Save: expected error for non-existent parent directory")
	}
}
