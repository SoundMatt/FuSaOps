package standards

import (
	"bytes"
	"os"
	"path/filepath"
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
//fusa:test REQ-FO-STD001
//fusa:test REQ-FO-STD002
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
//fusa:test REQ-FO-STD009
func TestDisplayName(t *testing.T) {
	cases := []struct{ id, want string }{
		{"iso26262", "ISO 26262"},
		{"iec61508", "IEC 61508"},
		{"do178c", "DO-178C"},
		{"iso21434", "ISO 21434"},
		{"iec62443-4-1", "IEC 62443-4-1"},
		{"iec62443-4-2", "IEC 62443-4-2"},
		{"iec62443", "IEC 62443"},
		{"unece", "UNECE R155/R156"},
		{"unknown-std", "unknown-std"},
	}
	for _, tc := range cases {
		if got := displayName(tc.id); got != tc.want {
			t.Errorf("displayName(%q) = %q, want %q", tc.id, got, tc.want)
		}
	}
}

// TestDisplayNameExported verifies the exported DisplayName wrapper delegates correctly.
//
//fusa:test REQ-FO-STD009
func TestDisplayNameExported(t *testing.T) {
	if got := DisplayName("iso26262"); got != "ISO 26262" {
		t.Errorf("DisplayName(iso26262) = %q, want ISO 26262", got)
	}
	if got := DisplayName("custom-standard"); got != "custom-standard" {
		t.Errorf("DisplayName(custom-standard) = %q, want identity", got)
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

// TestRecomputeSummaryFromObjectives verifies RecomputeSummary derives correct
// counts from objectives when the Summary is missing or inconsistent — e.g.
// cpp-FuSa v0.6.0 uses "addressed"/"gap" instead of spec "satisfied"/"gaps".
//
//fusa:test REQ-FO-STD010
func TestRecomputeSummaryFromObjectives(t *testing.T) {
	r := &GapReport{
		Objectives: []Objective{
			{ID: "1", Status: "satisfied"},
			{ID: "2", Status: "satisfied"},
			{ID: "3", Status: "partial"},
			{ID: "4", Status: "gap"},
			{ID: "5", Status: "gap"},
		},
		// Summary has wrong keys decoded (zeros for satisfied/gaps — as if
		// the tool emitted "addressed" and "gap" instead of spec key names).
		Summary: Summary{Total: 5, Satisfied: 0, Partial: 1, Gaps: 0},
	}
	r.RecomputeSummary()
	if r.Summary.Satisfied != 2 {
		t.Errorf("want Satisfied=2, got %d", r.Summary.Satisfied)
	}
	if r.Summary.Partial != 1 {
		t.Errorf("want Partial=1, got %d", r.Summary.Partial)
	}
	if r.Summary.Gaps != 2 {
		t.Errorf("want Gaps=2, got %d", r.Summary.Gaps)
	}
	if r.Summary.Total != 5 {
		t.Errorf("want Total=5, got %d", r.Summary.Total)
	}
}

// TestRecomputeSummaryNoOp verifies RecomputeSummary is a no-op when the
// invariant already holds.
//
//fusa:test REQ-FO-STD010
func TestRecomputeSummaryNoOp(t *testing.T) {
	r := &GapReport{
		Objectives: []Objective{
			{ID: "1", Status: "satisfied"},
			{ID: "2", Status: "gap"},
		},
		Summary: Summary{Total: 2, Satisfied: 1, Partial: 0, Gaps: 1},
	}
	r.RecomputeSummary()
	if r.Summary.Satisfied != 1 || r.Summary.Gaps != 1 {
		t.Errorf("no-op violated: %+v", r.Summary)
	}
}

// TestRecomputeSummaryUnknownStatus verifies unknown status maps to gap.
//
//fusa:test REQ-FO-STD010
func TestRecomputeSummaryUnknownStatus(t *testing.T) {
	r := &GapReport{
		Objectives: []Objective{
			{ID: "1", Status: "addressed"}, // non-conformant synonym
			{ID: "2", Status: "unknown-x"},
		},
		Summary: Summary{Total: 0}, // will be recomputed
	}
	r.RecomputeSummary()
	if r.Summary.Gaps != 2 {
		t.Errorf("unknown status should map to gap, got Gaps=%d", r.Summary.Gaps)
	}
}

// TestRecomputeSummaryEmpty verifies RecomputeSummary is a no-op on empty objectives.
//
//fusa:test REQ-FO-STD010
func TestRecomputeSummaryEmpty(t *testing.T) {
	r := &GapReport{Summary: Summary{Total: 5, Satisfied: 3, Partial: 1, Gaps: 1}}
	r.RecomputeSummary() // no objectives — should not change summary
	if r.Summary.Total != 5 {
		t.Errorf("empty objectives: summary should not change")
	}
}

// TestRenderHTML verifies the HTML standards report contains expected content.
//
//fusa:test REQ-FO-STD011
func TestRenderHTML(t *testing.T) {
	agg := &Aggregate{
		Standard:  "iso26262",
		Project:   "myproj",
		Generated: time.Now(),
		Components: []ComponentGap{
			{
				Language: "go", Tool: "gofusa",
				Report: sampleReport("ISO 26262", 2, 1, 1),
			},
			{Language: "c", Tool: "cfusa", Skipped: "not installed"},
		},
	}
	var buf bytes.Buffer
	if err := Render(&buf, agg, "html"); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{"<!doctype html>", "ISO 26262", "myproj", "gofusa", "satisfied", "cfusa", "not installed"} {
		if !strings.Contains(out, want) {
			t.Errorf("html missing %q", want)
		}
	}
}

// TestRenderHTMLNilReport verifies nil-report component shows fallback.
//
//fusa:test REQ-FO-STD011
func TestRenderHTMLNilReport(t *testing.T) {
	agg := &Aggregate{
		Standard:  "iec61508",
		Generated: time.Now(),
		Components: []ComponentGap{
			{Language: "go", Tool: "gofusa", Report: nil},
		},
	}
	var buf bytes.Buffer
	if err := Render(&buf, agg, "html"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "No report available") {
		t.Error("html should show fallback for nil report")
	}
}

// TestRenderMarkdown verifies GFM markdown rendering contains expected content.
//
//fusa:test REQ-FO-STD012
func TestRenderMarkdown(t *testing.T) {
	agg := &Aggregate{
		Standard:  "iso26262",
		Project:   "my-proj",
		Generated: time.Now(),
		Components: []ComponentGap{
			{
				Language: "go",
				Tool:     "gofusa",
				Report: &GapReport{
					Standard:    "ISO 26262",
					ToolVersion: "0.30.0",
					Objectives: []Objective{
						{ID: "A1", Status: "satisfied", Title: "Design"},
						{ID: "A2", Status: "gap", Title: "Verification"},
					},
					Summary: Summary{Total: 2, Satisfied: 1, Gaps: 1},
				},
			},
		},
	}
	var buf bytes.Buffer
	if err := Render(&buf, agg, "markdown"); err != nil {
		t.Fatalf("Render markdown: %v", err)
	}
	out := buf.String()
	for _, want := range []string{"# FuSaOps", "ISO 26262", "my-proj", "**GAP**", "| ID |", "gofusa", "A1", "A2", "❌"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q in markdown:\n%s", want, out)
		}
	}
}

// TestRenderMarkdownAlias verifies "md" is accepted as an alias.
//
//fusa:test REQ-FO-STD012
func TestRenderMarkdownAlias(t *testing.T) {
	agg := &Aggregate{Standard: "iso26262", Generated: time.Now()}
	var buf bytes.Buffer
	if err := Render(&buf, agg, "md"); err != nil {
		t.Fatalf("Render md alias: %v", err)
	}
	if !strings.Contains(buf.String(), "# FuSaOps") {
		t.Error("expected markdown header from md alias")
	}
}

// TestRenderMarkdownSkipped verifies skipped components show the skip reason.
//
//fusa:test REQ-FO-STD012
func TestRenderMarkdownSkipped(t *testing.T) {
	agg := &Aggregate{
		Standard:  "iec61508",
		Generated: time.Now(),
		Components: []ComponentGap{
			{Language: "go", Tool: "gofusa", Skipped: "not installed"},
		},
	}
	var buf bytes.Buffer
	if err := Render(&buf, agg, "markdown"); err != nil {
		t.Fatalf("Render markdown skipped: %v", err)
	}
	if !strings.Contains(buf.String(), "not installed") {
		t.Error("expected skip reason in markdown output")
	}
}

// TestRenderToFileCreateError verifies RenderToFile returns an error when it
// cannot create the output file (parent directory does not exist).
//
//fusa:test REQ-FO-STD007
func TestRenderToFileCreateError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing", "out.txt")
	agg := &Aggregate{Standard: "iso26262", Generated: time.Now()}
	if err := RenderToFile(agg, "text", path); err == nil {
		t.Error("RenderToFile: expected error for non-existent parent directory")
	}
}

// TestRenderMarkdownPass verifies green badge when there are no gaps.
//
//fusa:test REQ-FO-STD012
func TestRenderMarkdownPass(t *testing.T) {
	agg := &Aggregate{
		Standard:  "iso26262",
		Generated: time.Now(),
		Components: []ComponentGap{
			{
				Language: "go",
				Tool:     "gofusa",
				Report: &GapReport{
					Standard:    "ISO 26262",
					ToolVersion: "0.30.0",
					Objectives:  []Objective{{ID: "A1", Status: "satisfied"}},
					Summary:     Summary{Total: 1, Satisfied: 1},
				},
			},
		},
	}
	var buf bytes.Buffer
	if err := Render(&buf, agg, "markdown"); err != nil {
		t.Fatalf("Render markdown pass: %v", err)
	}
	if !strings.Contains(buf.String(), "🟢") {
		t.Error("expected green badge for gap-free standards report")
	}
}
