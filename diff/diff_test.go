package diff

import (
	"bytes"
	"encoding/json"
	"os"
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
