package trace

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const sampleMatrix = `{
  "requirements": [
    {"id":"REQ-A001","title":"A","standard":"ISO 26262","level":"HLR","asil":"ASIL-C"}
  ],
  "tags": [
    {"requirementId":"REQ-A001","file":"a.go","line":3,"kind":"req"},
    {"requirementId":"REQ-A001","file":"a_test.go","line":9,"kind":"test"}
  ],
  "coverage": {"totalRequirements":1,"tracedRequirements":1,"testedRequirements":1,"secTestedRequirements":1}
}`

//fusa:test REQ-FO-TRC001
//fusa:test REQ-FO-TRC002
//fusa:test REQ-FO-TRC003
//fusa:test REQ-FO-TRC004
func TestMatrixDecode(t *testing.T) {
	var m Matrix
	if err := json.Unmarshal([]byte(sampleMatrix), &m); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(m.Requirements) != 1 || m.Requirements[0].ASIL != "ASIL-C" {
		t.Errorf("requirement decode wrong: %+v", m.Requirements)
	}
	if len(m.Tags) != 2 || m.Tags[1].Kind != "test" {
		t.Errorf("tag decode wrong: %+v", m.Tags)
	}
	if m.Coverage.TestedRequirements != 1 || m.Coverage.SecTestedRequirements != 1 {
		t.Errorf("coverage decode wrong: %+v", m.Coverage)
	}
	if m.Requirements[0].Status != "" {
		t.Errorf("status should be absent when not emitted: %q", m.Requirements[0].Status)
	}
}

//fusa:test REQ-FO-TRC001
func TestRequirementStatus(t *testing.T) {
	const data = `{"requirements":[
		{"id":"R1","status":"covered"},
		{"id":"R2","status":"untraced"},
		{"id":"R3","status":"untested"}
	],"tags":[],"coverage":{}}`
	var m Matrix
	if err := json.Unmarshal([]byte(data), &m); err != nil {
		t.Fatal(err)
	}
	if m.Requirements[0].Status != "covered" || m.Requirements[1].Status != "untraced" || m.Requirements[2].Status != "untested" {
		t.Errorf("status decode wrong: %+v", m.Requirements)
	}
}

//fusa:test REQ-FO-TRC005
//fusa:test REQ-FO-TRC006
//fusa:test REQ-FO-TRC007
//fusa:test REQ-FO-TRC016
func TestComponentPct(t *testing.T) {
	c := ComponentTrace{
		Tool:          "gofusa",
		Coverage:      Coverage{TotalRequirements: 4, TracedRequirements: 3, TestedRequirements: 2, SecTestedRequirements: 1},
		Qualification: &Qualification{Total: 10, Passed: 10},
	}
	if c.TracedPct() != 75 || c.TestedPct() != 50 || c.SecTestedPct() != 25 {
		t.Errorf("pct wrong: traced=%d tested=%d sec-tested=%d", c.TracedPct(), c.TestedPct(), c.SecTestedPct())
	}
	// Zero requirements counts as fully covered, never a divide-by-zero.
	empty := ComponentTrace{}
	if empty.TracedPct() != 100 || empty.TestedPct() != 100 || empty.SecTestedPct() != 100 {
		t.Errorf("empty pct should be 100, got traced=%d tested=%d sec=%d", empty.TracedPct(), empty.TestedPct(), empty.SecTestedPct())
	}
}

//fusa:test REQ-FO-TRC008
//fusa:test REQ-FO-TRC009
//fusa:test REQ-FO-TRC010
//fusa:test REQ-FO-TRC016
func TestNewAggregates(t *testing.T) {
	a := New("/r", "proj", []ComponentTrace{
		{Tool: "gofusa", Available: true, Coverage: Coverage{TotalRequirements: 10, TracedRequirements: 10, TestedRequirements: 8, SecTestedRequirements: 6}},
		{Tool: "cfusa", Available: true, Coverage: Coverage{TotalRequirements: 5, TracedRequirements: 4, TestedRequirements: 4, SecTestedRequirements: 2}},
		{Tool: "cpfusa", Available: false, Skipped: "not installed", Coverage: Coverage{TotalRequirements: 99, SecTestedRequirements: 99}},
	})
	// Components are sorted by tool name.
	if a.Components[0].Tool != "cfusa" {
		t.Errorf("components not sorted: %s first", a.Components[0].Tool)
	}
	// Skipped component (cpfusa) must NOT inflate totals.
	if a.Coverage.TotalRequirements != 15 {
		t.Errorf("total should exclude skipped: got %d, want 15", a.Coverage.TotalRequirements)
	}
	if a.Coverage.TracedRequirements != 14 || a.Coverage.TestedRequirements != 12 {
		t.Errorf("sums wrong: %+v", a.Coverage)
	}
	if a.Coverage.SecTestedRequirements != 8 {
		t.Errorf("sec-tested sum wrong: got %d, want 8", a.Coverage.SecTestedRequirements)
	}
	if a.Coverage.TracedPct != 93 || a.Coverage.TestedPct != 80 {
		t.Errorf("pct wrong: %+v", a.Coverage)
	}
	if a.Coverage.SecTestedPct != 53 {
		t.Errorf("sec-tested pct wrong: got %d, want 53", a.Coverage.SecTestedPct)
	}
	if a.Project != "proj" || a.Root != "/r" {
		t.Errorf("metadata wrong: %+v", a)
	}
}

//fusa:test REQ-FO-TRC011
func TestStatus(t *testing.T) {
	full := New("/r", "", []ComponentTrace{{Tool: "t", Available: true, Coverage: Coverage{TotalRequirements: 2, TracedRequirements: 2, TestedRequirements: 2}}})
	if full.HasGaps() || full.Status() != "PASS" {
		t.Errorf("full coverage should PASS, got %s", full.Status())
	}
	gap := New("/r", "", []ComponentTrace{{Tool: "t", Available: true, Coverage: Coverage{TotalRequirements: 2, TracedRequirements: 1, TestedRequirements: 1}}})
	if !gap.HasGaps() || gap.Status() != "GAP" {
		t.Errorf("partial coverage should GAP, got %s", gap.Status())
	}
}

// TestFilterGaps verifies that FilterGaps reduces Requirements to gapped-only
// entries, using both Status field and tag-derived coverage.
//
//fusa:test REQ-FO-TRC017
func TestFilterGaps(t *testing.T) {
	agg := &Aggregate{
		Components: []ComponentTrace{
			{
				Tool: "gofusa",
				Requirements: []Requirement{
					{ID: "R1", Status: "covered"},
					{ID: "R2", Status: "untraced"},
					{ID: "R3", Status: "untested"},
				},
				Tags: []Tag{
					{RequirementID: "R1", Kind: "impl"},
					{RequirementID: "R1", Kind: "test"},
				},
				Coverage: Coverage{TotalRequirements: 3, TracedRequirements: 1, TestedRequirements: 1},
			},
		},
		Coverage: AggregateCoverage{TotalRequirements: 3},
	}

	filtered := FilterGaps(agg)

	// Only R2 and R3 should remain; R1 is "covered".
	comp := filtered.Components[0]
	if len(comp.Requirements) != 2 {
		t.Fatalf("expected 2 gaps, got %d: %+v", len(comp.Requirements), comp.Requirements)
	}
	ids := map[string]bool{comp.Requirements[0].ID: true, comp.Requirements[1].ID: true}
	if !ids["R2"] || !ids["R3"] {
		t.Errorf("expected R2 and R3 as gaps, got %+v", comp.Requirements)
	}
	// Coverage totals must be preserved (gate unchanged).
	if filtered.Coverage.TotalRequirements != 3 {
		t.Errorf("coverage total must not change: %+v", filtered.Coverage)
	}
}

// TestFilterGapsTagDerived verifies gap detection from tags when Status is empty.
//
//fusa:test REQ-FO-TRC017
func TestFilterGapsTagDerived(t *testing.T) {
	agg := &Aggregate{
		Components: []ComponentTrace{
			{
				Tool: "gofusa",
				Requirements: []Requirement{
					{ID: "R1"}, // traced+tested via tags → covered
					{ID: "R2"}, // impl only → untested
					{ID: "R3"}, // no tags → untraced
				},
				Tags: []Tag{
					{RequirementID: "R1", Kind: "impl"},
					{RequirementID: "R1", Kind: "test"},
					{RequirementID: "R2", Kind: "impl"},
				},
			},
		},
	}

	filtered := FilterGaps(agg)
	comp := filtered.Components[0]
	if len(comp.Requirements) != 2 {
		t.Fatalf("expected 2 gaps (R2, R3), got %d: %+v", len(comp.Requirements), comp.Requirements)
	}
	ids := map[string]bool{comp.Requirements[0].ID: true, comp.Requirements[1].ID: true}
	if !ids["R2"] || !ids["R3"] {
		t.Errorf("expected R2 and R3 as gaps, got %+v", comp.Requirements)
	}
}

func sampleAggregate() *Aggregate {
	return New("/r", "proj", []ComponentTrace{
		{Tool: "gofusa", Language: "go", Available: true,
			Coverage:      Coverage{TotalRequirements: 4, TracedRequirements: 4, TestedRequirements: 3},
			Qualification: &Qualification{Total: 2, Passed: 2}},
		{Tool: "cfusa", Language: "c", Available: false, Skipped: "cfusa binary not found on PATH"},
	})
}

//fusa:test REQ-FO-TRC012
//fusa:test REQ-FO-TRC013
func TestRenderJSON(t *testing.T) {
	var buf bytes.Buffer
	if err := Render(&buf, sampleAggregate(), "json"); err != nil {
		t.Fatal(err)
	}
	var back Aggregate
	if err := json.Unmarshal(buf.Bytes(), &back); err != nil {
		t.Fatalf("round-trip: %v", err)
	}
	if back.Coverage.TracedRequirements != 4 {
		t.Errorf("json lost coverage: %+v", back.Coverage)
	}
}

//fusa:test REQ-FO-TRC014
func TestRenderText(t *testing.T) {
	var buf bytes.Buffer
	if err := Render(&buf, sampleAggregate(), "text"); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{"Cross-Language Traceability", "gofusa", "skipped:", "qualification: 2/2", "TOTAL:"} {
		if !strings.Contains(out, want) {
			t.Errorf("text missing %q\n%s", want, out)
		}
	}
}

//fusa:test REQ-FO-TRC014
//fusa:test REQ-FO-TRC016
func TestRenderTextSecTested(t *testing.T) {
	agg := New("/r", "", []ComponentTrace{
		{Tool: "gofusa", Language: "go", Available: true,
			Coverage: Coverage{TotalRequirements: 10, TracedRequirements: 10, TestedRequirements: 8, SecTestedRequirements: 5}},
	})
	var buf bytes.Buffer
	if err := Render(&buf, agg, "text"); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "sec-tested") {
		t.Errorf("text should show sec-tested when non-zero:\n%s", out)
	}
}

//fusa:test REQ-FO-TRC017
func TestRenderTextGaps(t *testing.T) {
	agg := &Aggregate{
		Root: "/r",
		Components: []ComponentTrace{
			{Tool: "gofusa", Language: "go", Available: true,
				Requirements: []Requirement{{ID: "REQ-001", Title: "Safety init", Status: "untraced"}},
				Coverage:     Coverage{TotalRequirements: 5, TracedRequirements: 4, TestedRequirements: 4},
			},
		},
		Coverage: AggregateCoverage{TotalRequirements: 5, TracedRequirements: 4, TestedRequirements: 4},
	}
	agg.Coverage.TracedPct = pct(4, 5)
	agg.Coverage.TestedPct = pct(4, 5)

	var buf bytes.Buffer
	if err := Render(&buf, agg, "text"); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "REQ-001") || !strings.Contains(out, "Safety init") {
		t.Errorf("text should list gapped requirements:\n%s", out)
	}
}

//fusa:test REQ-FO-TRC015
func TestRenderHTML(t *testing.T) {
	var buf bytes.Buffer
	if err := Render(&buf, sampleAggregate(), "html"); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{"<!doctype html>", "Cross-Language Traceability", "gofusa", "skipped"} {
		if !strings.Contains(out, want) {
			t.Errorf("html missing %q", want)
		}
	}
}

//fusa:test REQ-FO-TRC012
func TestRenderUnknownAndToFile(t *testing.T) {
	if err := Render(&bytes.Buffer{}, sampleAggregate(), "yaml"); err == nil {
		t.Error("expected error for unknown format")
	}
	path := filepath.Join(t.TempDir(), "trace.json")
	if err := RenderToFile(sampleAggregate(), "json", path); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("file not written: %v", err)
	}
	if err := RenderToFile(sampleAggregate(), "json", filepath.Join(t.TempDir(), "nope", "x.json")); err == nil {
		t.Error("expected error creating file in missing dir")
	}
}
