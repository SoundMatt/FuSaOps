package conform

// Extra tests to push the conform package above the 80% coverage gate.
// They target the main uncovered branches:
//   - execRun (0%)  — direct calls on real system binaries
//   - Run (58%)     — auto-tmpDir and binary-not-found paths
//   - checkCheck    — bad ruleId, nil location, missing fingerprint/remediation
//   - checkTrace    — exit>1, nil requirements/tags/coverage, partial counters
//   - checkRelease  — exit>1, bad sbom JSON, missing module/components
//   - checkAuditPack — exit>1, not a ZIP, no manifest, bad hex, no files

import (
	"archive/zip"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// makeZIPFile writes a valid ZIP archive at path with the supplied entries.
func makeZIPFile(path string, entries map[string][]byte) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	zw := zip.NewWriter(f)
	for name, data := range entries {
		w, werr := zw.Create(name)
		if werr != nil {
			_ = zw.Close()
			_ = f.Close()
			return werr
		}
		if _, werr = w.Write(data); werr != nil {
			_ = zw.Close()
			_ = f.Close()
			return werr
		}
	}
	if err2 := zw.Close(); err2 != nil {
		_ = f.Close()
		return err2
	}
	return f.Close()
}

// findSystemBinary looks for a binary in common POSIX locations.
func findSystemBinary(name string) (string, error) {
	for _, dir := range []string{"/bin", "/usr/bin"} {
		p := filepath.Join(dir, name)
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	return "", os.ErrNotExist
}

// ── execRun ───────────────────────────────────────────────────────────────────

// TestExecRunEcho exercises execRun with the real /bin/echo binary (exit 0).
//
//fusa:test REQ-FO-CNF004
func TestExecRunEcho(t *testing.T) {
	echo, err := findSystemBinary("echo")
	if err != nil {
		t.Skip("no echo binary found:", err)
	}
	stdout, _, code := execRun(t.TempDir(), echo, "hello-conform")
	if code != 0 {
		t.Fatalf("execRun echo: exit %d", code)
	}
	if len(stdout) == 0 {
		t.Error("execRun echo: no stdout")
	}
}

// TestExecRunNonZeroExit exercises the ExitError branch of execRun.
//
//fusa:test REQ-FO-CNF004
func TestExecRunNonZeroExit(t *testing.T) {
	falseBin, err := findSystemBinary("false")
	if err != nil {
		t.Skip("no false binary found:", err)
	}
	_, _, code := execRun(t.TempDir(), falseBin)
	if code == 0 {
		t.Error("execRun false: expected non-zero exit")
	}
}

// ── Run — auto-tmpDir and binary-not-found ────────────────────────────────────

// TestRunAutoTempDir verifies Run creates its own temp dir when TempDir is empty.
//
//fusa:test REQ-FO-CNF005
func TestRunAutoTempDir(t *testing.T) {
	rf := func(dir, binary string, args ...string) ([]byte, []byte, int) {
		if len(args) > 0 && args[0] == "version" {
			if len(args) >= 3 && args[1] == "--format" {
				b, _ := json.Marshal(map[string]string{
					"tool": "auto-FuSa", "version": "0.1.0", "specVersion": "1.8",
				})
				return b, nil, 0
			}
			return []byte("auto-FuSa 0.1.0\n"), nil, 0
		}
		return nil, nil, 0
	}
	// No TempDir → Run creates and removes a temp dir internally.
	rep, err := Run("autoBinary", Options{RunFunc: rf})
	if err != nil {
		t.Fatalf("Run auto-tmpDir: %v", err)
	}
	if rep == nil {
		t.Fatal("Run auto-tmpDir: nil report")
	}
}

// TestRunBinaryNotFound verifies Run returns an error when the binary is absent from PATH.
//
//fusa:test REQ-FO-CNF005
func TestRunBinaryNotFound(t *testing.T) {
	_, err := Run("nonexistent-fusaops-binary-xyz-abc-99", Options{})
	if err == nil {
		t.Fatal("expected error for binary not on PATH")
	}
}

// ── checkCheck — uncovered failure branches ───────────────────────────────────

// TestCheckCheckInvalidJSON verifies check JSON parse failure is recorded.
//
//fusa:test REQ-FO-CNF011
func TestCheckCheckInvalidJSON(t *testing.T) {
	run := func(dir, binary string, args ...string) ([]byte, []byte, int) {
		if len(args) > 0 {
			switch args[0] {
			case "version":
				return []byte("parse-FuSa 0.1.0\n"), nil, 0
			case "check":
				return []byte("not json at all"), nil, 0
			}
		}
		return nil, nil, 0
	}
	rep, _ := Run("parseFailBinary", Options{TempDir: t.TempDir(), RunFunc: run})
	assertFail(t, rep, "check/json-parse")
}

// TestCheckCheckBadRuleID verifies invalid ruleId pattern is caught.
//
//fusa:test REQ-FO-CNF011
func TestCheckCheckBadRuleID(t *testing.T) {
	rep, _ := Run("ruleIDBinary", Options{TempDir: t.TempDir(), RunFunc: checkRunFuncWithFindings(
		"rule-FuSa", []map[string]interface{}{
			{"ruleId": "bad_lowercase_id", "severity": "INFO",
				"message":     "msg",
				"location":    map[string]interface{}{"file": "main.go"},
				"category":    "lint", "remediation": "fix it",
				"fingerprint": "sha256:5529f6e552b61370fcd04fdcf22d73c495c5c9d2746b9293233f0dbb08fe6b27"},
		},
	)})
	assertFail(t, rep, "check/finding-ruleId")
}

// TestCheckCheckEmptyRuleID verifies absent ruleId is caught.
//
//fusa:test REQ-FO-CNF011
func TestCheckCheckEmptyRuleID(t *testing.T) {
	rep, _ := Run("noridBinary", Options{TempDir: t.TempDir(), RunFunc: checkRunFuncWithFindings(
		"norid-FuSa", []map[string]interface{}{
			{"severity": "INFO", "message": "msg",
				"location": map[string]interface{}{"file": "main.go"}},
		},
	)})
	assertFail(t, rep, "check/finding-ruleId")
}

// TestCheckCheckNilFindings verifies absent findings key is caught.
//
//fusa:test REQ-FO-CNF011
func TestCheckCheckNilFindings(t *testing.T) {
	run := func(dir, binary string, args ...string) ([]byte, []byte, int) {
		if len(args) == 0 {
			return nil, nil, 1
		}
		switch args[0] {
		case "version":
			return []byte("nofind-FuSa 0.1.0\n"), nil, 0
		case "check":
			doc := map[string]interface{}{
				"schemaVersion": "1.8", "kind": "check-report",
				"tool": "nofind-FuSa", "toolVersion": "0.1.0",
				"language": "go", "generatedAt": "2026-06-10T00:00:00Z",
				"projectRoot": dir,
				"summary": map[string]int{"total": 0, "errors": 0, "warnings": 0, "infos": 0},
				// findings key absent
			}
			b, _ := json.Marshal(doc)
			return b, nil, 0
		}
		return nil, nil, 0
	}
	rep, _ := Run("nofindBinary", Options{TempDir: t.TempDir(), RunFunc: run})
	assertFail(t, rep, "check/findings-present")
}

// TestCheckCheckNilSummary verifies absent summary is caught.
//
//fusa:test REQ-FO-CNF011
func TestCheckCheckNilSummary(t *testing.T) {
	run := func(dir, binary string, args ...string) ([]byte, []byte, int) {
		if len(args) == 0 {
			return nil, nil, 1
		}
		switch args[0] {
		case "version":
			return []byte("nosum-FuSa 0.1.0\n"), nil, 0
		case "check":
			doc := map[string]interface{}{
				"schemaVersion": "1.8", "kind": "check-report",
				"tool": "nosum-FuSa", "toolVersion": "0.1.0",
				"language": "go", "generatedAt": "2026-06-10T00:00:00Z",
				"projectRoot": dir,
				"findings": []interface{}{},
				// summary absent
			}
			b, _ := json.Marshal(doc)
			return b, nil, 0
		}
		return nil, nil, 0
	}
	rep, _ := Run("nosumBinary", Options{TempDir: t.TempDir(), RunFunc: run})
	assertFail(t, rep, "check/summary")
}

// TestCheckCheckBadSeverity verifies invalid severity is caught.
//
//fusa:test REQ-FO-CNF011
func TestCheckCheckBadSeverity(t *testing.T) {
	rep, _ := Run("badsevBinary", Options{TempDir: t.TempDir(), RunFunc: checkRunFuncWithFindings(
		"badsev-FuSa", []map[string]interface{}{
			{"ruleId": "LINT001", "severity": "CRITICAL",
				"message":     "msg",
				"location":    map[string]interface{}{"file": "main.go"},
				"category":    "lint", "remediation": "fix",
				"fingerprint": "sha256:5529f6e552b61370fcd04fdcf22d73c495c5c9d2746b9293233f0dbb08fe6b27"},
		},
	)})
	assertFail(t, rep, "check/finding-severity")
}

// TestCheckCheckNilLocation verifies absent location object is caught.
//
//fusa:test REQ-FO-CNF011
func TestCheckCheckNilLocation(t *testing.T) {
	rep, _ := Run("nolocBinary", Options{TempDir: t.TempDir(), RunFunc: checkRunFuncWithFindings(
		"noloc-FuSa", []map[string]interface{}{
			{"ruleId": "LINT001", "severity": "INFO",
				"message":     "msg",
				"category":    "lint", "remediation": "fix",
				"fingerprint": "sha256:5529f6e552b61370fcd04fdcf22d73c495c5c9d2746b9293233f0dbb08fe6b27"},
		},
	)})
	assertFail(t, rep, "check/finding-location-nested")
}

// TestCheckCheckEmptyLocationFile verifies empty location.file is caught.
//
//fusa:test REQ-FO-CNF011
func TestCheckCheckEmptyLocationFile(t *testing.T) {
	rep, _ := Run("nofileBinary", Options{TempDir: t.TempDir(), RunFunc: checkRunFuncWithFindings(
		"nofile-FuSa", []map[string]interface{}{
			{"ruleId": "LINT001", "severity": "INFO",
				"message":     "msg",
				"location":    map[string]interface{}{"file": "", "line": 1},
				"category":    "lint", "remediation": "fix",
				"fingerprint": "sha256:5529f6e552b61370fcd04fdcf22d73c495c5c9d2746b9293233f0dbb08fe6b27"},
		},
	)})
	assertFail(t, rep, "check/finding-location-file")
}

// TestCheckCheckMissingFingerprint verifies absent fingerprint is caught.
//
//fusa:test REQ-FO-CNF011
func TestCheckCheckMissingFingerprint(t *testing.T) {
	rep, _ := Run("nofpBinary", Options{TempDir: t.TempDir(), RunFunc: checkRunFuncWithFindings(
		"nofp-FuSa", []map[string]interface{}{
			{"ruleId": "LINT001", "severity": "INFO",
				"message":     "msg",
				"location":    map[string]interface{}{"file": "main.go", "line": 1},
				"category":    "lint", "remediation": "fix it"},
			// no fingerprint field
		},
	)})
	assertFail(t, rep, "check/fingerprint-format")
}

// TestCheckCheckBadFingerprint verifies malformed fingerprint is caught.
//
//fusa:test REQ-FO-CNF011
func TestCheckCheckBadFingerprint(t *testing.T) {
	rep, _ := Run("badfpBinary", Options{TempDir: t.TempDir(), RunFunc: checkRunFuncWithFindings(
		"badfp-FuSa", []map[string]interface{}{
			{"ruleId": "LINT001", "severity": "INFO",
				"message":     "msg",
				"location":    map[string]interface{}{"file": "main.go", "line": 1},
				"category":    "lint", "remediation": "fix it",
				"fingerprint": "md5:abc123"},
		},
	)})
	assertFail(t, rep, "check/fingerprint-format")
}

// TestCheckCheckMissingRemediation verifies absent remediation field is caught.
//
//fusa:test REQ-FO-CNF011
func TestCheckCheckMissingRemediation(t *testing.T) {
	rep, _ := Run("noremBinary", Options{TempDir: t.TempDir(), RunFunc: checkRunFuncWithFindings(
		"norem-FuSa", []map[string]interface{}{
			{"ruleId": "LINT001", "severity": "INFO",
				"message":     "msg",
				"location":    map[string]interface{}{"file": "main.go", "line": 1},
				"category":    "lint",
				"fingerprint": "sha256:5529f6e552b61370fcd04fdcf22d73c495c5c9d2746b9293233f0dbb08fe6b27"},
			// no remediation field
		},
	)})
	assertFail(t, rep, "check/remediation")
}

// checkRunFuncWithFindings returns a RunFunc where "check" returns the supplied findings.
func checkRunFuncWithFindings(tool string, findings []map[string]interface{}) RunFunc {
	return func(dir, binary string, args ...string) ([]byte, []byte, int) {
		if len(args) == 0 {
			return nil, nil, 1
		}
		switch args[0] {
		case "version":
			return []byte(tool + " 0.1.0\n"), nil, 0
		case "check":
			doc := map[string]interface{}{
				"schemaVersion": "1.8", "kind": "check-report",
				"tool": tool, "toolVersion": "0.1.0",
				"language": "go", "generatedAt": "2026-06-10T00:00:00Z",
				"projectRoot": dir,
				"findings":    findings,
				"summary":     map[string]int{"total": len(findings), "errors": 0, "warnings": 0, "infos": len(findings)},
			}
			b, _ := json.Marshal(doc)
			return b, nil, 0
		}
		return nil, nil, 0
	}
}

// ── checkTrace — uncovered failure branches ───────────────────────────────────

// TestCheckTraceRuntimeError verifies exit>1 from trace is skipped gracefully.
//
//fusa:test REQ-FO-CNF012
func TestCheckTraceRuntimeError(t *testing.T) {
	run := func(dir, binary string, args ...string) ([]byte, []byte, int) {
		if len(args) > 0 {
			switch args[0] {
			case "version":
				return []byte("terr-FuSa 0.1.0\n"), nil, 0
			case "trace":
				return nil, nil, 5 // >1 → runtime error skip
			}
		}
		return nil, nil, 0
	}
	rep, _ := Run("terrBinary", Options{TempDir: t.TempDir(), RunFunc: run})
	assertSkip(t, rep, "trace/json")
}

// TestCheckTraceInvalidJSON verifies trace JSON parse failure is caught.
//
//fusa:test REQ-FO-CNF012
func TestCheckTraceInvalidJSON(t *testing.T) {
	run := func(dir, binary string, args ...string) ([]byte, []byte, int) {
		if len(args) > 0 {
			switch args[0] {
			case "version":
				return []byte("tjson-FuSa 0.1.0\n"), nil, 0
			case "trace":
				return []byte("not json"), nil, 0
			}
		}
		return nil, nil, 0
	}
	rep, _ := Run("tjsonBinary", Options{TempDir: t.TempDir(), RunFunc: run})
	assertFail(t, rep, "trace/json-parse")
}

// TestCheckTraceNilRequirements verifies missing requirements key is caught.
//
//fusa:test REQ-FO-CNF012
func TestCheckTraceNilRequirements(t *testing.T) {
	rep, _ := Run("treqsBinary", Options{TempDir: t.TempDir(),
		RunFunc: traceRunFuncWithDoc("treqs-FuSa", map[string]interface{}{
			"tags": []interface{}{},
			"coverage": map[string]int{
				"totalRequirements": 0, "tracedRequirements": 0,
				"testedRequirements": 0, "secTestedRequirements": 0,
			},
			// requirements absent
		})})
	assertFail(t, rep, "trace/requirements-key")
}

// TestCheckTraceNilTags verifies missing tags key is caught.
//
//fusa:test REQ-FO-CNF012
func TestCheckTraceNilTags(t *testing.T) {
	rep, _ := Run("ttagBinary", Options{TempDir: t.TempDir(),
		RunFunc: traceRunFuncWithDoc("ttag-FuSa", map[string]interface{}{
			"requirements": []interface{}{},
			"coverage": map[string]int{
				"totalRequirements": 0, "tracedRequirements": 0,
				"testedRequirements": 0, "secTestedRequirements": 0,
			},
			// tags absent
		})})
	assertFail(t, rep, "trace/tags-key")
}

// TestCheckTraceNilCoverage verifies missing coverage object is caught.
//
//fusa:test REQ-FO-CNF012
func TestCheckTraceNilCoverage(t *testing.T) {
	rep, _ := Run("tcovBinary", Options{TempDir: t.TempDir(),
		RunFunc: traceRunFuncWithDoc("tcov-FuSa", map[string]interface{}{
			"requirements": []interface{}{},
			"tags":         []interface{}{},
			// coverage absent
		})})
	assertFail(t, rep, "trace/coverage-schema")
}

// TestCheckTraceMissingCoverageCounters verifies partial coverage counters are caught.
//
//fusa:test REQ-FO-CNF012
func TestCheckTraceMissingCoverageCounters(t *testing.T) {
	rep, _ := Run("tpartBinary", Options{TempDir: t.TempDir(),
		RunFunc: traceRunFuncWithDoc("tpart-FuSa", map[string]interface{}{
			"requirements": []interface{}{},
			"tags":         []interface{}{},
			"coverage":     map[string]int{"totalRequirements": 0},
			// tracedRequirements, testedRequirements, secTestedRequirements absent
		})})
	assertFail(t, rep, "trace/coverage-schema")
}

// traceRunFuncWithDoc returns a RunFunc where "trace" returns extra merged with the base header.
func traceRunFuncWithDoc(tool string, extra map[string]interface{}) RunFunc {
	return func(dir, binary string, args ...string) ([]byte, []byte, int) {
		if len(args) == 0 {
			return nil, nil, 1
		}
		switch args[0] {
		case "version":
			return []byte(tool + " 0.1.0\n"), nil, 0
		case "trace":
			doc := map[string]interface{}{
				"schemaVersion": "1.8", "kind": "trace-matrix",
				"tool": tool, "toolVersion": "0.1.0",
				"language": "go", "generatedAt": "2026-06-10T00:00:00Z",
				"projectRoot": dir,
			}
			for k, v := range extra {
				doc[k] = v
			}
			b, _ := json.Marshal(doc)
			return b, nil, 0
		}
		return nil, nil, 0
	}
}

// ── checkRelease — uncovered failure branches ─────────────────────────────────

// TestCheckReleaseRuntimeError verifies exit>1 from release is skipped.
//
//fusa:test REQ-FO-CNF014
func TestCheckReleaseRuntimeError(t *testing.T) {
	run := func(dir, binary string, args ...string) ([]byte, []byte, int) {
		if len(args) > 0 {
			switch args[0] {
			case "version":
				return []byte("rerr-FuSa 0.1.0\n"), nil, 0
			case "release":
				return nil, nil, 5
			}
		}
		return nil, nil, 0
	}
	rep, _ := Run("rerrBinary", Options{TempDir: t.TempDir(), RunFunc: run})
	assertSkip(t, rep, "release/sbom-written")
}

// TestCheckReleaseInvalidSBOMJSON verifies non-JSON sbom.json is caught.
//
//fusa:test REQ-FO-CNF014
func TestCheckReleaseInvalidSBOMJSON(t *testing.T) {
	run := func(dir, binary string, args ...string) ([]byte, []byte, int) {
		if len(args) == 0 {
			return nil, nil, 1
		}
		switch args[0] {
		case "version":
			return []byte("rbadj-FuSa 0.1.0\n"), nil, 0
		case "release":
			outDir := dir
			for i, a := range args {
				if a == "--output-dir" && i+1 < len(args) {
					outDir = args[i+1]
				}
			}
			_ = writeTestFile(filepath.Join(outDir, "sbom.json"), []byte("not json"))
			return nil, nil, 0
		}
		return nil, nil, 0
	}
	rep, _ := Run("rbadjBinary", Options{TempDir: t.TempDir(), RunFunc: run})
	assertFail(t, rep, "release/sbom-json")
}

// TestCheckReleaseMissingModule verifies absent module field is caught.
//
//fusa:test REQ-FO-CNF014
func TestCheckReleaseMissingModule(t *testing.T) {
	rep, _ := Run("rmodBinary", Options{TempDir: t.TempDir(),
		RunFunc: releaseRunFuncWithSBOM("rmod-FuSa", map[string]interface{}{
			"format": "x-FuSa SBOM v1",
			// module absent
			"components": []interface{}{},
		})})
	assertFail(t, rep, "release/sbom-module")
}

// TestCheckReleaseNilComponents verifies absent components array is caught.
//
//fusa:test REQ-FO-CNF014
func TestCheckReleaseNilComponents(t *testing.T) {
	rep, _ := Run("rcompBinary", Options{TempDir: t.TempDir(),
		RunFunc: releaseRunFuncWithSBOM("rcomp-FuSa", map[string]interface{}{
			"format": "x-FuSa SBOM v1",
			"module": "github.com/test/test",
			// components absent
		})})
	assertFail(t, rep, "release/sbom-components")
}

// releaseRunFuncWithSBOM returns a RunFunc where "release" writes a sbom.json with extra fields.
func releaseRunFuncWithSBOM(tool string, extra map[string]interface{}) RunFunc {
	return func(dir, binary string, args ...string) ([]byte, []byte, int) {
		if len(args) == 0 {
			return nil, nil, 1
		}
		switch args[0] {
		case "version":
			return []byte(tool + " 0.1.0\n"), nil, 0
		case "release":
			outDir := dir
			for i, a := range args {
				if a == "--output-dir" && i+1 < len(args) {
					outDir = args[i+1]
				}
			}
			doc := map[string]interface{}{
				"schemaVersion": "1.8", "kind": "sbom",
				"tool": tool, "toolVersion": "0.1.0",
				"language": "go", "generatedAt": "2026-06-10T00:00:00Z",
			}
			for k, v := range extra {
				doc[k] = v
			}
			b, _ := json.Marshal(doc)
			_ = writeTestFile(filepath.Join(outDir, "sbom.json"), b)
			return nil, nil, 0
		}
		return nil, nil, 0
	}
}

// ── checkAuditPack — uncovered failure branches ───────────────────────────────

// TestCheckAuditPackRuntimeError verifies exit>1 from audit-pack is skipped.
//
//fusa:test REQ-FO-CNF015
func TestCheckAuditPackRuntimeError(t *testing.T) {
	run := func(dir, binary string, args ...string) ([]byte, []byte, int) {
		if len(args) > 0 {
			switch args[0] {
			case "version":
				return []byte("aperr-FuSa 0.1.0\n"), nil, 0
			case "audit-pack":
				return nil, nil, 5
			}
		}
		return nil, nil, 0
	}
	rep, _ := Run("aperrBinary", Options{TempDir: t.TempDir(), RunFunc: run})
	assertSkip(t, rep, "audit-pack/is-zip")
}

// TestCheckAuditPackNotZIP verifies a non-ZIP output file is caught.
//
//fusa:test REQ-FO-CNF015
func TestCheckAuditPackNotZIP(t *testing.T) {
	run := func(dir, binary string, args ...string) ([]byte, []byte, int) {
		if len(args) == 0 {
			return nil, nil, 1
		}
		switch args[0] {
		case "version":
			return []byte("notzip-FuSa 0.1.0\n"), nil, 0
		case "audit-pack":
			outFile := filepath.Join(dir, "audit-pack.zip")
			for i, a := range args {
				if a == "--output" && i+1 < len(args) {
					outFile = args[i+1]
				}
			}
			_ = writeTestFile(outFile, []byte("this is not a zip file"))
			return nil, nil, 0
		}
		return nil, nil, 0
	}
	rep, _ := Run("notzipBinary", Options{TempDir: t.TempDir(), RunFunc: run})
	assertFail(t, rep, "audit-pack/is-zip")
}

// TestCheckAuditPackNoManifest verifies a ZIP without manifest.json is caught.
//
//fusa:test REQ-FO-CNF015
func TestCheckAuditPackNoManifest(t *testing.T) {
	run := func(dir, binary string, args ...string) ([]byte, []byte, int) {
		if len(args) == 0 {
			return nil, nil, 1
		}
		switch args[0] {
		case "version":
			return []byte("noman-FuSa 0.1.0\n"), nil, 0
		case "audit-pack":
			outFile := filepath.Join(dir, "audit-pack.zip")
			for i, a := range args {
				if a == "--output" && i+1 < len(args) {
					outFile = args[i+1]
				}
			}
			// Valid ZIP but without manifest.json.
			_ = makeZIPFile(outFile, map[string][]byte{"other.json": []byte("{}")})
			return nil, nil, 0
		}
		return nil, nil, 0
	}
	rep, _ := Run("nomanBinary", Options{TempDir: t.TempDir(), RunFunc: run})
	assertFail(t, rep, "audit-pack/manifest-present")
}

// TestCheckAuditPackBadManifestJSON verifies invalid manifest JSON is caught.
//
//fusa:test REQ-FO-CNF015
func TestCheckAuditPackBadManifestJSON(t *testing.T) {
	run := func(dir, binary string, args ...string) ([]byte, []byte, int) {
		if len(args) == 0 {
			return nil, nil, 1
		}
		switch args[0] {
		case "version":
			return []byte("badmj-FuSa 0.1.0\n"), nil, 0
		case "audit-pack":
			outFile := filepath.Join(dir, "audit-pack.zip")
			for i, a := range args {
				if a == "--output" && i+1 < len(args) {
					outFile = args[i+1]
				}
			}
			_ = makeZIPFile(outFile, map[string][]byte{"manifest.json": []byte("not valid json")})
			return nil, nil, 0
		}
		return nil, nil, 0
	}
	rep, _ := Run("badmjBinary", Options{TempDir: t.TempDir(), RunFunc: run})
	assertFail(t, rep, "audit-pack/manifest-json")
}

// TestCheckAuditPackNilFiles verifies missing files array in manifest is caught.
//
//fusa:test REQ-FO-CNF015
func TestCheckAuditPackNilFiles(t *testing.T) {
	manifest := map[string]interface{}{
		"schemaVersion": "1.8", "kind": "audit-manifest",
		"tool": "nofiles-FuSa", "toolVersion": "0.1.0",
		"language": "go", "generatedAt": "2026-06-10T12:00:00Z",
		"module": "github.com/test/test",
		// files absent
	}
	mb, _ := json.Marshal(manifest)
	run := auditPackRunFuncWithManifestBytes("nofiles-FuSa", mb)
	rep, _ := Run("nofilesBinary", Options{TempDir: t.TempDir(), RunFunc: run})
	assertFail(t, rep, "audit-pack/manifest-files")
}

// TestCheckAuditPackBadHex verifies non-hex sha256 value in manifest is caught.
//
//fusa:test REQ-FO-CNF015
func TestCheckAuditPackBadHex(t *testing.T) {
	manifest := map[string]interface{}{
		"schemaVersion": "1.8", "kind": "audit-manifest",
		"tool": "badhex-FuSa", "toolVersion": "0.1.0",
		"language": "go", "generatedAt": "2026-06-10T12:00:00Z",
		"module": "github.com/test/test",
		"files": []map[string]interface{}{
			{"path": "sbom.json", "size": 2, "sha256": "not-valid-hex"},
		},
	}
	mb, _ := json.Marshal(manifest)
	run := auditPackRunFuncWithManifestBytes("badhex-FuSa", mb)
	rep, _ := Run("badhexBinary", Options{TempDir: t.TempDir(), RunFunc: run})
	assertFail(t, rep, "audit-pack/manifest-sha256-bare")
}

// auditPackRunFuncWithManifestBytes returns a RunFunc that writes a ZIP whose
// manifest.json contains the supplied bytes.
func auditPackRunFuncWithManifestBytes(tool string, manifestBytes []byte) RunFunc {
	return func(dir, binary string, args ...string) ([]byte, []byte, int) {
		if len(args) == 0 {
			return nil, nil, 1
		}
		switch args[0] {
		case "version":
			return []byte(tool + " 0.1.0\n"), nil, 0
		case "audit-pack":
			outFile := filepath.Join(dir, "audit-pack.zip")
			for i, a := range args {
				if a == "--output" && i+1 < len(args) {
					outFile = args[i+1]
				}
			}
			_ = makeZIPFile(outFile, map[string][]byte{"manifest.json": manifestBytes})
			return nil, nil, 0
		}
		return nil, nil, 0
	}
}

// ── assertion helpers ─────────────────────────────────────────────────────────

func assertFail(t *testing.T, rep *Report, id string) {
	t.Helper()
	for _, r := range rep.Results {
		if r.ID == id && r.Status == StatusFail {
			return
		}
	}
	t.Errorf("expected result %q with FAIL status; got: %v", id, summariseResults(rep))
}

func assertSkip(t *testing.T, rep *Report, id string) {
	t.Helper()
	for _, r := range rep.Results {
		if r.ID == id && r.Status == StatusSkip {
			return
		}
	}
	t.Errorf("expected result %q with SKIP status; got: %v", id, summariseResults(rep))
}

func summariseResults(rep *Report) []string {
	var out []string
	for _, r := range rep.Results {
		out = append(out, r.ID+"="+string(r.Status))
	}
	return out
}
