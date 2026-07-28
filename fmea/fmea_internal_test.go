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

// TestCoveragePctNeverExceeds100 verifies coveragePct clamps to 100 even when
// analyzed > total (x-FuSa spec §9.2: "coveragePct MUST NOT exceed 100").
//
//fusa:test REQ-FO-FMEA008
func TestCoveragePctNeverExceeds100(t *testing.T) {
	if got := coveragePct(12, 10); got != 100 {
		t.Errorf("coveragePct(12, 10) = %v, want 100 (clamped)", got)
	}
}
