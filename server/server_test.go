package server

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
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
