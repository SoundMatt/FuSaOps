package coverage

// Gap tests covering:
// - BuildFromFile parse error when the scanner buffer overflows (coverage.go:185)
// - renderText MCDC required block (coverage.go:218-220)
// - renderMarkdown yellow badge when StmtPct is in 80-100 range (coverage.go:239-240)
// - renderMarkdown MCDC required block (coverage.go:250-251)

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestBuildFromFileParseError verifies BuildFromFile returns an error when the
// coverage file contains a line exceeding bufio.Scanner's token buffer (64 KiB),
// which makes Parse return bufio.ErrTooLong, covering coverage.go:185.16,187.3.
//
//fusa:test REQ-FO-COV002
func TestBuildFromFileParseError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cov.out")
	// A line longer than 64 KiB triggers scanner.Err() == bufio.ErrTooLong.
	longLine := strings.Repeat("x", 65*1024) + "\n"
	if err := os.WriteFile(path, []byte(longLine), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := BuildFromFile(path, DALB)
	if err == nil {
		t.Fatal("BuildFromFile: expected error for overlong line, got nil")
	}
}

// TestRenderTextMCDCRequired verifies renderText prints the MC/DC manual-check
// note when rep.MCDCRequired is true, covering coverage.go:218.22,221.3.
//
//fusa:test REQ-FO-COV003
func TestRenderTextMCDCRequired(t *testing.T) {
	rep := &Report{
		DAL:          DALA,
		MCDCRequired: true,
		MCDCNote:     "MC/DC structural coverage required at DAL-A",
		StmtTotal:    10,
		StmtCovered:  10,
		StmtPct:      100.0,
	}
	var buf bytes.Buffer
	if err := Render(&buf, rep, "text"); err != nil {
		t.Fatalf("Render text MCDC: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "MANUAL CHECK REQUIRED") {
		t.Errorf("renderText: expected 'MANUAL CHECK REQUIRED' for MCDCRequired=true:\n%s", out)
	}
}

// TestRenderMarkdownYellowBadge verifies the yellow badge (🟡) is emitted when
// StmtPct is between 80 and 100, covering the else-if branch at coverage.go:239.
//
//fusa:test REQ-FO-COV003
func TestRenderMarkdownYellowBadge(t *testing.T) {
	rep := &Report{
		DAL:         DALB,
		StmtPct:     85.0,
		StmtTotal:   20,
		StmtCovered: 17,
	}
	var buf bytes.Buffer
	if err := Render(&buf, rep, "markdown"); err != nil {
		t.Fatalf("Render markdown yellow badge: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "🟡") {
		t.Errorf("renderMarkdown: expected yellow badge 🟡 for StmtPct=85.0:\n%s", out)
	}
}

// TestRenderMarkdownMCDCRequired verifies the MC/DC cell reads
// "MANUAL CHECK REQUIRED" when rep.MCDCRequired is true, covering coverage.go:250.
//
//fusa:test REQ-FO-COV003
func TestRenderMarkdownMCDCRequired(t *testing.T) {
	rep := &Report{
		DAL:          DALA,
		MCDCRequired: true,
		StmtPct:      100.0,
		StmtTotal:    5,
		StmtCovered:  5,
	}
	var buf bytes.Buffer
	if err := Render(&buf, rep, "markdown"); err != nil {
		t.Fatalf("Render markdown MCDC: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "MANUAL CHECK REQUIRED") {
		t.Errorf("renderMarkdown: expected 'MANUAL CHECK REQUIRED' for MCDCRequired=true:\n%s", out)
	}
}
