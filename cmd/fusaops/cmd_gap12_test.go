package main

// Gap tests covering uncovered branches in cmd_capabilities, cmd_history,
// cmd_suppress (add/verify/import), and cmd_conform.

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// ── cmd_capabilities.go ───────────────────────────────────────────────────────

// TestCapabilitiesEncodeError verifies runCapabilities returns 1 when
// json.Encode fails because stdout is a broken writer,
// covering cmd_capabilities.go:111.40,114.3.
//
//fusa:test REQ-FO-CLI006
func TestCapabilitiesEncodeError(t *testing.T) {
	var stderr bytes.Buffer
	if code := runCapabilities([]string{}, brokenWriter{}, &stderr); code != 1 {
		t.Errorf("runCapabilities encode error: want 1, got %d", code)
	}
}

// ── cmd_history.go ────────────────────────────────────────────────────────────

// TestHistoryListJSONEncodeError verifies runHistoryList returns 1 when
// json.Encode fails because stdout is a broken writer,
// covering cmd_history.go:62.43,65.4.
// history.Load returns an empty slice (no history file), so the encode is
// attempted and fails against the broken writer.
//
//fusa:test REQ-FO-CLI045
func TestHistoryListJSONEncodeError(t *testing.T) {
	dir := t.TempDir()
	var stderr bytes.Buffer
	code := runHistoryList([]string{"--dir", dir, "--format", "json"}, brokenWriter{}, &stderr)
	if code != 1 {
		t.Errorf("runHistoryList JSON encode error: want 1, got %d", code)
	}
}

// ── cmd_suppress.go — runSuppressAdd ─────────────────────────────────────────

// TestSuppressAddLoadConfigError verifies runSuppressAdd returns 1 when
// suppression.LoadConfig fails with a non-ErrNotExist error (bad JSON in
// existing suppress file), covering cmd_suppress.go:79.51,82.3.
//
//fusa:test REQ-FO-SUP005
func TestSuppressAddLoadConfigError(t *testing.T) {
	dir := t.TempDir()
	suppFile := filepath.Join(dir, "suppress.json")
	// Existing file with malformed JSON → LoadConfig returns json.SyntaxError
	// (not ErrNotExist), entering the error body.
	if err := os.WriteFile(suppFile, []byte("{bad json"), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := runSuppressAdd([]string{
		"--file", suppFile,
		"--fingerprint", "sha256:aaaa",
		"--reason", "test",
	}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("runSuppressAdd LoadConfig error: want 1, got %d", code)
	}
}

// TestSuppressAddSaveConfigError verifies runSuppressAdd returns 1 when
// suppression.SaveConfig fails because the parent directory does not exist,
// covering cmd_suppress.go:88.59,91.3.
// LoadConfig returns ErrNotExist (file missing) so the error body is skipped;
// then SaveConfig fails.
//
//fusa:test REQ-FO-SUP005
func TestSuppressAddSaveConfigError(t *testing.T) {
	badFile := filepath.Join(t.TempDir(), "nonexistent", "suppress.json")
	var stdout, stderr bytes.Buffer
	code := runSuppressAdd([]string{
		"--file", badFile,
		"--fingerprint", "sha256:bbbb",
		"--reason", "test",
	}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("runSuppressAdd SaveConfig error: want 1, got %d", code)
	}
}

// ── cmd_suppress.go — runSuppressVerify ──────────────────────────────────────

// TestSuppressVerifyOrchestratorError verifies runSuppressVerify returns 1 when
// the orchestrator scan fails (ErrNoAdapters for an empty directory),
// covering cmd_suppress.go:195.16,198.3.
// Using --file "" exercises the LoadConfig fast-path (returns Config{}, nil)
// so we reach the loadOptions → orchestrator.Run sequence.
//
//fusa:test REQ-FO-SUP008
func TestSuppressVerifyOrchestratorError(t *testing.T) {
	dir := t.TempDir() // empty — no languages detected
	var stdout, stderr bytes.Buffer
	code := runSuppressVerify([]string{"--file", "", "--dir", dir}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("runSuppressVerify orchestrator error: want 1 (ErrNoAdapters), got %d", code)
	}
}

// ── cmd_suppress.go — runSuppressImport ──────────────────────────────────────

// suppresImportReport is a minimal JSON fusaops check report used by import tests.
func suppresImportReport(t *testing.T, includeEmptyFP bool) string {
	t.Helper()
	type finding struct {
		Fingerprint string `json:"fingerprint"`
		RuleID      string `json:"ruleId"`
		Severity    string `json:"severity"`
		Message     string `json:"message"`
	}
	type component struct {
		Findings []finding `json:"findings"`
	}
	type aggReport struct {
		Components []component `json:"components"`
	}
	findings := []finding{
		{Fingerprint: "sha256:cccc", RuleID: "R001", Severity: "WARNING", Message: "test"},
	}
	if includeEmptyFP {
		findings = append(findings, finding{Fingerprint: "", RuleID: "R002", Severity: "INFO", Message: "no-fp"})
	}
	rep := aggReport{Components: []component{{Findings: findings}}}
	data, err := json.Marshal(rep)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// TestSuppressImportLoadConfigError verifies runSuppressImport returns 1 when
// suppression.LoadConfig on the suppress file fails with a non-ErrNotExist
// error (bad JSON), covering cmd_suppress.go:258.47,261.3.
//
//fusa:test REQ-FO-SUP009
func TestSuppressImportLoadConfigError(t *testing.T) {
	dir := t.TempDir()
	// Valid --from report.
	fromFile := filepath.Join(dir, "report.json")
	if err := os.WriteFile(fromFile, []byte(suppresImportReport(t, false)), 0o644); err != nil {
		t.Fatal(err)
	}
	// Suppress file exists but contains bad JSON.
	suppFile := filepath.Join(dir, "suppress.json")
	if err := os.WriteFile(suppFile, []byte("{bad json"), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := runSuppressImport([]string{"--from", fromFile, "--file", suppFile}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("runSuppressImport LoadConfig error: want 1, got %d", code)
	}
}

// TestSuppressImportEmptyFingerprint verifies that findings with an empty
// fingerprint are skipped during import, covering cmd_suppress.go:271.27,272.13.
// The import runs to completion (SaveConfig succeeds).
//
//fusa:test REQ-FO-SUP009
func TestSuppressImportEmptyFingerprint(t *testing.T) {
	dir := t.TempDir()
	fromFile := filepath.Join(dir, "report.json")
	if err := os.WriteFile(fromFile, []byte(suppresImportReport(t, true)), 0o644); err != nil {
		t.Fatal(err)
	}
	suppFile := filepath.Join(dir, "new-suppress.json") // does not exist yet → ErrNotExist tolerated
	var stdout, stderr bytes.Buffer
	code := runSuppressImport([]string{"--from", fromFile, "--file", suppFile}, &stdout, &stderr)
	if code != 0 {
		t.Errorf("runSuppressImport empty FP: want 0, got %d (stderr: %s)", code, &stderr)
	}
}

// TestSuppressImportSaveConfigError verifies runSuppressImport returns 1 when
// suppression.SaveConfig fails because the output path is in a non-existent
// directory, covering cmd_suppress.go:288.59,291.3.
//
//fusa:test REQ-FO-SUP009
func TestSuppressImportSaveConfigError(t *testing.T) {
	dir := t.TempDir()
	fromFile := filepath.Join(dir, "report.json")
	if err := os.WriteFile(fromFile, []byte(suppresImportReport(t, false)), 0o644); err != nil {
		t.Fatal(err)
	}
	// Bad suppress file path — parent dir doesn't exist → SaveConfig fails.
	badFile := filepath.Join(dir, "nonexistent", "suppress.json")
	var stdout, stderr bytes.Buffer
	code := runSuppressImport([]string{"--from", fromFile, "--file", badFile}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("runSuppressImport SaveConfig error: want 1, got %d", code)
	}
}

// ── cmd_conform.go ────────────────────────────────────────────────────────────

// TestConformOutputCreateError verifies runConform returns 1 when the output
// file cannot be created (parent directory does not exist), covering
// cmd_conform.go:60.17,63.4.
// /bin/sh is used as the binary — it exists on all POSIX systems and conform.Run
// completes with a report (non-conformant) rather than returning an error.
//
//fusa:test REQ-FO-CLI014
func TestConformOutputCreateError(t *testing.T) {
	badOutput := filepath.Join(t.TempDir(), "nonexistent", "conform.txt")
	var stdout, stderr bytes.Buffer
	code := runConform([]string{"/bin/sh", "--output", badOutput}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("runConform output create error: want 1, got %d (stderr: %s)", code, &stderr)
	}
}

// TestConformRenderError verifies runConform returns 1 when conform.Render
// fails because stdout is a broken writer, covering cmd_conform.go:68.56,71.3.
//
//fusa:test REQ-FO-CLI014
func TestConformRenderError(t *testing.T) {
	var stderr bytes.Buffer
	code := runConform([]string{"/bin/sh"}, brokenWriter{}, &stderr)
	if code != 1 {
		t.Errorf("runConform render error: want 1, got %d (stderr: %s)", code, &stderr)
	}
}
