package mcdc_test

import (
	"testing"

	"github.com/SoundMatt/FuSaOps/mcdc"
)

//fusa:test REQ-FO-MCDC001
//fusa:test REQ-FO-MCDC004
func TestReportCoveragePct(t *testing.T) {
	r := &mcdc.Report{TotalConditions: 10, CoveredConditions: 7}
	if got := r.CoveragePct(); got != 70 {
		t.Errorf("CoveragePct() = %d, want 70", got)
	}
}

//fusa:test REQ-FO-MCDC001
func TestReportCoveragePctZero(t *testing.T) {
	r := &mcdc.Report{TotalConditions: 0}
	if got := r.CoveragePct(); got != 100 {
		t.Errorf("CoveragePct() with 0 conditions = %d, want 100", got)
	}
}

//fusa:test REQ-FO-MCDC002
func TestNewAggregateEmpty(t *testing.T) {
	agg := mcdc.New("root", "proj", nil)
	if agg.GatePassed {
		t.Error("empty aggregate should not have GatePassed=true")
	}
	if agg.TotalConditions != 0 {
		t.Errorf("TotalConditions = %d, want 0", agg.TotalConditions)
	}
}

//fusa:test REQ-FO-MCDC002
func TestNewAggregateSkippedOnly(t *testing.T) {
	comps := []mcdc.MCDCComponent{
		{Language: "go", Tool: "gofusa", Skipped: "tool not installed"},
	}
	agg := mcdc.New(".", "p", comps)
	if agg.GatePassed {
		t.Error("skipped-only aggregate should not have GatePassed=true")
	}
}

//fusa:test REQ-FO-MCDC002
func TestNewAggregateSumsConditions(t *testing.T) {
	comps := []mcdc.MCDCComponent{
		{Language: "go", Tool: "gofusa", Report: &mcdc.Report{
			TotalConditions: 20, CoveredConditions: 18, GatePassed: true,
		}},
		{Language: "c", Tool: "cfusa", Report: &mcdc.Report{
			TotalConditions: 10, CoveredConditions: 9, GatePassed: true,
		}},
	}
	agg := mcdc.New(".", "p", comps)
	if agg.TotalConditions != 30 {
		t.Errorf("TotalConditions = %d, want 30", agg.TotalConditions)
	}
	if agg.CoveredConditions != 27 {
		t.Errorf("CoveredConditions = %d, want 27", agg.CoveredConditions)
	}
	if !agg.GatePassed {
		t.Error("GatePassed should be true when all components passed")
	}
}

//fusa:test REQ-FO-MCDC002
func TestNewAggregateGateFailsOnComponent(t *testing.T) {
	comps := []mcdc.MCDCComponent{
		{Language: "go", Tool: "gofusa", Report: &mcdc.Report{
			TotalConditions: 10, CoveredConditions: 10, GatePassed: true,
		}},
		{Language: "c", Tool: "cfusa", Report: &mcdc.Report{
			TotalConditions: 10, CoveredConditions: 5, GatePassed: false,
		}},
	}
	agg := mcdc.New(".", "p", comps)
	if agg.GatePassed {
		t.Error("GatePassed should be false when any component failed")
	}
}

//fusa:test REQ-FO-MCDC002
//fusa:test REQ-FO-MCDC004
func TestAggregateCoveragePct(t *testing.T) {
	comps := []mcdc.MCDCComponent{
		{Language: "go", Tool: "gofusa", Report: &mcdc.Report{
			TotalConditions: 4, CoveredConditions: 3, GatePassed: true,
		}},
	}
	agg := mcdc.New(".", "p", comps)
	if got := agg.CoveragePct(); got != 75 {
		t.Errorf("CoveragePct() = %d, want 75", got)
	}
}

//fusa:test REQ-FO-MCDC002
func TestAggregateCoveragePctZero(t *testing.T) {
	agg := mcdc.New(".", "p", nil)
	if got := agg.CoveragePct(); got != 100 {
		t.Errorf("CoveragePct() with 0 conditions = %d, want 100", got)
	}
}

//fusa:test REQ-FO-MCDC003
func TestMCDCComponentFields(t *testing.T) {
	c := mcdc.MCDCComponent{
		Language: "rust",
		Tool:     "rsfusa",
		Skipped:  "",
		Report: &mcdc.Report{
			TotalConditions: 5, CoveredConditions: 5, GatePassed: true,
		},
	}
	if c.Language != "rust" {
		t.Errorf("Language = %q, want %q", c.Language, "rust")
	}
	if c.Tool != "rsfusa" {
		t.Errorf("Tool = %q, want %q", c.Tool, "rsfusa")
	}
	if c.Skipped != "" {
		t.Errorf("Skipped = %q, want %q", c.Skipped, "")
	}
	if !c.Report.GatePassed {
		t.Error("GatePassed should be true")
	}
}
