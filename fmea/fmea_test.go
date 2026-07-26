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
	if f.TotalItems == 0 {
		t.Error("Build must return at least one failure mode")
	}
	if len(f.FailureModes) != f.TotalItems {
		t.Errorf("len(FailureModes)=%d, TotalItems=%d", len(f.FailureModes), f.TotalItems)
	}
}

//fusa:test REQ-FO-FMEA002
func TestBuildRPNComputed(t *testing.T) {
	f, err := fmea.Build(t.TempDir())
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	for _, fm := range f.FailureModes {
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
	}
}

//fusa:test REQ-FO-FMEA002
func TestBuildHighRPNCounter(t *testing.T) {
	f, err := fmea.Build(t.TempDir())
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	var count int
	for _, fm := range f.FailureModes {
		if fm.RPN > fmea.HighRPNThreshold {
			count++
		}
	}
	if f.HighRPNItems != count {
		t.Errorf("HighRPNItems=%d, counted %d", f.HighRPNItems, count)
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

//fusa:test REQ-FO-FMEA002
func TestBuildHashChanges(t *testing.T) {
	f1, _ := fmea.Build(t.TempDir())
	f2, _ := fmea.Build(t.TempDir())
	if f1.Hash == f2.Hash {
		t.Error("expected hashes to differ when ProjectRoot differs")
	}
}

//fusa:test REQ-FO-FMEA002
func TestHasHighRPN(t *testing.T) {
	f, err := fmea.Build(t.TempDir())
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	want := f.HighRPNItems > 0
	if f.HasHighRPN() != want {
		t.Errorf("HasHighRPN()=%v, want %v (HighRPNItems=%d)",
			f.HasHighRPN(), want, f.HighRPNItems)
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
	if loaded.TotalItems != f.TotalItems {
		t.Errorf("TotalItems mismatch: got %d, want %d", loaded.TotalItems, f.TotalItems)
	}
}

//fusa:test REQ-FO-FMEA003
func TestLoadMissing(t *testing.T) {
	_, err := fmea.Load(filepath.Join(t.TempDir(), "missing.json"))
	if err == nil {
		t.Fatal("expected error for missing file")
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
	if got.TotalItems != f.TotalItems {
		t.Errorf("TotalItems=%d, want %d", got.TotalItems, f.TotalItems)
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
		FailureModes: []fmea.FailureMode{
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
