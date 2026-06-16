package sas_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SoundMatt/FuSaOps/sas"
)

//fusa:test REQ-FO-SAS001
func TestConstants(t *testing.T) {
	if sas.ReportFile != ".fusaops-sas.json" {
		t.Errorf("ReportFile = %q", sas.ReportFile)
	}
}

//fusa:test REQ-FO-SAS001
func TestActivityStatuses(t *testing.T) {
	for _, st := range []sas.ActivityStatus{sas.StatusComplete, sas.StatusIncomplete, sas.StatusNA} {
		if string(st) == "" {
			t.Errorf("status is empty")
		}
	}
}

//fusa:test REQ-FO-SAS001
func TestTypes(t *testing.T) {
	a := sas.Activity{ID: "A-001", Title: "Planning", Status: sas.StatusComplete, Evidence: "evidence.json"}
	if a.ID != "A-001" || a.Status != sas.StatusComplete {
		t.Errorf("Activity fields: %+v", a)
	}
	s := &sas.SAS{Tool: "fusaops", SoftwareLevel: "DAL-C", TotalActivities: 5, CompleteActivities: 3}
	if !s.HasGaps() {
		t.Error("HasGaps should be true when incomplete activities remain")
	}
	s.CompleteActivities = 5
	if s.HasGaps() {
		t.Error("HasGaps should be false when all activities complete")
	}
}

//fusa:test REQ-FO-SAS002
func TestBuildEmptyDir(t *testing.T) {
	dir := t.TempDir()
	s, err := sas.Build(dir, "DAL-C")
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if s.Tool != "fusaops" {
		t.Errorf("Tool = %q, want fusaops", s.Tool)
	}
	if s.SoftwareLevel != "DAL-C" {
		t.Errorf("SoftwareLevel = %q, want DAL-C", s.SoftwareLevel)
	}
	if s.ProjectRoot != dir {
		t.Errorf("ProjectRoot = %q, want %q", s.ProjectRoot, dir)
	}
	if s.TotalActivities == 0 {
		t.Error("TotalActivities should be > 0")
	}
	if s.Hash == "" {
		t.Error("Hash should be set")
	}
	if s.GeneratedAt.IsZero() {
		t.Error("GeneratedAt should be set")
	}
	// A-001 (planning) has no evidence file → N/A → still counted as complete.
	// Activities with evidence files but no artefacts → incomplete.
	if s.CompleteActivities == s.TotalActivities {
		// Some activities should be incomplete in an empty dir.
		t.Error("some activities should be incomplete in empty dir")
	}
}

//fusa:test REQ-FO-SAS002
func TestBuildDefaultLevel(t *testing.T) {
	dir := t.TempDir()
	s, err := sas.Build(dir, "")
	if err != nil {
		t.Fatalf("Build with empty level: %v", err)
	}
	if s.SoftwareLevel != "DAL-C" {
		t.Errorf("default SoftwareLevel = %q, want DAL-C", s.SoftwareLevel)
	}
}

//fusa:test REQ-FO-SAS002
func TestBuildWithEvidence(t *testing.T) {
	dir := t.TempDir()
	evidenceFiles := []string{
		".fusaops-trace.json",
		".fusaops-evidence.json",
		".fusaops-qualify-report.json",
		".fusaops-sci.json",
		".fusaops-safety-case.json",
		".fusaops-problems.json",
		"sbom.json",
		"artifact-manifest.json",
	}
	for _, f := range evidenceFiles {
		if err := os.WriteFile(filepath.Join(dir, f), []byte(`{}`), 0o640); err != nil {
			t.Fatalf("WriteFile %s: %v", f, err)
		}
	}

	s, err := sas.Build(dir, "DAL-B")
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if s.HasGaps() {
		t.Errorf("should have no gaps with all evidence files present; complete=%d total=%d",
			s.CompleteActivities, s.TotalActivities)
	}
}

//fusa:test REQ-FO-SAS002
func TestBuildHashChanges(t *testing.T) {
	dir := t.TempDir()
	s1, _ := sas.Build(dir, "DAL-C")
	if err := os.WriteFile(filepath.Join(dir, ".fusaops-evidence.json"),
		[]byte(`{}`), 0o640); err != nil {
		t.Fatal(err)
	}
	s2, _ := sas.Build(dir, "DAL-C")
	if s1.Hash == s2.Hash {
		t.Error("Hash should differ after adding evidence")
	}
}

//fusa:test REQ-FO-SAS003
func TestSaveLoad(t *testing.T) {
	dir := t.TempDir()
	s, err := sas.Build(dir, "DAL-C")
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	path := filepath.Join(dir, sas.ReportFile)
	if saveErr := sas.Save(path, s); saveErr != nil {
		t.Fatalf("Save: %v", saveErr)
	}
	got, loadErr := sas.Load(path)
	if loadErr != nil {
		t.Fatalf("Load: %v", loadErr)
	}
	if got.SoftwareLevel != s.SoftwareLevel {
		t.Errorf("SoftwareLevel = %q, want %q", got.SoftwareLevel, s.SoftwareLevel)
	}
	if got.TotalActivities != s.TotalActivities {
		t.Errorf("TotalActivities = %d, want %d", got.TotalActivities, s.TotalActivities)
	}
	if got.Hash != s.Hash {
		t.Errorf("Hash mismatch: %q vs %q", got.Hash, s.Hash)
	}
}

//fusa:test REQ-FO-SAS003
func TestLoadMissing(t *testing.T) {
	_, err := sas.Load("/nonexistent/.fusaops-sas.json")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

//fusa:test REQ-FO-SAS003
func TestLoadBadJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(path, []byte("not json"), 0o640); err != nil {
		t.Fatal(err)
	}
	_, err := sas.Load(path)
	if err == nil {
		t.Fatal("expected error for bad JSON, got nil")
	}
}

//fusa:test REQ-FO-SAS004
func TestRenderText(t *testing.T) {
	dir := t.TempDir()
	s, _ := sas.Build(dir, "DAL-C")
	var buf bytes.Buffer
	if err := sas.Render(&buf, s, "text"); err != nil {
		t.Fatalf("Render text: %v", err)
	}
	out := buf.String()
	for _, want := range []string{"Software Accomplishment Summary", "DO-178C", "DAL-C", "A-001", "INCOMPLETE"} {
		if !strings.Contains(out, want) {
			t.Errorf("text output missing %q:\n%s", want, out)
		}
	}
}

//fusa:test REQ-FO-SAS004
func TestRenderTextAllComplete(t *testing.T) {
	dir := t.TempDir()
	for _, f := range []string{
		".fusaops-trace.json", ".fusaops-evidence.json",
		".fusaops-qualify-report.json", ".fusaops-sci.json",
		".fusaops-safety-case.json", ".fusaops-problems.json",
		"sbom.json", "artifact-manifest.json",
	} {
		if err := os.WriteFile(filepath.Join(dir, f), []byte(`{}`), 0o640); err != nil {
			t.Fatalf("WriteFile %s: %v", f, err)
		}
	}
	s, _ := sas.Build(dir, "DAL-C")
	var buf bytes.Buffer
	if err := sas.Render(&buf, s, "text"); err != nil {
		t.Fatalf("Render text all-complete: %v", err)
	}
	out := buf.String()
	if strings.Contains(out, "INCOMPLETE") {
		t.Errorf("should not show INCOMPLETE when all activities done:\n%s", out)
	}
}

//fusa:test REQ-FO-SAS004
func TestRenderJSON(t *testing.T) {
	dir := t.TempDir()
	s, _ := sas.Build(dir, "DAL-B")
	var buf bytes.Buffer
	if err := sas.Render(&buf, s, "json"); err != nil {
		t.Fatalf("Render json: %v", err)
	}
	var check sas.SAS
	if err := json.Unmarshal(buf.Bytes(), &check); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if check.SoftwareLevel != "DAL-B" {
		t.Errorf("SoftwareLevel = %q, want DAL-B", check.SoftwareLevel)
	}
	if check.TotalActivities != s.TotalActivities {
		t.Errorf("TotalActivities = %d, want %d", check.TotalActivities, s.TotalActivities)
	}
}

//fusa:test REQ-FO-SAS004
func TestRenderDefault(t *testing.T) {
	dir := t.TempDir()
	s, _ := sas.Build(dir, "DAL-C")
	var buf bytes.Buffer
	if err := sas.Render(&buf, s, ""); err != nil {
		t.Fatalf("Render default: %v", err)
	}
	if !strings.Contains(buf.String(), "Software Accomplishment Summary") {
		t.Error("default format should produce text output")
	}
}

//fusa:test REQ-FO-SAS004
func TestRenderUnknownFormat(t *testing.T) {
	dir := t.TempDir()
	s, _ := sas.Build(dir, "DAL-C")
	var buf bytes.Buffer
	if err := sas.Render(&buf, s, "xml"); err == nil {
		t.Fatal("expected error for unknown format, got nil")
	}
}
