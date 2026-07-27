// Package standards implements the §9.3 gap-report roll-up for the x-FuSa
// toolchain.  Each language tool can emit a <standard>-gap-report.json; this
// package aggregates them into one cross-language compliance view.
package standards

import (
	"encoding/json"
	"fmt"
	"html/template"
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
//
//fusa:req REQ-FO-STD013
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
//
//fusa:req REQ-FO-STD003
func (a *Aggregate) TotalSkipped() int {
	n := 0
	for _, c := range a.Components {
		if c.Skipped != "" {
			n++
		}
	}
	return n
}

// Render writes agg to w in the requested format (text, json, html, or markdown).
//
//fusa:req REQ-FO-STD006
//fusa:req REQ-FO-STD007
//fusa:req REQ-FO-STD011
//fusa:req REQ-FO-STD012
func Render(w io.Writer, agg *Aggregate, format string) error {
	switch format {
	case "json", "":
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(agg)
	case "text":
		return renderText(w, agg)
	case "html":
		if err := standardsTemplate.Execute(w, agg); err != nil {
			return fmt.Errorf("standards: html render: %w", err)
		}
		return nil
	case "markdown", "md":
		return renderMarkdown(w, agg)
	default:
		return fmt.Errorf("standards: unsupported format %q", format)
	}
}

// renderMarkdown writes a GFM markdown standards gap report to w.
//
//fusa:req REQ-FO-STD012
func renderMarkdown(w io.Writer, agg *Aggregate) error {
	name := displayName(agg.Standard)
	badge := "🟢"
	if agg.HasGaps() {
		badge = "🔴"
	}
	status := "PASS"
	if agg.HasGaps() {
		status = "GAP"
	}
	fmt.Fprintf(w, "# FuSaOps — %s Gap Report\n\n", name)
	if agg.Project != "" {
		fmt.Fprintf(w, "**Project:** %s\n\n", agg.Project)
	}
	fmt.Fprintf(w, "%s **%s** · Generated %s\n\n", badge, status, agg.Generated.Format("2006-01-02 15:04 MST"))
	for _, c := range agg.Components {
		fmt.Fprintf(w, "## %s (%s)\n\n", c.Tool, c.Language)
		if c.Skipped != "" {
			fmt.Fprintf(w, "_Skipped — %s_\n\n", c.Skipped)
			continue
		}
		if c.Report == nil {
			fmt.Fprintf(w, "_No report available._\n\n")
			continue
		}
		r := c.Report
		fmt.Fprintf(w, "%s %s · %d satisfied · %d partial · %d gap(s)\n\n",
			r.Standard, r.ToolVersion, r.Summary.Satisfied, r.Summary.Partial, r.Summary.Gaps)
		fmt.Fprintln(w, "| ID | Status | Title / Clause | Evidence |")
		fmt.Fprintln(w, "|---|---|---|---|")
		for _, o := range r.Objectives {
			sym := "✅"
			switch o.Status {
			case "partial":
				sym = "🟡"
			case "gap":
				sym = "❌"
			}
			title := strings.ReplaceAll(o.Title, "|", "\\|")
			if o.Clause != "" {
				title += " (" + o.Clause + ")"
			}
			ev := strings.Join(o.Evidence, ", ")
			fmt.Fprintf(w, "| %s | %s | %s | %s |\n", o.ID, sym, title, ev)
		}
		fmt.Fprintln(w)
	}
	return nil
}

// standardsTemplate is a self-contained HTML standards gap-report dashboard.
//
//fusa:req REQ-FO-STD011
var standardsTemplate = template.Must(template.New("standards").Funcs(template.FuncMap{
	"displayName": displayName,
}).Parse(`<!doctype html>
<html lang="en"><head><meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>FuSaOps — {{displayName .Standard}} Gap Report{{if .Project}}: {{.Project}}{{end}}</title>
<style>
 body{font:15px/1.5 system-ui,sans-serif;margin:0;background:#0f1115;color:#e6e6e6}
 header{padding:1.2rem 1.6rem;background:#171a21;border-bottom:1px solid #272b34}
 h1{margin:0;font-size:1.25rem} .meta{color:#9aa3b2;font-size:.85rem;margin-top:.3rem}
 main{padding:1.6rem;max-width:1000px;margin:0 auto}
 h2{font-size:1rem;color:#9aa3b2;margin:1.4rem 0 .4rem}
 table{width:100%;border-collapse:collapse;background:#171a21;border-radius:.6rem;overflow:hidden;margin-top:.4rem}
 th,td{padding:.55rem .8rem;text-align:left;border-bottom:1px solid #272b34;font-size:.9rem}
 th{background:#1d2129;color:#9aa3b2;font-weight:600}
 .sat{color:#7ee2a0} .par{color:#f0c463} .gap{color:#f07070} .skip{color:#9aa3b2;font-style:italic}
 .ev{color:#9aa3b2;font-size:.85rem}
</style></head><body>
<header>
 <h1>FuSaOps — {{displayName .Standard}} Gap Report{{if .Project}}: {{.Project}}{{end}}</h1>
 <div class="meta">Generated {{.Generated.Format "2006-01-02 15:04 MST"}} · {{len .Components}} component(s)</div>
</header>
<main>
 {{range .Components}}
  <h2>{{.Tool}} ({{.Language}})</h2>
  {{if .Skipped}}
   <p class="skip">Skipped — {{.Skipped}}</p>
  {{else if .Report}}
   <p>{{.Report.Standard}} · {{.Report.ToolVersion}} ·
    <span class="sat">{{.Report.Summary.Satisfied}} satisfied</span> ·
    <span class="par">{{.Report.Summary.Partial}} partial</span> ·
    <span class="gap">{{.Report.Summary.Gaps}} gap(s)</span>
   </p>
   <table>
    <thead><tr><th>ID</th><th>Status</th><th>Title / Clause</th><th>Evidence</th></tr></thead>
    <tbody>
    {{range .Report.Objectives}}
     <tr>
      <td>{{.ID}}</td>
      <td>{{if eq .Status "satisfied"}}<span class="sat">satisfied</span>{{else if eq .Status "partial"}}<span class="par">partial</span>{{else}}<span class="gap">gap</span>{{end}}</td>
      <td>{{if .Title}}{{.Title}}{{end}}{{if .Clause}} <span class="ev">({{.Clause}})</span>{{end}}</td>
      <td class="ev">{{range .Evidence}}{{.}} {{end}}</td>
     </tr>
    {{end}}
    </tbody>
   </table>
  {{else}}
   <p class="skip">No report available.</p>
  {{end}}
 {{end}}
</main></body></html>
`))

// RenderToFile writes agg to path in format, or stdout if path is empty.
//
//fusa:req REQ-FO-STD007
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
