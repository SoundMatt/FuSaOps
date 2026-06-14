package coverage

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const sampleProfile = `mode: set
pkg/foo.go:1.10,3.2 2 1
pkg/foo.go:5.10,7.2 2 0
pkg/bar.go:10.10,12.2 1 1
`

//fusa:test REQ-FO-COV002
func TestParse(t *testing.T) {
	blocks, err := Parse(strings.NewReader(sampleProfile))
	if err != nil {
		t.Fatal(err)
	}
	if len(blocks) != 3 {
		t.Fatalf("want 3 blocks, got %d", len(blocks))
	}
	if blocks[0].File != "pkg/foo.go" {
		t.Errorf("block[0].File = %q, want pkg/foo.go", blocks[0].File)
	}
	if blocks[0].Count != 1 {
		t.Errorf("block[0].Count = %d, want 1", blocks[0].Count)
	}
	if blocks[1].Count != 0 {
		t.Errorf("block[1].Count = %d, want 0 (uncovered)", blocks[1].Count)
	}
}

//fusa:test REQ-FO-COV002
func TestParseEmpty(t *testing.T) {
	blocks, err := Parse(strings.NewReader("mode: set\n"))
	if err != nil {
		t.Fatal(err)
	}
	if len(blocks) != 0 {
		t.Fatalf("want 0 blocks, got %d", len(blocks))
	}
}

//fusa:test REQ-FO-COV001
func TestAnalyseDALB(t *testing.T) {
	blocks, _ := Parse(strings.NewReader(sampleProfile))
	rep := Analyse(blocks, DALB)
	if rep.DAL != DALB {
		t.Errorf("DAL = %s, want DAL-B", rep.DAL)
	}
	// 3 covered stmts out of 5 total (2+2+1 stmts; foo.go block[1] has 0 count)
	if rep.StmtTotal != 5 {
		t.Errorf("StmtTotal = %d, want 5", rep.StmtTotal)
	}
	if rep.StmtCovered != 3 {
		t.Errorf("StmtCovered = %d, want 3", rep.StmtCovered)
	}
	if !rep.StmtRequired {
		t.Error("StmtRequired should be true")
	}
	if !rep.DecisionRequired {
		t.Error("DecisionRequired should be true for DAL-B")
	}
	if rep.MCDCRequired {
		t.Error("MCDCRequired should be false for DAL-B")
	}
	if len(rep.Gaps) == 0 {
		t.Error("want at least one gap")
	}
}

//fusa:test REQ-FO-COV001
func TestAnalyseDALA(t *testing.T) {
	blocks, _ := Parse(strings.NewReader(sampleProfile))
	rep := Analyse(blocks, DALA)
	if !rep.MCDCRequired {
		t.Error("MCDCRequired should be true for DAL-A")
	}
	if rep.MCDCNote == "" {
		t.Error("MCDCNote should be set for DAL-A")
	}
}

//fusa:test REQ-FO-COV001
func TestAnalyseDALC(t *testing.T) {
	blocks, _ := Parse(strings.NewReader(sampleProfile))
	rep := Analyse(blocks, DALC)
	if rep.DecisionRequired {
		t.Error("DecisionRequired should be false for DAL-C")
	}
	if rep.MCDCRequired {
		t.Error("MCDCRequired should be false for DAL-C")
	}
}

//fusa:test REQ-FO-COV001
func TestAnalyseFullCoverage(t *testing.T) {
	profile := "mode: set\npkg/x.go:1.10,3.2 2 1\n"
	blocks, _ := Parse(strings.NewReader(profile))
	rep := Analyse(blocks, DALB)
	if len(rep.Gaps) != 0 {
		t.Errorf("expected no gaps for 100%% coverage, got %v", rep.Gaps)
	}
	if rep.StmtPct != 100 {
		t.Errorf("StmtPct = %.1f, want 100", rep.StmtPct)
	}
}

//fusa:test REQ-FO-COV002
func TestBuildFromFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "coverage.out")
	if err := os.WriteFile(path, []byte(sampleProfile), 0o600); err != nil {
		t.Fatal(err)
	}
	rep, err := BuildFromFile(path, DALB)
	if err != nil {
		t.Fatalf("BuildFromFile: %v", err)
	}
	if rep.StmtTotal != 5 {
		t.Errorf("StmtTotal = %d, want 5", rep.StmtTotal)
	}
}

//fusa:test REQ-FO-COV002
func TestBuildFromFileMissing(t *testing.T) {
	_, err := BuildFromFile("/nonexistent/coverage.out", DALB)
	if err == nil {
		t.Error("expected error for missing file")
	}
}

//fusa:test REQ-FO-COV003
func TestRenderText(t *testing.T) {
	blocks, _ := Parse(strings.NewReader(sampleProfile))
	rep := Analyse(blocks, DALB)
	var buf bytes.Buffer
	if err := Render(&buf, rep, "text"); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "DO-178C") {
		t.Error("text output missing DO-178C header")
	}
	if !strings.Contains(out, "DAL-B") {
		t.Error("text output missing DAL-B")
	}
	if !strings.Contains(out, "Statement coverage") {
		t.Error("text output missing statement coverage line")
	}
}

//fusa:test REQ-FO-COV003
func TestRenderJSON(t *testing.T) {
	blocks, _ := Parse(strings.NewReader(sampleProfile))
	rep := Analyse(blocks, DALB)
	var buf bytes.Buffer
	if err := Render(&buf, rep, "json"); err != nil {
		t.Fatal(err)
	}
	var out Report
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("invalid JSON output: %v", err)
	}
	if out.StmtTotal != 5 {
		t.Errorf("JSON StmtTotal = %d, want 5", out.StmtTotal)
	}
}

//fusa:test REQ-FO-COV003
func TestRenderJSONDefault(t *testing.T) {
	blocks, _ := Parse(strings.NewReader(sampleProfile))
	rep := Analyse(blocks, DALC)
	var buf bytes.Buffer
	if err := Render(&buf, rep, ""); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "{") {
		t.Error("empty format should default to JSON")
	}
}

//fusa:test REQ-FO-COV003
func TestRenderUnknownFormat(t *testing.T) {
	blocks, _ := Parse(strings.NewReader(sampleProfile))
	rep := Analyse(blocks, DALC)
	err := Render(&bytes.Buffer{}, rep, "xml")
	if err == nil {
		t.Error("expected error for unknown format")
	}
}

//fusa:test REQ-FO-COV003
func TestRenderMarkdown(t *testing.T) {
	blocks, _ := Parse(strings.NewReader(sampleProfile))
	rep := Analyse(blocks, DALB)
	var buf bytes.Buffer
	if err := Render(&buf, rep, "markdown"); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "DO-178C") {
		t.Error("markdown missing DO-178C header")
	}
	if !strings.Contains(out, "DAL-B") {
		t.Error("markdown missing DAL-B")
	}
	if !strings.Contains(out, "Statement coverage") {
		t.Error("markdown missing statement coverage row")
	}
}

//fusa:test REQ-FO-COV003
func TestRenderMarkdownAlias(t *testing.T) {
	blocks, _ := Parse(strings.NewReader(sampleProfile))
	rep := Analyse(blocks, DALB)
	var buf bytes.Buffer
	if err := Render(&buf, rep, "md"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "DO-178C") {
		t.Error("md alias missing content")
	}
}

//fusa:test REQ-FO-COV003
func TestRenderMarkdownFullCoverage(t *testing.T) {
	profile := "mode: set\npkg/x.go:1.10,3.2 2 1\n"
	blocks, _ := Parse(strings.NewReader(profile))
	rep := Analyse(blocks, DALC)
	var buf bytes.Buffer
	if err := Render(&buf, rep, "markdown"); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if strings.Contains(out, "Coverage gaps") {
		t.Error("full coverage should not show gaps section")
	}
}

//fusa:test REQ-FO-COV001
func TestAnalyseDecisionCoverage(t *testing.T) {
	// 3 blocks: 2 covered, 1 not → decision pct ~66.7%
	blocks, _ := Parse(strings.NewReader(sampleProfile))
	rep := Analyse(blocks, DALB)
	if rep.DecisionPct <= 0 {
		t.Error("DecisionPct should be positive")
	}
	if rep.DecisionNote == "" {
		t.Error("DecisionNote should be set")
	}
}
