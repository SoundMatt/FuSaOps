package scan

// Gap tests covering uncovered branches in scan.go.

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// TestScanUnreadableSubdir verifies that Scan returns an error when the walk
// encounters a directory it cannot read, covering scan.go:78.17,80.4 and
// scan.go:97.16,99.3.
//
//fusa:test REQ-FO-SCAN003
func TestScanUnreadableSubdir(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix permission semantics (0o000) not available on Windows")
	}
	root := t.TempDir()
	// Create a subdirectory that cannot be entered on the walk.
	unreadable := filepath.Join(root, "restricted")
	if err := os.Mkdir(unreadable, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(unreadable, 0o755) })

	_, err := Scan(root)
	if err == nil {
		t.Error("Scan: expected error for unreadable subdirectory, got nil")
	}
}
