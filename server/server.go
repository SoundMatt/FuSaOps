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
	"crypto/tls"
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
	"github.com/SoundMatt/FuSaOps/comp"
	"github.com/SoundMatt/FuSaOps/diff"
	"github.com/SoundMatt/FuSaOps/fleet"
	"github.com/SoundMatt/FuSaOps/history"
	"github.com/SoundMatt/FuSaOps/orchestrator"
	"github.com/SoundMatt/FuSaOps/report"
	"github.com/SoundMatt/FuSaOps/vv"
)

// Server serves the FuSaOps dashboard and JSON API.
//
//fusa:req REQ-FO-SRV001
type Server struct {
	root            string
	project         string
	runner          *orchestrator.Runner
	opts            orchestrator.Options
	histDir         string // empty = history persistence disabled
	authUser        string // empty = no authentication required (rw)
	authPass        string
	authROUser      string // read-only credentials (optional)
	authROPass      string
	auditDir        string         // empty = no audit log
	fleetCfg        string         // empty = fleet dashboard disabled
	webhookURL      string         // empty = no webhook notifications
	refreshInterval time.Duration  // zero = no scheduled refresh
	baselineFile    string         // empty = no baseline configured
	vvDecl          vv.Declaration // populated via WithVandV
	qualifyPath     string         // empty = auto-discover from root
	compThreshold   int            // 0 = use DAL default
	compDAL         string         // empty = use DAL-B default

	mu         sync.RWMutex
	cached     *report.AggregateReport
	err        error
	prevStatus string // previous status for webhook change detection

	compMu  sync.RWMutex
	compAgg *comp.Aggregate
	compErr error

	fleetMu  sync.RWMutex
	fleetRep *fleet.FleetReport
	fleetErr error
}

// auditEntry is a single dashboard access record written to .fusaops-audit.jsonl.
//
//fusa:req REQ-FO-AUDIT001
type auditEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Method    string    `json:"method"`
	Path      string    `json:"path"`
	User      string    `json:"user"`
	Status    int       `json:"status"`
}

// statusRecorder captures the HTTP status code written by a handler.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
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

// WithAuth enables HTTP Basic Auth on all routes. Requests without valid
// credentials receive 401 Unauthorized with a WWW-Authenticate challenge.
// Calling with empty user and pass disables authentication.
// Credentials set via WithAuth have read-write access (including /refresh).
//
//fusa:req REQ-FO-AUTH001
func (s *Server) WithAuth(user, pass string) *Server {
	s.authUser, s.authPass = user, pass
	return s
}

// WithAuthRO adds a second set of credentials with read-only access. Users
// authenticating with these credentials may view dashboards and query the API
// but cannot trigger mutating actions such as /refresh.
//
//fusa:req REQ-FO-RBAC001
func (s *Server) WithAuthRO(user, pass string) *Server {
	s.authROUser, s.authROPass = user, pass
	return s
}

// WithAuditLog enables request audit logging. Each authenticated request is
// appended as a JSON record to .fusaops-audit.jsonl in dir.
//
//fusa:req REQ-FO-AUDIT001
func (s *Server) WithAuditLog(dir string) *Server {
	s.auditDir = dir
	return s
}

// WithFleetConfig sets the path to a fleet.json config file. When set,
// the server exposes a /fleet HTML dashboard and /api/fleet JSON endpoint
// showing the status of all repos in the fleet.
//
//fusa:req REQ-FO-FLT005
func (s *Server) WithFleetConfig(path string) *Server {
	s.fleetCfg = path
	return s
}

// WithWebhook sets a URL to POST status-change notifications to. When the
// aggregate status transitions (e.g. PASS→FAIL), the server sends a JSON
// payload with the old and new status and error counts. One retry on failure.
//
//fusa:req REQ-FO-HOOK001
func (s *Server) WithWebhook(url string) *Server {
	s.webhookURL = url
	return s
}

// WithRefreshInterval enables automatic background rescans at the given
// interval. The first scan always runs at startup; subsequent scans fire
// after each tick until the listener is closed. A zero or negative interval
// disables scheduled rescans.
//
//fusa:req REQ-FO-SCHD001
func (s *Server) WithRefreshInterval(d time.Duration) *Server {
	s.refreshInterval = d
	return s
}

// WithBaseline sets the path to a baseline JSON file used by /api/v1/diff and
// saved by POST /api/v1/baseline.
//
//fusa:req REQ-FO-SRV007
func (s *Server) WithBaseline(path string) *Server {
	s.baselineFile = path
	return s
}

// WithVandV supplies V&V independence declarations for the /api/v1/vv endpoint
// and /badge/vv.svg badge.
//
//fusa:req REQ-FO-SRV010
func (s *Server) WithVandV(d vv.Declaration) *Server {
	s.vvDecl = d
	return s
}

// compute runs the orchestrator, caches the result, and persists a snapshot.
//
//fusa:req REQ-FO-SRV003
//fusa:req REQ-FO-SRV012
func (s *Server) compute(ctx context.Context) {
	rep, err := s.runner.Run(ctx, s.root, s.opts)
	s.mu.Lock()
	prev := s.prevStatus
	s.cached, s.err = rep, err
	newStatus := ""
	if err == nil && rep != nil {
		newStatus = rep.Summary.Status()
		s.prevStatus = newStatus
	}
	s.mu.Unlock()
	if err == nil && rep != nil {
		if s.histDir != "" {
			snap := history.FromReport(rep)
			_ = history.Store(s.histDir, snap)
		}
		if s.webhookURL != "" && prev != "" && newStatus != prev {
			go fireWebhook(s.webhookURL, prev, newStatus, rep.Summary.Errors)
		}
	}
	// Run comp separately; missing Compler adapters are silently skipped.
	compAgg, compErr := s.runner.RunComp(ctx, s.root, s.opts, s.compThreshold, s.compDAL)
	s.compMu.Lock()
	s.compAgg, s.compErr = compAgg, compErr
	s.compMu.Unlock()
	if s.fleetCfg != "" {
		s.computeFleet(ctx)
	}
}

// computeFleet loads the fleet config, runs a parallel fleet check, and caches the result.
//
//fusa:req REQ-FO-FLT005
func (s *Server) computeFleet(ctx context.Context) {
	cfg, err := fleet.LoadConfig(s.fleetCfg)
	if err != nil {
		s.fleetMu.Lock()
		s.fleetRep, s.fleetErr = nil, err
		s.fleetMu.Unlock()
		return
	}
	fr := fleet.Run(ctx, cfg, s.runner)
	s.fleetMu.Lock()
	s.fleetRep, s.fleetErr = fr, nil
	s.fleetMu.Unlock()
}

// Handler returns the HTTP routes for the dashboard and API.
//
//fusa:req REQ-FO-SRV004
//fusa:req REQ-FO-SRV006
//fusa:req REQ-FO-SRV007
//fusa:req REQ-FO-SRV008
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleIndex)
	mux.HandleFunc("/api/report", s.handleAPIReport)
	mux.HandleFunc("/api/history", s.handleAPIHistory)
	mux.HandleFunc("/api/v1/report", s.handleAPIReport)
	mux.HandleFunc("/api/v1/status", s.handleAPIStatus)
	mux.HandleFunc("/api/v1/findings", s.handleAPIFindings)
	mux.HandleFunc("/api/v1/history", s.handleAPIHistory)
	mux.HandleFunc("/api/v1/export", s.handleExport)
	mux.HandleFunc("/api/v1/diff", s.handleAPIDiff)
	mux.HandleFunc("/api/v1/baseline", s.handleAPIBaseline)
	mux.HandleFunc("/history", s.handleHistory)
	mux.HandleFunc("/refresh", s.handleRefresh)
	mux.HandleFunc("/healthz", s.handleHealth)
	mux.HandleFunc("/badge/status.svg", s.handleBadge)
	mux.HandleFunc("/api/v1/comp", s.handleAPIComp)
	mux.HandleFunc("/comp", s.handleComp)
	mux.HandleFunc("/api/v1/vv", s.handleAPIVandV)
	mux.HandleFunc("/badge/vv.svg", s.handleVandVBadge)
	mux.HandleFunc("/badge/qualify.svg", s.handleQualifyBadge)
	mux.HandleFunc("/metrics", s.handleMetrics)
	if s.fleetCfg != "" {
		mux.HandleFunc("/fleet", s.handleFleet)
		mux.HandleFunc("/api/fleet", s.handleAPIFleet)
	}
	if s.authUser != "" || s.authROUser != "" {
		return s.authMiddleware(mux)
	}
	return mux
}

// mutatingPaths are routes that modify server state. Read-only credentials
// are rejected with 403 Forbidden on these paths.
var mutatingPaths = map[string]bool{"/refresh": true}

// authMiddleware wraps h with HTTP Basic Auth and optional role gating.
// Unauthenticated requests receive 401; ro credentials on mutating paths get 403.
//
//fusa:req REQ-FO-AUTH001
//fusa:req REQ-FO-AUTH002
//fusa:req REQ-FO-RBAC001
//fusa:req REQ-FO-RBAC002
func (s *Server) authMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, p, ok := r.BasicAuth()
		role := ""
		if ok {
			if s.authUser != "" && u == s.authUser && p == s.authPass {
				role = "rw"
			} else if s.authROUser != "" && u == s.authROUser && p == s.authROPass {
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
		if s.auditDir != "" {
			_ = appendAudit(s.auditDir, auditEntry{
				Timestamp: time.Now().UTC(),
				Method:    r.Method,
				Path:      r.URL.Path,
				User:      u,
				Status:    rec.status,
			})
		}
	})
}

// appendAudit appends an audit entry to .fusaops-audit.jsonl in dir.
//
//fusa:req REQ-FO-AUDIT002
func appendAudit(dir string, e auditEntry) error {
	f, err := openAppend(dir, "fusaops-audit.jsonl")
	if err != nil {
		return err
	}
	defer f.Close()
	return json.NewEncoder(f).Encode(e)
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
	s.compMu.RLock()
	compAgg := s.compAgg
	s.compMu.RUnlock()
	var compInfo *report.CompInfo
	if compAgg != nil {
		compInfo = compInfoFromAggregate(compAgg)
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	qi := loadQualifyInfo(s.root, s.qualifyPath)
	opts := report.RenderOptions{QualifyInfo: qi, CompInfo: compInfo}
	if err := report.RenderWithOptions(w, rep, "html", opts); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// compInfoFromAggregate converts a comp.Aggregate into a report.CompInfo for
// the HTML dashboard renderer. The conversion avoids a report→comp import cycle.
//
//fusa:req REQ-FO-RPT021
func compInfoFromAggregate(agg *comp.Aggregate) *report.CompInfo {
	info := &report.CompInfo{
		TotalFunctions: agg.TotalFunctions,
		Violations:     agg.Violations,
	}
	for _, c := range agg.Components {
		cc := report.CompComponent{
			Language: c.Language,
			Tool:     c.Tool,
			Skipped:  c.Skipped,
		}
		if c.Report != nil {
			cc.TotalFunctions = c.Report.TotalFunctions
			cc.Violations = c.Report.Violations
			cc.Threshold = c.Report.Threshold
			cc.DAL = c.Report.DAL
		}
		info.Components = append(info.Components, cc)
	}
	return info
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

// handleAPIStatus returns a lightweight status JSON object for polling.
//
//fusa:req REQ-FO-API001
func (s *Server) handleAPIStatus(w http.ResponseWriter, _ *http.Request) {
	s.mu.RLock()
	rep, cErr := s.cached, s.err
	s.mu.RUnlock()
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if cErr != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, `{"status":"ERROR","error":%q}`+"\n", cErr.Error())
		return
	}
	if rep == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprint(w, `{"status":"PENDING"}`+"\n")
		return
	}
	fmt.Fprintf(w, `{"status":%q,"errors":%d,"warnings":%d,"total":%d}`+"\n",
		rep.Summary.Status(), rep.Summary.Errors, rep.Summary.Warnings, rep.Summary.Total)
}

// handleAPIFindings returns findings filtered by severity and/or language query params.
//
//fusa:req REQ-FO-API002
func (s *Server) handleAPIFindings(w http.ResponseWriter, r *http.Request) {
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
	q := r.URL.Query()
	sev := q.Get("severity")
	lang := q.Get("language")
	tool := q.Get("tool")

	type apiFinding struct {
		Language    string `json:"language"`
		Tool        string `json:"tool"`
		RuleID      string `json:"ruleId"`
		Severity    string `json:"severity"`
		Message     string `json:"message"`
		File        string `json:"file"`
		Line        int    `json:"line,omitempty"`
		Category    string `json:"category,omitempty"`
		Fingerprint string `json:"fingerprint,omitempty"`
	}
	var out []apiFinding
	for _, c := range rep.Components {
		if lang != "" && string(c.Language) != lang {
			continue
		}
		if tool != "" && c.Tool != tool {
			continue
		}
		for _, f := range c.Findings {
			if sev != "" && string(f.Severity) != sev {
				continue
			}
			out = append(out, apiFinding{
				Language:    string(f.Language),
				Tool:        f.Tool,
				RuleID:      f.RuleID,
				Severity:    string(f.Severity),
				Message:     f.Message,
				File:        f.Location.File,
				Line:        f.Location.Line,
				Category:    f.Category,
				Fingerprint: f.Fingerprint,
			})
		}
	}
	if out == nil {
		out = []apiFinding{} // ensure JSON array, not null
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(out)
}

// handleExport serves the cached report in the requested format as a download.
// The format is taken from the ?format= query parameter (default: json).
// Supported: text, json, html, sarif, junit, csv, markdown, md.
//
//fusa:req REQ-FO-SRV006
func (s *Server) handleExport(w http.ResponseWriter, r *http.Request) {
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
	format := r.URL.Query().Get("format")
	if format == "" {
		format = "json"
	}
	ct, ext := exportMIME(format)
	w.Header().Set("Content-Type", ct)
	w.Header().Set("Content-Disposition", "attachment; filename=\"fusaops-report."+ext+"\"")
	if err := report.Render(w, rep, format); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// exportMIME maps a report format to a MIME type and file extension.
func exportMIME(format string) (contentType, ext string) {
	switch format {
	case "json":
		return "application/json; charset=utf-8", "json"
	case "csv":
		return "text/csv; charset=utf-8", "csv"
	case "junit":
		return "application/xml; charset=utf-8", "xml"
	case "sarif":
		return "application/json; charset=utf-8", "sarif.json"
	case "html":
		return "text/html; charset=utf-8", "html"
	case "markdown", "md":
		return "text/markdown; charset=utf-8", "md"
	default:
		return "text/plain; charset=utf-8", "txt"
	}
}

// handleAPIDiff compares the cached report against a baseline and returns the
// delta. The baseline path is taken from ?baseline= (required unless
// --baseline was given at startup). ?strict=true causes a 409 response when
// new ERROR-severity findings are present. ?format=json|text controls output.
//
//fusa:req REQ-FO-SRV007
func (s *Server) handleAPIDiff(w http.ResponseWriter, r *http.Request) {
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
	baselinePath := r.URL.Query().Get("baseline")
	if baselinePath == "" {
		baselinePath = s.baselineFile
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
	for _, c := range rep.Components {
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

// handleAPIBaseline saves the current cached report findings as a baseline file.
// The file path must be configured via WithBaseline; returns 400 otherwise.
//
//fusa:req REQ-FO-SRV008
func (s *Server) handleAPIBaseline(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}
	if s.baselineFile == "" {
		http.Error(w, "no baseline path configured: use --baseline", http.StatusBadRequest)
		return
	}
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
	var findings []fusaops.Finding
	for _, c := range rep.Components {
		findings = append(findings, c.Findings...)
	}
	if err := diff.SaveBaseline(s.baselineFile, findings); err != nil {
		http.Error(w, "save baseline: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	fmt.Fprintf(w, `{"saved":%q,"findings":%d}`+"\n", s.baselineFile, len(findings))
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

// handleFleet renders the HTML fleet status dashboard.
//
//fusa:req REQ-FO-FLT006
func (s *Server) handleFleet(w http.ResponseWriter, _ *http.Request) {
	s.fleetMu.RLock()
	fr, fErr := s.fleetRep, s.fleetErr
	s.fleetMu.RUnlock()
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if fErr != nil {
		http.Error(w, "fleet scan failed: "+fErr.Error(), http.StatusInternalServerError)
		return
	}
	if fr == nil {
		http.Error(w, "fleet report not available yet", http.StatusServiceUnavailable)
		return
	}
	fmt.Fprint(w, `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>FuSaOps — Fleet</title>
<style>
*{box-sizing:border-box;margin:0;padding:0}
body{font-family:system-ui,sans-serif;background:#f4f6f9;color:#1a1a2e;padding:1.5rem}
h1{font-size:1.4rem;margin-bottom:.5rem}
.sub{font-size:.85rem;color:#718096;margin-bottom:1rem}
a{color:#4361ee;text-decoration:none}a:hover{text-decoration:underline}
table{width:100%;border-collapse:collapse;background:#fff;border-radius:8px;overflow:hidden;box-shadow:0 1px 4px rgba(0,0,0,.08)}
th,td{padding:.6rem .9rem;text-align:left;border-bottom:1px solid #edf2f7;font-size:.85rem}
th{background:#f8fafc;font-weight:600;color:#4a5568}
tr:last-child td{border-bottom:none}
.pass{display:inline-block;padding:.15rem .5rem;border-radius:4px;background:#d1fae5;color:#065f46;font-weight:600;font-size:.78rem}
.warn{display:inline-block;padding:.15rem .5rem;border-radius:4px;background:#fef3c7;color:#92400e;font-weight:600;font-size:.78rem}
.fail{display:inline-block;padding:.15rem .5rem;border-radius:4px;background:#fee2e2;color:#7f1d1d;font-weight:600;font-size:.78rem}
.err{color:#c53030;font-weight:600}.wc{color:#d97706}.ic{color:#4a5568}
</style>
</head>
<body>
`)
	statusBadge := func(st string) string {
		switch st {
		case "FAIL":
			return `<span class="fail">FAIL</span>`
		case "WARN":
			return `<span class="warn">WARN</span>`
		default:
			return `<span class="pass">PASS</span>`
		}
	}
	fmt.Fprintf(w, `<h1>FuSaOps — Fleet: %s &nbsp;<a href="/" style="font-size:.8rem;font-weight:400">← dashboard</a></h1>`,
		html.EscapeString(fr.Project))
	fmt.Fprintf(w, `<p class="sub">%s · Overall: %s · %d repos · %d errors · %d warnings</p>`,
		fr.GeneratedAt.UTC().Format("2006-01-02 15:04 UTC"),
		statusBadge(fr.Status()), len(fr.Repos), fr.Errors, fr.Warnings)
	fmt.Fprint(w, `<table>
<thead><tr><th>Repo</th><th>Status</th><th>Total</th><th>Errors</th><th>Warnings</th><th>Infos</th></tr></thead><tbody>`)
	for _, r := range fr.Repos {
		fmt.Fprintf(w,
			`<tr><td>%s</td><td>%s</td><td>%d</td><td class="err">%d</td><td class="wc">%d</td><td class="ic">%d</td></tr>`,
			html.EscapeString(r.Name), statusBadge(r.Status), r.Total, r.Errors, r.Warnings, r.Infos)
	}
	fmt.Fprint(w, `</tbody></table>
<p style="margin-top:1rem;font-size:.8rem;color:#a0aec0"><a href="/api/fleet">JSON</a> · <a href="/refresh">Refresh</a></p>
</body></html>`)
}

// handleAPIFleet serves the cached fleet report as JSON.
//
//fusa:req REQ-FO-FLT006
func (s *Server) handleAPIFleet(w http.ResponseWriter, _ *http.Request) {
	s.fleetMu.RLock()
	fr, fErr := s.fleetRep, s.fleetErr
	s.fleetMu.RUnlock()
	if fErr != nil {
		http.Error(w, fErr.Error(), http.StatusInternalServerError)
		return
	}
	if fr == nil {
		http.Error(w, "fleet report not available yet", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(fr)
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

// ListenAndServeTLS computes the initial report then serves the dashboard over
// HTTPS on addr using the provided certificate and key files.
//
//fusa:req REQ-FO-TLS001
func (s *Server) ListenAndServeTLS(addr, certFile, keyFile string) error {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return err
	}
	ln, err := tls.Listen("tcp", addr, &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	})
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
//fusa:req REQ-FO-SCHD001
func (s *Server) Serve(ln net.Listener) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	s.compute(ctx)
	cancel()
	if s.refreshInterval > 0 {
		go s.runScheduler()
	}
	srv := &http.Server{
		Handler:           s.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}
	return srv.Serve(ln)
}

// runScheduler fires periodic rescans on s.refreshInterval until the process exits.
//
//fusa:req REQ-FO-SCHD001
func (s *Server) runScheduler() {
	ticker := time.NewTicker(s.refreshInterval)
	defer ticker.Stop()
	for range ticker.C {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		s.compute(ctx)
		cancel()
	}
}

// handleBadge renders an SVG status badge in shields.io flat style.
//
//fusa:req REQ-FO-BADGE001
func (s *Server) handleBadge(w http.ResponseWriter, _ *http.Request) {
	s.mu.RLock()
	rep, cErr := s.cached, s.err
	s.mu.RUnlock()

	status, color := "pending", "#9f9f9f"
	if cErr != nil {
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
	fmt.Fprint(w, svgBadge("fusaops", status, color))
}

// WithComp sets the cyclomatic complexity threshold and DAL for the
// /api/v1/comp endpoint. threshold=0 means use the DAL default.
//
//fusa:req REQ-FO-SRV012
func (s *Server) WithComp(threshold int, dal string) *Server {
	s.compThreshold = threshold
	s.compDAL = dal
	return s
}

// handleAPIComp serves the /api/v1/comp JSON endpoint with the cached
// cross-language cyclomatic complexity aggregate.
//
//fusa:req REQ-FO-SRV012
func (s *Server) handleAPIComp(w http.ResponseWriter, _ *http.Request) {
	s.compMu.RLock()
	agg, err := s.compAgg, s.compErr
	s.compMu.RUnlock()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if agg == nil {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, "{}\n")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(agg)
}

// handleComp renders the full cyclomatic complexity HTML page at /comp showing
// per-component function-level detail for all functions that exceed the threshold.
//
//fusa:req REQ-FO-SRV013
func (s *Server) handleComp(w http.ResponseWriter, _ *http.Request) {
	s.compMu.RLock()
	agg, aggErr := s.compAgg, s.compErr
	s.compMu.RUnlock()
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>FuSaOps — Cyclomatic Complexity</title>
<style>
*{box-sizing:border-box;margin:0;padding:0}
body{font-family:system-ui,sans-serif;background:#f4f6f9;color:#1a1a2e;padding:1.5rem}
h1{font-size:1.4rem;margin-bottom:1rem}
h2{font-size:1.1rem;margin:1.2rem 0 .6rem}
a{color:#4361ee;text-decoration:none}a:hover{text-decoration:underline}
.badge{display:inline-block;padding:.15rem .5rem;border-radius:4px;font-weight:600;font-size:.78rem}
.pass{background:#d1fae5;color:#065f46}
.fail{background:#fee2e2;color:#7f1d1d}
.sub{color:#718096;font-size:.85rem;margin-left:.5rem}
table{width:100%;border-collapse:collapse;background:#fff;border-radius:8px;overflow:hidden;box-shadow:0 1px 4px rgba(0,0,0,.08);margin-bottom:1.5rem}
th,td{padding:.6rem .9rem;text-align:left;border-bottom:1px solid #edf2f7;font-size:.85rem}
th{background:#f8fafc;font-weight:600;color:#4a5568}
tr:last-child td{border-bottom:none}
.viol{color:#c53030;font-weight:700}
.ok{color:#2d6a4f}
code{font-family:ui-monospace,monospace;font-size:.82rem;color:#4a5568}
.empty{color:#718096;padding:1rem 0}
</style>
</head>
<body>
<h1>FuSaOps — Cyclomatic Complexity &nbsp;<a href="/" style="font-size:.8rem;font-weight:400">← dashboard</a></h1>
`)
	if aggErr != nil {
		fmt.Fprintf(w, "<p style=\"color:#c53030\">Error: %s</p></body></html>\n", html.EscapeString(aggErr.Error()))
		return
	}
	if agg == nil {
		fmt.Fprint(w, `<p class="empty">No complexity data available. Run <code>fusaops comp</code> or wait for the server to compute.</p></body></html>`+"\n")
		return
	}
	badge := `<span class="badge pass">PASS</span>`
	if agg.Violations > 0 {
		badge = fmt.Sprintf(`<span class="badge fail">%d violations</span>`, agg.Violations)
	}
	fmt.Fprintf(w, "<p>%s <span class=\"sub\">%d functions across %d component(s)</span></p>\n",
		badge, agg.TotalFunctions, len(agg.Components))
	for _, c := range agg.Components {
		fmt.Fprintf(w, "<h2>%s <code>%s</code></h2>\n",
			html.EscapeString(c.Language), html.EscapeString(c.Tool))
		if c.Skipped != "" {
			fmt.Fprintf(w, "<p class=\"empty\">skipped: %s</p>\n", html.EscapeString(c.Skipped))
			continue
		}
		if c.Report == nil {
			fmt.Fprint(w, "<p class=\"empty\">no report available</p>\n")
			continue
		}
		r := c.Report
		dal := ""
		if r.DAL != "" {
			dal = fmt.Sprintf(" · %s (≤%d)", html.EscapeString(r.DAL), r.Threshold)
		} else if r.Threshold > 0 {
			dal = fmt.Sprintf(" · threshold ≤%d", r.Threshold)
		}
		fmt.Fprintf(w, "<p class=\"sub\" style=\"margin-bottom:.5rem\">%d functions · %d violations%s</p>\n",
			r.TotalFunctions, r.Violations, dal)
		// Show functions that exceed the threshold; if none, show brief summary.
		var violFuncs []comp.Function
		for _, f := range r.Results {
			if f.ExceedsThreshold {
				violFuncs = append(violFuncs, f)
			}
		}
		if len(violFuncs) == 0 {
			fmt.Fprint(w, "<p class=\"empty\">No functions exceed the threshold.</p>\n")
			continue
		}
		fmt.Fprint(w, "<table><thead><tr><th>Function</th><th>File</th><th>Line</th><th>V(G)</th></tr></thead><tbody>\n")
		for _, f := range violFuncs {
			locLine := ""
			if f.Line > 0 {
				locLine = fmt.Sprintf("%d", f.Line)
			}
			fmt.Fprintf(w,
				"<tr><td><code class=\"viol\">%s</code></td><td><code>%s</code></td><td>%s</td><td class=\"viol\">%d</td></tr>\n",
				html.EscapeString(f.Name),
				html.EscapeString(f.File),
				locLine,
				f.Complexity,
			)
		}
		fmt.Fprint(w, "</tbody></table>\n")
	}
	fmt.Fprintf(w, "<p style=\"font-size:.8rem;color:#a0aec0;margin-top:1rem\">V(G) = McCabe cyclomatic complexity · <a href=\"/api/v1/comp\">JSON</a></p>\n</body></html>\n")
}

// handleAPIVandV serves the /api/v1/vv JSON endpoint with V&V independence
// declarations and the derived achievable ASIL.
//
//fusa:req REQ-FO-SRV010
func (s *Server) handleAPIVandV(w http.ResponseWriter, _ *http.Request) {
	type response struct {
		Project                 string `json:"project,omitempty"`
		ImplementationAuthor    string `json:"implementationAuthor,omitempty"`
		IndependentReviewer     string `json:"independentReviewer,omitempty"`
		IndependentTestExecutor string `json:"independentTestExecutor,omitempty"`
		IndependenceLevel       int    `json:"independenceLevel"`
		AchievableASIL          string `json:"achievableAsil"`
	}
	d := s.vvDecl
	resp := response{
		Project:                 d.Project,
		ImplementationAuthor:    d.ImplementationAuthor,
		IndependentReviewer:     d.IndependentReviewer,
		IndependentTestExecutor: d.IndependentTestExecutor,
		IndependenceLevel:       vv.IndependenceLevel(d),
		AchievableASIL:          vv.AchievableASIL(d),
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(resp)
}

// handleVandVBadge renders an SVG badge showing the achievable ASIL from the
// V&V independence declarations.
//
//fusa:req REQ-FO-BADGE003
func (s *Server) handleVandVBadge(w http.ResponseWriter, _ *http.Request) {
	asil := vv.AchievableASIL(s.vvDecl)
	color := asilBadgeColor(asil)
	w.Header().Set("Content-Type", "image/svg+xml")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	fmt.Fprint(w, svgBadge("v&v", asil, color))
}

// asilBadgeColor returns the badge background color for the given achievable ASIL.
func asilBadgeColor(asil string) string {
	switch asil {
	case "ASIL-D":
		return "#4c1"
	case "ASIL-C":
		return "#97ca00"
	case "ASIL-B":
		return "#dfb317"
	default:
		return "#9f9f9f"
	}
}

// svgBadge returns a minimal shields.io-style flat SVG badge.
func svgBadge(label, message, color string) string {
	lw := len(label)*6 + 10
	mw := len(message)*6 + 10
	total := lw + mw
	lx := lw/2 + 1
	mx := lw + mw/2
	return fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="20">
<linearGradient id="s" x2="0" y2="100%%"><stop offset="0" stop-color="#bbb" stop-opacity=".1"/><stop offset="1" stop-opacity=".1"/></linearGradient>
<rect width="%d" height="20" rx="3" fill="#555"/>
<rect x="%d" width="%d" height="20" rx="3" fill="%s"/>
<rect width="%d" height="20" rx="3" fill="url(#s)"/>
<g fill="#fff" text-anchor="middle" font-family="DejaVu Sans,Verdana,Geneva,sans-serif" font-size="11">
<text x="%d" y="15" fill="#010101" fill-opacity=".3">%s</text>
<text x="%d" y="14">%s</text>
<text x="%d" y="15" fill="#010101" fill-opacity=".3">%s</text>
<text x="%d" y="14">%s</text>
</g></svg>`, total, total, lw, mw, color, total, lx, label, lx, label, mx, message, mx, message)
}

// fireWebhook POSTs a status-change notification to url. Retries once on failure.
//
//fusa:req REQ-FO-HOOK001
//fusa:req REQ-FO-HOOK002
func fireWebhook(url, prev, current string, errors int) {
	body := fmt.Sprintf(`{"status":%q,"prev":%q,"errors":%d}`+"\n", current, prev, errors)
	for i := range 2 {
		resp, err := http.Post(url, "application/json", strings.NewReader(body)) //nolint:noctx
		if err == nil {
			_ = resp.Body.Close()
			return
		}
		if i == 0 {
			time.Sleep(2 * time.Second)
		}
	}
}

// handleMetrics serves an OpenMetrics / Prometheus text exposition of current findings.
//
//fusa:req REQ-FO-MTR001
func (s *Server) handleMetrics(w http.ResponseWriter, _ *http.Request) {
	s.mu.RLock()
	rep, rErr := s.cached, s.err
	s.mu.RUnlock()
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	fmt.Fprint(w, buildMetrics(rep, rErr, ""))
}

// buildMetrics produces an OpenMetrics text exposition from an aggregate report.
// project is an optional label value added to all metrics (empty = single-project mode).
//
//fusa:req REQ-FO-MTR001
func buildMetrics(rep *report.AggregateReport, rErr error, project string) string {
	var errors, warnings, infos int
	status := "pending"
	if rErr != nil {
		status = "error"
	} else if rep != nil {
		status = rep.Summary.Status()
		errors = rep.Summary.Errors
		warnings = rep.Summary.Warnings
		infos = rep.Summary.Total - errors - warnings
		if infos < 0 {
			infos = 0
		}
	}
	statusCode := 0
	switch status {
	case "PASS":
		statusCode = 1
	case "WARN":
		statusCode = 2
	case "FAIL":
		statusCode = 3
	}

	plabel := ""
	if project != "" {
		plabel = `project="` + project + `",`
	}
	plabelTrimmed := strings.TrimSuffix(plabel, ",")
	if plabelTrimmed == "" {
		plabelTrimmed = ""
	}

	var b strings.Builder
	b.WriteString("# HELP fusaops_findings_total Total findings by severity\n")
	b.WriteString("# TYPE fusaops_findings_total gauge\n")
	fmt.Fprintf(&b, "fusaops_findings_total{%sseverity=\"error\"} %d\n", plabel, errors)
	fmt.Fprintf(&b, "fusaops_findings_total{%sseverity=\"warning\"} %d\n", plabel, warnings)
	fmt.Fprintf(&b, "fusaops_findings_total{%sseverity=\"info\"} %d\n", plabel, infos)
	b.WriteString("# HELP fusaops_status Aggregate status: 1=PASS 2=WARN 3=FAIL 0=pending/error\n")
	b.WriteString("# TYPE fusaops_status gauge\n")
	if plabelTrimmed != "" {
		fmt.Fprintf(&b, "fusaops_status{%s} %d\n", plabelTrimmed, statusCode)
	} else {
		fmt.Fprintf(&b, "fusaops_status %d\n", statusCode)
	}
	b.WriteString("# EOF\n")
	return b.String()
}

// openAppend opens a JSONL file in dir for appending, creating it if needed.
func openAppend(dir, name string) (*os.File, error) {
	return os.OpenFile(filepath.Join(dir, name), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
}
