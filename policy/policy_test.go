package policy

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	fusaops "github.com/SoundMatt/FuSaOps"
	"github.com/SoundMatt/FuSaOps/report"
)

func makeReport(errs, warns int) *report.AggregateReport {
	var findings []fusaops.Finding
	for range errs {
		findings = append(findings, fusaops.Finding{
			Language: fusaops.LangGo, Tool: "gofusa",
			Severity: fusaops.SeverityError, RuleID: "E001",
		})
	}
	for range warns {
		findings = append(findings, fusaops.Finding{
			Language: fusaops.LangGo, Tool: "gofusa",
			Severity: fusaops.SeverityWarning, RuleID: "W001",
		})
	}
	return report.New(".", "test", []report.Component{{
		Language: fusaops.LangGo, Tool: "gofusa", Available: true,
		Findings: findings,
	}})
}

// TestLoadPolicy verifies JSON config parsing.
//
//fusa:test REQ-FO-POL001
func TestLoadPolicy(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "policy.json")
	content := `{"name":"mypol","rules":[{"id":"R1","maxErrors":0}]}`
	_ = os.WriteFile(path, []byte(content), 0o644)
	p, err := LoadPolicy(path)
	if err != nil {
		t.Fatalf("LoadPolicy: %v", err)
	}
	if p.Name != "mypol" || len(p.Rules) != 1 || p.Rules[0].ID != "R1" {
		t.Errorf("unexpected policy: %+v", p)
	}
}

// TestLoadPolicyMissing verifies a missing file returns an error.
//
//fusa:test REQ-FO-POL001
func TestLoadPolicyMissing(t *testing.T) {
	_, err := LoadPolicy("/nonexistent/policy.json")
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestEvaluatePass verifies a clean report passes all rules.
//
//fusa:test REQ-FO-POL002
//fusa:test REQ-FO-POL003
func TestEvaluatePass(t *testing.T) {
	p := Policy{Name: "test", Rules: []Rule{
		{ID: "R1", MaxErrors: 0},
		{ID: "R2", RequireStatus: "PASS"},
	}}
	pr := Evaluate(p, makeReport(0, 0))
	if pr.Status() != "PASS" || pr.HasFailures() {
		t.Errorf("clean report should PASS all rules: %+v", pr)
	}
	if pr.Passed != 2 || pr.Failed != 0 {
		t.Errorf("counts: passed=%d failed=%d", pr.Passed, pr.Failed)
	}
}

// TestEvaluateMaxErrors verifies maxErrors rule fires when exceeded.
//
//fusa:test REQ-FO-POL003
func TestEvaluateMaxErrors(t *testing.T) {
	p := Policy{Name: "test", Rules: []Rule{{ID: "R1", MaxErrors: 1}}}
	pr := Evaluate(p, makeReport(2, 0))
	if !pr.HasFailures() {
		t.Error("expected failure for maxErrors exceeded")
	}
	if !strings.Contains(pr.Results[0].Message, "2 errors") {
		t.Errorf("message: %q", pr.Results[0].Message)
	}
}

// TestEvaluateMaxWarnings verifies maxWarnings rule fires when exceeded.
//
//fusa:test REQ-FO-POL003
func TestEvaluateMaxWarnings(t *testing.T) {
	p := Policy{Name: "test", Rules: []Rule{{ID: "R1", MaxWarnings: 1}}}
	pr := Evaluate(p, makeReport(0, 3))
	if !pr.HasFailures() {
		t.Error("expected failure for maxWarnings exceeded")
	}
}

// TestEvaluateMaxFindings verifies maxFindings rule fires when exceeded.
//
//fusa:test REQ-FO-POL003
func TestEvaluateMaxFindings(t *testing.T) {
	p := Policy{Name: "test", Rules: []Rule{{ID: "R1", MaxFindings: 2}}}
	pr := Evaluate(p, makeReport(1, 2))
	if !pr.HasFailures() {
		t.Error("expected failure for maxFindings exceeded")
	}
}

// TestEvaluateRequirePass verifies requireStatus:PASS fails when warnings exist.
//
//fusa:test REQ-FO-POL003
func TestEvaluateRequirePass(t *testing.T) {
	p := Policy{Name: "test", Rules: []Rule{{ID: "R1", RequireStatus: "PASS"}}}
	pr := Evaluate(p, makeReport(0, 1))
	if !pr.HasFailures() {
		t.Error("expected failure: requireStatus PASS but warnings exist")
	}
}

// TestEvaluateRequireWarn verifies requireStatus:WARN passes for warnings.
//
//fusa:test REQ-FO-POL003
func TestEvaluateRequireWarn(t *testing.T) {
	p := Policy{Name: "test", Rules: []Rule{{ID: "R1", RequireStatus: "WARN"}}}
	pr := Evaluate(p, makeReport(0, 1))
	if pr.HasFailures() {
		t.Error("requireStatus WARN should pass when only warnings")
	}
	pr2 := Evaluate(p, makeReport(1, 0))
	if !pr2.HasFailures() {
		t.Error("requireStatus WARN should fail when errors exist")
	}
}

// TestEvaluateScopeLanguage verifies a language-scoped rule ignores other languages.
//
//fusa:test REQ-FO-POL003
func TestEvaluateScopeLanguage(t *testing.T) {
	// Rule scoped to java — report only has Go findings
	p := Policy{Name: "test", Rules: []Rule{{ID: "R1", Language: "java", MaxErrors: 0}}}
	pr := Evaluate(p, makeReport(5, 0)) // 5 Go errors, but rule is for java
	if pr.HasFailures() {
		t.Error("language-scoped rule should not count errors from other languages")
	}
}

// TestRenderText verifies text rendering has expected structure.
//
//fusa:test REQ-FO-POL004
func TestRenderText(t *testing.T) {
	p := Policy{Name: "mypol", Rules: []Rule{{ID: "R1", MaxErrors: 0}}}
	pr := Evaluate(p, makeReport(0, 0))
	var sb strings.Builder
	if err := Render(&sb, pr, "text"); err != nil {
		t.Fatalf("Render text: %v", err)
	}
	out := sb.String()
	if !strings.Contains(out, "Policy:") {
		t.Error("missing Policy: header")
	}
	if !strings.Contains(out, "RESULT: PASS") {
		t.Error("missing RESULT: PASS")
	}
}

// TestRenderJSON verifies JSON rendering is parseable.
//
//fusa:test REQ-FO-POL004
func TestRenderJSON(t *testing.T) {
	p := Policy{Name: "mypol", Rules: []Rule{{ID: "R1", MaxErrors: 0}}}
	pr := Evaluate(p, makeReport(0, 0))
	var sb strings.Builder
	if err := Render(&sb, pr, "json"); err != nil {
		t.Fatalf("Render json: %v", err)
	}
	var out PolicyReport
	if err := json.Unmarshal([]byte(sb.String()), &out); err != nil {
		t.Fatalf("not valid JSON: %v", err)
	}
	if out.Policy != "mypol" {
		t.Errorf("policy name: got %q", out.Policy)
	}
}

// TestRenderUnsupportedFormat verifies unknown format returns error.
//
//fusa:test REQ-FO-POL004
func TestRenderUnsupportedFormat(t *testing.T) {
	if err := Render(nil, &PolicyReport{}, "bogus"); err == nil {
		t.Error("expected error for unsupported format")
	}
}

// TestRenderToFile verifies output is written to a file.
//
//fusa:test REQ-FO-POL004
func TestRenderToFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.json")
	pr := &PolicyReport{Policy: "test", Passed: 1}
	if err := RenderToFile(nil, pr, "json", path); err != nil {
		t.Fatalf("RenderToFile: %v", err)
	}
	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), `"policy"`) {
		t.Error("output missing policy field")
	}
}

// TestRenderHTML verifies the HTML policy report contains expected content.
//
//fusa:test REQ-FO-POL005
func TestRenderHTML(t *testing.T) {
	pr := &PolicyReport{
		Policy: "MyPolicy",
		Passed: 2,
		Failed: 1,
		Results: []RuleResult{
			{Rule: Rule{ID: "R001"}, Passed: true, Message: "max errors: OK"},
			{Rule: Rule{ID: "R002"}, Passed: true, Message: "max warnings: OK"},
			{Rule: Rule{ID: "R003"}, Passed: false, Message: "status: got FAIL, need PASS"},
		},
	}
	var buf bytes.Buffer
	if err := Render(&buf, pr, "html"); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{"<!doctype html>", "MyPolicy", "FAIL", "R001", "max errors: OK"} {
		if !strings.Contains(out, want) {
			t.Errorf("html missing %q", want)
		}
	}
}

// TestRenderHTMLPass verifies PASS status renders correctly.
//
//fusa:test REQ-FO-POL005
func TestRenderHTMLPass(t *testing.T) {
	pr := &PolicyReport{Policy: "P", Passed: 1}
	var buf bytes.Buffer
	if err := Render(&buf, pr, "html"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "PASS") {
		t.Error("html: expected PASS badge")
	}
}
