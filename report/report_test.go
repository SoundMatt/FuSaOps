package report

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	fusaops "github.com/SoundMatt/FuSaOps"
)

func sampleComponents() []Component {
	return []Component{
		{Language: fusaops.LangGo, Tool: "gofusa", Available: true, Findings: []fusaops.Finding{
			{Language: fusaops.LangGo, Tool: "gofusa", RuleID: "LINT001", Severity: fusaops.SeverityWarning, Message: "warn", Location: fusaops.Location{File: "a.go", Line: 5}},
			{Language: fusaops.LangGo, Tool: "gofusa", RuleID: "FUSA001", Severity: fusaops.SeverityError, Message: "err"},
		}},
		{Language: fusaops.LangC, Tool: "cfusa", Available: false, Skipped: "cfusa binary not found on PATH"},
	}
}

//fusa:test REQ-FO-RPT001
//fusa:test REQ-FO-RPT003
//fusa:test REQ-FO-RPT004
//fusa:test REQ-FO-RPT005
//fusa:test REQ-FO-RPT006
func TestNewComputesSummaries(t *testing.T) {
	r := New("/root", "demo", sampleComponents())
	if r.Summary.Total != 2 || r.Summary.Errors != 1 || r.Summary.Warnings != 1 {
		t.Errorf("aggregate summary wrong: %+v", r.Summary)
	}
	// Components are sorted by tool: cfusa before gofusa.
	if r.Components[0].Tool != "cfusa" {
		t.Errorf("components not sorted: %s first", r.Components[0].Tool)
	}
	if !r.HasErrors() {
		t.Error("HasErrors should be true")
	}
}

//fusa:test REQ-FO-RPT002
func TestStatus(t *testing.T) {
	cases := []struct {
		s    Summary
		want string
	}{
		{Summary{Errors: 1}, "FAIL"},
		{Summary{Warnings: 1}, "WARN"},
		{Summary{Infos: 3}, "PASS"},
		{Summary{}, "PASS"},
	}
	for _, c := range cases {
		if got := c.s.Status(); got != c.want {
			t.Errorf("Status(%+v): got %s, want %s", c.s, got, c.want)
		}
	}
}

//fusa:test REQ-FO-RPT007
//fusa:test REQ-FO-RPT009
func TestRenderJSONRoundTrip(t *testing.T) {
	r := New("/root", "demo", sampleComponents())
	var buf bytes.Buffer
	if err := Render(&buf, r, "json"); err != nil {
		t.Fatal(err)
	}
	var back AggregateReport
	if err := json.Unmarshal(buf.Bytes(), &back); err != nil {
		t.Fatalf("json round trip: %v", err)
	}
	if back.Summary.Errors != 1 {
		t.Errorf("decoded summary wrong: %+v", back.Summary)
	}
}

//fusa:test REQ-FO-RPT010
func TestRenderText(t *testing.T) {
	r := New("/root", "demo", sampleComponents())
	var buf bytes.Buffer
	if err := Render(&buf, r, "text"); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{"FuSaOps", "FAIL", "LINT001", "skipped"} {
		if !strings.Contains(out, want) {
			t.Errorf("text output missing %q", want)
		}
	}
}

//fusa:test REQ-FO-RPT012
func TestRenderHTML(t *testing.T) {
	r := New("/root", "demo", sampleComponents())
	var buf bytes.Buffer
	if err := Render(&buf, r, "html"); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{"<!DOCTYPE html>", "FuSaOps", "status-fail", "LINT001"} {
		if !strings.Contains(out, want) {
			t.Errorf("html output missing %q", want)
		}
	}
}

//fusa:test REQ-FO-RPT011
func TestRenderSARIF(t *testing.T) {
	r := New("/root", "demo", sampleComponents())
	var buf bytes.Buffer
	if err := Render(&buf, r, "sarif"); err != nil {
		t.Fatal(err)
	}
	var log map[string]any
	if err := json.Unmarshal(buf.Bytes(), &log); err != nil {
		t.Fatalf("sarif not valid json: %v", err)
	}
	if log["version"] != "2.1.0" {
		t.Errorf("sarif version: got %v", log["version"])
	}
}

func TestRenderUnsupported(t *testing.T) {
	r := New("/root", "demo", nil)
	if err := Render(&bytes.Buffer{}, r, "xml"); err == nil {
		t.Error("expected error for unsupported format")
	}
}
