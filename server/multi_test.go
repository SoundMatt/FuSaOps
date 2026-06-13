package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	fusaops "github.com/SoundMatt/FuSaOps"
	"github.com/SoundMatt/FuSaOps/adapter"
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
