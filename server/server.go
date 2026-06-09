// Package server provides the FuSaOps web reporting UI.
//
// It serves a single self-contained dashboard rendered from the aggregated
// multi-language report, plus a JSON API for programmatic consumers. The report
// is computed by the orchestrator on startup and on demand via /refresh, and
// cached between requests so the dashboard loads instantly.
package server

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

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

// compute runs the orchestrator and caches the result.
//
//fusa:req REQ-FO-SRV003
func (s *Server) compute(ctx context.Context) {
	rep, err := s.runner.Run(ctx, s.root, s.opts)
	s.mu.Lock()
	s.cached, s.err = rep, err
	s.mu.Unlock()
}

// Handler returns the HTTP routes for the dashboard and API.
//
//fusa:req REQ-FO-SRV004
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleIndex)
	mux.HandleFunc("/api/report", s.handleAPIReport)
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

// ListenAndServe computes the initial report then serves the dashboard on addr.
//
//fusa:req REQ-FO-SRV005
func (s *Server) ListenAndServe(addr string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	s.compute(ctx)
	cancel()
	srv := &http.Server{
		Addr:              addr,
		Handler:           s.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}
	return srv.ListenAndServe()
}
