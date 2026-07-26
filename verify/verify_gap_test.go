package verify

// Additional tests targeting the uncovered error branches in Save and Run.

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// TestSaveBadPath verifies Save returns an error when the output directory
// does not exist, covering the os.WriteFile failure branch.
//
//fusa:test REQ-FO-VER004
func TestSaveBadPath(t *testing.T) {
	b := New("/tmp", nil)
	err := Save("/nonexistent-dir-xyz/bundle.json", b)
	if err == nil {
		t.Error("Save to nonexistent dir: want error, got nil")
	}
}

// TestRunNonExistentDir verifies Run returns a non-nil error when the target
// directory does not exist, covering the !errors.As(err, &exitErr) branch.
//
//fusa:test REQ-FO-VER003
func TestRunNonExistentDir(t *testing.T) {
	_, err := Run(context.Background(), "/nonexistent-dir-xyz/no-module-here")
	if err == nil {
		t.Error("Run with non-existent dir: want error, got nil")
	}
}

// TestRunValidModule verifies Run reaches the Parse call on the success path
// (verify.go:142) by running go test on a minimal module with no test files.
// The command exits 0 and the results are returned without error.
//
//fusa:test REQ-FO-VER003
func TestRunValidModule(t *testing.T) {
	dir := t.TempDir()
	goMod := "module testmod\n\ngo 1.21\n"
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\nfunc main(){}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	results, err := Run(context.Background(), dir)
	if err != nil {
		t.Fatalf("Run on minimal module: %v", err)
	}
	_ = results // may be empty; we only care that Parse was called without error
}
