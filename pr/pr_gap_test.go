package pr_test

// Gap test for pr.go: Load read error when the problems file exists but cannot
// be read (it is a directory), covering pr.go:99.3,99.63.
// The bad-JSON path (pr.go:114) is already covered by TestLoadBadJSON in pr_test.go.

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SoundMatt/FuSaOps/pr"
)

// TestLoadReadErrorGap verifies Load returns a non-nil error when the problems
// file exists but is a directory (EISDIR), covering pr.go:99.3,99.63.
//
//fusa:test REQ-FO-PR002
func TestLoadReadErrorGap(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, pr.ProblemsFile), 0o755); err != nil {
		t.Fatal(err)
	}
	_, err := pr.Load(dir)
	if err == nil {
		t.Fatal("pr.Load: expected error when problems file is a directory, got nil")
	}
}
