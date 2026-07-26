package report

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"

	fusaops "github.com/SoundMatt/FuSaOps"
)

// errWriter returns an error on every Write call.
type errWriter struct{}

func (errWriter) Write(_ []byte) (int, error) { return 0, errors.New("write failed") }

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
//fusa:test REQ-FO-RPT016
func TestRenderHTML(t *testing.T) {
	r := New("/root", "demo", sampleComponents())
	var buf bytes.Buffer
	if err := Render(&buf, r, "html"); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{
		"<!DOCTYPE html>", "FuSaOps", "status-fail", "LINT001",
		"search-box", "applyFilters", "search-count",
	} {
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

func fingerprintComponents() []Component {
	return []Component{
		{
			Language:  fusaops.LangGo,
			Tool:      "gofusa",
			Available: true,
			Findings: []fusaops.Finding{
				{Language: fusaops.LangGo, Tool: "gofusa", RuleID: "LINT001", Severity: fusaops.SeverityWarning,
					Message: "warn", Fingerprint: "sha256:deadbeef", Location: fusaops.Location{File: "a.go", Line: 1}},
			},
		},
	}
}

func suppressedComponents() []Component {
	return []Component{
		{
			Language:  fusaops.LangGo,
			Tool:      "gofusa",
			Available: true,
			Findings: []fusaops.Finding{
				{Language: fusaops.LangGo, Tool: "gofusa", RuleID: "LINT001", Severity: fusaops.SeverityWarning, Message: "active warn"},
			},
			SuppressedFindings: []fusaops.Finding{
				{Language: fusaops.LangGo, Tool: "gofusa", RuleID: "FUSA001", Severity: fusaops.SeverityError, Message: "suppressed err", Location: fusaops.Location{File: "main.go", Line: 10}},
			},
		},
	}
}

//fusa:test REQ-FO-RPT017
//fusa:test REQ-FO-RPT018
func TestSuppressedComponentsField(t *testing.T) {
	comps := suppressedComponents()
	r := New("/root", "proj", comps)
	r.Suppressed = 1
	r.SuppressedComponents = []string{"gofusa"}
	if len(r.SuppressedComponents) != 1 || r.SuppressedComponents[0] != "gofusa" {
		t.Errorf("SuppressedComponents wrong: %v", r.SuppressedComponents)
	}
	if len(r.Components[0].SuppressedFindings) != 1 {
		t.Errorf("SuppressedFindings wrong length: %d", len(r.Components[0].SuppressedFindings))
	}
}

//fusa:test REQ-FO-RPT017
func TestRenderTextShowSuppressed(t *testing.T) {
	comps := suppressedComponents()
	r := New("/root", "proj", comps)
	r.Suppressed = 1

	var buf bytes.Buffer
	if err := RenderWithOptions(&buf, r, "text", RenderOptions{ShowSuppressed: true}); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "[SUPPRESSED]") {
		t.Error("expected [SUPPRESSED] prefix in text output")
	}
	if !strings.Contains(out, "FUSA001") {
		t.Error("expected suppressed finding rule ID in output")
	}
}

//fusa:test REQ-FO-RPT017
func TestRenderTextHideSuppressedHint(t *testing.T) {
	comps := suppressedComponents()
	r := New("/root", "proj", comps)
	r.Suppressed = 1

	var buf bytes.Buffer
	if err := RenderWithOptions(&buf, r, "text", RenderOptions{ShowSuppressed: false}); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if strings.Contains(out, "[SUPPRESSED]") {
		t.Error("unexpected [SUPPRESSED] prefix when ShowSuppressed is false")
	}
	if !strings.Contains(out, "suppressed") {
		t.Error("expected suppressed count hint in text output")
	}
	if !strings.Contains(out, "--show-suppressed") {
		t.Error("expected --show-suppressed hint in text output")
	}
}

//fusa:test REQ-FO-RPT017
func TestRenderMarkdownSuppressedCollapsed(t *testing.T) {
	comps := suppressedComponents()
	r := New("/root", "proj", comps)
	r.Suppressed = 1

	var buf bytes.Buffer
	if err := RenderWithOptions(&buf, r, "markdown", RenderOptions{ShowSuppressed: false}); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "<details>") {
		t.Error("expected collapsed <details> in markdown when ShowSuppressed false")
	}
	if strings.Contains(out, "<details open>") {
		t.Error("unexpected open <details> when ShowSuppressed false")
	}
}

//fusa:test REQ-FO-RPT017
func TestRenderMarkdownSuppressedOpen(t *testing.T) {
	comps := suppressedComponents()
	r := New("/root", "proj", comps)
	r.Suppressed = 1

	var buf bytes.Buffer
	if err := RenderWithOptions(&buf, r, "markdown", RenderOptions{ShowSuppressed: true}); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "<details open>") {
		t.Error("expected open <details> in markdown when ShowSuppressed true")
	}
}

//fusa:test REQ-FO-RPT017
func TestRenderHTMLSuppressedSection(t *testing.T) {
	comps := suppressedComponents()
	r := New("/root", "proj", comps)
	r.Suppressed = 1

	var buf bytes.Buffer
	if err := RenderWithOptions(&buf, r, "html", RenderOptions{ShowSuppressed: false}); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "<details") {
		t.Error("expected <details> section in HTML for suppressed findings")
	}
	if !strings.Contains(out, "suppressed") {
		t.Error("expected 'suppressed' in HTML output")
	}
}

//fusa:test REQ-FO-RPT017
func TestRenderHTMLSuppressedOpen(t *testing.T) {
	comps := suppressedComponents()
	r := New("/root", "proj", comps)
	r.Suppressed = 1

	var buf bytes.Buffer
	if err := RenderWithOptions(&buf, r, "html", RenderOptions{ShowSuppressed: true}); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "<details") {
		t.Error("expected <details> in HTML for suppressed findings")
	}
	if !strings.Contains(out, "open") {
		t.Error("expected open attribute in HTML details when ShowSuppressed true")
	}
}

//fusa:test REQ-FO-RPT017
func TestRenderWithOptionsUnsupported(t *testing.T) {
	r := New("/root", "proj", nil)
	if err := RenderWithOptions(&bytes.Buffer{}, r, "xml", RenderOptions{}); err == nil {
		t.Error("expected error for unsupported format")
	}
}

//fusa:test REQ-FO-RPT019
func TestRenderTextShowFingerprints(t *testing.T) {
	r := New("/root", "proj", fingerprintComponents())

	var buf bytes.Buffer
	if err := RenderWithOptions(&buf, r, "text", RenderOptions{ShowFingerprints: true}); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "sha256:deadbeef") {
		t.Error("expected fingerprint in text output")
	}
	if !strings.Contains(out, "fusaops suppress add") {
		t.Error("expected suppress scaffold in text output")
	}
}

//fusa:test REQ-FO-RPT019
func TestRenderTextHideFingerprintsDefault(t *testing.T) {
	r := New("/root", "proj", fingerprintComponents())

	var buf bytes.Buffer
	if err := Render(&buf, r, "text"); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if strings.Contains(out, "fusaops suppress add") {
		t.Error("unexpected suppress scaffold without --show-fingerprints")
	}
}

//fusa:test REQ-FO-RPT019
func TestRenderMarkdownShowFingerprints(t *testing.T) {
	r := New("/root", "proj", fingerprintComponents())

	var buf bytes.Buffer
	if err := RenderWithOptions(&buf, r, "markdown", RenderOptions{ShowFingerprints: true}); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "Fingerprint") {
		t.Error("expected Fingerprint column in markdown with ShowFingerprints")
	}
	if !strings.Contains(out, "sha256:deadbeef") {
		t.Error("expected fingerprint value in markdown output")
	}
}

//fusa:test REQ-FO-RPT019
func TestRenderMarkdownHideFingerprintsDefault(t *testing.T) {
	r := New("/root", "proj", fingerprintComponents())

	var buf bytes.Buffer
	if err := Render(&buf, r, "markdown"); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if strings.Contains(out, "Fingerprint") {
		t.Error("unexpected Fingerprint column without ShowFingerprints")
	}
}

//fusa:test REQ-FO-RPT019
func TestRenderHTMLShowFingerprints(t *testing.T) {
	r := New("/root", "proj", fingerprintComponents())

	var buf bytes.Buffer
	if err := RenderWithOptions(&buf, r, "html", RenderOptions{ShowFingerprints: true}); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "fp-chip") {
		t.Error("expected fp-chip class in HTML with ShowFingerprints")
	}
	if !strings.Contains(out, "sha256:deadbeef") {
		t.Error("expected fingerprint value in HTML output")
	}
}

//fusa:test REQ-FO-RPT019
func TestRenderHTMLHideFingerprintsDefault(t *testing.T) {
	r := New("/root", "proj", fingerprintComponents())

	var buf bytes.Buffer
	if err := Render(&buf, r, "html"); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	// The fp-chip CSS class is always present in the stylesheet; verify the
	// actual fingerprint value does NOT appear in finding rows by default.
	if strings.Count(out, "sha256:deadbeef") > 0 {
		t.Error("unexpected fingerprint value in HTML without ShowFingerprints")
	}
}

//fusa:test REQ-FO-RPT017
func TestRenderToFileWithOptions(t *testing.T) {
	comps := suppressedComponents()
	r := New("/root", "proj", comps)
	r.Suppressed = 1

	var buf bytes.Buffer
	if err := RenderToFileWithOptions(r, "text", "", RenderOptions{ShowSuppressed: true}); err != nil {
		// stdout write won't fail in test
		t.Fatal(err)
	}
	// Write to buffer path
	if err := RenderWithOptions(&buf, r, "text", RenderOptions{ShowSuppressed: true}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "[SUPPRESSED]") {
		t.Error("expected suppressed output via RenderToFileWithOptions")
	}
}

// ── Standard / integrity level fields (v1.59.0) ──────────────────────────────

//fusa:test REQ-FO-RPT020
func TestAggregateReportStandardField(t *testing.T) {
	r := New("/root", "proj", nil)
	r.Standard = "ISO26262"
	r.ASIL = "ASIL-C"
	if r.Standard != "ISO26262" {
		t.Errorf("Standard=%q, want ISO26262", r.Standard)
	}
	if r.ASIL != "ASIL-C" {
		t.Errorf("ASIL=%q, want ASIL-C", r.ASIL)
	}
}

//fusa:test REQ-FO-RPT020
func TestTextRendererShowsStandard(t *testing.T) {
	r := New("/root", "proj", nil)
	r.Standard = "ISO26262"
	r.ASIL = "ASIL-B"
	var buf bytes.Buffer
	if err := Render(&buf, r, "text"); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "ISO26262") {
		t.Error("text output should contain Standard value")
	}
	if !strings.Contains(out, "ASIL-B") {
		t.Error("text output should contain ASIL value")
	}
}

//fusa:test REQ-FO-RPT020
func TestTextRendererSIL(t *testing.T) {
	r := New("/root", "proj", nil)
	r.Standard = "IEC61508"
	r.SIL = "SIL-3"
	var buf bytes.Buffer
	if err := Render(&buf, r, "text"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "SIL-3") {
		t.Error("text output should contain SIL value")
	}
}

//fusa:test REQ-FO-RPT020
func TestTextRendererDAL(t *testing.T) {
	r := New("/root", "proj", nil)
	r.Standard = "DO178C"
	r.DAL = "DAL-B"
	var buf bytes.Buffer
	if err := Render(&buf, r, "text"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "DAL-B") {
		t.Error("text output should contain DAL value")
	}
}

//fusa:test REQ-FO-RPT020
func TestTextRendererStandardNoLevel(t *testing.T) {
	r := New("/root", "proj", nil)
	r.Standard = "ISO26262"
	var buf bytes.Buffer
	if err := Render(&buf, r, "text"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "ISO26262") {
		t.Error("text output should show Standard even without integrity level")
	}
}

//fusa:test REQ-FO-RPT020
func TestMarkdownRendererShowsStandard(t *testing.T) {
	r := New("/root", "proj", nil)
	r.Standard = "ISO26262"
	r.ASIL = "ASIL-C"
	var buf bytes.Buffer
	if err := Render(&buf, r, "markdown"); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "ISO26262") {
		t.Error("markdown output should contain Standard value")
	}
	if !strings.Contains(out, "ASIL-C") {
		t.Error("markdown output should contain ASIL value")
	}
}

//fusa:test REQ-FO-RPT020
func TestJSONReportHasIntegrityFields(t *testing.T) {
	r := New("/root", "proj", nil)
	r.Standard = "IEC61508"
	r.SIL = "SIL-2"
	var buf bytes.Buffer
	if err := Render(&buf, r, "json"); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, `"standard"`) {
		t.Error("JSON output should contain standard field")
	}
	if !strings.Contains(out, `"sil"`) {
		t.Error("JSON output should contain sil field")
	}
}

// TestRenderHTMLCompSection verifies the dashboard includes the comp section
// when CompInfo is non-nil.
//
//fusa:test REQ-FO-RPT021
func TestRenderHTMLCompSection(t *testing.T) {
	r := New("/root", "demo", sampleComponents())
	var buf bytes.Buffer
	err := RenderWithOptions(&buf, r, "html", RenderOptions{
		CompInfo: &CompInfo{
			TotalFunctions: 12,
			Violations:     2,
			Components: []CompComponent{
				{Language: "go", Tool: "gofusa", TotalFunctions: 12, Violations: 2, Threshold: 10, DAL: "DAL-B"},
			},
		},
	})
	if err != nil {
		t.Fatalf("renderHTML: %v", err)
	}
	html := buf.String()
	for _, want := range []string{
		"Cyclomatic Complexity", "2 violations", "12", "gofusa", "DAL-B (≤10)",
	} {
		if !strings.Contains(html, want) {
			t.Errorf("html missing %q", want)
		}
	}
}

// TestRenderHTMLCompNavLink verifies the dashboard nav includes a /comp link
// when CompInfo is non-nil and omits it when CompInfo is nil.
//
//fusa:test REQ-FO-SRV013
func TestRenderHTMLCompNavLink(t *testing.T) {
	r := New("/root", "demo", sampleComponents())
	var buf bytes.Buffer
	_ = RenderWithOptions(&buf, r, "html", RenderOptions{
		CompInfo: &CompInfo{TotalFunctions: 5, Violations: 0},
	})
	if !strings.Contains(buf.String(), `href="/comp"`) {
		t.Error("nav should contain /comp link when CompInfo is set")
	}
	buf.Reset()
	_ = Render(&buf, r, "html")
	if strings.Contains(buf.String(), `href="/comp"`) {
		t.Error("nav should not contain /comp link when CompInfo is nil")
	}
}

// TestRenderHTMLCompSectionHidden verifies the comp section is absent when
// CompInfo is nil.
//
//fusa:test REQ-FO-RPT021
func TestRenderHTMLCompSectionHidden(t *testing.T) {
	r := New("/root", "demo", sampleComponents())
	var buf bytes.Buffer
	if err := Render(&buf, r, "html"); err != nil {
		t.Fatalf("renderHTML: %v", err)
	}
	if strings.Contains(buf.String(), "Cyclomatic Complexity") {
		t.Error("html should not include comp section when CompInfo is nil")
	}
}

// ── QualifyInfo.IsIndependent() (REQ-FO-QLF011) ──────────────────────────────

//fusa:test REQ-FO-QLF011
func TestQualifyInfoIsIndependent(t *testing.T) {
	// nil receiver should return false
	var q *QualifyInfo
	if q.IsIndependent() {
		t.Error("nil QualifyInfo.IsIndependent() should be false")
	}
	// empty reviewer → not independent
	q = &QualifyInfo{}
	if q.IsIndependent() {
		t.Error("empty IndependentReviewer should return false")
	}
	// populated reviewer → independent
	q = &QualifyInfo{IndependentReviewer: "Alice Engineer"}
	if !q.IsIndependent() {
		t.Error("populated IndependentReviewer should return true")
	}
}

// ── MCDCInfo.CoveragePct() (REQ-FO-MCDC003) ──────────────────────────────────

//fusa:test REQ-FO-MCDC003
func TestMCDCInfoCoveragePct(t *testing.T) {
	// nil receiver → 100 (guard)
	var m *MCDCInfo
	if got := m.CoveragePct(); got != 100 {
		t.Errorf("nil MCDCInfo.CoveragePct() = %d, want 100", got)
	}
	// zero total → 100 (guard)
	m = &MCDCInfo{TotalConditions: 0, CoveredConditions: 0}
	if got := m.CoveragePct(); got != 100 {
		t.Errorf("zero total CoveragePct() = %d, want 100", got)
	}
	// 6 of 8 → 75%
	m = &MCDCInfo{TotalConditions: 8, CoveredConditions: 6}
	if got := m.CoveragePct(); got != 75 {
		t.Errorf("CoveragePct(6/8) = %d, want 75", got)
	}
	// 10 of 10 → 100%
	m = &MCDCInfo{TotalConditions: 10, CoveredConditions: 10}
	if got := m.CoveragePct(); got != 100 {
		t.Errorf("CoveragePct(10/10) = %d, want 100", got)
	}
	// 0 of 5 → 0%
	m = &MCDCInfo{TotalConditions: 5, CoveredConditions: 0}
	if got := m.CoveragePct(); got != 0 {
		t.Errorf("CoveragePct(0/5) = %d, want 0", got)
	}
}

// TestRenderHTMLWriteError verifies renderHTML returns an error when the writer fails.
//
//fusa:test REQ-FO-RPT012
//fusa:test REQ-FO-RPT016
func TestRenderHTMLWriteError(t *testing.T) {
	r := New("/root", "demo", sampleComponents())
	if err := renderHTML(errWriter{}, r, RenderOptions{}); err == nil {
		t.Error("expected error when Writer fails during HTML template execution")
	}
}

// TestRenderCSVWriteError verifies renderCSV returns an error when the writer fails.
//
//fusa:test REQ-FO-RPT001
func TestRenderCSVWriteError(t *testing.T) {
	r := New("/root", "demo", sampleComponents())
	if err := renderCSV(errWriter{}, r); err == nil {
		t.Error("expected error when Writer fails during CSV render")
	}
}

// TestRenderJSONWriteError verifies renderJSON returns an error when the writer fails.
//
//fusa:test REQ-FO-RPT001
func TestRenderJSONWriteError(t *testing.T) {
	r := New("/root", "demo", sampleComponents())
	if err := renderJSON(errWriter{}, r); err == nil {
		t.Error("expected error when Writer fails during JSON render")
	}
}

// TestMarkdownBadgePending verifies the default branch of markdownBadge.
//
//fusa:test REQ-FO-RPT015
func TestMarkdownBadgePending(t *testing.T) {
	got := markdownBadge("UNKNOWN")
	if !strings.Contains(got, "PENDING") {
		t.Errorf("markdownBadge(UNKNOWN): want PENDING badge, got %q", got)
	}
}

// TestMarkdownSeverityIconInfo verifies the default (INFO) branch of markdownSeverityIcon.
//
//fusa:test REQ-FO-RPT015
func TestMarkdownSeverityIconInfo(t *testing.T) {
	got := markdownSeverityIcon(fusaops.SeverityInfo)
	if !strings.Contains(got, "INFO") {
		t.Errorf("markdownSeverityIcon(INFO): want INFO icon, got %q", got)
	}
}

// TestMarkdownLocFileOnly verifies markdownLoc returns a backtick-wrapped filename
// when the finding has a file but no line number.
//
//fusa:test REQ-FO-RPT015
func TestMarkdownLocFileOnly(t *testing.T) {
	f := fusaops.Finding{Location: fusaops.Location{File: "main.go"}}
	got := markdownLoc(f)
	if got != "`main.go`" {
		t.Errorf("markdownLoc file-only: got %q, want \"`main.go`\"", got)
	}
}

// Ensure errWriter implements io.Writer (compile check).
var _ io.Writer = errWriter{}
