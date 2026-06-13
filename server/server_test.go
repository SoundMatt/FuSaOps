package server

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	fusaops "github.com/SoundMatt/FuSaOps"
	"github.com/SoundMatt/FuSaOps/adapter"
	"github.com/SoundMatt/FuSaOps/history"
	"github.com/SoundMatt/FuSaOps/orchestrator"
)

type fakeAdapter struct {
	tool     string
	lang     fusaops.Language
	findings []fusaops.Finding
}

func (f *fakeAdapter) Name() string                { return f.tool }
func (f *fakeAdapter) Language() fusaops.Language  { return f.lang }
func (f *fakeAdapter) Tool() string                { return f.tool }
func (f *fakeAdapter) Detect(string) (bool, error) { return true, nil }
func (f *fakeAdapter) Available() bool             { return true }
func (f *fakeAdapter) Check(context.Context, string) ([]fusaops.Finding, error) {
	return f.findings, nil
}

func newTestServer(t *testing.T) *Server {
	t.Helper()
	reg := adapter.NewRegistry()
	reg.MustRegister(&fakeAdapter{tool: "gofusa", lang: fusaops.LangGo, findings: []fusaops.Finding{
		{RuleID: "LINT001", Severity: fusaops.SeverityWarning, Message: "warn"},
	}})
	s := New(t.TempDir(), orchestrator.New(reg), orchestrator.Options{Project: "demo"})
	s.compute(context.Background())
	return s
}

//fusa:test REQ-FO-SRV001
//fusa:test REQ-FO-SRV002
func TestIndexServesDashboard(t *testing.T) {
	s := newTestServer(t)
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "<!DOCTYPE html>") {
		t.Error("index did not return HTML dashboard")
	}
}

//fusa:test REQ-FO-SRV004
func TestAPIReportServesJSON(t *testing.T) {
	s := newTestServer(t)
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/report", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	var out map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("api/report not JSON: %v", err)
	}
	if out["project"] != "demo" {
		t.Errorf("project: got %v", out["project"])
	}
}

func TestHealth(t *testing.T) {
	s := newTestServer(t)
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "ok") {
		t.Errorf("health check failed: %d %q", rec.Code, rec.Body.String())
	}
}

//fusa:test REQ-FO-SRV003
func TestRefreshRedirects(t *testing.T) {
	s := newTestServer(t)
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/refresh", nil))
	if rec.Code != http.StatusSeeOther {
		t.Errorf("refresh: got %d, want 303", rec.Code)
	}
}

func TestUnknownPath404(t *testing.T) {
	s := newTestServer(t)
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/nope", nil))
	if rec.Code != http.StatusNotFound {
		t.Errorf("got %d, want 404", rec.Code)
	}
}

//fusa:test REQ-FO-SRV005
func TestServe(t *testing.T) {
	reg := adapter.NewRegistry()
	reg.MustRegister(&fakeAdapter{tool: "gofusa", lang: fusaops.LangGo})
	s := New(t.TempDir(), orchestrator.New(reg), orchestrator.Options{Project: "demo"})

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	errc := make(chan error, 1)
	go func() { errc <- s.Serve(ln) }() // Serve computes the report then blocks

	url := "http://" + ln.Addr().String() + "/healthz"
	client := &http.Client{Timeout: time.Second}
	var resp *http.Response
	for i := 0; i < 200; i++ {
		req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
		resp, err = client.Do(req)
		if err == nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err != nil {
		t.Fatalf("server never came up: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("healthz status %d", resp.StatusCode)
	}

	_ = ln.Close() // closing the listener unblocks Serve
	if err := <-errc; err != nil && !strings.Contains(err.Error(), "closed") {
		t.Errorf("Serve returned unexpected error: %v", err)
	}
}

// TestListenAndServeBindError verifies ListenAndServe propagates bind errors.
//
//fusa:test REQ-FO-SRV005
func TestListenAndServeBindError(t *testing.T) {
	reg := adapter.NewRegistry()
	s := New(t.TempDir(), orchestrator.New(reg), orchestrator.Options{})
	err := s.ListenAndServe("!invalid-addr!")
	if err == nil {
		t.Fatal("expected bind error for invalid address")
	}
}

// TestAPIStatusPass verifies /api/v1/status returns PASS for a warning-only report.
//
//fusa:test REQ-FO-API001
func TestAPIStatusPass(t *testing.T) {
	s := newTestServer(t) // fakeAdapter produces one WARNING
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/status", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	var out map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("not JSON: %v", err)
	}
	if out["status"] != "WARN" {
		t.Errorf("status: got %v, want WARN", out["status"])
	}
	if w, _ := out["warnings"].(float64); w != 1 {
		t.Errorf("warnings: got %v", out["warnings"])
	}
}

// TestAPIStatusPending verifies /api/v1/status returns PENDING when no report yet.
//
//fusa:test REQ-FO-API001
func TestAPIStatusPending(t *testing.T) {
	reg := adapter.NewRegistry()
	s := New(t.TempDir(), orchestrator.New(reg), orchestrator.Options{})
	// do NOT call compute — no report available
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/status", nil))
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status %d, want 503", rec.Code)
	}
	var out map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("not JSON: %v", err)
	}
	if out["status"] != "PENDING" {
		t.Errorf("status: got %v, want PENDING", out["status"])
	}
}

// TestAPIFindingsNoFilter verifies /api/v1/findings returns all findings unfiltered.
//
//fusa:test REQ-FO-API002
func TestAPIFindingsNoFilter(t *testing.T) {
	s := newTestServer(t)
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/findings", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	var out []map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("not JSON: %v", err)
	}
	if len(out) != 1 {
		t.Errorf("want 1 finding, got %d", len(out))
	}
}

// TestAPIFindingsSeverityFilter verifies ?severity= filters correctly.
//
//fusa:test REQ-FO-API002
func TestAPIFindingsSeverityFilter(t *testing.T) {
	s := newTestServer(t) // one WARNING finding
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/findings?severity=ERROR", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	var out []map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("not JSON: %v", err)
	}
	if len(out) != 0 {
		t.Errorf("expected 0 ERROR findings (only WARNING exists), got %d", len(out))
	}
}

// TestAPIFindingsLanguageFilter verifies ?language= filters correctly.
//
//fusa:test REQ-FO-API002
func TestAPIFindingsLanguageFilter(t *testing.T) {
	s := newTestServer(t) // only Go findings
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/findings?language=java", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	var out []map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("not JSON: %v", err)
	}
	if len(out) != 0 {
		t.Errorf("expected 0 Java findings, got %d", len(out))
	}
}

// TestWithHistoryDir verifies WithHistoryDir sets the history directory.
//
//fusa:test REQ-FO-HST003
func TestWithHistoryDir(t *testing.T) {
	reg := adapter.NewRegistry()
	dir := t.TempDir()
	s := New(dir, orchestrator.New(reg), orchestrator.Options{}).WithHistoryDir(dir)
	if s.histDir != dir {
		t.Errorf("histDir: got %q, want %q", s.histDir, dir)
	}
}

// TestAPIHistoryEmpty verifies /api/history returns [] when no history stored.
//
//fusa:test REQ-FO-HST004
func TestAPIHistoryEmpty(t *testing.T) {
	s := newTestServer(t).WithHistoryDir(t.TempDir())
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/history", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	var out []any
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("not JSON: %v", err)
	}
	if len(out) != 0 {
		t.Errorf("want empty array, got %d items", len(out))
	}
}

// TestAPIHistoryReturnsSnapshots verifies stored snapshots are served as JSON.
//
//fusa:test REQ-FO-HST004
func TestAPIHistoryReturnsSnapshots(t *testing.T) {
	dir := t.TempDir()
	_ = history.Store(dir, history.Snapshot{RunAt: time.Now().UTC(), Status: "PASS", Total: 2})
	_ = history.Store(dir, history.Snapshot{RunAt: time.Now().UTC(), Status: "FAIL", Total: 5, Errors: 1})

	reg := adapter.NewRegistry()
	s := New(dir, orchestrator.New(reg), orchestrator.Options{}).WithHistoryDir(dir)

	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/history", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	var snaps []history.Snapshot
	if err := json.Unmarshal(rec.Body.Bytes(), &snaps); err != nil {
		t.Fatalf("not JSON: %v", err)
	}
	if len(snaps) != 2 {
		t.Fatalf("want 2 snapshots, got %d", len(snaps))
	}
	if snaps[1].Status != "FAIL" || snaps[1].Errors != 1 {
		t.Errorf("second snapshot: %+v", snaps[1])
	}
}

// TestHistoryPageHTML verifies /history returns an HTML page.
//
//fusa:test REQ-FO-HST004
func TestHistoryPageHTML(t *testing.T) {
	dir := t.TempDir()
	_ = history.Store(dir, history.Snapshot{RunAt: time.Now().UTC(), Status: "PASS", Total: 3, Warnings: 1})

	reg := adapter.NewRegistry()
	s := New(dir, orchestrator.New(reg), orchestrator.Options{}).WithHistoryDir(dir)

	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/history", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "<!DOCTYPE html>") {
		t.Error("response is not HTML")
	}
	if !strings.Contains(body, "PASS") {
		t.Error("PASS badge not found in history page")
	}
}

// TestHistoryPageNoHistDir verifies /history renders empty page when histDir unset.
//
//fusa:test REQ-FO-HST004
func TestHistoryPageNoHistDir(t *testing.T) {
	s := newTestServer(t) // histDir not set
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/history", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "No history yet") {
		t.Error("expected 'No history yet' message")
	}
}

// TestWithAuth verifies WithAuth sets credentials on the server.
//
//fusa:test REQ-FO-AUTH001
func TestWithAuth(t *testing.T) {
	s := newTestServer(t).WithAuth("admin", "secret")
	if s.authUser != "admin" || s.authPass != "secret" {
		t.Errorf("auth fields: user=%q pass=%q", s.authUser, s.authPass)
	}
}

// TestAuthMiddlewareBlocks verifies unauthenticated requests return 401.
//
//fusa:test REQ-FO-AUTH001
//fusa:test REQ-FO-AUTH002
func TestAuthMiddlewareBlocks(t *testing.T) {
	s := newTestServer(t).WithAuth("admin", "secret")
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", rec.Code)
	}
	if rec.Header().Get("WWW-Authenticate") == "" {
		t.Error("missing WWW-Authenticate header")
	}
}

// TestAuthMiddlewareAllows verifies valid credentials pass through.
//
//fusa:test REQ-FO-AUTH001
func TestAuthMiddlewareAllows(t *testing.T) {
	s := newTestServer(t).WithAuth("admin", "secret")
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	req.SetBasicAuth("admin", "secret")
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", rec.Code)
	}
}

// TestAuthWrongPassword verifies wrong password returns 401.
//
//fusa:test REQ-FO-AUTH002
func TestAuthWrongPassword(t *testing.T) {
	s := newTestServer(t).WithAuth("admin", "secret")
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	req.SetBasicAuth("admin", "wrong")
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", rec.Code)
	}
}

// TestWithFleetConfig verifies WithFleetConfig sets the fleet config path.
//
//fusa:test REQ-FO-FLT005
func TestWithFleetConfig(t *testing.T) {
	s := newTestServer(t).WithFleetConfig("/tmp/fleet.json")
	if s.fleetCfg != "/tmp/fleet.json" {
		t.Errorf("fleetCfg: got %q", s.fleetCfg)
	}
}

// TestFleetPageNoConfig verifies /fleet returns 404 when fleet is not configured.
//
//fusa:test REQ-FO-FLT005
func TestFleetPageNoConfig(t *testing.T) {
	s := newTestServer(t) // no fleet config
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/fleet", nil))
	if rec.Code != http.StatusNotFound {
		t.Errorf("want 404 when fleet not configured, got %d", rec.Code)
	}
}

// TestFleetPageHTML verifies /fleet returns an HTML page with repo names.
//
//fusa:test REQ-FO-FLT006
func TestFleetPageHTML(t *testing.T) {
	reg := adapter.NewRegistry()
	reg.MustRegister(&fakeAdapter{tool: "gofusa", lang: fusaops.LangGo})
	dir := t.TempDir()
	cfgData, err := json.Marshal(map[string]any{
		"project": "testfleet",
		"repos":   []map[string]string{{"name": "svc", "dir": dir}},
	})
	if err != nil {
		t.Fatal(err)
	}
	cfgPath := dir + "/fleet.json"
	if err := os.WriteFile(cfgPath, cfgData, 0o644); err != nil {
		t.Fatal(err)
	}

	s := New(dir, orchestrator.New(reg), orchestrator.Options{}).WithFleetConfig(cfgPath)
	s.compute(context.Background())

	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/fleet", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "<!DOCTYPE html>") {
		t.Error("not HTML")
	}
	if !strings.Contains(body, "svc") {
		t.Error("repo name 'svc' not found in fleet page")
	}
}

// TestAPIFleetJSON verifies /api/fleet returns JSON when fleet is configured.
//
//fusa:test REQ-FO-FLT006
func TestAPIFleetJSON(t *testing.T) {
	reg := adapter.NewRegistry()
	reg.MustRegister(&fakeAdapter{tool: "gofusa", lang: fusaops.LangGo})
	dir := t.TempDir()
	cfgData, err := json.Marshal(map[string]any{
		"project": "testfleet",
		"repos":   []map[string]string{{"name": "svc", "dir": dir}},
	})
	if err != nil {
		t.Fatal(err)
	}
	cfgPath := dir + "/fleet.json"
	if err := os.WriteFile(cfgPath, cfgData, 0o644); err != nil {
		t.Fatal(err)
	}

	s := New(dir, orchestrator.New(reg), orchestrator.Options{}).WithFleetConfig(cfgPath)
	s.compute(context.Background())

	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/fleet", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	var out map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("not JSON: %v", err)
	}
	if out["project"] != "testfleet" {
		t.Errorf("project: got %v", out["project"])
	}
}

// TestListenAndServeTLSBadCert verifies ListenAndServeTLS fails on a missing cert.
//
//fusa:test REQ-FO-TLS001
func TestListenAndServeTLSBadCert(t *testing.T) {
	reg := adapter.NewRegistry()
	s := New(t.TempDir(), orchestrator.New(reg), orchestrator.Options{})
	err := s.ListenAndServeTLS(":0", "/nonexistent/cert.pem", "/nonexistent/key.pem")
	if err == nil {
		t.Fatal("expected error for missing cert/key")
	}
}

// TestWithAuthRO verifies WithAuthRO sets read-only credentials.
//
//fusa:test REQ-FO-RBAC001
func TestWithAuthRO(t *testing.T) {
	s := newTestServer(t).WithAuthRO("viewer", "secret")
	if s.authROUser != "viewer" || s.authROPass != "secret" {
		t.Errorf("authRO fields: user=%q pass=%q", s.authROUser, s.authROPass)
	}
}

// TestAuthROCanReadDashboard verifies ro credentials can access read-only routes.
//
//fusa:test REQ-FO-RBAC001
func TestAuthROCanReadDashboard(t *testing.T) {
	s := newTestServer(t).WithAuth("admin", "secret").WithAuthRO("viewer", "ro-pass")
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	req.SetBasicAuth("viewer", "ro-pass")
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("ro user blocked from read route: %d", rec.Code)
	}
}

// TestAuthROBlockedFromRefresh verifies ro credentials cannot trigger /refresh.
//
//fusa:test REQ-FO-RBAC002
func TestAuthROBlockedFromRefresh(t *testing.T) {
	s := newTestServer(t).WithAuth("admin", "secret").WithAuthRO("viewer", "ro-pass")
	req := httptest.NewRequest(http.MethodGet, "/refresh", nil)
	req.SetBasicAuth("viewer", "ro-pass")
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("want 403 for ro on /refresh, got %d", rec.Code)
	}
}

// TestAuthROOnlyNoRW verifies ro-only mode (no rw set) blocks unauthenticated.
//
//fusa:test REQ-FO-RBAC001
func TestAuthROOnlyBlocksUnauthenticated(t *testing.T) {
	s := newTestServer(t).WithAuthRO("viewer", "ro-pass")
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", rec.Code)
	}
}

// TestWithAuditLog verifies WithAuditLog sets the audit directory.
//
//fusa:test REQ-FO-AUDIT001
func TestWithAuditLog(t *testing.T) {
	s := newTestServer(t).WithAuditLog("/tmp/audit")
	if s.auditDir != "/tmp/audit" {
		t.Errorf("auditDir: got %q", s.auditDir)
	}
}

// TestAuditLogWritten verifies authenticated requests are written to the audit log.
//
//fusa:test REQ-FO-AUDIT001
//fusa:test REQ-FO-AUDIT002
func TestAuditLogWritten(t *testing.T) {
	dir := t.TempDir()
	s := newTestServer(t).WithAuth("admin", "pw").WithAuditLog(dir)
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	req.SetBasicAuth("admin", "pw")
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	data, err := os.ReadFile(filepath.Join(dir, "fusaops-audit.jsonl"))
	if err != nil {
		t.Fatalf("audit log not written: %v", err)
	}
	if !strings.Contains(string(data), "/healthz") {
		t.Errorf("audit log missing /healthz: %s", data)
	}
	if !strings.Contains(string(data), `"user":"admin"`) {
		t.Errorf("audit log missing user: %s", data)
	}
}
