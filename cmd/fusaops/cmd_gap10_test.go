package main

// Gap tests covering uncovered branches in cmd_coverage.go and cmd_vv.go.

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// ── cmd_coverage.go ───────────────────────────────────────────────────────────

// TestCoverageOutputCreateError verifies runCoverage returns 1 when the output
// file cannot be created, covering cmd_coverage.go:61.17,64.4.
//
//fusa:test REQ-FO-CLI051
func TestCoverageOutputCreateError(t *testing.T) {
	badOutput := filepath.Join(t.TempDir(), "nonexistent", "subdir", "out.txt")
	var stdout, stderr bytes.Buffer
	code := runCoverage([]string{"--output", badOutput}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("runCoverage output create error: want 1, got %d", code)
	}
}

// TestCoverageStatFallback verifies runCoverage falls back to cwd/coverage.out
// when no positional argument and no --dir flag are given and coverage.out does
// not exist in the working directory, covering cmd_coverage.go:80.8,80.55 and
// 80.55,83.3. Returns 1 because BuildFromFile then fails for the missing file.
//
//fusa:test REQ-FO-CLI051
func TestCoverageStatFallback(t *testing.T) {
	// coverage.out does not exist in cmd/fusaops (the test CWD).
	var stdout, stderr bytes.Buffer
	code := runCoverage([]string{}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("runCoverage stat fallback: want 1 (BuildFromFile fails), got %d", code)
	}
}

// TestCoverageRenderError verifies runCoverage returns 1 when coverage.Render
// fails due to an unsupported format, covering cmd_coverage.go:92.57,95.3.
// A minimal valid profile is written so BuildFromFile succeeds first.
//
//fusa:test REQ-FO-CLI051
func TestCoverageRenderError(t *testing.T) {
	dir := t.TempDir()
	profilePath := filepath.Join(dir, "coverage.out")
	if err := os.WriteFile(profilePath, []byte("mode: set\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := runCoverage([]string{"--format", "xml", profilePath}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("runCoverage render error: want 1 (unsupported format), got %d", code)
	}
}

// ── cmd_vv.go ─────────────────────────────────────────────────────────────────

// TestVVShowConfigLoadError verifies runVVShow returns 1 when config.Load
// returns a non-ErrNoConfig error (invalid config file with empty version),
// covering cmd_vv.go:85.56,88.3.
//
//fusa:test REQ-FO-CLI075
func TestVVShowConfigLoadError(t *testing.T) {
	dir := t.TempDir()
	// Valid JSON but empty version → Validate fails (ErrInvalidConfig, not ErrNoConfig).
	if err := os.WriteFile(filepath.Join(dir, ".fusaops.json"), []byte(`{"project":{"name":"test"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := runVV([]string{"--dir", dir, "show"}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("runVVShow config load error: want 1, got %d", code)
	}
}

// TestVVShowOutputCreateError verifies runVVShow returns 1 when the output file
// cannot be created, covering cmd_vv.go:109.17,112.4.
//
//fusa:test REQ-FO-CLI075
func TestVVShowOutputCreateError(t *testing.T) {
	badOutput := filepath.Join(t.TempDir(), "nonexistent", "subdir", "vv.txt")
	var stdout, stderr bytes.Buffer
	code := runVV([]string{"show", "--output", badOutput}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("runVVShow output create error: want 1, got %d", code)
	}
}

// TestVVShowRenderError verifies runVVShow returns 1 when vv.Render fails due
// to an unsupported format, covering cmd_vv.go:117.52,120.3.
//
//fusa:test REQ-FO-CLI075
func TestVVShowRenderError(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runVV([]string{"show", "--format", "xml"}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("runVVShow render error: want 1 (unsupported format), got %d", code)
	}
}

// TestVVSetLoadNonErrNoConfig verifies runVVSet returns 1 via the non-ErrNoConfig
// else branch when config.Load returns ErrInvalidConfig (empty version in config),
// covering cmd_vv.go:150.9,152.4.
//
//fusa:test REQ-FO-CLI076
func TestVVSetLoadNonErrNoConfig(t *testing.T) {
	dir := t.TempDir()
	// Valid JSON but empty version → Load.Validate fails with ErrInvalidConfig.
	if err := os.WriteFile(filepath.Join(dir, ".fusaops.json"), []byte(`{"project":{"name":"test"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := runVV([]string{"--dir", dir, "set", "--implementation-author", "alice"}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("runVVSet non-ErrNoConfig load error: want 1, got %d", code)
	}
}
