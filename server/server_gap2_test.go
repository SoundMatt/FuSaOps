package server

// Additional tests targeting uncovered branches in Server: handleAPIReport
// error/nil states, handleBadge error/PASS/FAIL badges, handleAPIComp agg==nil
// state, handleComp error/nil/no-violations paths, and handleHistory FAIL badge.

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	fusaops "github.com/SoundMatt/FuSaOps"
	"github.com/SoundMatt/FuSaOps/adapter"
	"github.com/SoundMatt/FuSaOps/comp"
	"github.com/SoundMatt/FuSaOps/history"
	"github.com/SoundMatt/FuSaOps/orchestrator"
)

// newTestServerNoFindings creates a Server whose fakeAdapter produces no findings
// so that report.Summary.Status() == "PASS".
func newTestServerNoFindings(t *testing.T) *Server {
	t.Helper()
	reg := adapter.NewRegistry()
	reg.MustRegister(&fakeAdapter{tool: "gofusa", lang: fusaops.LangGo})
	s := New(t.TempDir(), orchestrator.New(reg), orchestrator.Options{Project: "demo"})
	s.compute(context.Background())
	return s
}

// newTestServerErrors creates a Server whose fakeAdapter produces ERROR findings
// so that report.Summary.Status() defaults to "FAIL".
func newTestServerErrors(t *testing.T) *Server {
	t.Helper()
	reg := adapter.NewRegistry()
	reg.MustRegister(&fakeAdapter{tool: "gofusa", lang: fusaops.LangGo, findings: []fusaops.Finding{
		{RuleID: "E001", Severity: fusaops.SeverityError, Message: "fatal error"},
	}})
	s := New(t.TempDir(), orchestrator.New(reg), orchestrator.Options{Project: "demo"})
	s.compute(context.Background())
	return s
}

// ── handleAPIReport ───────────────────────────────────────────────────────────

// TestAPIReportError verifies /api/v1/report returns 500 when the server has a
// cached scan error, covering the cErr != nil branch.
//
//fusa:test REQ-FO-SRV005
func TestAPIReportError(t *testing.T) {
	s := newTestServer(t)
	s.mu.Lock()
	s.err = errors.New("scan failed")
	s.mu.Unlock()

	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/report", nil))
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("want 500, got %d", rec.Code)
	}
}

// ── handleBadge (server.go) ───────────────────────────────────────────────────

// TestBadgeErrorState verifies /badge/status.svg shows "error" when the server
// has a cached scan error, covering the cErr != nil branch.
//
//fusa:test REQ-FO-BADGE001
func TestBadgeErrorState(t *testing.T) {
	s := newTestServer(t)
	s.mu.Lock()
	s.err = errors.New("scan failed")
	s.mu.Unlock()

	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/badge/status.svg", nil))
	if !strings.Contains(rec.Body.String(), "error") {
		t.Errorf("expected 'error' badge: %s", rec.Body.String())
	}
}

// TestBadgePASSState verifies /badge/status.svg shows "pass" when all checks pass,
// covering the case "PASS" branch.
//
//fusa:test REQ-FO-BADGE001
func TestBadgePASSState(t *testing.T) {
	s := newTestServerNoFindings(t)
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/badge/status.svg", nil))
	if !strings.Contains(rec.Body.String(), "pass") {
		t.Errorf("expected 'pass' badge: %s", rec.Body.String())
	}
}

// TestBadgeFAILState verifies /badge/status.svg shows "fail" when ERROR findings
// exist, covering the default (FAIL) badge branch.
//
//fusa:test REQ-FO-BADGE001
func TestBadgeFAILState(t *testing.T) {
	s := newTestServerErrors(t)
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/badge/status.svg", nil))
	if !strings.Contains(rec.Body.String(), "fail") {
		t.Errorf("expected 'fail' badge: %s", rec.Body.String())
	}
}

// ── handleAPIComp agg == nil ──────────────────────────────────────────────────

// TestAPICompNilAgg verifies /api/v1/comp returns {} when compAgg is nil and no
// error is set, covering the agg == nil branch in handleAPIComp.
//
//fusa:test REQ-FO-SRV012
func TestAPICompNilAgg(t *testing.T) {
	// Build a server without compute so compAgg stays nil.
	reg := adapter.NewRegistry()
	s := New(t.TempDir(), orchestrator.New(reg), orchestrator.Options{})
	// compAgg is nil, compErr is nil — no compute called.
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/comp", nil))
	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rec.Code)
	}
	body := strings.TrimSpace(rec.Body.String())
	if body != "{}" {
		t.Errorf("expected {}, got %q", body)
	}
}

// ── handleComp ────────────────────────────────────────────────────────────────

// TestCompPageErrorState verifies /comp renders the error paragraph when
// compErr is set, covering the aggErr != nil branch in handleComp.
//
//fusa:test REQ-FO-SRV013
func TestCompPageErrorState(t *testing.T) {
	s := newTestServer(t)
	s.compMu.Lock()
	s.compErr = errors.New("cyclomatic computation failed")
	s.compMu.Unlock()

	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/comp", nil))
	if rec.Code != http.StatusOK {
		t.Errorf("want 200 (error rendered in body), got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "cyclomatic computation failed") {
		t.Errorf("expected error message in /comp body: %s", rec.Body.String())
	}
}

// TestCompPageNilAgg verifies /comp renders the "no data" message when compAgg
// is nil, covering the agg == nil branch in handleComp.
//
//fusa:test REQ-FO-SRV013
func TestCompPageNilAgg(t *testing.T) {
	reg := adapter.NewRegistry()
	s := New(t.TempDir(), orchestrator.New(reg), orchestrator.Options{})
	// No compute — compAgg stays nil, compErr stays nil.
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/comp", nil))
	if !strings.Contains(rec.Body.String(), "No complexity data") {
		t.Errorf("expected 'No complexity data' message: %s", rec.Body.String())
	}
}

// TestCompPageNoReport verifies /comp shows "no report available" for a
// component whose Report field is nil, covering the c.Report == nil branch.
//
//fusa:test REQ-FO-SRV013
func TestCompPageNoReport(t *testing.T) {
	s := newTestServer(t)
	s.compMu.Lock()
	s.compErr = nil
	s.compAgg = &comp.Aggregate{
		Root:    "/repo",
		Project: "demo",
		Components: []comp.ComponentComp{
			{Language: "go", Tool: "gofusa", Available: true, Report: nil},
		},
	}
	s.compMu.Unlock()

	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/comp", nil))
	if !strings.Contains(rec.Body.String(), "no report available") {
		t.Errorf("expected 'no report available' for nil Report: %s", rec.Body.String())
	}
}

// TestCompPageNoViolations verifies /comp shows "No functions exceed the
// threshold" when no functions in the report violate the threshold, covering
// the len(violFuncs) == 0 branch.
//
//fusa:test REQ-FO-SRV013
func TestCompPageNoViolations(t *testing.T) {
	s := newTestServer(t)
	s.compMu.Lock()
	s.compErr = nil
	s.compAgg = &comp.Aggregate{
		Root:    "/repo",
		Project: "demo",
		Components: []comp.ComponentComp{{
			Language:  "go",
			Tool:      "gofusa",
			Available: true,
			Report: &comp.Report{
				Threshold:      10,
				TotalFunctions: 3,
				Violations:     0,
				Results: []comp.Function{
					{Name: "Foo", Complexity: 3, ExceedsThreshold: false},
				},
			},
		}},
	}
	s.compMu.Unlock()

	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/comp", nil))
	if !strings.Contains(rec.Body.String(), "No functions exceed the threshold") {
		t.Errorf("expected 'No functions exceed the threshold': %s", rec.Body.String())
	}
}

// ── handleHistory ─────────────────────────────────────────────────────────────

// TestHistoryPageFAILBadge verifies /history renders the FAIL badge for a
// snapshot with Status="FAIL", covering the sn.Status == "FAIL" branch.
//
//fusa:test REQ-FO-HST004
func TestHistoryPageFAILBadge(t *testing.T) {
	histDir := t.TempDir()
	snap := history.Snapshot{
		RunAt:    time.Now().UTC(),
		Status:   "FAIL",
		Total:    1,
		Errors:   1,
		Warnings: 0,
		Languages: []history.LanguageSummary{
			{Language: "go", Errors: 1, Warnings: 0},
		},
	}
	if err := history.Store(histDir, snap); err != nil {
		t.Fatalf("store snapshot: %v", err)
	}

	reg := adapter.NewRegistry()
	s := New(t.TempDir(), orchestrator.New(reg), orchestrator.Options{}).WithHistoryDir(histDir)

	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/history", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `class="fail"`) {
		t.Errorf("expected FAIL badge in history page: %s", rec.Body.String())
	}
}

// TestHistoryPageMultiLanguages verifies /history renders the error-count span
// when a snapshot has a language entry with Errors > 0, covering the
// l.Errors > 0 branch and the langs != "" join path.
//
//fusa:test REQ-FO-HST004
func TestHistoryPageMultiLanguages(t *testing.T) {
	histDir := t.TempDir()
	snap := history.Snapshot{
		RunAt:  time.Now().UTC(),
		Status: "FAIL",
		Total:  3,
		Errors: 2,
		Languages: []history.LanguageSummary{
			{Language: "go", Errors: 1},
			{Language: "c", Errors: 1},
		},
	}
	if err := history.Store(histDir, snap); err != nil {
		t.Fatalf("store snapshot: %v", err)
	}

	reg := adapter.NewRegistry()
	s := New(t.TempDir(), orchestrator.New(reg), orchestrator.Options{}).WithHistoryDir(histDir)

	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/history", nil))
	body := rec.Body.String()
	if !strings.Contains(body, "(1e)") {
		t.Errorf("expected error count span '(1e)' in history: %s", body)
	}
	if !strings.Contains(body, ", ") {
		t.Errorf("expected comma-join between languages: %s", body)
	}
}

// TestHistoryAPIError verifies /api/v1/history returns 500 when history.Load
// fails because histDir is a regular file (not a directory), so the OS cannot
// open the JSONL path inside it, covering the history.Load error branch.
//
//fusa:test REQ-FO-HST004
func TestHistoryAPIError(t *testing.T) {
	// Create a regular file and use its path as histDir so that
	// filepath.Join(histDir, history.Filename) is inside a non-directory, causing
	// os.Open to fail with "not a directory" (not IsNotExist → propagated error).
	histFile := filepath.Join(t.TempDir(), "hist-as-file")
	if err := os.WriteFile(histFile, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	reg := adapter.NewRegistry()
	s := New(t.TempDir(), orchestrator.New(reg), orchestrator.Options{}).WithHistoryDir(histFile)

	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/history", nil))
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("want 500, got %d", rec.Code)
	}
}

// ── MultiServer handleAPIBaseline ─────────────────────────────────────────────

// TestMultiBaselineMethodNotAllowed verifies GET /api/v1/baseline on MultiServer
// returns 405, covering the r.Method != POST branch.
//
//fusa:test REQ-FO-SRV008
func TestMultiBaselineMethodNotAllowed(t *testing.T) {
	bl := filepath.Join(t.TempDir(), "baseline.json")
	ms := newTestMulti(t).WithBaseline(bl)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/baseline", nil)
	rec := httptest.NewRecorder()
	ms.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("want 405, got %d", rec.Code)
	}
}

// TestMultiBaselineNoFindings verifies POST /api/v1/baseline on MultiServer
// returns 503 when all projects have scan errors (no findings to save),
// covering the findings == nil branch.
//
//fusa:test REQ-FO-SRV008
func TestMultiBaselineNoFindings(t *testing.T) {
	bl := filepath.Join(t.TempDir(), "baseline.json")
	ms := newTestMulti(t).WithBaseline(bl)
	// Inject errors into all projects so the findings loop skips them all.
	for _, p := range ms.projects {
		p.mu.Lock()
		p.err = errors.New("injected error")
		p.cached = nil
		p.mu.Unlock()
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/baseline", nil)
	rec := httptest.NewRecorder()
	ms.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("want 503, got %d", rec.Code)
	}
}

// ── handleAPIBaseline (server.go) cErr branch ─────────────────────────────────

// TestAPIBaselineWithError verifies POST /api/v1/baseline on a Server returns
// 500 when the server has a cached scan error, covering the cErr != nil branch.
//
//fusa:test REQ-FO-SRV008
func TestAPIBaselineWithError(t *testing.T) {
	bl := filepath.Join(t.TempDir(), "baseline.json")
	s := newTestServer(t).WithBaseline(bl)
	s.mu.Lock()
	s.err = errors.New("scan error")
	s.mu.Unlock()

	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/v1/baseline", nil))
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("want 500, got %d", rec.Code)
	}
}

// TestAPIReportJSON verifies /api/v1/report returns JSON for the existing test
// server, ensuring the json encode path is covered.
//
//fusa:test REQ-FO-SRV005
func TestAPIReportJSON(t *testing.T) {
	s := newTestServer(t)
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/report", nil))
	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rec.Code)
	}
	var out map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("api/v1/report not JSON: %v", err)
	}
}
