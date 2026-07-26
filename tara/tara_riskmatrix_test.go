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
