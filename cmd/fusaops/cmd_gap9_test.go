package main

// Gap tests covering uncovered branches across multiple cmd files:
// cmd_check.go, cmd_comp.go, cmd_version.go, cmd_hooks.go,
// cmd_config.go, cmd_suppress.go.

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/SoundMatt/FuSaOps/config"
)

// ── cmd_check.go ──────────────────────────────────────────────────────────────

// TestCheckBadFlag verifies runCheck returns 2 for an unknown flag,
// covering the fs.Parse error branch at cmd_check.go:48.39,50.3.
//
//fusa:test REQ-FO-CLI008
func TestCheckBadFlag(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := runCheck([]string{"--bogus-flag-xyz"}, &stdout, &stderr); code != 2 {
		t.Errorf("runCheck bad flag: want 2, got %d", code)
	}
}

// TestCheckOutputFlagLeavesStdoutClean verifies that when --output is given,
// runCheck writes the report to the file only — stdout stays clean per x-FuSa
// spec §2.2 (a document written to a file must not also be echoed to stdout).
//
//fusa:test REQ-FO-SPEC001
func TestCheckOutputFlagLeavesStdoutClean(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main"), 0o644); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "report.json")
	var stdout, stderr bytes.Buffer
	runCheck([]string{"--dir", dir, "--format", "json", "--output", out}, &stdout, &stderr)
	if stdout.Len() != 0 {
		t.Errorf("runCheck with --output: stdout must be empty, got %q", stdout.String())
	}
	if _, err := os.Stat(out); err != nil {
		t.Errorf("runCheck with --output: report file not written: %v", err)
	}
}

// TestCheckLoadOptionsError verifies runCheck returns 1 when loadOptions fails
// due to malformed .fusaops.json, covering cmd_check.go:53.16,56.3.
//
//fusa:test REQ-FO-CLI008
func TestCheckLoadOptionsError(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".fusaops.json"), []byte("{bad json"), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if code := runCheck([]string{"--dir", dir}, &stdout, &stderr); code != 1 {
		t.Errorf("runCheck loadOptions error: want 1, got %d", code)
	}
}

// ── cmd_comp.go ───────────────────────────────────────────────────────────────

// TestCompLoadOptionsError verifies runComp returns 1 when loadOptions fails
// due to malformed .fusaops.json, covering cmd_comp.go:58.16,61.3.
//
//fusa:test REQ-FO-CLI082
func TestCompLoadOptionsError(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".fusaops.json"), []byte("{bad json"), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if code := runComp([]string{"--dir", dir}, &stdout, &stderr); code != 1 {
		t.Errorf("runComp loadOptions error: want 1, got %d", code)
	}
}

// TestCompTimeoutParsed verifies that opts.Timeout is set when --timeout is a
// valid duration, covering cmd_comp.go:71.3,71.19. The command returns 1
// (ErrNoAdapters) for an empty directory but line 71 is executed first.
//
//fusa:test REQ-FO-CLI082
func TestCompTimeoutParsed(t *testing.T) {
	dir := t.TempDir()
	var stdout, stderr bytes.Buffer
	code := runComp([]string{"--dir", dir, "--timeout", "30s"}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("runComp --timeout 30s: want 1 (ErrNoAdapters), got %d", code)
	}
}

// ── cmd_version.go ────────────────────────────────────────────────────────────

// TestVersionJSONEncodeError verifies runVersion returns 1 when json.Encode
// fails because stdout is a broken writer, covering cmd_version.go:38.39,41.4.
//
//fusa:test REQ-FO-CLI003
func TestVersionJSONEncodeError(t *testing.T) {
	var stderr bytes.Buffer
	if code := runVersion([]string{"--format", "json"}, brokenWriter{}, &stderr); code != 1 {
		t.Errorf("runVersion json encode error: want 1, got %d", code)
	}
}

// ── cmd_hooks.go ──────────────────────────────────────────────────────────────

// TestHooksBadFlag verifies runHooks returns 2 for an unknown flag,
// covering the fs.Parse error branch at cmd_hooks.go:39.39,41.3.
//
//fusa:test REQ-FO-CLI058
func TestHooksBadFlag(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := runHooks([]string{"--bogus-flag-xyz"}, &stdout, &stderr); code != 2 {
		t.Errorf("runHooks bad flag: want 2, got %d", code)
	}
}

// ── cmd_config.go ─────────────────────────────────────────────────────────────

// TestConfigShowEncodeError verifies runConfigShow returns 1 when json.Encode
// fails because stdout is a broken writer, covering cmd_config.go:106.40,109.3.
// config.Load is called with a valid .fusaops.json so it succeeds first.
//
//fusa:test REQ-FO-CLI044
func TestConfigShowEncodeError(t *testing.T) {
	dir := t.TempDir()
	cfg := config.Default("testproject")
	if err := config.Save(filepath.Join(dir, config.ConfigFile), cfg); err != nil {
		t.Fatal(err)
	}
	var stderr bytes.Buffer
	if code := runConfigShow([]string{"--dir", dir}, brokenWriter{}, &stderr); code != 1 {
		t.Errorf("runConfigShow encode error: want 1, got %d", code)
	}
}

// ── cmd_suppress.go ───────────────────────────────────────────────────────────

// TestSuppressImportBadFlag verifies runSuppressImport returns 2 for an unknown
// flag, covering the fs.Parse error branch at cmd_suppress.go:238.39,240.3.
//
//fusa:test REQ-FO-CLI048
func TestSuppressImportBadFlag(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := runSuppressImport([]string{"--bogus-flag-xyz"}, &stdout, &stderr); code != 2 {
		t.Errorf("runSuppressImport bad flag: want 2, got %d", code)
	}
}

// TestSuppressImportFromNotFound verifies runSuppressImport returns 1 when
// --from points to a non-existent file, covering cmd_suppress.go:247.16,250.3.
//
//fusa:test REQ-FO-CLI048
func TestSuppressImportFromNotFound(t *testing.T) {
	nonExist := filepath.Join(t.TempDir(), "no-such-report.json")
	var stdout, stderr bytes.Buffer
	code := runSuppressImport([]string{"--from", nonExist}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("runSuppressImport --from missing: want 1, got %d", code)
	}
}

// TestSuppressVerifyLoadOptionsError verifies runSuppressVerify returns 1 when
// loadOptions fails. Passing --file "" makes suppression.LoadConfig return nil
// (empty string fast-path), so we reach loadOptions with a bad .fusaops.json,
// covering cmd_suppress.go:190.16,193.3.
//
//fusa:test REQ-FO-SUP008
func TestSuppressVerifyLoadOptionsError(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".fusaops.json"), []byte("{bad json"), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := runSuppressVerify([]string{"--file", "", "--dir", dir}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("runSuppressVerify loadOptions error: want 1, got %d", code)
	}
}
