package main

// Additional tests targeting uncovered branches not covered by prior gap files.

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/SoundMatt/FuSaOps/config"
	"github.com/SoundMatt/FuSaOps/disposition"
)

// ── runDispositionList ────────────────────────────────────────────────────────

// TestDispositionListLoadError verifies runDispositionList returns 1 when the
// dispositions file path exists as a directory (non-IsNotExist read error),
// covering the disposition.Load error branch.
//
//fusa:test REQ-FO-CLI060
func TestDispositionListLoadError(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, disposition.DispositionsFile), 0o750); err != nil {
		t.Fatal(err)
	}
	code := runDispositionList(dir, &bytes.Buffer{}, &bytes.Buffer{})
	if code != 1 {
		t.Errorf("disposition list load error: want 1, got %d", code)
	}
}

// ── runRelease ────────────────────────────────────────────────────────────────

// TestReleaseBadConfig verifies runRelease returns 1 when the project directory
// contains a malformed .fusaops.json, triggering the loadOptions error path.
//
//fusa:test REQ-FO-CLI065
func TestReleaseBadConfig(t *testing.T) {
	dir := t.TempDir()
	bad := []byte("{not valid json")
	if err := os.WriteFile(filepath.Join(dir, config.ConfigFile), bad, 0o644); err != nil {
		t.Fatal(err)
	}
	outDir := t.TempDir()
	var stdout, stderr bytes.Buffer
	code := runRelease([]string{"--dir", dir, "--output-dir", outDir}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("runRelease bad config: want 1, got %d (stderr=%q)", code, stderr.String())
	}
}
