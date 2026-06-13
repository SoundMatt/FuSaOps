// Package server provides the FuSaOps web reporting UI.
//
// It serves a self-contained dashboard rendered from the aggregated
// multi-language report, a JSON API, and a run-history trend page.
// The report is computed by the orchestrator on startup and on demand
// via /refresh, and cached between requests so the dashboard loads instantly.
// Each successful compute is persisted to .fusaops-history.jsonl when a
// history directory has been configured via WithHistoryDir.
package server

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/SoundMatt/FuSaOps/history"
	"github.com/SoundMatt/FuSaOps/orchestrator"
	"github.com/SoundMatt/FuSaOps/report"
)

// Server serves the FuSaOps dashboard and JSON API.
//
//fusa:req REQ-FO-SRV001
type Server struct {
	root    string
	project string
	runner  *orchestrator.Runner
	opts    orchestrator.Options
	histDir string // empty = history persistence disabled

	mu     sync.RWMutex
	cached *report.AggregateReport
	err    error
}

// New returns a Server that scans root using the given runner and options.
//
//fusa:req REQ-FO-SRV002
func New(root string, runner *orchestrator.Runner, opts orchestrator.Options) *Server {
	return &Server{root: root, project: opts.Project, runner: runner, opts: opts}
}

// WithHistoryDir enables run-history persistence: each successful compute
// appends a Snapshot to .fusaops-history.jsonl in dir. dir is typically the
// project root. Passing an empty string disables persistence.
//
//fusa:req REQ-FO-HST003
func (s *Server) WithHistoryDir(dir string) *Server {
	s.histDir = dir
	return s
}

// compute runs the orchestrator, caches the result, and persists a snapshot.
//
//fusa:req REQ-FO-SRV003
func (s *Server) compute(ctx context.Context) {
	rep, err := s.runner.Run(ctx, s.root, s.opts)
	s.mu.Lock()
	s.cached, s.err = rep, err
	s.mu.Unlock()
	if err == nil && rep != nil && s.histDir != "" {
		snap := history.FromReport(rep)
		_ = history.Store(s.histDir, snap)
	}
}

// Handler returns the HTTP routes for the dashboard and API.
//
//fusa:req REQ-FO-SRV004
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleIndex)
	mux.HandleFunc("/api/report", s.handleAPIReport)
	mux.HandleFunc("/api/history", s.handleAPIHistory)
	mux.HandleFunc("/history", s.handleHistory)
	mux.HandleFunc("/refresh", s.handleRefresh)
	mux.HandleFunc("/healthz", s.handleHealth)
	return mux
}

// handleIndex renders the HTML dashboard from the cached report.
func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	s.mu.RLock()
	rep, cErr := s.cached, s.err
	s.mu.RUnlock()
	if cErr != nil {
		http.Error(w, "scan failed: "+cErr.Error(), http.StatusInternalServerError)
		return
	}
	if rep == nil {
		http.Error(w, "no report available yet", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := report.Render(w, rep, "html"); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// handleAPIReport serves the cached report as JSON.
func (s *Server) handleAPIReport(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	rep, cErr := s.cached, s.err
	s.mu.RUnlock()
	if cErr != nil {
		http.Error(w, cErr.Error(), http.StatusInternalServerError)
		return
	}
	if rep == nil {
		http.Error(w, "no report available yet", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if err := report.Render(w, rep, "json"); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// handleRefresh recomputes the report then redirects back to the dashboard.
func (s *Server) handleRefresh(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Minute)
	defer cancel()
	s.compute(ctx)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// handleHealth reports liveness.
func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	fmt.Fprintln(w, "ok")
}

// handleAPIHistory serves the run-history snapshots as JSON.
//
//fusa:req REQ-FO-HST004
func (s *Server) handleAPIHistory(w http.ResponseWriter, _ *http.Request) {
	if s.histDir == "" {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_, _ = w.Write([]byte("[]\n"))
		return
	}
	snaps, err := history.Load(s.histDir, 0)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(snaps)
}

// handleHistory renders the HTML run-history trend page.
//
//fusa:req REQ-FO-HST004
func (s *Server) handleHistory(w http.ResponseWriter, _ *http.Request) {
	var snaps []history.Snapshot
	if s.histDir != "" {
		var err error
		snaps, err = history.Load(s.histDir, 30)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>FuSaOps — Run History</title>
<style>
*{box-sizing:border-box;margin:0;padding:0}
body{font-family:system-ui,sans-serif;background:#f4f6f9;color:#1a1a2e;padding:1.5rem}
h1{font-size:1.4rem;margin-bottom:1rem}
a{color:#4361ee;text-decoration:none}a:hover{text-decoration:underline}
table{width:100%;border-collapse:collapse;background:#fff;border-radius:8px;overflow:hidden;box-shadow:0 1px 4px rgba(0,0,0,.08)}
th,td{padding:.6rem .9rem;text-align:left;border-bottom:1px solid #edf2f7;font-size:.85rem}
th{background:#f8fafc;font-weight:600;color:#4a5568}
tr:last-child td{border-bottom:none}
.pass{display:inline-block;padding:.15rem .5rem;border-radius:4px;background:#d1fae5;color:#065f46;font-weight:600;font-size:.78rem}
.fail{display:inline-block;padding:.15rem .5rem;border-radius:4px;background:#fee2e2;color:#7f1d1d;font-weight:600;font-size:.78rem}
.err{color:#c53030;font-weight:600}
.warn{color:#d97706}
.info{color:#4a5568}
.bar-row{display:flex;gap:2px;align-items:center;height:14px;min-width:60px}
.bar{height:100%;border-radius:2px}
.bar-e{background:#fc8181}
.bar-w{background:#f6ad55}
.bar-i{background:#90cdf4}
</style>
</head>
<body>
<h1>FuSaOps — Run History &nbsp;<a href="/" style="font-size:.8rem;font-weight:400">← dashboard</a></h1>
`)
	if len(snaps) == 0 {
		fmt.Fprint(w, `<p style="margin-top:1rem;color:#718096">No history yet. Run <code>fusaops serve</code> or trigger a <code>/refresh</code> to record the first snapshot.</p>`)
	} else {
		maxTotal := 1
		for _, sn := range snaps {
			if sn.Total > maxTotal {
				maxTotal = sn.Total
			}
		}
		fmt.Fprint(w, `<table>
<thead><tr>
<th>Run</th><th>Status</th><th>Errors</th><th>Warnings</th><th>Infos</th><th>Severity trend</th><th>Languages</th>
</tr></thead><tbody>`)
		for i := len(snaps) - 1; i >= 0; i-- {
			sn := snaps[i]
			badge := `<span class="pass">PASS</span>`
			if sn.Status == "FAIL" {
				badge = `<span class="fail">FAIL</span>`
			}
			eW, wW, iW := 0, 0, 0
			if maxTotal > 0 {
				eW = sn.Errors * 100 / maxTotal
				wW = sn.Warnings * 100 / maxTotal
				iW = sn.Infos * 100 / maxTotal
			}
			var langs string
			for _, l := range sn.Languages {
				if langs != "" {
					langs += ", "
				}
				langs += html.EscapeString(l.Language)
				if l.Errors > 0 {
					langs += fmt.Sprintf(" <span class=err>(%de)</span>", l.Errors)
				}
			}
			if langs == "" {
				langs = "—"
			}
			fmt.Fprintf(w,
				`<tr><td>%s</td><td>%s</td>`+
					`<td class="err">%d</td><td class="warn">%d</td><td class="info">%d</td>`+
					`<td><div class="bar-row">`+
					`<div class="bar bar-e" style="width:%d%%"></div>`+
					`<div class="bar bar-w" style="width:%d%%"></div>`+
					`<div class="bar bar-i" style="width:%d%%"></div>`+
					`</div></td><td>%s</td></tr>`,
				sn.RunAt.UTC().Format("2006-01-02 15:04 UTC"),
				badge, sn.Errors, sn.Warnings, sn.Infos,
				eW, wW, iW, langs)
		}
		fmt.Fprint(w, "</tbody></table>")
	}
	fmt.Fprintf(w, `
<p style="margin-top:1rem;font-size:.8rem;color:#a0aec0">Showing last %d runs · <a href="/api/history">JSON</a></p>
</body></html>`, len(snaps))
}

// ListenAndServe computes the initial report then serves the dashboard on addr.
//
//fusa:req REQ-FO-SRV005
func (s *Server) ListenAndServe(addr string) error {
	ln, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", addr)
	if err != nil {
		return err
	}
	return s.Serve(ln)
}

// Serve computes the initial report then serves the dashboard on ln. Closing ln
// stops the server. Split from ListenAndServe so the serve path is testable on
// an ephemeral listener.
//
//fusa:req REQ-FO-SRV005
func (s *Server) Serve(ln net.Listener) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	s.compute(ctx)
	cancel()
	srv := &http.Server{
		Handler:           s.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}
	return srv.Serve(ln)
}
