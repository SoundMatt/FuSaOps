package trace

import (
	"strings"
	"testing"
)

// helper builds a minimal Aggregate with a single non-skipped component.
func aggWith(reqs []Requirement) *Aggregate {
	return &Aggregate{
		Components: []ComponentTrace{
			{Tool: "gofusa", Language: "go", Available: true, Requirements: reqs},
		},
	}
}

// TestCheckDecompositionNoLevels verifies that requirements with empty Level
// produce zero violations so legacy matrices remain unaffected.
//
//fusa:test REQ-FO-TRC021
func TestCheckDecompositionNoLevels(t *testing.T) {
	agg := aggWith([]Requirement{
		{ID: "REQ-001"},
		{ID: "REQ-002"},
	})
	report := CheckDecomposition(agg)
	if report == nil {
		t.Fatal("expected non-nil report")
	}
	if !report.Valid() {
		t.Errorf("expected PASS for legacy requirements, got violations: %v", report.Violations)
	}
	if report.HLRCount != 0 || report.LLRCount != 0 {
		t.Errorf("expected zero HLR/LLR counts, got %d/%d", report.HLRCount, report.LLRCount)
	}
}

// TestCheckDecompositionValid verifies that 2 HLRs each with >=1 LLR child
// produce a passing report.
//
//fusa:test REQ-FO-TRC021
func TestCheckDecompositionValid(t *testing.T) {
	agg := aggWith([]Requirement{
		{ID: "HLR-001", Level: LevelHLR},
		{ID: "HLR-002", Level: LevelHLR},
		{ID: "LLR-001", Level: LevelLLR, Parent: "HLR-001"},
		{ID: "LLR-002", Level: LevelLLR, Parent: "HLR-001"},
		{ID: "LLR-003", Level: LevelLLR, Parent: "HLR-002"},
	})
	report := CheckDecomposition(agg)
	if !report.Valid() {
		t.Errorf("expected PASS, got violations: %v", report.Violations)
	}
	if report.HLRCount != 2 || report.LLRCount != 3 {
		t.Errorf("expected 2 HLR and 3 LLR, got %d/%d", report.HLRCount, report.LLRCount)
	}
}

// TestCheckDecompositionOrphanLLR verifies that an LLR referencing an unknown
// HLR produces an "orphan-llr" violation.
//
//fusa:test REQ-FO-TRC021
func TestCheckDecompositionOrphanLLR(t *testing.T) {
	agg := aggWith([]Requirement{
		{ID: "HLR-001", Level: LevelHLR},
		{ID: "LLR-001", Level: LevelLLR, Parent: "HLR-001"},
		{ID: "LLR-002", Level: LevelLLR, Parent: "HLR-NOPE"}, // unknown parent
	})
	report := CheckDecomposition(agg)
	if report.Valid() {
		t.Fatal("expected violations, got none")
	}
	found := false
	for _, v := range report.Violations {
		if v.Kind == "orphan-llr" && v.RequirementID == "LLR-002" && v.ParentID == "HLR-NOPE" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected orphan-llr for LLR-002, got %v", report.Violations)
	}
}

// TestCheckDecompositionUnparentedLLR verifies that an LLR with no Parent
// produces an "unparented-llr" violation.
//
//fusa:test REQ-FO-TRC021
func TestCheckDecompositionUnparentedLLR(t *testing.T) {
	agg := aggWith([]Requirement{
		{ID: "HLR-001", Level: LevelHLR},
		{ID: "LLR-001", Level: LevelLLR, Parent: "HLR-001"},
		{ID: "LLR-002", Level: LevelLLR}, // no parent
	})
	report := CheckDecomposition(agg)
	if report.Valid() {
		t.Fatal("expected violations, got none")
	}
	found := false
	for _, v := range report.Violations {
		if v.Kind == "unparented-llr" && v.RequirementID == "LLR-002" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected unparented-llr for LLR-002, got %v", report.Violations)
	}
}

// TestCheckDecompositionChildlessHLR verifies that an HLR with no LLR
// children produces a "childless-hlr" violation.
//
//fusa:test REQ-FO-TRC021
func TestCheckDecompositionChildlessHLR(t *testing.T) {
	agg := aggWith([]Requirement{
		{ID: "HLR-001", Level: LevelHLR},
		{ID: "HLR-002", Level: LevelHLR}, // no children
		{ID: "LLR-001", Level: LevelLLR, Parent: "HLR-001"},
	})
	report := CheckDecomposition(agg)
	if report.Valid() {
		t.Fatal("expected violations, got none")
	}
	found := false
	for _, v := range report.Violations {
		if v.Kind == "childless-hlr" && v.RequirementID == "HLR-002" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected childless-hlr for HLR-002, got %v", report.Violations)
	}
}

// TestCheckDecompositionSkipped verifies that skipped components are not
// inspected by the decomposition gate.
//
//fusa:test REQ-FO-TRC021
func TestCheckDecompositionSkipped(t *testing.T) {
	agg := &Aggregate{
		Components: []ComponentTrace{
			{
				Tool:     "cfusa",
				Language: "c",
				Skipped:  "not installed",
				// These HLR requirements without LLR would fail if inspected.
				Requirements: []Requirement{
					{ID: "HLR-001", Level: LevelHLR},
					{ID: "HLR-002", Level: LevelHLR},
				},
			},
		},
	}
	report := CheckDecomposition(agg)
	if !report.Valid() {
		t.Errorf("skipped component should produce no violations, got: %v", report.Violations)
	}
}

// TestCheckDecompositionCrossComponent verifies that an LLR in one component
// can reference an HLR from another component (cross-language linking).
//
//fusa:test REQ-FO-TRC021
func TestCheckDecompositionCrossComponent(t *testing.T) {
	agg := &Aggregate{
		Components: []ComponentTrace{
			{
				Tool:     "gofusa",
				Language: "go",
				Available: true,
				Requirements: []Requirement{
					{ID: "HLR-001", Level: LevelHLR},
				},
			},
			{
				Tool:     "cfusa",
				Language: "c",
				Available: true,
				Requirements: []Requirement{
					{ID: "LLR-001", Level: LevelLLR, Parent: "HLR-001"}, // cross-component reference
				},
			},
		},
	}
	report := CheckDecomposition(agg)
	if !report.Valid() {
		t.Errorf("cross-component HLR/LLR link should be valid, got violations: %v", report.Violations)
	}
}

// TestCheckDecompositionDeterministic verifies that two calls with the same
// aggregate produce identical violation order.
//
//fusa:test REQ-FO-TRC021
func TestCheckDecompositionDeterministic(t *testing.T) {
	agg := aggWith([]Requirement{
		{ID: "HLR-001", Level: LevelHLR},
		{ID: "HLR-002", Level: LevelHLR}, // childless
		{ID: "LLR-001", Level: LevelLLR, Parent: "HLR-001"},
		{ID: "LLR-002", Level: LevelLLR}, // unparented
		{ID: "LLR-003", Level: LevelLLR, Parent: "HLR-NOPE"}, // orphan
	})

	r1 := CheckDecomposition(agg)
	r2 := CheckDecomposition(agg)
	if len(r1.Violations) != len(r2.Violations) {
		t.Fatalf("non-deterministic violation count: %d vs %d", len(r1.Violations), len(r2.Violations))
	}
	for i := range r1.Violations {
		if r1.Violations[i] != r2.Violations[i] {
			t.Errorf("violation[%d] differs: %v vs %v", i, r1.Violations[i], r2.Violations[i])
		}
	}
}

// TestSeverityForDecompositionDAL verifies DAL severity mapping.
//
//fusa:test REQ-FO-TRC022
func TestSeverityForDecompositionDAL(t *testing.T) {
	cases := []struct {
		dal  string
		want string
	}{
		{"DAL-A", "error"},
		{"DAL-B", "error"},
		{"DAL-C", "warn"},
		{"DAL-D", "warn"},
		{"DAL-E", "warn"},
	}
	for _, tc := range cases {
		got := SeverityForDecomposition("", tc.dal, "")
		if got != tc.want {
			t.Errorf("DAL=%s: got %q, want %q", tc.dal, got, tc.want)
		}
	}
}

// TestSeverityForDecompositionASIL verifies ASIL severity mapping.
//
//fusa:test REQ-FO-TRC022
func TestSeverityForDecompositionASIL(t *testing.T) {
	cases := []struct {
		asil string
		want string
	}{
		{"ASIL-D", "error"},
		{"ASIL-C", "error"},
		{"ASIL-B", "warn"},
		{"ASIL-A", "warn"},
	}
	for _, tc := range cases {
		got := SeverityForDecomposition("", "", tc.asil)
		if got != tc.want {
			t.Errorf("ASIL=%s: got %q, want %q", tc.asil, got, tc.want)
		}
	}
}

// TestSeverityForDecompositionExplicit verifies that explicit enforce values
// override any DAL/ASIL.
//
//fusa:test REQ-FO-TRC022
func TestSeverityForDecompositionExplicit(t *testing.T) {
	if got := SeverityForDecomposition("error", "DAL-E", "ASIL-A"); got != "error" {
		t.Errorf("enforce=error: got %q", got)
	}
	if got := SeverityForDecomposition("warn", "DAL-A", "ASIL-D"); got != "warn" {
		t.Errorf("enforce=warn: got %q", got)
	}
	if got := SeverityForDecomposition("off", "DAL-A", "ASIL-D"); got != "off" {
		t.Errorf("enforce=off: got %q", got)
	}
}

// TestSeverityForDecompositionAuto verifies that enforce="" with no integrity
// level defaults to "warn".
//
//fusa:test REQ-FO-TRC022
func TestSeverityForDecompositionAuto(t *testing.T) {
	if got := SeverityForDecomposition("", "", ""); got != "warn" {
		t.Errorf("no integrity level: got %q, want %q", got, "warn")
	}
	if got := SeverityForDecomposition("auto", "", ""); got != "warn" {
		t.Errorf("auto with no level: got %q, want %q", got, "warn")
	}
}

// TestDecompositionViolationString verifies String() output for all three kinds.
//
//fusa:test REQ-FO-TRC020
func TestDecompositionViolationString(t *testing.T) {
	cases := []struct {
		v    DecompositionViolation
		want string
	}{
		{
			DecompositionViolation{Kind: "orphan-llr", RequirementID: "LLR-001", ParentID: "HLR-X", Component: "gofusa"},
			"[gofusa] LLR LLR-001 references unknown HLR HLR-X",
		},
		{
			DecompositionViolation{Kind: "unparented-llr", RequirementID: "LLR-002", Component: "cfusa"},
			"[cfusa] LLR LLR-002 has no parent HLR",
		},
		{
			DecompositionViolation{Kind: "childless-hlr", RequirementID: "HLR-001", Component: "gofusa"},
			"[gofusa] HLR HLR-001 has no LLR children",
		},
	}
	for _, tc := range cases {
		got := tc.v.String()
		if got != tc.want {
			t.Errorf("String() = %q, want %q", got, tc.want)
		}
	}
	// Unknown kind should not panic.
	unknown := DecompositionViolation{Kind: "unknown-kind", RequirementID: "X", Component: "tool"}
	s := unknown.String()
	if !strings.Contains(s, "unknown-kind") {
		t.Errorf("unknown kind String() missing kind: %q", s)
	}
}

// TestDecompositionReportValid verifies Valid() behaviour.
//
//fusa:test REQ-FO-TRC020
func TestDecompositionReportValid(t *testing.T) {
	var r *DecompositionReport
	if !r.Valid() {
		t.Error("nil DecompositionReport.Valid() should return true")
	}
	empty := &DecompositionReport{}
	if !empty.Valid() {
		t.Error("zero-violation DecompositionReport.Valid() should return true")
	}
	withViol := &DecompositionReport{
		Violations: []DecompositionViolation{{Kind: "childless-hlr", RequirementID: "X", Component: "tool"}},
	}
	if withViol.Valid() {
		t.Error("DecompositionReport with violations should return false")
	}
}
