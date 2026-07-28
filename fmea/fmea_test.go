package fmea_test

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SoundMatt/FuSaOps/fmea"
)

//fusa:test REQ-FO-FMEA001
func TestConstants(t *testing.T) {
	if fmea.ReportFile == "" {
		t.Fatal("ReportFile must not be empty")
	}
	if !strings.HasSuffix(fmea.ReportFile, ".json") {
		t.Errorf("ReportFile should end in .json, got %q", fmea.ReportFile)
	}
	if fmea.HighRPNThreshold <= 0 {
		t.Errorf("HighRPNThreshold must be positive, got %d", fmea.HighRPNThreshold)
	}
}

//fusa:test REQ-FO-FMEA002
func TestBuildReturnsItems(t *testing.T) {
	f, err := fmea.Build(t.TempDir())
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if f.Summary.Total == 0 {
		t.Error("Build must return at least one failure mode")
	}
	if len(f.Entries) != f.Summary.Total {
		t.Errorf("len(Entries)=%d, Summary.Total=%d", len(f.Entries), f.Summary.Total)
	}
}

//fusa:test REQ-FO-FMEA002
func TestBuildRPNComputed(t *testing.T) {
	f, err := fmea.Build(t.TempDir())
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	for _, fm := range f.Entries {
		if fm.ID == "" {
			t.Error("failure mode has empty ID")
		}
		want := fm.Severity * fm.Occurrence * fm.Detection
		if fm.RPN != want {
			t.Errorf("%s: RPN=%d, want S×O×D=%d×%d×%d=%d",
				fm.ID, fm.RPN, fm.Severity, fm.Occurrence, fm.Detection, want)
		}
		if fm.RPN <= 0 {
			t.Errorf("%s: RPN must be positive", fm.ID)
		}
		if len(fm.Controls) == 0 {
			t.Errorf("%s: controls must not be empty", fm.ID)
		}
		if fm.Action == "" {
			t.Errorf("%s: action must not be empty", fm.ID)
		}
		if fm.Item == "" {
			t.Errorf("%s: item must not be empty", fm.ID)
		}
		if fm.ActionPriority == "" {
			t.Errorf("%s: actionPriority must not be empty", fm.ID)
		}
		if len(fm.RequirementIDs) == 0 {
			t.Errorf("%s: requirementIds must not be empty", fm.ID)
		}
	}
}

//fusa:test REQ-FO-FMEA002
func TestBuildHighRPNCounter(t *testing.T) {
	f, err := fmea.Build(t.TempDir())
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	var count int
	for _, fm := range f.Entries {
		if fm.RPN > fmea.HighRPNThreshold {
			count++
		}
	}
	if f.Summary.HighPriority != count {
		t.Errorf("Summary.HighPriority=%d, counted %d", f.Summary.HighPriority, count)
	}
}

//fusa:test REQ-FO-FMEA002
func TestBuildMetadata(t *testing.T) {
	root := t.TempDir()
	f, err := fmea.Build(root)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if f.ProjectRoot != root {
		t.Errorf("ProjectRoot=%q, want %q", f.ProjectRoot, root)
	}
	if f.Tool != "fusaops" {
		t.Errorf("Tool=%q, want fusaops", f.Tool)
	}
	if f.ToolVersion == "" {
		t.Error("ToolVersion must not be empty")
	}
	if f.Standard == "" {
		t.Error("Standard must not be empty")
	}
	if f.Hash == "" {
		t.Error("Hash must not be empty after Build")
	}
	if f.GeneratedAt.IsZero() {
		t.Error("GeneratedAt must not be zero")
	}
}

// TestBuildHashHasAlgoPrefix verifies Hash carries the "sha256:" prefix
// required by x-FuSa spec §2.7 for a field named "hash" (as opposed to a
// field named "sha256", which stays bare hex).
//
//fusa:test REQ-FO-FMEA006
func TestBuildHashHasAlgoPrefix(t *testing.T) {
	f, err := fmea.Build(t.TempDir())
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if !strings.HasPrefix(f.Hash, "sha256:") {
		t.Errorf("Hash = %q, want sha256: prefix", f.Hash)
	}
}

// TestAttestationContentHashStableAcrossRebuild verifies
// AttestationContentHash is deterministic across two Build calls (same
// content should hash the same, since it excludes GeneratedAt).
//
//fusa:test REQ-FO-FMEA007
func TestAttestationContentHashStableAcrossRebuild(t *testing.T) {
	dir := t.TempDir()
	f1, err := fmea.Build(dir)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	f2, err := fmea.Build(dir)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	h1 := fmea.AttestationContentHash(f1)
	h2 := fmea.AttestationContentHash(f2)
	if h1 == "" {
		t.Error("AttestationContentHash must not be empty")
	}
	if h1 != h2 {
		t.Errorf("AttestationContentHash differs across identical builds (GeneratedAt not excluded?): %q != %q", h1, h2)
	}
}

// TestSummaryCoverageMetrics verifies Summary's coverage fields are
// populated and internally consistent.
//
//fusa:test REQ-FO-FMEA008
func TestSummaryCoverageMetrics(t *testing.T) {
	f, err := fmea.Build(t.TempDir())
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if f.Summary.ComponentsInProject != fmea.ComponentsInProject {
		t.Errorf("ComponentsInProject = %d, want %d", f.Summary.ComponentsInProject, fmea.ComponentsInProject)
	}
	if f.Summary.ComponentsAnalyzed != f.Summary.Total {
		t.Errorf("ComponentsAnalyzed = %d, want %d (Summary.Total)", f.Summary.ComponentsAnalyzed, f.Summary.Total)
	}
	want := 100 * float64(f.Summary.ComponentsAnalyzed) / float64(f.Summary.ComponentsInProject)
	if diff := f.Summary.CoveragePct - want; diff > 0.05 || diff < -0.05 {
		t.Errorf("CoveragePct = %v, want ~%v", f.Summary.CoveragePct, want)
	}
	if f.Summary.ComponentInventoryMethod == "" {
		t.Error("ComponentInventoryMethod must be documented, not empty")
	}
}

//fusa:test REQ-FO-FMEA002
func TestBuildHashChanges(t *testing.T) {
	f1, _ := fmea.Build(t.TempDir())
	f2, _ := fmea.Build(t.TempDir())
	if f1.Hash == f2.Hash {
		t.Error("expected hashes to differ when ProjectRoot differs")
	}
}

//fusa:test REQ-FO-FMEA002
//fusa:test REQ-FO-FMEA005
func TestHasHighRPN(t *testing.T) {
	f, err := fmea.Build(t.TempDir())
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	want := f.Summary.HighPriority > 0
	if f.HasHighRPN() != want {
		t.Errorf("HasHighRPN()=%v, want %v (Summary.HighPriority=%d)",
			f.HasHighRPN(), want, f.Summary.HighPriority)
	}
}

//fusa:test REQ-FO-FMEA003
func TestSaveLoad(t *testing.T) {
	dir := t.TempDir()
	f, err := fmea.Build(dir)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	path := filepath.Join(dir, fmea.ReportFile)
	if saveErr := fmea.Save(path, f); saveErr != nil {
		t.Fatalf("Save: %v", saveErr)
	}
	loaded, err := fmea.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.Hash != f.Hash {
		t.Errorf("Hash mismatch after round-trip: got %q, want %q", loaded.Hash, f.Hash)
	}
	if loaded.Summary.Total != f.Summary.Total {
		t.Errorf("Summary.Total mismatch: got %d, want %d", loaded.Summary.Total, f.Summary.Total)
	}
}

//fusa:test REQ-FO-FMEA003
func TestLoadMissing(t *testing.T) {
	_, err := fmea.Load(filepath.Join(t.TempDir(), "missing.json"))
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

// TestLoadReadError verifies Load returns an error for a non-IsNotExist read
// failure (e.g. the path is a directory rather than a file).
//
//fusa:test REQ-FO-FMEA003
func TestLoadReadError(t *testing.T) {
	_, err := fmea.Load(t.TempDir())
	if err == nil {
		t.Fatal("expected error when path is a directory")
	}
}

//fusa:test REQ-FO-FMEA003
func TestLoadBadJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.json")
	if err := os.WriteFile(path, []byte("{bad json"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := fmea.Load(path)
	if err == nil {
		t.Fatal("expected error for malformed JSON")
	}
}

//fusa:test REQ-FO-FMEA004
func TestRenderText(t *testing.T) {
	f, _ := fmea.Build(t.TempDir())
	var buf bytes.Buffer
	if err := fmea.Render(&buf, f, "text"); err != nil {
		t.Fatalf("Render text: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "dFMEA") {
		t.Error("text output should contain 'dFMEA'")
	}
	if !strings.Contains(out, "FM-001") {
		t.Error("text output should contain FM-001")
	}
	if !strings.Contains(out, "RPN=") {
		t.Error("text output should contain RPN values")
	}
}

//fusa:test REQ-FO-FMEA004
func TestRenderJSON(t *testing.T) {
	f, _ := fmea.Build(t.TempDir())
	var buf bytes.Buffer
	if err := fmea.Render(&buf, f, "json"); err != nil {
		t.Fatalf("Render json: %v", err)
	}
	var got fmea.FMEA
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("JSON unmarshal: %v", err)
	}
	if got.Summary.Total != f.Summary.Total {
		t.Errorf("Summary.Total=%d, want %d", got.Summary.Total, f.Summary.Total)
	}
}

//fusa:test REQ-FO-FMEA004
func TestRenderDefault(t *testing.T) {
	f, _ := fmea.Build(t.TempDir())
	var buf bytes.Buffer
	if err := fmea.Render(&buf, f, ""); err != nil {
		t.Fatalf("Render default: %v", err)
	}
	if buf.Len() == 0 {
		t.Error("default render must produce output")
	}
}

//fusa:test REQ-FO-FMEA004
func TestRenderUnknownFormat(t *testing.T) {
	f, _ := fmea.Build(t.TempDir())
	err := fmea.Render(io.Discard, f, "xml")
	if err == nil {
		t.Fatal("expected error for unknown format")
	}
}

// TestRenderTextLowRPN verifies the "LOW" priority label (RPN ≤ 50) is rendered.
//
//fusa:test REQ-FO-FMEA004
func TestRenderTextLowRPN(t *testing.T) {
	f := &fmea.FMEA{
		Entries: []fmea.FailureMode{
			{ID: "FM-LOW", Component: "comp", Mode: "test", RPN: 10},
		},
	}
	var buf bytes.Buffer
	if err := fmea.Render(&buf, f, "text"); err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(buf.String(), "LOW") {
		t.Errorf("expected LOW priority label for RPN=10, got:\n%s", buf.String())
	}
}

// TestSaveWriteError verifies Save returns an error when the parent directory
// does not exist.
//
//fusa:test REQ-FO-FMEA003
func TestSaveWriteError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing", "out.json")
	if err := fmea.Save(path, &fmea.FMEA{}); err == nil {
		t.Error("Save: expected error for non-existent parent directory")
	}
}
