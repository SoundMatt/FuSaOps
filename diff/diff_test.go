package diff

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	fusaops "github.com/SoundMatt/FuSaOps"
)

func finding(ruleID, file, msg string, sev fusaops.Severity) fusaops.Finding {
	return fusaops.Finding{
		RuleID:   ruleID,
		Severity: sev,
		Message:  msg,
		Location: fusaops.Location{File: file, Line: 1},
	}
}

func withFP(f fusaops.Finding) fusaops.Finding {
	f.Fingerprint = fusaops.ComputeFingerprint(f)
	return f
}

// TestCompareUnchanged verifies identical fingerprint sets produce no diff.
//
//fusa:test REQ-FO-DIF002
func TestCompareUnchanged(t *testing.T) {
	f := withFP(finding("LINT001", "main.go", "unused variable", fusaops.SeverityWarning))
	baseline := &Baseline{Findings: []fusaops.Finding{f}}
	res := Compare(baseline, []fusaops.Finding{f})
	if len(res.Added) != 0 || len(res.Removed) != 0 || res.Unchanged != 1 {
		t.Errorf("expected 0 added, 0 removed, 1 unchanged; got %+v", res)
	}
}

// TestCompareAdded verifies a new finding in current is detected.
//
//fusa:test REQ-FO-DIF002
func TestCompareAdded(t *testing.T) {
	old := withFP(finding("LINT001", "a.go", "msg", fusaops.SeverityWarning))
	newF := withFP(finding("SAFETY001", "b.go", "critical", fusaops.SeverityError))
	baseline := &Baseline{Findings: []fusaops.Finding{old}}
	res := Compare(baseline, []fusaops.Finding{old, newF})
	if len(res.Added) != 1 || res.Added[0].RuleID != "SAFETY001" {
		t.Errorf("expected 1 added SAFETY001, got %+v", res.Added)
	}
	if res.Unchanged != 1 {
		t.Errorf("expected 1 unchanged, got %d", res.Unchanged)
	}
}

// TestCompareRemoved verifies a finding absent from current appears in Removed.
//
//fusa:test REQ-FO-DIF002
func TestCompareRemoved(t *testing.T) {
	f := withFP(finding("LINT001", "main.go", "unused variable", fusaops.SeverityWarning))
	baseline := &Baseline{Findings: []fusaops.Finding{f}}
	res := Compare(baseline, []fusaops.Finding{})
	if len(res.Removed) != 1 || res.Removed[0].RuleID != "LINT001" {
		t.Errorf("expected 1 removed LINT001, got %+v", res.Removed)
	}
}

// TestCompareNoFingerprint verifies ComputeFingerprint is used when absent.
//
//fusa:test REQ-FO-DIF002
func TestCompareNoFingerprint(t *testing.T) {
	f := finding("LINT001", "main.go", "unused variable", fusaops.SeverityWarning)
	// no fingerprint on either side — should still match
	baseline := &Baseline{Findings: []fusaops.Finding{f}}
	res := Compare(baseline, []fusaops.Finding{f})
	if len(res.Added) != 0 || len(res.Removed) != 0 || res.Unchanged != 1 {
		t.Errorf("fingerprint computed on the fly should match: %+v", res)
	}
}

// TestHasNewErrors verifies gate logic.
//
//fusa:test REQ-FO-DIF002
func TestHasNewErrors(t *testing.T) {
	res := &Result{Added: []fusaops.Finding{
		{Severity: fusaops.SeverityWarning},
	}}
	if res.HasNewErrors() {
		t.Error("WARNING should not set HasNewErrors")
	}
	res.Added = append(res.Added, fusaops.Finding{Severity: fusaops.SeverityError})
	if !res.HasNewErrors() {
		t.Error("ERROR added should set HasNewErrors")
	}
}

//fusa:test REQ-FO-DIF007
func TestHasNewFindings(t *testing.T) {
	res := &Result{}
	if res.HasNewFindings() {
		t.Error("empty Added should not set HasNewFindings")
	}
	res.Added = append(res.Added, fusaops.Finding{Severity: fusaops.SeverityInfo})
	if !res.HasNewFindings() {
		t.Error("any added finding, even INFO, should set HasNewFindings")
	}
}

// TestLoadBaselineAggregateFormat verifies decoding a FuSaOps aggregate JSON.
//
//fusa:test REQ-FO-DIF001
func TestLoadBaselineAggregateFormat(t *testing.T) {
	f := withFP(finding("LINT001", "main.go", "unused variable", fusaops.SeverityWarning))
	doc := map[string]any{
		"root":    "/repo",
		"project": "myproject",
		"components": []any{
			map[string]any{"findings": []any{map[string]any{
				"ruleId":      f.RuleID,
				"severity":    string(f.Severity),
				"message":     f.Message,
				"location":    map[string]any{"file": f.Location.File, "line": f.Location.Line},
				"fingerprint": f.Fingerprint,
			}}},
		},
	}
	data, _ := json.Marshal(doc)
	path := t.TempDir() + "/baseline.json"
	_ = os.WriteFile(path, data, 0o600)

	b, err := LoadBaseline(path)
	if err != nil {
		t.Fatal(err)
	}
	if b.Root != "/repo" {
		t.Errorf("want root /repo, got %q", b.Root)
	}
	if len(b.Findings) != 1 || b.Findings[0].RuleID != "LINT001" {
		t.Errorf("expected 1 finding LINT001, got %+v", b.Findings)
	}
}

// TestLoadBaselineFlatFormat verifies decoding a single-tool flat JSON.
//
//fusa:test REQ-FO-DIF001
func TestLoadBaselineFlatFormat(t *testing.T) {
	doc := map[string]any{
		"findings": []any{map[string]any{
			"ruleId":   "SAFETY001",
			"severity": "ERROR",
			"message":  "critical",
			"location": map[string]any{"file": "main.c"},
		}},
	}
	data, _ := json.Marshal(doc)
	path := t.TempDir() + "/baseline.json"
	_ = os.WriteFile(path, data, 0o600)

	b, err := LoadBaseline(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(b.Findings) != 1 || b.Findings[0].RuleID != "SAFETY001" {
		t.Errorf("expected 1 finding SAFETY001, got %+v", b.Findings)
	}
}

// TestLoadBaselineMissing verifies a missing file is an error.
//
//fusa:test REQ-FO-DIF001
func TestLoadBaselineMissing(t *testing.T) {
	_, err := LoadBaseline("/no/such/file.json")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

// TestRenderText verifies text output contains gate and counts.
//
//fusa:test REQ-FO-DIF003
func TestRenderText(t *testing.T) {
	res := &Result{
		Added: []fusaops.Finding{
			{RuleID: "SAFETY001", Severity: fusaops.SeverityError, Message: "critical",
				Location: fusaops.Location{File: "main.go", Line: 10}},
		},
		Unchanged: 5,
	}
	var buf bytes.Buffer
	if err := Render(&buf, res, "text", false); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{"1 added", "5 unchanged", "SAFETY001", "Gate: FAIL"} {
		if !strings.Contains(out, want) {
			t.Errorf("text render missing %q:\n%s", want, out)
		}
	}
}

// TestRenderJSON verifies JSON output has expected fields.
//
//fusa:test REQ-FO-DIF003
func TestRenderJSON(t *testing.T) {
	res := &Result{Unchanged: 3}
	var buf bytes.Buffer
	if err := Render(&buf, res, "json", false); err != nil {
		t.Fatal(err)
	}
	var out map[string]any
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out["gate"] != "PASS" {
		t.Errorf("expected gate PASS, got %v", out["gate"])
	}
	if _, ok := out["added"]; !ok {
		t.Error("JSON output missing 'added'")
	}
}

// TestRenderStrictGate verifies --strict promotes any new finding to FAIL.
//
//fusa:test REQ-FO-DIF003
func TestRenderStrictGate(t *testing.T) {
	res := &Result{
		Added: []fusaops.Finding{{Severity: fusaops.SeverityWarning, RuleID: "WARN001", Message: "w",
			Location: fusaops.Location{File: "f.go"}}},
	}
	var buf bytes.Buffer
	if err := Render(&buf, res, "text", true); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "Gate: FAIL") {
		t.Errorf("strict mode should fail on new WARNING: %s", buf.String())
	}
}

// TestRenderUnknownFormat verifies error on unsupported format.
//
//fusa:test REQ-FO-DIF003
func TestRenderUnknownFormat(t *testing.T) {
	if err := Render(new(bytes.Buffer), &Result{}, "xml", false); err == nil {
		t.Error("expected error for unknown format")
	}
}

// TestCompareSortOrder verifies added/removed slices are sorted deterministically.
//
//fusa:test REQ-FO-DIF002
func TestCompareSortOrder(t *testing.T) {
	a := withFP(finding("LINT001", "b.go", "msg one", fusaops.SeverityWarning))
	b := withFP(finding("LINT002", "a.go", "msg two", fusaops.SeverityWarning))
	c := withFP(finding("LINT003", "a.go", "msg three", fusaops.SeverityError))
	// baseline is empty; all three are "added"
	baseline := &Baseline{Findings: []fusaops.Finding{}}
	res := Compare(baseline, []fusaops.Finding{a, b, c})
	if len(res.Added) != 3 {
		t.Fatalf("expected 3 added, got %d", len(res.Added))
	}
	// sorted by file first: a.go < b.go
	if res.Added[0].Location.File != "a.go" {
		t.Errorf("first added should be from a.go, got %q", res.Added[0].Location.File)
	}
	// within a.go, same line 1 — sorted by ruleID: LINT002 < LINT003
	if res.Added[0].RuleID != "LINT002" && res.Added[1].RuleID != "LINT003" {
		t.Errorf("wrong sort within same file: %q %q", res.Added[0].RuleID, res.Added[1].RuleID)
	}
}

// TestCompareWithRemediation verifies Remediation is shown in text output.
//
//fusa:test REQ-FO-DIF003
func TestCompareWithRemediation(t *testing.T) {
	f := fusaops.Finding{
		RuleID:      "SAFETY001",
		Severity:    fusaops.SeverityError,
		Message:     "critical",
		Remediation: "use safe_call instead",
		Category:    "safety",
		Location:    fusaops.Location{File: "main.c", Line: 5},
		Fingerprint: "sha256:aabbcc",
	}
	res := &Result{Added: []fusaops.Finding{f}}
	var buf bytes.Buffer
	if err := Render(&buf, res, "text", false); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "use safe_call instead") {
		t.Errorf("remediation not in text output: %s", out)
	}
	if !strings.Contains(out, "[safety]") {
		t.Errorf("category not in text output: %s", out)
	}
}

// TestSummary verifies per-severity counts for added and removed findings.
//
//fusa:test REQ-FO-DIF004
func TestSummary(t *testing.T) {
	res := &Result{
		Added: []fusaops.Finding{
			{Severity: fusaops.SeverityError},
			{Severity: fusaops.SeverityWarning},
			{Severity: fusaops.SeverityInfo},
		},
		Removed: []fusaops.Finding{
			{Severity: fusaops.SeverityError},
		},
	}
	s := res.Summary()
	if s.AddedErrors != 1 || s.AddedWarnings != 1 || s.AddedInfos != 1 {
		t.Errorf("added summary wrong: %+v", s)
	}
	if s.RemovedErrors != 1 || s.RemovedWarnings != 0 {
		t.Errorf("removed summary wrong: %+v", s)
	}
}

// TestRenderTextSeverityDetail verifies severity breakdown appears in text output.
//
//fusa:test REQ-FO-DIF004
func TestRenderTextSeverityDetail(t *testing.T) {
	res := &Result{
		Added: []fusaops.Finding{
			{RuleID: "E1", Severity: fusaops.SeverityError, Message: "e", Location: fusaops.Location{File: "f.go"}},
			{RuleID: "W1", Severity: fusaops.SeverityWarning, Message: "w", Location: fusaops.Location{File: "f.go"}},
		},
	}
	var buf bytes.Buffer
	if err := Render(&buf, res, "text", false); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "1 error") {
		t.Errorf("severity detail missing '1 error': %s", out)
	}
	if !strings.Contains(out, "1 warning") {
		t.Errorf("severity detail missing '1 warning': %s", out)
	}
}

// TestRenderTextSeverityDetailPlural verifies plural severity labels (2 errors, etc.)
// and info-severity detail in text output.
//
//fusa:test REQ-FO-DIF004
func TestRenderTextSeverityDetailPlural(t *testing.T) {
	finding := func(id string, sev fusaops.Severity) fusaops.Finding {
		return fusaops.Finding{RuleID: id, Severity: sev, Message: "m", Location: fusaops.Location{File: "f.go"}}
	}
	res := &Result{
		Added: []fusaops.Finding{
			finding("E1", fusaops.SeverityError),
			finding("E2", fusaops.SeverityError),
			finding("W1", fusaops.SeverityWarning),
			finding("W2", fusaops.SeverityWarning),
			finding("I1", fusaops.SeverityInfo),
		},
	}
	var buf bytes.Buffer
	if err := Render(&buf, res, "text", false); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "2 errors") {
		t.Errorf("expected '2 errors' in output: %s", out)
	}
	if !strings.Contains(out, "2 warnings") {
		t.Errorf("expected '2 warnings' in output: %s", out)
	}
	if !strings.Contains(out, "1 info") {
		t.Errorf("expected '1 info' in output: %s", out)
	}
}

// TestRenderTextSeverityDetailInfoPlural verifies plural "infos" label.
//
//fusa:test REQ-FO-DIF004
func TestRenderTextSeverityDetailInfoPlural(t *testing.T) {
	finding := func(id string, sev fusaops.Severity) fusaops.Finding {
		return fusaops.Finding{RuleID: id, Severity: sev, Message: "m", Location: fusaops.Location{File: "f.go"}}
	}
	res := &Result{
		Added: []fusaops.Finding{
			finding("I1", fusaops.SeverityInfo),
			finding("I2", fusaops.SeverityInfo),
		},
	}
	var buf bytes.Buffer
	if err := Render(&buf, res, "text", false); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "2 infos") {
		t.Errorf("expected '2 infos' in output: %s", out)
	}
}

// TestRenderJSONSummary verifies JSON output includes generatedAt and summary fields.
//
//fusa:test REQ-FO-DIF004
func TestRenderJSONSummary(t *testing.T) {
	res := &Result{
		Added: []fusaops.Finding{
			{Severity: fusaops.SeverityError, RuleID: "E1", Message: "e", Location: fusaops.Location{File: "f.go"}},
		},
	}
	var buf bytes.Buffer
	if err := Render(&buf, res, "json", false); err != nil {
		t.Fatal(err)
	}
	var out map[string]any
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if _, ok := out["generatedAt"]; !ok {
		t.Error("JSON missing generatedAt")
	}
	summary, ok := out["summary"].(map[string]any)
	if !ok {
		t.Fatalf("JSON missing summary, got: %v", out)
	}
	if summary["addedErrors"] != float64(1) {
		t.Errorf("addedErrors want 1, got %v", summary["addedErrors"])
	}
}

// TestSaveBaseline verifies that SaveBaseline writes a loadable flat baseline.
//
//fusa:test REQ-FO-DIF005
func TestSaveBaseline(t *testing.T) {
	f := withFP(finding("LINT001", "main.go", "unused variable", fusaops.SeverityWarning))
	path := t.TempDir() + "/baseline.json"
	if err := SaveBaseline(path, []fusaops.Finding{f}); err != nil {
		t.Fatal(err)
	}
	b, err := LoadBaseline(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(b.Findings) != 1 || b.Findings[0].RuleID != "LINT001" {
		t.Errorf("loaded baseline wrong: %+v", b.Findings)
	}
}

// TestSaveBaselineEmpty verifies SaveBaseline writes an empty findings array (not null).
//
//fusa:test REQ-FO-DIF005
func TestSaveBaselineEmpty(t *testing.T) {
	path := t.TempDir() + "/baseline.json"
	if err := SaveBaseline(path, nil); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), `"findings": []`) {
		t.Errorf("expected empty array, got: %s", data)
	}
}

//fusa:test REQ-FO-DIF006
func TestRenderDiffHTML(t *testing.T) {
	f := withFP(finding("SAFETY001", "main.go", "critical err", fusaops.SeverityError))
	baseline := &Baseline{Findings: []fusaops.Finding{}}
	res := Compare(baseline, []fusaops.Finding{f})

	var buf bytes.Buffer
	if err := Render(&buf, res, "html", false); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "<!DOCTYPE html>") {
		t.Error("expected HTML doctype")
	}
	if !strings.Contains(out, "SAFETY001") {
		t.Error("expected finding rule ID in HTML output")
	}
	if !strings.Contains(out, "gate-fail") {
		t.Error("expected fail gate in HTML output")
	}
}

//fusa:test REQ-FO-DIF006
func TestRenderDiffHTMLPass(t *testing.T) {
	baseline := &Baseline{Findings: []fusaops.Finding{}}
	res := Compare(baseline, []fusaops.Finding{})

	var buf bytes.Buffer
	if err := Render(&buf, res, "html", false); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "gate-pass") {
		t.Error("expected pass gate in HTML output")
	}
}

//fusa:test REQ-FO-DIF006
func TestRenderDiffMarkdown(t *testing.T) {
	f := withFP(finding("SAFETY001", "main.go", "critical err", fusaops.SeverityError))
	baseline := &Baseline{Findings: []fusaops.Finding{}}
	res := Compare(baseline, []fusaops.Finding{f})

	var buf bytes.Buffer
	if err := Render(&buf, res, "markdown", false); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "# FuSaOps Diff") {
		t.Error("expected markdown heading")
	}
	if !strings.Contains(out, "SAFETY001") {
		t.Error("expected finding rule ID in markdown output")
	}
	if !strings.Contains(out, "Added") {
		t.Error("expected Added section in markdown output")
	}
}

//fusa:test REQ-FO-DIF006
func TestRenderDiffMarkdownAlias(t *testing.T) {
	baseline := &Baseline{Findings: []fusaops.Finding{}}
	res := Compare(baseline, []fusaops.Finding{})

	var buf bytes.Buffer
	if err := Render(&buf, res, "md", false); err != nil {
		t.Fatalf("md alias failed: %v", err)
	}
	if !strings.Contains(buf.String(), "# FuSaOps Diff") {
		t.Error("md alias should produce same output as markdown")
	}
}

//fusa:test REQ-FO-DIF006
func TestRenderDiffUnsupportedFormat(t *testing.T) {
	baseline := &Baseline{}
	res := Compare(baseline, nil)
	if err := Render(&bytes.Buffer{}, res, "xml", false); err == nil {
		t.Error("expected error for unsupported format")
	}
}

// TestCompareSortByLine verifies that when added findings share a file but have
// different line numbers, they are sorted by line ascending.
//
//fusa:test REQ-FO-DIF002
func TestCompareSortByLine(t *testing.T) {
	a := fusaops.Finding{RuleID: "R1", Severity: fusaops.SeverityWarning, Location: fusaops.Location{File: "a.go", Line: 10}, Fingerprint: "sha256:aaa111"}
	b := fusaops.Finding{RuleID: "R1", Severity: fusaops.SeverityWarning, Location: fusaops.Location{File: "a.go", Line: 3}, Fingerprint: "sha256:bbb222"}
	res := Compare(&Baseline{Findings: []fusaops.Finding{}}, []fusaops.Finding{a, b})
	if len(res.Added) != 2 {
		t.Fatalf("expected 2 added, got %d", len(res.Added))
	}
	if res.Added[0].Location.Line != 3 {
		t.Errorf("expected line 3 first (sorted ascending), got %d", res.Added[0].Location.Line)
	}
}

// TestCompareSortByRuleID verifies that when added findings share file and line,
// they are sorted by ruleID ascending.
//
//fusa:test REQ-FO-DIF002
func TestCompareSortByRuleID(t *testing.T) {
	a := fusaops.Finding{RuleID: "ZZZ", Severity: fusaops.SeverityWarning, Location: fusaops.Location{File: "a.go", Line: 1}, Fingerprint: "sha256:zzz999"}
	b := fusaops.Finding{RuleID: "AAA", Severity: fusaops.SeverityWarning, Location: fusaops.Location{File: "a.go", Line: 1}, Fingerprint: "sha256:aaa000"}
	res := Compare(&Baseline{Findings: []fusaops.Finding{}}, []fusaops.Finding{a, b})
	if len(res.Added) != 2 {
		t.Fatalf("expected 2 added, got %d", len(res.Added))
	}
	if res.Added[0].RuleID != "AAA" {
		t.Errorf("expected AAA first (sorted by ruleID), got %q", res.Added[0].RuleID)
	}
}

// TestRenderDiffMarkdownWithRemovals verifies the Removed section appears in
// markdown output when findings exist in baseline but not in current.
//
//fusa:test REQ-FO-DIF006
func TestRenderDiffMarkdownWithRemovals(t *testing.T) {
	f := fusaops.Finding{
		RuleID: "LINT001", Severity: fusaops.SeverityError, Message: "old issue",
		Location: fusaops.Location{File: "x.go", Line: 5}, Fingerprint: "sha256:rem001",
	}
	baseline := &Baseline{Findings: []fusaops.Finding{f}}
	res := Compare(baseline, []fusaops.Finding{})
	var buf bytes.Buffer
	if err := Render(&buf, res, "markdown", false); err != nil {
		t.Fatalf("Render markdown: %v", err)
	}
	if !strings.Contains(buf.String(), "## ➖ Removed") {
		t.Errorf("expected '## ➖ Removed' section in markdown output:\n%s", buf.String())
	}
}

// TestSaveBaselineWriteError verifies SaveBaseline returns an error when the
// parent directory does not exist.
//
//fusa:test REQ-FO-DIF005
func TestSaveBaselineWriteError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing", "baseline.json")
	if err := SaveBaseline(path, nil); err == nil {
		t.Error("SaveBaseline: expected error for non-existent parent directory")
	}
}

// TestRemovedSortByLine verifies that when removed findings share a file but
// differ by line, the Removed slice is sorted by line ascending.
//
//fusa:test REQ-FO-DIF002
func TestRemovedSortByLine(t *testing.T) {
	a := fusaops.Finding{RuleID: "R1", Severity: fusaops.SeverityWarning, Location: fusaops.Location{File: "b.go", Line: 20}, Fingerprint: "sha256:rsl001"}
	b := fusaops.Finding{RuleID: "R1", Severity: fusaops.SeverityWarning, Location: fusaops.Location{File: "b.go", Line: 7}, Fingerprint: "sha256:rsl002"}
	baseline := &Baseline{Findings: []fusaops.Finding{a, b}}
	res := Compare(baseline, []fusaops.Finding{})
	if len(res.Removed) != 2 {
		t.Fatalf("expected 2 removed, got %d", len(res.Removed))
	}
	if res.Removed[0].Location.Line != 7 {
		t.Errorf("expected line 7 first (sorted ascending), got %d", res.Removed[0].Location.Line)
	}
}

// TestRemovedSortByRuleID verifies that when removed findings share file and
// line, the Removed slice is sorted by ruleID ascending.
//
//fusa:test REQ-FO-DIF002
func TestRemovedSortByRuleID(t *testing.T) {
	a := fusaops.Finding{RuleID: "ZZZ", Severity: fusaops.SeverityWarning, Location: fusaops.Location{File: "b.go", Line: 1}, Fingerprint: "sha256:rsrid01"}
	b := fusaops.Finding{RuleID: "AAA", Severity: fusaops.SeverityWarning, Location: fusaops.Location{File: "b.go", Line: 1}, Fingerprint: "sha256:rsrid02"}
	baseline := &Baseline{Findings: []fusaops.Finding{a, b}}
	res := Compare(baseline, []fusaops.Finding{})
	if len(res.Removed) != 2 {
		t.Fatalf("expected 2 removed, got %d", len(res.Removed))
	}
	if res.Removed[0].RuleID != "AAA" {
		t.Errorf("expected AAA first (sorted by ruleID), got %q", res.Removed[0].RuleID)
	}
}

// TestSummaryRemovedInfo verifies that INFO-severity removed findings increment
// RemovedInfos (the default branch in Summary).
//
//fusa:test REQ-FO-DIF004
func TestSummaryRemovedInfo(t *testing.T) {
	res := &Result{
		Removed: []fusaops.Finding{
			{Severity: fusaops.SeverityInfo},
		},
	}
	s := res.Summary()
	if s.RemovedInfos != 1 {
		t.Errorf("RemovedInfos: want 1, got %d", s.RemovedInfos)
	}
}

// TestRenderTextWithRemovals verifies the renderText path when Removed findings
// are present, covering the removedStr detail suffix branch.
//
//fusa:test REQ-FO-DIF003
func TestRenderTextWithRemovals(t *testing.T) {
	f := fusaops.Finding{
		RuleID: "LINT001", Severity: fusaops.SeverityError, Message: "fixed issue",
		Location: fusaops.Location{File: "a.go", Line: 3}, Fingerprint: "sha256:txtrem01",
	}
	baseline := &Baseline{Findings: []fusaops.Finding{f}}
	res := Compare(baseline, []fusaops.Finding{})
	var buf bytes.Buffer
	if err := Render(&buf, res, "text", false); err != nil {
		t.Fatalf("Render text: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "1 removed") {
		t.Errorf("text output missing '1 removed':\n%s", out)
	}
	if !strings.Contains(out, "──── Removed ────") {
		t.Errorf("text output missing Removed section:\n%s", out)
	}
}
