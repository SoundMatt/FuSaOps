package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The v0.2 roll-up commands are invariant to whether the x-FuSa tools are on
// PATH: an absent tool yields a skipped component, an installed one runs against
// the bare temp project. Either way the command must build its artefact and
// exit cleanly, so these tests assert on structure, not on tool presence.

//fusa:test REQ-FO-CLI011
func TestTraceCommand(t *testing.T) {
	dir := goProject(t)
	code, stdout, errb := runArgs(t, "trace", "--dir", dir, "--format", "text")
	if code != 0 {
		t.Fatalf("trace: code=%d err=%q", code, errb)
	}
	if !strings.Contains(stdout, "Cross-Language Traceability") {
		t.Errorf("trace stdout: %q", stdout)
	}
}

//fusa:test REQ-FO-CLI011
func TestTraceWritesFile(t *testing.T) {
	dir := goProject(t)
	out := filepath.Join(dir, "trace.json")
	code, stdout, errb := runArgs(t, "trace", "--dir", dir, "--format", "json", "--output", out)
	if code != 0 {
		t.Fatalf("trace: code=%d err=%q", code, errb)
	}
	if !strings.Contains(errb, "Wrote") {
		t.Errorf("trace stderr: %q", errb)
	}
	if stdout != "" {
		t.Errorf("trace stdout must be empty when --output given: %q", stdout)
	}
	if _, err := os.Stat(out); err != nil {
		t.Errorf("trace file not written: %v", err)
	}
}

//fusa:test REQ-FO-CLI011
func TestTraceBadFlag(t *testing.T) {
	if code, _, _ := runArgs(t, "trace", "--bogus"); code != 2 {
		t.Errorf("trace --bogus: got %d, want 2", code)
	}
}

// TestTraceGapsFlag verifies --gaps is accepted and does not cause a crash.
//
//fusa:test REQ-FO-CLI021
func TestTraceGapsFlag(t *testing.T) {
	dir := goProject(t)
	code, stdout, errb := runArgs(t, "trace", "--dir", dir, "--gaps")
	if code != 0 {
		t.Fatalf("trace --gaps: code=%d err=%q", code, errb)
	}
	if !strings.Contains(stdout, "Cross-Language Traceability") {
		t.Errorf("trace --gaps stdout: %q", stdout)
	}
}

// TestTraceReqCoverage verifies --req-coverage gates on traced%.
// With 0% required coverage, any scan passes; with 101% it always fails.
//
//fusa:test REQ-FO-CLI021
func TestTraceReqCoverage(t *testing.T) {
	dir := goProject(t)
	// 0% threshold always passes
	code, _, errb := runArgs(t, "trace", "--dir", dir, "--req-coverage", "0")
	if code != 0 {
		t.Errorf("trace --req-coverage 0: expected 0, got %d (err: %s)", code, errb)
	}
	// 101% threshold always fails (no scanner can hit 101%)
	code, _, _ = runArgs(t, "trace", "--dir", dir, "--req-coverage", "101")
	if code != 1 {
		t.Errorf("trace --req-coverage 101: expected 1, got %d", code)
	}
}

// TestTraceSecTested verifies --sec-tested gates on secTestedPct.
//
//fusa:test REQ-FO-CLI021
func TestTraceSecTested(t *testing.T) {
	dir := goProject(t)
	// 0% always passes; 101% always fails
	if code, _, _ := runArgs(t, "trace", "--dir", dir, "--sec-tested", "0"); code != 0 {
		t.Errorf("trace --sec-tested 0: expected 0, got %d", code)
	}
	if code, _, _ := runArgs(t, "trace", "--dir", dir, "--sec-tested", "101"); code != 1 {
		t.Errorf("trace --sec-tested 101: expected 1, got %d", code)
	}
}

//fusa:test REQ-FO-CLI012
func TestSBOMCommand(t *testing.T) {
	dir := goProject(t)
	code, stdout, errb := runArgs(t, "sbom", "--dir", dir, "--format", "spdx")
	if code != 0 {
		t.Fatalf("sbom: code=%d err=%q", code, errb)
	}
	if !strings.Contains(stdout, "SPDX-2.3") {
		t.Errorf("sbom spdx stdout: %q", stdout)
	}
}

//fusa:test REQ-FO-CLI012
func TestSBOMWritesFile(t *testing.T) {
	dir := goProject(t)
	out := filepath.Join(dir, "sbom.json")
	code, stdout, errb := runArgs(t, "sbom", "--dir", dir, "--format", "json", "--output", out)
	if code != 0 {
		t.Fatalf("sbom: code=%d err=%q", code, errb)
	}
	if !strings.Contains(errb, "Wrote") {
		t.Errorf("sbom stderr: %q", errb)
	}
	if stdout != "" {
		t.Errorf("sbom stdout must be empty when --output given: %q", stdout)
	}
	if _, err := os.Stat(out); err != nil {
		t.Errorf("sbom file not written: %v", err)
	}
}

//fusa:test REQ-FO-CLI012
func TestSBOMBadFlag(t *testing.T) {
	if code, _, _ := runArgs(t, "sbom", "--bogus"); code != 2 {
		t.Errorf("sbom --bogus: got %d, want 2", code)
	}
}

//fusa:test REQ-FO-CLI013
func TestAuditPackCommand(t *testing.T) {
	dir := goProject(t)
	out := filepath.Join(dir, "audit-pack.zip")
	code, stdout, errb := runArgs(t, "audit-pack", "--dir", dir, "--output", out)
	if code != 0 {
		t.Fatalf("audit-pack: code=%d err=%q", code, errb)
	}
	if !strings.Contains(stdout, "Wrote audit-pack") {
		t.Errorf("audit-pack stdout: %q", stdout)
	}
	if _, err := os.Stat(out); err != nil {
		t.Errorf("audit-pack not written: %v", err)
	}
}

//fusa:test REQ-FO-CLI013
func TestAuditPackBadFlag(t *testing.T) {
	if code, _, _ := runArgs(t, "audit-pack", "--bogus"); code != 2 {
		t.Errorf("audit-pack --bogus: got %d, want 2", code)
	}
}
