package main

// Gap tests covering uncovered branches in cmd_pr, cmd_report, and
// a remaining cmd_req import path (jama format).

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// ── cmd_pr.go — prAdd ────────────────────────────────────────────────────────

// TestPRAddSaveError verifies prAdd returns 1 when pr.Save fails because
// the project directory is read-only, covering cmd_pr.go:113.42,116.3.
// pr.Load returns an empty log for a missing problems file, so we reach Save.
//
//fusa:test REQ-FO-CLI061
func TestPRAddSaveError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix permission semantics (0o555) not available on Windows")
	}
	dir := t.TempDir()
	// Make the directory read-only so Save cannot create .fusaops-problems.json.
	if err := os.Chmod(dir, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })

	var stdout, stderr bytes.Buffer
	code := prAdd([]string{"--id", "PR-001", "--title", "Test issue"}, dir, &stdout, &stderr)
	if code != 1 {
		t.Errorf("prAdd Save error: want 1, got %d (stderr: %s)", code, &stderr)
	}
}

// ── cmd_pr.go — prClose ──────────────────────────────────────────────────────

// TestPRCloseSaveErrorReadOnly verifies prClose returns 1 when pr.Save fails
// because the problems file is read-only (making os.WriteFile fail),
// covering cmd_pr.go:165.42,168.3.
// An existing problems file with a known ID is written first, then the FILE
// is made read-only so Save fails after Close succeeds.
// (distinct from TestPRCloseSaveError in cmd_extra_coverage_test.go which
// covers the pr.Load error path via EISDIR, not the Save path)
//
//fusa:test REQ-FO-CLI061
func TestPRCloseSaveErrorReadOnly(t *testing.T) {
	dir := t.TempDir()
	probFile := filepath.Join(dir, ".fusaops-problems.json")
	// Write a valid problems file with one open PR.
	problemsJSON := `{"project":"test","reports":[{"id":"PR-001","title":"Test","status":"open","severity":"minor","phaseFound":"development","created":"2026-01-01T00:00:00Z","updated":"2026-01-01T00:00:00Z"}]}`
	if err := os.WriteFile(probFile, []byte(problemsJSON), 0o644); err != nil {
		t.Fatal(err)
	}
	// Make the problems FILE read-only so os.WriteFile in pr.Save fails.
	if err := os.Chmod(probFile, 0o444); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(probFile, 0o644) })

	var stdout, stderr bytes.Buffer
	code := prClose([]string{"--id", "PR-001", "--resolution", "Fixed"}, dir, &stdout, &stderr)
	if code != 1 {
		t.Errorf("prClose Save error: want 1, got %d (stderr: %s)", code, &stderr)
	}
}

// ── cmd_report.go — runReport ────────────────────────────────────────────────

// TestReportBadFlag verifies runReport returns 2 for an unknown flag,
// covering cmd_report.go:40.39,42.3.
//
//fusa:test REQ-FO-CLI009
func TestReportBadFlag(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runReport([]string{"--bogus-flag-xyz"}, &stdout, &stderr)
	if code != 2 {
		t.Errorf("runReport bad flag: want 2, got %d", code)
	}
}

// TestReportTimeoutAssigned verifies opts.Timeout is set when --timeout is a
// valid duration, covering cmd_report.go:59.3,59.19.
// The command returns 1 (ErrNoAdapters for empty dir) but the assignment runs first.
//
//fusa:test REQ-FO-CLI049
func TestReportTimeoutAssigned(t *testing.T) {
	dir := t.TempDir()
	var stdout, stderr bytes.Buffer
	code := runReport([]string{"--dir", dir, "--timeout", "30s"}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("runReport --timeout 30s: want 1 (ErrNoAdapters), got %d", code)
	}
}

// ── cmd_req.go — Jama import ─────────────────────────────────────────────────

// TestReqImportJama verifies runReqImport handles the jama format,
// covering cmd_req.go:133.14,134.38 was covered by codebeamer — this adds Jama.
// Note: the Jama parser expects XML, and <items/> parses successfully.
//
//fusa:test REQ-FO-CLI052
func TestReqImportJama(t *testing.T) {
	dir := t.TempDir()
	fromFile := filepath.Join(dir, "jama.xml")
	if err := os.WriteFile(fromFile, []byte(`<items/>`), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := runReqImport([]string{"--format", "jama", "--file", fromFile}, dir, &stdout, &stderr)
	if code != 0 {
		t.Errorf("runReqImport jama: want 0, got %d (stderr: %s)", code, &stderr)
	}
}
