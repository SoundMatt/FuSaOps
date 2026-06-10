package standards

import (
	"bytes"
	"os"
	"strings"
	"testing"
	"time"
)

func sampleReport(standard string, satisfied, partial, gaps int) *GapReport {
	objs := make([]Objective, 0, satisfied+partial+gaps)
	for i := range satisfied {
		objs = append(objs, Objective{ID: "S" + itoa(i+1), Status: "satisfied"})
	}
	for i := range partial {
		objs = append(objs, Objective{ID: "P" + itoa(i+1), Title: "partial obj", Status: "partial"})
	}
	for i := range gaps {
		objs = append(objs, Objective{ID: "G" + itoa(i+1), Title: "gap obj", Status: "gap"})
	}
	return &GapReport{
		Standard:   standard,
		Tool:       "gofusa",
		Language:   "go",
		Objectives: objs,
		Summary:    Summary{Total: satisfied + partial + gaps, Satisfied: satisfied, Partial: partial, Gaps: gaps},
	}
}

func itoa(n int) string {
	if n < 10 {
		return string(rune('0' + n))
	}
	return "X"
}

// TestHasGaps checks HasGaps and TotalGaps.
//
//fusa:test REQ-FO-STD005
func TestHasGaps(t *testing.T) {
	agg := &Aggregate{
		Standard: "iso26262",
		Components: []ComponentGap{
			{Tool: "gofusa", Report: sampleReport("iso26262", 10, 0, 2)},
			{Tool: "cfusa", Skipped: "binary not found"},
		},
	}
	if !agg.HasGaps() {
		t.Error("expected HasGaps() == true")
	}
	if agg.TotalGaps() != 2 {
		t.Errorf("expected TotalGaps() == 2, got %d", agg.TotalGaps())
	}
}

// TestHasGapsNone checks the all-satisfied case.
//
//fusa:test REQ-FO-STD005
func TestHasGapsNone(t *testing.T) {
	agg := &Aggregate{
		Standard: "iec61508",
		Components: []ComponentGap{
			{Tool: "gofusa", Report: sampleReport("iec61508", 5, 1, 0)},
		},
	}
	if agg.HasGaps() {
		t.Error("expected HasGaps() == false")
	}
	if agg.TotalGaps() != 0 {
		t.Errorf("expected TotalGaps() == 0, got %d", agg.TotalGaps())
	}
}

// TestHasGapsNilReport verifies nil Report is not counted as a gap.
//
//fusa:test REQ-FO-STD005
func TestHasGapsNilReport(t *testing.T) {
	agg := &Aggregate{
		Components: []ComponentGap{
			{Tool: "gofusa", Report: nil, Skipped: "not installed"},
		},
	}
	if agg.HasGaps() {
		t.Error("nil Report should not count as a gap")
	}
}

// TestTotalSkipped verifies TotalSkipped counts skipped components.
//
//fusa:test REQ-FO-STD003
func TestTotalSkipped(t *testing.T) {
	agg := &Aggregate{
		Components: []ComponentGap{
			{Tool: "gofusa", Report: sampleReport("iso26262", 5, 0, 0)},
			{Tool: "cfusa", Skipped: "no binary"},
			{Tool: "cpfusa", Skipped: "no binary"},
		},
	}
	if agg.TotalSkipped() != 2 {
		t.Errorf("want 2, got %d", agg.TotalSkipped())
	}
}

// TestNew checks that New sets standard, project, and components.
//
//fusa:test REQ-FO-STD004
func TestNew(t *testing.T) {
	comps := []ComponentGap{{Tool: "gofusa", Report: sampleReport("iso26262", 3, 0, 0)}}
	agg := New("myproj", "iso26262", comps)
	if agg.Standard != "iso26262" {
		t.Errorf("want iso26262, got %s", agg.Standard)
	}
	if agg.Project != "myproj" {
		t.Errorf("want myproj, got %s", agg.Project)
	}
	if len(agg.Components) != 1 {
		t.Errorf("want 1 component, got %d", len(agg.Components))
	}
	if agg.Generated.IsZero() {
		t.Error("Generated must be set")
	}
}

// TestRenderJSON verifies JSON render contains key fields.
//
//fusa:test REQ-FO-STD007
func TestRenderJSON(t *testing.T) {
	agg := New("p", "iso26262", []ComponentGap{
		{Tool: "gofusa", Language: "go", Report: sampleReport("iso26262", 5, 0, 1)},
	})
	var buf bytes.Buffer
	if err := Render(&buf, agg, "json"); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{"iso26262", "gofusa", "components"} {
		if !strings.Contains(out, want) {
			t.Errorf("JSON output missing %q", want)
		}
	}
}

// TestRenderJSONDefault verifies empty format defaults to JSON.
//
//fusa:test REQ-FO-STD007
func TestRenderJSONDefault(t *testing.T) {
	agg := New("", "iec61508", nil)
	var buf bytes.Buffer
	if err := Render(&buf, agg, ""); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "iec61508") {
		t.Error("expected standard in output")
	}
}

// TestRenderTextPass verifies text render for a passing aggregate.
//
//fusa:test REQ-FO-STD006
func TestRenderTextPass(t *testing.T) {
	agg := &Aggregate{
		Standard:  "iso26262",
		Project:   "myproject",
		Generated: time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC),
		Components: []ComponentGap{
			{Tool: "gofusa", Language: "go", Report: sampleReport("iso26262", 10, 1, 0)},
		},
	}
	var buf bytes.Buffer
	if err := Render(&buf, agg, "text"); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "ISO 26262") {
		t.Errorf("missing display name: %s", out)
	}
	if !strings.Contains(out, "myproject") {
		t.Errorf("missing project: %s", out)
	}
	if !strings.Contains(out, "RESULT: PASS") {
		t.Errorf("expected PASS result: %s", out)
	}
}

// TestRenderTextGap verifies text render for a gapped aggregate.
//
//fusa:test REQ-FO-STD006
func TestRenderTextGap(t *testing.T) {
	agg := &Aggregate{
		Standard:  "iec61508",
		Generated: time.Now(),
		Components: []ComponentGap{
			{Tool: "gofusa", Language: "go", Report: sampleReport("iec61508", 8, 0, 2)},
		},
	}
	var buf bytes.Buffer
	if err := Render(&buf, agg, "text"); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "RESULT: GAP") {
		t.Errorf("expected GAP result: %s", out)
	}
	if !strings.Contains(out, "gap obj") {
		t.Errorf("expected gap objective title in output: %s", out)
	}
}

// TestRenderTextSkipped verifies text render shows SKIP when all components skipped.
//
//fusa:test REQ-FO-STD006
func TestRenderTextSkipped(t *testing.T) {
	agg := &Aggregate{
		Standard:  "do178c",
		Generated: time.Now(),
		Components: []ComponentGap{
			{Tool: "gofusa", Language: "go", Skipped: "binary not found on PATH"},
		},
	}
	var buf bytes.Buffer
	if err := Render(&buf, agg, "text"); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "skipped") {
		t.Errorf("expected 'skipped' in output: %s", out)
	}
	if !strings.Contains(out, "RESULT: SKIP") {
		t.Errorf("expected SKIP result: %s", out)
	}
}

// TestRenderTextNilReport verifies nil Report renders as skipped.
//
//fusa:test REQ-FO-STD006
func TestRenderTextNilReport(t *testing.T) {
	agg := &Aggregate{
		Standard:  "iso26262",
		Generated: time.Now(),
		Components: []ComponentGap{
			{Tool: "gofusa", Language: "go", Report: nil},
		},
	}
	var buf bytes.Buffer
	if err := Render(&buf, agg, "text"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "skipped: no report") {
		t.Errorf("expected 'skipped: no report': %s", buf.String())
	}
}

// TestRenderUnknownFormat verifies error for unknown format.
//
//fusa:test REQ-FO-STD006
func TestRenderUnknownFormat(t *testing.T) {
	var buf bytes.Buffer
	err := Render(&buf, &Aggregate{}, "xml")
	if err == nil {
		t.Error("expected error for unknown format")
	}
}

// TestCommandStandard verifies the do178 alias and pass-throughs.
//
//fusa:test REQ-FO-STD008
func TestCommandStandard(t *testing.T) {
	cases := []struct{ cmd, want string }{
		{"do178", "do178c"},
		{"iso26262", "iso26262"},
		{"iec61508", "iec61508"},
		{"ISO26262", "iso26262"},
	}
	for _, tc := range cases {
		if got := CommandStandard(tc.cmd); got != tc.want {
			t.Errorf("CommandStandard(%q) = %q, want %q", tc.cmd, got, tc.want)
		}
	}
}

// TestDisplayName verifies human-readable labels.
//
//fusa:test REQ-FO-STD006
func TestDisplayName(t *testing.T) {
	cases := []struct{ id, want string }{
		{"iso26262", "ISO 26262"},
		{"iec61508", "IEC 61508"},
		{"do178c", "DO-178C"},
		{"iso21434", "ISO 21434"},
		{"iec62443-4-1", "IEC 62443-4-1"},
		{"iec62443-4-2", "IEC 62443-4-2"},
		{"unknown-std", "unknown-std"},
	}
	for _, tc := range cases {
		if got := displayName(tc.id); got != tc.want {
			t.Errorf("displayName(%q) = %q, want %q", tc.id, got, tc.want)
		}
	}
}

// TestRenderToFile verifies RenderToFile writes to a file.
//
//fusa:test REQ-FO-STD007
func TestRenderToFile(t *testing.T) {
	path := t.TempDir() + "/out.json"
	agg := New("p", "iso26262", nil)
	if err := RenderToFile(agg, "json", path); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "iso26262") {
		t.Errorf("output missing standard: %s", data)
	}
}
