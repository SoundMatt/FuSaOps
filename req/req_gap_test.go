package req

// Gap tests covering uncovered branches in req.go.

import (
	"path/filepath"
	"strings"
	"testing"
)

// TestParseCSVBadFormat verifies ParseCSV returns an error when the CSV
// contains a malformed quoted field (parse error), covering req.go:82.16,84.3.
//
//fusa:test REQ-FO-REQ002
func TestParseCSVBadFormat(t *testing.T) {
	// Extraneous quote in a field causes csv.Reader.ReadAll to fail.
	badCSV := "id,title\n\"unclosed\n"
	_, err := ParseCSV(strings.NewReader(badCSV))
	if err == nil {
		t.Error("ParseCSV: expected error for malformed CSV, got nil")
	}
}

// TestSaveRegistryWriteError verifies SaveRegistry returns an error when the
// target directory does not exist (os.WriteFile fails), covering
// req.go:67 (the write error path; the marshal error at line 63 is unreachable).
//
//fusa:test REQ-FO-REQ001
func TestSaveRegistryWriteError(t *testing.T) {
	badDir := filepath.Join(t.TempDir(), "nonexistent")
	err := SaveRegistry(badDir, []Entry{{ID: "R001", Title: "Test"}})
	if err == nil {
		t.Error("SaveRegistry: expected error for non-existent directory, got nil")
	}
}
