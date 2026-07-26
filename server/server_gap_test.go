package server

// Additional tests targeting uncovered branches in WithComp, handleAPIComp,
// handleAPIBaseline, handleAPIHistory, handleFleet, handleAPIFleet, and computeFleet.

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/SoundMatt/FuSaOps/adapter"
	"github.com/SoundMatt/FuSaOps/orchestrator"
)

// ── WithComp ─────────────────────────────────────────────────────────────────

// TestWithComp verifies WithComp sets compThreshold and compDAL on the server.
//
//fusa:test REQ-FO-SRV012
func TestWithComp(t *testing.T) {
	s := newTestServer(t)
	s2 := s.WithComp(15, "DAL-B")
	if s2.compThreshold != 15 {
		t.Errorf("compThreshold: got %d, want 15", s2.compThreshold)
	}
	if s2.compDAL != "DAL-B" {
		t.Errorf("compDAL: got %q, want DAL-B", s2.compDAL)
	}
}

// ── handleAPIComp ─────────────────────────────────────────────────────────────

// TestAPICompErrorState verifies /api/v1/comp returns 500 when compErr is set.
//
//fusa:test REQ-FO-SRV012
func TestAPICompErrorState(t *testing.T) {
	s := newTestServer(t)
	s.compMu.Lock()
	s.compErr = errors.New("comp tool failed")
	s.compMu.Unlock()

	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/comp", nil))
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("api/v1/comp with error: want 500, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "comp tool failed") {
		t.Errorf("expected error message in response: %q", rec.Body.String())
	}
}

// ── handleAPIBaseline ─────────────────────────────────────────────────────────

// TestAPIBaselineMethodNotAllowed verifies GET /api/v1/baseline returns 405.
//
//fusa:test REQ-FO-SRV008
func TestAPIBaselineMethodNotAllowed(t *testing.T) {
	bl := t.TempDir() + "/baseline.json"
	s := newTestServer(t).WithBaseline(bl)
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/baseline", nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("GET /api/v1/baseline: want 405, got %d", rec.Code)
	}
}

// TestAPIBaselineNoReport verifies POST /api/v1/baseline returns 503 when no
// report has been computed yet.
//
//fusa:test REQ-FO-SRV008
func TestAPIBaselineNoReport(t *testing.T) {
	bl := t.TempDir() + "/baseline.json"
	// Build a server with WithBaseline but do NOT call compute.
	reg := adapter.NewRegistry()
	s := New(t.TempDir(), orchestrator.New(reg), orchestrator.Options{}).WithBaseline(bl)
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/v1/baseline", nil))
	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("POST /api/v1/baseline (no report): want 503, got %d", rec.Code)
	}
}

// ── handleAPIHistory ──────────────────────────────────────────────────────────

// TestAPIHistoryNoHistDir verifies /api/v1/history returns [] when histDir is not set.
// This covers the s.histDir == "" early-return branch.
//
//fusa:test REQ-FO-HST004
func TestAPIHistoryNoHistDir(t *testing.T) {
	// newTestServer does not call WithHistoryDir → histDir remains "".
	s := newTestServer(t)
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/history", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("api/v1/history (no histDir): want 200, got %d", rec.Code)
	}
	body := strings.TrimSpace(rec.Body.String())
	if body != "[]" {
		t.Errorf("expected [], got %q", body)
	}
}

// ── computeFleet ──────────────────────────────────────────────────────────────

// TestComputeFleetBadConfig verifies computeFleet sets fleetErr when the fleet
// config file does not exist.
//
//fusa:test REQ-FO-FLT005
func TestComputeFleetBadConfig(t *testing.T) {
	s := newTestServer(t)
	s.fleetCfg = "/nonexistent-dir-xyz/fleet.json"
	s.computeFleet(context.Background())

	s.fleetMu.RLock()
	err := s.fleetErr
	s.fleetMu.RUnlock()
	if err == nil {
		t.Error("expected fleetErr after bad fleet config, got nil")
	}
}

// ── handleFleet ───────────────────────────────────────────────────────────────

// TestFleetPageError verifies /fleet returns 500 when fleetErr is set.
//
//fusa:test REQ-FO-FLT006
func TestFleetPageError(t *testing.T) {
	s := newTestServer(t)
	// Enable fleet routes by setting fleetCfg.
	s.fleetCfg = "/some/fleet.json"
	// Inject an error directly.
	s.fleetMu.Lock()
	s.fleetErr = errors.New("fleet scan failed")
	s.fleetMu.Unlock()

	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/fleet", nil))
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("/fleet with error: want 500, got %d", rec.Code)
	}
}

// TestFleetPageNotReady verifies /fleet returns 503 when the fleet report is nil
// and no error has occurred yet.
//
//fusa:test REQ-FO-FLT006
func TestFleetPageNotReady(t *testing.T) {
	s := newTestServer(t)
	// Enable fleet routes but do not compute fleet → fleetRep remains nil.
	s.fleetCfg = "/some/fleet.json"

	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/fleet", nil))
	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("/fleet (not ready): want 503, got %d", rec.Code)
	}
}

// ── handleAPIFleet ────────────────────────────────────────────────────────────

// TestAPIFleetError verifies /api/fleet returns 500 when fleetErr is set.
//
//fusa:test REQ-FO-FLT006
func TestAPIFleetError(t *testing.T) {
	s := newTestServer(t)
	s.fleetCfg = "/some/fleet.json"
	s.fleetMu.Lock()
	s.fleetErr = errors.New("fleet api error")
	s.fleetMu.Unlock()

	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/fleet", nil))
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("/api/fleet with error: want 500, got %d", rec.Code)
	}
}

// TestAPIFleetNotReady verifies /api/fleet returns 503 when the fleet report is nil.
//
//fusa:test REQ-FO-FLT006
func TestAPIFleetNotReady(t *testing.T) {
	s := newTestServer(t)
	s.fleetCfg = "/some/fleet.json"

	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/fleet", nil))
	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("/api/fleet (not ready): want 503, got %d", rec.Code)
	}
}

// ── statusRecorder.WriteHeader ────────────────────────────────────────────────

// TestWriteHeaderViaAuth verifies that statusRecorder.WriteHeader is exercised
// when the auth middleware is active and the inner handler writes an explicit
// non-200 status code.
//
//fusa:test REQ-FO-AUTH001
func TestWriteHeaderViaAuth(t *testing.T) {
	s := newTestServer(t).WithAuth("admin", "secret")
	// Inject a scan error so /api/v1/status returns 500 explicitly.
	s.mu.Lock()
	s.err = errors.New("injected scan error for WriteHeader coverage")
	s.mu.Unlock()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/status", nil)
	req.SetBasicAuth("admin", "secret")
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("want 500, got %d", rec.Code)
	}
}
