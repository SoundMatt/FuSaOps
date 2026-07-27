package doctemplate_test

// Gap tests covering uncovered branches in doctemplate.go.

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SoundMatt/FuSaOps/doctemplate"
)

// TestGenerateMkdirError verifies Generate returns an error when outputDir
// cannot be created (a file exists at a path component), covering
// doctemplate.go:439.54,441.3.
//
//fusa:test REQ-FO-TMPL002
func TestGenerateMkdirError(t *testing.T) {
	dir := t.TempDir()
	// Create a regular file where the output directory should be.
	blocker := filepath.Join(dir, "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	// MkdirAll("blocker/sub") fails because "blocker" is a file.
	outputDir := filepath.Join(blocker, "sub")
	_, err := doctemplate.Generate(dir, outputDir, nil)
	if err == nil {
		t.Error("Generate: expected error when outputDir cannot be created, got nil")
	}
}

// TestGenerateWriteError verifies Generate returns an error when a template
// file cannot be written (outputDir is read-only), covering
// doctemplate.go:446.73,448.4.
//
//fusa:test REQ-FO-TMPL002
func TestGenerateWriteError(t *testing.T) {
	root := t.TempDir()
	outputDir := filepath.Join(t.TempDir(), "out")
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Make directory read-only so template file creation fails.
	if err := os.Chmod(outputDir, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(outputDir, 0o755) })
	_, err := doctemplate.Generate(root, outputDir, []string{"iso26262"})
	if err == nil {
		t.Error("Generate: expected error when template file cannot be written, got nil")
	}
}
