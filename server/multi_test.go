package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	fusaops "github.com/SoundMatt/FuSaOps"
	"github.com/SoundMatt/FuSaOps/adapter"
	"github.com/SoundMatt/FuSaOps/diff"
	"github.com/SoundMatt/FuSaOps/orchestrator"
)

func newTestMulti(t *testing.T) *MultiServer {
	t.Helper()
	reg := adapter.NewRegistry()
	reg.MustRegister(&fakeAdapter{tool: "gofusa", lang: fusaops.LangGo, findings: []fusaops.Finding{
		{RuleID: "LINT001", Severity: fusaops.SeverityWarning, Message: "warn"},
	}})
	cfg := ProjectsConfig{
		Projects: []ProjectConfig{
			{Name: "alpha", Dir: t.TempDir()},
			{Name: "beta", Dir: t.TempDir()},
		},
	}
	ms := NewMulti(cfg, orchestrator.New(reg))
	ms.compute(context.Background())
	return ms
}

// TestNewMulti verifies NewMulti creates project entries.
//
//fusa:test REQ-FO-MPJ001
func TestNewMulti(t *testing.T) {
	ms := newTestMulti(t)
	if len(ms.projects) != 2 {
		t.Fatalf("want 2 projects, got %d", len(ms.projects))
	}
	if ms.projects[0].name != "alpha" || ms.projects[1].name != "beta" {
		t.Errorf("project names: %q %q", ms.projects[0].name, ms.projects[1].name)
	}
}

// TestMultiComputeAll verifies all project reports are populated after compute.
//
//fusa:test REQ-FO-MPJ002
func TestMultiComputeAll(t *testing.T) {
	ms := newTestMulti(t)
	for _, p := range ms.projects {
		p.mu.RLock()
		rep, err := p.cached, p.err
		p.mu.RUnlock()
		if err != nil {
			t.Errorf("project %s: unexpected error: %v", p.name, err)
		}
		if rep == nil {
			t.Errorf("project %s: report is nil", p.name)
		}
	}
}

// TestMultiOverviewHTML verifies the / overview page returns HTML with all project names.
//
//fusa:test REQ-FO-MPJ003
func TestMultiOverviewHTML(t *testing.T) {
	ms := newTestMulti(t)
	rec := httptest.NewRecorder()
	ms.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "<!DOCTYPE html>") {
		t.Error("not HTML")
	}
	for _, name := range []string{"alpha", "beta"} {
		if !strings.Contains(body, name) {
			t.Errorf("project %q not found in overview", name)
		}
	}
}

// TestMultiAPIProjects verifies /api/projects returns JSON with all projects.
//
//fusa:test REQ-FO-MPJ003
func TestMultiAPIProjects(t *testing.T) {
	ms := newTestMulti(t)
	rec := httptest.NewRecorder()
	ms.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/projects", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	var out []map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("not JSON: %v", err)
	}
	if len(out) != 2 {
		t.Fatalf("want 2 projects in JSON, got %d", len(out))
	}
}

// TestMultiProjectDetailPage verifies /p/{name} returns per-project HTML.
//
//fusa:test REQ-FO-MPJ004
func TestMultiProjectDetailPage(t *testing.T) {
	ms := newTestMulti(t)
	rec := httptest.NewRecorder()
	ms.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/p/alpha", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "<!DOCTYPE html>") {
		t.Error("not HTML")
	}
	if !strings.Contains(body, "alpha") {
		t.Error("project name 'alpha' not found in detail page")
	}
}

// TestMultiUnknownProject verifies /p/{unknown} returns 404.
//
//fusa:test REQ-FO-MPJ004
func TestMultiUnknownProject(t *testing.T) {
	ms := newTestMulti(t)
	rec := httptest.NewRecorder()
	ms.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/p/unknown", nil))
	if rec.Code != http.StatusNotFound {
		t.Errorf("want 404 for unknown project, got %d", rec.Code)
	}
}

// TestMultiHealth verifies /healthz returns 200 ok.
//
//fusa:test REQ-FO-MPJ001
func TestMultiHealth(t *testing.T) {
	ms := newTestMulti(t)
	rec := httptest.NewRecorder()
	ms.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "ok") {
		t.Errorf("health: %d %q", rec.Code, rec.Body.String())
	}
}

// TestMultiAuthBlocks verifies unauthenticated requests are rejected.
//
//fusa:test REQ-FO-AUTH001
func TestMultiAuthBlocks(t *testing.T) {
	ms := newTestMulti(t).WithAuth("admin", "pw")
	rec := httptest.NewRecorder()
	ms.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", rec.Code)
	}
}

// TestMultiROBlocksRefresh verifies ro credentials cannot trigger /refresh.
//
//fusa:test REQ-FO-RBAC002
func TestMultiROBlocksRefresh(t *testing.T) {
	ms := newTestMulti(t).WithAuth("admin", "pw").WithAuthRO("viewer", "ro")
	req := httptest.NewRequest(http.MethodGet, "/refresh", nil)
	req.SetBasicAuth("viewer", "ro")
	rec := httptest.NewRecorder()
	ms.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("want 403, got %d", rec.Code)
	}
}

// TestMultiBadgeSVG verifies /badge/status.svg returns SVG on MultiServer.
//
//fusa:test REQ-FO-BADGE001
//fusa:test REQ-FO-MPJ003
func TestMultiBadgeSVG(t *testing.T) {
	ms := newTestMulti(t)
	req := httptest.NewRequest(http.MethodGet, "/badge/status.svg", nil)
	rec := httptest.NewRecorder()
	ms.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "image/svg+xml" {
		t.Errorf("Content-Type: got %q", ct)
	}
	if !strings.Contains(rec.Body.String(), "<svg") {
		t.Errorf("not SVG: %s", rec.Body.String())
	}
}

// TestMultiProjectBadge verifies /badge/{name}/status.svg uses the project name.
//
//fusa:test REQ-FO-BADGE002
//fusa:test REQ-FO-MPJ003
func TestMultiProjectBadge(t *testing.T) {
	ms := newTestMulti(t)
	req := httptest.NewRequest(http.MethodGet, "/badge/alpha/status.svg", nil)
	rec := httptest.NewRecorder()
	ms.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "alpha") {
		t.Errorf("badge missing project name 'alpha': %s", body)
	}
}

// TestMultiWithAuditLog verifies WithAuditLog sets the audit directory.
//
//fusa:test REQ-FO-AUDIT001
func TestMultiWithAuditLog(t *testing.T) {
	ms := newTestMulti(t).WithAuditLog("/tmp/audit")
	if ms.auditDir != "/tmp/audit" {
		t.Errorf("auditDir: got %q", ms.auditDir)
	}
}

// TestMultiMetrics verifies /metrics returns per-project OpenMetrics output.
//
//fusa:test REQ-FO-MTR001
//fusa:test REQ-FO-MTR002
func TestMultiMetrics(t *testing.T) {
	ms := newTestMulti(t)
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	ms.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `project="alpha"`) {
		t.Errorf("metrics missing project=alpha: %s", body)
	}
	if !strings.Contains(body, `project="beta"`) {
		t.Errorf("metrics missing project=beta: %s", body)
	}
	if !strings.Contains(body, "fusaops_findings_total") {
		t.Errorf("metrics missing fusaops_findings_total: %s", body)
	}
}

// TestMultiWithRefreshInterval verifies WithRefreshInterval sets the interval.
//
//fusa:test REQ-FO-SCHD001
func TestMultiWithRefreshInterval(t *testing.T) {
	ms := newTestMulti(t).WithRefreshInterval(5 * time.Minute)
	if ms.refreshInterval != 5*time.Minute {
		t.Errorf("refreshInterval: got %v", ms.refreshInterval)
	}
}

// TestMultiRefreshAll verifies /refresh redirects to / after recomputing.
//
//fusa:test REQ-FO-MPJ002
func TestMultiRefreshAll(t *testing.T) {
	ms := newTestMulti(t)
	req := httptest.NewRequest(http.MethodGet, "/refresh", nil)
	rec := httptest.NewRecorder()
	ms.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("want 303, got %d", rec.Code)
	}
	if loc := rec.Header().Get("Location"); loc != "/" {
		t.Errorf("Location: got %q, want /", loc)
	}
}

// TestMultiExportJSON verifies /api/v1/export merges all projects and returns JSON.
//
//fusa:test REQ-FO-SRV006
func TestMultiExportJSON(t *testing.T) {
	ms := newTestMulti(t)
	rec := httptest.NewRecorder()
	ms.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/export", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200", rec.Code)
	}
	ct := rec.Header().Get("Content-Type")
	if !strings.Contains(ct, "application/json") {
		t.Errorf("Content-Type: got %q, want application/json", ct)
	}
	cd := rec.Header().Get("Content-Disposition")
	if !strings.Contains(cd, "fusaops-report.json") {
		t.Errorf("Content-Disposition: got %q", cd)
	}
}

// TestMultiExportCSV verifies /api/v1/export?format=csv returns CSV with header.
//
//fusa:test REQ-FO-SRV006
func TestMultiExportCSV(t *testing.T) {
	ms := newTestMulti(t)
	rec := httptest.NewRecorder()
	ms.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/export?format=csv", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "text/csv") {
		t.Errorf("Content-Type: got %q", ct)
	}
	if body := rec.Body.String(); !strings.Contains(body, "language") {
		t.Errorf("CSV body missing header: %q", body)
	}
}

// TestMultiWithBaseline verifies WithBaseline sets the baseline path on MultiServer.
//
//fusa:test REQ-FO-SRV007
func TestMultiWithBaseline(t *testing.T) {
	ms := newTestMulti(t).WithBaseline("/tmp/bl.json")
	if ms.baselineFile != "/tmp/bl.json" {
		t.Errorf("baselineFile: got %q", ms.baselineFile)
	}
}

// TestMultiAPIDiffNoBaseline verifies /api/v1/diff returns 400 when no baseline configured.
//
//fusa:test REQ-FO-SRV007
func TestMultiAPIDiffNoBaseline(t *testing.T) {
	ms := newTestMulti(t)
	rec := httptest.NewRecorder()
	ms.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/diff", nil))
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status: got %d, want 400", rec.Code)
	}
}

// TestMultiAPIDiffWithBaseline verifies /api/v1/diff returns JSON given a valid baseline.
//
//fusa:test REQ-FO-SRV007
func TestMultiAPIDiffWithBaseline(t *testing.T) {
	ms := newTestMulti(t)
	bl := filepath.Join(t.TempDir(), "baseline.json")
	if err := diff.SaveBaseline(bl, []fusaops.Finding{}); err != nil {
		t.Fatalf("save baseline: %v", err)
	}
	rec := httptest.NewRecorder()
	ms.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/diff?baseline="+bl, nil))
	if rec.Code != http.StatusOK {
		t.Errorf("status: got %d, want 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "application/json") {
		t.Errorf("Content-Type: %q", ct)
	}
}

// TestMultiAPIBaselineNoPath verifies POST /api/v1/baseline returns 400 when not configured.
//
//fusa:test REQ-FO-SRV008
func TestMultiAPIBaselineNoPath(t *testing.T) {
	ms := newTestMulti(t)
	rec := httptest.NewRecorder()
	ms.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/v1/baseline", nil))
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status: got %d, want 400", rec.Code)
	}
}

// TestMultiAPIBaselineSave verifies POST /api/v1/baseline saves the baseline file.
//
//fusa:test REQ-FO-SRV008
func TestMultiAPIBaselineSave(t *testing.T) {
	bl := filepath.Join(t.TempDir(), "baseline.json")
	ms := newTestMulti(t).WithBaseline(bl)
	rec := httptest.NewRecorder()
	ms.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/v1/baseline", nil))
	if rec.Code != http.StatusOK {
		t.Errorf("status: got %d, want 200", rec.Code)
	}
	if _, err := os.Stat(bl); err != nil {
		t.Errorf("baseline file not created: %v", err)
	}
}

// TestProjectConfigSuppression verifies per-project suppression field is wired into opts.
//
//fusa:test REQ-FO-MPJ005
func TestProjectConfigSuppression(t *testing.T) {
	reg := adapter.NewRegistry()
	cfg := ProjectsConfig{
		Projects: []ProjectConfig{
			{Name: "alpha", Dir: t.TempDir(), Suppression: "/tmp/suppress.json"},
		},
	}
	ms := NewMulti(cfg, orchestrator.New(reg))
	if ms.projects[0].opts.SuppressFile != "/tmp/suppress.json" {
		t.Errorf("SuppressFile: got %q", ms.projects[0].opts.SuppressFile)
	}
}

// TestProjectConfigAutoLoad verifies .fusaops.json is auto-loaded from project dir.
//
//fusa:test REQ-FO-MPJ006
func TestProjectConfigAutoLoad(t *testing.T) {
	dir := t.TempDir()
	cfg := `{"version":"1","project":{"name":"myproj"},"scan":{"adapters":["gofusa"]},"report":{"format":"text"}}`
	if err := os.WriteFile(filepath.Join(dir, ".fusaops.json"), []byte(cfg), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	reg := adapter.NewRegistry()
	ms := NewMulti(ProjectsConfig{Projects: []ProjectConfig{{Name: "alpha", Dir: dir}}}, orchestrator.New(reg))
	if ms.projects[0].opts.Project != "myproj" {
		t.Errorf("project name override: got %q, want myproj", ms.projects[0].opts.Project)
	}
	if len(ms.projects[0].opts.Only) == 0 || ms.projects[0].opts.Only[0] != "gofusa" {
		t.Errorf("adapter filter: got %v", ms.projects[0].opts.Only)
	}
}

// TestValidateProjectDirsOK verifies ValidateProjectDirs returns nil for valid dirs.
//
//fusa:test REQ-FO-MPJ007
func TestValidateProjectDirsOK(t *testing.T) {
	ms := newTestMulti(t)
	if errs := ms.ValidateProjectDirs(); len(errs) > 0 {
		t.Errorf("unexpected errors: %v", errs)
	}
}

// TestValidateProjectDirsMissing verifies ValidateProjectDirs returns errors for missing dirs.
//
//fusa:test REQ-FO-MPJ007
func TestValidateProjectDirsMissing(t *testing.T) {
	reg := adapter.NewRegistry()
	ms := NewMulti(ProjectsConfig{Projects: []ProjectConfig{
		{Name: "ghost", Dir: "/nonexistent/path/xyz"},
	}}, orchestrator.New(reg))
	errs := ms.ValidateProjectDirs()
	if len(errs) == 0 {
		t.Error("expected error for missing dir, got none")
	}
}

// TestMultiAPIDiffProjectFilter verifies ?project=name restricts diff to one project.
//
//fusa:test REQ-FO-SRV009
func TestMultiAPIDiffProjectFilter(t *testing.T) {
	ms := newTestMulti(t)
	bl := filepath.Join(t.TempDir(), "baseline.json")
	if err := diff.SaveBaseline(bl, []fusaops.Finding{}); err != nil {
		t.Fatalf("save baseline: %v", err)
	}
	rec := httptest.NewRecorder()
	ms.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/diff?baseline="+bl+"&project=alpha", nil))
	if rec.Code != http.StatusOK {
		t.Errorf("status: got %d, want 200 (project filter)", rec.Code)
	}
}

// TestMultiAPIDiffUnknownProject verifies ?project=unknown returns 503 (no components).
//
//fusa:test REQ-FO-SRV009
func TestMultiAPIDiffUnknownProject(t *testing.T) {
	ms := newTestMulti(t)
	bl := filepath.Join(t.TempDir(), "baseline.json")
	if err := diff.SaveBaseline(bl, []fusaops.Finding{}); err != nil {
		t.Fatalf("save baseline: %v", err)
	}
	rec := httptest.NewRecorder()
	ms.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/diff?baseline="+bl+"&project=unknown", nil))
	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("status: got %d, want 503 (unknown project)", rec.Code)
	}
}
