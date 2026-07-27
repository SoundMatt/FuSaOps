package history

// Gap tests covering uncovered branches in history.go.

import (
	"os"
	"path/filepath"
	"testing"
)

// TestLoadAllOpenError verifies loadAll and Load return an error when the
// history file exists but cannot be opened (it is a directory, causing EISDIR),
// covering history.go:123.16,125.3 (non-ErrNotExist open error).
//
//fusa:test REQ-FO-HST002
func TestLoadAllOpenError(t *testing.T) {
	dir := t.TempDir()
	// Create a directory at the history file path so os.Open returns EISDIR.
	if err := os.MkdirAll(filepath.Join(dir, Filename), 0o755); err != nil {
		t.Fatal(err)
	}
	_, err := Load(dir, 0)
	if err == nil {
		t.Error("Load: expected error when history path is a directory, got nil")
	}
}

// TestPruneLoadAllError verifies Prune returns an error when loadAll fails
// (history file is a directory), covering history.go:149.15,151.3.
//
//fusa:test REQ-FO-HST003
func TestPruneLoadAllError(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, Filename), 0o755); err != nil {
		t.Fatal(err)
	}
	_, err := Prune(dir, 1)
	if err == nil {
		t.Error("Prune: expected error when history path is a directory, got nil")
	}
}

// TestLoadEmptyLine verifies that an empty line in the JSONL history file is
// silently skipped, covering history.go:132.21,133.12.
//
//fusa:test REQ-FO-HST002
func TestLoadEmptyLine(t *testing.T) {
	dir := t.TempDir()
	// Write a JSONL file with an empty line between two valid entries.
	data := `{"runAt":"2026-01-01T00:00:00Z","status":"PASS"}` + "\n\n" +
		`{"runAt":"2026-01-02T00:00:00Z","status":"FAIL"}` + "\n"
	if err := os.WriteFile(filepath.Join(dir, Filename), []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
	snaps, err := Load(dir, 0)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(snaps) != 2 {
		t.Errorf("want 2 snapshots (empty line skipped), got %d", len(snaps))
	}
}
