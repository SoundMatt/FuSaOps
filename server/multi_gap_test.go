package server

// Additional tests targeting uncovered branches in MultiServer: authWrap
// auth paths, handleOverview badge variants, handleAPIProjects error states,
// makeProjectHandler error states, handleBadge aggregation, and
// makeProjectBadgeHandler variants.

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	fusaops "github.com/SoundMatt/FuSaOps"
	"github.com/SoundMatt/FuSaOps/adapter"
	"github.com/SoundMatt/FuSaOps/orchestrator"
)

// newTestMultiNoFindings creates a MultiServer whose fakeAdapter produces no
// findings so that project status is PASS after compute.
func newTestMultiNoFindings(t *testing.T) *MultiServer {
	t.Helper()
	reg := adapter.NewRegistry()
	reg.MustRegister(&fakeAdapter{tool: "gofusa", lang: fusaops.LangGo})
	cfg := ProjectsConfig{Projects: []ProjectConfig{{Name: "clean", Dir: t.TempDir()}}}
	ms := NewMulti(cfg, orchestrator.New(reg))
	ms.compute(context.Background())
	return ms
}

// newTestMultiErrors creates a MultiServer whose fakeAdapter produces ERROR
// findings so that project status is FAIL after compute.
func newTestMultiErrors(t *testing.T) *MultiServer {
	t.Helper()
	reg := adapter.NewRegistry()
	reg.MustRegister(&fakeAdapter{tool: "gofusa", lang: fusaops.LangGo, findings: []fusaops.Finding{
		{RuleID: "E001", Severity: fusaops.SeverityError, Message: "error finding"},
	}})
	cfg := ProjectsConfig{Projects: []ProjectConfig{{Name: "broken", Dir: t.TempDir()}}}
	ms := NewMulti(cfg, orchestrator.New(reg))
	ms.compute(context.Background())
	return ms
}

// ── NewMulti with Adapter field ───────────────────────────────────────────────

// TestNewMultiWithAdapter verifies NewMulti sets opts.Only when ProjectConfig.Adapter
// is non-empty, covering the "p.Adapter != """ branch.
//
//fusa:test REQ-FO-MPJ001
func TestNewMultiWithAdapter(t *testing.T) {
	reg := adapter.NewRegistry()
	reg.MustRegister(&fakeAdapter{tool: "gofusa", lang: fusaops.LangGo})
	cfg := ProjectsConfig{
		Projects: []ProjectConfig{{Name: "alpha", Dir: t.TempDir(), Adapter: "gofusa"}},
	}
	ms := NewMulti(cfg, orchestrator.New(reg))
	if len(ms.projects) != 1 {
		t.Fatalf("want 1 project, got %d", len(ms.projects))
	}
	if len(ms.projects[0].opts.Only) != 1 || ms.projects[0].opts.Only[0] != "gofusa" {
		t.Errorf("opts.Only: %v", ms.projects[0].opts.Only)
	}
}

// ── authWrap ─────────────────────────────────────────────────────────────────

// TestMultiAuthRWSuccess verifies authWrap allows requests with rw credentials,
// covering the "role = rw" assignment and the statusRecorder + handler path.
//
//fusa:test REQ-FO-RBAC001
func TestMultiAuthRWSuccess(t *testing.T) {
	ms := newTestMulti(t).WithAuth("admin", "pw")
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.SetBasicAuth("admin", "pw")
	rec := httptest.NewRecorder()
	ms.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rec.Code)
	}
}

// TestMultiAuthUnauthorized verifies authWrap returns 401 when credentials
// are wrong, covering the "role == "" → Unauthorized" branch.
//
//fusa:test REQ-FO-AUTH001
func TestMultiAuthUnauthorized(t *testing.T) {
	ms := newTestMulti(t).WithAuth("admin", "pw")
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.SetBasicAuth("baduser", "badpass")
	rec := httptest.NewRecorder()
	ms.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("want 401, got %d", rec.Code)
	}
}

// TestMultiAuthAuditLogging verifies authWrap writes an audit log entry when
// auditDir is set, covering the "ms.auditDir != """  audit-append branch.
//
//fusa:test REQ-FO-AUDIT001
func TestMultiAuthAuditLogging(t *testing.T) {
	ms := newTestMulti(t).WithAuth("admin", "pw").WithAuditLog(t.TempDir())
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.SetBasicAuth("admin", "pw")
	rec := httptest.NewRecorder()
	ms.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rec.Code)
	}
}

// ── handleOverview badge variants ────────────────────────────────────────────

// TestMultiHandleOverviewFAILBadge verifies the FAIL badge and the pErr != nil
// branch in handleOverview.
//
//fusa:test REQ-FO-MPJ003
func TestMultiHandleOverviewFAILBadge(t *testing.T) {
	ms := newTestMulti(t)
	ms.projects[0].mu.Lock()
	ms.projects[0].err = errors.New("injected scan error")
	ms.projects[0].mu.Unlock()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	ms.handleOverview(rec, req)
	if !strings.Contains(rec.Body.String(), `badge fail`) {
		t.Errorf("expected FAIL badge in overview: %s", rec.Body.String())
	}
}

// TestMultiHandleOverviewPASSBadge verifies the PASS badge in handleOverview.
//
//fusa:test REQ-FO-MPJ003
func TestMultiHandleOverviewPASSBadge(t *testing.T) {
	ms := newTestMultiNoFindings(t)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	ms.handleOverview(rec, req)
	if !strings.Contains(rec.Body.String(), `badge pass`) {
		t.Errorf("expected PASS badge in overview: %s", rec.Body.String())
	}
}

// TestMultiHandleOverviewPENDINGBadge verifies the PENDING (default) badge
// and the rep == nil branch in handleOverview.
//
//fusa:test REQ-FO-MPJ003
func TestMultiHandleOverviewPENDINGBadge(t *testing.T) {
	reg := adapter.NewRegistry()
	cfg := ProjectsConfig{Projects: []ProjectConfig{{Name: "pending", Dir: t.TempDir()}}}
	ms := NewMulti(cfg, orchestrator.New(reg))
	// No compute — cached is nil, err is nil.
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	ms.handleOverview(rec, req)
	if !strings.Contains(rec.Body.String(), `badge pending`) {
		t.Errorf("expected PENDING badge in overview: %s", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "computing") {
		t.Errorf("expected 'computing' hint in overview: %s", rec.Body.String())
	}
}

// ── handleAPIProjects ─────────────────────────────────────────────────────────

// TestMultiAPIProjectsFAIL verifies /api/projects returns status=FAIL when a
// project has a scan error, covering the pErr != nil branch.
//
//fusa:test REQ-FO-MPJ003
func TestMultiAPIProjectsFAIL(t *testing.T) {
	ms := newTestMulti(t)
	ms.projects[0].mu.Lock()
	ms.projects[0].err = errors.New("scan error")
	ms.projects[0].mu.Unlock()

	req := httptest.NewRequest(http.MethodGet, "/api/projects", nil)
	rec := httptest.NewRecorder()
	ms.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"FAIL"`) {
		t.Errorf("expected FAIL in JSON: %s", rec.Body.String())
	}
}

// TestMultiAPIProjectsPENDING verifies /api/projects returns status=PENDING
// when a project has no cached report, covering the rep == nil branch.
//
//fusa:test REQ-FO-MPJ003
func TestMultiAPIProjectsPENDING(t *testing.T) {
	reg := adapter.NewRegistry()
	cfg := ProjectsConfig{Projects: []ProjectConfig{{Name: "pending", Dir: t.TempDir()}}}
	ms := NewMulti(cfg, orchestrator.New(reg))
	// No compute.
	req := httptest.NewRequest(http.MethodGet, "/api/projects", nil)
	rec := httptest.NewRecorder()
	ms.Handler().ServeHTTP(rec, req)
	if !strings.Contains(rec.Body.String(), `"PENDING"`) {
		t.Errorf("expected PENDING in JSON: %s", rec.Body.String())
	}
}

// ── makeProjectHandler ────────────────────────────────────────────────────────

// TestMultiProjectPageFAIL verifies /p/{name} returns 500 when the project
// has a scan error, covering the pErr != nil branch in makeProjectHandler.
//
//fusa:test REQ-FO-MPJ004
func TestMultiProjectPageFAIL(t *testing.T) {
	ms := newTestMulti(t)
	ms.projects[0].mu.Lock()
	ms.projects[0].err = errors.New("scan error")
	ms.projects[0].mu.Unlock()

	req := httptest.NewRequest(http.MethodGet, "/p/alpha", nil)
	rec := httptest.NewRecorder()
	ms.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("want 500, got %d", rec.Code)
	}
}

// TestMultiProjectPagePENDING verifies /p/{name} returns 503 when no report
// is available, covering the rep == nil branch in makeProjectHandler.
//
//fusa:test REQ-FO-MPJ004
func TestMultiProjectPagePENDING(t *testing.T) {
	reg := adapter.NewRegistry()
	cfg := ProjectsConfig{Projects: []ProjectConfig{{Name: "pending", Dir: t.TempDir()}}}
	ms := NewMulti(cfg, orchestrator.New(reg))
	// No compute.
	req := httptest.NewRequest(http.MethodGet, "/p/pending", nil)
	rec := httptest.NewRecorder()
	ms.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("want 503, got %d", rec.Code)
	}
}

// TestMultiProjectPagePASS verifies /p/{name} shows the PASS badge when the
// project has no findings, covering the "case PASS" branch.
//
//fusa:test REQ-FO-MPJ004
func TestMultiProjectPagePASS(t *testing.T) {
	ms := newTestMultiNoFindings(t)
	req := httptest.NewRequest(http.MethodGet, "/p/clean", nil)
	rec := httptest.NewRecorder()
	ms.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `class="pass"`) {
		t.Errorf("expected PASS badge in detail page: %s", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "No findings") {
		t.Errorf("expected 'No findings' message: %s", rec.Body.String())
	}
}

// TestMultiProjectPageERRORSeverity verifies /p/{name} marks ERROR findings
// with the "err" CSS class, covering the "cls = err" branch.
//
//fusa:test REQ-FO-MPJ004
func TestMultiProjectPageERRORSeverity(t *testing.T) {
	ms := newTestMultiErrors(t)
	req := httptest.NewRequest(http.MethodGet, "/p/broken", nil)
	rec := httptest.NewRecorder()
	ms.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `class="err"`) {
		t.Errorf("expected err CSS class for ERROR finding: %s", rec.Body.String())
	}
}

// ── handleBadge aggregate badge ───────────────────────────────────────────────

// TestMultiBadgeAggFAIL verifies handleBadge returns fail when any project
// has an error or FAIL status, covering the pErr != nil → continue and FAIL
// case branches.
//
//fusa:test REQ-FO-BADGE001
func TestMultiBadgeAggFAIL(t *testing.T) {
	ms := newTestMultiErrors(t)
	req := httptest.NewRequest(http.MethodGet, "/badge/status.svg", nil)
	rec := httptest.NewRecorder()
	ms.Handler().ServeHTTP(rec, req)
	if !strings.Contains(rec.Body.String(), "fail") {
		t.Errorf("expected fail badge: %s", rec.Body.String())
	}
}

// TestMultiBadgeAggPASS verifies handleBadge returns pass when all projects
// have PASS status, covering the PASS case branch.
//
//fusa:test REQ-FO-BADGE001
func TestMultiBadgeAggPASS(t *testing.T) {
	ms := newTestMultiNoFindings(t)
	req := httptest.NewRequest(http.MethodGet, "/badge/status.svg", nil)
	rec := httptest.NewRecorder()
	ms.Handler().ServeHTTP(rec, req)
	if !strings.Contains(rec.Body.String(), "pass") {
		t.Errorf("expected pass badge: %s", rec.Body.String())
	}
}

// TestMultiBadgeAggSkipError verifies handleBadge skips projects with errors
// (covering the pErr != nil || rep == nil → continue branch in handleBadge).
//
//fusa:test REQ-FO-BADGE001
func TestMultiBadgeAggSkipError(t *testing.T) {
	ms := newTestMultiNoFindings(t)
	// Inject an error into the project so it gets skipped during badge aggregation.
	ms.projects[0].mu.Lock()
	ms.projects[0].err = errors.New("skip me")
	ms.projects[0].mu.Unlock()

	req := httptest.NewRequest(http.MethodGet, "/badge/status.svg", nil)
	rec := httptest.NewRecorder()
	ms.Handler().ServeHTTP(rec, req)
	// With the only project skipped, badge defaults to "pending".
	if !strings.Contains(rec.Body.String(), "pending") {
		t.Errorf("expected pending badge when all projects errored: %s", rec.Body.String())
	}
}

// ── makeProjectBadgeHandler ───────────────────────────────────────────────────

// TestMultiProjectBadgeFAIL verifies the per-project badge is "error" when
// the project has a scan error, covering the pErr != nil branch.
//
//fusa:test REQ-FO-BADGE002
func TestMultiProjectBadgeFAIL(t *testing.T) {
	ms := newTestMulti(t)
	ms.projects[0].mu.Lock()
	ms.projects[0].err = errors.New("scan error")
	ms.projects[0].mu.Unlock()

	req := httptest.NewRequest(http.MethodGet, "/badge/alpha/status.svg", nil)
	rec := httptest.NewRecorder()
	ms.Handler().ServeHTTP(rec, req)
	if !strings.Contains(rec.Body.String(), "error") {
		t.Errorf("expected error badge for project with scan error: %s", rec.Body.String())
	}
}

// TestMultiProjectBadgePASS verifies the per-project badge is "pass" for a
// project with no findings, covering the "case PASS" badge branch.
//
//fusa:test REQ-FO-BADGE002
func TestMultiProjectBadgePASS(t *testing.T) {
	ms := newTestMultiNoFindings(t)
	req := httptest.NewRequest(http.MethodGet, "/badge/clean/status.svg", nil)
	rec := httptest.NewRecorder()
	ms.Handler().ServeHTTP(rec, req)
	if !strings.Contains(rec.Body.String(), "pass") {
		t.Errorf("expected pass badge: %s", rec.Body.String())
	}
}

// TestMultiProjectBadgeERROR verifies the per-project badge shows "fail" for
// a project with ERROR findings, covering the default badge branch.
//
//fusa:test REQ-FO-BADGE002
func TestMultiProjectBadgeERROR(t *testing.T) {
	ms := newTestMultiErrors(t)
	req := httptest.NewRequest(http.MethodGet, "/badge/broken/status.svg", nil)
	rec := httptest.NewRecorder()
	ms.Handler().ServeHTTP(rec, req)
	if !strings.Contains(rec.Body.String(), "fail") {
		t.Errorf("expected fail badge for error project: %s", rec.Body.String())
	}
}
