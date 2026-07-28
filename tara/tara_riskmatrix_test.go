package tara

import "testing"

// TestRiskMatrixUncoveredBranches directly exercises riskMatrix combinations
// that the standard scenarios do not cover: ImpactMajor×default,
// ImpactModerate×High/default, and the outer default (ImpactNegligible).
//
//fusa:test REQ-FO-TARA001
func TestRiskMatrixUncoveredBranches(t *testing.T) {
	cases := []struct {
		imp  Impact
		feas Feasibility
		want RiskLevel
	}{
		// ImpactMajor × default (FeasibilityLow) → RiskMedium
		{ImpactMajor, FeasibilityLow, RiskMedium},
		// ImpactModerate × FeasibilityHigh → RiskMedium
		{ImpactModerate, FeasibilityHigh, RiskMedium},
		// ImpactModerate × default (FeasibilityLow) → RiskLow
		{ImpactModerate, FeasibilityLow, RiskLow},
		// outer default (ImpactNegligible) → RiskLow
		{ImpactNegligible, FeasibilityHigh, RiskLow},
	}
	for _, tc := range cases {
		got := riskMatrix(tc.imp, tc.feas)
		if got != tc.want {
			t.Errorf("riskMatrix(%q, %q) = %q, want %q", tc.imp, tc.feas, got, tc.want)
		}
	}
}

// TestCoveragePctNeverExceeds100 verifies coveragePct clamps to 100 even when
// analyzed > total (x-FuSa spec §9.2: "coveragePct MUST NOT exceed 100").
//
//fusa:test REQ-FO-TARA008
func TestCoveragePctNeverExceeds100(t *testing.T) {
	if got := coveragePct(12, 10); got != 100 {
		t.Errorf("coveragePct(12, 10) = %v, want 100 (clamped)", got)
	}
}
