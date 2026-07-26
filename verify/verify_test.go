package verify_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SoundMatt/FuSaOps/verify"
)

//fusa:test REQ-FO-VER001
func TestTypes(t *testing.T) {
	r := verify.TestResult{
		Name:    "TestFoo",
		Package: "github.com/SoundMatt/FuSaOps/verify",
		Status:  verify.StatusPass,
		Elapsed: 0.001,
	}
	if r.Name != "TestFoo" || r.Status != verify.StatusPass {
		t.Errorf("TestResult fields not set: %+v", r)
	}
	s := verify.Summary{Total: 3, Passed: 2, Failed: 0, Skipped: 1}
	if s.Total != 3 || s.Skipped != 1 {
		t.Errorf("Summary fields not set: %+v", s)
	}
	b := verify.New("/tmp", []verify.TestResult{r})
	if b.ProjectRoot != "/tmp" || b.GoVersion == "" || b.GeneratedAt.IsZero() || len(b.Results) != 1 || b.Summary.Total != 1 {
		t.Errorf("Bundle fields not set: %+v", b)
	}
}

//fusa:test REQ-FO-VER002
func TestParse(t *testing.T) {
	input := `{"Action":"run","Test":"TestA","Package":"pkg/a"}
{"Action":"pass","Test":"TestA","Package":"pkg/a","Elapsed":0.001}
{"Action":"run","Test":"TestB","Package":"pkg/a"}
{"Action":"fail","Test":"TestB","Package":"pkg/a","Elapsed":0.002}
{"Action":"run","Test":"TestC","Package":"pkg/b"}
{"Action":"skip","Test":"TestC","Package":"pkg/b","Elapsed":0.000}
{"Action":"pass","Package":"pkg/a","Elapsed":0.003}
`
	results, err := verify.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("got %d results, want 3", len(results))
	}
	if results[0].Status != verify.StatusPass || results[0].Name != "TestA" {
		t.Errorf("result[0] = %+v, want pass TestA", results[0])
	}
	if results[1].Status != verify.StatusFail || results[1].Name != "TestB" {
		t.Errorf("result[1] = %+v, want fail TestB", results[1])
	}
	if results[2].Status != verify.StatusSkip || results[2].Name != "TestC" {
		t.Errorf("result[2] = %+v, want skip TestC", results[2])
	}
}

//fusa:test REQ-FO-VER002
func TestParseEmpty(t *testing.T) {
	results, err := verify.Parse(strings.NewReader(""))
	if err != nil {
		t.Fatalf("Parse empty: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

//fusa:test REQ-FO-VER002
func TestParseInvalidJSON(t *testing.T) {
	_, err := verify.Parse(strings.NewReader("not json\n"))
	if err == nil {
		t.Fatal("expected error on invalid JSON, got nil")
	}
}

//fusa:test REQ-FO-VER002
func TestSummarise(t *testing.T) {
	results := []verify.TestResult{
		{Status: verify.StatusPass},
		{Status: verify.StatusPass},
		{Status: verify.StatusFail},
		{Status: verify.StatusSkip},
	}
	s := verify.Summarise(results)
	if s.Total != 4 || s.Passed != 2 || s.Failed != 1 || s.Skipped != 1 {
		t.Errorf("Summarise = %+v, want {4 2 1 1}", s)
	}
}

//fusa:test REQ-FO-VER002
func TestSummariseEmpty(t *testing.T) {
	s := verify.Summarise(nil)
	if s.Total != 0 {
		t.Errorf("expected zero summary, got %+v", s)
	}
}

//fusa:test REQ-FO-VER004
func TestSaveLoad(t *testing.T) {
	dir := t.TempDir()
	b := verify.New(dir, []verify.TestResult{
		{Name: "TestX", Package: "pkg/x", Status: verify.StatusPass, Elapsed: 0.01},
	})

	path := filepath.Join(dir, verify.BundleFile)
	if err := verify.Save(path, b); err != nil {
		t.Fatalf("Save: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	var check verify.Bundle
	if unmarshalErr := json.Unmarshal(data, &check); unmarshalErr != nil {
		t.Fatalf("Unmarshal: %v", unmarshalErr)
	}

	loaded, err := verify.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.ProjectRoot != dir {
		t.Errorf("ProjectRoot = %q, want %q", loaded.ProjectRoot, dir)
	}
	if loaded.Summary.Total != 1 || loaded.Summary.Passed != 1 {
		t.Errorf("Summary = %+v, want {1 1 0 0}", loaded.Summary)
	}
}

//fusa:test REQ-FO-VER004
func TestLoadMissing(t *testing.T) {
	dir := t.TempDir()
	_, err := verify.Load(filepath.Join(dir, verify.BundleFile))
	if err == nil {
		t.Fatal("expected error on missing file, got nil")
	}
}

// TestLoadReadError verifies Load returns an error for a non-IsNotExist read
// failure (e.g. the path is a directory rather than a file).
//
//fusa:test REQ-FO-VER004
func TestLoadReadError(t *testing.T) {
	_, err := verify.Load(t.TempDir())
	if err == nil {
		t.Fatal("expected error when path is a directory")
	}
}

//fusa:test REQ-FO-VER004
func TestLoadBadJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, verify.BundleFile)
	if err := os.WriteFile(path, []byte("not json"), 0o640); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	_, err := verify.Load(path)
	if err == nil {
		t.Fatal("expected error on bad JSON, got nil")
	}
}

//fusa:test REQ-FO-VER004
func TestNew(t *testing.T) {
	results := []verify.TestResult{
		{Name: "TestA", Status: verify.StatusPass},
		{Name: "TestB", Status: verify.StatusFail},
	}
	b := verify.New("/my/project", results)
	if b.ProjectRoot != "/my/project" {
		t.Errorf("ProjectRoot = %q, want /my/project", b.ProjectRoot)
	}
	if b.GoVersion == "" {
		t.Error("GoVersion should be set")
	}
	if b.GeneratedAt.IsZero() {
		t.Error("GeneratedAt should be set")
	}
	if b.Summary.Total != 2 || b.Summary.Failed != 1 {
		t.Errorf("Summary = %+v, want {2 1 1 0}", b.Summary)
	}
}

//fusa:test REQ-FO-VER005
func TestRenderText(t *testing.T) {
	b := verify.New("/proj", []verify.TestResult{
		{Name: "TestPass", Package: "pkg/a", Status: verify.StatusPass, Elapsed: 0.1},
		{Name: "TestFail", Package: "pkg/b", Status: verify.StatusFail, Elapsed: 0.2},
	})
	var buf bytes.Buffer
	if err := verify.Render(&buf, b, "text"); err != nil {
		t.Fatalf("Render text: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "2 total") {
		t.Errorf("expected '2 total' in output:\n%s", out)
	}
	if !strings.Contains(out, "TestFail") {
		t.Errorf("expected failed test name in output:\n%s", out)
	}
}

//fusa:test REQ-FO-VER005
func TestRenderTextNoFailures(t *testing.T) {
	b := verify.New("/proj", []verify.TestResult{
		{Name: "TestA", Package: "pkg/a", Status: verify.StatusPass},
	})
	var buf bytes.Buffer
	if err := verify.Render(&buf, b, "text"); err != nil {
		t.Fatalf("Render text: %v", err)
	}
	out := buf.String()
	if strings.Contains(out, "Failed tests") {
		t.Errorf("should not show 'Failed tests' section when none failed:\n%s", out)
	}
}

//fusa:test REQ-FO-VER005
func TestRenderJSON(t *testing.T) {
	b := verify.New("/proj", []verify.TestResult{
		{Name: "TestA", Status: verify.StatusPass},
	})
	var buf bytes.Buffer
	if err := verify.Render(&buf, b, "json"); err != nil {
		t.Fatalf("Render json: %v", err)
	}
	var check verify.Bundle
	if err := json.Unmarshal(buf.Bytes(), &check); err != nil {
		t.Fatalf("Unmarshal json output: %v", err)
	}
	if check.ProjectRoot != "/proj" {
		t.Errorf("ProjectRoot = %q, want /proj", check.ProjectRoot)
	}
}

//fusa:test REQ-FO-VER005
func TestRenderDefaultIsText(t *testing.T) {
	b := verify.New("/proj", nil)
	var buf bytes.Buffer
	if err := verify.Render(&buf, b, ""); err != nil {
		t.Fatalf("Render default: %v", err)
	}
	if !strings.Contains(buf.String(), "Tests:") {
		t.Errorf("expected 'Tests:' in text output:\n%s", buf.String())
	}
}

//fusa:test REQ-FO-VER005
func TestRenderUnknownFormat(t *testing.T) {
	b := verify.New("/proj", nil)
	var buf bytes.Buffer
	if err := verify.Render(&buf, b, "xml"); err == nil {
		t.Fatal("expected error for unknown format, got nil")
	}
}

// TestSaveWriteError verifies Save returns an error when the parent directory
// does not exist.
//
//fusa:test REQ-FO-VER004
func TestSaveWriteError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing", "bundle.json")
	if err := verify.Save(path, verify.New("/proj", nil)); err == nil {
		t.Error("Save: expected error for non-existent parent directory")
	}
}
