package server

// Targeted tests for the /api/v1/mcdc and /mcdc endpoints, raising server
// package coverage above the 80% gate.

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/SoundMatt/FuSaOps/adapter"
	"github.com/SoundMatt/FuSaOps/mcdc"
	"github.com/SoundMatt/FuSaOps/orchestrator"
)

// TestAPIMCDCBeforeCompute verifies /api/v1/mcdc returns {} when mcdcAgg is nil
// (i.e. the server has not yet computed any MC/DC data).
//
//fusa:test REQ-FO-MCDC002
func TestAPIMCDCBeforeCompute(t *testing.T) {
	// Create a server without calling compute so mcdcAgg stays nil.
	reg := adapter.NewRegistry()
	s := New(t.TempDir(), orchestrator.New(reg), orchestrator.Options{})
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/mcdc", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200", rec.Code)
	}
	body := strings.TrimSpace(rec.Body.String())
	if body != "{}" {
		t.Errorf("/api/v1/mcdc before compute: expected {}, got %q", body)
	}
}

// TestAPIMCDCWithData verifies /api/v1/mcdc returns the cached MC/DC aggregate.
//
//fusa:test REQ-FO-MCDC002
func TestAPIMCDCWithData(t *testing.T) {
	s := newTestServer(t)
	s.mcdcMu.Lock()
	s.mcdcAgg = &mcdc.MCDCAggregate{
		Root:              "/repo",
		Project:           "demo",
		TotalConditions:   10,
		CoveredConditions: 8,
		GatePassed:        false,
		Components: []mcdc.MCDCComponent{{
			Language: "go",
			Tool:     "gofusa",
			Report: &mcdc.Report{
				TotalConditions:   10,
				CoveredConditions: 8,
				GatePassed:        false,
			},
		}},
	}
	s.mcdcMu.Unlock()

	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/mcdc", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200", rec.Code)
	}
	ct := rec.Header().Get("Content-Type")
	if !strings.Contains(ct, "application/json") {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
	var out map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("response not JSON: %v", err)
	}
	if v, _ := out["totalConditions"].(float64); int(v) != 10 {
		t.Errorf("totalConditions = %v, want 10", out["totalConditions"])
	}
	if out["gatePassed"] != false {
		t.Errorf("gatePassed = %v, want false", out["gatePassed"])
	}
}

// TestMCDCPageBeforeCompute verifies /mcdc returns a 200 HTML page with a "no data"
// message when the mcdcAgg is nil (server not yet computed).
//
//fusa:test REQ-FO-MCDC003
func TestMCDCPageBeforeCompute(t *testing.T) {
	// Create a server without calling compute so mcdcAgg stays nil.
	reg := adapter.NewRegistry()
	s := New(t.TempDir(), orchestrator.New(reg), orchestrator.Options{})
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/mcdc", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "MC/DC") {
		t.Error("mcdc page missing heading")
	}
	if !strings.Contains(body, "No MC/DC data") {
		t.Errorf("/mcdc page missing 'No MC/DC data' message")
	}
}

// TestMCDCPageWithData verifies /mcdc renders per-component MC/DC coverage detail.
//
//fusa:test REQ-FO-MCDC003
func TestMCDCPageWithData(t *testing.T) {
	s := newTestServer(t)
	s.mcdcMu.Lock()
	s.mcdcAgg = &mcdc.MCDCAggregate{
		Root:              "/repo",
		Project:           "demo",
		TotalConditions:   12,
		CoveredConditions: 10,
		GatePassed:        false,
		Components: []mcdc.MCDCComponent{{
			Language: "go",
			Tool:     "gofusa",
			Report: &mcdc.Report{
				TotalConditions:   12,
				CoveredConditions: 10,
				GatePassed:        false,
				Decisions: []mcdc.Decision{{
					Name:        "checkSafety",
					File:        "safety.go",
					MCDCCovered: false,
				}},
			},
		}},
	}
	s.mcdcMu.Unlock()

	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/mcdc", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{"MC/DC Coverage", "gofusa", "12", "10", "FAIL"} {
		if !strings.Contains(body, want) {
			t.Errorf("/mcdc page missing %q", want)
		}
	}
}

// TestMCDCPageSkippedComponent verifies /mcdc handles skipped components gracefully.
//
//fusa:test REQ-FO-MCDC003
func TestMCDCPageSkippedComponent(t *testing.T) {
	s := newTestServer(t)
	s.mcdcMu.Lock()
	s.mcdcAgg = &mcdc.MCDCAggregate{
		Root:              "/repo",
		TotalConditions:   0,
		CoveredConditions: 0,
		GatePassed:        true,
		Components: []mcdc.MCDCComponent{{
			Language: "rust",
			Tool:     "rsfusa",
			Skipped:  "tool not installed",
		}},
	}
	s.mcdcMu.Unlock()

	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/mcdc", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "skipped") {
		t.Errorf("/mcdc page missing 'skipped' for skipped component")
	}
}

// TestDashboardShowsMCDCSection verifies the HTML dashboard includes the MC/DC
// section when mcdcAgg is cached with at least one condition.
//
//fusa:test REQ-FO-MCDC003
func TestDashboardShowsMCDCSection(t *testing.T) {
	s := newTestServer(t)
	s.mcdcMu.Lock()
	s.mcdcAgg = &mcdc.MCDCAggregate{
		Root:              "/repo",
		Project:           "demo",
		TotalConditions:   8,
		CoveredConditions: 6,
		GatePassed:        false,
		Components: []mcdc.MCDCComponent{{
			Language: "go",
			Tool:     "gofusa",
			Report: &mcdc.Report{
				TotalConditions:   8,
				CoveredConditions: 6,
				GatePassed:        false,
			},
		}},
	}
	s.mcdcMu.Unlock()

	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "MC/DC") {
		t.Errorf("dashboard missing MC/DC section")
	}
}

// TestIndexNilReport verifies /  returns 503 when no report has been computed yet.
//
//fusa:test REQ-FO-SRV001
func TestIndexNilReport(t *testing.T) {
	reg := adapter.NewRegistry()
	s := New(t.TempDir(), orchestrator.New(reg), orchestrator.Options{})
	// Do NOT call compute — cached is nil.
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("index with nil cached: got %d, want 503", rec.Code)
	}
}

// TestAPIReportNilReport verifies /api/v1/report returns 503 when no report is cached.
//
//fusa:test REQ-FO-SRV004
func TestAPIReportNilReport(t *testing.T) {
	reg := adapter.NewRegistry()
	s := New(t.TempDir(), orchestrator.New(reg), orchestrator.Options{})
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/report", nil))
	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("/api/v1/report nil cached: got %d, want 503", rec.Code)
	}
}
