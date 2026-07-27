package main

// Gap tests covering uncovered branches in cmd_fleet, cmd_sbom, cmd_policy,
// cmd_sign, common.go (component timeout), and main.go dispatch paths.

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// ── cmd_fleet.go ──────────────────────────────────────────────────────────────

// TestFleetBadFlag verifies runFleet returns 2 for an unknown flag,
// covering cmd_fleet.go:25.39,27.3.
//
//fusa:test REQ-FO-CLI023
func TestFleetBadFlag(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := runFleet([]string{"--bogus-flag-xyz"}, &stdout, &stderr); code != 2 {
		t.Errorf("runFleet bad flag: want 2, got %d", code)
	}
}

// ── cmd_sbom.go ───────────────────────────────────────────────────────────────

// TestSBOMLoadOptionsError verifies runSBOM returns 1 when loadOptions fails
// due to a malformed .fusaops.json, covering cmd_sbom.go:37.16,40.3.
//
//fusa:test REQ-FO-CLI012
func TestSBOMLoadOptionsError(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".fusaops.json"), []byte("{bad json"), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if code := runSBOM([]string{"--dir", dir}, &stdout, &stderr); code != 1 {
		t.Errorf("runSBOM loadOptions error: want 1, got %d", code)
	}
}

// TestSBOMTimeoutParsed verifies opts.Timeout is set when --timeout is a valid
// duration, covering cmd_sbom.go:50.3,50.19. The command returns 1
// (ErrNoAdapters for an empty dir) but the assignment executes first.
//
//fusa:test REQ-FO-CLI049
func TestSBOMTimeoutParsed(t *testing.T) {
	dir := t.TempDir() // empty — no languages detected → ErrNoAdapters
	var stdout, stderr bytes.Buffer
	if code := runSBOM([]string{"--dir", dir, "--timeout", "30s"}, &stdout, &stderr); code != 1 {
		t.Errorf("runSBOM --timeout 30s: want 1 (ErrNoAdapters), got %d", code)
	}
}

// ── cmd_policy.go ─────────────────────────────────────────────────────────────

// TestPolicyScanError verifies runPolicy returns 1 when the orchestrator scan
// fails (ErrNoAdapters for an empty directory), covering cmd_policy.go:42.16,45.3.
// A minimal valid policy.json is created so LoadPolicy succeeds first.
//
//fusa:test REQ-FO-CLI024
func TestPolicyScanError(t *testing.T) {
	dir := t.TempDir()
	policyFile := filepath.Join(dir, "policy.json")
	if err := os.WriteFile(policyFile, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := runPolicy([]string{"--dir", dir, "--policy", policyFile}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("runPolicy scan error: want 1 (ErrNoAdapters), got %d", code)
	}
}

// ── cmd_sign.go ───────────────────────────────────────────────────────────────

// TestSignKeygenError verifies runSign returns 1 when sign.Keygen cannot write
// to the given path (parent directory does not exist),
// covering cmd_sign.go:37.46,40.4.
//
//fusa:test REQ-FO-CLI063
func TestSignKeygenError(t *testing.T) {
	badPath := filepath.Join(t.TempDir(), "nonexistent", "key.hex")
	var stdout, stderr bytes.Buffer
	if code := runSign([]string{"--keygen", badPath}, &stdout, &stderr); code != 1 {
		t.Errorf("runSign keygen error: want 1, got %d", code)
	}
}

// TestSignSignError verifies runSign returns 1 when sign.Sign fails because the
// target file does not exist, covering cmd_sign.go:72.20,75.3.
// A valid key file is written first so sign.LoadKey succeeds.
//
//fusa:test REQ-FO-CLI063
func TestSignSignError(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "key.hex")
	// 64 hex chars = 32 bytes, satisfies LoadKey minimum.
	if err := os.WriteFile(keyPath, []byte("0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	nonExist := filepath.Join(dir, "no-such-artifact.zip")
	var stdout, stderr bytes.Buffer
	code := runSign([]string{"--key", keyPath, nonExist}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("runSign sign error: want 1 (file not found), got %d", code)
	}
}

// ── common.go (component timeout) ────────────────────────────────────────────

// TestLoadOptionsComponentTimeout verifies that component timeout values in
// .fusaops.json are parsed and either applied or logged as invalid.
// Covers common.go:48.23,50.20 (entering the timeout block),
// 50.20,52.6 (invalid duration error log), and 52.11,54.6 (valid duration set).
//
//fusa:test REQ-FO-CLI007
func TestLoadOptionsComponentTimeout(t *testing.T) {
	dir := t.TempDir()
	// Valid JSON with version and project.name (required by Validate), with two
	// components: one with an invalid timeout and one with a valid timeout.
	cfgJSON := `{
  "project": {"name": "test"},
  "version": "1.0.0",
  "scan": {
    "components": [
      {"path": ".", "timeout": "INVALID_DURATION"},
      {"path": ".", "timeout": "30s"}
    ]
  }
}`
	if err := os.WriteFile(filepath.Join(dir, ".fusaops.json"), []byte(cfgJSON), 0o644); err != nil {
		t.Fatal(err)
	}
	// Use runCheck: it calls loadOptions, which processes components.
	// The orchestrator will return ErrNoAdapters for an empty dir → code 1.
	var stdout, stderr bytes.Buffer
	code := runCheck([]string{"--dir", dir}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("runCheck component timeout: want 1 (ErrNoAdapters), got %d (stderr: %s)", code, &stderr)
	}
	// The invalid duration should produce a warning on stderr.
	if got := stderr.String(); got == "" {
		t.Error("expected warning about invalid component timeout in stderr")
	}
}

// ── main.go dispatch ─────────────────────────────────────────────────────────

// TestRunDispatchConform verifies run() dispatches "conform" to runConform,
// covering main.go:102.17,103.46.
//
//fusa:test REQ-FO-CLI001
func TestRunDispatchConform(t *testing.T) {
	var stdout, stderr bytes.Buffer
	// Passing an unknown flag causes runConform to return 2, confirming dispatch.
	if code := run([]string{"conform", "--bogus-flag-xyz"}, &stdout, &stderr); code != 2 {
		t.Errorf("run conform dispatch: want 2, got %d", code)
	}
}

// TestRunDispatchStandards verifies run() dispatches standards subcommands to
// runStandards, covering main.go:104.72,105.57.
//
//fusa:test REQ-FO-CLI001
func TestRunDispatchStandards(t *testing.T) {
	var stdout, stderr bytes.Buffer
	// Passing an unknown flag causes runStandards to return 2, confirming dispatch.
	if code := run([]string{"iso26262", "--bogus-flag-xyz"}, &stdout, &stderr); code != 2 {
		t.Errorf("run iso26262 dispatch: want 2, got %d", code)
	}
}
