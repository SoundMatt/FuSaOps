package report

import (
	"bytes"
	"encoding/csv"
	"strings"
	"testing"

	fusaops "github.com/SoundMatt/FuSaOps"
)

// TestCSVHeader verifies the CSV output has the expected column header row.
//
//fusa:test REQ-FO-RPT014
func TestCSVHeader(t *testing.T) {
	r := New("/root", "demo", sampleComponents())
	var buf bytes.Buffer
	if err := Render(&buf, r, "csv"); err != nil {
		t.Fatalf("Render csv: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) == 0 {
		t.Fatal("empty CSV output")
	}
	header := lines[0]
	for _, col := range []string{"language", "tool", "ruleId", "severity", "message", "file", "line"} {
		if !strings.Contains(header, col) {
			t.Errorf("header missing column %q: %s", col, header)
		}
	}
}

// TestCSVFindingRows verifies each finding appears as a CSV row.
//
//fusa:test REQ-FO-RPT014
func TestCSVFindingRows(t *testing.T) {
	r := New("/root", "demo", sampleComponents())
	var buf bytes.Buffer
	if err := Render(&buf, r, "csv"); err != nil {
		t.Fatalf("Render csv: %v", err)
	}
	reader := csv.NewReader(&buf)
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("parse CSV: %v", err)
	}
	// header + 2 findings (the skipped cfusa component contributes no rows)
	if len(records) != 3 {
		t.Errorf("want 3 rows (header + 2 findings), got %d", len(records))
	}
}

// TestCSVSeverityColumn verifies finding severities appear in the severity column.
//
//fusa:test REQ-FO-RPT014
func TestCSVSeverityColumn(t *testing.T) {
	r := New("/root", "demo", sampleComponents())
	var buf bytes.Buffer
	if err := Render(&buf, r, "csv"); err != nil {
		t.Fatalf("Render csv: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "WARNING") {
		t.Errorf("expected WARNING in csv output: %.200s", out)
	}
	if !strings.Contains(out, "ERROR") {
		t.Errorf("expected ERROR in csv output: %.200s", out)
	}
}

// TestCSVLocationColumns verifies file and line appear in their columns.
//
//fusa:test REQ-FO-RPT014
func TestCSVLocationColumns(t *testing.T) {
	comps := []Component{
		{Language: fusaops.LangGo, Tool: "gofusa", Available: true, Findings: []fusaops.Finding{
			{RuleID: "R1", Severity: fusaops.SeverityWarning, Message: "msg", Location: fusaops.Location{File: "main.go", Line: 42, Column: 7}},
		}},
	}
	r := New("/root", "demo", comps)
	var buf bytes.Buffer
	if err := Render(&buf, r, "csv"); err != nil {
		t.Fatalf("Render csv: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "main.go") {
		t.Errorf("expected file in csv: %.200s", out)
	}
	if !strings.Contains(out, "42") {
		t.Errorf("expected line in csv: %.200s", out)
	}
	if !strings.Contains(out, "7") {
		t.Errorf("expected column in csv: %.200s", out)
	}
}

// TestCSVEmptyReport verifies a report with no findings produces only a header row.
//
//fusa:test REQ-FO-RPT014
func TestCSVEmptyReport(t *testing.T) {
	r := New("/root", "demo", []Component{
		{Language: fusaops.LangGo, Tool: "gofusa", Available: true, Findings: nil},
	})
	var buf bytes.Buffer
	if err := Render(&buf, r, "csv"); err != nil {
		t.Fatalf("Render csv: %v", err)
	}
	reader := csv.NewReader(&buf)
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("parse CSV: %v", err)
	}
	if len(records) != 1 {
		t.Errorf("want 1 row (header only), got %d", len(records))
	}
}
