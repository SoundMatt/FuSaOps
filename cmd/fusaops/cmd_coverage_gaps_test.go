package main

// Tests covering the highest-impact uncovered CLI dispatch paths, raising
// cmd/fusaops coverage above the 80% gate.
//
// These tests target functions with <70% coverage that are not already covered
// by the existing test files: runDispositionList (with entries), runReqExport
// (bad flag, output file), and runServeMulti (bad JSON config, invalid dirs).

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var _ = json.Marshal // ensure import is used

// ── fusaops disposition list (with entries) ───────────────────────────────────

// TestDispositionListWithEntries verifies listing shows recorded disposition entries.
// The path through runDispositionList when the log is non-empty was not covered.
//
//fusa:test REQ-FO-CLI060
func TestDispositionListWithEntries(t *testing.T) {
	dir := t.TempDir()
	// First add an entry via the CLI, then list it.
	var stdout, stderr bytes.Buffer
	addCode := runDispositionAdd(
		[]string{"--rule", "LINT001", "--lang", "go", "--reviewer", "Alice", "--rationale", "accepted"},
		dir, &stdout, &stderr,
	)
	if addCode != 0 {
		t.Fatalf("disposition add: code=%d err=%q", addCode, stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	code := runDispositionList(dir, &stdout, &stderr)
	if code != 0 {
		t.Errorf("disposition list: code=%d err=%q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "LINT001") {
		t.Errorf("disposition list output missing LINT001:\n%s", stdout.String())
	}
}

// ── fusaops req export ────────────────────────────────────────────────────────

// emptyReqs is the canonical empty registry JSON for req export tests.
const emptyReqs = `{"requirements":[]}`

// oneReq is a minimal single-entry registry for req export tests.
const oneReq = `{"requirements":[{"id":"REQ-001","title":"Test req"}]}`

// TestReqExportBadFlag verifies req export with a bad flag returns exit 2.
//
//fusa:test REQ-FO-CLI052
func TestReqExportBadFlag(t *testing.T) {
	dir := t.TempDir()
	var stdout, stderr bytes.Buffer
	code := runReqExport([]string{"--notaflag"}, dir, &stdout, &stderr)
	if code != 2 {
		t.Errorf("req export --notaflag: code=%d, want 2", code)
	}
}

// TestReqExportToFile verifies req export writes to a file when --output is set.
//
//fusa:test REQ-FO-CLI052
func TestReqExportToFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".fusa-reqs.json"), []byte(oneReq), 0o600); err != nil {
		t.Fatal(err)
	}
	outFile := filepath.Join(dir, "export.csv")
	var stdout, stderr bytes.Buffer
	code := runReqExport([]string{"--format", "csv", "--output", outFile}, dir, &stdout, &stderr)
	if code != 0 {
		t.Errorf("req export to file: code=%d err=%q", code, stderr.String())
	}
	if _, err := os.Stat(outFile); err != nil {
		t.Errorf("req export output file not written: %v", err)
	}
}

// TestReqExportBadFormat verifies req export with unknown format returns exit 2.
//
//fusa:test REQ-FO-CLI052
func TestReqExportBadFormat(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".fusa-reqs.json"), []byte(emptyReqs), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := runReqExport([]string{"--format", "xml"}, dir, &stdout, &stderr)
	if code != 2 {
		t.Errorf("req export xml: code=%d, want 2", code)
	}
}

// TestReqExportPolarion verifies req export in polarion format.
//
//fusa:test REQ-FO-CLI052
func TestReqExportPolarion(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".fusa-reqs.json"), []byte(emptyReqs), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := runReqExport([]string{"--format", "polarion"}, dir, &stdout, &stderr)
	if code != 0 {
		t.Errorf("req export polarion: code=%d err=%q", code, stderr.String())
	}
}

// TestReqExportCodebeamer verifies req export in codebeamer format.
//
//fusa:test REQ-FO-CLI052
func TestReqExportCodebeamer(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".fusa-reqs.json"), []byte(emptyReqs), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := runReqExport([]string{"--format", "codebeamer"}, dir, &stdout, &stderr)
	if code != 0 {
		t.Errorf("req export codebeamer: code=%d err=%q", code, stderr.String())
	}
}

// TestReqExportJama verifies req export in jama format.
//
//fusa:test REQ-FO-CLI052
func TestReqExportJama(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".fusa-reqs.json"), []byte(emptyReqs), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := runReqExport([]string{"--format", "jama"}, dir, &stdout, &stderr)
	if code != 0 {
		t.Errorf("req export jama: code=%d err=%q", code, stderr.String())
	}
}

// ── fusaops serve --projects (runServeMulti) ──────────────────────────────────

// TestServeMultiBadJSON verifies runServeMulti with an invalid JSON config returns exit 1.
//
//fusa:test REQ-FO-CLI030
//fusa:test REQ-FO-MPJ007
func TestServeMultiBadJSON(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "projects.json")
	if err := os.WriteFile(cfgPath, []byte("{bad json"), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := runServeMulti(cfgPath, ":0", "http", "", "",
		"", "", "", "", "", "", 0, &stdout, &stderr)
	if code != 1 {
		t.Errorf("runServeMulti bad JSON: code=%d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "projects config") {
		t.Errorf("runServeMulti bad JSON: expected 'projects config' in stderr: %q", stderr.String())
	}
}

// TestServeMultiInvalidProjectDirs verifies runServeMulti exits 1 when project
// directories are missing (ValidateProjectDirs returns errors).
//
//fusa:test REQ-FO-CLI030
//fusa:test REQ-FO-MPJ007
func TestServeMultiInvalidProjectDirs(t *testing.T) {
	dir := t.TempDir()
	cfg := map[string]interface{}{
		"projects": []map[string]string{
			{"name": "alpha", "dir": "/nonexistent/alpha-xyz-missing"},
		},
	}
	data, _ := json.Marshal(cfg)
	cfgPath := filepath.Join(dir, "projects.json")
	if err := os.WriteFile(cfgPath, data, 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := runServeMulti(cfgPath, ":0", "http", "", "",
		"", "", "", "", "", "", 0, &stdout, &stderr)
	if code != 1 {
		t.Errorf("runServeMulti invalid dirs: code=%d, want 1", code)
	}
}
