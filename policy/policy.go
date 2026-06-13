// Package policy evaluates org-wide safety rules over an aggregated report.
//
// A Policy is a JSON config listing Rules; each Rule defines a constraint
// (max findings of a given severity, or a required PASS/WARN status) optionally
// scoped to a specific language or tool. Evaluate returns a PolicyReport
// recording PASS/FAIL for each rule with an explanatory message.
package policy

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	fusaops "github.com/SoundMatt/FuSaOps"
	"github.com/SoundMatt/FuSaOps/report"
)

// Policy is the JSON configuration for a policy evaluation.
//
//fusa:req REQ-FO-POL001
type Policy struct {
	Name  string `json:"name"`
	Rules []Rule `json:"rules"`
}

// Rule is a single policy constraint.
//
//fusa:req REQ-FO-POL001
type Rule struct {
	// ID is an optional human-readable rule identifier.
	ID string `json:"id,omitempty"`
	// Language restricts the rule to a specific language (empty = all).
	Language string `json:"language,omitempty"`
	// Tool restricts the rule to a specific tool (empty = all).
	Tool string `json:"tool,omitempty"`
	// MaxFindings sets an upper bound on total findings.
	// Negative = not set.
	MaxFindings int `json:"maxFindings,omitempty"`
	// MaxErrors sets an upper bound on ERROR-severity findings.
	// Negative = not set.
	MaxErrors int `json:"maxErrors,omitempty"`
	// MaxWarnings sets an upper bound on WARNING-severity findings.
	// Negative = not set.
	MaxWarnings int `json:"maxWarnings,omitempty"`
	// RequireStatus mandates that the scoped area has at most this verdict.
	// "PASS" means zero errors AND zero warnings; "WARN" means zero errors.
	// Empty = not enforced.
	RequireStatus string `json:"requireStatus,omitempty"`
}

// RuleResult records the outcome of evaluating one rule.
//
//fusa:req REQ-FO-POL002
type RuleResult struct {
	Rule    Rule   `json:"rule"`
	Passed  bool   `json:"passed"`
	Message string `json:"message"`
}

// PolicyReport aggregates rule results into an overall verdict.
//
//fusa:req REQ-FO-POL002
type PolicyReport struct {
	Policy  string       `json:"policy"`
	Results []RuleResult `json:"results"`
	Passed  int          `json:"passed"`
	Failed  int          `json:"failed"`
}

// HasFailures returns true when at least one rule failed.
//
//fusa:req REQ-FO-POL002
func (r *PolicyReport) HasFailures() bool { return r.Failed > 0 }

// Status returns "PASS" or "FAIL" for the policy report.
//
//fusa:req REQ-FO-POL002
func (r *PolicyReport) Status() string {
	if r.HasFailures() {
		return "FAIL"
	}
	return "PASS"
}

// LoadPolicy reads and parses a policy JSON config file.
//
//fusa:req REQ-FO-POL001
func LoadPolicy(path string) (Policy, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Policy{}, fmt.Errorf("policy: read %s: %w", path, err)
	}
	var p Policy
	if err := json.Unmarshal(data, &p); err != nil {
		return Policy{}, fmt.Errorf("policy: parse %s: %w", path, err)
	}
	return p, nil
}

// Evaluate runs every rule in p against rep and returns a PolicyReport.
//
//fusa:req REQ-FO-POL003
func Evaluate(p Policy, rep *report.AggregateReport) *PolicyReport {
	pr := &PolicyReport{Policy: p.Name}
	for _, rule := range p.Rules {
		rr := evalRule(rule, rep)
		pr.Results = append(pr.Results, rr)
		if rr.Passed {
			pr.Passed++
		} else {
			pr.Failed++
		}
	}
	return pr
}

func evalRule(rule Rule, rep *report.AggregateReport) RuleResult {
	var total, errors, warnings int
	for _, c := range rep.Components {
		if rule.Language != "" && string(c.Language) != rule.Language {
			continue
		}
		if rule.Tool != "" && c.Tool != rule.Tool {
			continue
		}
		for _, f := range c.Findings {
			total++
			switch f.Severity {
			case fusaops.SeverityError:
				errors++
			case fusaops.SeverityWarning:
				warnings++
			}
		}
	}

	id := rule.ID
	if id == "" {
		id = "rule"
	}
	scope := scopeLabel(rule)

	if rule.RequireStatus != "" {
		switch rule.RequireStatus {
		case "PASS":
			if errors > 0 || warnings > 0 {
				return fail(rule, fmt.Sprintf("%s: require PASS%s but got %d errors, %d warnings", id, scope, errors, warnings))
			}
		case "WARN":
			if errors > 0 {
				return fail(rule, fmt.Sprintf("%s: require WARN or better%s but got %d errors", id, scope, errors))
			}
		}
	}
	if rule.MaxErrors > 0 && errors > rule.MaxErrors {
		return fail(rule, fmt.Sprintf("%s: %d errors%s exceeds max %d", id, errors, scope, rule.MaxErrors))
	}
	if rule.MaxWarnings > 0 && warnings > rule.MaxWarnings {
		return fail(rule, fmt.Sprintf("%s: %d warnings%s exceeds max %d", id, warnings, scope, rule.MaxWarnings))
	}
	if rule.MaxFindings > 0 && total > rule.MaxFindings {
		return fail(rule, fmt.Sprintf("%s: %d findings%s exceeds max %d", id, total, scope, rule.MaxFindings))
	}
	return RuleResult{Rule: rule, Passed: true, Message: fmt.Sprintf("%s: PASS%s", id, scope)}
}

func fail(rule Rule, msg string) RuleResult {
	return RuleResult{Rule: rule, Passed: false, Message: msg}
}

func scopeLabel(rule Rule) string {
	if rule.Language != "" && rule.Tool != "" {
		return fmt.Sprintf(" [%s/%s]", rule.Language, rule.Tool)
	}
	if rule.Language != "" {
		return fmt.Sprintf(" [language=%s]", rule.Language)
	}
	if rule.Tool != "" {
		return fmt.Sprintf(" [tool=%s]", rule.Tool)
	}
	return ""
}

// Render writes the PolicyReport to w in text or json format.
//
//fusa:req REQ-FO-POL004
func Render(w io.Writer, pr *PolicyReport, format string) error {
	switch format {
	case "json", "":
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(pr)
	case "text":
		return renderText(w, pr)
	default:
		return fmt.Errorf("policy: unsupported format %q (want text or json)", format)
	}
}

func renderText(w io.Writer, pr *PolicyReport) error {
	fmt.Fprintf(w, "Policy: %s  [%s]\n", pr.Policy, pr.Status())
	fmt.Fprintf(w, "%d passed, %d failed\n\n", pr.Passed, pr.Failed)
	for _, r := range pr.Results {
		status := "PASS"
		if !r.Passed {
			status = "FAIL"
		}
		fmt.Fprintf(w, "  [%s] %s\n", status, r.Message)
	}
	fmt.Fprintf(w, "\nRESULT: %s\n", pr.Status())
	return nil
}

// RenderToFile writes the policy report to path, or to w if path is empty.
//
//fusa:req REQ-FO-POL004
func RenderToFile(w io.Writer, pr *PolicyReport, format, path string) error {
	if path == "" {
		return Render(w, pr, format)
	}
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("policy: create output: %w", err)
	}
	defer f.Close()
	return Render(f, pr, format)
}
