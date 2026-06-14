package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func runArgs(t *testing.T, args ...string) (int, string, string) {
	t.Helper()
	var out, errb bytes.Buffer
	code := run(args, &out, &errb)
	return code, out.String(), errb.String()
}

//fusa:test REQ-FO-CLI002
func TestNoArgsShowsUsage(t *testing.T) {
	code, out, _ := runArgs(t)
	if code != 1 || !strings.Contains(out, "Usage:") {
		t.Errorf("no-args: code=%d out=%q", code, out)
	}
}

//fusa:test REQ-FO-CLI001
func TestUnknownCommand(t *testing.T) {
	code, _, errb := runArgs(t, "frobnicate")
	if code != 1 || !strings.Contains(errb, "unknown command") {
		t.Errorf("unknown cmd: code=%d err=%q", code, errb)
	}
}

//fusa:test REQ-FO-CLI003
func TestVersion(t *testing.T) {
	code, out, _ := runArgs(t, "version")
	if code != 0 || !strings.Contains(out, "fusaops") {
		t.Errorf("version: code=%d out=%q", code, out)
	}
}

//fusa:test REQ-FO-CLI053
func TestVersionJSON(t *testing.T) {
	code, out, errb := runArgs(t, "version", "--format", "json")
	if code != 0 {
		t.Fatalf("version --format json: code=%d err=%q", code, errb)
	}
	for _, want := range []string{`"tool"`, `"version"`, `"specVersion"`, "fusaops"} {
		if !strings.Contains(out, want) {
			t.Errorf("version json missing %q: %s", want, out)
		}
	}
}

//fusa:test REQ-FO-CLI053
func TestVersionInvalidFormat(t *testing.T) {
	code, _, errb := runArgs(t, "version", "--format", "xml")
	if code != 2 || !strings.Contains(errb, "format") {
		t.Errorf("invalid format: code=%d err=%q", code, errb)
	}
}

func TestHelp(t *testing.T) {
	code, out, _ := runArgs(t, "help")
	if code != 0 || !strings.Contains(out, "Commands:") {
		t.Errorf("help: code=%d out=%q", code, out)
	}
}

//fusa:test REQ-FO-CLI004
func TestInitCreatesConfig(t *testing.T) {
	dir := t.TempDir()
	code, out, errb := runArgs(t, "init", "--dir", dir, "--name", "demo")
	if code != 0 {
		t.Fatalf("init failed: %d %q", code, errb)
	}
	if !strings.Contains(out, "Wrote") {
		t.Errorf("init out: %q", out)
	}
	if _, err := os.Stat(filepath.Join(dir, ".fusaops.json")); err != nil {
		t.Errorf("config not written: %v", err)
	}
	// Second init without --force must fail.
	code, _, _ = runArgs(t, "init", "--dir", dir)
	if code != 1 {
		t.Errorf("re-init without --force should fail, got %d", code)
	}
}

//fusa:test REQ-FO-CLI005
func TestScanDetectsLanguages(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main"), 0o600); err != nil {
		t.Fatal(err)
	}
	code, out, _ := runArgs(t, "scan", "--dir", dir)
	if code != 0 || !strings.Contains(out, "go") {
		t.Errorf("scan: code=%d out=%q", code, out)
	}
}

func TestScanEmptyDir(t *testing.T) {
	code, out, _ := runArgs(t, "scan", "--dir", t.TempDir())
	if code != 0 || !strings.Contains(out, "No supported languages") {
		t.Errorf("scan empty: code=%d out=%q", code, out)
	}
}

//fusa:test REQ-FO-CLI006
func TestAdaptersLists(t *testing.T) {
	code, out, _ := runArgs(t, "adapters")
	if code != 0 {
		t.Fatalf("adapters: code=%d", code)
	}
	for _, want := range []string{"gofusa", "cfusa", "cpfusa"} {
		if !strings.Contains(out, want) {
			t.Errorf("adapters output missing %q", want)
		}
	}
}

//fusa:test REQ-FO-CLI050
func TestAdaptersJSON(t *testing.T) {
	code, out, _ := runArgs(t, "adapters", "--format", "json")
	if code != 0 {
		t.Fatalf("adapters --format json: code=%d", code)
	}
	for _, want := range []string{`"name"`, `"tool"`, `"language"`, `"available"`, "gofusa"} {
		if !strings.Contains(out, want) {
			t.Errorf("adapters json missing %q:\n%s", want, out)
		}
	}
}

//fusa:test REQ-FO-CLI008
func TestCheckNoLanguages(t *testing.T) {
	// Empty dir → no adapters → exit 1 with message.
	code, _, errb := runArgs(t, "check", "--dir", t.TempDir())
	if code != 1 || !strings.Contains(errb, "no supported languages") {
		t.Errorf("check empty: code=%d err=%q", code, errb)
	}
}

// TestPolicyCheck verifies fusaops policy evaluates rules and produces a report.
//
//fusa:test REQ-FO-CLI024
func TestPolicyCheck(t *testing.T) {
	dir := goProject(t)
	polPath := filepath.Join(dir, "policy.json")
	polData, _ := json.Marshal(map[string]any{
		"name":  "test",
		"rules": []map[string]any{{"id": "R1", "maxErrors": 10}},
	})
	_ = os.WriteFile(polPath, polData, 0o644)
	code, out, errb := runArgs(t, "policy", "--dir", dir, "--policy", polPath, "--format", "text")
	if code != 0 {
		t.Fatalf("policy: code=%d err=%q", code, errb)
	}
	if !strings.Contains(out, "Policy:") {
		t.Errorf("policy output missing header: %q", out)
	}
}

// TestPolicyMissingFile verifies a missing policy file returns exit 1.
//
//fusa:test REQ-FO-CLI024
func TestPolicyMissingFile(t *testing.T) {
	dir := goProject(t)
	code, _, errb := runArgs(t, "policy", "--dir", dir, "--policy", "/nonexistent/policy.json")
	if code != 1 || !strings.Contains(errb, "policy") {
		t.Errorf("missing policy: code=%d err=%q", code, errb)
	}
}

// TestFleetCheck verifies fusaops fleet runs and produces a report.
//
//fusa:test REQ-FO-CLI023
func TestFleetCheck(t *testing.T) {
	dir := goProject(t)
	cfgPath := filepath.Join(dir, "fleet.json")
	cfgData, err := json.Marshal(map[string]any{
		"project": "testfleet",
		"repos":   []map[string]string{{"name": "svc", "dir": dir}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cfgPath, cfgData, 0o644); err != nil {
		t.Fatal(err)
	}
	code, out, errb := runArgs(t, "fleet", "--config", cfgPath, "--format", "text")
	if code != 0 {
		t.Fatalf("fleet: code=%d err=%q", code, errb)
	}
	if !strings.Contains(out, "Fleet:") {
		t.Errorf("fleet output missing Fleet header: %q", out)
	}
	if !strings.Contains(out, "svc") {
		t.Errorf("fleet output missing repo name: %q", out)
	}
}

// TestFleetMissingConfig verifies a missing config produces exit 1.
//
//fusa:test REQ-FO-CLI023
func TestFleetMissingConfig(t *testing.T) {
	code, _, errb := runArgs(t, "fleet", "--config", "/nonexistent/fleet.json")
	if code != 1 || !strings.Contains(errb, "fleet") {
		t.Errorf("missing config: code=%d err=%q", code, errb)
	}
}

// TestServeBadAuthFormat verifies --auth without colon returns exit 1.
//
//fusa:test REQ-FO-CLI025
func TestServeBadAuthFormat(t *testing.T) {
	code, _, errb := runArgs(t, "serve", "--auth", "nocolon")
	if code != 1 || !strings.Contains(errb, "user:pass") {
		t.Errorf("bad auth format: code=%d err=%q", code, errb)
	}
}

// TestServeTLSMissingKey verifies --tls-cert without --tls-key returns exit 1.
//
//fusa:test REQ-FO-CLI027
func TestServeTLSMissingKey(t *testing.T) {
	dir := t.TempDir()
	code, _, errb := runArgs(t, "serve", "--dir", dir, "--tls-cert", "cert.pem")
	if code != 1 || !strings.Contains(errb, "tls-key") {
		t.Errorf("missing tls-key: code=%d err=%q", code, errb)
	}
}

// TestServeAuthROBadFormat verifies --auth-ro without colon returns exit 1.
//
//fusa:test REQ-FO-CLI028
func TestServeAuthROBadFormat(t *testing.T) {
	code, _, errb := runArgs(t, "serve", "--auth-ro", "nocolon")
	if code != 1 || !strings.Contains(errb, "user:pass") {
		t.Errorf("bad auth-ro format: code=%d err=%q", code, errb)
	}
}

// TestServeMissingProjectsConfig verifies a missing projects.json returns exit 1.
//
//fusa:test REQ-FO-CLI030
func TestServeMissingProjectsConfig(t *testing.T) {
	code, _, errb := runArgs(t, "serve", "--projects", "/nonexistent/projects.json")
	if code != 1 || !strings.Contains(errb, "projects") {
		t.Errorf("missing projects config: code=%d err=%q", code, errb)
	}
}

// TestCheckSuppressFileMissing verifies --suppress-file with missing file returns exit 1.
//
//fusa:test REQ-FO-CLI033
func TestCheckSuppressFileMissing(t *testing.T) {
	dir := t.TempDir()
	// Write a Go file so a language is detected.
	_ = os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main"), 0o644)
	code, _, errb := runArgs(t, "check", "--dir", dir, "--suppress-file", "/nonexistent/suppress.json")
	if code != 1 || !strings.Contains(errb, "suppress") {
		t.Errorf("missing suppress file: code=%d err=%q", code, errb)
	}
}

// TestServeBadRefreshInterval verifies --refresh-interval with invalid duration returns exit 1.
//
//fusa:test REQ-FO-CLI032
func TestServeBadRefreshInterval(t *testing.T) {
	code, _, errb := runArgs(t, "serve", "--refresh-interval", "not-a-duration")
	if code != 1 || !strings.Contains(errb, "refresh-interval") {
		t.Errorf("bad refresh-interval: code=%d err=%q", code, errb)
	}
}

// TestServeZeroRefreshInterval verifies --refresh-interval 0 returns exit 1.
//
//fusa:test REQ-FO-CLI032
func TestServeZeroRefreshInterval(t *testing.T) {
	code, _, errb := runArgs(t, "serve", "--refresh-interval", "0")
	if code != 1 || !strings.Contains(errb, "refresh-interval") {
		t.Errorf("zero refresh-interval: code=%d err=%q", code, errb)
	}
}

// TestCheckOutputFlag verifies fusaops check --output writes report to a file.
//
//fusa:test REQ-FO-CLI037
func TestCheckOutputFlag(t *testing.T) {
	dir := goProject(t)
	out := filepath.Join(t.TempDir(), "report.txt")
	code, stdout, errb := runArgs(t, "check", "--dir", dir, "--output", out)
	if code != 0 {
		t.Fatalf("check --output: code=%d err=%q", code, errb)
	}
	if !strings.Contains(stdout, "report.txt") {
		t.Errorf("expected confirmation in stdout: %q", stdout)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("output file not created: %v", err)
	}
	if len(data) == 0 {
		t.Error("output file is empty")
	}
}

// TestCheckMarkdownFormat verifies fusaops check --format markdown produces Markdown output.
//
//fusa:test REQ-FO-CLI036
func TestCheckMarkdownFormat(t *testing.T) {
	dir := goProject(t)
	code, out, errb := runArgs(t, "check", "--dir", dir, "--format", "markdown")
	if code != 0 {
		t.Fatalf("check --format markdown: code=%d err=%q", code, errb)
	}
	if !strings.HasPrefix(out, "# FuSaOps Report") {
		t.Errorf("markdown output missing heading: %.100s", out)
	}
}

// TestCheckCSVFormat verifies fusaops check --format csv produces CSV output.
//
//fusa:test REQ-FO-CLI035
func TestCheckCSVFormat(t *testing.T) {
	dir := goProject(t)
	code, out, errb := runArgs(t, "check", "--dir", dir, "--format", "csv")
	if code != 0 {
		t.Fatalf("check --format csv: code=%d err=%q", code, errb)
	}
	if !strings.Contains(out, "language") || !strings.Contains(out, "severity") {
		t.Errorf("csv output missing header columns: %.200s", out)
	}
}

// TestCheckJUnitFormat verifies fusaops check --format junit produces XML output.
//
//fusa:test REQ-FO-CLI034
func TestCheckJUnitFormat(t *testing.T) {
	dir := goProject(t)
	code, out, errb := runArgs(t, "check", "--dir", dir, "--format", "junit")
	if code != 0 {
		t.Fatalf("check --format junit: code=%d err=%q", code, errb)
	}
	if !strings.Contains(out, "<?xml") || !strings.Contains(out, "<testsuites") {
		t.Errorf("junit output missing XML elements: %.200s", out)
	}
}

// TestCapabilities verifies fusaops capabilities emits a JSON discovery document.
//
//fusa:test REQ-FO-CLI054
func TestCapabilities(t *testing.T) {
	code, out, errb := runArgs(t, "capabilities")
	if code != 0 {
		t.Fatalf("capabilities: code=%d err=%q", code, errb)
	}
	for _, want := range []string{`"tool"`, `"version"`, `"specVersion"`, `"commands"`, `"formats"`, "fusaops"} {
		if !strings.Contains(out, want) {
			t.Errorf("capabilities missing %q: %.200s", want, out)
		}
	}
}

//fusa:test REQ-FO-CLI054
func TestCapabilitiesInvalidFormat(t *testing.T) {
	code, _, errb := runArgs(t, "capabilities", "--format", "text")
	if code != 2 || !strings.Contains(errb, "only json") {
		t.Errorf("invalid format: code=%d err=%q", code, errb)
	}
}

// TestCoverageText verifies fusaops coverage produces a DO-178C text report.
//
//fusa:test REQ-FO-CLI051
func TestCoverageText(t *testing.T) {
	dir := t.TempDir()
	profile := "mode: set\npkg/foo.go:1.10,3.2 2 1\npkg/foo.go:5.10,7.2 2 0\n"
	path := filepath.Join(dir, "coverage.out")
	if err := os.WriteFile(path, []byte(profile), 0o600); err != nil {
		t.Fatal(err)
	}
	code, out, errb := runArgs(t, "coverage", "--format", "text", "--dal", "DAL-B", path)
	if code != 0 {
		t.Fatalf("coverage: code=%d err=%q", code, errb)
	}
	if !strings.Contains(out, "DO-178C") {
		t.Errorf("coverage text missing DO-178C header: %q", out)
	}
	if !strings.Contains(out, "DAL-B") {
		t.Errorf("coverage text missing DAL-B: %q", out)
	}
}

//fusa:test REQ-FO-CLI051
func TestCoverageJSON(t *testing.T) {
	dir := t.TempDir()
	profile := "mode: set\npkg/foo.go:1.10,3.2 2 1\n"
	path := filepath.Join(dir, "coverage.out")
	if err := os.WriteFile(path, []byte(profile), 0o600); err != nil {
		t.Fatal(err)
	}
	code, out, errb := runArgs(t, "coverage", "--format", "json", path)
	if code != 0 {
		t.Fatalf("coverage json: code=%d err=%q", code, errb)
	}
	if !strings.Contains(out, `"stmtTotal"`) {
		t.Errorf("coverage json missing stmtTotal: %q", out)
	}
}

//fusa:test REQ-FO-CLI051
func TestCoverageMissingProfile(t *testing.T) {
	code, _, errb := runArgs(t, "coverage", "/nonexistent/coverage.out")
	if code != 1 || !strings.Contains(errb, "coverage") {
		t.Errorf("missing profile: code=%d err=%q", code, errb)
	}
}

//fusa:test REQ-FO-CLI051
func TestCoverageInvalidDAL(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "coverage.out")
	_ = os.WriteFile(path, []byte("mode: set\n"), 0o600)
	code, _, errb := runArgs(t, "coverage", "--dal", "DAL-Z", path)
	if code != 2 || !strings.Contains(errb, "dal") {
		t.Errorf("invalid dal: code=%d err=%q", code, errb)
	}
}

//fusa:test REQ-FO-CLI051
func TestCoverageOutputFlag(t *testing.T) {
	dir := t.TempDir()
	profile := "mode: set\npkg/foo.go:1.10,3.2 2 1\n"
	inPath := filepath.Join(dir, "coverage.out")
	outPath := filepath.Join(dir, "report.txt")
	if err := os.WriteFile(inPath, []byte(profile), 0o600); err != nil {
		t.Fatal(err)
	}
	code, _, errb := runArgs(t, "coverage", "--output", outPath, inPath)
	if code != 0 {
		t.Fatalf("coverage --output: code=%d err=%q", code, errb)
	}
	if _, err := os.Stat(outPath); err != nil {
		t.Errorf("output file not created: %v", err)
	}
}

// TestReqShow verifies fusaops req shows requirements from .fusa-reqs.json.
//
//fusa:test REQ-FO-CLI052
func TestReqShow(t *testing.T) {
	dir := t.TempDir()
	data := `{"requirements":[{"id":"REQ-A","title":"Requirement Alpha","priority":"MUST"}]}`
	if err := os.WriteFile(filepath.Join(dir, ".fusa-reqs.json"), []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	code, out, errb := runArgs(t, "req", "--dir", dir)
	if code != 0 {
		t.Fatalf("req show: code=%d err=%q", code, errb)
	}
	if !strings.Contains(out, "REQ-A") || !strings.Contains(out, "Requirement Alpha") {
		t.Errorf("req show missing content: %q", out)
	}
}

//fusa:test REQ-FO-CLI052
func TestReqShowFilterID(t *testing.T) {
	dir := t.TempDir()
	data := `{"requirements":[{"id":"REQ-A","title":"Alpha"},{"id":"REQ-B","title":"Beta"}]}`
	if err := os.WriteFile(filepath.Join(dir, ".fusa-reqs.json"), []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	code, out, _ := runArgs(t, "req", "--dir", dir, "REQ-A")
	if code != 0 {
		t.Fatalf("req show filter: code=%d", code)
	}
	if !strings.Contains(out, "REQ-A") {
		t.Errorf("missing REQ-A in output")
	}
	if strings.Contains(out, "REQ-B") {
		t.Errorf("REQ-B should not appear when filtering for REQ-A")
	}
}

//fusa:test REQ-FO-CLI052
func TestReqShowMissingID(t *testing.T) {
	dir := t.TempDir()
	data := `{"requirements":[{"id":"REQ-A","title":"Alpha"}]}`
	if err := os.WriteFile(filepath.Join(dir, ".fusa-reqs.json"), []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	code, _, errb := runArgs(t, "req", "--dir", dir, "REQ-NONEXISTENT")
	if code != 1 || !strings.Contains(errb, "not found") {
		t.Errorf("missing id: code=%d err=%q", code, errb)
	}
}

//fusa:test REQ-FO-CLI052
func TestReqImportCSV(t *testing.T) {
	dir := t.TempDir()
	csvData := "id,title,standard\nREQ-1,First req,ISO 26262\nREQ-2,Second req,DO-178C\n"
	csvPath := filepath.Join(dir, "reqs.csv")
	if err := os.WriteFile(csvPath, []byte(csvData), 0o600); err != nil {
		t.Fatal(err)
	}
	code, out, errb := runArgs(t, "req", "--dir", dir, "import", "--file", csvPath)
	if code != 0 {
		t.Fatalf("req import: code=%d err=%q", code, errb)
	}
	if !strings.Contains(out, "Imported 2") {
		t.Errorf("import output: %q", out)
	}
	// Verify registry was updated
	code2, out2, _ := runArgs(t, "req", "--dir", dir)
	if code2 != 0 || !strings.Contains(out2, "REQ-1") {
		t.Errorf("after import, show failed: code=%d out=%q", code2, out2)
	}
}

//fusa:test REQ-FO-CLI052
func TestReqImportMissingFile(t *testing.T) {
	code, _, errb := runArgs(t, "req", "import", "--file", "/nonexistent/reqs.csv")
	if code != 1 || !strings.Contains(errb, "open") {
		t.Errorf("missing file: code=%d err=%q", code, errb)
	}
}

//fusa:test REQ-FO-CLI052
func TestReqImportMissingFileFlag(t *testing.T) {
	code, _, errb := runArgs(t, "req", "import")
	if code != 2 || !strings.Contains(errb, "--file") {
		t.Errorf("missing --file: code=%d err=%q", code, errb)
	}
}

//fusa:test REQ-FO-CLI052
func TestReqExportCSV(t *testing.T) {
	dir := t.TempDir()
	data := `{"requirements":[{"id":"REQ-A","title":"Alpha","standard":"ISO 26262"}]}`
	if err := os.WriteFile(filepath.Join(dir, ".fusa-reqs.json"), []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	code, out, errb := runArgs(t, "req", "--dir", dir, "export", "--format", "csv")
	if code != 0 {
		t.Fatalf("req export csv: code=%d err=%q", code, errb)
	}
	if !strings.Contains(out, "REQ-A") || !strings.Contains(out, "id") {
		t.Errorf("csv export missing content: %q", out)
	}
}

//fusa:test REQ-FO-CLI052
func TestReqExportDOORS(t *testing.T) {
	dir := t.TempDir()
	data := `{"requirements":[{"id":"REQ-A","title":"Alpha","text":"Some text"}]}`
	if err := os.WriteFile(filepath.Join(dir, ".fusa-reqs.json"), []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	code, out, errb := runArgs(t, "req", "--dir", dir, "export", "--format", "doors")
	if code != 0 {
		t.Fatalf("req export doors: code=%d err=%q", code, errb)
	}
	if !strings.Contains(out, "REQ-IF") || !strings.Contains(out, "REQ-A") {
		t.Errorf("doors export missing content: %q", out)
	}
}

//fusa:test REQ-FO-CLI052
func TestReqExportMissingRegistry(t *testing.T) {
	code, _, errb := runArgs(t, "req", "--dir", t.TempDir(), "export")
	if code != 1 || !strings.Contains(errb, "req") {
		t.Errorf("missing registry: code=%d err=%q", code, errb)
	}
}

func TestSplitCSV(t *testing.T) {
	got := splitCSV("a,b,,c")
	if len(got) != 3 || got[0] != "a" || got[2] != "c" {
		t.Errorf("splitCSV: %v", got)
	}
	if len(splitCSV("")) != 0 {
		t.Error("splitCSV empty should be empty")
	}
}

//fusa:test REQ-FO-CLI055
func TestMetricsRecord(t *testing.T) {
	dir := t.TempDir()
	code, out, errb := runArgs(t, "metrics", "--dir", dir, "record")
	if code != 0 {
		t.Fatalf("metrics record: code=%d err=%q", code, errb)
	}
	if !strings.Contains(out, "Metrics recorded") {
		t.Errorf("record output missing confirmation: %q", out)
	}
	if !strings.Contains(out, ".fusaops-metrics.json") {
		t.Errorf("record output missing file path: %q", out)
	}
}

//fusa:test REQ-FO-CLI055
func TestMetricsShowEmpty(t *testing.T) {
	code, out, errb := runArgs(t, "metrics", "--dir", t.TempDir(), "show")
	if code != 0 {
		t.Fatalf("metrics show empty: code=%d err=%q", code, errb)
	}
	if !strings.Contains(out, "No snapshots") {
		t.Errorf("expected no-snapshots message: %q", out)
	}
}

//fusa:test REQ-FO-CLI055
func TestMetricsShowJSON(t *testing.T) {
	dir := t.TempDir()
	// Record first so there is data.
	if code, _, errb := runArgs(t, "metrics", "--dir", dir, "record"); code != 0 {
		t.Fatalf("metrics record: code=%d err=%q", code, errb)
	}
	code, out, errb := runArgs(t, "metrics", "--dir", dir, "show", "--format", "json")
	if code != 0 {
		t.Fatalf("metrics show json: code=%d err=%q", code, errb)
	}
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(out), &m); err != nil {
		t.Fatalf("invalid JSON: %v\nout=%q", err, out)
	}
}

//fusa:test REQ-FO-CLI055
func TestMetricsShowOutput(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "metrics.txt")
	code, _, errb := runArgs(t, "metrics", "--dir", dir, "show", "--output", out)
	if code != 0 {
		t.Fatalf("metrics show --output: code=%d err=%q", code, errb)
	}
	if _, err := os.Stat(out); err != nil {
		t.Errorf("output file not created: %v", err)
	}
}

//fusa:test REQ-FO-CLI055
func TestMetricsUnknownSubcommand(t *testing.T) {
	code, _, errb := runArgs(t, "metrics", "bogus")
	if code != 2 || !strings.Contains(errb, "unknown subcommand") {
		t.Errorf("unknown subcommand: code=%d err=%q", code, errb)
	}
}

//fusa:test REQ-FO-CLI055
func TestMetricsNoSubcommand(t *testing.T) {
	code, _, errb := runArgs(t, "metrics")
	if code != 2 || !strings.Contains(errb, "Usage") {
		t.Errorf("no subcommand: code=%d err=%q", code, errb)
	}
}

//fusa:test REQ-FO-CLI056
func TestBadgeFromFile(t *testing.T) {
	dir := t.TempDir()
	reportJSON := `{"generatedAt":"2026-06-13T00:00:00Z","root":".","components":[],"summary":{"total":0,"errors":0,"warnings":0,"infos":0}}`
	reportFile := filepath.Join(dir, "report.json")
	if err := os.WriteFile(reportFile, []byte(reportJSON), 0o600); err != nil {
		t.Fatal(err)
	}
	code, out, errb := runArgs(t, "badge", reportFile)
	if code != 0 {
		t.Fatalf("badge: code=%d err=%q", code, errb)
	}
	if !strings.Contains(out, "<svg") || !strings.Contains(out, "fusaops") {
		t.Errorf("badge output missing SVG content: %q", out[:min(len(out), 200)])
	}
	if !strings.Contains(out, "passing") {
		t.Errorf("pass badge missing 'passing': %q", out)
	}
}

//fusa:test REQ-FO-CLI056
func TestBadgeFailReport(t *testing.T) {
	dir := t.TempDir()
	reportJSON := `{"generatedAt":"2026-06-13T00:00:00Z","root":".","components":[],"summary":{"total":2,"errors":2,"warnings":0,"infos":0}}`
	reportFile := filepath.Join(dir, "report.json")
	if err := os.WriteFile(reportFile, []byte(reportJSON), 0o600); err != nil {
		t.Fatal(err)
	}
	code, out, errb := runArgs(t, "badge", reportFile)
	if code != 0 {
		t.Fatalf("badge fail: code=%d err=%q", code, errb)
	}
	if !strings.Contains(out, "failing") {
		t.Errorf("fail badge missing 'failing': %q", out)
	}
}

//fusa:test REQ-FO-CLI056
func TestBadgeOutputFile(t *testing.T) {
	dir := t.TempDir()
	reportJSON := `{"generatedAt":"2026-06-13T00:00:00Z","root":".","components":[],"summary":{"total":0,"errors":0,"warnings":0,"infos":0}}`
	reportFile := filepath.Join(dir, "report.json")
	if err := os.WriteFile(reportFile, []byte(reportJSON), 0o600); err != nil {
		t.Fatal(err)
	}
	outFile := filepath.Join(dir, "badge.svg")
	code, _, errb := runArgs(t, "badge", "--output", outFile, reportFile)
	if code != 0 {
		t.Fatalf("badge --output: code=%d err=%q", code, errb)
	}
	if _, err := os.Stat(outFile); err != nil {
		t.Errorf("output file not created: %v", err)
	}
}

//fusa:test REQ-FO-CLI056
func TestBadgeMissingFile(t *testing.T) {
	code, _, errb := runArgs(t, "badge", "/nonexistent/report.json")
	if code != 1 || !strings.Contains(errb, "badge") {
		t.Errorf("missing file: code=%d err=%q", code, errb)
	}
}

//fusa:test REQ-FO-CLI056
func TestBadgeTooManyArgs(t *testing.T) {
	code, _, _ := runArgs(t, "badge", "a.json", "b.json")
	if code != 2 {
		t.Errorf("too many args: code=%d", code)
	}
}

//fusa:test REQ-FO-CLI057
func TestSLSAText(t *testing.T) {
	dir := t.TempDir()
	code, out, errb := runArgs(t, "slsa", "--dir", dir, "--format", "text")
	// Non-zero exit is expected when there are gaps (empty dir has many gaps).
	if code > 1 {
		t.Fatalf("slsa text: code=%d err=%q", code, errb)
	}
	if !strings.Contains(out, "SLSA") {
		t.Errorf("slsa text output missing SLSA header: %q", out[:min(len(out), 300)])
	}
}

//fusa:test REQ-FO-CLI057
func TestSLSAJSON(t *testing.T) {
	dir := t.TempDir()
	code, out, errb := runArgs(t, "slsa", "--dir", dir, "--format", "json")
	if code > 1 {
		t.Fatalf("slsa json: code=%d err=%q", code, errb)
	}
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(out), &m); err != nil {
		t.Fatalf("invalid JSON: %v\nout=%q", err, out)
	}
	if _, ok := m["objectives"]; !ok {
		t.Errorf("JSON missing objectives field: %q", out)
	}
}

//fusa:test REQ-FO-CLI057
func TestSLSAOutputFile(t *testing.T) {
	dir := t.TempDir()
	outFile := filepath.Join(dir, "slsa-report.txt")
	runArgs(t, "slsa", "--dir", dir, "--output", outFile)
	if _, err := os.Stat(outFile); err != nil {
		t.Errorf("output file not created: %v", err)
	}
}

//fusa:test REQ-FO-CLI057
func TestSLSAInvalidLevel(t *testing.T) {
	code, _, errb := runArgs(t, "slsa", "--level", "L9")
	if code != 2 || !strings.Contains(errb, "level") {
		t.Errorf("invalid level: code=%d err=%q", code, errb)
	}
}

//fusa:test REQ-FO-CLI057
func TestSLSAInvalidFormat(t *testing.T) {
	code, _, errb := runArgs(t, "slsa", "--format", "xml")
	if code != 1 || !strings.Contains(errb, "unsupported") {
		t.Errorf("invalid format: code=%d err=%q", code, errb)
	}
}

//fusa:test REQ-FO-CLI058
func TestHooksShow(t *testing.T) {
	code, out, errb := runArgs(t, "hooks", "show")
	if code != 0 {
		t.Fatalf("hooks show: code=%d err=%q", code, errb)
	}
	if !strings.Contains(out, "fusaops check") {
		t.Errorf("hooks show missing fusaops check: %q", out)
	}
}

//fusa:test REQ-FO-CLI058
func TestHooksInstallRemove(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".git", "hooks"), 0o700); err != nil {
		t.Fatal(err)
	}
	code, out, errb := runArgs(t, "hooks", "--dir", dir, "install")
	if code != 0 {
		t.Fatalf("hooks install: code=%d err=%q", code, errb)
	}
	if !strings.Contains(out, "installed") {
		t.Errorf("install output missing confirmation: %q", out)
	}
	hookPath := filepath.Join(dir, ".git", "hooks", "pre-commit")
	if _, err := os.Stat(hookPath); err != nil {
		t.Fatalf("pre-commit hook not created: %v", err)
	}
	// Second install should fail (already exists).
	code2, _, errb2 := runArgs(t, "hooks", "--dir", dir, "install")
	if code2 != 1 || !strings.Contains(errb2, "already exists") {
		t.Errorf("double install: code=%d err=%q", code2, errb2)
	}
	// Remove.
	code3, out3, errb3 := runArgs(t, "hooks", "--dir", dir, "remove")
	if code3 != 0 {
		t.Fatalf("hooks remove: code=%d err=%q", code3, errb3)
	}
	if !strings.Contains(out3, "removed") {
		t.Errorf("remove output missing confirmation: %q", out3)
	}
	if _, err := os.Stat(hookPath); err == nil {
		t.Error("pre-commit hook still exists after remove")
	}
}

//fusa:test REQ-FO-CLI058
func TestHooksRemoveMissing(t *testing.T) {
	code, _, errb := runArgs(t, "hooks", "--dir", t.TempDir(), "remove")
	if code != 1 || !strings.Contains(errb, "not found") {
		t.Errorf("remove missing: code=%d err=%q", code, errb)
	}
}

//fusa:test REQ-FO-CLI058
func TestHooksUnknownSubcommand(t *testing.T) {
	code, _, errb := runArgs(t, "hooks", "bogus")
	if code != 2 || !strings.Contains(errb, "unknown subcommand") {
		t.Errorf("unknown subcommand: code=%d err=%q", code, errb)
	}
}

//fusa:test REQ-FO-CLI058
func TestHooksNoSubcommand(t *testing.T) {
	code, _, errb := runArgs(t, "hooks")
	if code != 2 || !strings.Contains(errb, "Usage") {
		t.Errorf("no subcommand: code=%d err=%q", code, errb)
	}
}

//fusa:test REQ-FO-CLI059
func TestImpactText(t *testing.T) {
	// No .git in TempDir — git diff fails, but report still renders.
	dir := t.TempDir()
	code, out, errb := runArgs(t, "impact", "--dir", dir, "--format", "text")
	if code != 0 {
		t.Fatalf("impact text: code=%d err=%q", code, errb)
	}
	if !strings.Contains(out, "FuSaOps Impact") {
		t.Errorf("impact text missing header: %q", out[:min(len(out), 200)])
	}
}

//fusa:test REQ-FO-CLI059
func TestImpactJSON(t *testing.T) {
	dir := t.TempDir()
	code, out, errb := runArgs(t, "impact", "--dir", dir, "--format", "json")
	if code != 0 {
		t.Fatalf("impact json: code=%d err=%q", code, errb)
	}
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(out), &m); err != nil {
		t.Fatalf("invalid JSON: %v\nout=%q", err, out)
	}
}

//fusa:test REQ-FO-CLI059
func TestImpactOutputFile(t *testing.T) {
	dir := t.TempDir()
	outFile := filepath.Join(dir, "impact.txt")
	code, _, errb := runArgs(t, "impact", "--dir", dir, "--output", outFile)
	if code != 0 {
		t.Fatalf("impact --output: code=%d err=%q", code, errb)
	}
	if _, err := os.Stat(outFile); err != nil {
		t.Errorf("output file not created: %v", err)
	}
}

//fusa:test REQ-FO-CLI060
func TestDispositionListEmpty(t *testing.T) {
	code, out, errb := runArgs(t, "disposition", "--dir", t.TempDir(), "list")
	if code != 0 {
		t.Fatalf("disposition list empty: code=%d err=%q", code, errb)
	}
	if !strings.Contains(out, "No disposition") {
		t.Errorf("empty list missing message: %q", out)
	}
}

//fusa:test REQ-FO-CLI060
func TestDispositionAddAndShow(t *testing.T) {
	dir := t.TempDir()
	code, out, errb := runArgs(t, "disposition", "--dir", dir, "add",
		"--rule", "RULE001", "--reviewer", "alice",
		"--rationale", "accepted by design", "--action", "accept")
	if code != 0 {
		t.Fatalf("disposition add: code=%d err=%q", code, errb)
	}
	if !strings.Contains(out, "RULE001") {
		t.Errorf("add confirmation missing rule: %q", out)
	}
	// Show the entry.
	code2, out2, errb2 := runArgs(t, "disposition", "--dir", dir, "show", "--rule", "RULE001")
	if code2 != 0 {
		t.Fatalf("disposition show: code=%d err=%q", code2, errb2)
	}
	if !strings.Contains(out2, "alice") || !strings.Contains(out2, "accept") {
		t.Errorf("show output missing expected content: %q", out2)
	}
}

//fusa:test REQ-FO-CLI060
func TestDispositionShowMissing(t *testing.T) {
	code, _, errb := runArgs(t, "disposition", "--dir", t.TempDir(), "show", "--rule", "NOSUCHRULE")
	if code != 1 || !strings.Contains(errb, "no disposition") {
		t.Errorf("show missing: code=%d err=%q", code, errb)
	}
}

//fusa:test REQ-FO-CLI060
func TestDispositionAddMissingFlags(t *testing.T) {
	code, _, errb := runArgs(t, "disposition", "add", "--rule", "R1")
	if code != 2 || !strings.Contains(errb, "reviewer") {
		t.Errorf("missing reviewer: code=%d err=%q", code, errb)
	}
}

//fusa:test REQ-FO-CLI060
func TestDispositionAddInvalidAction(t *testing.T) {
	dir := t.TempDir()
	code, _, errb := runArgs(t, "disposition", "--dir", dir, "add",
		"--rule", "R1", "--reviewer", "alice", "--rationale", "x", "--action", "ignore")
	if code != 2 || !strings.Contains(errb, "action") {
		t.Errorf("invalid action: code=%d err=%q", code, errb)
	}
}

//fusa:test REQ-FO-CLI060
func TestDispositionUnknownSubcommand(t *testing.T) {
	code, _, errb := runArgs(t, "disposition", "bogus")
	if code != 2 || !strings.Contains(errb, "unknown subcommand") {
		t.Errorf("unknown subcommand: code=%d err=%q", code, errb)
	}
}

//fusa:test REQ-FO-CLI060
func TestDispositionNoSubcommand(t *testing.T) {
	code, _, errb := runArgs(t, "disposition")
	if code != 2 || !strings.Contains(errb, "Usage") {
		t.Errorf("no subcommand: code=%d err=%q", code, errb)
	}
}

//fusa:test REQ-FO-CLI059
func TestImpactInvalidFormat(t *testing.T) {
	code, _, errb := runArgs(t, "impact", "--format", "xml")
	if code != 1 || !strings.Contains(errb, "unsupported") {
		t.Errorf("invalid format: code=%d err=%q", code, errb)
	}
}

//fusa:test REQ-FO-CLI057
func TestSLSAL1(t *testing.T) {
	dir := t.TempDir()
	// Write go.mod and .git so L1 has more passes.
	if err := os.MkdirAll(filepath.Join(dir, ".git"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test"), 0o600); err != nil {
		t.Fatal(err)
	}
	code, out, errb := runArgs(t, "slsa", "--dir", dir, "--level", "L1", "--format", "text")
	if code > 1 {
		t.Fatalf("slsa L1: code=%d err=%q", code, errb)
	}
	if !strings.Contains(out, "PASS") {
		t.Errorf("L1 with .git+go.mod should have at least one PASS: %q", out)
	}
}

//fusa:test REQ-FO-CLI061
func TestPRNoSubcommand(t *testing.T) {
	code, _, errb := runArgs(t, "pr")
	if code != 2 || !strings.Contains(errb, "Usage") {
		t.Errorf("no subcommand: code=%d err=%q", code, errb)
	}
}

//fusa:test REQ-FO-CLI061
func TestPRUnknownSubcommand(t *testing.T) {
	code, _, errb := runArgs(t, "pr", "bogus")
	if code != 2 || !strings.Contains(errb, "unknown subcommand") {
		t.Errorf("unknown subcommand: code=%d err=%q", code, errb)
	}
}

//fusa:test REQ-FO-CLI061
func TestPRInitAddListClose(t *testing.T) {
	dir := t.TempDir()

	// init
	code, out, errb := runArgs(t, "pr", "--dir", dir, "init")
	if code != 0 {
		t.Fatalf("pr init: code=%d err=%q", code, errb)
	}
	if !strings.Contains(out, "created") {
		t.Errorf("pr init: expected 'created' in %q", out)
	}

	// add
	code, out, errb = runArgs(t, "pr", "--dir", dir, "add",
		"--id", "PR-001", "--title", "Stack overflow in ISR",
		"--severity", "critical", "--phase", "integration")
	if code != 0 {
		t.Fatalf("pr add: code=%d err=%q", code, errb)
	}
	if !strings.Contains(out, "PR-001") {
		t.Errorf("pr add: expected PR-001 in %q", out)
	}

	// list text
	code, out, errb = runArgs(t, "pr", "--dir", dir, "list")
	if code != 0 {
		t.Fatalf("pr list: code=%d err=%q", code, errb)
	}
	if !strings.Contains(out, "Stack overflow") {
		t.Errorf("pr list: missing title in %q", out)
	}

	// close
	code, out, errb = runArgs(t, "pr", "--dir", dir, "close",
		"--id", "PR-001", "--resolution", "fixed in commit abc")
	if code != 0 {
		t.Fatalf("pr close: code=%d err=%q", code, errb)
	}
	if !strings.Contains(out, "PR-001") {
		t.Errorf("pr close: missing ID in %q", out)
	}
}

//fusa:test REQ-FO-CLI061
func TestPRInitAlreadyExists(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".fusaops-problems.json"),
		[]byte(`{"project":"p","reports":[]}`), 0o640); err != nil {
		t.Fatal(err)
	}
	code, _, errb := runArgs(t, "pr", "--dir", dir, "init")
	if code != 2 || !strings.Contains(errb, "already exists") {
		t.Errorf("double init: code=%d err=%q", code, errb)
	}
}

//fusa:test REQ-FO-CLI061
func TestPRAddMissingFlags(t *testing.T) {
	dir := t.TempDir()
	code, _, errb := runArgs(t, "pr", "--dir", dir, "add", "--title", "T")
	if code != 2 || !strings.Contains(errb, "--id") {
		t.Errorf("missing --id: code=%d err=%q", code, errb)
	}
	code, _, errb = runArgs(t, "pr", "--dir", dir, "add", "--id", "PR-001")
	if code != 2 || !strings.Contains(errb, "--title") {
		t.Errorf("missing --title: code=%d err=%q", code, errb)
	}
}

//fusa:test REQ-FO-CLI061
func TestPRCloseNotFound(t *testing.T) {
	dir := t.TempDir()
	code, _, errb := runArgs(t, "pr", "--dir", dir, "close", "--id", "PR-999")
	if code != 1 || !strings.Contains(errb, "not found") {
		t.Errorf("close not found: code=%d err=%q", code, errb)
	}
}

//fusa:test REQ-FO-CLI061
func TestPRCloseMissingID(t *testing.T) {
	dir := t.TempDir()
	code, _, errb := runArgs(t, "pr", "--dir", dir, "close")
	if code != 2 || !strings.Contains(errb, "--id") {
		t.Errorf("close missing --id: code=%d err=%q", code, errb)
	}
}

//fusa:test REQ-FO-CLI061
func TestPRListJSON(t *testing.T) {
	dir := t.TempDir()
	runArgs(t, "pr", "--dir", dir, "add", "--id", "PR-001", "--title", "Test")
	code, out, errb := runArgs(t, "pr", "--dir", dir, "list", "--format", "json")
	if code != 0 {
		t.Fatalf("pr list json: code=%d err=%q", code, errb)
	}
	if !strings.Contains(out, `"reports"`) {
		t.Errorf("json output missing reports key: %q", out)
	}
}
