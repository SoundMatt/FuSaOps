// Package report defines the aggregated multi-language FuSaOps report and its
// renderers (text, json, html, sarif, junit, csv, markdown).
//
// An AggregateReport is the union of every per-language component report
// produced by the orchestrator. It is the single artefact the CLI prints, the
// web UI renders, and CI uploads as evidence.
package report

import (
	"fmt"
	"io"
	"os"
	"sort"
	"time"

	fusaops "github.com/SoundMatt/FuSaOps"
)

// Summary holds aggregate finding counts.
//
//fusa:req REQ-FO-RPT001
type Summary struct {
	Total    int `json:"total"`
	Errors   int `json:"errors"`
	Warnings int `json:"warnings"`
	Infos    int `json:"infos"`
}

// add folds a finding's severity into the summary.
func (s *Summary) add(sev fusaops.Severity) {
	s.AddFinding(sev)
}

// AddFinding increments the appropriate counter for a finding's severity.
// Exported for use by packages that recompute summaries after post-processing
// (e.g. suppression filtering).
//
//fusa:req REQ-FO-SUP004
func (s *Summary) AddFinding(sev fusaops.Severity) {
	s.Total++
	switch sev {
	case fusaops.SeverityError:
		s.Errors++
	case fusaops.SeverityWarning:
		s.Warnings++
	case fusaops.SeverityInfo:
		s.Infos++
	}
}

// Status returns the overall PASS/WARN/FAIL verdict for a summary.
//
//fusa:req REQ-FO-RPT002
func (s Summary) Status() string {
	switch {
	case s.Errors > 0:
		return "FAIL"
	case s.Warnings > 0:
		return "WARN"
	default:
		return "PASS"
	}
}

// Component is one language's contribution to the aggregate report.
//
//fusa:req REQ-FO-RPT003
//fusa:req REQ-FO-RPT017
type Component struct {
	Language           fusaops.Language  `json:"language"`
	Tool               string            `json:"tool"`
	Dir                string            `json:"dir,omitempty"` // relative path when component-pinned
	Available          bool              `json:"available"`
	Skipped            string            `json:"skipped,omitempty"` // reason, if not run
	Findings           []fusaops.Finding `json:"findings"`
	SuppressedFindings []fusaops.Finding `json:"suppressedFindings,omitempty"`
	Summary            Summary           `json:"summary"`
}

// AggregateReport is the top-level multi-language FuSaOps report.
//
//fusa:req REQ-FO-RPT004
//fusa:req REQ-FO-RPT018
type AggregateReport struct {
	GeneratedAt time.Time   `json:"generatedAt"`
	Root        string      `json:"root"`
	Project     string      `json:"project,omitempty"`
	Components  []Component `json:"components"`
	Summary     Summary     `json:"summary"`
	// Suppressed is the count of findings filtered by a suppression config.
	//
	//fusa:req REQ-FO-SUP004
	Suppressed int `json:"suppressed,omitempty"`
	// SuppressedComponents lists the tool names of components that had one or
	// more findings suppressed.
	//
	//fusa:req REQ-FO-RPT018
	SuppressedComponents []string `json:"suppressedComponents,omitempty"`
}

// RenderOptions controls optional behaviour during report rendering.
//
//fusa:req REQ-FO-RPT017
//fusa:req REQ-FO-RPT019
type RenderOptions struct {
	// ShowSuppressed causes suppressed findings to be included in the output.
	ShowSuppressed bool
	// ShowFingerprints causes each finding's fingerprint to be shown inline,
	// together with a scaffold fusaops suppress add command.
	ShowFingerprints bool
}

// New builds an AggregateReport from a set of components, computing per
// component and overall summaries. Components are sorted by tool name for
// deterministic output.
//
//fusa:req REQ-FO-RPT005
func New(root, project string, components []Component) *AggregateReport {
	sort.Slice(components, func(i, j int) bool { return components[i].Tool < components[j].Tool })
	r := &AggregateReport{
		GeneratedAt: time.Now().UTC(),
		Root:        root,
		Project:     project,
		Components:  components,
	}
	for i := range r.Components {
		c := &r.Components[i]
		c.Summary = Summary{}
		for _, f := range c.Findings {
			c.Summary.add(f.Severity)
			r.Summary.add(f.Severity)
		}
	}
	return r
}

// HasErrors reports whether any component produced an ERROR-severity finding.
//
//fusa:req REQ-FO-RPT006
func (r *AggregateReport) HasErrors() bool { return r.Summary.Errors > 0 }

// Render writes r to w in the requested format.
//
//fusa:req REQ-FO-RPT007
//fusa:req REQ-FO-RPT013
//fusa:req REQ-FO-RPT014
//fusa:req REQ-FO-RPT015
func Render(w io.Writer, r *AggregateReport, format string) error {
	return RenderWithOptions(w, r, format, RenderOptions{})
}

// RenderWithOptions writes r to w in the requested format, honouring opts.
//
//fusa:req REQ-FO-RPT017
func RenderWithOptions(w io.Writer, r *AggregateReport, format string, opts RenderOptions) error {
	switch format {
	case "", "text":
		return renderText(w, r, opts)
	case "json":
		return renderJSON(w, r)
	case "html":
		return renderHTML(w, r, opts)
	case "sarif":
		return renderSARIF(w, r)
	case "junit":
		return renderJUnit(w, r)
	case "csv":
		return renderCSV(w, r)
	case "markdown", "md":
		return renderMarkdown(w, r, opts)
	default:
		return fmt.Errorf("report: unsupported format %q", format)
	}
}

// RenderToFile writes r to path in format, or to stdout if path is empty.
//
//fusa:req REQ-FO-RPT008
func RenderToFile(r *AggregateReport, format, path string) error {
	return RenderToFileWithOptions(r, format, path, RenderOptions{})
}

// RenderToFileWithOptions writes r to path in format with opts, or to stdout if path is empty.
//
//fusa:req REQ-FO-RPT017
func RenderToFileWithOptions(r *AggregateReport, format, path string, opts RenderOptions) error {
	if path == "" {
		return RenderWithOptions(os.Stdout, r, format, opts)
	}
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("report: create %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()
	return RenderWithOptions(f, r, format, opts)
}
