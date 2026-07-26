package fmea

import "testing"

// TestPriorityLabelCritical verifies priorityLabel returns "CRITICAL" for
// RPN values above 200, covering the first case branch.
//
//fusa:test REQ-FO-FMEA004
func TestPriorityLabelCritical(t *testing.T) {
	got := priorityLabel(201)
	if got != "CRITICAL" {
		t.Errorf("priorityLabel(201): want %q, got %q", "CRITICAL", got)
	}
}
