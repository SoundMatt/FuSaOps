package comp_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/SoundMatt/FuSaOps/comp"
)

func makeAgg() *comp.Aggregate {
	return comp.New("/repo", "p", []comp.ComponentComp{
		{
			Language: "go", Tool: "gofusa",
			Report: &comp.Report{
				Threshold:      10,
				TotalFunctions: 3,
				Violations:     1,
				Results: []comp.Function{
					{File: "main.go", Line: 5, Name: "Foo", Complexity: 12, ExceedsThreshold: true},
					{File: "util.go", Line: 1, Name: "Bar", Complexity: 4, ExceedsThreshold: false},
				},
			},
		},
		{Language: "c", Tool: "cfusa", Skipped: "binary not found"},
	})
}

//fusa:test REQ-FO-COMP003
func TestRenderText(t *testing.T) {
	var buf bytes.Buffer
	if err := comp.Render(&buf, makeAgg(), "text"); err != nil {
		t.Fatalf("Render text: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "FAIL") {
		t.Errorf("expected FAIL in output:\n%s", out)
	}
	if !strings.Contains(out, "Foo") {
		t.Errorf("expected violating function name in output:\n%s", out)
	}
	if !strings.Contains(out, "SKIPPED") {
		t.Errorf("expected SKIPPED for unavailable component:\n%s", out)
	}
	if !strings.Contains(out, "TOTAL") {
		t.Errorf("expected TOTAL line:\n%s", out)
	}
}

//fusa:test REQ-FO-COMP003
func TestRenderTextPass(t *testing.T) {
	agg := comp.New("/r", "", []comp.ComponentComp{
		{Language: "go", Tool: "gofusa", Report: &comp.Report{Threshold: 10, TotalFunctions: 5, Violations: 0}},
	})
	var buf bytes.Buffer
	if err := comp.Render(&buf, agg, "text"); err != nil {
		t.Fatalf("Render text: %v", err)
	}
	if !strings.Contains(buf.String(), "PASS") {
		t.Errorf("expected PASS:\n%s", buf.String())
	}
}

//fusa:test REQ-FO-COMP003
func TestRenderJSON(t *testing.T) {
	var buf bytes.Buffer
	if err := comp.Render(&buf, makeAgg(), "json"); err != nil {
		t.Fatalf("Render json: %v", err)
	}
	var out comp.Aggregate
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if out.Violations != 1 {
		t.Errorf("Violations = %d, want 1", out.Violations)
	}
}

//fusa:test REQ-FO-COMP003
func TestRenderUnknownFormat(t *testing.T) {
	var buf bytes.Buffer
	if err := comp.Render(&buf, makeAgg(), "xml"); err == nil {
		t.Error("expected error for unknown format")
	}
}

//fusa:test REQ-FO-COMP003
func TestRenderToFile(t *testing.T) {
	path := t.TempDir() + "/comp.json"
	if err := comp.RenderToFile(makeAgg(), "json", path); err != nil {
		t.Fatalf("RenderToFile: %v", err)
	}
}
