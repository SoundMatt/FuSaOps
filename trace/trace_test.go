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
  "coverage": {"totalRequirements":1,"tracedRequirements":1,"testedRequirements":1}
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
	if m.Coverage.TestedRequirements != 1 {
		t.Errorf("coverage decode wrong: %+v", m.Coverage)
	}
}

//fusa:test REQ-FO-TRC005
//fusa:test REQ-FO-TRC006
//fusa:test REQ-FO-TRC007
func TestComponentPct(t *testing.T) {
	c := ComponentTrace{
		Tool:          "gofusa",
		Coverage:      Coverage{TotalRequirements: 4, TracedRequirements: 3, TestedRequirements: 2},
		Qualification: &Qualification{Total: 10, Passed: 10},
	}
	if c.TracedPct() != 75 || c.TestedPct() != 50 {
		t.Errorf("pct wrong: traced=%d tested=%d", c.TracedPct(), c.TestedPct())
	}
	// Zero requirements counts as fully covered, never a divide-by-zero.
	empty := ComponentTrace{}
	if empty.TracedPct() != 100 || empty.TestedPct() != 100 {
		t.Errorf("empty pct should be 100, got %d/%d", empty.TracedPct(), empty.TestedPct())
	}
}

//fusa:test REQ-FO-TRC008
//fusa:test REQ-FO-TRC009
//fusa:test REQ-FO-TRC010
func TestNewAggregates(t *testing.T) {
	a := New("/r", "proj", []ComponentTrace{
		{Tool: "gofusa", Available: true, Coverage: Coverage{TotalRequirements: 10, TracedRequirements: 10, TestedRequirements: 8}},
		{Tool: "cfusa", Available: true, Coverage: Coverage{TotalRequirements: 5, TracedRequirements: 4, TestedRequirements: 4}},
		{Tool: "cpfusa", Available: false, Skipped: "not installed", Coverage: Coverage{TotalRequirements: 99}},
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
	if a.Coverage.TracedPct != 93 || a.Coverage.TestedPct != 80 {
		t.Errorf("pct wrong: %+v", a.Coverage)
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
