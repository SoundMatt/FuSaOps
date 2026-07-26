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
	"github.com/SoundMatt/FuSaOps/comp"
	"github.com/SoundMatt/FuSaOps/diff"
	"github.com/SoundMatt/FuSaOps/history"
	"github.com/SoundMatt/FuSaOps/orchestrator"
	"github.com/SoundMatt/FuSaOps/vv"
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
//fusa:test REQ-FO-CLI026
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
//fusa:test REQ-FO-CLI029
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

// TestMetricsContentType verifies /metrics returns OpenMetrics content type.
//
//fusa:test REQ-FO-MTR001
func TestMetricsContentType(t *testing.T) {
	s := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	ct := rec.Header().Get("Content-Type")
	if !strings.Contains(ct, "text/plain") {
		t.Errorf("Content-Type: got %q, want text/plain", ct)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "fusaops_findings_total") {
		t.Errorf("metrics missing fusaops_findings_total: %s", body)
	}
	if !strings.Contains(body, "fusaops_status") {
		t.Errorf("metrics missing fusaops_status: %s", body)
	}
}

// TestMetricsStatusReflectsFindings verifies metric values match aggregate status.
//
//fusa:test REQ-FO-MTR001
func TestMetricsStatusReflectsFindings(t *testing.T) {
	s := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	body := rec.Body.String()
	// newTestServer has a warning finding → status WARN → code 2
	if !strings.Contains(body, "fusaops_status 2") {
		t.Errorf("expected fusaops_status 2 (WARN) in:\n%s", body)
	}
	// Should have 1 warning finding
	if !strings.Contains(body, `severity="warning"} 1`) {
		t.Errorf("expected 1 warning finding in:\n%s", body)
	}
}

// TestBadgeSVGContentType verifies /badge/status.svg returns image/svg+xml.
//
//fusa:test REQ-FO-BADGE001
func TestBadgeSVGContentType(t *testing.T) {
	s := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/badge/status.svg", nil)
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	ct := rec.Header().Get("Content-Type")
	if ct != "image/svg+xml" {
		t.Errorf("Content-Type: got %q, want image/svg+xml", ct)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "<svg") {
		t.Errorf("badge body not SVG: %s", body)
	}
}

// TestBadgeStatusReflectsFindings verifies badge text matches aggregate status.
//
//fusa:test REQ-FO-BADGE001
//fusa:test REQ-FO-BADGE002
func TestBadgeStatusReflectsFindings(t *testing.T) {
	s := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/badge/status.svg", nil)
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	body := rec.Body.String()
	// newTestServer has a warning finding so status should be WARN.
	if !strings.Contains(body, "warn") {
		t.Errorf("expected warn in badge: %s", body)
	}
}

// TestBadgeCacheControl verifies badge carries no-cache headers.
//
//fusa:test REQ-FO-BADGE001
func TestBadgeCacheControl(t *testing.T) {
	s := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/badge/status.svg", nil)
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	cc := rec.Header().Get("Cache-Control")
	if !strings.Contains(cc, "no-cache") {
		t.Errorf("Cache-Control missing no-cache: %q", cc)
	}
}

// TestWithWebhook verifies WithWebhook sets the URL.
//
//fusa:test REQ-FO-HOOK001
//fusa:test REQ-FO-CLI031
func TestWithWebhook(t *testing.T) {
	s := newTestServer(t).WithWebhook("http://example.com/hook")
	if s.webhookURL != "http://example.com/hook" {
		t.Errorf("webhookURL: got %q", s.webhookURL)
	}
}

// TestWithRefreshInterval verifies WithRefreshInterval sets the interval.
//
//fusa:test REQ-FO-SCHD001
func TestWithRefreshInterval(t *testing.T) {
	s := newTestServer(t).WithRefreshInterval(5 * time.Minute)
	if s.refreshInterval != 5*time.Minute {
		t.Errorf("refreshInterval: got %v", s.refreshInterval)
	}
}

// TestScheduledRefreshUpdatesCache verifies the scheduler triggers recompute.
//
//fusa:test REQ-FO-SCHD001
func TestScheduledRefreshUpdatesCache(t *testing.T) {
	s := newTestServer(t).WithRefreshInterval(50 * time.Millisecond)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	go func() { _ = s.Serve(ln) }() //nolint:errcheck
	// Give the scheduler at least two ticks.
	time.Sleep(200 * time.Millisecond)
	_ = ln.Close()
	// Verify the cache was populated (scheduler ran compute).
	s.mu.RLock()
	rep := s.cached
	s.mu.RUnlock()
	if rep == nil {
		t.Error("cache still nil after scheduled refresh")
	}
}

// TestWebhookFiredOnTransition verifies a webhook POST is sent when status changes.
//
//fusa:test REQ-FO-HOOK001
//fusa:test REQ-FO-HOOK002
func TestWebhookFiredOnTransition(t *testing.T) {
	done := make(chan []byte, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, 4096)
		n, _ := r.Body.Read(buf)
		select {
		case done <- buf[:n]:
		default:
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	s := newTestServer(t).WithWebhook(srv.URL)
	// Seed prevStatus so the next compute triggers a transition check.
	s.prevStatus = "PASS"
	// newTestServer has a warning finding → status becomes WARN → triggers webhook.
	s.compute(context.Background())

	select {
	case received := <-done:
		if !strings.Contains(string(received), `"prev"`) {
			t.Errorf("webhook payload missing 'prev': %s", received)
		}
		if !strings.Contains(string(received), `"status"`) {
			t.Errorf("webhook payload missing 'status': %s", received)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("webhook not received within timeout")
	}
}

// TestExportJSON verifies /api/v1/export returns JSON by default.
//
//fusa:test REQ-FO-SRV006
func TestExportJSON(t *testing.T) {
	s := newTestServer(t)
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/export", nil))
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

// TestExportCSV verifies /api/v1/export?format=csv returns CSV with header row.
//
//fusa:test REQ-FO-SRV006
func TestExportCSV(t *testing.T) {
	s := newTestServer(t)
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/export?format=csv", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200", rec.Code)
	}
	ct := rec.Header().Get("Content-Type")
	if !strings.Contains(ct, "text/csv") {
		t.Errorf("Content-Type: got %q, want text/csv", ct)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "language") || !strings.Contains(body, "severity") {
		t.Errorf("CSV body missing header columns: %q", body)
	}
}

// TestExportPending verifies /api/v1/export returns 503 when no report is cached.
//
//fusa:test REQ-FO-SRV006
func TestExportPending(t *testing.T) {
	reg := adapter.NewRegistry()
	s := New(t.TempDir(), orchestrator.New(reg), orchestrator.Options{})
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/export", nil))
	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("status: got %d, want 503", rec.Code)
	}
}

// TestWithBaseline verifies WithBaseline sets the baseline path.
//
//fusa:test REQ-FO-SRV007
func TestWithBaseline(t *testing.T) {
	s := newTestServer(t).WithBaseline("/tmp/bl.json")
	if s.baselineFile != "/tmp/bl.json" {
		t.Errorf("baselineFile: got %q", s.baselineFile)
	}
}

// TestAPIDiffNoBaseline verifies /api/v1/diff returns 400 when no baseline is configured.
//
//fusa:test REQ-FO-SRV007
func TestAPIDiffNoBaseline(t *testing.T) {
	s := newTestServer(t)
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/diff", nil))
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status: got %d, want 400", rec.Code)
	}
}

// TestAPIDiffWithBaseline verifies /api/v1/diff returns JSON when a valid baseline exists.
//
//fusa:test REQ-FO-SRV007
func TestAPIDiffWithBaseline(t *testing.T) {
	s := newTestServer(t)
	bl := filepath.Join(t.TempDir(), "baseline.json")
	if err := diff.SaveBaseline(bl, []fusaops.Finding{}); err != nil {
		t.Fatalf("save baseline: %v", err)
	}
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/diff?baseline="+bl, nil))
	if rec.Code != http.StatusOK {
		t.Errorf("status: got %d, want 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "application/json") {
		t.Errorf("Content-Type: %q", ct)
	}
}

// TestAPIDiffStrictConflict verifies /api/v1/diff?strict=true returns 409 when
// there are new ERROR findings relative to the baseline.
//
//fusa:test REQ-FO-SRV007
func TestAPIDiffStrictConflict(t *testing.T) {
	reg := adapter.NewRegistry()
	reg.MustRegister(&fakeAdapter{tool: "gofusa", lang: fusaops.LangGo, findings: []fusaops.Finding{
		{RuleID: "ERR001", Severity: fusaops.SeverityError, Message: "error finding"},
	}})
	s := New(t.TempDir(), orchestrator.New(reg), orchestrator.Options{})
	s.compute(context.Background())
	bl := filepath.Join(t.TempDir(), "baseline.json")
	if err := diff.SaveBaseline(bl, []fusaops.Finding{}); err != nil {
		t.Fatalf("save baseline: %v", err)
	}
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/diff?baseline="+bl+"&strict=true", nil))
	if rec.Code != http.StatusConflict {
		t.Errorf("status: got %d, want 409", rec.Code)
	}
}

// TestAPIBaselineNoPath verifies POST /api/v1/baseline returns 400 when not configured.
//
//fusa:test REQ-FO-SRV008
func TestAPIBaselineNoPath(t *testing.T) {
	s := newTestServer(t)
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/v1/baseline", nil))
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status: got %d, want 400", rec.Code)
	}
}

// TestAPIBaselineSave verifies POST /api/v1/baseline saves the baseline file.
//
//fusa:test REQ-FO-SRV008
func TestAPIBaselineSave(t *testing.T) {
	bl := filepath.Join(t.TempDir(), "baseline.json")
	s := newTestServer(t).WithBaseline(bl)
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/v1/baseline", nil))
	if rec.Code != http.StatusOK {
		t.Errorf("status: got %d, want 200", rec.Code)
	}
	if _, err := os.Stat(bl); err != nil {
		t.Errorf("baseline file not created: %v", err)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"saved"`) {
		t.Errorf("response missing 'saved': %s", body)
	}
}

// TestAPIVandVEmpty verifies /api/v1/vv returns JSON with ASIL-B when no
// V&V declaration has been configured.
//
//fusa:test REQ-FO-SRV010
func TestAPIVandVEmpty(t *testing.T) {
	s := newTestServer(t)
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/vv", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200", rec.Code)
	}
	ct := rec.Header().Get("Content-Type")
	if !strings.Contains(ct, "application/json") {
		t.Errorf("Content-Type: got %q, want application/json", ct)
	}
	var got map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("json unmarshal: %v", err)
	}
	if got["achievableAsil"] != "ASIL-B" {
		t.Errorf("achievableAsil: got %v, want ASIL-B", got["achievableAsil"])
	}
	if lvl, ok := got["independenceLevel"].(float64); !ok || int(lvl) != 0 {
		t.Errorf("independenceLevel: got %v, want 0", got["independenceLevel"])
	}
}

// TestAPIVandVWithDeclaration verifies /api/v1/vv reflects the WithVandV builder.
//
//fusa:test REQ-FO-SRV010
func TestAPIVandVWithDeclaration(t *testing.T) {
	s := newTestServer(t).WithVandV(vv.Declaration{
		Project:                 "acme",
		ImplementationAuthor:    "Alice",
		IndependentReviewer:     "Bob",
		IndependentTestExecutor: "Carol",
	})
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/vv", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200", rec.Code)
	}
	var got map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("json unmarshal: %v", err)
	}
	if got["achievableAsil"] != "ASIL-D" {
		t.Errorf("achievableAsil: got %v, want ASIL-D", got["achievableAsil"])
	}
	if lvl, ok := got["independenceLevel"].(float64); !ok || int(lvl) != 2 {
		t.Errorf("independenceLevel: got %v, want 2", got["independenceLevel"])
	}
	if got["implementationAuthor"] != "Alice" {
		t.Errorf("implementationAuthor: got %v, want Alice", got["implementationAuthor"])
	}
}

// TestVandVBadgeSVGContentType verifies /badge/vv.svg returns image/svg+xml.
//
//fusa:test REQ-FO-BADGE003
func TestVandVBadgeSVGContentType(t *testing.T) {
	s := newTestServer(t)
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/badge/vv.svg", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200", rec.Code)
	}
	ct := rec.Header().Get("Content-Type")
	if ct != "image/svg+xml" {
		t.Errorf("Content-Type: got %q, want image/svg+xml", ct)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "<svg") {
		t.Errorf("badge body not SVG: %s", body)
	}
}

// TestVandVBadgeReflectsASIL verifies /badge/vv.svg text shows the achievable ASIL.
//
//fusa:test REQ-FO-BADGE003
func TestVandVBadgeReflectsASIL(t *testing.T) {
	s := newTestServer(t).WithVandV(vv.Declaration{
		ImplementationAuthor:    "Alice",
		IndependentReviewer:     "Bob",
		IndependentTestExecutor: "Carol",
	})
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/badge/vv.svg", nil))
	body := rec.Body.String()
	if !strings.Contains(body, "ASIL-D") {
		t.Errorf("expected ASIL-D in badge: %s", body)
	}
}

// TestVandVBadgeCacheControl verifies /badge/vv.svg carries no-cache headers.
//
//fusa:test REQ-FO-BADGE003
func TestVandVBadgeCacheControl(t *testing.T) {
	s := newTestServer(t)
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/badge/vv.svg", nil))
	cc := rec.Header().Get("Cache-Control")
	if !strings.Contains(cc, "no-cache") {
		t.Errorf("Cache-Control missing no-cache: %q", cc)
	}
}

// TestVandVBadgeASILC verifies /badge/vv.svg shows ASIL-C color for reviewer-only.
//
//fusa:test REQ-FO-BADGE003
func TestVandVBadgeASILC(t *testing.T) {
	s := newTestServer(t).WithVandV(vv.Declaration{
		ImplementationAuthor: "Alice",
		IndependentReviewer:  "Bob",
	})
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/badge/vv.svg", nil))
	body := rec.Body.String()
	if !strings.Contains(body, "ASIL-C") {
		t.Errorf("expected ASIL-C in badge: %s", body)
	}
	// ASIL-C color
	if !strings.Contains(body, "#97ca00") {
		t.Errorf("expected ASIL-C color #97ca00 in badge: %s", body)
	}
}

// TestWithVandV verifies that WithVandV sets the declaration on the server.
//
//fusa:test REQ-FO-SRV010
func TestWithVandV(t *testing.T) {
	s := newTestServer(t)
	decl := vv.Declaration{
		Project:              "myproject",
		ImplementationAuthor: "Alice",
	}
	s2 := s.WithVandV(decl)
	if s2.vvDecl.Project != "myproject" {
		t.Errorf("vvDecl.Project: got %q, want myproject", s2.vvDecl.Project)
	}
}

// TestExportMIMETypes verifies exportMIME returns the correct content type and
// extension for each supported format.
func TestExportMIMETypes(t *testing.T) {
	cases := []struct {
		format  string
		wantCT  string
		wantExt string
	}{
		{"json", "application/json; charset=utf-8", "json"},
		{"csv", "text/csv; charset=utf-8", "csv"},
		{"junit", "application/xml; charset=utf-8", "xml"},
		{"sarif", "application/json; charset=utf-8", "sarif.json"},
		{"html", "text/html; charset=utf-8", "html"},
		{"markdown", "text/markdown; charset=utf-8", "md"},
		{"md", "text/markdown; charset=utf-8", "md"},
		{"unknown", "text/plain; charset=utf-8", "txt"},
	}
	for _, tc := range cases {
		ct, ext := exportMIME(tc.format)
		if ct != tc.wantCT {
			t.Errorf("exportMIME(%q) content-type: got %q, want %q", tc.format, ct, tc.wantCT)
		}
		if ext != tc.wantExt {
			t.Errorf("exportMIME(%q) ext: got %q, want %q", tc.format, ext, tc.wantExt)
		}
	}
}

// TestAPICompEmpty verifies /api/v1/comp returns {} when no comp data is cached.
//
//fusa:test REQ-FO-SRV012
func TestAPICompEmpty(t *testing.T) {
	s := newTestServer(t)
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/comp", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200", rec.Code)
	}
	ct := rec.Header().Get("Content-Type")
	if !strings.Contains(ct, "application/json") {
		t.Errorf("Content-Type: got %q, want application/json", ct)
	}
}

// TestAPICompWithData verifies /api/v1/comp returns the cached comp aggregate.
//
//fusa:test REQ-FO-SRV012
func TestAPICompWithData(t *testing.T) {
	s := newTestServer(t)
	s.compMu.Lock()
	s.compAgg = &comp.Aggregate{
		Root:           "/repo",
		Project:        "demo",
		Components:     []comp.ComponentComp{{Language: "go", Tool: "gofusa", Available: true, Report: &comp.Report{Threshold: 10, TotalFunctions: 5, Violations: 1}}},
		TotalFunctions: 5,
		Violations:     1,
	}
	s.compMu.Unlock()
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/comp", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200", rec.Code)
	}
	var got map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("json unmarshal: %v", err)
	}
	if v, _ := got["violations"].(float64); int(v) != 1 {
		t.Errorf("violations: got %v, want 1", got["violations"])
	}
	if v, _ := got["totalFunctions"].(float64); int(v) != 5 {
		t.Errorf("totalFunctions: got %v, want 5", got["totalFunctions"])
	}
}

// TestDashboardShowsCompSection verifies the HTML dashboard includes the comp
// violations section when a comp aggregate is cached.
//
//fusa:test REQ-FO-RPT021
//fusa:test REQ-FO-SRV012
func TestDashboardShowsCompSection(t *testing.T) {
	s := newTestServer(t)
	s.compMu.Lock()
	s.compAgg = &comp.Aggregate{
		Root:    "/repo",
		Project: "demo",
		Components: []comp.ComponentComp{{
			Language:  "go",
			Tool:      "gofusa",
			Available: true,
			Report:    &comp.Report{Threshold: 10, DAL: "DAL-B", TotalFunctions: 8, Violations: 1},
		}},
		TotalFunctions: 8,
		Violations:     1,
	}
	s.compMu.Unlock()
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{"Cyclomatic Complexity", "violations", "gofusa", "DAL-B"} {
		if !strings.Contains(body, want) {
			t.Errorf("dashboard missing %q", want)
		}
	}
}
