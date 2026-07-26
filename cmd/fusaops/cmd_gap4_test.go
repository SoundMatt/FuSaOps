package main

// Additional tests targeting uncovered branches not covered by prior gap files.

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/SoundMatt/FuSaOps/config"
)

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
