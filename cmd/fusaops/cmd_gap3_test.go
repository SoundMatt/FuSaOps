package main

// Additional tests targeting uncovered branches in runDisposition,
// runDispositionAdd, runDispositionShow, and runTARA.

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/SoundMatt/FuSaOps/disposition"
	"github.com/SoundMatt/FuSaOps/tara"
)

// ── runDisposition ────────────────────────────────────────────────────────────

// TestDispositionFlagParseError verifies runDisposition returns 2 on flag parse
// failure, covering the fs.Parse error branch.
//
//fusa:test REQ-FO-CLI060
func TestDispositionFlagParseError(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runDisposition([]string{"--bogus-flag-xyz"}, &stdout, &stderr)
	if code != 2 {
		t.Errorf("disposition --bogus-flag: want 2, got %d", code)
	}
}

// ── runDispositionAdd ─────────────────────────────────────────────────────────

// TestDispositionAddFlagParseError verifies runDispositionAdd returns 2 on flag
// parse failure.
//
//fusa:test REQ-FO-CLI060
func TestDispositionAddFlagParseError(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runDispositionAdd([]string{"--bogus-flag-xyz"}, ".", &stdout, &stderr)
	if code != 2 {
		t.Errorf("disposition add --bogus-flag: want 2, got %d", code)
	}
}

// TestDispositionAddLoadError verifies runDispositionAdd returns 1 when the
// dispositions file path exists as a directory (non-IsNotExist read error).
//
//fusa:test REQ-FO-CLI060
func TestDispositionAddLoadError(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, disposition.DispositionsFile), 0o750); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := runDispositionAdd([]string{
		"--rule", "R001", "--reviewer", "dev", "--rationale", "ok",
	}, dir, &stdout, &stderr)
	if code != 1 {
		t.Errorf("disposition add load error: want 1, got %d (stderr=%q)", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "disposition") {
		t.Errorf("expected 'disposition' in stderr: %q", stderr.String())
	}
}

// TestDispositionAddSaveError verifies runDispositionAdd returns 1 when Save
// fails because the project directory is not writable.
//
//fusa:test REQ-FO-CLI060
func TestDispositionAddSaveError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("os.Chmod read-only semantics not enforced on Windows")
	}
	if os.Getuid() == 0 {
		t.Skip("cannot test permission denied as root")
	}
	dir := t.TempDir()
	if err := os.Chmod(dir, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })
	var stdout, stderr bytes.Buffer
	code := runDispositionAdd([]string{
		"--rule", "R001", "--reviewer", "dev", "--rationale", "ok",
	}, dir, &stdout, &stderr)
	if code != 1 {
		t.Errorf("disposition add save error: want 1, got %d (stderr=%q)", code, stderr.String())
	}
}

// ── runDispositionShow ────────────────────────────────────────────────────────

// TestDispositionShowLoadError verifies runDispositionShow returns 1 when the
// dispositions file path exists as a directory (non-IsNotExist read error).
//
//fusa:test REQ-FO-CLI060
func TestDispositionShowLoadError(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, disposition.DispositionsFile), 0o750); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := runDispositionShow([]string{"--rule", "R001"}, dir, &stdout, &stderr)
	if code != 1 {
		t.Errorf("disposition show load error: want 1, got %d (stderr=%q)", code, stderr.String())
	}
}

// TestDispositionShowWithLanguage verifies runDispositionShow prints the
// Language field when a disposition entry has a language set, covering the
// "e.Language != """ branch.
//
//fusa:test REQ-FO-CLI060
func TestDispositionShowWithLanguage(t *testing.T) {
	dir := t.TempDir()
	var addOut, addErr bytes.Buffer
	_ = runDispositionAdd([]string{
		"--rule", "R-LANG-001", "--lang", "go", "--reviewer", "dev", "--rationale", "ok",
	}, dir, &addOut, &addErr)
	var stdout, stderr bytes.Buffer
	code := runDispositionShow([]string{"--rule", "R-LANG-001"}, dir, &stdout, &stderr)
	if code != 0 {
		t.Errorf("disposition show with lang: want 0, got %d (stderr=%q)", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Language") {
		t.Errorf("expected 'Language' in output: %q", stdout.String())
	}
}

// TestDispositionShowWithReference verifies runDispositionShow prints the
// Reference field when a disposition entry has a reference set, covering the
// "e.Reference != """ branch.
//
//fusa:test REQ-FO-CLI060
func TestDispositionShowWithReference(t *testing.T) {
	dir := t.TempDir()
	var addOut, addErr bytes.Buffer
	_ = runDispositionAdd([]string{
		"--rule", "R-REF-001", "--reviewer", "dev", "--rationale", "ok",
		"--ref", "https://issue.example/42",
	}, dir, &addOut, &addErr)
	var stdout, stderr bytes.Buffer
	code := runDispositionShow([]string{"--rule", "R-REF-001"}, dir, &stdout, &stderr)
	if code != 0 {
		t.Errorf("disposition show with ref: want 0, got %d (stderr=%q)", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Reference") {
		t.Errorf("expected 'Reference' in output: %q", stdout.String())
	}
}

// ── runTARA ───────────────────────────────────────────────────────────────────

// TestTARANoDirFlag verifies that runTARA uses os.Getwd() when --dir is
// omitted, covering the "projectRoot == ""  → os.Getwd()" branch.
//
//fusa:test REQ-FO-CLI069
func TestTARANoDirFlag(t *testing.T) {
	outPath := filepath.Join(t.TempDir(), tara.ReportFile)
	var stdout, stderr bytes.Buffer
	code := runTARA([]string{"--output", outPath}, &stdout, &stderr)
	// hardcoded TARA scenarios always have critical items → exit 1
	if code != 1 {
		t.Errorf("tara no-dir: want 1 (critical items), got %d (stderr=%q)", code, stderr.String())
	}
}
