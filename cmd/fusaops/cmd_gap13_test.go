package main

// Gap tests covering uncovered branches in cmd_suppress (prune), cmd_req
// (show, import, export), targeting low-coverage functions.

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// ── cmd_suppress.go — runSuppressPrune ───────────────────────────────────────

// TestSuppressPruneSaveConfigError verifies runSuppressPrune returns 1 when
// suppression.SaveConfig fails after prune removes expired entries,
// covering cmd_suppress.go:163.62,166.3.
// Strategy: write a suppress file with an expired suppression (Expires in the
// past), then make the file read-only so SaveConfig cannot overwrite it.
//
//fusa:test REQ-FO-SUP007
func TestSuppressPruneSaveConfigError(t *testing.T) {
	dir := t.TempDir()
	suppFile := filepath.Join(dir, "suppress.json")
	// Suppression with expiry in 2000 → definitely expired.
	expired := `{"suppressions":[{"fingerprint":"sha256:dead","reason":"old","expires":"2000-01-01"}]}`
	if err := os.WriteFile(suppFile, []byte(expired), 0o644); err != nil {
		t.Fatal(err)
	}
	// Make the file read-only so SaveConfig fails.
	if err := os.Chmod(suppFile, 0o444); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(suppFile, 0o644) })

	var stdout, stderr bytes.Buffer
	code := runSuppressPrune([]string{"--file", suppFile}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("runSuppressPrune SaveConfig error: want 1, got %d (stderr: %s)", code, &stderr)
	}
}

// ── cmd_req.go — runReqShow ──────────────────────────────────────────────────

// makeReqRegistry writes a .fusa-reqs.json with the provided JSON body to dir.
func makeReqRegistry(t *testing.T, dir, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, ".fusa-reqs.json"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestReqShowLoadError verifies runReqShow returns 1 when LoadRegistry fails
// (missing .fusa-reqs.json), covering cmd_req.go:47.16,50.3.
//
//fusa:test REQ-FO-CLI052
func TestReqShowLoadError(t *testing.T) {
	dir := t.TempDir() // no .fusa-reqs.json
	var stdout, stderr bytes.Buffer
	code := runReqShow(nil, dir, &stdout, &stderr)
	if code != 1 {
		t.Errorf("runReqShow LoadRegistry error: want 1, got %d", code)
	}
}

// TestReqShowTextAndLevel verifies runReqShow prints entry.Text and
// entry.Standard+entry.Level when those fields are set, covering
// cmd_req.go:64.33,66.4 (Text branch) and cmd_req.go:69.21,71.5 (Level branch).
//
//fusa:test REQ-FO-CLI052
func TestReqShowTextAndLevel(t *testing.T) {
	dir := t.TempDir()
	body := `{"requirements":[{"id":"R001","title":"Req One","text":"Full text here","standard":"ISO26262","level":"ASIL-C","priority":"HIGH"}]}`
	makeReqRegistry(t, dir, body)
	var stdout, stderr bytes.Buffer
	code := runReqShow(nil, dir, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("runReqShow text+level: want 0, got %d (stderr: %s)", code, &stderr)
	}
	out := stdout.String()
	if !bytes.Contains([]byte(out), []byte("Full text here")) {
		t.Errorf("expected Text field in output:\n%s", out)
	}
	if !bytes.Contains([]byte(out), []byte("Level:")) {
		t.Errorf("expected Level field in output:\n%s", out)
	}
}

// ── cmd_req.go — runReqImport ────────────────────────────────────────────────

// TestReqImportReadErrorNonCSV verifies runReqImport returns 1 when the input
// file cannot be read for non-csv formats (file does not exist),
// covering cmd_req.go:107.29,109.3.
//
//fusa:test REQ-FO-CLI052
func TestReqImportReadErrorNonCSV(t *testing.T) {
	dir := t.TempDir()
	nonExist := filepath.Join(dir, "no-such.xml")
	var stdout, stderr bytes.Buffer
	code := runReqImport([]string{"--format", "doors", "--file", nonExist}, dir, &stdout, &stderr)
	if code != 1 {
		t.Errorf("runReqImport read error non-csv: want 1, got %d", code)
	}
}

// TestReqImportDOORS verifies runReqImport handles the doors format,
// covering cmd_req.go:129.18,130.42 (DOORS parse branch).
//
//fusa:test REQ-FO-CLI052
func TestReqImportDOORS(t *testing.T) {
	dir := t.TempDir()
	fromFile := filepath.Join(dir, "doors.xml")
	if err := os.WriteFile(fromFile, []byte(`<REQ-IF/>`), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := runReqImport([]string{"--format", "doors", "--file", fromFile}, dir, &stdout, &stderr)
	if code != 0 {
		t.Errorf("runReqImport doors: want 0, got %d (stderr: %s)", code, &stderr)
	}
}

// TestReqImportPolarion verifies runReqImport handles the polarion format,
// covering cmd_req.go:131.20,132.44 (Polarion parse branch).
//
//fusa:test REQ-FO-CLI052
func TestReqImportPolarion(t *testing.T) {
	dir := t.TempDir()
	fromFile := filepath.Join(dir, "polarion.xml")
	if err := os.WriteFile(fromFile, []byte(`<workitems/>`), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := runReqImport([]string{"--format", "polarion", "--file", fromFile}, dir, &stdout, &stderr)
	if code != 0 {
		t.Errorf("runReqImport polarion: want 0, got %d (stderr: %s)", code, &stderr)
	}
}

// TestReqImportCodebeamer verifies runReqImport handles the codebeamer format,
// covering cmd_req.go:133.14,134.38 (Codebeamer parse branch).
//
//fusa:test REQ-FO-CLI052
func TestReqImportCodebeamer(t *testing.T) {
	dir := t.TempDir()
	fromFile := filepath.Join(dir, "cb.xml")
	if err := os.WriteFile(fromFile, []byte(`<tracker/>`), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := runReqImport([]string{"--format", "codebeamer", "--file", fromFile}, dir, &stdout, &stderr)
	if code != 0 {
		t.Errorf("runReqImport codebeamer: want 0, got %d (stderr: %s)", code, &stderr)
	}
}

// TestReqImportDuplicateSkip verifies that duplicate requirement IDs are skipped
// during import, covering cmd_req.go:146.24,148.12.
// An existing registry with REQ-001 is present; importing REQ-001 again skips it.
//
//fusa:test REQ-FO-CLI052
func TestReqImportDuplicateSkip(t *testing.T) {
	dir := t.TempDir()
	// Existing registry has REQ-001.
	makeReqRegistry(t, dir, `{"requirements":[{"id":"REQ-001","title":"Existing"}]}`)
	// Import file has REQ-001 (duplicate) and REQ-002 (new).
	csvContent := "id,title,text,description,standard,level,parent,priority,rationale\nREQ-001,Existing,,,,,,, \nREQ-002,New,,,,,,,\n"
	fromFile := filepath.Join(dir, "reqs.csv")
	if err := os.WriteFile(fromFile, []byte(csvContent), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := runReqImport([]string{"--format", "csv", "--file", fromFile}, dir, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("runReqImport duplicate skip: want 0, got %d (stderr: %s)", code, &stderr)
	}
	if out := stdout.String(); !bytes.Contains([]byte(out), []byte("1 skipped")) {
		t.Errorf("expected skip count in output:\n%s", out)
	}
}

// TestReqImportSaveRegistryError verifies runReqImport returns 1 when
// SaveRegistry fails because the directory does not exist,
// covering cmd_req.go:155.56,158.3.
//
//fusa:test REQ-FO-CLI052
func TestReqImportSaveRegistryError(t *testing.T) {
	// Use a directory that doesn't exist → SaveRegistry cannot write.
	badDir := filepath.Join(t.TempDir(), "nonexistent")
	fromFile := filepath.Join(t.TempDir(), "doors.xml")
	if err := os.WriteFile(fromFile, []byte(`<REQ-IF/>`), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := runReqImport([]string{"--format", "doors", "--file", fromFile}, badDir, &stdout, &stderr)
	if code != 1 {
		t.Errorf("runReqImport SaveRegistry error: want 1, got %d", code)
	}
}

// ── cmd_req.go — runReqExport ────────────────────────────────────────────────

// TestReqExportCSVRenderError verifies runReqExport returns 1 when RenderCSV
// fails because stdout is a broken writer, covering cmd_req.go:192.51,195.4.
//
//fusa:test REQ-FO-CLI052
func TestReqExportCSVRenderError(t *testing.T) {
	dir := t.TempDir()
	makeReqRegistry(t, dir, `{"requirements":[{"id":"R001","title":"T"}]}`)
	var stderr bytes.Buffer
	code := runReqExport([]string{"--format", "csv"}, dir, brokenWriter{}, &stderr)
	if code != 1 {
		t.Errorf("runReqExport CSV render error: want 1, got %d", code)
	}
}

// TestReqExportDOORSWriteError verifies runReqExport returns 1 when writing
// the marshalled DOORS export to stdout fails, covering cmd_req.go:218.44,221.4.
//
//fusa:test REQ-FO-CLI052
func TestReqExportDOORSWriteError(t *testing.T) {
	dir := t.TempDir()
	makeReqRegistry(t, dir, `{"requirements":[{"id":"R001","title":"T"}]}`)
	var stderr bytes.Buffer
	code := runReqExport([]string{"--format", "doors"}, dir, brokenWriter{}, &stderr)
	if code != 1 {
		t.Errorf("runReqExport DOORS write error: want 1, got %d", code)
	}
}
