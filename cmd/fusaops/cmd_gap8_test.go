package main

// Gap tests covering uncovered branches across multiple cmd files:
// cmd_auditpack.go, cmd_report.go, cmd_scan.go, cmd_pr.go,
// cmd_metrics.go, cmd_badge.go.

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// brokenWriter always returns an error from Write, used to trigger render error paths.
type brokenWriter struct{}

func (brokenWriter) Write([]byte) (int, error) { return 0, fmt.Errorf("write: broken") }

// ── cmd_auditpack.go ──────────────────────────────────────────────────────────

// TestAuditPackLoadOptionsError verifies runAuditPack returns 1 when loadOptions
// fails due to malformed .fusaops.json, covering cmd_auditpack.go:33.16,36.3.
//
//fusa:test REQ-FO-CLI013
func TestAuditPackLoadOptionsError(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".fusaops.json"), []byte("{bad json"), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := runAuditPack([]string{"--dir", dir}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("runAuditPack loadOptions error: want 1, got %d", code)
	}
}

// ── cmd_report.go ─────────────────────────────────────────────────────────────

// TestReportLoadOptionsError verifies runReport returns 1 when loadOptions
// fails due to malformed .fusaops.json, covering cmd_report.go:45.16,48.3.
//
//fusa:test REQ-FO-CLI009
func TestReportLoadOptionsError(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".fusaops.json"), []byte("{bad json"), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := runReport([]string{"--dir", dir}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("runReport loadOptions error: want 1, got %d", code)
	}
}

// ── cmd_scan.go ───────────────────────────────────────────────────────────────

// TestScanNonExistentDir verifies runScan returns 1 when the target directory
// does not exist, covering the scan.Scan error branch at cmd_scan.go:30.16,33.3.
//
//fusa:test REQ-FO-CLI005
func TestScanNonExistentDir(t *testing.T) {
	nonExist := filepath.Join(t.TempDir(), "does-not-exist")
	var stdout, stderr bytes.Buffer
	code := runScan([]string{"--dir", nonExist}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("runScan non-existent dir: want 1, got %d", code)
	}
}

// ── cmd_pr.go ─────────────────────────────────────────────────────────────────

// TestPRBadFlag verifies runPR returns 2 for an unknown flag,
// covering the fs.Parse error branch at cmd_pr.go:31.39,33.3.
//
//fusa:test REQ-FO-CLI061
func TestPRBadFlag(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := runPR([]string{"--bogus-flag-xyz"}, &stdout, &stderr); code != 2 {
		t.Errorf("runPR bad flag: want 2, got %d", code)
	}
}

// TestPRAddBadFlag verifies prAdd returns 2 for an unknown flag (via runPR
// "add --bogus"), covering the fs.Parse error branch at cmd_pr.go:85.39,87.3.
//
//fusa:test REQ-FO-CLI061
func TestPRAddBadFlag(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runPR([]string{"add", "--bogus-flag-xyz"}, &stdout, &stderr)
	if code != 2 {
		t.Errorf("runPR add bad flag: want 2, got %d", code)
	}
}

// TestPRAddLoadError verifies prAdd returns 1 when pr.Load fails because the
// problems file is a directory (EISDIR), covering cmd_pr.go:98.16,101.3.
//
//fusa:test REQ-FO-CLI061
func TestPRAddLoadError(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".fusaops-problems.json"), 0o755); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := runPR([]string{"--dir", dir, "add", "--id", "PR-001", "--title", "T"}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("runPR add load error: want 1, got %d", code)
	}
}

// TestPRCloseBadFlag verifies prClose returns 2 for an unknown flag (via runPR
// "close --bogus"), covering the fs.Parse error branch at cmd_pr.go:148.39,150.3.
//
//fusa:test REQ-FO-CLI061
func TestPRCloseBadFlag(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runPR([]string{"close", "--bogus-flag-xyz"}, &stdout, &stderr)
	if code != 2 {
		t.Errorf("runPR close bad flag: want 2, got %d", code)
	}
}

// ── cmd_metrics.go ────────────────────────────────────────────────────────────

// TestMetricsBadFlag verifies runMetrics returns 2 for an unknown flag,
// covering the fs.Parse error branch at cmd_metrics.go:29.39,31.3.
//
//fusa:test REQ-FO-CLI055
func TestMetricsBadFlag(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := runMetrics([]string{"--bogus-flag-xyz"}, &stdout, &stderr); code != 2 {
		t.Errorf("runMetrics bad flag: want 2, got %d", code)
	}
}

// TestMetricsShowLoadError verifies runMetricsShow returns 1 when metrics.Load
// fails due to malformed .fusaops-metrics.json, covering cmd_metrics.go:94.16,97.3.
//
//fusa:test REQ-FO-CLI055
func TestMetricsShowLoadError(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".fusaops-metrics.json"), []byte("{bad json"), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := runMetrics([]string{"--dir", dir, "show"}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("runMetrics show load error: want 1, got %d", code)
	}
}

// TestMetricsShowRenderError verifies runMetricsShow returns 1 when metrics.Render
// returns an error for an unsupported format, covering cmd_metrics.go:110.55,113.3.
//
//fusa:test REQ-FO-CLI055
func TestMetricsShowRenderError(t *testing.T) {
	dir := t.TempDir()
	var stdout, stderr bytes.Buffer
	code := runMetrics([]string{"--dir", dir, "show", "--format", "xml"}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("runMetrics show render error: want 1, got %d", code)
	}
}

// ── cmd_badge.go ──────────────────────────────────────────────────────────────

// TestBadgeRenderError verifies runBadge returns 1 when badge.Render fails
// because the output writer is broken, covering cmd_badge.go:67.43,70.3.
//
//fusa:test REQ-FO-CLI056
func TestBadgeRenderError(t *testing.T) {
	dir := t.TempDir()
	reportFile := filepath.Join(dir, "report.json")
	if err := os.WriteFile(reportFile, []byte(`{"summary":{"errors":0,"warnings":0}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	var stderr bytes.Buffer
	code := runBadge([]string{reportFile}, brokenWriter{}, &stderr)
	if code != 1 {
		t.Errorf("runBadge render error: want 1, got %d", code)
	}
}
