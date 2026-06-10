package trace

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"os"
)

// Render writes the aggregate matrix to w in the requested format.
//
//fusa:req REQ-FO-TRC012
func Render(w io.Writer, a *Aggregate, format string) error {
	switch format {
	case "", "text":
		return renderText(w, a)
	case "json":
		return renderJSON(w, a)
	case "html":
		return renderHTML(w, a)
	default:
		return fmt.Errorf("trace: unsupported format %q", format)
	}
}

// RenderToFile writes the matrix to path in format, or to stdout if path empty.
//
//fusa:req REQ-FO-TRC012
func RenderToFile(a *Aggregate, format, path string) error {
	if path == "" {
		return Render(os.Stdout, a, format)
	}
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("trace: create %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()
	return Render(f, a, format)
}

//fusa:req REQ-FO-TRC013
func renderJSON(w io.Writer, a *Aggregate) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(a); err != nil {
		return fmt.Errorf("trace: json encode: %w", err)
	}
	return nil
}

//fusa:req REQ-FO-TRC014
func renderText(w io.Writer, a *Aggregate) error {
	fmt.Fprintln(w, "FuSaOps Cross-Language Traceability")
	fmt.Fprintf(w, "Generated: %s\n", a.GeneratedAt.Format("2006-01-02 15:04:05 MST"))
	if a.Project != "" {
		fmt.Fprintf(w, "Project:   %s\n", a.Project)
	}
	fmt.Fprintf(w, "Root:      %s\n", a.Root)
	fmt.Fprintf(w, "Status:    %s\n\n", a.Status())

	for _, c := range a.Components {
		fmt.Fprintf(w, "── %s (%s) ──\n", c.Tool, c.Language)
		if c.Skipped != "" {
			fmt.Fprintf(w, "  skipped: %s\n\n", c.Skipped)
			continue
		}
		fmt.Fprintf(w, "  requirements: %d  traced: %d (%d%%)  tested: %d (%d%%)\n",
			c.Coverage.TotalRequirements,
			c.Coverage.TracedRequirements, c.TracedPct(),
			c.Coverage.TestedRequirements, c.TestedPct())
		if c.Qualification != nil {
			fmt.Fprintf(w, "  qualification: %d/%d passed", c.Qualification.Passed, c.Qualification.Total)
			if c.Qualification.Failed > 0 {
				fmt.Fprintf(w, " (%d failed)", c.Qualification.Failed)
			}
			fmt.Fprintln(w)
		}
		fmt.Fprintln(w)
	}

	c := a.Coverage
	fmt.Fprintf(w, "TOTAL: %s — %d requirements across %d component(s): %d traced (%d%%), %d tested (%d%%)\n",
		a.Status(), c.TotalRequirements, len(a.Components),
		c.TracedRequirements, c.TracedPct, c.TestedRequirements, c.TestedPct)
	return nil
}

//fusa:req REQ-FO-TRC015
func renderHTML(w io.Writer, a *Aggregate) error {
	if err := traceTemplate.Execute(w, a); err != nil {
		return fmt.Errorf("trace: html render: %w", err)
	}
	return nil
}

// traceTemplate is a self-contained, dependency-free dashboard for the
// cross-language traceability matrix.
var traceTemplate = template.Must(template.New("trace").Funcs(template.FuncMap{
	"tracedPct": func(c ComponentTrace) int { return c.TracedPct() },
	"testedPct": func(c ComponentTrace) int { return c.TestedPct() },
}).Parse(`<!doctype html>
<html lang="en"><head><meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>FuSaOps — Traceability{{if .Project}}: {{.Project}}{{end}}</title>
<style>
 body{font:15px/1.5 system-ui,sans-serif;margin:0;background:#0f1115;color:#e6e6e6}
 header{padding:1.2rem 1.6rem;background:#171a21;border-bottom:1px solid #272b34}
 h1{margin:0;font-size:1.25rem} .meta{color:#9aa3b2;font-size:.85rem;margin-top:.3rem}
 main{padding:1.6rem;max-width:1000px;margin:0 auto}
 .badge{display:inline-block;padding:.15rem .6rem;border-radius:.5rem;font-weight:600;font-size:.85rem}
 .PASS{background:#16361f;color:#7ee2a0} .GAP{background:#3a2f12;color:#f0c463}
 table{width:100%;border-collapse:collapse;margin-top:1rem;background:#171a21;border-radius:.6rem;overflow:hidden}
 th,td{padding:.55rem .8rem;text-align:left;border-bottom:1px solid #272b34;font-size:.9rem}
 th{background:#1d2129;color:#9aa3b2;font-weight:600} td.num{text-align:right;font-variant-numeric:tabular-nums}
 .skip{color:#9aa3b2;font-style:italic}
 .bar{height:.5rem;background:#272b34;border-radius:.25rem;overflow:hidden;min-width:90px}
 .bar>span{display:block;height:100%;background:#4f8cff}
</style></head><body>
<header>
 <h1>FuSaOps — Cross-Language Traceability{{if .Project}}: {{.Project}}{{end}}</h1>
 <div class="meta">Generated {{.GeneratedAt.Format "2006-01-02 15:04 MST"}} · {{.Root}} ·
  <span class="badge {{.Status}}">{{.Status}}</span></div>
</header>
<main>
 <table>
  <thead><tr><th>Tool</th><th>Language</th><th class="num">Requirements</th>
   <th>Traced</th><th>Tested</th><th class="num">Qualification</th></tr></thead>
  <tbody>
  {{range .Components}}
   <tr>
    <td>{{.Tool}}</td><td>{{.Language}}</td>
    {{if .Skipped}}
     <td colspan="4" class="skip">skipped — {{.Skipped}}</td>
    {{else}}
     <td class="num">{{.Coverage.TotalRequirements}}</td>
     <td><div class="bar"><span style="width:{{tracedPct .}}%"></span></div>{{.Coverage.TracedRequirements}} ({{tracedPct .}}%)</td>
     <td><div class="bar"><span style="width:{{testedPct .}}%"></span></div>{{.Coverage.TestedRequirements}} ({{testedPct .}}%)</td>
     <td class="num">{{if .Qualification}}{{.Qualification.Passed}}/{{.Qualification.Total}}{{else}}—{{end}}</td>
    {{end}}
   </tr>
  {{end}}
  </tbody>
  <tfoot><tr><th>TOTAL</th><th></th>
   <th class="num">{{.Coverage.TotalRequirements}}</th>
   <th>{{.Coverage.TracedRequirements}} ({{.Coverage.TracedPct}}%)</th>
   <th>{{.Coverage.TestedRequirements}} ({{.Coverage.TestedPct}}%)</th><th></th></tr></tfoot>
 </table>
</main></body></html>
`))
