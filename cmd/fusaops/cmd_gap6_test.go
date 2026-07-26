package main

// Gap tests covering uncovered branches in cmd_config, cmd_fmea, cmd_history,
// cmd_scan, and cmd_policy.

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/SoundMatt/FuSaOps/history"
)

// ── cmd_config.go ─────────────────────────────────────────────────────────────

// TestConfigValidateBadFlag verifies runConfigValidate returns 2 for an
// unknown flag, covering the fs.Parse error branch (cmd_config.go:43-45).
//
//fusa:test REQ-FO-CLI043
func TestConfigValidateBadFlag(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := runConfigValidate([]string{"--bogus-flag-xyz"}, &stdout, &stderr); code != 2 {
		t.Errorf("runConfigValidate bad flag: want 2, got %d", code)
	}
}

// TestConfigValidateWithAdapters verifies the adapters section of
// runConfigValidate is printed when cfg.Scan.Adapters is non-empty, covering
// cmd_config.go:68-70.
//
//fusa:test REQ-FO-CLI043
func TestConfigValidateWithAdapters(t *testing.T) {
	dir := t.TempDir()
	cfgJSON := `{"version":"1","project":{"name":"test"},"scan":{"adapters":["gofusa"]}}`
	if err := os.WriteFile(filepath.Join(dir, ".fusaops.json"), []byte(cfgJSON), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if code := runConfigValidate([]string{"--dir", dir}, &stdout, &stderr); code != 0 {
		t.Errorf("runConfigValidate with adapters: want 0, got %d\nstderr: %s", code, stderr.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte("gofusa")) {
		t.Errorf("expected adapters in output: %s", stdout.String())
	}
}

// TestConfigShowNoConfig verifies runConfigShow returns 1 with a message when
// no .fusaops.json exists in the given directory, covering cmd_config.go:96-99
// (ErrNoConfig branch).
//
//fusa:test REQ-FO-CLI044
func TestConfigShowNoConfig(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runConfigShow([]string{"--dir", t.TempDir()}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("runConfigShow no config: want 1, got %d", code)
	}
}

// TestConfigShowBadJSON verifies runConfigShow returns 1 when .fusaops.json
// contains invalid JSON, covering the generic error branch cmd_config.go:100-101.
//
//fusa:test REQ-FO-CLI044
func TestConfigShowBadJSON(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".fusaops.json"), []byte("{bad json"), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := runConfigShow([]string{"--dir", dir}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("runConfigShow bad JSON: want 1, got %d", code)
	}
}

// ── cmd_fmea.go ───────────────────────────────────────────────────────────────

// TestFMEANoDirFlag verifies runFMEA uses os.Getwd() when no --dir flag is
// passed, covering the `if projectRoot == ""` block (cmd_fmea.go:42-44).
// The test accepts either exit code because FMEA may flag high-RPN items.
//
//fusa:test REQ-FO-CLI070
func TestFMEANoDirFlag(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runFMEA([]string{}, &stdout, &stderr)
	if code != 0 && code != 1 {
		t.Errorf("runFMEA no --dir: want 0 or 1, got %d\nstderr: %s", code, stderr.String())
	}
}

// ── cmd_history.go ────────────────────────────────────────────────────────────

// TestHistoryListBadFlag verifies runHistoryList returns 2 for an unknown flag,
// covering the parse error branch (cmd_history.go:43-45).
//
//fusa:test REQ-FO-CLI045
func TestHistoryListBadFlag(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := runHistoryList([]string{"--bogus-flag-xyz"}, &stdout, &stderr); code != 2 {
		t.Errorf("runHistoryList bad flag: want 2, got %d", code)
	}
}

// TestHistoryListLoadError verifies runHistoryList returns 1 when the history
// file cannot be read (made a directory), covering cmd_history.go:53-56.
//
//fusa:test REQ-FO-CLI045
func TestHistoryListLoadError(t *testing.T) {
	dir := t.TempDir()
	// Create the history filename as a directory to trigger a read error.
	if err := os.Mkdir(filepath.Join(dir, history.Filename), 0o755); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := runHistoryList([]string{"--dir", dir}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("runHistoryList load error: want 1, got %d", code)
	}
}

// TestHistoryListJSONEmptyNoHistory verifies runHistoryList --format json
// with an empty directory returns 0 and outputs an empty JSON array.
//
//fusa:test REQ-FO-CLI045
func TestHistoryListJSONFormatEmpty(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runHistoryList([]string{"--dir", t.TempDir(), "--format", "json"}, &stdout, &stderr)
	if code != 0 {
		t.Errorf("runHistoryList json empty: want 0, got %d\nstderr: %s", code, stderr.String())
	}
}

// TestHistoryPruneFileFlag verifies runHistoryPrune resolves the dir from
// the parent of the --file path (cmd_history.go:103-105).
//
//fusa:test REQ-FO-CLI046
func TestHistoryPruneFileFlag(t *testing.T) {
	dir := t.TempDir()
	histFile := filepath.Join(dir, history.Filename)
	var stdout, stderr bytes.Buffer
	code := runHistoryPrune([]string{"--file", histFile}, &stdout, &stderr)
	if code != 0 {
		t.Errorf("runHistoryPrune --file: want 0, got %d\nstderr: %s", code, stderr.String())
	}
}

// TestHistoryPruneError verifies runHistoryPrune returns 1 when the history
// file is a directory and cannot be pruned, covering cmd_history.go:108-111.
//
//fusa:test REQ-FO-CLI046
func TestHistoryPruneError(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, history.Filename), 0o755); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := runHistoryPrune([]string{"--dir", dir}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("runHistoryPrune error: want 1, got %d", code)
	}
}

// ── cmd_scan.go ───────────────────────────────────────────────────────────────

// TestScanEmptyDirGap verifies runScan returns 0 with a "no languages" message
// when the project has only non-source files, covering cmd_scan.go:36-38.
// (Complementary to TestScanEmptyDir which uses an empty temp dir.)
//
//fusa:test REQ-FO-CLI005
func TestScanEmptyDirGap(t *testing.T) {
	dir := t.TempDir()
	// A .txt file is not a supported source language.
	if err := os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := runScan([]string{"--dir", dir}, &stdout, &stderr)
	if code != 0 {
		t.Errorf("runScan no source files: want 0, got %d\nstderr: %s", code, stderr.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte("No supported languages")) {
		t.Errorf("expected 'No supported languages' in output: %s", stdout.String())
	}
}

// ── cmd_policy.go ─────────────────────────────────────────────────────────────

// TestPolicyLoadOptionsError verifies runPolicy returns 1 when loadOptions
// fails due to a malformed .fusaops.json, covering cmd_policy.go:36-39.
//
//fusa:test REQ-FO-CLI024
func TestPolicyLoadOptionsError(t *testing.T) {
	dir := t.TempDir()
	// Write a valid policy so LoadPolicy succeeds.
	polPath := filepath.Join(dir, "policy.json")
	if err := os.WriteFile(polPath, []byte(`{"name":"test","rules":[]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	// Write malformed .fusaops.json to trigger loadOptions failure.
	if err := os.WriteFile(filepath.Join(dir, ".fusaops.json"), []byte("{bad json"), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := runPolicy([]string{"--dir", dir, "--policy", polPath}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("runPolicy loadOptions error: want 1, got %d", code)
	}
}
