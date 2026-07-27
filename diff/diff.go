// Package diff compares two FuSaOps check runs to detect new or resolved
// findings. It is the engine for the fusaops diff command and for CI baseline
// gating: a run that introduces no new ERROR-severity findings against a stored
// baseline exits 0; a run that adds new errors exits 1.
//
// Findings are matched by their fingerprint (§4.2 of the x-FuSa spec). If a
// finding in either the baseline or the current run lacks a fingerprint,
// ComputeFingerprint is called to derive one on the fly so that baselines
// produced before fingerprint adoption can still be compared.
package diff

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"

	fusaops "github.com/SoundMatt/FuSaOps"
)

// Baseline holds findings decoded from a prior check run, used as the
// comparison point for a new run.
//
//fusa:req REQ-FO-DIF001
type Baseline struct {
	Root        string
	Project     string
	GeneratedAt time.Time
	Findings    []fusaops.Finding
}

// baselineDoc decodes the FuSaOps aggregate report format (components[].findings)
// and the flat single-tool format (findings[]) so either can serve as a baseline.
type baselineDoc struct {
	Root        string    `json:"root"`
	Project     string    `json:"project,omitempty"`
	GeneratedAt time.Time `json:"generatedAt"`
	Components  []struct {
		Findings []fusaops.Finding `json:"findings"`
	} `json:"components"`
	Findings []fusaops.Finding `json:"findings"`
}

// LoadBaseline reads a JSON file produced by fusaops check (aggregate format)
// or a single x-FuSa tool check (flat format) and returns a Baseline for
// comparison.
//
//fusa:req REQ-FO-DIF001
func LoadBaseline(path string) (*Baseline, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("diff: read baseline %s: %w", path, err)
	}
	var doc baselineDoc
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("diff: parse baseline: %w", err)
	}
	b := &Baseline{
		Root:        doc.Root,
		Project:     doc.Project,
		GeneratedAt: doc.GeneratedAt,
	}
	for _, c := range doc.Components {
		b.Findings = append(b.Findings, c.Findings...)
	}
	b.Findings = append(b.Findings, doc.Findings...)
	return b, nil
}

// Result is the output of Compare: findings added or removed between two runs,
// and the count of findings present in both.
//
//fusa:req REQ-FO-DIF002
type Result struct {
	Added     []fusaops.Finding `json:"added"`
	Removed   []fusaops.Finding `json:"removed"`
	Unchanged int               `json:"unchanged"`
}

// HasNewErrors reports whether any added findings carry ERROR severity.
//
//fusa:req REQ-FO-DIF002
func (r *Result) HasNewErrors() bool {
	for _, f := range r.Added {
		if f.Severity == fusaops.SeverityError {
			return true
		}
	}
	return false
}

// HasNewFindings reports whether any findings were added.
//
//fusa:req REQ-FO-DIF007
func (r *Result) HasNewFindings() bool { return len(r.Added) > 0 }

// FindingSummary holds per-severity counts for added and removed findings.
//
//fusa:req REQ-FO-DIF004
type FindingSummary struct {
	AddedErrors     int `json:"addedErrors"`
	AddedWarnings   int `json:"addedWarnings"`
	AddedInfos      int `json:"addedInfos"`
	RemovedErrors   int `json:"removedErrors"`
	RemovedWarnings int `json:"removedWarnings"`
	RemovedInfos    int `json:"removedInfos"`
}

// Summary returns per-severity counts for added and removed findings.
//
//fusa:req REQ-FO-DIF004
func (r *Result) Summary() FindingSummary {
	var s FindingSummary
	for _, f := range r.Added {
		switch f.Severity {
		case fusaops.SeverityError:
			s.AddedErrors++
		case fusaops.SeverityWarning:
			s.AddedWarnings++
		default:
			s.AddedInfos++
		}
	}
	for _, f := range r.Removed {
		switch f.Severity {
		case fusaops.SeverityError:
			s.RemovedErrors++
		case fusaops.SeverityWarning:
			s.RemovedWarnings++
		default:
			s.RemovedInfos++
		}
	}
	return s
}

// gate returns the CI gate verdict string.
func (r *Result) gate(strict bool) string {
	if strict && r.HasNewFindings() {
		return "FAIL"
	}
	if r.HasNewErrors() {
		return "FAIL"
	}
	return "PASS"
}

// fp returns the fingerprint for a finding, computing it if absent.
func fp(f fusaops.Finding) string {
	if f.Fingerprint != "" {
		return f.Fingerprint
	}
	return fusaops.ComputeFingerprint(f)
}

// Compare returns the diff between baseline and current findings.
// Findings are matched by fingerprint; if absent, ComputeFingerprint is used.
//
//fusa:req REQ-FO-DIF002
func Compare(baseline *Baseline, current []fusaops.Finding) *Result {
	baseSet := make(map[string]fusaops.Finding, len(baseline.Findings))
	for _, f := range baseline.Findings {
		baseSet[fp(f)] = f
	}
	currSet := make(map[string]fusaops.Finding, len(current))
	for _, f := range current {
		currSet[fp(f)] = f
	}

	var res Result
	for h, f := range currSet {
		if _, inBase := baseSet[h]; inBase {
			res.Unchanged++
		} else {
			res.Added = append(res.Added, f)
		}
	}
	for h, f := range baseSet {
		if _, inCurr := currSet[h]; !inCurr {
			res.Removed = append(res.Removed, f)
		}
	}
	sort.Slice(res.Added, func(i, j int) bool {
		a, b := res.Added[i], res.Added[j]
		if a.Location.File != b.Location.File {
			return a.Location.File < b.Location.File
		}
		if a.Location.Line != b.Location.Line {
			return a.Location.Line < b.Location.Line
		}
		return a.RuleID < b.RuleID
	})
	sort.Slice(res.Removed, func(i, j int) bool {
		a, b := res.Removed[i], res.Removed[j]
		if a.Location.File != b.Location.File {
			return a.Location.File < b.Location.File
		}
		if a.Location.Line != b.Location.Line {
			return a.Location.Line < b.Location.Line
		}
		return a.RuleID < b.RuleID
	})
	return &res
}

// Render writes the Result in the requested format to w.
// format is "text", "json", "html", or "markdown"/"md"; empty defaults to "text".
// strict causes Render to treat any new finding (not just errors) as a failure
// in the gate verdict.
//
//fusa:req REQ-FO-DIF003
//fusa:req REQ-FO-DIF006
func Render(w io.Writer, res *Result, format string, strict bool) error {
	switch format {
	case "", "text":
		return renderText(w, res, strict)
	case "json":
		return renderJSON(w, res, strict)
	case "html":
		return renderDiffHTML(w, res, strict)
	case "markdown", "md":
		return renderDiffMarkdown(w, res, strict)
	default:
		return fmt.Errorf("diff: unsupported format %q", format)
	}
}

func renderText(w io.Writer, res *Result, strict bool) error {
	s := res.Summary()
	addedStr := fmt.Sprintf("%d added", len(res.Added))
	if len(res.Added) > 0 {
		addedStr += " (" + severityDetail(s.AddedErrors, s.AddedWarnings, s.AddedInfos) + ")"
	}
	removedStr := fmt.Sprintf("%d removed", len(res.Removed))
	if len(res.Removed) > 0 {
		removedStr += " (" + severityDetail(s.RemovedErrors, s.RemovedWarnings, s.RemovedInfos) + ")"
	}
	fmt.Fprintf(w, "FuSaOps Diff — %s, %s, %d unchanged\n", addedStr, removedStr, res.Unchanged)
	if len(res.Added) > 0 {
		fmt.Fprintln(w, "──── Added ────")
		for _, f := range res.Added {
			printFinding(w, "+", f)
		}
	}
	if len(res.Removed) > 0 {
		fmt.Fprintln(w, "──── Removed ────")
		for _, f := range res.Removed {
			printFinding(w, "-", f)
		}
	}
	fmt.Fprintf(w, "Gate: %s\n", res.gate(strict))
	return nil
}

// severityDetail formats non-zero severity counts as a comma-separated string.
func severityDetail(errors, warnings, infos int) string {
	var parts []string
	if errors > 0 {
		noun := "error"
		if errors != 1 {
			noun = "errors"
		}
		parts = append(parts, fmt.Sprintf("%d %s", errors, noun))
	}
	if warnings > 0 {
		noun := "warning"
		if warnings != 1 {
			noun = "warnings"
		}
		parts = append(parts, fmt.Sprintf("%d %s", warnings, noun))
	}
	if infos > 0 {
		noun := "info"
		if infos != 1 {
			noun = "infos"
		}
		parts = append(parts, fmt.Sprintf("%d %s", infos, noun))
	}
	return strings.Join(parts, ", ")
}

func printFinding(w io.Writer, prefix string, f fusaops.Finding) {
	fmt.Fprintf(w, "%s [%s] %-10s", prefix, f.Severity, f.RuleID)
	if f.Category != "" {
		fmt.Fprintf(w, " [%s]", f.Category)
	}
	fmt.Fprintf(w, " %s", f.Message)
	if f.Location.File != "" {
		fmt.Fprintf(w, " (%s", f.Location.File)
		if f.Location.Line > 0 {
			fmt.Fprintf(w, ":%d", f.Location.Line)
		}
		fmt.Fprint(w, ")")
	}
	fmt.Fprintln(w)
	if f.Remediation != "" {
		fmt.Fprintf(w, "    → %s\n", f.Remediation)
	}
}

func renderJSON(w io.Writer, res *Result, strict bool) error {
	type jsonResult struct {
		GeneratedAt time.Time         `json:"generatedAt"`
		Added       []fusaops.Finding `json:"added"`
		Removed     []fusaops.Finding `json:"removed"`
		Unchanged   int               `json:"unchanged"`
		Summary     FindingSummary    `json:"summary"`
		Gate        string            `json:"gate"`
	}
	out := jsonResult{
		GeneratedAt: time.Now().UTC(),
		Added:       res.Added,
		Removed:     res.Removed,
		Unchanged:   res.Unchanged,
		Summary:     res.Summary(),
		Gate:        res.gate(strict),
	}
	if out.Added == nil {
		out.Added = []fusaops.Finding{}
	}
	if out.Removed == nil {
		out.Removed = []fusaops.Finding{}
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

// renderDiffHTML writes the diff result as a self-contained HTML report.
//
//fusa:req REQ-FO-DIF006
func renderDiffHTML(w io.Writer, res *Result, strict bool) error {
	s := res.Summary()
	gate := res.gate(strict)
	gateClass := "pass"
	if gate == "FAIL" {
		gateClass = "fail"
	}
	fmt.Fprintf(w, `<!DOCTYPE html>
<html lang="en"><head><meta charset="utf-8">
<title>FuSaOps Diff</title>
<style>
body{font:14px/1.5 -apple-system,sans-serif;background:#0f1115;color:#e6e9ef;margin:0;padding:24px}
h1{margin:0 0 4px;font-size:20px}
.gate-pass{color:#2ecc71;font-weight:700} .gate-fail{color:#e74c3c;font-weight:700}
table{width:100%%;border-collapse:collapse;font-size:13px;margin-top:12px}
th,td{text-align:left;padding:6px 8px;border-bottom:1px solid #262b36}
th{color:#8a93a6} .add{color:#2ecc71} .rem{color:#e74c3c} .unch{color:#8a93a6}
</style></head><body>
<h1>FuSaOps Diff — Gate: <span class="gate-%s">%s</span></h1>
<p>%d added (%d err, %d warn, %d info) · %d removed (%d err, %d warn, %d info) · %d unchanged</p>
`, gateClass, gate,
		len(res.Added), s.AddedErrors, s.AddedWarnings, s.AddedInfos,
		len(res.Removed), s.RemovedErrors, s.RemovedWarnings, s.RemovedInfos,
		res.Unchanged)

	if len(res.Added) > 0 || len(res.Removed) > 0 {
		fmt.Fprintf(w, "<table><thead><tr><th>Δ</th><th>Severity</th><th>Rule</th><th>Message</th><th>Location</th></tr></thead><tbody>\n")
		for _, f := range res.Added {
			fmt.Fprintf(w, "<tr class=\"add\"><td>+</td><td>%s</td><td>%s</td><td>%s</td><td>%s:%d</td></tr>\n",
				f.Severity, f.RuleID, f.Message, f.Location.File, f.Location.Line)
		}
		for _, f := range res.Removed {
			fmt.Fprintf(w, "<tr class=\"rem\"><td>-</td><td>%s</td><td>%s</td><td>%s</td><td>%s:%d</td></tr>\n",
				f.Severity, f.RuleID, f.Message, f.Location.File, f.Location.Line)
		}
		fmt.Fprintf(w, "</tbody></table>\n")
	} else {
		fmt.Fprintf(w, "<p>No changes.</p>\n")
	}
	fmt.Fprintf(w, "</body></html>\n")
	return nil
}

// renderDiffMarkdown writes the diff result as GitHub-Flavored Markdown.
//
//fusa:req REQ-FO-DIF006
func renderDiffMarkdown(w io.Writer, res *Result, strict bool) error {
	s := res.Summary()
	gate := res.gate(strict)
	gateBadge := "![PASS](https://img.shields.io/badge/Diff-PASS-brightgreen)"
	if gate == "FAIL" {
		gateBadge = "![FAIL](https://img.shields.io/badge/Diff-FAIL-red)"
	}
	fmt.Fprintf(w, "# FuSaOps Diff %s\n\n", gateBadge)
	fmt.Fprintf(w, "| | Count |\n|---|---|\n")
	fmt.Fprintf(w, "| ➕ Added | %d (%d err, %d warn, %d info) |\n", len(res.Added), s.AddedErrors, s.AddedWarnings, s.AddedInfos)
	fmt.Fprintf(w, "| ➖ Removed | %d (%d err, %d warn, %d info) |\n", len(res.Removed), s.RemovedErrors, s.RemovedWarnings, s.RemovedInfos)
	fmt.Fprintf(w, "| 🔁 Unchanged | %d |\n\n", res.Unchanged)

	if len(res.Added) > 0 {
		fmt.Fprintf(w, "## ➕ Added\n\n| Severity | Rule | Message | Location |\n|---|---|---|---|\n")
		for _, f := range res.Added {
			fmt.Fprintf(w, "| %s | `%s` | %s | `%s:%d` |\n", f.Severity, f.RuleID, f.Message, f.Location.File, f.Location.Line)
		}
		fmt.Fprintln(w)
	}
	if len(res.Removed) > 0 {
		fmt.Fprintf(w, "## ➖ Removed\n\n| Severity | Rule | Message | Location |\n|---|---|---|---|\n")
		for _, f := range res.Removed {
			fmt.Fprintf(w, "| %s | `%s` | %s | `%s:%d` |\n", f.Severity, f.RuleID, f.Message, f.Location.File, f.Location.Line)
		}
		fmt.Fprintln(w)
	}
	return nil
}

// SaveBaseline writes current findings to path in the flat baseline format so
// a passing diff run can update its own baseline in place.
//
//fusa:req REQ-FO-DIF005
func SaveBaseline(path string, findings []fusaops.Finding) error {
	type baselineOut struct {
		GeneratedAt time.Time         `json:"generatedAt"`
		Findings    []fusaops.Finding `json:"findings"`
	}
	out := baselineOut{
		GeneratedAt: time.Now().UTC(),
		Findings:    findings,
	}
	if out.Findings == nil {
		out.Findings = []fusaops.Finding{}
	}
	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return fmt.Errorf("diff: marshal baseline: %w", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("diff: write baseline %s: %w", path, err)
	}
	return nil
}
