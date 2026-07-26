package slsa

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

//fusa:test REQ-FO-SLSA001
func TestLevelTypes(t *testing.T) {
	if LevelL1 != "L1" || LevelL2 != "L2" || LevelL3 != "L3" || LevelL4 != "L4" {
		t.Error("level constants wrong")
	}
}

//fusa:test REQ-FO-SLSA002
func TestAssessEmptyDir(t *testing.T) {
	dir := t.TempDir()
	rep, err := Assess(dir, "test", LevelL1)
	if err != nil {
		t.Fatalf("Assess: %v", err)
	}
	if rep.Project != "test" || rep.Level != LevelL1 {
		t.Errorf("unexpected report header: %+v", rep)
	}
	if len(rep.Objectives) == 0 {
		t.Error("no objectives in report")
	}
}

//fusa:test REQ-FO-SLSA002
func TestAssessGitPresent(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0o700); err != nil {
		t.Fatal(err)
	}
	rep, err := Assess(dir, "proj", LevelL1)
	if err != nil {
		t.Fatalf("Assess: %v", err)
	}
	for _, obj := range rep.Objectives {
		if obj.ID == "SLSA-L1.1" && obj.Status != "PASS" {
			t.Errorf("SLSA-L1.1 should PASS with .git present, got %s", obj.Status)
		}
	}
}

//fusa:test REQ-FO-SLSA002
func TestAssessModuleFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module foo"), 0o600); err != nil {
		t.Fatal(err)
	}
	rep, err := Assess(dir, "proj", LevelL1)
	if err != nil {
		t.Fatalf("Assess: %v", err)
	}
	for _, obj := range rep.Objectives {
		if obj.ID == "SLSA-L1.2" && obj.Status != "PASS" {
			t.Errorf("SLSA-L1.2 should PASS with go.mod, got %s: %s", obj.Status, obj.Note)
		}
	}
}

//fusa:test REQ-FO-SLSA002
func TestAssessL2NAforL1Target(t *testing.T) {
	dir := t.TempDir()
	rep, err := Assess(dir, "proj", LevelL1)
	if err != nil {
		t.Fatalf("Assess: %v", err)
	}
	for _, obj := range rep.Objectives {
		if obj.ID == "SLSA-L2.1" && obj.Status != "N/A" {
			t.Errorf("SLSA-L2.1 should be N/A for L1 target, got %s", obj.Status)
		}
	}
}

//fusa:test REQ-FO-SLSA002
func TestAssessProvenanceFields(t *testing.T) {
	dir := t.TempDir()
	prov := `{"builder":"github-actions","vcsRevision":"abc123"}`
	if err := os.WriteFile(filepath.Join(dir, "provenance.json"), []byte(prov), 0o600); err != nil {
		t.Fatal(err)
	}
	rep, err := Assess(dir, "proj", LevelL2)
	if err != nil {
		t.Fatalf("Assess: %v", err)
	}
	for _, obj := range rep.Objectives {
		switch obj.ID {
		case "SLSA-L2.1", "SLSA-L2.2":
			if obj.Status != "PASS" {
				t.Errorf("%s should PASS with provenance fields set, got %s: %s", obj.ID, obj.Status, obj.Note)
			}
		}
	}
}

//fusa:test REQ-FO-SLSA002
func TestAssessCODEOWNERS(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "CODEOWNERS"), []byte("* @owner"), 0o600); err != nil {
		t.Fatal(err)
	}
	rep, err := Assess(dir, "proj", LevelL3)
	if err != nil {
		t.Fatalf("Assess: %v", err)
	}
	for _, obj := range rep.Objectives {
		if obj.ID == "SLSA-L3.1" && obj.Status != "PASS" {
			t.Errorf("SLSA-L3.1 should PASS with CODEOWNERS, got %s", obj.Status)
		}
	}
}

//fusa:test REQ-FO-SLSA003
func TestRenderText(t *testing.T) {
	rep, _ := Assess(t.TempDir(), "myproj", LevelL1)
	var buf bytes.Buffer
	if err := Render(&buf, rep, "text"); err != nil {
		t.Fatalf("Render text: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "myproj") || !strings.Contains(out, "SLSA") {
		t.Errorf("text output missing expected content: %q", out[:min(len(out), 300)])
	}
}

//fusa:test REQ-FO-SLSA003
func TestRenderJSON(t *testing.T) {
	rep, _ := Assess(t.TempDir(), "myproj", LevelL2)
	var buf bytes.Buffer
	if err := Render(&buf, rep, "json"); err != nil {
		t.Fatalf("Render json: %v", err)
	}
	var out Report
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if out.Project != "myproj" {
		t.Errorf("project = %q", out.Project)
	}
}

//fusa:test REQ-FO-SLSA003
func TestRenderUnknownFormat(t *testing.T) {
	rep, _ := Assess(t.TempDir(), "proj", LevelL1)
	err := Render(&bytes.Buffer{}, rep, "xml")
	if err == nil || !strings.Contains(err.Error(), "unsupported") {
		t.Errorf("expected unsupported format error, got %v", err)
	}
}

//fusa:test REQ-FO-SLSA002
func TestAssessSBOMHashesWithPackages(t *testing.T) {
	dir := t.TempDir()
	sbomJSON := `{"packages":[{"name":"dep","version":"v1"}]}`
	if err := os.WriteFile(filepath.Join(dir, "sbom.json"), []byte(sbomJSON), 0o600); err != nil {
		t.Fatal(err)
	}
	rep, err := Assess(dir, "proj", LevelL3)
	if err != nil {
		t.Fatalf("Assess: %v", err)
	}
	for _, obj := range rep.Objectives {
		if obj.ID == "SLSA-L3.2" && obj.Status != "PASS" {
			t.Errorf("SLSA-L3.2 should PASS with packages in sbom.json, got %s: %s", obj.Status, obj.Note)
		}
	}
}

//fusa:test REQ-FO-SLSA002
func TestAssessSBOMHashesMalformedJSON(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "sbom.json"), []byte("not-json{{{"), 0o600); err != nil {
		t.Fatal(err)
	}
	rep, err := Assess(dir, "proj", LevelL3)
	if err != nil {
		t.Fatalf("Assess: %v", err)
	}
	for _, obj := range rep.Objectives {
		if obj.ID == "SLSA-L3.2" {
			if obj.Status != "GAP" {
				t.Errorf("SLSA-L3.2 should be GAP for malformed sbom.json, got %s", obj.Status)
			}
			if !strings.Contains(obj.Note, "not valid JSON") {
				t.Errorf("SLSA-L3.2 note should mention invalid JSON, got %q", obj.Note)
			}
		}
	}
}

//fusa:test REQ-FO-SLSA002
func TestAssessSBOMHashesNoPackages(t *testing.T) {
	dir := t.TempDir()
	// Valid JSON but no packages/components/dependencies arrays.
	if err := os.WriteFile(filepath.Join(dir, "sbom.json"), []byte(`{"version":"1.0"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	rep, err := Assess(dir, "proj", LevelL3)
	if err != nil {
		t.Fatalf("Assess: %v", err)
	}
	for _, obj := range rep.Objectives {
		if obj.ID == "SLSA-L3.2" {
			if obj.Status != "GAP" {
				t.Errorf("SLSA-L3.2 should be GAP when sbom.json has no packages, got %s", obj.Status)
			}
		}
	}
}
