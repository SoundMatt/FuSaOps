package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SoundMatt/FuSaOps/qualify"
)

// writeQualifyReport writes a qualify.Report as JSON to a temp file and returns the path.
func writeQualifyReport(t *testing.T, dir string, r *qualify.Report) string {
	t.Helper()
	path := filepath.Join(dir, qualify.ReportFile)
	if err := qualify.Save(path, r); err != nil {
		t.Fatalf("writeQualifyReport: %v", err)
	}
	return path
}

//fusa:test REQ-FO-SRV011
func TestQualifyBadgePending(t *testing.T) {
	s := newTestServer(t)
	// No qualify report on disk — badge should show "pending".
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/badge/qualify.svg", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "pending") {
		t.Errorf("expected 'pending' in badge SVG, got:\n%s", body)
	}
}

//fusa:test REQ-FO-SRV011
func TestQualifyBadgeSelfPass(t *testing.T) {
	dir := t.TempDir()
	path := writeQualifyReport(t, dir, &qualify.Report{
		QualificationType: "self",
		Total:             5,
		Passed:            5,
		Failed:            0,
		Components:        []qualify.ComponentResult{},
	})
	s := newTestServer(t)
	s = s.WithQualifyReport(path)

	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/badge/qualify.svg", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "self") {
		t.Errorf("expected 'self' in badge SVG, got:\n%s", body)
	}
	if !strings.Contains(body, "pass") {
		t.Errorf("expected 'pass' in badge SVG, got:\n%s", body)
	}
}

//fusa:test REQ-FO-SRV011
func TestQualifyBadgeIndependentFailing(t *testing.T) {
	dir := t.TempDir()
	path := writeQualifyReport(t, dir, &qualify.Report{
		QualificationType: "independent",
		Total:             5,
		Passed:            4,
		Failed:            1,
		Components:        []qualify.ComponentResult{},
	})
	s := newTestServer(t)
	s = s.WithQualifyReport(path)

	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/badge/qualify.svg", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "independent") {
		t.Errorf("expected 'independent' in badge SVG, got:\n%s", body)
	}
	if !strings.Contains(body, "failing") {
		t.Errorf("expected 'failing' in badge SVG, got:\n%s", body)
	}
}

//fusa:test REQ-FO-SRV011
func TestQualifyBadgeContentType(t *testing.T) {
	s := newTestServer(t)
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/badge/qualify.svg", nil))
	ct := rec.Header().Get("Content-Type")
	if ct != "image/svg+xml" {
		t.Errorf("Content-Type = %q, want image/svg+xml", ct)
	}
}

//fusa:test REQ-FO-SRV011
func TestQualifyBadgeCacheControl(t *testing.T) {
	s := newTestServer(t)
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/badge/qualify.svg", nil))
	cc := rec.Header().Get("Cache-Control")
	if cc != "no-cache, no-store, must-revalidate" {
		t.Errorf("Cache-Control = %q, want no-cache, no-store, must-revalidate", cc)
	}
}

//fusa:test REQ-FO-SRV011
func TestDashboardQualifySection(t *testing.T) {
	dir := t.TempDir()
	path := writeQualifyReport(t, dir, &qualify.Report{
		QualificationType:      "independent",
		QualificationRecordUri: "https://cert.example/record.pdf",
		Total:                  8,
		Passed:                 8,
		Failed:                 0,
		Components:             []qualify.ComponentResult{},
	})
	s := newTestServer(t)
	s = s.WithQualifyReport(path)

	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "qualified") {
		t.Errorf("expected 'qualified' in HTML body, got length %d", len(body))
	}
	if !strings.Contains(body, "independent") {
		t.Errorf("expected 'independent' in HTML body")
	}
	if !strings.Contains(body, "https://cert.example/record.pdf") {
		t.Errorf("expected record URI in HTML body")
	}
}

//fusa:test REQ-FO-CLI079
func TestWithQualifyReport(t *testing.T) {
	dir := t.TempDir()
	customPath := filepath.Join(dir, "custom-qualify.json")

	r := &qualify.Report{
		QualificationType: "independent",
		Total:             3,
		Passed:            3,
		Failed:            0,
		Components:        []qualify.ComponentResult{},
	}
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := os.WriteFile(customPath, append(data, '\n'), 0o640); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	s := newTestServer(t)
	s = s.WithQualifyReport(customPath)
	if s.qualifyPath != customPath {
		t.Errorf("qualifyPath = %q, want %q", s.qualifyPath, customPath)
	}

	// Badge should reflect the custom-path report.
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/badge/qualify.svg", nil))
	body := rec.Body.String()
	if !strings.Contains(body, "pass") {
		t.Errorf("expected 'pass' in badge from custom path, got:\n%s", body)
	}
}

// TestAPIQualifyEndpointPending verifies /api/v1/qualify returns empty JSON
// when no qualify report is on disk.
//
//fusa:test REQ-FO-SRV014
func TestAPIQualifyEndpointPending(t *testing.T) {
	s := newTestServer(t)
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/qualify", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	// With no qualify report the body should be the empty-object sentinel.
	body := strings.TrimSpace(rec.Body.String())
	if body != "{}" {
		t.Errorf("expected {}, got: %s", body)
	}
}

// TestAPIQualifyEndpointWithReport verifies /api/v1/qualify returns the full
// qualify report as JSON, including independence fields.
//
//fusa:test REQ-FO-SRV014
//fusa:test REQ-FO-QLF010
func TestAPIQualifyEndpointWithReport(t *testing.T) {
	dir := t.TempDir()
	path := writeQualifyReport(t, dir, &qualify.Report{
		QualificationType:   "independent",
		IndependentReviewer: "Bob Auditor",
		QualificationMethod: "TQL-5",
		AchievableASIL:      "ASIL-D",
		Total:               10,
		Passed:              10,
		Failed:              0,
		Components:          []qualify.ComponentResult{},
	})
	s := newTestServer(t)
	s = s.WithQualifyReport(path)

	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/qualify", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	ct := rec.Header().Get("Content-Type")
	if !strings.Contains(ct, "application/json") {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
	var out map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("response not JSON: %v", err)
	}
	if out["type"] != "independent" {
		t.Errorf("type = %v, want independent", out["type"])
	}
	if out["independentReviewer"] != "Bob Auditor" {
		t.Errorf("independentReviewer = %v, want Bob Auditor", out["independentReviewer"])
	}
	if out["achievableAsil"] != "ASIL-D" {
		t.Errorf("achievableAsil = %v, want ASIL-D", out["achievableAsil"])
	}
	if out["isIndependent"] != true {
		t.Errorf("isIndependent = %v, want true", out["isIndependent"])
	}
	if out["allPassed"] != true {
		t.Errorf("allPassed = %v, want true", out["allPassed"])
	}
}
