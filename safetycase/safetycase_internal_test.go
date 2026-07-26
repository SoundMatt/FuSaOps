package safetycase

import (
	"testing"
)

// TestHashFileOpenError verifies hashFile returns an empty string when
// os.Open fails (non-existent path), covering the error-return branch.
//
//fusa:test REQ-FO-SC002
func TestHashFileOpenError(t *testing.T) {
	got := hashFile("/nonexistent/path/that/does/not/exist.txt")
	if got != "" {
		t.Errorf("hashFile non-existent: want empty string, got %q", got)
	}
}
