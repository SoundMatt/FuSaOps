package report

import (
	"bytes"
	"strings"
	"testing"

	fusaops "github.com/SoundMatt/FuSaOps"
)

// TestJUnitXMLHeader verifies the JUnit output starts with an XML declaration.
//
//fusa:test REQ-FO-RPT013
func TestJUnitXMLHeader(t *testing.T) {
	r := New("/root", "demo", sampleComponents())
	var buf bytes.Buffer
	if err := Render(&buf, r, "junit"); err != nil {
		t.Fatalf("Render junit: %v", err)
	}
	if !strings.HasPrefix(buf.String(), "<?xml") {
		t.Errorf("expected XML declaration, got: %.40s", buf.String())
	}
}

// TestJUnitTestsuitesElement verifies <testsuites> wraps all suites.
//
//fusa:test REQ-FO-RPT013
func TestJUnitTestsuitesElement(t *testing.T) {
	r := New("/root", "demo", sampleComponents())
	var buf bytes.Buffer
	if err := Render(&buf, r, "junit"); err != nil {
		t.Fatalf("Render junit: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "<testsuites") {
		t.Errorf("missing <testsuites> element: %.200s", out)
	}
	if !strings.Contains(out, "</testsuites>") {
		t.Errorf("missing </testsuites>: %.200s", out)
	}
}

// TestJUnitPerComponentSuites verifies each component gets its own <testsuite>.
//
//fusa:test REQ-FO-RPT013
func TestJUnitPerComponentSuites(t *testing.T) {
	r := New("/root", "demo", sampleComponents())
	var buf bytes.Buffer
	if err := Render(&buf, r, "junit"); err != nil {
		t.Fatalf("Render junit: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, `name="go/gofusa"`) {
		t.Errorf("expected go/gofusa suite: %.400s", out)
	}
	if !strings.Contains(out, `name="c/cfusa"`) {
		t.Errorf("expected c/cfusa suite: %.400s", out)
	}
}

// TestJUnitFailureForErrorFinding verifies ERROR findings produce <failure> elements.
//
//fusa:test REQ-FO-RPT013
func TestJUnitFailureForErrorFinding(t *testing.T) {
	r := New("/root", "demo", sampleComponents())
	var buf bytes.Buffer
	if err := Render(&buf, r, "junit"); err != nil {
		t.Fatalf("Render junit: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "<failure") {
		t.Errorf("expected <failure> element for ERROR/WARNING findings: %.400s", out)
	}
	if !strings.Contains(out, `type="ERROR"`) {
		t.Errorf("expected type=ERROR: %.400s", out)
	}
}

// TestJUnitSkippedComponent verifies skipped components produce <skipped> elements.
//
//fusa:test REQ-FO-RPT013
func TestJUnitSkippedComponent(t *testing.T) {
	r := New("/root", "demo", sampleComponents())
	var buf bytes.Buffer
	if err := Render(&buf, r, "junit"); err != nil {
		t.Fatalf("Render junit: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "<skipped") {
		t.Errorf("expected <skipped> for unavailable component: %.400s", out)
	}
}

// TestJUnitNoFindingsSyntheticPass verifies components with zero findings emit a pass testcase.
//
//fusa:test REQ-FO-RPT013
func TestJUnitNoFindingsSyntheticPass(t *testing.T) {
	comps := []Component{
		{Language: fusaops.LangGo, Tool: "gofusa", Available: true, Findings: nil},
	}
	r := New("/root", "demo", comps)
	var buf bytes.Buffer
	if err := Render(&buf, r, "junit"); err != nil {
		t.Fatalf("Render junit: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "(no findings)") {
		t.Errorf("expected synthetic pass testcase: %.400s", out)
	}
	if strings.Contains(out, "<failure") {
		t.Errorf("clean component should have no <failure>: %.400s", out)
	}
}

// TestJUnitLocationInCaseName verifies file:line appears in testcase name.
//
//fusa:test REQ-FO-RPT013
func TestJUnitLocationInCaseName(t *testing.T) {
	comps := []Component{
		{Language: fusaops.LangGo, Tool: "gofusa", Available: true, Findings: []fusaops.Finding{
			{RuleID: "SA1001", Severity: fusaops.SeverityWarning, Message: "unused", Location: fusaops.Location{File: "main.go", Line: 42}},
		}},
	}
	r := New("/root", "demo", comps)
	var buf bytes.Buffer
	if err := Render(&buf, r, "junit"); err != nil {
		t.Fatalf("Render junit: %v", err)
	}
	if !strings.Contains(buf.String(), "main.go:42") {
		t.Errorf("expected file:line in testcase name: %.400s", buf.String())
	}
}

// TestJUnitUnsupportedFormat verifies unknown formats are rejected by Render.
//
//fusa:test REQ-FO-RPT007
func TestJUnitUnsupportedFormat(t *testing.T) {
	r := New("/root", "demo", sampleComponents())
	var buf bytes.Buffer
	if err := Render(&buf, r, "xml"); err == nil {
		t.Error("expected error for unsupported format 'xml'")
	}
}
