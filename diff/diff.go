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
func (r *Result) HasNewErrors() bool {
	for _, f := range r.Added {
		if f.Severity == fusaops.SeverityError {
			return true
		}
	}
	return false
}

// HasNewFindings reports whether any findings were added.
func (r *Result) HasNewFindings() bool { return len(r.Added) > 0 }

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
// format is "text" or "json"; empty defaults to "text".
// strict causes Render to treat any new finding (not just errors) as a failure
// in the gate verdict.
//
//fusa:req REQ-FO-DIF003
func Render(w io.Writer, res *Result, format string, strict bool) error {
	switch format {
	case "", "text":
		return renderText(w, res, strict)
	case "json":
		return renderJSON(w, res, strict)
	default:
		return fmt.Errorf("diff: unsupported format %q", format)
	}
}

func renderText(w io.Writer, res *Result, strict bool) error {
	fmt.Fprintf(w, "FuSaOps Diff — %d added, %d removed, %d unchanged\n",
		len(res.Added), len(res.Removed), res.Unchanged)
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
		Added     []fusaops.Finding `json:"added"`
		Removed   []fusaops.Finding `json:"removed"`
		Unchanged int               `json:"unchanged"`
		Gate      string            `json:"gate"`
	}
	out := jsonResult{
		Added:     res.Added,
		Removed:   res.Removed,
		Unchanged: res.Unchanged,
		Gate:      res.gate(strict),
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
