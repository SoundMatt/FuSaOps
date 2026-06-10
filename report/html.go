package report

import (
	"fmt"
	"html/template"
	"io"

	fusaops "github.com/SoundMatt/FuSaOps"
)

// renderHTML writes a self-contained HTML dashboard for the aggregate report.
// The page has no external dependencies: CSS and the client-side filter logic
// are inlined, matching the zero-dependency ethos of the x-FuSa toolchain.
//
//fusa:req REQ-FO-RPT012
func renderHTML(w io.Writer, r *AggregateReport) error {
	t, err := template.New("dashboard").Funcs(htmlFuncs).Parse(dashboardTemplate)
	if err != nil {
		return fmt.Errorf("report: parse html template: %w", err)
	}
	if err := t.Execute(w, r); err != nil {
		return fmt.Errorf("report: execute html template: %w", err)
	}
	return nil
}

var htmlFuncs = template.FuncMap{
	"sevClass": func(s fusaops.Severity) string {
		switch s {
		case fusaops.SeverityError:
			return "sev-error"
		case fusaops.SeverityWarning:
			return "sev-warning"
		default:
			return "sev-info"
		}
	},
	"statusClass": func(status string) string {
		switch status {
		case "FAIL":
			return "status-fail"
		case "WARN":
			return "status-warn"
		default:
			return "status-pass"
		}
	},
}

const dashboardTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>FuSaOps — {{if .Project}}{{.Project}}{{else}}Safety Report{{end}}</title>
<style>
  :root { --bg:#0f1115; --card:#181b22; --muted:#8a93a6; --line:#262b36;
          --pass:#2ecc71; --warn:#f1c40f; --fail:#e74c3c; --txt:#e6e9ef; }
  * { box-sizing:border-box; }
  body { margin:0; font:14px/1.5 -apple-system,Segoe UI,Roboto,Helvetica,Arial,sans-serif;
         background:var(--bg); color:var(--txt); }
  header { padding:24px 32px; border-bottom:1px solid var(--line);
           display:flex; align-items:center; gap:16px; flex-wrap:wrap; }
  h1 { font-size:20px; margin:0; font-weight:600; }
  .sub { color:var(--muted); font-size:13px; }
  .badge { padding:4px 12px; border-radius:999px; font-weight:700; font-size:13px; letter-spacing:.5px; }
  .status-pass { background:rgba(46,204,113,.15); color:var(--pass); }
  .status-warn { background:rgba(241,196,15,.15); color:var(--warn); }
  .status-fail { background:rgba(231,76,60,.15); color:var(--fail); }
  main { padding:24px 32px; max-width:1200px; margin:0 auto; }
  .cards { display:grid; grid-template-columns:repeat(auto-fill,minmax(220px,1fr)); gap:16px; margin-bottom:28px; }
  .card { background:var(--card); border:1px solid var(--line); border-radius:12px; padding:16px; }
  .card h3 { margin:0 0 4px; font-size:15px; }
  .card .tool { color:var(--muted); font-size:12px; margin-bottom:10px; }
  .counts { display:flex; gap:14px; font-size:13px; }
  .counts b { font-weight:700; }
  .c-err { color:var(--fail); } .c-warn { color:var(--warn); } .c-info { color:var(--muted); }
  .filters { display:flex; gap:8px; margin-bottom:14px; flex-wrap:wrap; }
  .filters button { background:var(--card); color:var(--txt); border:1px solid var(--line);
                    border-radius:8px; padding:6px 12px; cursor:pointer; font-size:13px; }
  .filters button.active { border-color:var(--muted); background:#222734; }
  table { width:100%; border-collapse:collapse; font-size:13px; }
  th,td { text-align:left; padding:8px 10px; border-bottom:1px solid var(--line); vertical-align:top; }
  th { color:var(--muted); font-weight:600; }
  .pill { padding:2px 8px; border-radius:6px; font-size:11px; font-weight:700; }
  .sev-error { background:rgba(231,76,60,.15); color:var(--fail); }
  .sev-warning { background:rgba(241,196,15,.15); color:var(--warn); }
  .sev-info { background:rgba(138,147,166,.15); color:var(--muted); }
  .loc { color:var(--muted); font-family:ui-monospace,Menlo,monospace; font-size:12px; }
  .empty { color:var(--muted); padding:40px; text-align:center; }
  footer { color:var(--muted); font-size:12px; padding:24px 32px; border-top:1px solid var(--line); }
</style>
</head>
<body>
<header>
  <h1>FuSaOps</h1>
  <span class="badge {{statusClass .Summary.Status}}">{{.Summary.Status}}</span>
  <div class="sub">
    {{if .Project}}{{.Project}} · {{end}}{{.Summary.Total}} findings ·
    {{.Summary.Errors}} errors · {{.Summary.Warnings}} warnings · {{.Summary.Infos}} infos
    <br>{{.Root}} · {{.GeneratedAt.Format "2006-01-02 15:04:05 MST"}}
  </div>
</header>
<main>
  <section class="cards">
  {{range .Components}}
    <div class="card">
      <h3>{{.Language}} <span class="badge {{statusClass .Summary.Status}}">{{.Summary.Status}}</span></h3>
      <div class="tool">{{.Tool}}{{if not .Available}} · not installed{{end}}</div>
      {{if .Skipped}}
        <div class="c-info">skipped: {{.Skipped}}</div>
      {{else}}
        <div class="counts">
          <span class="c-err"><b>{{.Summary.Errors}}</b> err</span>
          <span class="c-warn"><b>{{.Summary.Warnings}}</b> warn</span>
          <span class="c-info"><b>{{.Summary.Infos}}</b> info</span>
        </div>
      {{end}}
    </div>
  {{end}}
  </section>

  <div class="filters">
    <button data-sev="all" class="active" onclick="filter(this,'all')">All</button>
    <button data-sev="ERROR" onclick="filter(this,'ERROR')">Errors</button>
    <button data-sev="WARNING" onclick="filter(this,'WARNING')">Warnings</button>
    <button data-sev="INFO" onclick="filter(this,'INFO')">Infos</button>
  </div>

  <table id="findings">
    <thead><tr><th>Severity</th><th>Lang</th><th>Rule</th><th>Category</th><th>Message</th><th>Location</th></tr></thead>
    <tbody>
    {{range .Components}}{{$lang := .Language}}{{range .Findings}}
      <tr data-sev="{{.Severity}}">
        <td><span class="pill {{sevClass .Severity}}">{{.Severity}}</span></td>
        <td>{{$lang}}</td>
        <td>{{.RuleID}}</td>
        <td>{{.Category}}</td>
        <td>{{.Message}}{{if .Remediation}}<br><span class="loc">→ {{.Remediation}}</span>{{end}}</td>
        <td class="loc">{{.Location.File}}{{if .Location.Line}}:{{.Location.Line}}{{end}}</td>
      </tr>
    {{end}}{{end}}
    </tbody>
  </table>
  {{if eq .Summary.Total 0}}<div class="empty">No findings. All components passed.</div>{{end}}
</main>
<footer>Generated by FuSaOps v{{.Version}} · multi-language functional safety orchestration</footer>
<script>
function filter(btn, sev){
  document.querySelectorAll('.filters button').forEach(b=>b.classList.remove('active'));
  btn.classList.add('active');
  document.querySelectorAll('#findings tbody tr').forEach(function(tr){
    tr.style.display = (sev==='all'||tr.dataset.sev===sev)?'':'none';
  });
}
</script>
</body>
</html>
`

// Version is exposed to the template via the report struct method below.

// Version returns the FuSaOps version for footer rendering in the template.
func (r *AggregateReport) Version() string { return fusaops.Version }
