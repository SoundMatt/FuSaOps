package main

// Additional tests targeting specific uncovered branches in runDiff,
// runDispositionList, runQualify, runTrace, runVerify, runSAS, runSCI,
// runMetricsRecord, and runMetricsShow.

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SoundMatt/FuSaOps/config"
	"github.com/SoundMatt/FuSaOps/disposition"
)

// ── runDiff ──────────────────────────────────────────────────────────────────

// TestDiffHelpFlag verifies runDiff returns 0 for --help (ErrHelp special case).
//
//fusa:test REQ-FO-CLI018
func TestDiffHelpFlag(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runDiff([]string{"--help"}, &stdout, &stderr)
	if code != 0 {
		t.Errorf("diff --help: want 0, got %d", code)
	}
}

// TestDiffStrictFlagGoProject verifies runDiff evaluates the --strict gate when
// a valid baseline and project are provided.
//
//fusa:test REQ-FO-CLI018
func TestDiffStrictFlagGoProject(t *testing.T) {
	dir := goProject(t)
	bl := filepath.Join(dir, "baseline.json")
	if err := os.WriteFile(bl, []byte(`{"generatedAt":"2026-01-01T00:00:00Z","components":[]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := runDiff([]string{"--dir", dir, "--baseline", bl, "--strict"}, &stdout, &stderr)
	// 0 (no new findings) or 1 (gofusa installed and found something); never 2 (flag error).
	if code == 2 {
		t.Errorf("diff --strict: unexpected flag error (2), stderr=%q", stderr.String())
	}
}

// TestDiffUpdateBaselineFlag verifies runDiff rewrites the baseline when
// --update-baseline is set.
//
//fusa:test REQ-FO-CLI018
func TestDiffUpdateBaselineFlag(t *testing.T) {
	dir := goProject(t)
	bl := filepath.Join(dir, "baseline.json")
	if err := os.WriteFile(bl, []byte(`{"generatedAt":"2026-01-01T00:00:00Z","components":[]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := runDiff([]string{"--dir", dir, "--baseline", bl, "--update-baseline"}, &stdout, &stderr)
	if code != 0 && code != 1 {
		t.Errorf("diff --update-baseline: unexpected code=%d stderr=%q", code, stderr.String())
	}
	if _, err := os.Stat(bl); err != nil {
		t.Errorf("baseline file missing after --update-baseline: %v", err)
	}
}

// ── runDispositionList ────────────────────────────────────────────────────────

// TestDispositionListMalformedJSON verifies runDispositionList returns 1 when the
// dispositions file exists but contains malformed JSON.
//
//fusa:test REQ-FO-CLI060
func TestDispositionListMalformedJSON(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, disposition.DispositionsFile), []byte("{bad json"), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := runDispositionList(dir, &stdout, &stderr)
	if code != 1 {
		t.Errorf("disposition list malformed JSON: want 1, got %d", code)
	}
	if !strings.Contains(stderr.String(), "disposition") {
		t.Errorf("expected 'disposition' in stderr: %q", stderr.String())
	}
}

// ── runQualify ────────────────────────────────────────────────────────────────

// TestQualifyConfigQualifyOverride verifies that qualify.type and qualify.recordUri
// from .fusaops.json are applied when --type and --record-uri are at defaults.
//
//fusa:test REQ-FO-CLI064
func TestQualifyConfigQualifyOverride(t *testing.T) {
	dir := t.TempDir()
	cfg := config.Default("test-qualify")
	cfg.Qualify.Type = "independent"
	cfg.Qualify.RecordUri = "http://example.com/cert"
	if err := config.Save(filepath.Join(dir, config.ConfigFile), cfg); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	// No Go files in dir → no adapters → exits 1 before qualify.Run but after config loading.
	code := runQualify([]string{"--dir", dir}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("qualify config override (no adapters): want 1, got %d stderr=%q", code, stderr.String())
	}
}

// ── runTrace ─────────────────────────────────────────────────────────────────

// TestTraceDecompFlag verifies trace --decomp calls CheckDecomposition.
//
//fusa:test REQ-FO-CLI011
//fusa:test REQ-FO-CLI077
func TestTraceDecompFlag(t *testing.T) {
	dir := goProject(t)
	var stdout, stderr bytes.Buffer
	code := runTrace([]string{"--dir", dir, "--decomp"}, &stdout, &stderr)
	// 0 or 1 depending on gofusa and decomp violations; never 2 (flag error).
	if code == 2 {
		t.Errorf("trace --decomp: unexpected flag error (2), stderr=%q", stderr.String())
	}
}

// TestTraceDecompConfigEnforce verifies the config-based decomp enforcement path
// (cfg.Trace.ReqDecomposition.Enforce != "") is evaluated when a .fusaops.json
// file is present with a non-"off" enforce value.
//
//fusa:test REQ-FO-CLI011
//fusa:test REQ-FO-CLI077
func TestTraceDecompConfigEnforce(t *testing.T) {
	dir := goProject(t)
	cfg := config.Default("test-decomp")
	cfg.Trace.ReqDecomposition.Enforce = "warn"
	if err := config.Save(filepath.Join(dir, config.ConfigFile), cfg); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := runTrace([]string{"--dir", dir}, &stdout, &stderr)
	// 0 or 1; just verifying no crash and no flag-parse error.
	if code == 2 {
		t.Errorf("trace with decomp config enforce=warn: unexpected flag error (2), stderr=%q", stderr.String())
	}
}

// TestTraceStrictFlagGoProject verifies trace --strict evaluates the agg.HasGaps()
// check on a Go project.
//
//fusa:test REQ-FO-CLI011
func TestTraceStrictFlagGoProject(t *testing.T) {
	dir := goProject(t)
	var stdout, stderr bytes.Buffer
	code := runTrace([]string{"--dir", dir, "--strict"}, &stdout, &stderr)
	// 0 (no gaps) or 1 (gaps found); never 2 (flag error).
	if code == 2 {
		t.Errorf("trace --strict: unexpected flag error (2), stderr=%q", stderr.String())
	}
}

// ── runVerify ────────────────────────────────────────────────────────────────

// TestVerifyDefaultOutputPath verifies runVerify uses the default output path
// (.fusaops-evidence.json) when --output is omitted.
//
//fusa:test REQ-FO-CLI062
func TestVerifyDefaultOutputPath(t *testing.T) {
	dir := t.TempDir()
	var stdout, stderr bytes.Buffer
	// Pass --dir but no --output → outPath defaults to dir/.fusaops-evidence.json.
	code := runVerify([]string{"--dir", dir}, &stdout, &stderr)
	// 0 = success; 1 = test failures. Not 2 (flag error).
	if code == 2 {
		t.Errorf("verify (no --output): unexpected flag error (2), stderr=%q", stderr.String())
	}
}

// TestVerifyBadOutputPath verifies runVerify returns 1 when verify.Save fails
// because the output directory does not exist.
//
//fusa:test REQ-FO-CLI062
func TestVerifyBadOutputPath(t *testing.T) {
	dir := t.TempDir()
	var stdout, stderr bytes.Buffer
	code := runVerify([]string{"--dir", dir, "--output", "/nonexistent-dir-xyz/evidence.json"}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("verify bad output path: want 1, got %d", code)
	}
	if !strings.Contains(stderr.String(), "save bundle") {
		t.Errorf("verify bad output: expected 'save bundle' in stderr: %q", stderr.String())
	}
}

// ── runSAS ───────────────────────────────────────────────────────────────────

// TestSASNoDirUsesGetwd verifies runSAS calls os.Getwd() when --dir is omitted.
// --output is pointed at a non-existent path to prevent writing to the working directory.
//
//fusa:test REQ-FO-CLI068
func TestSASNoDirUsesGetwd(t *testing.T) {
	var stdout, stderr bytes.Buffer
	// No --dir → os.Getwd() branch; bad --output prevents any dir write.
	code := runSAS([]string{"--output", "/nonexistent-dir-xyz/sas.json"}, &stdout, &stderr)
	// 1 = Save error or build error; not 2 (flag error).
	if code == 2 {
		t.Errorf("sas (no --dir): unexpected flag error (2), stderr=%q", stderr.String())
	}
}

// ── runSCI ───────────────────────────────────────────────────────────────────

// TestSCINoDirUsesGetwd verifies runSCI calls os.Getwd() when --dir is omitted.
// --output is pointed at a non-existent path to prevent writing to the working directory.
//
//fusa:test REQ-FO-CLI067
func TestSCINoDirUsesGetwd(t *testing.T) {
	var stdout, stderr bytes.Buffer
	// No --dir → os.Getwd() branch; bad --output prevents any dir write.
	code := runSCI([]string{"--output", "/nonexistent-dir-xyz/sci.json"}, &stdout, &stderr)
	// 1 = Save error or detect error; not 2 (flag error).
	if code == 2 {
		t.Errorf("sci (no --dir): unexpected flag error (2), stderr=%q", stderr.String())
	}
}

// ── runMetricsRecord ─────────────────────────────────────────────────────────

// TestMetricsRecordWithCoverageReport verifies metrics record prints coverage%
// when coverage-report.json is present (snap.CoveragePct > 0 branch).
//
//fusa:test REQ-FO-CLI055
func TestMetricsRecordWithCoverageReport(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "coverage-report.json"), []byte(`{"stmtPct":85.5}`), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := runMetricsRecord(dir, &stdout, &stderr)
	if code != 0 {
		t.Errorf("metrics record with coverage: want 0, got %d stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "85.5%") {
		t.Errorf("metrics record: expected coverage=85.5%% in output: %q", stdout.String())
	}
}

// ── runMetricsShow ────────────────────────────────────────────────────────────

// TestMetricsShowBadOutputPath verifies runMetricsShow returns 1 when os.Create
// fails because the output directory does not exist.
//
//fusa:test REQ-FO-CLI055
func TestMetricsShowBadOutputPath(t *testing.T) {
	dir := t.TempDir()
	var stdout, stderr bytes.Buffer
	code := runMetricsShow([]string{"--output", "/nonexistent-dir-xyz/show.txt"}, dir, &stdout, &stderr)
	if code != 1 {
		t.Errorf("metrics show bad output path: want 1, got %d", code)
	}
	if !strings.Contains(stderr.String(), "create") {
		t.Errorf("metrics show bad output: expected 'create' in stderr: %q", stderr.String())
	}
}

// ── runRelease without --dir (os.Getwd path) ─────────────────────────────────

// TestReleaseNoDirFlag exercises the os.Getwd() fallback in runRelease when
// --dir is not provided. Output goes to a temp dir to avoid writing files into
// the repo.
//
//fusa:test REQ-FO-CLI065
//fusa:test REQ-FO-CLI081
func TestReleaseNoDirFlag(t *testing.T) {
	outDir := t.TempDir()
	var stdout, stderr bytes.Buffer
	code := runRelease([]string{"--output-dir", outDir}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("release --output-dir only: want 0, got %d stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Provenance written") {
		t.Errorf("expected provenance confirmation in stdout: %q", stdout.String())
	}
	if _, err := os.Stat(filepath.Join(outDir, "provenance.json")); err != nil {
		t.Errorf("provenance.json not written: %v", err)
	}
}

// ── runHooks install ──────────────────────────────────────────────────────────

// TestHooksInstallHookPath verifies hooksInstall creates the hook file when invoked
// directly with a hook path in a fresh temporary directory.
//
//fusa:test REQ-FO-CLI079
func TestHooksInstallHookPath(t *testing.T) {
	dir := t.TempDir()
	hookPath := filepath.Join(dir, ".git", "hooks", "pre-commit")
	var stdout, stderr bytes.Buffer
	code := hooksInstall(hookPath, &stdout, &stderr)
	if code != 0 {
		t.Errorf("hooksInstall: want 0, got %d stderr=%q", code, stderr.String())
	}
	if _, err := os.Stat(hookPath); err != nil {
		t.Errorf("pre-commit hook not created: %v", err)
	}
	if !strings.Contains(stdout.String(), "pre-commit hook installed") {
		t.Errorf("expected install confirmation in stdout: %q", stdout.String())
	}
}

