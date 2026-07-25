package comp_test

import (
	"testing"

	"github.com/SoundMatt/FuSaOps/comp"
)

//fusa:test REQ-FO-COMP001
func TestReportTypes(t *testing.T) {
	r := comp.Report{
		Threshold:      10,
		TotalFunctions: 5,
		Violations:     2,
		Results: []comp.Function{
			{File: "main.go", Line: 10, Name: "Foo", Complexity: 12, ExceedsThreshold: true},
		},
	}
	if r.Threshold != 10 || r.TotalFunctions != 5 || r.Violations != 2 {
		t.Errorf("Report fields wrong: %+v", r)
	}
	if len(r.Results) != 1 || !r.Results[0].ExceedsThreshold {
		t.Errorf("Results wrong: %+v", r.Results)
	}
}

//fusa:test REQ-FO-COMP002
func TestNew(t *testing.T) {
	components := []comp.ComponentComp{
		{
			Language: "go", Tool: "gofusa",
			Report: &comp.Report{TotalFunctions: 20, Violations: 2},
		},
		{
			Language: "c", Tool: "cfusa",
			Report: &comp.Report{TotalFunctions: 15, Violations: 0},
		},
		{
			Language: "cpp", Tool: "cpfusa",
			Skipped: "binary not found",
		},
	}
	agg := comp.New("/repo", "my-project", components)
	if agg.TotalFunctions != 35 {
		t.Errorf("TotalFunctions = %d, want 35", agg.TotalFunctions)
	}
	if agg.Violations != 2 {
		t.Errorf("Violations = %d, want 2", agg.Violations)
	}
	if !agg.HasViolations() {
		t.Error("HasViolations should be true")
	}
	if agg.Root != "/repo" || agg.Project != "my-project" {
		t.Errorf("Root/Project wrong: %q %q", agg.Root, agg.Project)
	}
}

//fusa:test REQ-FO-COMP002
func TestNewNoViolations(t *testing.T) {
	agg := comp.New("/r", "", []comp.ComponentComp{
		{Language: "go", Tool: "gofusa", Report: &comp.Report{TotalFunctions: 10, Violations: 0}},
	})
	if agg.HasViolations() {
		t.Error("HasViolations should be false")
	}
}

//fusa:test REQ-FO-COMP002
func TestNewSkippedIgnored(t *testing.T) {
	agg := comp.New("/r", "", []comp.ComponentComp{
		{Language: "go", Tool: "gofusa", Skipped: "unavailable"},
	})
	if agg.TotalFunctions != 0 || agg.Violations != 0 {
		t.Errorf("skipped component must not contribute: %+v", agg)
	}
}

//fusa:test REQ-FO-COMP001
func TestDALThreshold(t *testing.T) {
	cases := []struct {
		dal  string
		want int
	}{
		{"DAL-A", 4},
		{"DAL-B", 10},
		{"DAL-C", 15},
		{"DAL-D", 20},
		{"", 0},
		{"unknown", 0},
	}
	for _, c := range cases {
		if got := comp.DALThreshold(c.dal); got != c.want {
			t.Errorf("DALThreshold(%q) = %d, want %d", c.dal, got, c.want)
		}
	}
}

//fusa:test REQ-FO-COMP001
func TestValidateDAL(t *testing.T) {
	for _, valid := range []string{"", "DAL-A", "DAL-B", "DAL-C", "DAL-D"} {
		if err := comp.ValidateDAL(valid); err != nil {
			t.Errorf("ValidateDAL(%q) unexpected error: %v", valid, err)
		}
	}
	if err := comp.ValidateDAL("DAL-Z"); err == nil {
		t.Error("ValidateDAL(DAL-Z) should return error")
	}
}
