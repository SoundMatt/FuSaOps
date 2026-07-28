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
	if !strings.Contains(out, "iso21434") {
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

// ── fusaops sci ──────────────────────────────────────────────────────────────

//fusa:test REQ-FO-CLI067
func TestSCIBadFlag(t *testing.T) {
	code, _, errb := runArgs(t, "sci", "--bogus")
	if code != 2 {
		t.Errorf("sci --bogus: code=%d err=%q", code, errb)
	}
}

//fusa:test REQ-FO-CLI067
func TestSCIEmptyDir(t *testing.T) {
	// Empty dir → no adapters, all artefacts missing → exit 0 (SCI always succeeds).
	dir := t.TempDir()
	code, out, errb := runArgs(t, "sci", "--dir", dir)
	if code != 0 {
		t.Fatalf("sci empty dir: code=%d err=%q", code, errb)
	}
	if !strings.Contains(out, "Software Configuration Index") {
		t.Errorf("sci output missing header: %q", out[:min(len(out), 200)])
	}
	if !strings.Contains(out, "SCI written to") {
		t.Errorf("sci output missing confirmation: %q", out)
	}
}

//fusa:test REQ-FO-CLI067
func TestSCIJSON(t *testing.T) {
	dir := t.TempDir()
	code, out, errb := runArgs(t, "sci", "--dir", dir, "--format", "json")
	if code != 0 {
		t.Fatalf("sci json: code=%d err=%q", code, errb)
	}
	if !strings.Contains(out, `"tool"`) || !strings.Contains(out, `"items"`) {
		t.Errorf("sci json missing expected fields: %q", out[:min(len(out), 200)])
	}
}

//fusa:test REQ-FO-CLI067
func TestSCIOutputFile(t *testing.T) {
	dir := t.TempDir()
	outFile := filepath.Join(dir, "sci.json")
	code, _, errb := runArgs(t, "sci", "--dir", dir, "--output", outFile)
	if code != 0 {
		t.Fatalf("sci --output: code=%d err=%q", code, errb)
	}
	if _, err := os.Stat(outFile); err != nil {
		t.Errorf("output file not created: %v", err)
	}
}

//fusa:test REQ-FO-CLI067
func TestSCIGoProject(t *testing.T) {
	// A project with Go source detects the go adapter.
	dir := goProject(t)
	code, out, _ := runArgs(t, "sci", "--dir", dir)
	if code != 0 {
		t.Errorf("sci goProject: code=%d", code)
	}
	if !strings.Contains(out, "Tool Items") {
		t.Errorf("sci output missing Tool Items section: %q", out[:min(len(out), 200)])
	}
}

//fusa:test REQ-FO-CLI067
func TestSCIInvalidFormat(t *testing.T) {
	dir := t.TempDir()
	code, _, errb := runArgs(t, "sci", "--dir", dir, "--format", "xml")
	if code != 2 || !strings.Contains(errb, "format") {
		t.Errorf("sci invalid format: code=%d err=%q", code, errb)
	}
}

// ── fusaops sas ──────────────────────────────────────────────────────────────

//fusa:test REQ-FO-CLI068
func TestSASBadFlag(t *testing.T) {
	code, _, errb := runArgs(t, "sas", "--bogus")
	if code != 2 {
		t.Errorf("sas --bogus: code=%d err=%q", code, errb)
	}
}

//fusa:test REQ-FO-CLI068
func TestSASEmptyDir(t *testing.T) {
	// Empty dir → some activities incomplete → exit 1 with report.
	dir := t.TempDir()
	code, out, errb := runArgs(t, "sas", "--dir", dir)
	if code != 1 {
		t.Fatalf("sas empty dir: code=%d err=%q", code, errb)
	}
	if !strings.Contains(out, "Software Accomplishment Summary") {
		t.Errorf("sas output missing header: %q", out[:min(len(out), 200)])
	}
	if !strings.Contains(out, "INCOMPLETE") {
		t.Errorf("sas output missing incomplete warning: %q", out)
	}
}

//fusa:test REQ-FO-CLI068
func TestSASJSON(t *testing.T) {
	dir := t.TempDir()
	code, out, _ := runArgs(t, "sas", "--dir", dir, "--format", "json")
	if code > 1 {
		t.Fatalf("sas json: unexpected exit code %d", code)
	}
	if !strings.Contains(out, `"activities"`) || !strings.Contains(out, `"softwareLevel"`) {
		t.Errorf("sas json missing expected fields: %q", out[:min(len(out), 200)])
	}
}

//fusa:test REQ-FO-CLI068
func TestSASOutputFile(t *testing.T) {
	dir := t.TempDir()
	outFile := filepath.Join(dir, "sas.json")
	runArgs(t, "sas", "--dir", dir, "--output", outFile)
	if _, err := os.Stat(outFile); err != nil {
		t.Errorf("output file not created: %v", err)
	}
}

//fusa:test REQ-FO-CLI068
func TestSASLevel(t *testing.T) {
	dir := t.TempDir()
	code, out, _ := runArgs(t, "sas", "--dir", dir, "--level", "DAL-A")
	if code > 1 {
		t.Fatalf("sas --level DAL-A: unexpected exit code %d", code)
	}
	if !strings.Contains(out, "DAL-A") {
		t.Errorf("sas --level output missing level: %q", out[:min(len(out), 200)])
	}
}

//fusa:test REQ-FO-CLI068
func TestSASInvalidFormat(t *testing.T) {
	dir := t.TempDir()
	code, _, errb := runArgs(t, "sas", "--dir", dir, "--format", "xml")
	if code != 2 || !strings.Contains(errb, "format") {
		t.Errorf("sas invalid format: code=%d err=%q", code, errb)
	}
}

// ── fusaops tara ─────────────────────────────────────────────────────────────

//fusa:test REQ-FO-CLI069
func TestTARABadFlag(t *testing.T) {
	code, _, errb := runArgs(t, "tara", "--bogus")
	if code != 2 {
		t.Errorf("tara --bogus: code=%d err=%q", code, errb)
	}
}

//fusa:test REQ-FO-CLI069
func TestTARASuccess(t *testing.T) {
	dir := t.TempDir()
	// tara always produces output; exits 1 when any critical scenario exists.
	code, out, errb := runArgs(t, "tara", "--dir", dir)
	if code > 1 {
		t.Fatalf("tara: unexpected exit code %d, err=%q", code, errb)
	}
	if !strings.Contains(out, "TARA") {
		t.Errorf("tara output missing header: %q", out[:min(len(out), 200)])
	}
	if !strings.Contains(out, "TARA written to") {
		t.Errorf("tara output missing confirmation: %q", out)
	}
}

//fusa:test REQ-FO-CLI069
func TestTARAJSON(t *testing.T) {
	dir := t.TempDir()
	code, out, _ := runArgs(t, "tara", "--dir", dir, "--format", "json")
	if code > 1 {
		t.Fatalf("tara json: unexpected exit code %d", code)
	}
	if !strings.Contains(out, `"threats"`) || !strings.Contains(out, `"risk"`) {
		t.Errorf("tara json missing expected fields: %q", out[:min(len(out), 200)])
	}
}

//fusa:test REQ-FO-CLI069
func TestTARAOutputFile(t *testing.T) {
	dir := t.TempDir()
	outFile := filepath.Join(dir, "tara.json")
	runArgs(t, "tara", "--dir", dir, "--output", outFile)
	if _, err := os.Stat(outFile); err != nil {
		t.Errorf("output file not created: %v", err)
	}
}

//fusa:test REQ-FO-CLI069
func TestTARAInvalidFormat(t *testing.T) {
	dir := t.TempDir()
	code, _, errb := runArgs(t, "tara", "--dir", dir, "--format", "xml")
	if code != 2 || !strings.Contains(errb, "format") {
		t.Errorf("tara invalid format: code=%d err=%q", code, errb)
	}
}

// ── fusaops fmea ─────────────────────────────────────────────────────────────

//fusa:test REQ-FO-CLI070
func TestFMEABadFlag(t *testing.T) {
	code, _, errb := runArgs(t, "fmea", "--bogus")
	if code != 2 {
		t.Errorf("fmea --bogus: code=%d err=%q", code, errb)
	}
}

//fusa:test REQ-FO-CLI070
func TestFMEASuccess(t *testing.T) {
	dir := t.TempDir()
	// fmea always produces output; exits 1 when high-RPN items exist.
	code, out, errb := runArgs(t, "fmea", "--dir", dir)
	if code > 1 {
		t.Fatalf("fmea: unexpected exit code %d, err=%q", code, errb)
	}
	if !strings.Contains(out, "dFMEA") {
		t.Errorf("fmea output missing header: %q", out[:min(len(out), 200)])
	}
	if !strings.Contains(out, "FMEA written to") {
		t.Errorf("fmea output missing confirmation: %q", out)
	}
}

//fusa:test REQ-FO-CLI070
func TestFMEAJSON(t *testing.T) {
	dir := t.TempDir()
	code, out, _ := runArgs(t, "fmea", "--dir", dir, "--format", "json")
	if code > 1 {
		t.Fatalf("fmea json: unexpected exit code %d", code)
	}
	if !strings.Contains(out, `"entries"`) || !strings.Contains(out, `"rpn"`) {
		t.Errorf("fmea json missing expected fields: %q", out[:min(len(out), 200)])
	}
}

//fusa:test REQ-FO-CLI070
func TestFMEAOutputFile(t *testing.T) {
	dir := t.TempDir()
	outFile := filepath.Join(dir, "fmea.json")
	runArgs(t, "fmea", "--dir", dir, "--output", outFile)
	if _, err := os.Stat(outFile); err != nil {
		t.Errorf("output file not created: %v", err)
	}
}

//fusa:test REQ-FO-CLI070
func TestFMEAInvalidFormat(t *testing.T) {
	dir := t.TempDir()
	code, _, errb := runArgs(t, "fmea", "--dir", dir, "--format", "xml")
	if code != 2 || !strings.Contains(errb, "format") {
		t.Errorf("fmea invalid format: code=%d err=%q", code, errb)
	}
}

// ── fusaops vuln ─────────────────────────────────────────────────────────────

//fusa:test REQ-FO-CLI071
func TestVulnBadFlag(t *testing.T) {
	code, _, errb := runArgs(t, "vuln", "--bogus")
	if code != 2 {
		t.Errorf("vuln --bogus: code=%d err=%q", code, errb)
	}
}

//fusa:test REQ-FO-CLI071
func TestVulnEmptyDir(t *testing.T) {
	// Empty dir, osv-scanner likely not on PATH in CI → exit 0.
	dir := t.TempDir()
	code, out, errb := runArgs(t, "vuln", "--dir", dir)
	if code > 1 {
		t.Fatalf("vuln empty dir: unexpected exit code %d, err=%q", code, errb)
	}
	if !strings.Contains(out, "Vulnerability Scan") {
		t.Errorf("vuln output missing header: %q", out[:min(len(out), 200)])
	}
	if !strings.Contains(out, "Vulnerability report written to") {
		t.Errorf("vuln output missing confirmation: %q", out)
	}
}

//fusa:test REQ-FO-CLI071
func TestVulnJSON(t *testing.T) {
	dir := t.TempDir()
	code, out, _ := runArgs(t, "vuln", "--dir", dir, "--format", "json")
	if code > 1 {
		t.Fatalf("vuln json: unexpected exit code %d", code)
	}
	if !strings.Contains(out, `"manifests"`) || !strings.Contains(out, `"scannerTool"`) {
		t.Errorf("vuln json missing expected fields: %q", out[:min(len(out), 200)])
	}
}

//fusa:test REQ-FO-CLI071
func TestVulnOutputFile(t *testing.T) {
	dir := t.TempDir()
	outFile := filepath.Join(dir, "vuln.json")
	runArgs(t, "vuln", "--dir", dir, "--output", outFile)
	if _, err := os.Stat(outFile); err != nil {
		t.Errorf("output file not created: %v", err)
	}
}

//fusa:test REQ-FO-CLI071
func TestVulnInvalidFormat(t *testing.T) {
	dir := t.TempDir()
	code, _, errb := runArgs(t, "vuln", "--dir", dir, "--format", "xml")
	if code != 2 || !strings.Contains(errb, "format") {
		t.Errorf("vuln invalid format: code=%d err=%q", code, errb)
	}
}

// ── fusaops template ──────────────────────────────────────────────────────────

//fusa:test REQ-FO-CLI072
func TestTemplateBadFlag(t *testing.T) {
	code, _, errb := runArgs(t, "template", "--bogus")
	if code != 2 {
		t.Errorf("template --bogus: code=%d err=%q", code, errb)
	}
}

//fusa:test REQ-FO-CLI072
func TestTemplateSuccess(t *testing.T) {
	dir := t.TempDir()
	code, out, errb := runArgs(t, "template", "--dir", dir)
	if code != 0 {
		t.Fatalf("template: code=%d err=%q", code, errb)
	}
	if !strings.Contains(out, "Safety Documentation") {
		t.Errorf("template output missing header: %q", out[:min(len(out), 200)])
	}
	if !strings.Contains(out, "Templates written to") {
		t.Errorf("template output missing confirmation: %q", out)
	}
}

//fusa:test REQ-FO-CLI072
func TestTemplateStandardFilter(t *testing.T) {
	dir := t.TempDir()
	code, out, errb := runArgs(t, "template", "--dir", dir, "--standards", "DO-178C")
	if code != 0 {
		t.Fatalf("template DO-178C: code=%d err=%q", code, errb)
	}
	if !strings.Contains(out, "SP-") {
		t.Errorf("template output missing template IDs: %q", out[:min(len(out), 200)])
	}
}

//fusa:test REQ-FO-CLI072
func TestTemplateJSON(t *testing.T) {
	dir := t.TempDir()
	code, out, _ := runArgs(t, "template", "--dir", dir, "--format", "json")
	if code != 0 {
		t.Fatalf("template json: unexpected exit code %d", code)
	}
	if !strings.Contains(out, `"generated"`) || !strings.Contains(out, `"totalGenerated"`) {
		t.Errorf("template json missing expected fields: %q", out[:min(len(out), 200)])
	}
}

//fusa:test REQ-FO-CLI072
func TestTemplateOutputDir(t *testing.T) {
	dir := t.TempDir()
	outDir := filepath.Join(dir, "my-docs")
	code, _, errb := runArgs(t, "template", "--dir", dir, "--output-dir", outDir)
	if code != 0 {
		t.Fatalf("template --output-dir: code=%d err=%q", code, errb)
	}
	if _, err := os.Stat(outDir); err != nil {
		t.Errorf("output dir not created: %v", err)
	}
}

//fusa:test REQ-FO-CLI072
func TestTemplateInvalidFormat(t *testing.T) {
	dir := t.TempDir()
	code, _, errb := runArgs(t, "template", "--dir", dir, "--format", "xml")
	if code != 2 || !strings.Contains(errb, "format") {
		t.Errorf("template invalid format: code=%d err=%q", code, errb)
	}
}

// ── fusaops hara ──────────────────────────────────────────────────────────────

//fusa:test REQ-FO-CLI073
func TestHaraBadFlag(t *testing.T) {
	code, _, errb := runArgs(t, "hara", "--bogus")
	if code != 2 {
		t.Errorf("hara --bogus: code=%d err=%q", code, errb)
	}
}

//fusa:test REQ-FO-CLI073
func TestHaraShowEmpty(t *testing.T) {
	dir := t.TempDir()
	code, out, errb := runArgs(t, "hara", "--dir", dir, "show")
	if code != 0 {
		t.Fatalf("hara show empty: code=%d err=%q", code, errb)
	}
	if !strings.Contains(out, "Hazard Analysis") {
		t.Errorf("hara show output missing header: %q", out[:min(len(out), 200)])
	}
}

//fusa:test REQ-FO-CLI073
func TestHaraInit(t *testing.T) {
	dir := t.TempDir()
	code, out, errb := runArgs(t, "hara", "--dir", dir, "init", "--project", "TestProj")
	if code != 0 {
		t.Fatalf("hara init: code=%d err=%q", code, errb)
	}
	if !strings.Contains(out, ".fusa-hara.json") {
		t.Errorf("hara init output missing filename: %q", out)
	}
	// reinit should fail (file exists)
	code2, _, _ := runArgs(t, "hara", "--dir", dir, "init")
	if code2 != 2 {
		t.Errorf("hara init on existing file: code=%d, want 2", code2)
	}
}

//fusa:test REQ-FO-CLI073
func TestHaraASIL(t *testing.T) {
	code, out, errb := runArgs(t, "hara", "asil", "-s", "S2", "-e", "E3", "-c", "C2")
	if code != 0 {
		t.Fatalf("hara asil: code=%d err=%q", code, errb)
	}
	if !strings.Contains(out, "ASIL-B") {
		t.Errorf("hara asil output: %q, want ASIL-B", out)
	}
}

//fusa:test REQ-FO-CLI073
func TestHaraASILMissingFlags(t *testing.T) {
	code, _, errb := runArgs(t, "hara", "asil", "-s", "S2")
	if code != 2 || !strings.Contains(errb, "required") {
		t.Errorf("hara asil missing flags: code=%d err=%q", code, errb)
	}
}

//fusa:test REQ-FO-CLI073
func TestHaraUnknownSubcommand(t *testing.T) {
	code, _, errb := runArgs(t, "hara", "bogus")
	if code != 2 || !strings.Contains(errb, "unknown subcommand") {
		t.Errorf("hara unknown subcommand: code=%d err=%q", code, errb)
	}
}

//fusa:test REQ-FO-CLI073
func TestHaraShowJSON(t *testing.T) {
	dir := t.TempDir()
	if code, _, errb := runArgs(t, "hara", "--dir", dir, "init"); code != 0 {
		t.Fatalf("hara init: code=%d err=%q", code, errb)
	}
	code, out, errb := runArgs(t, "hara", "--dir", dir, "show", "--format", "json")
	if code != 0 {
		t.Fatalf("hara show json: code=%d err=%q", code, errb)
	}
	if !strings.Contains(out, `"hazards"`) {
		t.Errorf("hara show json missing fields: %q", out[:min(len(out), 200)])
	}
}
