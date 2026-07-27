package vuln_test

// Gap tests covering uncovered branches in vuln.go.

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SoundMatt/FuSaOps/vuln"
)

// fakeRunnerScanFail reports the scanner as present (--version succeeds) but
// fails with no output on the actual scan, triggering the empty-output error
// path in runOSVScanner (vuln.go:253.33,255.3).
func fakeRunnerScanFail(args ...string) ([]byte, error) {
	if len(args) >= 2 && args[1] == "--version" {
		return []byte("osv-scanner version 1.0.0"), nil
	}
	return nil, fmt.Errorf("scan: internal error")
}

// fakeRunnerBadJSON reports the scanner as present but returns non-JSON scan
// output, triggering the JSON decode error path (vuln.go:261.44,263.3).
func fakeRunnerBadJSON(args ...string) ([]byte, error) {
	if len(args) >= 2 && args[1] == "--version" {
		return []byte("osv-scanner version 1.0.0"), nil
	}
	return []byte("not valid json"), nil
}

// TestScanNilRunner verifies Scan uses the default runner when runner is nil,
// covering vuln.go:141.19,143.3.
//
//fusa:test REQ-FO-VULN002
func TestScanNilRunner(t *testing.T) {
	r, err := vuln.Scan(t.TempDir(), nil)
	if err != nil {
		t.Fatalf("Scan with nil runner: %v", err)
	}
	if r == nil {
		t.Error("Scan with nil runner: expected non-nil report")
	}
}

// TestScanUnreadableSubdir verifies that discoverManifests silently skips
// unreadable directory entries (walk error → return nil path), covering
// vuln.go:190.17,192.4.
//
//fusa:test REQ-FO-VULN002
func TestScanUnreadableSubdir(t *testing.T) {
	dir := t.TempDir()
	// Create a subdirectory that cannot be read.
	unreadable := filepath.Join(dir, "private")
	if err := os.Mkdir(unreadable, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(unreadable, 0o755) })
	// Scan should succeed despite the unreadable subdirectory.
	r, err := vuln.Scan(dir, fakeRunnerAbsent)
	if err != nil {
		t.Errorf("Scan: unexpected error with unreadable subdir: %v", err)
	}
	if r == nil {
		t.Error("Scan: expected non-nil report")
	}
}

// TestScanOSVRunnerError verifies that a scanner runner failure with no output
// is treated as non-fatal (scan continues without findings), covering
// vuln.go:253.33,255.3 inside runOSVScanner.
//
//fusa:test REQ-FO-VULN002
func TestScanOSVRunnerError(t *testing.T) {
	dir := t.TempDir()
	// Write a go.mod so there is a manifest to scan.
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Scanner errors are non-fatal — Scan returns a report regardless.
	r, err := vuln.Scan(dir, fakeRunnerScanFail)
	if err != nil {
		t.Fatalf("Scan: unexpected error when runner fails with no output: %v", err)
	}
	if r == nil {
		t.Error("Scan: expected non-nil report even when runner fails")
	}
}

// TestScanOSVBadJSON verifies that unparseable JSON output from the scanner is
// treated as non-fatal, covering vuln.go:261.44,263.3 inside runOSVScanner.
//
//fusa:test REQ-FO-VULN002
func TestScanOSVBadJSON(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	r, err := vuln.Scan(dir, fakeRunnerBadJSON)
	if err != nil {
		t.Fatalf("Scan: unexpected error when runner returns bad JSON: %v", err)
	}
	if r == nil {
		t.Error("Scan: expected non-nil report even when JSON is bad")
	}
}

// TestRenderTextScannerPresentNoFindings verifies the "No vulnerabilities
// found" text is shown when the scanner ran but found nothing, covering
// vuln.go:378.29,380.3.
//
//fusa:test REQ-FO-VULN004
func TestRenderTextScannerPresentNoFindings(t *testing.T) {
	r := &vuln.VulnReport{ScannerPresent: true}
	var buf bytes.Buffer
	if err := vuln.Render(&buf, r, "text"); err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(buf.String(), "No vulnerabilities found") {
		t.Errorf("render text: expected 'No vulnerabilities found' in output:\n%s", buf.String())
	}
}
