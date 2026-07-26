package sci

import "testing"

// TestKindLabelDefault verifies kindLabel returns the raw string for unknown kinds.
//
//fusa:test REQ-FO-SCI004
func TestKindLabelDefault(t *testing.T) {
	got := kindLabel(ItemKind("unknown"))
	if got != "unknown" {
		t.Errorf("kindLabel(unknown): got %q, want %q", got, "unknown")
	}
}
