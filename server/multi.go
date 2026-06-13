package server

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	fusaops "github.com/SoundMatt/FuSaOps"
	"github.com/SoundMatt/FuSaOps/config"
	"github.com/SoundMatt/FuSaOps/diff"
	"github.com/SoundMatt/FuSaOps/orchestrator"
	"github.com/SoundMatt/FuSaOps/report"
)

// ProjectConfig is one entry in a multi-project serve configuration.
//
//fusa:req REQ-FO-MPJ001
//fusa:req REQ-FO-MPJ005
//fusa:req REQ-FO-MPJ006
type ProjectConfig struct {
	Name        string `json:"name"`
	Dir         string `json:"dir"`
	Adapter     string `json:"adapter,omitempty"`
	Suppression string `json:"suppression,omitempty"`
}

// ProjectsConfig is the JSON file format for fusaops serve --projects.
//
//fusa:req REQ-FO-MPJ001
type ProjectsConfig struct {
	Projects []ProjectConfig `json:"projects"`
}

// projectEntry holds one project's live state inside a MultiServer.
//
//fusa:req REQ-FO-MPJ002
type projectEntry struct {
	name string
	dir  string
	opts orchestrator.Options

	mu     sync.RWMutex
	cached *report.AggregateReport
	err    error
}

// MultiServer serves a combined dashboard for multiple project roots.
//
//fusa:req REQ-FO-MPJ001
type MultiServer struct {
	runner   *orchestrator.Runner
	projects []*projectEntry

	authUser        string
	authPass        string
	authROUser      string
	authROPass      string
	auditDir        string
	refreshInterval time.Duration // zero = no scheduled refresh
	baselineFile    string        // empty = no baseline configured
}

// NewMulti returns a MultiServer from a ProjectsConfig.
//
//fusa:req REQ-FO-MPJ001
//fusa:req REQ-FO-MPJ005
//fusa:req REQ-FO-MPJ006
func NewMulti(cfg ProjectsConfig, runner *orchestrator.Runner) *MultiServer {
	ms := &MultiServer{runner: runner}
	for _, p := range cfg.Projects {
		opts := orchestrator.Options{Project: p.Name}
		if p.Adapter != "" {
			opts.Only = []string{p.Adapter}
		}
		if p.Suppression != "" {
			opts.SuppressFile = p.Suppression
		}
		// Auto-load .fusaops.json from the project directory if present.
		if cfgData, err := config.Load(filepath.Join(p.Dir, config.ConfigFile)); err == nil && cfgData != nil {
			if cfgData.Project.Name != "" {
				opts.Project = cfgData.Project.Name
			}
			if len(cfgData.Scan.Adapters) > 0 && p.Adapter == "" {
				opts.Only = cfgData.Scan.Adapters
			}
		}
		ms.projects = append(ms.projects, &projectEntry{name: p.Name, dir: p.Dir, opts: opts})
	}
	return ms
}

// ValidateProjectDirs checks that every project directory exists and returns a
// slice of errors for any that are missing.
//
//fusa:req REQ-FO-MPJ007
func (ms *MultiServer) ValidateProjectDirs() []error {
	var errs []error
	for _, p := range ms.projects {
		if _, err := os.Stat(p.dir); err != nil {
			errs = append(errs, fmt.Errorf("project %q: directory %q not found", p.name, p.dir))
		}
	}
	return errs
}

// WithAuth enables HTTP Basic Auth (rw) on all routes of the MultiServer.
//
//fusa:req REQ-FO-AUTH001
func (ms *MultiServer) WithAuth(user, pass string) *MultiServer {
	ms.authUser, ms.authPass = user, pass
	return ms
}

// WithAuthRO enables read-only credentials on the MultiServer.
//
//fusa:req REQ-FO-RBAC001
func (ms *MultiServer) WithAuthRO(user, pass string) *MultiServer {
	ms.authROUser, ms.authROPass = user, pass
	return ms
}

// WithAuditLog enables request audit logging on the MultiServer.
//
//fusa:req REQ-FO-AUDIT001
func (ms *MultiServer) WithAuditLog(dir string) *MultiServer {
	ms.auditDir = dir
	return ms
}

// WithBaseline sets the path to a baseline JSON file used by /api/v1/diff and
// saved by POST /api/v1/baseline on the MultiServer.
//
//fusa:req REQ-FO-SRV007
func (ms *MultiServer) WithBaseline(path string) *MultiServer {
	ms.baselineFile = path
	return ms
}

// WithRefreshInterval enables automatic background rescans on the MultiServer.
//
//fusa:req REQ-FO-SCHD001
func (ms *MultiServer) WithRefreshInterval(d time.Duration) *MultiServer {
	ms.refreshInterval = d
	return ms
}

// compute runs all project scans in parallel.
//
//fusa:req REQ-FO-MPJ002
func (ms *MultiServer) compute(ctx context.Context) {
	var wg sync.WaitGroup
	for _, p := range ms.projects {
		wg.Add(1)
		go func() {
			defer wg.Done()
			rep, err := ms.runner.Run(ctx, p.dir, p.opts)
			p.mu.Lock()
			p.cached, p.err = rep, err
			p.mu.Unlock()
		}()
	}
	wg.Wait()
}

// Handler returns the HTTP routes for the multi-project dashboard.
//
//fusa:req REQ-FO-MPJ003
//fusa:req REQ-FO-SRV006
//fusa:req REQ-FO-SRV007
//fusa:req REQ-FO-SRV008
func (ms *MultiServer) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", ms.handleOverview)
	mux.HandleFunc("/refresh", ms.handleRefreshAll)
	mux.HandleFunc("/healthz", ms.handleHealth)
	mux.HandleFunc("/api/projects", ms.handleAPIProjects)
	mux.HandleFunc("/api/v1/export", ms.handleExport)
	mux.HandleFunc("/api/v1/diff", ms.handleAPIDiff)
	mux.HandleFunc("/api/v1/baseline", ms.handleAPIBaseline)
	mux.HandleFunc("/badge/status.svg", ms.handleBadge)
	mux.HandleFunc("/metrics", ms.handleMetrics)
	for _, p := range ms.projects {
		prefix := "/p/" + p.name
		mux.HandleFunc(prefix, ms.makeProjectHandler(p))
		mux.HandleFunc(prefix+"/", ms.makeProjectHandler(p))
		mux.HandleFunc("/badge/"+p.name+"/status.svg", ms.makeProjectBadgeHandler(p))
	}
	if ms.authUser != "" || ms.authROUser != "" {
		return ms.authWrap(mux)
	}
	return mux
}

// authWrap wraps h with Basic Auth + role gating (re-uses Server's logic inline).
//
//fusa:req REQ-FO-AUTH001
//fusa:req REQ-FO-RBAC001
//fusa:req REQ-FO-RBAC002
func (ms *MultiServer) authWrap(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, p, ok := r.BasicAuth()
		role := ""
		if ok {
			if ms.authUser != "" && u == ms.authUser && p == ms.authPass {
				role = "rw"
			} else if ms.authROUser != "" && u == ms.authROUser && p == ms.authROPass {
				role = "ro"
			}
		}
		if role == "" {
			w.Header().Set("WWW-Authenticate", `Basic realm="fusaops"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		if role == "ro" && mutatingPaths[r.URL.Path] {
			http.Error(w, "Forbidden: read-only credentials", http.StatusForbidden)
			return
		}
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		h.ServeHTTP(rec, r)
		if ms.auditDir != "" {
			_ = appendAudit(ms.auditDir, auditEntry{
				Timestamp: time.Now().UTC(),
				Method:    r.Method,
				Path:      r.URL.Path,
				User:      u,
				Status:    rec.status,
			})
		}
	})
}

// handleOverview renders the multi-project status overview page.
//
//fusa:req REQ-FO-MPJ003
func (ms *MultiServer) handleOverview(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>FuSaOps — Projects</title>
<style>
*{box-sizing:border-box;margin:0;padding:0}
body{font-family:system-ui,sans-serif;background:#f4f6f9;color:#1a1a2e;padding:1.5rem}
h1{font-size:1.4rem;margin-bottom:1rem}
a{color:#4361ee;text-decoration:none}a:hover{text-decoration:underline}
.grid{display:grid;grid-template-columns:repeat(auto-fill,minmax(260px,1fr));gap:1rem}
.card{background:#fff;border-radius:8px;box-shadow:0 1px 4px rgba(0,0,0,.08);padding:1rem}
.card h2{font-size:1rem;margin-bottom:.4rem}
.badge{display:inline-block;padding:.15rem .5rem;border-radius:4px;font-weight:600;font-size:.78rem}
.pass{background:#d1fae5;color:#065f46}.warn{background:#fef3c7;color:#92400e}.fail{background:#fee2e2;color:#7f1d1d}.pending{background:#e2e8f0;color:#4a5568}
.counts{font-size:.82rem;color:#718096;margin-top:.3rem}
.link{margin-top:.5rem;font-size:.82rem}
.footer{margin-top:1rem;font-size:.8rem;color:#a0aec0}
</style>
</head>
<body>
<h1>FuSaOps — Projects <a href="/refresh" style="font-size:.8rem;font-weight:400">Refresh all</a></h1>
<div class="grid">
`)
	badge := func(status string) string {
		switch status {
		case "FAIL":
			return `<span class="badge fail">FAIL</span>`
		case "WARN":
			return `<span class="badge warn">WARN</span>`
		case "PASS":
			return `<span class="badge pass">PASS</span>`
		default:
			return `<span class="badge pending">PENDING</span>`
		}
	}
	for _, p := range ms.projects {
		p.mu.RLock()
		rep, pErr := p.cached, p.err
		p.mu.RUnlock()

		var st, counts string
		if pErr != nil {
			st = badge("FAIL")
			counts = html.EscapeString("error: " + pErr.Error())
		} else if rep == nil {
			st = badge("")
			counts = "computing…"
		} else {
			st = badge(rep.Summary.Status())
			counts = fmt.Sprintf("%d errors · %d warnings · %d total",
				rep.Summary.Errors, rep.Summary.Warnings, rep.Summary.Total)
		}
		fmt.Fprintf(w,
			`<div class="card"><h2>%s %s</h2><p class="counts">%s</p><p class="link"><a href="/p/%s">Details →</a></p></div>`,
			html.EscapeString(p.name), st, counts, html.EscapeString(p.name))
	}
	fmt.Fprint(w, `</div>
<p class="footer"><a href="/api/projects">JSON</a></p>
</body></html>`)
}

// handleAPIProjects serves all projects' status as JSON.
//
//fusa:req REQ-FO-MPJ003
func (ms *MultiServer) handleAPIProjects(w http.ResponseWriter, _ *http.Request) {
	type projectStatus struct {
		Name     string `json:"name"`
		Dir      string `json:"dir"`
		Status   string `json:"status"`
		Total    int    `json:"total"`
		Errors   int    `json:"errors"`
		Warnings int    `json:"warnings"`
		Error    string `json:"error,omitempty"`
	}
	var out []projectStatus
	for _, p := range ms.projects {
		p.mu.RLock()
		rep, pErr := p.cached, p.err
		p.mu.RUnlock()
		ps := projectStatus{Name: p.name, Dir: p.dir}
		if pErr != nil {
			ps.Status = "FAIL"
			ps.Error = pErr.Error()
		} else if rep == nil {
			ps.Status = "PENDING"
		} else {
			ps.Status = rep.Summary.Status()
			ps.Total = rep.Summary.Total
			ps.Errors = rep.Summary.Errors
			ps.Warnings = rep.Summary.Warnings
		}
		out = append(out, ps)
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(out)
}

// makeProjectHandler returns a handler that renders one project's detail page.
//
//fusa:req REQ-FO-MPJ004
func (ms *MultiServer) makeProjectHandler(p *projectEntry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p.mu.RLock()
		rep, pErr := p.cached, p.err
		p.mu.RUnlock()
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if pErr != nil {
			http.Error(w, "scan error: "+pErr.Error(), http.StatusInternalServerError)
			return
		}
		if rep == nil {
			http.Error(w, "no report yet", http.StatusServiceUnavailable)
			return
		}
		fmt.Fprintf(w, `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>FuSaOps — %s</title>
<style>
*{box-sizing:border-box;margin:0;padding:0}
body{font-family:system-ui,sans-serif;background:#f4f6f9;color:#1a1a2e;padding:1.5rem}
h1{font-size:1.4rem;margin-bottom:.5rem}
a{color:#4361ee;text-decoration:none}a:hover{text-decoration:underline}
.sub{font-size:.85rem;color:#718096;margin-bottom:1rem}
.pass{display:inline-block;padding:.15rem .5rem;border-radius:4px;background:#d1fae5;color:#065f46;font-weight:600;font-size:.78rem}
.warn{display:inline-block;padding:.15rem .5rem;border-radius:4px;background:#fef3c7;color:#92400e;font-weight:600;font-size:.78rem}
.fail{display:inline-block;padding:.15rem .5rem;border-radius:4px;background:#fee2e2;color:#7f1d1d;font-weight:600;font-size:.78rem}
table{width:100%%;border-collapse:collapse;background:#fff;border-radius:8px;overflow:hidden;box-shadow:0 1px 4px rgba(0,0,0,.08)}
th,td{padding:.6rem .9rem;text-align:left;border-bottom:1px solid #edf2f7;font-size:.85rem}
th{background:#f8fafc;font-weight:600;color:#4a5568}
tr:last-child td{border-bottom:none}
.err{color:#c53030;font-weight:600}.wc{color:#d97706}
</style>
</head>
<body>
<h1>%s &nbsp;<a href="/" style="font-size:.8rem;font-weight:400">← projects</a></h1>
`, html.EscapeString(p.name), html.EscapeString(p.name))

		badge := ""
		switch rep.Summary.Status() {
		case "PASS":
			badge = `<span class="pass">PASS</span>`
		case "WARN":
			badge = `<span class="warn">WARN</span>`
		default:
			badge = `<span class="fail">FAIL</span>`
		}
		fmt.Fprintf(w, `<p class="sub">%s · %d errors · %d warnings · %d total</p>`,
			badge, rep.Summary.Errors, rep.Summary.Warnings, rep.Summary.Total)

		if rep.Summary.Total == 0 {
			fmt.Fprint(w, `<p style="margin-top:1rem;color:#065f46">No findings — all checks passed.</p>`)
		} else {
			fmt.Fprint(w, `<table>
<thead><tr><th>Rule</th><th>Severity</th><th>Language</th><th>File</th><th>Message</th></tr></thead><tbody>`)
			for _, c := range rep.Components {
				for _, f := range c.Findings {
					cls := "wc"
					if strings.ToUpper(string(f.Severity)) == "ERROR" {
						cls = "err"
					}
					fmt.Fprintf(w,
						`<tr><td>%s</td><td class="%s">%s</td><td>%s</td><td>%s</td><td>%s</td></tr>`,
						html.EscapeString(f.RuleID),
						cls, html.EscapeString(string(f.Severity)),
						html.EscapeString(string(f.Language)),
						html.EscapeString(f.Location.File),
						html.EscapeString(f.Message))
				}
			}
			fmt.Fprint(w, `</tbody></table>`)
		}
		fmt.Fprint(w, `</body></html>`)
	}
}

// handleBadge renders the overall fleet status badge.
//
//fusa:req REQ-FO-BADGE001
func (ms *MultiServer) handleBadge(w http.ResponseWriter, _ *http.Request) {
	status, color := "pending", "#9f9f9f"
	for _, p := range ms.projects {
		p.mu.RLock()
		rep, pErr := p.cached, p.err
		p.mu.RUnlock()
		if pErr != nil || rep == nil {
			continue
		}
		switch rep.Summary.Status() {
		case "FAIL":
			status, color = "fail", "#e05d44"
		case "WARN":
			if status != "fail" {
				status, color = "warn", "#dfb317"
			}
		case "PASS":
			if status == "pending" {
				status, color = "pass", "#4c1"
			}
		}
	}
	w.Header().Set("Content-Type", "image/svg+xml")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	fmt.Fprint(w, svgBadge("fusaops", status, color))
}

// makeProjectBadgeHandler returns a badge handler for one project.
//
//fusa:req REQ-FO-BADGE002
func (ms *MultiServer) makeProjectBadgeHandler(p *projectEntry) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		p.mu.RLock()
		rep, pErr := p.cached, p.err
		p.mu.RUnlock()
		status, color := "pending", "#9f9f9f"
		if pErr != nil {
			status, color = "error", "#e05d44"
		} else if rep != nil {
			switch rep.Summary.Status() {
			case "PASS":
				status, color = "pass", "#4c1"
			case "WARN":
				status, color = "warn", "#dfb317"
			default:
				status, color = "fail", "#e05d44"
			}
		}
		w.Header().Set("Content-Type", "image/svg+xml")
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		fmt.Fprint(w, svgBadge(p.name, status, color))
	}
}

// handleExport serves the merged fleet report in the requested format as a download.
// All project components are combined into a single AggregateReport.
//
//fusa:req REQ-FO-SRV006
func (ms *MultiServer) handleExport(w http.ResponseWriter, r *http.Request) {
	var allComponents []report.Component
	for _, p := range ms.projects {
		p.mu.RLock()
		rep, pErr := p.cached, p.err
		p.mu.RUnlock()
		if pErr != nil || rep == nil {
			continue
		}
		allComponents = append(allComponents, rep.Components...)
	}
	if len(allComponents) == 0 {
		http.Error(w, "no report available yet", http.StatusServiceUnavailable)
		return
	}
	combined := report.New("", "fleet", allComponents)
	format := r.URL.Query().Get("format")
	if format == "" {
		format = "json"
	}
	ct, ext := exportMIME(format)
	w.Header().Set("Content-Type", ct)
	w.Header().Set("Content-Disposition", `attachment; filename="fusaops-report.`+ext+`"`)
	if err := report.Render(w, combined, format); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// handleAPIDiff compares the merged fleet report (or a single project when
// ?project=name is set) against a baseline.
//
//fusa:req REQ-FO-SRV007
//fusa:req REQ-FO-SRV009
func (ms *MultiServer) handleAPIDiff(w http.ResponseWriter, r *http.Request) {
	projectFilter := r.URL.Query().Get("project")
	var allComponents []report.Component
	for _, p := range ms.projects {
		if projectFilter != "" && p.name != projectFilter {
			continue
		}
		p.mu.RLock()
		rep, pErr := p.cached, p.err
		p.mu.RUnlock()
		if pErr != nil || rep == nil {
			continue
		}
		allComponents = append(allComponents, rep.Components...)
	}
	if len(allComponents) == 0 {
		http.Error(w, "no report available yet", http.StatusServiceUnavailable)
		return
	}
	baselinePath := r.URL.Query().Get("baseline")
	if baselinePath == "" {
		baselinePath = ms.baselineFile
	}
	if baselinePath == "" {
		http.Error(w, "baseline path required: set ?baseline= or configure --baseline", http.StatusBadRequest)
		return
	}
	bl, err := diff.LoadBaseline(baselinePath)
	if err != nil {
		http.Error(w, "load baseline: "+err.Error(), http.StatusBadRequest)
		return
	}
	var current []fusaops.Finding
	for _, c := range allComponents {
		current = append(current, c.Findings...)
	}
	result := diff.Compare(bl, current)
	format := r.URL.Query().Get("format")
	if format == "" {
		format = "json"
	}
	strict := r.URL.Query().Get("strict") == "true"
	if strict && result.HasNewErrors() {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusConflict)
		_ = diff.Render(w, result, format, strict)
		return
	}
	if format == "json" {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
	} else {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	}
	_ = diff.Render(w, result, format, strict)
}

// handleAPIBaseline saves the merged fleet findings as a baseline file.
//
//fusa:req REQ-FO-SRV008
func (ms *MultiServer) handleAPIBaseline(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}
	if ms.baselineFile == "" {
		http.Error(w, "no baseline path configured: use --baseline", http.StatusBadRequest)
		return
	}
	var findings []fusaops.Finding
	for _, p := range ms.projects {
		p.mu.RLock()
		rep, pErr := p.cached, p.err
		p.mu.RUnlock()
		if pErr != nil || rep == nil {
			continue
		}
		for _, c := range rep.Components {
			findings = append(findings, c.Findings...)
		}
	}
	if findings == nil {
		http.Error(w, "no report available yet", http.StatusServiceUnavailable)
		return
	}
	if err := diff.SaveBaseline(ms.baselineFile, findings); err != nil {
		http.Error(w, "save baseline: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	fmt.Fprintf(w, `{"saved":%q,"findings":%d}`+"\n", ms.baselineFile, len(findings))
}

// handleMetrics serves an OpenMetrics exposition with per-project finding counts.
//
//fusa:req REQ-FO-MTR001
//fusa:req REQ-FO-MTR002
func (ms *MultiServer) handleMetrics(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	var b strings.Builder
	b.WriteString("# HELP fusaops_findings_total Total findings by project and severity\n")
	b.WriteString("# TYPE fusaops_findings_total gauge\n")
	for _, p := range ms.projects {
		p.mu.RLock()
		rep, pErr := p.cached, p.err
		p.mu.RUnlock()
		text := buildMetrics(rep, pErr, p.name)
		// Append the findings lines; skip the repeated HELP/TYPE headers.
		for _, line := range strings.Split(text, "\n") {
			if strings.HasPrefix(line, "fusaops_findings_total{") {
				b.WriteString(line)
				b.WriteByte('\n')
			}
		}
	}
	b.WriteString("# HELP fusaops_status Aggregate project status: 1=PASS 2=WARN 3=FAIL 0=pending/error\n")
	b.WriteString("# TYPE fusaops_status gauge\n")
	for _, p := range ms.projects {
		p.mu.RLock()
		rep, pErr := p.cached, p.err
		p.mu.RUnlock()
		text := buildMetrics(rep, pErr, p.name)
		for _, line := range strings.Split(text, "\n") {
			if strings.HasPrefix(line, "fusaops_status{") {
				b.WriteString(line)
				b.WriteByte('\n')
			}
		}
	}
	b.WriteString("# EOF\n")
	fmt.Fprint(w, b.String())
}

// handleRefreshAll re-scans all projects and redirects to the overview.
//
//fusa:req REQ-FO-MPJ002
func (ms *MultiServer) handleRefreshAll(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Minute)
	defer cancel()
	ms.compute(ctx)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// handleHealth reports liveness for the MultiServer.
func (ms *MultiServer) handleHealth(w http.ResponseWriter, _ *http.Request) {
	fmt.Fprintln(w, "ok")
}

// ListenAndServe computes all projects then serves the dashboard on addr.
//
//fusa:req REQ-FO-MPJ002
func (ms *MultiServer) ListenAndServe(addr string) error {
	ln, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", addr)
	if err != nil {
		return err
	}
	return ms.Serve(ln)
}

// Serve computes all projects then serves on ln. Closing ln stops the server.
//
//fusa:req REQ-FO-MPJ002
//fusa:req REQ-FO-SCHD001
func (ms *MultiServer) Serve(ln net.Listener) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	ms.compute(ctx)
	cancel()
	if ms.refreshInterval > 0 {
		go ms.runScheduler()
	}
	srv := &http.Server{
		Handler:           ms.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}
	return srv.Serve(ln)
}

// runScheduler fires periodic rescans on ms.refreshInterval until the process exits.
//
//fusa:req REQ-FO-SCHD001
func (ms *MultiServer) runScheduler() {
	ticker := time.NewTicker(ms.refreshInterval)
	defer ticker.Stop()
	for range ticker.C {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		ms.compute(ctx)
		cancel()
	}
}
