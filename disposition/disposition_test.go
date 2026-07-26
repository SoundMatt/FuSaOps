package disposition

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

//fusa:test REQ-FO-DISP001
func TestActionConstants(t *testing.T) {
	if ActionAccept != "accept" || ActionFix != "fix" {
		t.Error("action constants wrong")
	}
}

//fusa:test REQ-FO-DISP001
func TestEntryFields(t *testing.T) {
	e := Entry{
		RuleID:    "RULE001",
		Language:  "go",
		Rationale: "accepted by design",
		Reviewer:  "alice",
		Date:      time.Now(),
		Action:    ActionAccept,
		Reference: "TICKET-123",
	}
	if e.RuleID != "RULE001" || e.Action != ActionAccept {
		t.Errorf("unexpected entry: %+v", e)
	}
}

//fusa:test REQ-FO-DISP002
func TestLoadMissing(t *testing.T) {
	log, err := Load(t.TempDir())
	if err != nil {
		t.Fatalf("Load missing: %v", err)
	}
	if len(log.Entries) != 0 {
		t.Errorf("want empty, got %d entries", len(log.Entries))
	}
}

//fusa:test REQ-FO-DISP002
func TestSaveLoad(t *testing.T) {
	dir := t.TempDir()
	log := &Log{Project: "testproj", Entries: []Entry{
		{RuleID: "RULE001", Language: "go", Action: ActionAccept, Reviewer: "alice",
			Rationale: "by design", Date: time.Now().UTC()},
	}}
	if err := Save(dir, log); err != nil {
		t.Fatalf("Save: %v", err)
	}
	loaded, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.Project != "testproj" || len(loaded.Entries) != 1 {
		t.Errorf("unexpected loaded: %+v", loaded)
	}
	if loaded.Entries[0].RuleID != "RULE001" {
		t.Errorf("entry mismatch: %+v", loaded.Entries[0])
	}
}

//fusa:test REQ-FO-DISP002
func TestAdd(t *testing.T) {
	log := &Log{}
	e := Entry{RuleID: "R1", Action: ActionFix}
	log = Add(log, e)
	if len(log.Entries) != 1 || log.Entries[0].RuleID != "R1" {
		t.Errorf("Add: %+v", log.Entries)
	}
}

//fusa:test REQ-FO-DISP002
func TestFind(t *testing.T) {
	log := &Log{Entries: []Entry{
		{RuleID: "R1", Language: "go", Action: ActionAccept},
		{RuleID: "R2", Language: "rust", Action: ActionFix},
	}}
	e := Find(log, "R1", "go")
	if e == nil || e.RuleID != "R1" {
		t.Errorf("Find R1/go: %v", e)
	}
	if Find(log, "R99", "") != nil {
		t.Error("Find R99 should be nil")
	}
}

//fusa:test REQ-FO-DISP002
func TestFindAnyLanguage(t *testing.T) {
	log := &Log{Entries: []Entry{
		{RuleID: "R1", Language: "", Action: ActionAccept},
	}}
	// Should match even when searching with a specific language, because entry has no language.
	e := Find(log, "R1", "go")
	if e == nil {
		t.Error("Find should match entry with empty language")
	}
}

//fusa:test REQ-FO-DISP003
func TestRenderEntriesEmpty(t *testing.T) {
	var buf bytes.Buffer
	if err := RenderEntries(&buf, &Log{}); err != nil {
		t.Fatalf("RenderEntries empty: %v", err)
	}
	if !strings.Contains(buf.String(), "No disposition") {
		t.Errorf("empty output missing message: %q", buf.String())
	}
}

//fusa:test REQ-FO-DISP003
func TestRenderEntries(t *testing.T) {
	log := &Log{
		Project: "myproj",
		Entries: []Entry{
			{RuleID: "RULE001", Language: "go", Action: ActionAccept,
				Reviewer: "alice", Rationale: "by design",
				Date: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)},
		},
	}
	var buf bytes.Buffer
	if err := RenderEntries(&buf, log); err != nil {
		t.Fatalf("RenderEntries: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "RULE001") || !strings.Contains(out, "alice") {
		t.Errorf("render output missing expected content: %q", out)
	}
}

// TestSaveWriteError verifies Save returns an error when the project root
// directory does not exist.
//
//fusa:test REQ-FO-DISP002
func TestSaveWriteError(t *testing.T) {
	root := filepath.Join(t.TempDir(), "missing")
	if err := Save(root, &Log{}); err == nil {
		t.Error("Save: expected error for non-existent root directory")
	}
}
