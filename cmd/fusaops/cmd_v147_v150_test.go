package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Tests for the v1.47–v1.51 commands: verify, sign, qualify, release, safety-case.
// Exercises CLI flag parsing, error paths, and the happy path where possible
// without requiring the x-FuSa tools to be installed.

// ── fusaops verify ──────────────────────────────────────────────────────────

//fusa:test REQ-FO-CLI062
func TestVerifyBadFlag(t *testing.T) {
	code, _, errb := runArgs(t, "verify", "--bogus")
	if code != 2 {
		t.Errorf("verify --bogus: code=%d err=%q", code, errb)
	}
}

//fusa:test REQ-FO-CLI062
func TestVerifyNoGoModule(t *testing.T) {
	// Empty dir: go test exits non-zero but that is treated as "test failures
	// to parse", yielding 0 results. runVerify exits 0 with a summary of 0 tests.
	dir := t.TempDir()
	code, out, errb := runArgs(t, "verify", "--dir", dir)
	if code != 0 {
		t.Errorf("verify empty dir: code=%d err=%q", code, errb)
	}
	if !strings.Contains(out, "Evidence bundle written") {
		t.Errorf("verify empty dir: output missing bundle confirmation: %q", out)
	}
}

//fusa:test REQ-FO-CLI062
func TestVerifyOutputFlag(t *testing.T) {
	// Even in a failing dir, --output flag must be parsed without a 2 exit.
	dir := t.TempDir()
	out := filepath.Join(dir, "evidence.json")
	code, _, _ := runArgs(t, "verify", "--dir", dir, "--output", out)
	// go test will fail in an empty dir → exit 1 (not 2)
	if code == 2 {
		t.Errorf("verify --output: unexpected flag parse error, code=%d", code)
	}
}

// ── fusaops sign ─────────────────────────────────────────────────────────────

//fusa:test REQ-FO-CLI063
func TestSignBadFlag(t *testing.T) {
	code, _, errb := runArgs(t, "sign", "--bogus")
	if code != 2 {
		t.Errorf("sign --bogus: code=%d err=%q", code, errb)
	}
}

//fusa:test REQ-FO-CLI063
func TestSignNoArgs(t *testing.T) {
	// No --keygen and no file argument → exit 2 (usage).
	code, _, _ := runArgs(t, "sign", "--key", "keyfile")
	if code != 2 {
		t.Errorf("sign no args: code=%d, want 2", code)
	}
}

//fusa:test REQ-FO-CLI063
func TestSignMissingKey(t *testing.T) {
	// File exists but key file does not → LoadKey fails → exit 1.
	dir := t.TempDir()
	target := filepath.Join(dir, "artifact.bin")
	if err := os.WriteFile(target, []byte("data"), 0o640); err != nil {
		t.Fatal(err)
	}
	code, _, errb := runArgs(t, "sign", "--key", filepath.Join(dir, "nonexistent.key"), target)
	if code != 1 || !strings.Contains(errb, "sign") {
		t.Errorf("sign missing key: code=%d err=%q", code, errb)
	}
}

//fusa:test REQ-FO-CLI063
func TestSignKeygen(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "hmac.key")
	code, out, errb := runArgs(t, "sign", "--keygen", keyPath)
	if code != 0 {
		t.Fatalf("sign --keygen: code=%d err=%q", code, errb)
	}
	if !strings.Contains(out, keyPath) {
		t.Errorf("keygen output missing path: %q", out)
	}
	if _, err := os.Stat(keyPath); err != nil {
		t.Errorf("key file not created: %v", err)
	}
}

//fusa:test REQ-FO-CLI063
func TestSignFile(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "hmac.key")
	// Generate a key first.
	if code, _, errb := runArgs(t, "sign", "--keygen", keyPath); code != 0 {
		t.Fatalf("keygen: code=%d err=%q", code, errb)
	}
	target := filepath.Join(dir, "artifact.bin")
	if err := os.WriteFile(target, []byte("payload"), 0o640); err != nil {
		t.Fatal(err)
	}
	code, out, errb := runArgs(t, "sign", "--key", keyPath, target)
	if code != 0 {
		t.Fatalf("sign file: code=%d err=%q", code, errb)
	}
	if !strings.Contains(out, "Signature written") {
		t.Errorf("sign output missing confirmation: %q", out)
	}
	if _, err := os.Stat(target + ".sig"); err != nil {
		t.Errorf("sig file not created: %v", err)
	}
}

//fusa:test REQ-FO-CLI063
func TestSignVerifyFile(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "hmac.key")
	if code, _, errb := runArgs(t, "sign", "--keygen", keyPath); code != 0 {
		t.Fatalf("keygen: code=%d err=%q", code, errb)
	}
	target := filepath.Join(dir, "artifact.bin")
	if err := os.WriteFile(target, []byte("payload"), 0o640); err != nil {
		t.Fatal(err)
	}
	// Sign it.
	if code, _, errb := runArgs(t, "sign", "--key", keyPath, target); code != 0 {
		t.Fatalf("sign: code=%d err=%q", code, errb)
	}
	// Verify it.
	code, out, errb := runArgs(t, "sign", "--verify", "--key", keyPath, target)
	if code != 0 {
		t.Fatalf("sign --verify: code=%d err=%q", code, errb)
	}
	if !strings.Contains(out, "Signature OK") {
		t.Errorf("verify output missing confirmation: %q", out)
	}
}

//fusa:test REQ-FO-CLI063
func TestSignVerifyMissingSig(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "hmac.key")
	if code, _, errb := runArgs(t, "sign", "--keygen", keyPath); code != 0 {
		t.Fatalf("keygen: code=%d err=%q", code, errb)
	}
	target := filepath.Join(dir, "artifact.bin")
	if err := os.WriteFile(target, []byte("payload"), 0o640); err != nil {
		t.Fatal(err)
	}
	// No .sig file exists → verify must fail.
	code, _, errb := runArgs(t, "sign", "--verify", "--key", keyPath, target)
	if code != 1 || !strings.Contains(errb, "sign") {
		t.Errorf("sign verify missing sig: code=%d err=%q", code, errb)
	}
}

//fusa:test REQ-FO-CLI063
func TestSignMissingKeyFlag(t *testing.T) {
	// Signing a file without --key → exit 2.
	dir := t.TempDir()
	target := filepath.Join(dir, "artifact.bin")
	if err := os.WriteFile(target, []byte("data"), 0o640); err != nil {
		t.Fatal(err)
	}
	code, _, errb := runArgs(t, "sign", target)
	if code != 2 || !strings.Contains(errb, "--key") {
		t.Errorf("sign no --key: code=%d err=%q", code, errb)
	}
}

// ── fusaops qualify ──────────────────────────────────────────────────────────

//fusa:test REQ-FO-CLI064
func TestQualifyBadFlag(t *testing.T) {
	code, _, errb := runArgs(t, "qualify", "--bogus")
	if code != 2 {
		t.Errorf("qualify --bogus: code=%d err=%q", code, errb)
	}
}

//fusa:test REQ-FO-CLI064
func TestQualifyNoAdapters(t *testing.T) {
	// Empty temp dir → no applicable adapters → exit 1.
	code, _, errb := runArgs(t, "qualify", "--dir", t.TempDir())
	if code != 1 || !strings.Contains(errb, "no applicable adapters") {
		t.Errorf("qualify no adapters: code=%d err=%q", code, errb)
	}
}

//fusa:test REQ-FO-CLI064
func TestQualifyFormatFlagParsed(t *testing.T) {
	// Ensure --format is parsed before adapter check (parse returns 2, not 1).
	code, _, errb := runArgs(t, "qualify", "--bogus-flag")
	if code == 1 {
		t.Errorf("qualify flag parse should exit 2, not 1: err=%q", errb)
	}
}

//fusa:test REQ-FO-CLI064
func TestQualifyWithGoProject(t *testing.T) {
	// A directory with a .go file causes the Go adapter to be applicable.
	// qualify.Run will attempt to shell out to gofusa qualify; if gofusa is not
	// installed the component is recorded as skipped — either way exit must be 0
	// (no failures) or 1 (failures), never 2.
	dir := goProject(t)
	code, out, _ := runArgs(t, "qualify", "--dir", dir)
	if code == 2 {
		t.Errorf("qualify goProject: unexpected parse error, code=2 out=%q", out)
	}
	// Output must contain "adapter" or "qualification" regardless of outcome.
	combined := out
	if !strings.Contains(combined, "adapter") && !strings.Contains(combined, "qualification") &&
		!strings.Contains(combined, "Qualification") {
		t.Errorf("qualify goProject: unexpected output: %q", combined)
	}
}

// ── fusaops release ──────────────────────────────────────────────────────────

//fusa:test REQ-FO-CLI065
func TestReleaseBadFlag(t *testing.T) {
	code, _, errb := runArgs(t, "release", "--bogus")
	if code != 2 {
		t.Errorf("release --bogus: code=%d err=%q", code, errb)
	}
}

//fusa:test REQ-FO-CLI065
func TestReleaseEmptyDir(t *testing.T) {
	// No adapters detected → SBOM skipped; provenance + manifest still written.
	dir := t.TempDir()
	code, out, errb := runArgs(t, "release", "--dir", dir)
	if code != 0 {
		t.Fatalf("release empty dir: code=%d err=%q", code, errb)
	}
	if !strings.Contains(out, "Provenance written") {
		t.Errorf("release output missing provenance confirmation: %q", out)
	}
	if !strings.Contains(out, "Artifact manifest written") {
		t.Errorf("release output missing manifest confirmation: %q", out)
	}
}

//fusa:test REQ-FO-CLI065
func TestReleaseOutputDir(t *testing.T) {
	dir := t.TempDir()
	outDir := filepath.Join(dir, "dist")
	code, out, errb := runArgs(t, "release", "--dir", dir, "--output-dir", outDir)
	if code != 0 {
		t.Fatalf("release --output-dir: code=%d err=%q", code, errb)
	}
	if !strings.Contains(out, outDir) {
		t.Errorf("release output missing output dir path: %q", out)
	}
	if _, err := os.Stat(filepath.Join(outDir, "provenance.json")); err != nil {
		t.Errorf("provenance.json not created in output dir: %v", err)
	}
	if _, err := os.Stat(filepath.Join(outDir, "artifact-manifest.json")); err != nil {
		t.Errorf("artifact-manifest.json not created in output dir: %v", err)
	}
}

//fusa:test REQ-FO-CLI065
func TestReleaseGoProject(t *testing.T) {
	// A project with Go source triggers SBOM generation attempt.
	// gofusa may or may not be installed — either way release must not exit 2.
	dir := goProject(t)
	code, _, _ := runArgs(t, "release", "--dir", dir)
	if code == 2 {
		t.Errorf("release goProject: unexpected parse error, code=2")
	}
}

// ── fusaops safety-case ──────────────────────────────────────────────────────

//fusa:test REQ-FO-CLI066
func TestSafetyCaseBadFlag(t *testing.T) {
	code, _, errb := runArgs(t, "safety-case", "--bogus")
	if code != 2 {
		t.Errorf("safety-case --bogus: code=%d err=%q", code, errb)
	}
}

//fusa:test REQ-FO-CLI066
func TestSafetyCaseBadStandard(t *testing.T) {
	code, _, errb := runArgs(t, "safety-case", "--standard", "BOGUS-9999")
	if code != 2 || !strings.Contains(errb, "standard") {
		t.Errorf("safety-case bad standard: code=%d err=%q", code, errb)
	}
}

//fusa:test REQ-FO-CLI066
func TestSafetyCaseEmptyDir(t *testing.T) {
	// Empty dir → gaps in all claims → exit 1 with report.
	dir := t.TempDir()
	code, out, errb := runArgs(t, "safety-case", "--dir", dir)
	if code != 1 {
		t.Fatalf("safety-case empty dir: code=%d err=%q", code, errb)
	}
	if !strings.Contains(out, "FuSaOps Safety Case") {
		t.Errorf("safety-case output missing header: %q", out[:min(len(out), 200)])
	}
	if !strings.Contains(out, "GAPS DETECTED") {
		t.Errorf("safety-case output missing gap warning: %q", out)
	}
}

//fusa:test REQ-FO-CLI066
func TestSafetyCaseJSON(t *testing.T) {
	dir := t.TempDir()
	code, out, _ := runArgs(t, "safety-case", "--dir", dir, "--format", "json")
	if code > 1 {
		t.Fatalf("safety-case json: unexpected exit code %d", code)
	}
	if !strings.Contains(out, `"standard"`) || !strings.Contains(out, `"claims"`) {
		t.Errorf("safety-case json missing expected fields: %q", out[:min(len(out), 200)])
	}
}

//fusa:test REQ-FO-CLI066
func TestSafetyCaseOutputFile(t *testing.T) {
	dir := t.TempDir()
	outFile := filepath.Join(dir, "safety-case.json")
	runArgs(t, "safety-case", "--dir", dir, "--output", outFile)
	if _, err := os.Stat(outFile); err != nil {
		t.Errorf("output file not created: %v", err)
	}
}

//fusa:test REQ-FO-CLI066
func TestSafetyCaseISO21434(t *testing.T) {
	dir := t.TempDir()
	code, out, _ := runArgs(t, "safety-case", "--dir", dir, "--standard", "ISO 21434")
	if code > 1 {
		t.Fatalf("safety-case ISO 21434: unexpected exit code %d", code)
	}
	if !strings.Contains(out, "ISO 21434") {
		t.Errorf("safety-case output missing standard name: %q", out[:min(len(out), 200)])
	}
}

//fusa:test REQ-FO-CLI066
func TestSafetyCaseInvalidFormat(t *testing.T) {
	dir := t.TempDir()
	code, _, errb := runArgs(t, "safety-case", "--dir", dir, "--format", "xml")
	if code != 2 || !strings.Contains(errb, "format") {
		t.Errorf("safety-case invalid format: code=%d err=%q", code, errb)
	}
}
