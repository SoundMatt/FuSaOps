package policy

// Gap tests covering uncovered branches in policy.go.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	fusaops "github.com/SoundMatt/FuSaOps"
	"github.com/SoundMatt/FuSaOps/report"
)

// TestLoadPolicyBadJSON verifies LoadPolicy returns an error when the file
// contains malformed JSON, covering policy.go:98.49,100.3.
//
//fusa:test REQ-FO-POL001
func TestLoadPolicyBadJSON(t *testing.T) {
	f := filepath.Join(t.TempDir(), "policy.json")
	if err := os.WriteFile(f, []byte("{bad json"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := LoadPolicy(f)
	if err == nil {
		t.Error("LoadPolicy: expected error for malformed JSON, got nil")
	}
}

// TestEvalRuleToolScopeContinue verifies that evalRule skips components whose
// Tool does not match the rule's Tool constraint, covering
// policy.go:127.45,128.12 (the continue branch).
//
//fusa:test REQ-FO-POL003
func TestEvalRuleToolScopeContinue(t *testing.T) {
	// Report has one component with Tool "cfusa" and an ERROR finding.
	rep := report.New(".", "test", []report.Component{{
		Language: fusaops.LangC, Tool: "cfusa", Available: true,
		Findings: []fusaops.Finding{{
			Language: fusaops.LangC, Tool: "cfusa",
			Severity: fusaops.SeverityError, RuleID: "E001",
		}},
	}})
	// Rule scoped to Tool "gofusa" — cfusa component is skipped, so no errors counted.
	rule := Rule{ID: "R1", Tool: "gofusa", RequireStatus: "PASS"}
	result := evalRule(rule, rep)
	if !result.Passed {
		t.Errorf("evalRule: expected PASS when component tool is filtered out, got: %s", result.Message)
	}
}

// TestEvalRuleEmptyID verifies that evalRule substitutes "rule" when the rule
// ID is empty, covering policy.go:142.14,144.3.
//
//fusa:test REQ-FO-POL003
func TestEvalRuleEmptyID(t *testing.T) {
	result := evalRule(Rule{}, report.New(".", "test", nil))
	if !strings.HasPrefix(result.Message, "rule:") {
		t.Errorf("evalRule: expected message to start with \"rule:\", got %q", result.Message)
	}
}

// TestScopeLabelBoth verifies scopeLabel returns a combined label when both
// Language and Tool are set, covering policy.go:176.44,178.3.
//
//fusa:test REQ-FO-POL003
func TestScopeLabelBoth(t *testing.T) {
	got := scopeLabel(Rule{Language: "go", Tool: "gofusa"})
	if got != " [go/gofusa]" {
		t.Errorf("scopeLabel both: got %q, want \" [go/gofusa]\"", got)
	}
}

// TestRenderTextFail verifies renderText renders a FAIL status line when a
// result is not passed, covering policy.go:215.16,217.4.
//
//fusa:test REQ-FO-POL004
func TestRenderTextFail(t *testing.T) {
	pr := &PolicyReport{
		Policy: "P",
		Failed: 1,
		Results: []RuleResult{
			{Rule: Rule{ID: "R1"}, Passed: false, Message: "R1: error exceeded"},
		},
	}
	var sb strings.Builder
	if err := renderText(&sb, pr); err != nil {
		t.Fatalf("renderText: %v", err)
	}
	if !strings.Contains(sb.String(), "[FAIL]") {
		t.Errorf("renderText: expected [FAIL] in output, got:\n%s", sb.String())
	}
}

// TestRenderMarkdownEmptyPolicyName verifies renderMarkdown substitutes
// "Policy" when PolicyReport.Policy is empty, covering
// policy.go:249.16,251.3.
//
//fusa:test REQ-FO-POL006
func TestRenderMarkdownEmptyPolicyName(t *testing.T) {
	pr := &PolicyReport{Passed: 1}
	var sb strings.Builder
	if err := renderMarkdown(&sb, pr); err != nil {
		t.Fatalf("renderMarkdown: %v", err)
	}
	if !strings.Contains(sb.String(), "FuSaOps — Policy") {
		t.Errorf("renderMarkdown: expected \"FuSaOps — Policy\" in output, got:\n%s", sb.String())
	}
}

// TestRenderToFileEmptyPath verifies RenderToFile calls Render directly when
// path is empty, covering policy.go:312.16,314.3.
//
//fusa:test REQ-FO-POL004
func TestRenderToFileEmptyPath(t *testing.T) {
	pr := &PolicyReport{Policy: "test", Passed: 1}
	var sb strings.Builder
	if err := RenderToFile(&sb, pr, "json", ""); err != nil {
		t.Fatalf("RenderToFile empty path: %v", err)
	}
	if !strings.Contains(sb.String(), `"policy"`) {
		t.Errorf("RenderToFile empty path: expected JSON output, got:\n%s", sb.String())
	}
}
