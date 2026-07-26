package verify

// Additional tests targeting the uncovered error branches in Save and Run.

import (
	"context"
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
