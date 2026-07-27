package main

// Gap tests covering uncovered branches in cmd_coverage (MCDC scan-warning),
// cmd_vv (vv set config.Save error), and cmd_vuln (scan error).

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// ── cmd_vv.go — runVVSet ─────────────────────────────────────────────────────

// TestVVSetConfigSaveError verifies runVVSet returns 1 when config.Save fails
// because the .fusaops.json file is read-only, covering cmd_vv.go:178.50,181.3.
// A valid .fusaops.json is written with required fields, then made read-only.
//
//fusa:test REQ-FO-CLI057
func TestVVSetConfigSaveError(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, ".fusaops.json")
	cfgJSON := `{"version":"1.0.0","project":{"name":"test"}}`
	if err := os.WriteFile(cfgPath, []byte(cfgJSON), 0o644); err != nil {
		t.Fatal(err)
	}
	// Make the config file read-only so config.Save fails.
	if err := os.Chmod(cfgPath, 0o444); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(cfgPath, 0o644) })

	var stdout, stderr bytes.Buffer
	code := runVVSet(
		[]string{"--implementation-author", "Alice"},
		dir, &stdout, &stderr,
	)
	if code != 1 {
		t.Errorf("runVVSet config.Save error: want 1, got %d (stderr: %s)", code, &stderr)
	}
}

// ── cmd_vuln.go — runVuln ────────────────────────────────────────────────────

// TestVulnScanError verifies runVuln returns 1 when vuln.Scan fails because
// the project directory does not exist, covering cmd_vuln.go:51.16,54.3.
// Passing --dir to a non-existent path bypasses os.Getwd() and feeds the
// bad path directly to vuln.Scan → discoverManifests → filepath.WalkDir.
//
//fusa:test REQ-FO-CLI071
func TestVulnScanError(t *testing.T) {
	nonExist := filepath.Join(t.TempDir(), "no-such-project")
	var stdout, stderr bytes.Buffer
	code := runVuln([]string{"--dir", nonExist}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("runVuln scan error: want 1 (discoverManifests), got %d (stderr: %s)", code, &stderr)
	}
}
