package req

// Gap tests covering uncovered branches in req.go.

import (
	"bytes"
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

// TestExportCodebeamerDescriptionAndLevel verifies that ExportCodebeamer uses
// e.Description when e.Text is empty and sets Fields when e.Level is non-empty,
// covering req.go:316.17,318.4 and req.go:320.20,322.4.
//
//fusa:test REQ-FO-REQ003
func TestExportCodebeamerDescriptionAndLevel(t *testing.T) {
	entries := []Entry{{
		ID: "REQ-1", Title: "T1",
		Text:        "",
		Description: "desc used as text",
		Level:       "ASIL-B",
	}}
	data, err := ExportCodebeamer(entries)
	if err != nil {
		t.Fatalf("ExportCodebeamer: %v", err)
	}
	if !bytes.Contains(data, []byte("desc used as text")) {
		t.Error("ExportCodebeamer: expected description in output")
	}
	if !bytes.Contains(data, []byte("ASIL-B")) {
		t.Error("ExportCodebeamer: expected level ASIL-B in output")
	}
}

// TestExportJamaDescriptionAndLevel verifies that ExportJama uses e.Description
// when e.Text is empty and sets Fields when e.Level is non-empty, covering
// req.go:397.17,399.4 and req.go:401.20,403.4.
//
//fusa:test REQ-FO-REQ003
func TestExportJamaDescriptionAndLevel(t *testing.T) {
	entries := []Entry{{
		ID: "REQ-1", Title: "T1",
		Text:        "",
		Description: "jama desc",
		Level:       "SIL-2",
	}}
	data, err := ExportJama(entries)
	if err != nil {
		t.Fatalf("ExportJama: %v", err)
	}
	if !bytes.Contains(data, []byte("jama desc")) {
		t.Error("ExportJama: expected description in output")
	}
	if !bytes.Contains(data, []byte("SIL-2")) {
		t.Error("ExportJama: expected level SIL-2 in output")
	}
}

// TestExportPolarionLevel verifies that ExportPolarion sets Fields when
// e.Level is non-empty, covering req.go:478.20,480.4.
//
//fusa:test REQ-FO-REQ003
func TestExportPolarionLevel(t *testing.T) {
	entries := []Entry{{
		ID: "REQ-1", Title: "T1",
		Text:  "text",
		Level: "ASIL-C",
	}}
	data, err := ExportPolarion(entries)
	if err != nil {
		t.Fatalf("ExportPolarion: %v", err)
	}
	if !bytes.Contains(data, []byte("ASIL-C")) {
		t.Error("ExportPolarion: expected level ASIL-C in output")
	}
}
