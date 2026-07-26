package history

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	fusaops "github.com/SoundMatt/FuSaOps"
	"github.com/SoundMatt/FuSaOps/report"
)

func makeReport(errors, warnings, infos int) *report.AggregateReport {
	var findings []fusaops.Finding
	for range errors {
		findings = append(findings, fusaops.Finding{
			Language: fusaops.LangGo, Severity: fusaops.SeverityError,
		})
	}
	for range warnings {
		findings = append(findings, fusaops.Finding{
			Language: fusaops.LangGo, Severity: fusaops.SeverityWarning,
		})
	}
	return report.New(".", "test", []report.Component{{
		Language: fusaops.LangGo, Tool: "gofusa", Available: true,
		Findings: findings,
	}})
}

// TestFromReport verifies Snapshot is populated correctly from a report.
//
//fusa:test REQ-FO-HST001
func TestFromReport(t *testing.T) {
	rep := makeReport(2, 3, 0)
	s := FromReport(rep)
	if s.Status != "FAIL" {
		t.Errorf("status: got %q, want FAIL", s.Status)
	}
	if s.Errors != 2 || s.Warnings != 3 || s.Total != 5 {
		t.Errorf("counts: errors=%d warnings=%d total=%d", s.Errors, s.Warnings, s.Total)
	}
	if len(s.Languages) != 1 || s.Languages[0].Language != "go" {
		t.Errorf("languages: %+v", s.Languages)
	}
}

// TestFromReportPassStatus verifies a clean report produces PASS.
//
//fusa:test REQ-FO-HST001
func TestFromReportPassStatus(t *testing.T) {
	s := FromReport(makeReport(0, 1, 0))
	if s.Status != "PASS" {
		t.Errorf("want PASS with no errors, got %q", s.Status)
	}
}

// TestStoreLoad verifies round-trip persistence to JSONL.
//
//fusa:test REQ-FO-HST002
func TestStoreLoad(t *testing.T) {
	dir := t.TempDir()
	now := time.Now().UTC().Truncate(time.Second)

	snaps := []Snapshot{
		{RunAt: now.Add(-2 * time.Hour), Status: "PASS", Total: 0},
		{RunAt: now.Add(-time.Hour), Status: "FAIL", Total: 3, Errors: 1},
		{RunAt: now, Status: "PASS", Total: 1},
	}
	for _, s := range snaps {
		if err := Store(dir, s); err != nil {
			t.Fatalf("Store: %v", err)
		}
	}

	got, err := Load(dir, 0)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("Load returned %d snapshots, want 3", len(got))
	}
	if got[1].Errors != 1 || got[1].Status != "FAIL" {
		t.Errorf("snapshot[1]: %+v", got[1])
	}
}

// TestLoadLimit verifies the limit parameter trims the result.
//
//fusa:test REQ-FO-HST002
func TestLoadLimit(t *testing.T) {
	dir := t.TempDir()
	for i := range 5 {
		_ = Store(dir, Snapshot{RunAt: time.Now(), Total: i})
	}
	got, err := Load(dir, 3)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(got) != 3 {
		t.Errorf("want 3, got %d", len(got))
	}
	// Most recent 3 should be totals 2, 3, 4
	if got[0].Total != 2 {
		t.Errorf("oldest of limit-3: want total=2, got %d", got[0].Total)
	}
}

// TestLoadMissingFile verifies a missing file returns empty slice without error.
//
//fusa:test REQ-FO-HST002
func TestLoadMissingFile(t *testing.T) {
	got, err := Load(t.TempDir(), 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("want empty, got %d", len(got))
	}
}

// TestStoreTrim verifies that MaxSnapshots is enforced.
//
//fusa:test REQ-FO-HST002
func TestStoreTrim(t *testing.T) {
	dir := t.TempDir()
	for i := range MaxSnapshots + 5 {
		_ = Store(dir, Snapshot{RunAt: time.Now(), Total: i})
	}
	got, err := Load(dir, 0)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(got) > MaxSnapshots {
		t.Errorf("want <= %d snapshots, got %d", MaxSnapshots, len(got))
	}
	// Last entry should be the most-recent (Total = MaxSnapshots+4)
	if got[len(got)-1].Total != MaxSnapshots+4 {
		t.Errorf("last total: want %d, got %d", MaxSnapshots+4, got[len(got)-1].Total)
	}
}

// TestPrune verifies that Prune removes old entries and returns the count.
//
//fusa:test REQ-FO-HST003
func TestPrune(t *testing.T) {
	dir := t.TempDir()
	for i := range 10 {
		_ = Store(dir, Snapshot{RunAt: time.Now(), Total: i})
	}
	removed, err := Prune(dir, 6)
	if err != nil {
		t.Fatalf("Prune: %v", err)
	}
	if removed != 4 {
		t.Errorf("removed: want 4, got %d", removed)
	}
	got, _ := Load(dir, 0)
	if len(got) != 6 {
		t.Errorf("after prune: want 6, got %d", len(got))
	}
	if got[len(got)-1].Total != 9 {
		t.Errorf("last entry should be newest (Total=9), got %d", got[len(got)-1].Total)
	}
}

//fusa:test REQ-FO-HST003
func TestPruneMissingFile(t *testing.T) {
	removed, err := Prune(t.TempDir(), 10)
	if err != nil || removed != 0 {
		t.Errorf("missing file: want 0, nil; got %d, %v", removed, err)
	}
}

//fusa:test REQ-FO-HST003
func TestPruneNothingToRemove(t *testing.T) {
	dir := t.TempDir()
	for range 3 {
		_ = Store(dir, Snapshot{RunAt: time.Now(), Total: 1})
	}
	removed, err := Prune(dir, 10)
	if err != nil || removed != 0 {
		t.Errorf("nothing to remove: want 0, nil; got %d, %v", removed, err)
	}
}

// TestCorruptLineSkipped verifies that a corrupt JSONL line is silently skipped.
func TestCorruptLineSkipped(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, Filename)
	_ = os.WriteFile(path, []byte(`{"runAt":"2026-01-01T00:00:00Z","status":"PASS"}
NOT_JSON
{"runAt":"2026-01-02T00:00:00Z","status":"FAIL"}
`), 0o644)
	got, err := Load(dir, 0)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("want 2 valid lines, got %d", len(got))
	}
}

// TestPruneWriteAllError verifies Prune propagates errors from writeAll when
// the history file cannot be overwritten.
//
//fusa:test REQ-FO-HST004
func TestPruneWriteAllError(t *testing.T) {
	dir := t.TempDir()
	snap := FromReport(makeReport(0, 0, 0))
	// Write enough snapshots to exceed keep=1 so Prune calls writeAll.
	for range 3 {
		if err := Store(dir, snap); err != nil {
			t.Fatalf("Store: %v", err)
		}
	}
	// Replace the history file with a directory so os.Create fails.
	histPath := filepath.Join(dir, Filename)
	if err := os.Remove(histPath); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if err := os.Mkdir(histPath, 0o750); err != nil {
		t.Fatalf("Mkdir: %v", err)
	}
	if n, err := Prune(dir, 1); err == nil {
		t.Errorf("Prune: expected error when history path is a directory, got nil (removed %d)", n)
	}
}

// TestStoreWriteError verifies Store returns an error when the history dir does
// not exist (covers the os.Create error path in writeAll).
//
//fusa:test REQ-FO-HST002
func TestStoreWriteError(t *testing.T) {
	snap := FromReport(makeReport(0, 0, 0))
	// Non-existent subdirectory → writeAll os.Create fails.
	err := Store(filepath.Join(t.TempDir(), "nonexistent-subdir"), snap)
	if err == nil {
		t.Error("Store: expected error for non-existent directory")
	}
}
