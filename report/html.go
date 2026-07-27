package report

import (
	"fmt"
	"html/template"
	"io"

	fusaops "github.com/SoundMatt/FuSaOps"
)

// htmlData is the template data envelope carrying the report and render options.
type htmlData struct {
	*AggregateReport
	ShowSuppressed   bool
	ShowFingerprints bool
	Qualify          *QualifyInfo
	Comp             *CompInfo
	MCDC             *MCDCInfo
}

// renderHTML writes a self-contained HTML dashboard for the aggregate report.
// The page has no external dependencies: CSS and the client-side filter logic
// are inlined, matching the zero-dependency ethos of the x-FuSa toolchain.
//
//fusa:req REQ-FO-RPT012
//fusa:req REQ-FO-RPT016
//fusa:req REQ-FO-RPT017
//fusa:req REQ-FO-RPT019
func renderHTML(w io.Writer, r *AggregateReport, opts RenderOptions) error {
	t, err := template.New("dashboard").Funcs(htmlFuncs).Parse(dashboardTemplate)
	if err != nil {
		return fmt.Errorf("report: parse html template: %w", err)
	}
	if err := t.Execute(w, htmlData{r, opts.ShowSuppressed, opts.ShowFingerprints, opts.QualifyInfo, opts.CompInfo, opts.MCDCInfo}); err != nil {
		return fmt.Errorf("report: execute html template: %w", err)
	}
	return nil
}

var htmlFuncs = template.FuncMap{
	"compThresholdLabel": func(dal string, threshold int) string {
		if dal != "" {
			return fmt.Sprintf("%s (≤%d)", dal, threshold)
		}
		if threshold > 0 {
			return fmt.Sprintf("≤%d", threshold)
		}
		return "—"
	},
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
  .nav-links { margin-left:auto; display:flex; gap:12px; font-size:12px; }
  .nav-links a { color:var(--muted); text-decoration:none; }
  .nav-links a:hover { color:var(--txt); }
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
  .filters { display:flex; gap:8px; margin-bottom:14px; flex-wrap:wrap; align-items:center; }
  .filters button { background:var(--card); color:var(--txt); border:1px solid var(--line);
                    border-radius:8px; padding:6px 12px; cursor:pointer; font-size:13px; }
  .filters button.active { border-color:var(--muted); background:#222734; }
  .filters input[type=search] { background:var(--card); color:var(--txt); border:1px solid var(--line);
    border-radius:8px; padding:6px 12px; font-size:13px; outline:none; min-width:220px; }
  .filters input[type=search]:focus { border-color:var(--muted); }
  .search-count { color:var(--muted); font-size:12px; margin-left:auto; }
  table { width:100%; border-collapse:collapse; font-size:13px; }
  th,td { text-align:left; padding:8px 10px; border-bottom:1px solid var(--line); vertical-align:top; }
  th { color:var(--muted); font-weight:600; }
  .pill { padding:2px 8px; border-radius:6px; font-size:11px; font-weight:700; }
  .sev-error { background:rgba(231,76,60,.15); color:var(--fail); }
  .sev-warning { background:rgba(241,196,15,.15); color:var(--warn); }
  .sev-info { background:rgba(138,147,166,.15); color:var(--muted); }
  .loc { color:var(--muted); font-family:ui-monospace,Menlo,monospace; font-size:12px; }
  .fp-chip { display:inline-block; font-family:ui-monospace,Menlo,monospace; font-size:10px;
             color:var(--muted); background:rgba(138,147,166,.08); border:1px solid var(--line);
             border-radius:4px; padding:1px 5px; margin-top:3px; cursor:default; }
  .empty { color:var(--muted); padding:40px; text-align:center; }
  footer { color:var(--muted); font-size:12px; padding:24px 32px; border-top:1px solid var(--line); }
  .comp-section { margin-top:28px; }
  .comp-section h2 { font-size:16px; margin:0 0 12px; font-weight:600; }
  .mcdc-section { margin-top:28px; }
  .mcdc-section h2 { font-size:16px; margin:0 0 12px; font-weight:600; }
</style>
</head>
<body>
<header>
  <h1>FuSaOps</h1>
  <span class="badge {{statusClass .Summary.Status}}">{{.Summary.Status}}</span>
  {{if .Qualify}}
  <div class="qual-section">
    <span class="badge {{if .Qualify.AllPassed}}status-pass{{else}}status-fail{{end}}">
      {{if .Qualify.AllPassed}}qualified{{else}}qual-failing{{end}}
    </span>
    {{if .Qualify.IsIndependent}}
    <span class="badge status-pass" style="margin-left:4px;font-size:11px">independently-qualified</span>
    {{else}}
    <span class="badge" style="margin-left:4px;font-size:11px;background:rgba(138,147,166,.15);color:var(--muted)">self-qualified</span>
    {{end}}
    <span class="sub" style="font-size:11px;margin-left:4px">
      {{.Qualify.Type}}
      · {{.Qualify.PassedCount}}/{{.Qualify.Total}} checks
      {{if .Qualify.RecordUri}}· <a href="{{.Qualify.RecordUri}}" style="color:var(--muted)">certificate</a>{{end}}
      {{if .Qualify.IndependentReviewer}}· reviewer: {{.Qualify.IndependentReviewer}}{{end}}
      {{if .Qualify.AchievableASIL}}· achievable: {{.Qualify.AchievableASIL}}{{end}}
    </span>
  </div>
  {{end}}
  <div class="sub">
    {{if .Project}}{{.Project}} · {{end}}{{.Summary.Total}} findings ·
    {{.Summary.Errors}} errors · {{.Summary.Warnings}} warnings · {{.Summary.Infos}} infos
    <br>{{.Root}} · {{.GeneratedAt.Format "2006-01-02 15:04:05 MST"}}
  </div>
  <nav class="nav-links">
    <a href="/history">History</a>
    {{if .Comp}}<a href="/comp">Complexity</a>{{end}}
    {{if .MCDC}}<a href="/mcdc">MC/DC</a>{{end}}
    <a href="/api/report">JSON</a>
    <a href="/refresh">Refresh</a>
  </nav>
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
        {{if .SuppressedFindings}}<div class="c-info" style="font-size:11px;margin-top:6px">{{len .SuppressedFindings}} suppressed</div>{{end}}
      {{end}}
    </div>
  {{end}}
  </section>

  <div class="filters">
    <button data-sev="all" class="active" onclick="setSev(this,'all')">All</button>
    <button data-sev="ERROR" onclick="setSev(this,'ERROR')">Errors</button>
    <button data-sev="WARNING" onclick="setSev(this,'WARNING')">Warnings</button>
    <button data-sev="INFO" onclick="setSev(this,'INFO')">Infos</button>
    <input type="search" id="search-box" placeholder="Search rule, message, category…" oninput="applyFilters()">
    <span class="search-count" id="search-count"></span>
  </div>

  <table id="findings">
    <thead><tr><th>Severity</th><th>Lang</th><th>Rule</th><th>Category</th><th>Message</th><th>Location</th></tr></thead>
    <tbody>
    {{$showFP := .ShowFingerprints}}
    {{range .Components}}{{$lang := .Language}}{{range .Findings}}
      <tr data-sev="{{.Severity}}">
        <td><span class="pill {{sevClass .Severity}}">{{.Severity}}</span></td>
        <td>{{$lang}}</td>
        <td>{{.RuleID}}</td>
        <td>{{.Category}}</td>
        <td>{{.Message}}{{if .Remediation}}<br><span class="loc">→ {{.Remediation}}</span>{{end}}{{if and $showFP .Fingerprint}}<br><span class="fp-chip" title="fusaops suppress add --fingerprint {{.Fingerprint}} --reason &quot;&quot;">{{.Fingerprint}}</span>{{end}}</td>
        <td class="loc">{{.Location.File}}{{if .Location.Line}}:{{.Location.Line}}{{end}}</td>
      </tr>
    {{end}}{{end}}
    </tbody>
  </table>
  {{if eq .Summary.Total 0}}<div class="empty">No findings. All components passed.</div>{{end}}

  {{if .Comp}}
  <section class="comp-section">
    <h2><a href="/comp" style="color:inherit;text-decoration:none">Cyclomatic Complexity</a>
      <span class="badge {{if gt .Comp.Violations 0}}status-fail{{else}}status-pass{{end}}" style="margin-left:8px;font-size:13px">
        {{if gt .Comp.Violations 0}}{{.Comp.Violations}} violations{{else}}PASS{{end}}
      </span>
      <span class="sub" style="font-size:12px;margin-left:8px">{{.Comp.TotalFunctions}} functions total</span>
    </h2>
    <table>
      <thead><tr><th>Language</th><th>Tool</th><th>Functions</th><th>Violations</th><th>Threshold</th></tr></thead>
      <tbody>
      {{range .Comp.Components}}
        <tr>
          <td>{{.Language}}</td>
          <td>{{.Tool}}</td>
          {{if .Skipped}}
            <td colspan="3" class="c-info">skipped: {{.Skipped}}</td>
          {{else}}
            <td>{{.TotalFunctions}}</td>
            <td>{{if gt .Violations 0}}<span class="c-err"><b>{{.Violations}}</b></span>{{else}}0{{end}}</td>
            <td>{{compThresholdLabel .DAL .Threshold}}</td>
          {{end}}
        </tr>
      {{end}}
      </tbody>
    </table>
  </section>
  {{end}}

  {{if .MCDC}}
  <section class="mcdc-section">
    <h2><a href="/mcdc" style="color:inherit;text-decoration:none">MC/DC Coverage</a>
      <span class="badge {{if .MCDC.GatePassed}}status-pass{{else}}status-fail{{end}}" style="margin-left:8px;font-size:13px">
        {{if .MCDC.GatePassed}}PASS{{else}}FAIL{{end}}
      </span>
      <span class="sub" style="font-size:12px;margin-left:8px">{{.MCDC.CoveredConditions}}/{{.MCDC.TotalConditions}} conditions ({{.MCDC.CoveragePct}}%)</span>
    </h2>
    <table>
      <thead><tr><th>Language</th><th>Tool</th><th>Conditions</th><th>Covered</th><th>Gate</th></tr></thead>
      <tbody>
      {{range .MCDC.Components}}
        <tr>
          <td>{{.Language}}</td>
          <td>{{.Tool}}</td>
          {{if .Skipped}}
            <td colspan="3" class="c-info">skipped: {{.Skipped}}</td>
          {{else}}
            <td>{{.TotalConditions}}</td>
            <td>{{.CoveredConditions}}</td>
            <td>{{if .GatePassed}}<span class="c-info">PASS</span>{{else}}<span class="c-err"><b>FAIL</b></span>{{end}}</td>
          {{end}}
        </tr>
      {{end}}
      </tbody>
    </table>
  </section>
  {{end}}

  {{$showSup := .ShowSuppressed}}
  {{range .Components}}{{if .SuppressedFindings}}
  <details style="margin-top:16px" {{if $showSup}}open{{end}}>
    <summary style="cursor:pointer;color:var(--muted);font-size:13px;padding:8px 0">
      {{.Tool}} ({{.Language}}) — {{len .SuppressedFindings}} suppressed finding(s)
    </summary>
    <table style="opacity:0.6;margin-top:8px">
      <thead><tr><th>Severity</th><th>Rule</th><th>Message</th><th>Location</th></tr></thead>
      <tbody>
      {{range .SuppressedFindings}}
        <tr>
          <td><span class="pill {{sevClass .Severity}}">{{.Severity}}</span></td>
          <td>{{.RuleID}}</td>
          <td>{{.Message}}</td>
          <td class="loc">{{.Location.File}}{{if .Location.Line}}:{{.Location.Line}}{{end}}</td>
        </tr>
      {{end}}
      </tbody>
    </table>
  </details>
  {{end}}{{end}}
</main>
<footer>Generated by FuSaOps v{{.Version}} · multi-language functional safety orchestration</footer>
<script>
var activeSev = 'all';
function setSev(btn, sev) {
  document.querySelectorAll('.filters button').forEach(b=>b.classList.remove('active'));
  btn.classList.add('active');
  activeSev = sev;
  applyFilters();
}
function applyFilters() {
  var q = (document.getElementById('search-box').value||'').toLowerCase();
  var rows = document.querySelectorAll('#findings tbody tr');
  var shown = 0;
  rows.forEach(function(tr) {
    var sevOk = activeSev==='all' || tr.dataset.sev===activeSev;
    var textOk = !q || tr.textContent.toLowerCase().indexOf(q) !== -1;
    var vis = sevOk && textOk;
    tr.style.display = vis ? '' : 'none';
    if (vis) shown++;
  });
  var total = rows.length;
  document.getElementById('search-count').textContent = (shown < total) ? shown+' / '+total+' shown' : '';
}
</script>
</body>
</html>
`

// Version is exposed to the template via the report struct method below.

// Version returns the FuSaOps version for footer rendering in the template.
//
//fusa:req REQ-FO-RPT022
func (r *AggregateReport) Version() string { return fusaops.Version }
