package main

// Gap tests covering uncovered branches across multiple cmd files:
// cmd_adapters.go, cmd_init.go, cmd_diff.go, cmd_trace.go,
// cmd_impact.go, cmd_slsa.go, cmd_hara.go.

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// ── cmd_adapters.go ─��─────────────────────────────────────────────────────────

// TestAdaptersBadFlag verifies runAdapters returns 2 for an unknown flag,
// covering the fs.Parse error branch (cmd_adapters.go:20.39,22.3).
//
//fusa:test REQ-FO-CLI050
func TestAdaptersBadFlag(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := runAdapters([]string{"--bogus-flag-xyz"}, &stdout, &stderr); code != 2 {
		t.Errorf("runAdapters bad flag: want 2, got %d", code)
	}
}

// ── cmd_init.go ───────────────────────────────────────────────────────────────

// TestInitBadFlagGap verifies runInit returns 2 for an unknown flag,
// covering the fs.Parse error branch (cmd_init.go:22.39,24.3).
//
//fusa:test REQ-FO-CLI004
func TestInitBadFlagGap(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := runInit([]string{"--bogus-flag-xyz"}, &stdout, &stderr); code != 2 {
		t.Errorf("runInit bad flag: want 2, got %d", code)
	}
}

// ── cmd_diff.go ───────────────────────────────────────────────────────────────

// TestDiffBadFlagNonHelp verifies runDiff returns 1 (not 0) for an unknown
// flag, covering the non-ErrHelp branch at cmd_diff.go:41.3,41.11.
//
//fusa:test REQ-FO-CLI018
func TestDiffBadFlagNonHelp(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runDiff([]string{"--bogus-flag-xyz"}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("runDiff bad flag: want 1, got %d", code)
	}
}

// TestDiffLoadOptionsErrorGap verifies runDiff returns 1 when loadOptions
// fails due to malformed .fusaops.json, covering cmd_diff.go:45.16,48.3.
//
//fusa:test REQ-FO-CLI018
func TestDiffLoadOptionsErrorGap(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".fusaops.json"), []byte("{bad json"), 0o644); err != nil {
		t.Fatal(err)
	}
	bl := filepath.Join(dir, "baseline.json")
	if err := os.WriteFile(bl, []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := runDiff([]string{"--dir", dir, "--baseline", bl}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("runDiff loadOptions error: want 1, got %d", code)
	}
}

// ── cmd_trace.go ──────────────────────────────────────────────────────────────

// TestTraceLoadOptionsErrorGap verifies runTrace returns 1 when loadOptions
// fails due to malformed .fusaops.json, covering cmd_trace.go:44.16,47.3.
//
//fusa:test REQ-FO-CLI011
func TestTraceLoadOptionsErrorGap(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".fusaops.json"), []byte("{bad json"), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := runTrace([]string{"--dir", dir}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("runTrace loadOptions error: want 1, got %d", code)
	}
}

// ── cmd_impact.go ─────────────────────────────────────────────────────────────

// TestImpactOutputCreateError verifies runImpact returns 1 when the output
// file cannot be created, covering cmd_impact.go:45.18,48.4.
// impact.Analyse returns nil error for a non-git dir, so os.Create is reached.
//
//fusa:test REQ-FO-CLI059
func TestImpactOutputCreateError(t *testing.T) {
	dir := t.TempDir()
	badOutput := filepath.Join(t.TempDir(), "nonexistent", "subdir", "impact.txt")
	var stdout, stderr bytes.Buffer
	code := runImpact([]string{"--dir", dir, "--output", badOutput}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("runImpact output create error: want 1, got %d", code)
	}
}

// ── cmd_slsa.go ───────────────────────────────────────────────────────────────

// TestSLSAOutputCreateError verifies runSLSA returns 1 when the output file
// cannot be created, covering cmd_slsa.go:50.18,53.4.
//
//fusa:test REQ-FO-CLI057
func TestSLSAOutputCreateError(t *testing.T) {
	dir := t.TempDir()
	badOutput := filepath.Join(t.TempDir(), "nonexistent", "subdir", "slsa.txt")
	var stdout, stderr bytes.Buffer
	code := runSLSA([]string{"--dir", dir, "--output", badOutput}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("runSLSA output create error: want 1, got %d", code)
	}
}

// ── cmd_hara.go ───────────────────────────────────────────────────────────────

// TestHaraInitBadFlag verifies runHaraInit returns 2 for an unknown flag,
// covering the fs.Parse error branch at cmd_hara.go:108.39,110.3.
//
//fusa:test REQ-FO-CLI073
func TestHaraInitBadFlag(t *testing.T) {
	dir := t.TempDir()
	var stdout, stderr bytes.Buffer
	if code := runHaraInit([]string{"--bogus-flag-xyz"}, dir, &stdout, &stderr); code != 2 {
		t.Errorf("runHaraInit bad flag: want 2, got %d", code)
	}
}

// TestHaraASILBadFlag verifies runHaraASIL returns 2 for an unknown flag,
// covering the fs.Parse error branch at cmd_hara.go:178.39,180.3.
//
//fusa:test REQ-FO-CLI073
func TestHaraASILBadFlag(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := runHaraASIL([]string{"--bogus-flag-xyz"}, &stdout, &stderr); code != 2 {
		t.Errorf("runHaraASIL bad flag: want 2, got %d", code)
	}
}
