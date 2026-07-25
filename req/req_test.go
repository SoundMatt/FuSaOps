package req

import (
	"bytes"
	"encoding/csv"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

//fusa:test REQ-FO-REQ001
func TestLoadRegistry(t *testing.T) {
	dir := t.TempDir()
	data := `{"requirements":[{"id":"REQ-1","title":"Test req","priority":"MUST"}]}` + "\n"
	if err := os.WriteFile(filepath.Join(dir, ReqsFile), []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	entries, err := LoadRegistry(dir)
	if err != nil {
		t.Fatalf("LoadRegistry: %v", err)
	}
	if len(entries) != 1 || entries[0].ID != "REQ-1" {
		t.Errorf("unexpected entries: %v", entries)
	}
	if entries[0].Priority != "MUST" {
		t.Errorf("priority = %q, want MUST", entries[0].Priority)
	}
}

//fusa:test REQ-FO-REQ001
func TestLoadRegistryMissing(t *testing.T) {
	_, err := LoadRegistry(t.TempDir())
	if err == nil {
		t.Error("expected error for missing registry")
	}
}

//fusa:test REQ-FO-REQ001
func TestSaveRegistry(t *testing.T) {
	dir := t.TempDir()
	entries := []Entry{{ID: "REQ-A", Title: "Alpha", Priority: "MUST"}}
	if err := SaveRegistry(dir, entries); err != nil {
		t.Fatalf("SaveRegistry: %v", err)
	}
	loaded, err := LoadRegistry(dir)
	if err != nil {
		t.Fatalf("LoadRegistry after save: %v", err)
	}
	if len(loaded) != 1 || loaded[0].ID != "REQ-A" {
		t.Errorf("round-trip failed: %v", loaded)
	}
}

//fusa:test REQ-FO-REQ002
func TestParseCSV(t *testing.T) {
	data := "id,title,text,standard,level,priority,rationale\nREQ-1,My Req,some text,ISO26262,ASIL-B,MUST,because\n"
	entries, err := ParseCSV(strings.NewReader(data))
	if err != nil {
		t.Fatalf("ParseCSV: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("want 1 entry, got %d", len(entries))
	}
	e := entries[0]
	if e.ID != "REQ-1" || e.Title != "My Req" || e.Standard != "ISO26262" {
		t.Errorf("unexpected entry: %+v", e)
	}
}

//fusa:test REQ-FO-REQ002
func TestParseCSVMissingHeader(t *testing.T) {
	_, err := ParseCSV(strings.NewReader("foo,bar\n1,2\n"))
	if err == nil {
		t.Error("expected error for bad header")
	}
}

//fusa:test REQ-FO-REQ002
func TestParseCSVEmpty(t *testing.T) {
	_, err := ParseCSV(strings.NewReader(""))
	if err == nil {
		t.Error("expected error for empty CSV")
	}
}

//fusa:test REQ-FO-REQ003
func TestRenderCSV(t *testing.T) {
	entries := []Entry{{ID: "REQ-1", Title: "Foo", Standard: "DO-178C", Priority: "MUST"}}
	var buf bytes.Buffer
	if err := RenderCSV(&buf, entries); err != nil {
		t.Fatalf("RenderCSV: %v", err)
	}
	r := csv.NewReader(&buf)
	records, _ := r.ReadAll()
	if len(records) < 2 {
		t.Fatalf("want header+1 row, got %d", len(records))
	}
	if records[0][0] != "id" {
		t.Errorf("header[0] = %q, want id", records[0][0])
	}
	if records[1][0] != "REQ-1" {
		t.Errorf("row[0] id = %q, want REQ-1", records[1][0])
	}
}

//fusa:test REQ-FO-REQ002
func TestParseDOORS(t *testing.T) {
	xmlData := `<?xml version="1.0"?><REQ-IF><CORE-CONTENT><SPEC-OBJECTS>
<SPEC-OBJECT><VALUES>
<ATTRIBUTE-VALUE-STRING THE-VALUE="REQ-1"/>
<ATTRIBUTE-VALUE-STRING THE-VALUE="My Title"/>
<ATTRIBUTE-VALUE-STRING THE-VALUE="Some text"/>
</VALUES></SPEC-OBJECT>
</SPEC-OBJECTS></CORE-CONTENT></REQ-IF>`
	entries, err := ParseDOORS([]byte(xmlData))
	if err != nil {
		t.Fatalf("ParseDOORS: %v", err)
	}
	if len(entries) != 1 || entries[0].ID != "REQ-1" {
		t.Fatalf("unexpected: %v", entries)
	}
	if entries[0].Text != "Some text" {
		t.Errorf("text = %q, want 'Some text'", entries[0].Text)
	}
}

//fusa:test REQ-FO-REQ002
func TestParseDOORSInvalidXML(t *testing.T) {
	_, err := ParseDOORS([]byte("not xml"))
	if err == nil {
		t.Error("expected error for invalid XML")
	}
}

//fusa:test REQ-FO-REQ003
func TestExportDOORSRoundTrip(t *testing.T) {
	original := []Entry{{ID: "REQ-1", Title: "T1", Text: "text1"}, {ID: "REQ-2", Title: "T2"}}
	data, err := ExportDOORS(original)
	if err != nil {
		t.Fatalf("ExportDOORS: %v", err)
	}
	parsed, err := ParseDOORS(data)
	if err != nil {
		t.Fatalf("ParseDOORS: %v", err)
	}
	if len(parsed) != len(original) {
		t.Fatalf("count: want %d, got %d", len(original), len(parsed))
	}
	if parsed[0].ID != "REQ-1" || parsed[0].Text != "text1" {
		t.Errorf("round-trip mismatch: %+v", parsed[0])
	}
}

//fusa:test REQ-FO-REQ002
func TestParseCodebeamer(t *testing.T) {
	xmlData := `<?xml version="1.0"?><tracker>
<item id="REQ-1"><name>REQ-1</name><summary>My Title</summary><description>Desc</description></item>
</tracker>`
	entries, err := ParseCodebeamer([]byte(xmlData))
	if err != nil {
		t.Fatalf("ParseCodebeamer: %v", err)
	}
	if len(entries) != 1 || entries[0].ID != "REQ-1" || entries[0].Title != "My Title" {
		t.Errorf("unexpected: %+v", entries)
	}
}

//fusa:test REQ-FO-REQ003
func TestExportCodebeamerRoundTrip(t *testing.T) {
	original := []Entry{{ID: "REQ-1", Title: "T1", Text: "text"}}
	data, err := ExportCodebeamer(original)
	if err != nil {
		t.Fatalf("ExportCodebeamer: %v", err)
	}
	parsed, err := ParseCodebeamer(data)
	if err != nil {
		t.Fatalf("ParseCodebeamer: %v", err)
	}
	if len(parsed) != 1 || parsed[0].ID != "REQ-1" {
		t.Errorf("round-trip: %v", parsed)
	}
}

//fusa:test REQ-FO-REQ002
func TestParseJama(t *testing.T) {
	xmlData := `<?xml version="1.0"?><items>
<item id="REQ-1"><name>My Title</name><description>Desc</description></item>
</items>`
	entries, err := ParseJama([]byte(xmlData))
	if err != nil {
		t.Fatalf("ParseJama: %v", err)
	}
	if len(entries) != 1 || entries[0].ID != "REQ-1" {
		t.Errorf("unexpected: %v", entries)
	}
}

//fusa:test REQ-FO-REQ003
func TestExportJamaRoundTrip(t *testing.T) {
	original := []Entry{{ID: "REQ-1", Title: "T1", Text: "text"}}
	data, err := ExportJama(original)
	if err != nil {
		t.Fatalf("ExportJama: %v", err)
	}
	parsed, err := ParseJama(data)
	if err != nil {
		t.Fatalf("ParseJama: %v", err)
	}
	if len(parsed) != 1 || parsed[0].ID != "REQ-1" {
		t.Errorf("round-trip: %v", parsed)
	}
}

//fusa:test REQ-FO-REQ002
func TestParsePolarion(t *testing.T) {
	xmlData := `<?xml version="1.0"?><workitems>
<workitem id="REQ-1"><title>My Title</title><description>Desc</description></workitem>
</workitems>`
	entries, err := ParsePolarion([]byte(xmlData))
	if err != nil {
		t.Fatalf("ParsePolarion: %v", err)
	}
	if len(entries) != 1 || entries[0].ID != "REQ-1" || entries[0].Title != "My Title" {
		t.Errorf("unexpected: %v", entries)
	}
}

//fusa:test REQ-FO-REQ003
func TestExportPolarionRoundTrip(t *testing.T) {
	original := []Entry{{ID: "REQ-1", Title: "T1", Text: "text"}, {ID: "REQ-2", Title: "T2"}}
	data, err := ExportPolarion(original)
	if err != nil {
		t.Fatalf("ExportPolarion: %v", err)
	}
	parsed, err := ParsePolarion(data)
	if err != nil {
		t.Fatalf("ParsePolarion: %v", err)
	}
	if len(parsed) != 2 {
		t.Fatalf("want 2, got %d", len(parsed))
	}
}

// TestParseCSVParentColumn verifies that a CSV with a "parent" column populates
// Entry.Parent correctly on round-trip.
//
//fusa:test REQ-FO-REQ001
func TestParseCSVParentColumn(t *testing.T) {
	data := "id,title,level,parent,priority\nHLR-001,Safety init,HLR,,MUST\nLLR-001,Init sequence,LLR,HLR-001,MUST\n"
	entries, err := ParseCSV(strings.NewReader(data))
	if err != nil {
		t.Fatalf("ParseCSV: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("want 2 entries, got %d", len(entries))
	}
	if entries[0].Parent != "" {
		t.Errorf("HLR parent should be empty, got %q", entries[0].Parent)
	}
	if entries[1].Parent != "HLR-001" {
		t.Errorf("LLR parent = %q, want HLR-001", entries[1].Parent)
	}
	if entries[1].Level != "LLR" {
		t.Errorf("LLR level = %q, want LLR", entries[1].Level)
	}

	// Round-trip: render and re-parse.
	var buf bytes.Buffer
	if err := RenderCSV(&buf, entries); err != nil {
		t.Fatalf("RenderCSV: %v", err)
	}
	reparsed, err := ParseCSV(&buf)
	if err != nil {
		t.Fatalf("ParseCSV (reparsed): %v", err)
	}
	if len(reparsed) != 2 {
		t.Fatalf("reparsed: want 2, got %d", len(reparsed))
	}
	if reparsed[1].Parent != "HLR-001" {
		t.Errorf("reparsed LLR parent = %q, want HLR-001", reparsed[1].Parent)
	}
}
