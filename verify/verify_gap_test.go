package verify

// Additional tests targeting the uncovered error branch in Save.

import (
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
