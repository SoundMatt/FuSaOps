package report

import (
	"bytes"
	"strings"
	"testing"

	fusaops "github.com/SoundMatt/FuSaOps"
)

// TestMarkdownHeading verifies the Markdown output starts with a level-1 heading.
//
//fusa:test REQ-FO-RPT015
func TestMarkdownHeading(t *testing.T) {
	r := New("/root", "demo", sampleComponents())
	var buf bytes.Buffer
	if err := Render(&buf, r, "markdown"); err != nil {
		t.Fatalf("Render markdown: %v", err)
	}
	if !strings.HasPrefix(buf.String(), "# FuSaOps Report") {
		t.Errorf("expected H1 heading, got: %.60s", buf.String())
	}
}

// TestMarkdownSummaryTable verifies the summary table is present.
//
//fusa:test REQ-FO-RPT015
func TestMarkdownSummaryTable(t *testing.T) {
	r := New("/root", "demo", sampleComponents())
	var buf bytes.Buffer
	if err := Render(&buf, r, "markdown"); err != nil {
		t.Fatalf("Render markdown: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "## Summary") {
		t.Errorf("missing Summary section: %.400s", out)
	}
	if !strings.Contains(out, "Errors") || !strings.Contains(out, "Warnings") {
		t.Errorf("summary table missing counts: %.400s", out)
	}
}

// TestMarkdownComponentSection verifies each component gets its own section.
//
//fusa:test REQ-FO-RPT015
func TestMarkdownComponentSection(t *testing.T) {
	r := New("/root", "demo", sampleComponents())
	var buf bytes.Buffer
	if err := Render(&buf, r, "markdown"); err != nil {
		t.Fatalf("Render markdown: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "## Components") {
		t.Errorf("missing Components section: %.400s", out)
	}
	if !strings.Contains(out, "gofusa") {
		t.Errorf("missing gofusa component: %.400s", out)
	}
}

// TestMarkdownSkippedComponent verifies skipped components show skip reason.
//
//fusa:test REQ-FO-RPT015
func TestMarkdownSkippedComponent(t *testing.T) {
	r := New("/root", "demo", sampleComponents())
	var buf bytes.Buffer
	if err := Render(&buf, r, "markdown"); err != nil {
		t.Fatalf("Render markdown: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "Skipped") {
		t.Errorf("expected Skipped text for unavailable component: %.400s", out)
	}
}

// TestMarkdownFindingTable verifies findings appear in GFM table rows.
//
//fusa:test REQ-FO-RPT015
func TestMarkdownFindingTable(t *testing.T) {
	r := New("/root", "demo", sampleComponents())
	var buf bytes.Buffer
	if err := Render(&buf, r, "markdown"); err != nil {
		t.Fatalf("Render markdown: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "| Severity |") {
		t.Errorf("expected GFM table header: %.400s", out)
	}
	if !strings.Contains(out, "LINT001") || !strings.Contains(out, "FUSA001") {
		t.Errorf("expected rule IDs in table rows: %.400s", out)
	}
}

// TestMarkdownMdAlias verifies "md" is accepted as an alias for "markdown".
//
//fusa:test REQ-FO-RPT015
func TestMarkdownMdAlias(t *testing.T) {
	r := New("/root", "demo", sampleComponents())
	var buf bytes.Buffer
	if err := Render(&buf, r, "md"); err != nil {
		t.Fatalf("Render md alias: %v", err)
	}
	if !strings.HasPrefix(buf.String(), "# FuSaOps Report") {
		t.Errorf("md alias did not produce markdown output: %.60s", buf.String())
	}
}

// TestMarkdownNoFindingsSection verifies zero-finding components show no-findings message.
//
//fusa:test REQ-FO-RPT015
func TestMarkdownNoFindingsSection(t *testing.T) {
	comps := []Component{
		{Language: fusaops.LangGo, Tool: "gofusa", Available: true, Findings: nil},
	}
	r := New("/root", "demo", comps)
	var buf bytes.Buffer
	if err := Render(&buf, r, "markdown"); err != nil {
		t.Fatalf("Render markdown: %v", err)
	}
	if !strings.Contains(buf.String(), "No findings") {
		t.Errorf("expected 'No findings' for clean component: %.300s", buf.String())
	}
}
