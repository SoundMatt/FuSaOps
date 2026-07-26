package tara_test

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SoundMatt/FuSaOps/tara"
)

//fusa:test REQ-FO-TARA001
func TestConstants(t *testing.T) {
	if tara.ReportFile == "" {
		t.Fatal("ReportFile must not be empty")
	}
	if !strings.HasSuffix(tara.ReportFile, ".json") {
		t.Errorf("ReportFile should end in .json, got %q", tara.ReportFile)
	}
}

//fusa:test REQ-FO-TARA001
func TestImpactValues(t *testing.T) {
	for _, imp := range []tara.Impact{
		tara.ImpactCritical, tara.ImpactMajor,
		tara.ImpactModerate, tara.ImpactNegligible,
	} {
		if string(imp) == "" {
			t.Errorf("Impact constant must not be empty")
		}
	}
}

//fusa:test REQ-FO-TARA001
func TestFeasibilityValues(t *testing.T) {
	for _, f := range []tara.Feasibility{
		tara.FeasibilityHigh, tara.FeasibilityMedium,
		tara.FeasibilityLow, tara.FeasibilityVeryLow,
	} {
		if string(f) == "" {
			t.Errorf("Feasibility constant must not be empty")
		}
	}
}

//fusa:test REQ-FO-TARA001
func TestRiskLevelValues(t *testing.T) {
	for _, r := range []tara.RiskLevel{
		tara.RiskCritical, tara.RiskHigh, tara.RiskMedium, tara.RiskLow,
	} {
		if string(r) == "" {
			t.Errorf("RiskLevel constant must not be empty")
		}
	}
}

//fusa:test REQ-FO-TARA001
func TestTreatmentValues(t *testing.T) {
	for _, tr := range []tara.TreatmentDecision{
		tara.TreatmentMitigate, tara.TreatmentTransfer,
		tara.TreatmentAvoid, tara.TreatmentAccept,
	} {
		if string(tr) == "" {
			t.Errorf("TreatmentDecision constant must not be empty")
		}
	}
}

//fusa:test REQ-FO-TARA002
func TestBuildReturnsScenarios(t *testing.T) {
	tr, err := tara.Build(t.TempDir())
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if tr.TotalScenarios == 0 {
		t.Error("Build must return at least one scenario")
	}
	if len(tr.Scenarios) != tr.TotalScenarios {
		t.Errorf("len(Scenarios)=%d, TotalScenarios=%d", len(tr.Scenarios), tr.TotalScenarios)
	}
}

//fusa:test REQ-FO-TARA002
func TestBuildScenarioIDs(t *testing.T) {
	tr, err := tara.Build(t.TempDir())
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	seen := map[string]bool{}
	for _, s := range tr.Scenarios {
		if s.ID == "" {
			t.Error("scenario has empty ID")
		}
		if seen[s.ID] {
			t.Errorf("duplicate scenario ID %q", s.ID)
		}
		seen[s.ID] = true
		if s.Asset == "" {
			t.Errorf("%s: asset must not be empty", s.ID)
		}
		if s.RiskLevel == "" {
			t.Errorf("%s: RiskLevel must not be empty", s.ID)
		}
		if len(s.Controls) == 0 {
			t.Errorf("%s: controls must not be empty", s.ID)
		}
	}
}

//fusa:test REQ-FO-TARA002
func TestBuildCounters(t *testing.T) {
	tr, err := tara.Build(t.TempDir())
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	var critCount, highCount int
	for _, s := range tr.Scenarios {
		switch s.RiskLevel {
		case tara.RiskCritical:
			critCount++
		case tara.RiskHigh:
			highCount++
		}
	}
	if tr.CriticalScenarios != critCount {
		t.Errorf("CriticalScenarios=%d, counted %d", tr.CriticalScenarios, critCount)
	}
	if tr.HighScenarios != highCount {
		t.Errorf("HighScenarios=%d, counted %d", tr.HighScenarios, highCount)
	}
}

//fusa:test REQ-FO-TARA002
func TestBuildMetadata(t *testing.T) {
	root := t.TempDir()
	tr, err := tara.Build(root)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if tr.ProjectRoot != root {
		t.Errorf("ProjectRoot=%q, want %q", tr.ProjectRoot, root)
	}
	if tr.Tool != "fusaops" {
		t.Errorf("Tool=%q, want fusaops", tr.Tool)
	}
	if tr.ToolVersion == "" {
		t.Error("ToolVersion must not be empty")
	}
	if tr.Standard == "" {
		t.Error("Standard must not be empty")
	}
	if tr.Hash == "" {
		t.Error("Hash must not be empty after Build")
	}
	if tr.GeneratedAt.IsZero() {
		t.Error("GeneratedAt must not be zero")
	}
}

//fusa:test REQ-FO-TARA002
func TestBuildHashChanges(t *testing.T) {
	tr1, _ := tara.Build(t.TempDir())
	tr2, _ := tara.Build(t.TempDir())
	// hashes differ because ProjectRoot differs
	if tr1.Hash == tr2.Hash {
		t.Error("expected hashes to differ when ProjectRoot differs")
	}
}

//fusa:test REQ-FO-TARA002
func TestHasCritical(t *testing.T) {
	tr, err := tara.Build(t.TempDir())
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	want := tr.CriticalScenarios > 0
	if tr.HasCritical() != want {
		t.Errorf("HasCritical()=%v, want %v (CriticalScenarios=%d)",
			tr.HasCritical(), want, tr.CriticalScenarios)
	}
}

//fusa:test REQ-FO-TARA003
func TestSaveLoad(t *testing.T) {
	dir := t.TempDir()
	tr, err := tara.Build(dir)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	path := filepath.Join(dir, tara.ReportFile)
	if saveErr := tara.Save(path, tr); saveErr != nil {
		t.Fatalf("Save: %v", saveErr)
	}
	loaded, err := tara.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.Hash != tr.Hash {
		t.Errorf("Hash mismatch after round-trip: got %q, want %q", loaded.Hash, tr.Hash)
	}
	if loaded.TotalScenarios != tr.TotalScenarios {
		t.Errorf("TotalScenarios mismatch: got %d, want %d", loaded.TotalScenarios, tr.TotalScenarios)
	}
}

//fusa:test REQ-FO-TARA003
func TestLoadMissing(t *testing.T) {
	_, err := tara.Load(filepath.Join(t.TempDir(), "missing.json"))
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

//fusa:test REQ-FO-TARA003
func TestLoadBadJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.json")
	if err := os.WriteFile(path, []byte("{bad json"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := tara.Load(path)
	if err == nil {
		t.Fatal("expected error for malformed JSON")
	}
}

//fusa:test REQ-FO-TARA004
func TestRenderText(t *testing.T) {
	tr, _ := tara.Build(t.TempDir())
	var buf bytes.Buffer
	if err := tara.Render(&buf, tr, "text"); err != nil {
		t.Fatalf("Render text: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "TARA") {
		t.Error("text output should contain 'TARA'")
	}
	if !strings.Contains(out, "TS-001") {
		t.Error("text output should contain scenario TS-001")
	}
}

//fusa:test REQ-FO-TARA004
func TestRenderJSON(t *testing.T) {
	tr, _ := tara.Build(t.TempDir())
	var buf bytes.Buffer
	if err := tara.Render(&buf, tr, "json"); err != nil {
		t.Fatalf("Render json: %v", err)
	}
	var got tara.TARA
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("JSON unmarshal: %v", err)
	}
	if got.TotalScenarios != tr.TotalScenarios {
		t.Errorf("TotalScenarios=%d, want %d", got.TotalScenarios, tr.TotalScenarios)
	}
}

//fusa:test REQ-FO-TARA004
func TestRenderDefault(t *testing.T) {
	tr, _ := tara.Build(t.TempDir())
	var buf bytes.Buffer
	if err := tara.Render(&buf, tr, ""); err != nil {
		t.Fatalf("Render default: %v", err)
	}
	if buf.Len() == 0 {
		t.Error("default render must produce output")
	}
}

//fusa:test REQ-FO-TARA004
func TestRenderUnknownFormat(t *testing.T) {
	tr, _ := tara.Build(t.TempDir())
	err := tara.Render(io.Discard, tr, "xml")
	if err == nil {
		t.Fatal("expected error for unknown format")
	}
}

// TestSaveWriteError verifies Save returns an error when the parent directory
// does not exist.
//
//fusa:test REQ-FO-TARA003
func TestSaveWriteError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing", "out.json")
	if err := tara.Save(path, &tara.TARA{}); err == nil {
		t.Error("Save: expected error for non-existent parent directory")
	}
}
