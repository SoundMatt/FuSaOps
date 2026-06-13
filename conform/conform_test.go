package conform

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeTestFile creates path with data (helper for fake tools).
func writeTestFile(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// writeTestZIP creates a minimal conforming audit-pack ZIP.
func writeTestZIP(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	zw := zip.NewWriter(f)
	defer zw.Close()
	manifest := map[string]interface{}{
		"schemaVersion": "1.8", "kind": "audit-manifest",
		"tool": "test-FuSa", "toolVersion": "0.1.0",
		"language": "go", "generatedAt": "2026-06-10T12:00:00Z",
		"module": "github.com/test/test",
		"files": []map[string]interface{}{
			{"path": "sbom.json", "size": 2,
				"sha256": "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2"},
		},
	}
	mb, _ := json.Marshal(manifest)
	type zipEntry struct {
		name string
		data []byte
	}
	for _, entry := range []zipEntry{
		{"manifest.json", mb},
		{"sbom.json", []byte("{}")},
	} {
		w, werr := zw.Create(entry.name)
		if werr != nil {
			return werr
		}
		if _, werr = w.Write(entry.data); werr != nil {
			return werr
		}
	}
	return nil
}

// newConformingRunFunc returns a RunFunc that passes every MUST check.
//
//fusa:test REQ-FO-CNF005
func newConformingRunFunc() RunFunc {
	return func(dir, binary string, args ...string) ([]byte, []byte, int) {
		if len(args) == 0 {
			return nil, nil, 1
		}
		switch args[0] {
		case "version":
			if len(args) >= 3 && args[1] == "--format" && args[2] == "json" {
				b, _ := json.Marshal(map[string]string{
					"tool": "test-FuSa", "version": "0.1.0", "specVersion": "1.8",
				})
				return b, nil, 0
			}
			return []byte("test-FuSa 0.1.0\n"), nil, 0

		case "init":
			fusa := `{"configVersion":"1.0","project":{"name":"conform-test","version":"0.1.0"},"standard":"iso26262"}`
			reqs := `{"requirements":[]}`
			_ = writeTestFile(filepath.Join(dir, ".fusa.json"), []byte(fusa))
			_ = writeTestFile(filepath.Join(dir, ".fusa-reqs.json"), []byte(reqs))
			return nil, nil, 0

		case "check":
			doc := map[string]interface{}{
				"schemaVersion": "1.8", "kind": "check-report",
				"tool": "test-FuSa", "toolVersion": "0.1.0",
				"language": "go", "generatedAt": "2026-06-10T12:00:00Z",
				"projectRoot": dir,
				"findings": []map[string]interface{}{
					{"ruleId": "LINT001", "severity": "WARNING",
						"message":  "function has 65 lines",
						"location": map[string]interface{}{"file": "main.go", "line": 1},
						"category": "lint", "remediation": "split function",
						"fingerprint": "sha256:b2beafa767506a074d84e95dd9955427dc9806c197272b84e1dd7360e50cb602"},
				},
				"summary": map[string]int{"total": 1, "errors": 0, "warnings": 1, "infos": 0},
			}
			b, _ := json.Marshal(doc)
			return b, nil, 1

		case "trace":
			doc := map[string]interface{}{
				"schemaVersion": "1.8", "kind": "trace-matrix",
				"tool": "test-FuSa", "toolVersion": "0.1.0",
				"language": "go", "generatedAt": "2026-06-10T12:00:00Z",
				"projectRoot":  dir,
				"requirements": []interface{}{}, "tags": []interface{}{},
				"coverage": map[string]int{
					"totalRequirements": 0, "tracedRequirements": 0,
					"testedRequirements": 0, "secTestedRequirements": 0,
				},
			}
			b, _ := json.Marshal(doc)
			return b, nil, 0

		case "qualify":
			outFile := ""
			for i, a := range args {
				if a == "--output" && i+1 < len(args) {
					outFile = args[i+1]
				}
			}
			doc := map[string]interface{}{
				"schemaVersion": "1.8", "kind": "qualification",
				"tool": "test-FuSa", "toolVersion": "0.1.0",
				"language": "go", "generatedAt": "2026-06-10T12:00:00Z",
				"total": 1, "passed": 1, "failed": 0,
				"results": []map[string]string{{"name": "rule-LINT001", "result": "PASS"}},
			}
			b, _ := json.Marshal(doc)
			if outFile != "" {
				_ = writeTestFile(outFile, b)
				return nil, nil, 0
			}
			return b, nil, 0

		case "release":
			outDir := dir
			for i, a := range args {
				if a == "--output-dir" && i+1 < len(args) {
					outDir = args[i+1]
				}
			}
			sbom := map[string]interface{}{
				"schemaVersion": "1.8", "kind": "sbom",
				"tool": "test-FuSa", "toolVersion": "0.1.0",
				"language": "go", "generatedAt": "2026-06-10T12:00:00Z",
				"format": "x-FuSa SBOM v1", "module": "github.com/test/test",
				"components": []interface{}{},
			}
			b, _ := json.Marshal(sbom)
			_ = writeTestFile(filepath.Join(outDir, "sbom.json"), b)
			return nil, nil, 0

		case "audit-pack":
			outFile := filepath.Join(dir, "audit-pack.zip")
			for i, a := range args {
				if a == "--output" && i+1 < len(args) {
					outFile = args[i+1]
				}
			}
			_ = writeTestZIP(outFile)
			return nil, nil, 0

		case "capabilities":
			doc := map[string]interface{}{
				"schemaVersion": "1.8", "kind": "capabilities",
				"tool": "test-FuSa", "toolVersion": "0.1.0",
				"language": "go", "generatedAt": "2026-06-10T12:00:00Z",
				"specVersion": "1.8",
				"commands":    []string{"check", "trace", "qualify", "release", "audit-pack"},
				"formats":     map[string][]string{"check": {"text", "json"}},
				"standards":   []string{"iso26262"},
			}
			b, _ := json.Marshal(doc)
			return b, nil, 0
		}
		return nil, nil, 1
	}
}

// TestRunConformingTool verifies all checks pass for a fully conforming fake tool.
//
//fusa:test REQ-FO-CNF004
//fusa:test REQ-FO-CNF005
//fusa:test REQ-FO-CNF010
func TestRunConformingTool(t *testing.T) {
	rep, err := Run("testBinary", Options{
		TempDir: t.TempDir(),
		RunFunc: newConformingRunFunc(),
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	pass, fail, skip := rep.Summary()
	if fail > 0 {
		t.Errorf("%d FAIL:", fail)
		for _, r := range rep.Results {
			if r.Status == StatusFail {
				t.Errorf("  [%s] %s — %s", r.Level, r.ID, r.Detail)
			}
		}
	}
	t.Logf("PASS=%d FAIL=%d SKIP=%d", pass, fail, skip)
}

// TestHasFailures verifies MUST-fail detection.
//
//fusa:test REQ-FO-CNF001
//fusa:test REQ-FO-CNF002
//fusa:test REQ-FO-CNF003
func TestHasFailures(t *testing.T) {
	rep := &Report{}
	if rep.HasFailures() {
		t.Fatal("empty report should not have failures")
	}
	rep.Results = []Result{{Level: LevelSHOULD, Status: StatusFail}}
	if rep.HasFailures() {
		t.Fatal("SHOULD fail should not trigger HasFailures")
	}
	rep.Results = append(rep.Results, Result{Level: LevelMUST, Status: StatusFail})
	if !rep.HasFailures() {
		t.Fatal("MUST fail must trigger HasFailures")
	}
}

// TestSummaryCount verifies PASS/FAIL/SKIP totals.
//
//fusa:test REQ-FO-CNF003
func TestSummaryCount(t *testing.T) {
	rep := &Report{Results: []Result{
		{Status: StatusPass}, {Status: StatusPass},
		{Status: StatusFail}, {Status: StatusSkip},
	}}
	pass, fail, skip := rep.Summary()
	if pass != 2 || fail != 1 || skip != 1 {
		t.Errorf("want 2/1/1, got %d/%d/%d", pass, fail, skip)
	}
}

// TestLangFromBinary validates §1.1 registry mapping.
//
//fusa:test REQ-FO-CNF007
func TestLangFromBinary(t *testing.T) {
	cases := map[string]string{
		"gofusa": "go", "cfusa": "c", "cpfusa": "cpp",
		"rsfusa": "rust", "pyfusa": "python", "jfusa": "java",
		"unknown": "",
	}
	for bin, want := range cases {
		if got := langFromBinary(bin); got != want {
			t.Errorf("langFromBinary(%q) = %q, want %q", bin, got, want)
		}
	}
}

// TestValidateHeaderPass verifies a correct header produces no errors.
//
//fusa:test REQ-FO-CNF008
func TestValidateHeaderPass(t *testing.T) {
	h := commonHeader{
		SchemaVersion: "1.8", Kind: "check-report",
		Tool: "go-FuSa", ToolVersion: "0.23.0",
		Language: "go", GeneratedAt: "2026-06-10T12:00:00Z",
	}
	if errs := validateHeader(h, "check-report"); len(errs) > 0 {
		t.Errorf("unexpected errors: %v", errs)
	}
}

// TestValidateHeaderAlsoAcceptsReport verifies "report" satisfies "check-report" expected kind.
//
//fusa:test REQ-FO-CNF008
func TestValidateHeaderAlsoAcceptsReport(t *testing.T) {
	h := commonHeader{
		SchemaVersion: "1.8", Kind: "report",
		Tool: "go-FuSa", ToolVersion: "0.1.0",
		Language: "go", GeneratedAt: "2026-06-10T12:00:00Z",
	}
	if errs := validateHeader(h, "check-report"); len(errs) > 0 {
		t.Errorf("'report' kind should be accepted for 'check-report': %v", errs)
	}
}

// TestValidateHeaderFail verifies missing/wrong fields are caught.
//
//fusa:test REQ-FO-CNF008
func TestValidateHeaderFail(t *testing.T) {
	cases := []struct {
		name, want string
		h          commonHeader
	}{
		{"missing schemaVersion",
			"schemaVersion",
			commonHeader{Kind: "sbom", Tool: "t", ToolVersion: "0.1", Language: "go", GeneratedAt: "x"}},
		{"missing kind",
			"kind",
			commonHeader{SchemaVersion: "1.8", Tool: "t", ToolVersion: "0.1", Language: "go", GeneratedAt: "x"}},
		{"wrong kind",
			"kind",
			commonHeader{SchemaVersion: "1.8", Kind: "sbom", Tool: "t", ToolVersion: "0.1", Language: "go", GeneratedAt: "x"}},
		{"missing language",
			"language",
			commonHeader{SchemaVersion: "1.8", Kind: "qualification", Tool: "t", ToolVersion: "0.1", GeneratedAt: "x"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			errs := validateHeader(tc.h, "qualification")
			found := false
			for _, e := range errs {
				if strings.Contains(e, tc.want) {
					found = true
				}
			}
			if !found {
				t.Errorf("expected error containing %q, got %v", tc.want, errs)
			}
		})
	}
}

// TestVersionLineRE validates the version-output regex from §9.1.
//
//fusa:test REQ-FO-CNF009
func TestVersionLineRE(t *testing.T) {
	valid := []string{"go-FuSa 0.23.0", "c-FuSa 1.2.3-alpha", "cpp-FuSa 0.1.0+build42"}
	invalid := []string{"go-FuSa", "version 1", ""}
	for _, s := range valid {
		if m := versionLineRE.FindStringSubmatch(s); m == nil {
			t.Errorf("valid %q did not match", s)
		}
	}
	for _, s := range invalid {
		if m := versionLineRE.FindStringSubmatch(s); m != nil {
			t.Errorf("invalid %q matched: %v", s, m)
		}
	}
}

// TestRenderText verifies text rendering.
//
//fusa:test REQ-FO-CNF006
func TestRenderText(t *testing.T) {
	rep := &Report{
		Binary: "/usr/local/bin/gofusa", Tool: "go-FuSa", ToolVersion: "0.23.0",
		Language: "go", SpecVersion: "1.8",
		Results: []Result{
			{ID: "version/output-format", Section: "§9.1", Level: LevelMUST, Name: "version", Status: StatusPass},
			{ID: "check/common-header", Section: "§3.1", Level: LevelMUST, Name: "header", Status: StatusFail, Detail: "missing kind"},
			{ID: "capabilities/schema", Section: "§9.1", Level: LevelSHOULD, Name: "caps", Status: StatusSkip},
		},
	}
	var buf bytes.Buffer
	if err := Render(&buf, rep, "text"); err != nil {
		t.Fatalf("Render text: %v", err)
	}
	out := buf.String()
	for _, want := range []string{"FAIL", "§3.1", "missing kind"} {
		if !strings.Contains(out, want) {
			t.Errorf("text output missing %q", want)
		}
	}
}

// TestRenderJSON verifies JSON rendering round-trips.
//
//fusa:test REQ-FO-CNF006
func TestRenderJSON(t *testing.T) {
	rep := &Report{Tool: "go-FuSa", ToolVersion: "0.1.0"}
	var buf bytes.Buffer
	if err := Render(&buf, rep, "json"); err != nil {
		t.Fatalf("Render json: %v", err)
	}
	var got Report
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Tool != rep.Tool {
		t.Errorf("tool %q, want %q", got.Tool, rep.Tool)
	}
}

// TestRenderUnknownFormat returns an error for unknown formats.
//
//fusa:test REQ-FO-CNF006
func TestRenderUnknownFormat(t *testing.T) {
	if err := Render(&bytes.Buffer{}, &Report{}, "xml"); err == nil {
		t.Error("expected error for unknown format")
	}
}

// TestRenderHTML verifies HTML rendering contains expected content.
//
//fusa:test REQ-FO-CNF018
func TestRenderHTML(t *testing.T) {
	rep := &Report{
		Binary: "/usr/local/bin/gofusa", Tool: "go-FuSa", ToolVersion: "0.30.0",
		Language: "go", SpecVersion: "1.10",
		Results: []Result{
			{ID: "version/output-format", Section: "§9.1", Level: LevelMUST, Name: "version output", Status: StatusPass},
			{ID: "check/common-header", Section: "§3.1", Level: LevelMUST, Name: "header", Status: StatusFail, Detail: "missing kind"},
			{ID: "capabilities/schema", Section: "§9.1", Level: LevelSHOULD, Name: "caps", Status: StatusSkip},
		},
	}
	var buf bytes.Buffer
	if err := Render(&buf, rep, "html"); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{"<!doctype html>", "go-FuSa", "§3.1", "missing kind", "FAIL"} {
		if !strings.Contains(out, want) {
			t.Errorf("html missing %q", want)
		}
	}
}

// TestRenderHTMLPass verifies a fully-passing report shows PASS badge.
//
//fusa:test REQ-FO-CNF018
func TestRenderHTMLPass(t *testing.T) {
	rep := &Report{Tool: "go-FuSa", ToolVersion: "0.30.0", Language: "go",
		Results: []Result{
			{Status: StatusPass, Level: LevelMUST, Section: "§1", Name: "ok"},
		},
	}
	var buf bytes.Buffer
	if err := Render(&buf, rep, "html"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), ">PASS<") {
		t.Error("html: expected PASS badge for all-passing report")
	}
}

// TestRenderMarkdown verifies GFM markdown rendering contains expected content.
//
//fusa:test REQ-FO-CNF019
func TestRenderMarkdown(t *testing.T) {
	rep := &Report{
		Tool:        "gofusa",
		ToolVersion: "0.30.0",
		Language:    "go",
		SpecVersion: "1.9",
		Results: []Result{
			{Status: StatusPass, Level: LevelMUST, Section: "§3.1", Name: "valid kind"},
			{Status: StatusFail, Level: LevelMUST, Section: "§4.1", Name: "missing kind", Detail: "got nil"},
			{Status: StatusSkip, Level: LevelSHOULD, Section: "§5", Name: "skipped check"},
		},
	}
	var buf bytes.Buffer
	if err := Render(&buf, rep, "markdown"); err != nil {
		t.Fatalf("Render markdown: %v", err)
	}
	out := buf.String()
	for _, want := range []string{"# FuSaOps", "gofusa", "**FAIL**", "| Result |", "§3.1", "missing kind", "got nil", "⏭"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q in markdown:\n%s", want, out)
		}
	}
}

// TestRenderMarkdownAlias verifies "md" is accepted as an alias.
//
//fusa:test REQ-FO-CNF019
func TestRenderMarkdownAlias(t *testing.T) {
	rep := &Report{Tool: "gofusa", ToolVersion: "0.30.0", Results: []Result{
		{Status: StatusPass, Level: LevelMUST, Section: "§1", Name: "ok"},
	}}
	var buf bytes.Buffer
	if err := Render(&buf, rep, "md"); err != nil {
		t.Fatalf("Render md alias: %v", err)
	}
	if !strings.Contains(buf.String(), "# FuSaOps") {
		t.Error("expected markdown header from md alias")
	}
}

// TestRenderMarkdownPass verifies green badge for all-PASS report.
//
//fusa:test REQ-FO-CNF019
func TestRenderMarkdownPass(t *testing.T) {
	rep := &Report{Tool: "gofusa", ToolVersion: "0.30.0", Results: []Result{
		{Status: StatusPass, Level: LevelMUST, Section: "§1", Name: "ok"},
	}}
	var buf bytes.Buffer
	if err := Render(&buf, rep, "markdown"); err != nil {
		t.Fatalf("Render markdown pass: %v", err)
	}
	if !strings.Contains(buf.String(), "🟢") {
		t.Error("expected green badge for PASS conformance report")
	}
}

// TestDecodeJSON verifies the noise-stripping JSON decoder.
func TestDecodeJSON(t *testing.T) {
	raw := []byte("noise\n{\"foo\":1}\nmore")
	var v map[string]int
	if err := decodeJSON(raw, &v); err != nil {
		t.Fatalf("decodeJSON: %v", err)
	}
	if v["foo"] != 1 {
		t.Errorf("want foo=1, got %d", v["foo"])
	}
}

// TestNonConformingQualifyKeys verifies tests_passed/tests_failed are detected.
//
//fusa:test REQ-FO-CNF013
func TestNonConformingQualifyKeys(t *testing.T) {
	bad := func(dir, binary string, args ...string) ([]byte, []byte, int) {
		if len(args) == 0 {
			return nil, nil, 1
		}
		switch args[0] {
		case "version":
			return []byte("bad-FuSa 0.1.0\n"), nil, 0
		case "qualify":
			outFile := ""
			for i, a := range args {
				if a == "--output" && i+1 < len(args) {
					outFile = args[i+1]
				}
			}
			doc := map[string]interface{}{
				"schemaVersion": "1.8", "kind": "qualification",
				"tool": "bad-FuSa", "toolVersion": "0.1.0",
				"language": "go", "generatedAt": "2026-06-10T12:00:00Z",
				"tests_passed": 3, "tests_failed": 0,
				"results": []map[string]string{{"name": "t1", "result": "PASS"}},
			}
			b, _ := json.Marshal(doc)
			if outFile != "" {
				_ = writeTestFile(outFile, b)
				return nil, nil, 0
			}
			return b, nil, 0
		default:
			return nil, nil, 0
		}
	}
	rep, err := Run("badBinary", Options{TempDir: t.TempDir(), RunFunc: bad})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	for _, r := range rep.Results {
		if r.ID == "qualify/key-names" && r.Status == StatusFail {
			return
		}
	}
	t.Error("expected qualify/key-names FAIL for tests_passed/tests_failed keys")
}

// TestNonConformingAbsolutePath verifies absolute location.file is detected.
//
//fusa:test REQ-FO-CNF011
func TestNonConformingAbsolutePath(t *testing.T) {
	bad := func(dir, binary string, args ...string) ([]byte, []byte, int) {
		if len(args) == 0 {
			return nil, nil, 1
		}
		switch args[0] {
		case "version":
			return []byte("abs-FuSa 0.1.0\n"), nil, 0
		case "check":
			doc := fmt.Sprintf(
				`{"schemaVersion":"1.8","kind":"check-report","tool":"abs-FuSa","toolVersion":"0.1.0","language":"go","generatedAt":"2026-06-10T00:00:00Z","projectRoot":"%s","findings":[{"ruleId":"LINT001","severity":"WARNING","message":"msg","location":{"file":"/absolute/path.go","line":1}}],"summary":{"total":1,"errors":0,"warnings":1,"infos":0}}`,
				filepath.ToSlash(dir)) // ToSlash: backslashes in Windows tempdir are invalid JSON escapes
			return []byte(doc), nil, 0
		default:
			return nil, nil, 0
		}
	}
	rep, _ := Run("absBinary", Options{TempDir: t.TempDir(), RunFunc: bad})
	for _, r := range rep.Results {
		if r.ID == "check/finding-location-relative" && r.Status == StatusFail {
			return
		}
	}
	t.Error("expected check/finding-location-relative FAIL for absolute path")
}

// TestNonConformingManifestPrefixedSHA verifies sha256: prefix in manifest is caught.
//
//fusa:test REQ-FO-CNF015
func TestNonConformingManifestPrefixedSHA(t *testing.T) {
	bad := func(dir, binary string, args ...string) ([]byte, []byte, int) {
		if len(args) == 0 {
			return nil, nil, 1
		}
		switch args[0] {
		case "version":
			return []byte("pfx-FuSa 0.1.0\n"), nil, 0
		case "audit-pack":
			outFile := filepath.Join(dir, "audit-pack.zip")
			for i, a := range args {
				if a == "--output" && i+1 < len(args) {
					outFile = args[i+1]
				}
			}
			manifest := map[string]interface{}{
				"schemaVersion": "1.8", "kind": "audit-manifest",
				"tool": "pfx-FuSa", "toolVersion": "0.1.0",
				"language": "go", "generatedAt": "2026-06-10T12:00:00Z",
				"module": "github.com/test/test",
				"files": []map[string]interface{}{
					{"path": "sbom.json", "size": 2,
						"sha256": "sha256:a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2"},
				},
			}
			mb, _ := json.Marshal(manifest)
			f, _ := os.Create(outFile)
			zw := zip.NewWriter(f)
			w, _ := zw.Create("manifest.json")
			_, _ = w.Write(mb)
			_ = zw.Close()
			_ = f.Close()
			return nil, nil, 0
		default:
			return nil, nil, 0
		}
	}
	rep, _ := Run("pfxBinary", Options{TempDir: t.TempDir(), RunFunc: bad})
	for _, r := range rep.Results {
		if r.ID == "audit-pack/manifest-sha256-bare" && r.Status == StatusFail {
			return
		}
	}
	t.Error("expected audit-pack/manifest-sha256-bare FAIL for sha256:-prefixed value")
}

// TestFingerprintVectors validates the §4.2 algorithm against the spec/vectors/fingerprint-cases.json values.
//
//fusa:test REQ-FO-CNF017
func TestFingerprintVectors(t *testing.T) {
	cases := []struct {
		ruleID, file, msg, want string
	}{
		{"LINT001", "src/foo.go", "function foo has 65 lines",
			"sha256:a1b252d0e9771755eb62612606b6291d4e725a8a6d0aa9ee830209e481f50f1e"},
		{"FUSA004", "src/bar.go", "unreachable code after return statement",
			"sha256:5194d908a5c8e4a479735b81fb52956e3116aa48adba51fb6e21cfdf0b2f8260"},
		{"SEC001", "cmd/main.go", "use of unsafe.Pointer without documentation",
			"sha256:586acffd2484322e01678f5c3b5d5839fd3af0323fcbf7a47102ae552a2799a4"},
		{"LINT001", "src/foo.go", "function bar has 120 lines (limit 60)",
			"sha256:65f1feb1219e4fcdea4d142fbb7c9e735dfd705f2579aad9d3a4d373fd1bb3d4"},
	}
	for _, tc := range cases {
		got := Fingerprint(tc.ruleID, tc.file, tc.msg)
		if got != tc.want {
			t.Errorf("Fingerprint(%q,%q,%q)\n  got  %s\n  want %s",
				tc.ruleID, tc.file, tc.msg, got, tc.want)
		}
	}
}

// TestNormalizeMessage verifies digit normalisation and whitespace collapsing.
//
//fusa:test REQ-FO-CNF017
func TestNormalizeMessage(t *testing.T) {
	cases := []struct{ in, want string }{
		{"function foo has 65 lines", "function foo has # lines"},
		{"unreachable code after return statement", "unreachable code after return statement"},
		{"function bar has 120 lines (limit 60)", "function bar has # lines (limit #)"},
		{"  multiple   spaces  ", "multiple spaces"},
		{"non-ascii café message", "non-ascii café message"},
	}
	for _, tc := range cases {
		if got := normalizeMessage(tc.in); got != tc.want {
			t.Errorf("normalizeMessage(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// TestWriteSourceFilesLanguages verifies scaffold creates language-specific files.
//
//fusa:test REQ-FO-CNF005
func TestWriteSourceFilesLanguages(t *testing.T) {
	cases := map[string]string{
		"go":      "main.go",
		"c":       "main.c",
		"cpp":     "main.cpp",
		"rust":    filepath.Join("src", "main.rs"),
		"python":  "main.py",
		"java":    "Main.java",
		"unknown": "",
	}
	for lang, relPath := range cases {
		t.Run(lang, func(t *testing.T) {
			dir := t.TempDir()
			r := &runner{dir: dir, lang: lang}
			if err := r.writeSourceFiles(); err != nil {
				t.Fatalf("writeSourceFiles(%q): %v", lang, err)
			}
			if relPath != "" {
				if _, err := os.Stat(filepath.Join(dir, relPath)); err != nil {
					t.Errorf("expected %q to exist: %v", relPath, err)
				}
			}
		})
	}
}

// TestCheckCheckNoFindings verifies finding-level checks are vacuously satisfied with no findings.
//
//fusa:test REQ-FO-CNF011
func TestCheckCheckNoFindings(t *testing.T) {
	base := newConformingRunFunc()
	noFindings := func(dir, binary string, args ...string) ([]byte, []byte, int) {
		if len(args) > 0 && args[0] == "check" {
			doc := map[string]interface{}{
				"schemaVersion": "1.8", "kind": "check-report",
				"tool": "test-FuSa", "toolVersion": "0.1.0",
				"language": "go", "generatedAt": "2026-06-10T12:00:00Z",
				"projectRoot": dir,
				"findings":    []interface{}{},
				"summary":     map[string]int{"total": 0, "errors": 0, "warnings": 0, "infos": 0},
			}
			b, _ := json.Marshal(doc)
			return b, nil, 0
		}
		return base(dir, binary, args...)
	}
	rep, err := Run("emptyBinary", Options{TempDir: t.TempDir(), RunFunc: noFindings})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	for _, r := range rep.Results {
		if r.Status == StatusFail {
			t.Errorf("unexpected FAIL: [%s] %s — %s", r.Level, r.ID, r.Detail)
		}
	}
}

// TestCheckCategoryFail verifies unknown category is caught.
//
//fusa:test REQ-FO-CNF011
func TestCheckCategoryFail(t *testing.T) {
	base := newConformingRunFunc()
	badCat := func(dir, binary string, args ...string) ([]byte, []byte, int) {
		if len(args) > 0 && args[0] == "check" {
			doc := map[string]interface{}{
				"schemaVersion": "1.8", "kind": "check-report",
				"tool": "test-FuSa", "toolVersion": "0.1.0",
				"language": "go", "generatedAt": "2026-06-10T12:00:00Z",
				"projectRoot": dir,
				"findings": []map[string]interface{}{
					{"ruleId": "LINT001", "severity": "INFO",
						"message":     "note",
						"location":    map[string]interface{}{"file": "main.go"},
						"category":    "notacategory",
						"remediation": "fix it",
						"fingerprint": "sha256:5529f6e552b61370fcd04fdcf22d73c495c5c9d2746b9293233f0dbb08fe6b27"},
				},
				"summary": map[string]int{"total": 1, "errors": 0, "warnings": 0, "infos": 1},
			}
			b, _ := json.Marshal(doc)
			return b, nil, 0
		}
		return base(dir, binary, args...)
	}
	rep, _ := Run("catBinary", Options{TempDir: t.TempDir(), RunFunc: badCat})
	for _, r := range rep.Results {
		if r.ID == "check/category-enum" && r.Status == StatusFail {
			return
		}
	}
	t.Error("expected check/category-enum FAIL for unknown category")
}

// TestCheckTraceFlatSchema verifies flat trace schema is detected.
//
//fusa:test REQ-FO-CNF012
func TestCheckTraceFlatSchema(t *testing.T) {
	flat := func(dir, binary string, args ...string) ([]byte, []byte, int) {
		switch args[0] {
		case "version":
			return []byte("flat-FuSa 0.1.0\n"), nil, 0
		case "trace":
			// Non-conformant: flat total/traced/tested instead of qualified names
			doc := map[string]interface{}{
				"schemaVersion": "1.8", "kind": "trace-matrix",
				"tool": "flat-FuSa", "toolVersion": "0.1.0",
				"language": "go", "generatedAt": "2026-06-10T12:00:00Z",
				"projectRoot":  dir,
				"requirements": []interface{}{},
				"tags":         []interface{}{},
				"coverage": map[string]int{
					"total": 0, "traced": 0, "tested": 0,
				},
			}
			b, _ := json.Marshal(doc)
			return b, nil, 0
		default:
			return nil, nil, 0
		}
	}
	rep, _ := Run("flatTraceBinary", Options{TempDir: t.TempDir(), RunFunc: flat})
	for _, r := range rep.Results {
		if r.ID == "trace/coverage-schema" && r.Status == StatusFail {
			return
		}
	}
	t.Error("expected trace/coverage-schema FAIL for flat schema")
}

// TestCheckVersionFail verifies bad version output is caught.
//
//fusa:test REQ-FO-CNF009
func TestCheckVersionFail(t *testing.T) {
	bad := func(dir, binary string, args ...string) ([]byte, []byte, int) {
		if args[0] == "version" {
			return []byte("not a valid version\n"), nil, 0
		}
		return nil, nil, 0
	}
	rep, _ := Run("vbadBinary", Options{TempDir: t.TempDir(), RunFunc: bad})
	for _, r := range rep.Results {
		if r.ID == "version/output-format" && r.Status == StatusFail {
			return
		}
	}
	t.Error("expected version/output-format FAIL for invalid output")
}

// TestCheckVersionExitFail verifies non-zero version exit is caught.
//
//fusa:test REQ-FO-CNF009
func TestCheckVersionExitFail(t *testing.T) {
	bad := func(dir, binary string, args ...string) ([]byte, []byte, int) {
		return nil, nil, 1
	}
	rep, _ := Run("vexitBinary", Options{TempDir: t.TempDir(), RunFunc: bad})
	for _, r := range rep.Results {
		if r.ID == "version/exit-code" && r.Status == StatusFail {
			return
		}
	}
	t.Error("expected version/exit-code FAIL for non-zero exit")
}

// TestCheckVersionJSONFail verifies bad version --format json is caught.
//
//fusa:test REQ-FO-CNF009
func TestCheckVersionJSONFail(t *testing.T) {
	bad := func(dir, binary string, args ...string) ([]byte, []byte, int) {
		if args[0] == "version" {
			if len(args) >= 3 && args[1] == "--format" {
				// Missing specVersion
				b, _ := json.Marshal(map[string]string{"tool": "t", "version": "0.1.0"})
				return b, nil, 0
			}
			return []byte("t 0.1.0\n"), nil, 0
		}
		return nil, nil, 0
	}
	rep, _ := Run("vjsonBinary", Options{TempDir: t.TempDir(), RunFunc: bad})
	for _, r := range rep.Results {
		if r.ID == "version/json-format" && r.Status == StatusFail {
			return
		}
	}
	t.Error("expected version/json-format FAIL for missing specVersion")
}

// TestCheckQualifyRuntimeError verifies runtime error from qualify is skipped.
//
//fusa:test REQ-FO-CNF013
func TestCheckQualifyRuntimeError(t *testing.T) {
	bad := func(dir, binary string, args ...string) ([]byte, []byte, int) {
		if args[0] == "version" {
			return []byte("q-FuSa 0.1.0\n"), nil, 0
		}
		if args[0] == "qualify" {
			return nil, nil, 3
		}
		return nil, nil, 0
	}
	rep, _ := Run("qualBinary", Options{TempDir: t.TempDir(), RunFunc: bad})
	for _, r := range rep.Results {
		if r.ID == "qualify/output" && r.Status == StatusSkip {
			return
		}
	}
	t.Error("expected qualify/output SKIP for runtime error exit")
}

// TestCheckReleaseMissingSBOM verifies missing sbom.json is caught.
//
//fusa:test REQ-FO-CNF014
func TestCheckReleaseMissingSBOM(t *testing.T) {
	bad := func(dir, binary string, args ...string) ([]byte, []byte, int) {
		if args[0] == "version" {
			return []byte("rel-FuSa 0.1.0\n"), nil, 0
		}
		if args[0] == "release" {
			// Exits 0 but writes nothing.
			return nil, nil, 0
		}
		return nil, nil, 0
	}
	rep, _ := Run("relBinary", Options{TempDir: t.TempDir(), RunFunc: bad})
	for _, r := range rep.Results {
		if r.ID == "release/sbom-written" && r.Status == StatusFail {
			return
		}
	}
	t.Error("expected release/sbom-written FAIL when sbom.json is absent")
}

// TestCheckCapabilitiesFail verifies bad capabilities schema is caught.
//
//fusa:test REQ-FO-CNF016
func TestCheckCapabilitiesFail(t *testing.T) {
	bad := func(dir, binary string, args ...string) ([]byte, []byte, int) {
		if args[0] == "version" {
			return []byte("cap-FuSa 0.1.0\n"), nil, 0
		}
		if args[0] == "capabilities" {
			// Missing specVersion and commands
			b, _ := json.Marshal(map[string]string{
				"schemaVersion": "1.8", "kind": "capabilities",
				"tool": "cap-FuSa", "toolVersion": "0.1.0",
				"language": "go", "generatedAt": "2026-06-10T12:00:00Z",
			})
			return b, nil, 0
		}
		return nil, nil, 0
	}
	rep, _ := Run("capBinary", Options{TempDir: t.TempDir(), RunFunc: bad})
	for _, r := range rep.Results {
		if r.ID == "capabilities/schema" && r.Status == StatusFail {
			return
		}
	}
	t.Error("expected capabilities/schema FAIL for missing specVersion/commands")
}

// TestFingerprintNonASCII verifies hasNonASCII detection.
//
//fusa:test REQ-FO-CNF017
func TestFingerprintNonASCII(t *testing.T) {
	if !hasNonASCII("café") {
		t.Error("café should be detected as non-ASCII")
	}
	if hasNonASCII("pure ASCII") {
		t.Error("pure ASCII should not be detected as non-ASCII")
	}
}

// TestScaffoldWritesFiles verifies scaffold creates the fixture files.
//
//fusa:test REQ-FO-CNF005
func TestScaffoldWritesFiles(t *testing.T) {
	for _, tc := range []struct {
		lang, binary, wantFile string
	}{
		{"go", "gofusa", "main.go"},
		{"rust", "rsfusa", filepath.Join("src", "main.rs")},
		{"python", "pyfusa", "main.py"},
		{"java", "jfusa", "Main.java"},
	} {
		t.Run(tc.lang, func(t *testing.T) {
			dir := t.TempDir()
			r := &runner{
				dir:    dir,
				binary: tc.binary,
				run:    func(d, b string, a ...string) ([]byte, []byte, int) { return nil, nil, 0 },
				report: &Report{},
				lang:   tc.lang,
			}
			if err := r.scaffold(); err != nil {
				t.Fatalf("scaffold: %v", err)
			}
			for _, name := range []string{".fusa.json", ".fusa-reqs.json", tc.wantFile} {
				if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
					t.Errorf("expected %q after scaffold: %v", name, err)
				}
			}
		})
	}
}
