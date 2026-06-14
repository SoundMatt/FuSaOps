package metrics

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

//fusa:test REQ-FO-MET001
func TestSnapshotFields(t *testing.T) {
	snap := Snapshot{
		Timestamp:         time.Now(),
		ErrorCount:        2,
		WarningCount:      5,
		InfoCount:         10,
		TotalRequirements: 100,
		CoveragePct:       82.5,
	}
	if snap.ErrorCount != 2 || snap.CoveragePct != 82.5 {
		t.Errorf("unexpected snapshot: %+v", snap)
	}
}

//fusa:test REQ-FO-MET002
func TestLoadMissing(t *testing.T) {
	ts, err := Load(t.TempDir())
	if err != nil {
		t.Fatalf("Load missing: %v", err)
	}
	if len(ts.Snapshots) != 0 {
		t.Errorf("want empty, got %d snapshots", len(ts.Snapshots))
	}
}

//fusa:test REQ-FO-MET002
func TestSaveLoad(t *testing.T) {
	dir := t.TempDir()
	ts := &TimeSeries{Project: "test-project", Snapshots: []Snapshot{
		{Timestamp: time.Now().UTC(), ErrorCount: 1, TotalRequirements: 50},
	}}
	if err := Save(dir, ts); err != nil {
		t.Fatalf("Save: %v", err)
	}
	loaded, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.Project != "test-project" {
		t.Errorf("project = %q, want test-project", loaded.Project)
	}
	if len(loaded.Snapshots) != 1 || loaded.Snapshots[0].ErrorCount != 1 {
		t.Errorf("unexpected snapshots: %v", loaded.Snapshots)
	}
}

//fusa:test REQ-FO-MET002
func TestAppend(t *testing.T) {
	ts := &TimeSeries{}
	snap := Snapshot{ErrorCount: 3}
	ts = Append(ts, snap)
	if len(ts.Snapshots) != 1 || ts.Snapshots[0].ErrorCount != 3 {
		t.Errorf("unexpected after append: %v", ts.Snapshots)
	}
}

//fusa:test REQ-FO-MET002
func TestCollectRequirements(t *testing.T) {
	dir := t.TempDir()
	reqs := `{"requirements":[{"id":"REQ-1"},{"id":"REQ-2"},{"id":"REQ-3"}]}`
	if err := os.WriteFile(filepath.Join(dir, ".fusa-reqs.json"), []byte(reqs), 0o600); err != nil {
		t.Fatal(err)
	}
	snap, err := Collect(dir)
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	if snap.TotalRequirements != 3 {
		t.Errorf("TotalRequirements = %d, want 3", snap.TotalRequirements)
	}
}

//fusa:test REQ-FO-MET002
func TestCollectCoverageReport(t *testing.T) {
	dir := t.TempDir()
	covReport := `{"stmtPct":85.5}`
	if err := os.WriteFile(filepath.Join(dir, "coverage-report.json"), []byte(covReport), 0o600); err != nil {
		t.Fatal(err)
	}
	snap, err := Collect(dir)
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	if snap.CoveragePct != 85.5 {
		t.Errorf("CoveragePct = %.1f, want 85.5", snap.CoveragePct)
	}
}

//fusa:test REQ-FO-MET002
func TestCollectCheckReport(t *testing.T) {
	dir := t.TempDir()
	report := `{"components":[{"findings":[{"severity":"ERROR"},{"severity":"WARNING"},{"severity":"INFO"}]}]}`
	if err := os.WriteFile(filepath.Join(dir, "check-report.json"), []byte(report), 0o600); err != nil {
		t.Fatal(err)
	}
	snap, err := Collect(dir)
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	if snap.ErrorCount != 1 || snap.WarningCount != 1 || snap.InfoCount != 1 {
		t.Errorf("counts: err=%d warn=%d info=%d", snap.ErrorCount, snap.WarningCount, snap.InfoCount)
	}
}

//fusa:test REQ-FO-MET002
func TestCollectEmpty(t *testing.T) {
	snap, err := Collect(t.TempDir())
	if err != nil {
		t.Fatalf("Collect empty dir: %v", err)
	}
	if snap.ErrorCount != 0 || snap.TotalRequirements != 0 {
		t.Errorf("unexpected non-zero: %+v", snap)
	}
}

//fusa:test REQ-FO-MET003
func TestRenderText(t *testing.T) {
	ts := &TimeSeries{Project: "myproj", Snapshots: []Snapshot{
		{Timestamp: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC), ErrorCount: 2, WarningCount: 5, TotalRequirements: 100, CoveragePct: 82.1},
	}}
	var buf bytes.Buffer
	if err := Render(&buf, ts, "text"); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "myproj") {
		t.Error("text output missing project")
	}
	if !strings.Contains(out, "2026-01-01") {
		t.Error("text output missing date")
	}
}

//fusa:test REQ-FO-MET003
func TestRenderTextEmpty(t *testing.T) {
	ts := &TimeSeries{}
	var buf bytes.Buffer
	if err := Render(&buf, ts, "text"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "No snapshots") {
		t.Error("empty series should print no-snapshots message")
	}
}

//fusa:test REQ-FO-MET003
func TestRenderJSON(t *testing.T) {
	ts := &TimeSeries{Project: "proj", Snapshots: []Snapshot{{ErrorCount: 3}}}
	var buf bytes.Buffer
	if err := Render(&buf, ts, "json"); err != nil {
		t.Fatal(err)
	}
	var out TimeSeries
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if out.Project != "proj" {
		t.Errorf("project = %q", out.Project)
	}
}

//fusa:test REQ-FO-MET003
func TestRenderUnknown(t *testing.T) {
	ts := &TimeSeries{}
	err := Render(&bytes.Buffer{}, ts, "xml")
	if err == nil {
		t.Error("expected error for unknown format")
	}
}
