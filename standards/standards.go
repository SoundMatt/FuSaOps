// Package standards implements the §9.3 gap-report roll-up for the x-FuSa
// toolchain.  Each language tool can emit a <standard>-gap-report.json; this
// package aggregates them into one cross-language compliance view.
package standards

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

// GapReport mirrors the §9.3 gap-report document emitted by each x-FuSa tool.
//
//fusa:req REQ-FO-STD001
type GapReport struct {
	SchemaVersion string      `json:"schemaVersion"`
	Kind          string      `json:"kind"`
	Tool          string      `json:"tool"`
	ToolVersion   string      `json:"toolVersion"`
	Language      string      `json:"language"`
	GeneratedAt   string      `json:"generatedAt"`
	Standard      string      `json:"standard"`
	Objectives    []Objective `json:"objectives"`
	Summary       Summary     `json:"summary"`
}

// Objective is one standards clause's compliance status.
//
//fusa:req REQ-FO-STD002
type Objective struct {
	ID       string   `json:"id"`
	Title    string   `json:"title,omitempty"`
	Clause   string   `json:"clause,omitempty"`
	Status   string   `json:"status"` // satisfied | partial | gap
	Evidence []string `json:"evidence,omitempty"`
	Findings []string `json:"findings,omitempty"`
}

// Summary counts objectives by status.
//
//fusa:req REQ-FO-STD001
type Summary struct {
	Total     int `json:"total"`
	Satisfied int `json:"satisfied"`
	Partial   int `json:"partial"`
	Gaps      int `json:"gaps"`
}

// ComponentGap holds one language tool's gap report contribution.
//
//fusa:req REQ-FO-STD003
type ComponentGap struct {
	Language string     `json:"language"`
	Tool     string     `json:"tool"`
	Report   *GapReport `json:"report,omitempty"`
	Skipped  string     `json:"skipped,omitempty"`
}

// Aggregate is the cross-language roll-up for one standard.
//
//fusa:req REQ-FO-STD004
type Aggregate struct {
	Standard   string         `json:"standard"`
	Project    string         `json:"project,omitempty"`
	Generated  time.Time      `json:"generated"`
	Components []ComponentGap `json:"components"`
}

// RecomputeSummary derives Summary counts from Objectives.  Call this after
// decoding a GapReport from a tool that uses non-canonical summary key names
// (e.g. "addressed"/"gap" instead of spec §9.3 "satisfied"/"gaps").  It is a
// no-op when the Summary invariant (satisfied + partial + gaps == total) already
// holds.
//
//fusa:req REQ-FO-STD010
func (r *GapReport) RecomputeSummary() {
	if len(r.Objectives) == 0 {
		return
	}
	var sat, par, gap int
	for _, o := range r.Objectives {
		switch o.Status {
		case "satisfied":
			sat++
		case "partial":
			par++
		default:
			gap++ // unknown status maps to gap (fail-safe, per §9.3)
		}
	}
	if sat+par+gap == r.Summary.Total && r.Summary.Satisfied+r.Summary.Partial+r.Summary.Gaps == r.Summary.Total {
		return // already consistent
	}
	r.Summary.Total = sat + par + gap
	r.Summary.Satisfied = sat
	r.Summary.Partial = par
	r.Summary.Gaps = gap
}

// New builds an Aggregate from a slice of ComponentGap entries.
//
//fusa:req REQ-FO-STD004
func New(project, standard string, components []ComponentGap) *Aggregate {
	return &Aggregate{
		Standard:   standard,
		Project:    project,
		Generated:  time.Now().UTC(),
		Components: components,
	}
}

// HasGaps reports whether any reporting component has at least one gap-status
// objective.  Skipped components do not count as gaps here — they are coverage
// holes, not confirmed failures.
//
//fusa:req REQ-FO-STD005
func (a *Aggregate) HasGaps() bool {
	for _, c := range a.Components {
		if c.Report != nil && c.Report.Summary.Gaps > 0 {
			return true
		}
	}
	return false
}

// TotalGaps sums gap-status objectives across all reporting components.
func (a *Aggregate) TotalGaps() int {
	total := 0
	for _, c := range a.Components {
		if c.Report != nil {
			total += c.Report.Summary.Gaps
		}
	}
	return total
}

// TotalSkipped returns the count of components that could not contribute a report.
func (a *Aggregate) TotalSkipped() int {
	n := 0
	for _, c := range a.Components {
		if c.Skipped != "" {
			n++
		}
	}
	return n
}

// Render writes agg to w in the requested format (text or json).
//
//fusa:req REQ-FO-STD006
//fusa:req REQ-FO-STD007
func Render(w io.Writer, agg *Aggregate, format string) error {
	switch format {
	case "json", "":
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(agg)
	case "text":
		return renderText(w, agg)
	default:
		return fmt.Errorf("standards: unsupported format %q", format)
	}
}

// RenderToFile writes agg to path in format, or stdout if path is empty.
func RenderToFile(agg *Aggregate, format, path string) error {
	if path == "" {
		return Render(os.Stdout, agg, format)
	}
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("standards: create %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()
	return Render(f, agg, format)
}

func renderText(w io.Writer, agg *Aggregate) error {
	header := displayName(agg.Standard) + " gap report"
	if agg.Project != "" {
		header += " — " + agg.Project
	}
	fmt.Fprintln(w, header)
	fmt.Fprintf(w, "Generated: %s\n\n", agg.Generated.Format("2006-01-02 15:04:05 UTC"))

	for _, c := range agg.Components {
		fmt.Fprintf(w, "── %s (%s) ──\n", c.Tool, c.Language)
		if c.Skipped != "" {
			fmt.Fprintf(w, "  skipped: %s\n\n", c.Skipped)
			continue
		}
		r := c.Report
		if r == nil {
			fmt.Fprintf(w, "  skipped: no report\n\n")
			continue
		}
		satisfiedPct := 0
		if r.Summary.Total > 0 {
			satisfiedPct = 100 * r.Summary.Satisfied / r.Summary.Total
		}
		fmt.Fprintf(w, "  satisfied: %d/%d (%d%%)  partial: %d  gaps: %d\n",
			r.Summary.Satisfied, r.Summary.Total, satisfiedPct,
			r.Summary.Partial, r.Summary.Gaps)
		if r.Summary.Gaps > 0 {
			for _, obj := range r.Objectives {
				if obj.Status == "gap" {
					fmt.Fprintf(w, "    GAP  %s  %s\n", obj.ID, obj.Title)
				}
			}
		}
		fmt.Fprintln(w)
	}

	if agg.HasGaps() {
		fmt.Fprintf(w, "RESULT: GAP — %d objective(s) with gap status\n", agg.TotalGaps())
	} else if agg.TotalSkipped() == len(agg.Components) {
		fmt.Fprintln(w, "RESULT: SKIP — all components skipped")
	} else {
		fmt.Fprintln(w, "RESULT: PASS")
	}
	return nil
}

// CommandStandard maps a CLI command name to the canonical §2.4.1 standard id.
// Most command names match the standard id directly; exceptions are aliases.
//
//fusa:req REQ-FO-STD008
func CommandStandard(cmd string) string {
	switch strings.ToLower(cmd) {
	case "do178":
		return "do178c"
	default:
		return strings.ToLower(cmd)
	}
}

// DisplayName returns a human-readable label for a canonical standard id.
//
//fusa:req REQ-FO-STD009
func DisplayName(standard string) string {
	return displayName(standard)
}

// displayName is the internal implementation.
func displayName(standard string) string {
	switch standard {
	case "iso26262":
		return "ISO 26262"
	case "iec61508":
		return "IEC 61508"
	case "do178c":
		return "DO-178C"
	case "iso21434":
		return "ISO 21434"
	case "iec62443-4-1":
		return "IEC 62443-4-1"
	case "iec62443-4-2":
		return "IEC 62443-4-2"
	case "iec62443":
		return "IEC 62443"
	case "unece":
		return "UNECE R155/R156"
	default:
		return standard
	}
}
