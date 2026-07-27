package safetycase_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SoundMatt/FuSaOps/safetycase"
)

//fusa:test REQ-FO-SC001
func TestConstants(t *testing.T) {
	if safetycase.ReportFile != ".fusaops-safety-case.json" {
		t.Errorf("ReportFile = %q", safetycase.ReportFile)
	}
}

//fusa:test REQ-FO-SC001
func TestStandards(t *testing.T) {
	stds := []safetycase.Standard{
		safetycase.StandardISO26262,
		safetycase.StandardDO178C,
		safetycase.StandardIEC61508,
		safetycase.StandardISO21434,
	}
	for _, s := range stds {
		if string(s) == "" {
			t.Errorf("standard is empty")
		}
	}
	if len(safetycase.ValidStandards) != len(stds) {
		t.Errorf("ValidStandards length = %d, want %d", len(safetycase.ValidStandards), len(stds))
	}
}

//fusa:test REQ-FO-SC001
//fusa:test REQ-FO-SC005
func TestTypes(t *testing.T) {
	e := safetycase.EvidenceRef{
		Title:  "Test evidence",
		Path:   "/tmp/evidence.json",
		Status: safetycase.StatusPresent,
		SHA256: "abc",
		Size:   100,
	}
	if e.Title != "Test evidence" || e.Status != safetycase.StatusPresent || e.Size != 100 {
		t.Errorf("EvidenceRef fields: %+v", e)
	}
	c := safetycase.Claim{ID: "C-001", Title: "test", Strategy: "by inspection", Evidence: []safetycase.EvidenceRef{e}, Passed: true}
	if c.ID != "C-001" || !c.Passed || len(c.Evidence) != 1 {
		t.Errorf("Claim fields: %+v", c)
	}
	sc := &safetycase.SafetyCase{Tool: "fusaops", Standard: safetycase.StandardISO26262, TotalClaims: 2, PassedClaims: 1}
	if !sc.HasGaps() {
		t.Error("HasGaps should be true when PassedClaims < TotalClaims")
	}
	sc.PassedClaims = 2
	if sc.HasGaps() {
		t.Error("HasGaps should be false when all claims pass")
	}
}

//fusa:test REQ-FO-SC002
func TestBuildEmptyDir(t *testing.T) {
	dir := t.TempDir()
	sc, err := safetycase.Build(dir, safetycase.StandardISO26262)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if sc.Tool != "fusaops" {
		t.Errorf("Tool = %q, want fusaops", sc.Tool)
	}
	if sc.Standard != safetycase.StandardISO26262 {
		t.Errorf("Standard = %q, want ISO 26262", sc.Standard)
	}
	if sc.ProjectRoot != dir {
		t.Errorf("ProjectRoot = %q, want %q", sc.ProjectRoot, dir)
	}
	if sc.TotalClaims == 0 {
		t.Error("TotalClaims should be > 0")
	}
	// Empty dir → no evidence → all claims fail.
	if sc.PassedClaims != 0 {
		t.Errorf("PassedClaims = %d in empty dir, want 0", sc.PassedClaims)
	}
	if !sc.HasGaps() {
		t.Error("HasGaps should be true in empty dir")
	}
	if sc.Hash == "" {
		t.Error("Hash should be set")
	}
	if sc.GeneratedAt.IsZero() {
		t.Error("GeneratedAt should be set")
	}
}

//fusa:test REQ-FO-SC002
func TestBuildWithEvidence(t *testing.T) {
	dir := t.TempDir()
	// Write the test evidence bundle so claim C-003 passes.
	if err := os.WriteFile(filepath.Join(dir, ".fusaops-evidence.json"),
		[]byte(`{"results":[]}`), 0o640); err != nil {
		t.Fatal(err)
	}
	// Write the qualify report so claim C-001 passes.
	if err := os.WriteFile(filepath.Join(dir, ".fusaops-qualify-report.json"),
		[]byte(`{"components":[]}`), 0o640); err != nil {
		t.Fatal(err)
	}

	sc, err := safetycase.Build(dir, safetycase.StandardDO178C)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if sc.PassedClaims < 2 {
		t.Errorf("PassedClaims = %d, want at least 2 (with 2 evidence files present)", sc.PassedClaims)
	}
	// Verify the specific claims that should pass.
	byID := map[string]safetycase.Claim{}
	for _, c := range sc.Claims {
		byID[c.ID] = c
	}
	if !byID["C-001"].Passed {
		t.Error("C-001 (tool qualification) should pass with qualify report present")
	}
	if !byID["C-003"].Passed {
		t.Error("C-003 (test evidence) should pass with evidence bundle present")
	}
}

//fusa:test REQ-FO-SC002
func TestBuildEvidenceHash(t *testing.T) {
	dir := t.TempDir()
	sc1, _ := safetycase.Build(dir, safetycase.StandardISO26262)
	// Write a file and rebuild — hash should change.
	if err := os.WriteFile(filepath.Join(dir, ".fusaops-evidence.json"),
		[]byte(`{"results":[]}`), 0o640); err != nil {
		t.Fatal(err)
	}
	sc2, _ := safetycase.Build(dir, safetycase.StandardISO26262)
	if sc1.Hash == sc2.Hash {
		t.Error("Hash should differ when evidence files change")
	}
}

//fusa:test REQ-FO-SC002
func TestBuildAllStandards(t *testing.T) {
	dir := t.TempDir()
	for _, std := range safetycase.ValidStandards {
		sc, err := safetycase.Build(dir, std)
		if err != nil {
			t.Errorf("Build(%s): %v", std, err)
			continue
		}
		if sc.Standard != std {
			t.Errorf("Standard = %q, want %q", sc.Standard, std)
		}
	}
}

//fusa:test REQ-FO-SC003
func TestSaveLoad(t *testing.T) {
	dir := t.TempDir()
	sc, err := safetycase.Build(dir, safetycase.StandardDO178C)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	path := filepath.Join(dir, safetycase.ReportFile)
	if saveErr := safetycase.Save(path, sc); saveErr != nil {
		t.Fatalf("Save: %v", saveErr)
	}
	got, err := safetycase.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.Standard != sc.Standard {
		t.Errorf("Standard = %q, want %q", got.Standard, sc.Standard)
	}
	if got.TotalClaims != sc.TotalClaims {
		t.Errorf("TotalClaims = %d, want %d", got.TotalClaims, sc.TotalClaims)
	}
	if got.Hash != sc.Hash {
		t.Errorf("Hash mismatch: %q vs %q", got.Hash, sc.Hash)
	}
}

//fusa:test REQ-FO-SC003
func TestLoadMissing(t *testing.T) {
	_, err := safetycase.Load("/nonexistent/.fusaops-safety-case.json")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

// TestLoadReadError verifies Load returns an error for a non-IsNotExist read
// failure (e.g. the path is a directory rather than a file).
//
//fusa:test REQ-FO-SC003
func TestLoadReadError(t *testing.T) {
	_, err := safetycase.Load(t.TempDir())
	if err == nil {
		t.Fatal("expected error when path is a directory")
	}
}

//fusa:test REQ-FO-SC003
func TestLoadBadJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(path, []byte("not json"), 0o640); err != nil {
		t.Fatal(err)
	}
	_, err := safetycase.Load(path)
	if err == nil {
		t.Fatal("expected error for bad JSON, got nil")
	}
}

//fusa:test REQ-FO-SC004
func TestRenderText(t *testing.T) {
	dir := t.TempDir()
	sc, _ := safetycase.Build(dir, safetycase.StandardISO26262)
	var buf bytes.Buffer
	if err := safetycase.Render(&buf, sc, "text"); err != nil {
		t.Fatalf("Render text: %v", err)
	}
	out := buf.String()
	for _, want := range []string{"FuSaOps Safety Case", "ISO 26262", "C-001", "GAPS DETECTED"} {
		if !strings.Contains(out, want) {
			t.Errorf("text output missing %q:\n%s", want, out)
		}
	}
}

//fusa:test REQ-FO-SC004
func TestRenderTextAllPass(t *testing.T) {
	dir := t.TempDir()
	// Write all evidence files so every claim passes.
	for _, fname := range []string{
		".fusaops-qualify-report.json",
		".fusaops-trace.json",
		".fusaops-evidence.json",
		"sbom.json",
		"provenance.json",
		"artifact-manifest.json",
		".fusaops-problems.json",
		"audit-pack.zip",
	} {
		path := filepath.Join(dir, fname)
		if err := os.WriteFile(path, []byte(`{}`), 0o640); err != nil {
			t.Fatalf("WriteFile %s: %v", fname, err)
		}
	}
	sc, _ := safetycase.Build(dir, safetycase.StandardISO26262)
	var buf bytes.Buffer
	if err := safetycase.Render(&buf, sc, "text"); err != nil {
		t.Fatalf("Render text all-pass: %v", err)
	}
	out := buf.String()
	if strings.Contains(out, "GAPS DETECTED") {
		t.Errorf("should not show GAPS when all claims pass:\n%s", out)
	}
	if !strings.Contains(out, "PASS") {
		t.Errorf("expected PASS in output:\n%s", out)
	}
}

//fusa:test REQ-FO-SC004
func TestRenderJSON(t *testing.T) {
	dir := t.TempDir()
	sc, _ := safetycase.Build(dir, safetycase.StandardDO178C)
	var buf bytes.Buffer
	if err := safetycase.Render(&buf, sc, "json"); err != nil {
		t.Fatalf("Render json: %v", err)
	}
	var check safetycase.SafetyCase
	if err := json.Unmarshal(buf.Bytes(), &check); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if check.Standard != safetycase.StandardDO178C {
		t.Errorf("Standard = %q, want DO-178C", check.Standard)
	}
	if check.TotalClaims != sc.TotalClaims {
		t.Errorf("TotalClaims = %d, want %d", check.TotalClaims, sc.TotalClaims)
	}
}

//fusa:test REQ-FO-SC004
func TestRenderDefault(t *testing.T) {
	dir := t.TempDir()
	sc, _ := safetycase.Build(dir, safetycase.StandardISO26262)
	var buf bytes.Buffer
	if err := safetycase.Render(&buf, sc, ""); err != nil {
		t.Fatalf("Render default: %v", err)
	}
	if !strings.Contains(buf.String(), "FuSaOps Safety Case") {
		t.Error("default format should produce text output")
	}
}

//fusa:test REQ-FO-SC004
func TestRenderUnknownFormat(t *testing.T) {
	dir := t.TempDir()
	sc, _ := safetycase.Build(dir, safetycase.StandardISO26262)
	var buf bytes.Buffer
	if err := safetycase.Render(&buf, sc, "xml"); err == nil {
		t.Fatal("expected error for unknown format, got nil")
	}
}

// TestSaveWriteError verifies Save returns an error when the parent directory
// does not exist.
//
//fusa:test REQ-FO-SC003
func TestSaveWriteError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing", "out.json")
	if err := safetycase.Save(path, &safetycase.SafetyCase{}); err == nil {
		t.Error("Save: expected error for non-existent parent directory")
	}
}

// TestRenderTextNonPrefixPath verifies the shortPath fallback when an evidence
// path does not share the project root prefix (returns the path unchanged).
//
//fusa:test REQ-FO-SC004
func TestRenderTextNonPrefixPath(t *testing.T) {
	sc := &safetycase.SafetyCase{
		ProjectRoot: "/project",
		Standard:    safetycase.StandardISO26262,
		Claims: []safetycase.Claim{
			{
				ID: "SC-001", Title: "test claim", Strategy: "direct",
				Evidence: []safetycase.EvidenceRef{
					{Title: "External evidence", Path: "/other/path.json", Status: safetycase.StatusPresent},
				},
				Passed: true,
			},
		},
		PassedClaims: 1, TotalClaims: 1,
	}
	var buf bytes.Buffer
	if err := safetycase.Render(&buf, sc, "text"); err != nil {
		t.Fatalf("Render text: %v", err)
	}
	if !strings.Contains(buf.String(), "/other/path.json") {
		t.Errorf("expected full path when not under project root:\n%s", buf.String())
	}
}
