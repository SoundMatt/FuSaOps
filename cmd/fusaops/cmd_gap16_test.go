package main

// Gap tests covering uncovered branches in:
//   - cmd_release.go: runRelease SaveJSON manifest error (line 113-116)

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/SoundMatt/FuSaOps/release"
)

// TestReleaseSaveManifestError verifies runRelease returns 1 when SaveJSON for
// the artifact manifest fails because the manifest path is pre-occupied by a
// directory. The provenance file is created and hashed successfully before the
// manifest save is attempted, so this specifically covers the manifest save
// error branch at cmd_release.go:113.73,116.3.
//
//fusa:test REQ-FO-CLI065
func TestReleaseSaveManifestError(t *testing.T) {
	dir := t.TempDir()
	outDir := t.TempDir()

	// Pre-create artifact-manifest.json as a directory so release.SaveJSON
	// cannot write to it (os.WriteFile will fail with "is a directory").
	manifestDir := filepath.Join(outDir, release.ManifestFile)
	if err := os.MkdirAll(manifestDir, 0o750); err != nil {
		t.Fatalf("mkdir manifest dir: %v", err)
	}

	var stdout, stderr bytes.Buffer
	code := runRelease([]string{"--dir", dir, "--output-dir", outDir}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("runRelease save-manifest error: want 1, got %d (stderr: %s)", code, stderr.String())
	}
}
