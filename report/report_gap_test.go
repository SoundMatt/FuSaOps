package report

// Gap tests covering uncovered branches in report sub-files.

import (
	"bytes"
	"strings"
	"testing"

	fusaops "github.com/SoundMatt/FuSaOps"
)

// TestTextFindingCategory verifies that a finding with a non-empty Category
// is rendered with the category label, covering text.go:73.22,75.3.
//
//fusa:test REQ-FO-RPT010
func TestTextFindingCategory(t *testing.T) {
	r := New("/root", "proj", []Component{{
		Language: fusaops.LangGo, Tool: "gofusa", Available: true,
		Findings: []fusaops.Finding{{
			Language: fusaops.LangGo, Tool: "gofusa",
			RuleID: "E001", Severity: fusaops.SeverityError,
			Message: "something", Category: "style",
		}},
	}})
	var buf bytes.Buffer
	if err := Render(&buf, r, "text"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "[style]") {
		t.Errorf("text: expected category [style] in output:\n%s", buf.String())
	}
}

// TestTextFindingRemediation verifies that a finding with a non-empty
// Remediation is rendered with the remediation hint, covering
// text.go:85.25,87.3.
//
//fusa:test REQ-FO-RPT010
func TestTextFindingRemediation(t *testing.T) {
	r := New("/root", "proj", []Component{{
		Language: fusaops.LangGo, Tool: "gofusa", Available: true,
		Findings: []fusaops.Finding{{
			Language: fusaops.LangGo, Tool: "gofusa",
			RuleID: "E001", Severity: fusaops.SeverityError,
			Message: "something", Remediation: "apply the fix",
		}},
	}})
	var buf bytes.Buffer
	if err := Render(&buf, r, "text"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "apply the fix") {
		t.Errorf("text: expected remediation in output:\n%s", buf.String())
	}
}

// TestMarkdownRendererSIL verifies that the SIL level is shown when r.SIL is
// set in the markdown renderer, covering markdown.go:31.18,33.4.
//
//fusa:test REQ-FO-RPT015
func TestMarkdownRendererSIL(t *testing.T) {
	r := New("/root", "proj", nil)
	r.Standard = "IEC61508"
	r.SIL = "SIL-2"
	var buf bytes.Buffer
	if err := Render(&buf, r, "markdown"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "SIL-2") {
		t.Errorf("markdown: expected SIL-2 in output:\n%s", buf.String())
	}
}

// TestMarkdownRendererDAL verifies that the DAL level is shown when r.DAL is
// set (and SIL is empty) in the markdown renderer, covering
// markdown.go:33.25,35.4.
//
//fusa:test REQ-FO-RPT015
func TestMarkdownRendererDAL(t *testing.T) {
	r := New("/root", "proj", nil)
	r.Standard = "DO-178C"
	r.DAL = "DAL-A"
	var buf bytes.Buffer
	if err := Render(&buf, r, "markdown"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "DAL-A") {
		t.Errorf("markdown: expected DAL-A in output:\n%s", buf.String())
	}
}

// TestMarkdownRendererStandardNoLevel verifies that the Standard is shown
// without a level when ASIL, SIL, and DAL are all empty, covering
// markdown.go:38.9,40.4.
//
//fusa:test REQ-FO-RPT015
func TestMarkdownRendererStandardNoLevel(t *testing.T) {
	r := New("/root", "proj", nil)
	r.Standard = "ISO21434"
	var buf bytes.Buffer
	if err := Render(&buf, r, "markdown"); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "ISO21434") {
		t.Errorf("markdown: expected ISO21434 in output:\n%s", out)
	}
	if strings.Contains(out, "ISO21434 (") {
		t.Errorf("markdown: unexpected parenthetical level after standard name:\n%s", out)
	}
}

// TestMarkdownEscapePipe verifies that markdownEscape replaces pipe characters
// with escaped forms, covering markdown.go:148.18,150.4.
//
//fusa:test REQ-FO-RPT015
func TestMarkdownEscapePipe(t *testing.T) {
	r := New("/root", "proj", []Component{{
		Language: fusaops.LangGo, Tool: "gofusa", Available: true,
		Findings: []fusaops.Finding{{
			Language: fusaops.LangGo, Tool: "gofusa",
			RuleID: "E001", Severity: fusaops.SeverityError,
			Message: "value a|b is invalid",
		}},
	}})
	var buf bytes.Buffer
	if err := Render(&buf, r, "markdown"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), `\|`) {
		t.Errorf("markdown: expected escaped pipe \\| in output:\n%s", buf.String())
	}
}

// TestHTMLCompThresholdLabel verifies that the compThresholdLabel template
// function is called correctly, covering html.go:45.20,47.4 (threshold>0, dal
// empty) and html.go:48.3,48.15 (both zero/empty).
//
//fusa:test REQ-FO-RPT021
func TestHTMLCompThresholdLabel(t *testing.T) {
	r := New("/root", "proj", []Component{{
		Language: fusaops.LangGo, Tool: "gofusa", Available: true,
	}})
	ci := &CompInfo{
		TotalFunctions: 5,
		Components: []CompComponent{
			{Language: "go", Tool: "gofusa", Threshold: 10, DAL: "", TotalFunctions: 3},
			{Language: "c", Tool: "cfusa", Threshold: 0, DAL: "", TotalFunctions: 2},
		},
	}
	var buf bytes.Buffer
	if err := RenderWithOptions(&buf, r, "html", RenderOptions{CompInfo: ci}); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "≤10") {
		t.Errorf("html: expected ≤10 threshold label, got output (truncated): %s", out[:min(200, len(out))])
	}
	if !strings.Contains(out, "—") {
		t.Errorf("html: expected — label for zero threshold")
	}
}

// TestHTMLSevClassInfo verifies the default (SeverityInfo) branch in the
// sevClass template function, covering html.go:56.11,57.21.
//
//fusa:test REQ-FO-RPT012
func TestHTMLSevClassInfo(t *testing.T) {
	r := New("/root", "proj", []Component{{
		Language: fusaops.LangGo, Tool: "gofusa", Available: true,
		Findings: []fusaops.Finding{{
			Language: fusaops.LangGo, Tool: "gofusa",
			RuleID: "I001", Severity: fusaops.SeverityInfo,
			Message: "info finding",
		}},
	}})
	var buf bytes.Buffer
	if err := Render(&buf, r, "html"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "sev-info") {
		t.Errorf("html: expected sev-info class in output")
	}
}

// TestJUnitCaseNameFileNoLine verifies that junitCaseName formats correctly
// when Location.File is set but Location.Line is zero, covering
// junit.go:59.9,61.4.
//
//fusa:test REQ-FO-RPT013
func TestJUnitCaseNameFileNoLine(t *testing.T) {
	f := fusaops.Finding{RuleID: "E001", Location: fusaops.Location{File: "main.go", Line: 0}}
	got := junitCaseName(f)
	want := "E001 (main.go)"
	if got != want {
		t.Errorf("junitCaseName: got %q, want %q", got, want)
	}
}

// TestJUnitHeaderWriteError verifies renderJUnit returns an error when writing
// the XML header fails, covering junit.go:134.62,136.3.
//
//fusa:test REQ-FO-RPT013
func TestJUnitHeaderWriteError(t *testing.T) {
	r := New("/root", "proj", nil)
	if err := renderJUnit(errWriter{}, r); err == nil {
		t.Error("renderJUnit: expected error when writer fails, got nil")
	}
}

// TestSARIFEncodeError verifies renderSARIF returns an error when the JSON
// encoder fails, covering sarif.go:112.40,114.3.
//
//fusa:test REQ-FO-RPT011
func TestSARIFEncodeError(t *testing.T) {
	r := New("/root", "proj", nil)
	if err := renderSARIF(errWriter{}, r); err == nil {
		t.Error("renderSARIF: expected error when writer fails, got nil")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
