package disposition

// Gap tests for disposition.go:
// - Load JSON parse error (line 74-76); read error (line 71) already covered.
// - RenderEntries reference branch (line 136-138).
// The Save MarshalIndent error (line 86-88) is unreachable for Log structs.

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestLoadBadJSONGap verifies Load returns a non-nil error when the dispositions
// file contains invalid JSON, covering disposition.go:74.51,76.3.
//
//fusa:test REQ-FO-DISP002
func TestLoadBadJSONGap(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, DispositionsFile), []byte("{bad json"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Load(dir)
	if err == nil {
		t.Fatal("Load: expected error for malformed JSON, got nil")
	}
}

// TestRenderEntriesWithReference verifies RenderEntries includes the reference
// string when Entry.Reference is non-empty, covering disposition.go:136.24,138.4.
//
//fusa:test REQ-FO-DISP003
func TestRenderEntriesWithReference(t *testing.T) {
	log := &Log{
		Project: "testproj",
		Entries: []Entry{
			{
				RuleID:    "REQ-TEST-001",
				Rationale: "Accepted per design review",
				Reviewer:  "alice",
				Date:      time.Date(2026, 7, 26, 0, 0, 0, 0, time.UTC),
				Action:    ActionAccept,
				Reference: "DR-2026-042",
			},
		},
	}
	var buf bytes.Buffer
	if err := RenderEntries(&buf, log); err != nil {
		t.Fatalf("RenderEntries: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "DR-2026-042") {
		t.Errorf("RenderEntries: expected reference DR-2026-042 in output:\n%s", out)
	}
}
