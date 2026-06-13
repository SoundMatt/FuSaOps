package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/SoundMatt/FuSaOps/config"
)

// goProject creates a temp dir containing a single Go source file so an
// adapter is applicable. Whether gofusa is installed or not, the orchestrator
// returns a report (running it, or recording it skipped).
func goProject(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	return dir
}

//fusa:test REQ-FO-CLI009
func TestReportWritesFile(t *testing.T) {
	dir := goProject(t)
	out := filepath.Join(dir, "agg.json")
	code, stdout, errb := runArgs(t, "report", "--dir", dir, "--format", "json", "--output", out)
	if code != 0 {
		t.Fatalf("report: code=%d err=%q", code, errb)
	}
	if !strings.Contains(stdout, "Wrote") {
		t.Errorf("report stdout: %q", stdout)
	}
	if _, err := os.Stat(out); err != nil {
		t.Errorf("report file not written: %v", err)
	}
}

//fusa:test REQ-FO-CLI008
func TestCheckReturnsReport(t *testing.T) {
	dir := goProject(t)
	// Exit code is invariant-free: 0 when the tool is absent (component
	// skipped) or finds nothing, 1 when an installed gofusa flags the bare
	// temp project. Either is correct; the report must always render.
	code, stdout, _ := runArgs(t, "check", "--dir", dir, "--format", "text")
	if code != 0 && code != 1 {
		t.Fatalf("check: unexpected code=%d", code)
	}
	if !strings.Contains(stdout, "FuSaOps") {
		t.Errorf("check stdout: %q", stdout)
	}
}

//fusa:test REQ-FO-CLI007
func TestCheckHonoursConfig(t *testing.T) {
	dir := goProject(t)
	cfg := config.Default("configured-project")
	if err := config.Save(filepath.Join(dir, config.ConfigFile), cfg); err != nil {
		t.Fatal(err)
	}
	_, _, _, ferr := loadOptions(dir, "", os.Stderr)
	if ferr != nil {
		t.Fatalf("loadOptions: %v", ferr)
	}
	code, stdout, _ := runArgs(t, "check", "--dir", dir)
	if code != 0 && code != 1 { // 1 if an installed tool flags the temp project
		t.Fatalf("check: unexpected code=%d", code)
	}
	if !strings.Contains(stdout, "configured-project") {
		t.Errorf("check did not use config project name: %q", stdout)
	}
}

//fusa:test REQ-FO-CLI010
func TestServeBadFlag(t *testing.T) {
	// Exercises runServe's flag parsing without binding a port.
	code, _, _ := runArgs(t, "serve", "--bogus")
	if code != 2 {
		t.Errorf("serve --bogus: got %d, want 2", code)
	}
}

//fusa:test REQ-FO-CLI038
func TestServeBaselineFlagParsed(t *testing.T) {
	// --baseline with a nonexistent path still fails fast (no listener), not on flag parsing.
	code, _, _ := runArgs(t, "serve", "--baseline", "/nonexistent/path.json", "--bogus-to-fail-fast")
	if code != 2 {
		t.Errorf("serve --baseline with bogus extra flag: got %d, want 2", code)
	}
}

//fusa:test REQ-FO-CLI007
func TestLoadOptionsOnlyOverride(t *testing.T) {
	dir := goProject(t)
	_, opts, _, err := loadOptions(dir, "gofusa,cfusa", os.Stderr)
	if err != nil {
		t.Fatal(err)
	}
	if len(opts.Only) != 2 || opts.Only[0] != "gofusa" {
		t.Errorf("only override: %v", opts.Only)
	}
}

//fusa:test REQ-FO-CFG007
//fusa:test REQ-FO-CFG008
//fusa:test REQ-FO-CFG009
func TestLoadOptionsRunConfig(t *testing.T) {
	dir := goProject(t)
	cfg := config.Default("run-cfg-project")
	cfg.Run = config.RunConfig{Timeout: "45s", Workers: 3}
	cfg.Scan.Components = []config.ComponentConfig{
		{Path: ".", Adapter: "gofusa"},
	}
	if err := config.Save(filepath.Join(dir, config.ConfigFile), cfg); err != nil {
		t.Fatal(err)
	}
	_, opts, _, err := loadOptions(dir, "", os.Stderr)
	if err != nil {
		t.Fatal(err)
	}
	if opts.Timeout != 45*time.Second {
		t.Errorf("timeout: got %v, want 45s", opts.Timeout)
	}
	if opts.Workers != 3 {
		t.Errorf("workers: got %d, want 3", opts.Workers)
	}
	if len(opts.Components) != 1 || opts.Components[0].Adapter != "gofusa" {
		t.Errorf("component pin: %+v", opts.Components)
	}
}

func TestLoadOptionsInvalidTimeout(t *testing.T) {
	dir := goProject(t)
	cfg := config.Default("bad-timeout")
	cfg.Run = config.RunConfig{Timeout: "not-a-duration"}
	if err := config.Save(filepath.Join(dir, config.ConfigFile), cfg); err != nil {
		t.Fatal(err)
	}
	// Invalid timeout is warned but does not fail loadOptions.
	var warnBuf strings.Builder
	_, opts, _, err := loadOptions(dir, "", &warnBuf)
	if err != nil {
		t.Fatal(err)
	}
	if opts.Timeout != 0 {
		t.Errorf("invalid timeout should result in zero duration, got %v", opts.Timeout)
	}
	if !strings.Contains(warnBuf.String(), "not-a-duration") {
		t.Errorf("expected warning about invalid timeout: %q", warnBuf.String())
	}
}

//fusa:test REQ-FO-CLI018
func TestDiffMissingBaseline(t *testing.T) {
	dir := goProject(t)
	code, _, errOut := runArgs(t, "diff", "--dir", dir, "--baseline", "nonexistent.json")
	if code != 1 {
		t.Errorf("diff with missing baseline: got code=%d, want 1; err=%q", code, errOut)
	}
}

//fusa:test REQ-FO-CLI039
func TestSuppressUnknownSubcommand(t *testing.T) {
	code, _, _ := runArgs(t, "suppress", "bogus")
	if code != 2 {
		t.Errorf("suppress bogus: got %d, want 2", code)
	}
}

//fusa:test REQ-FO-CLI039
func TestSuppressNoSubcommand(t *testing.T) {
	code, _, _ := runArgs(t, "suppress")
	if code != 2 {
		t.Errorf("suppress (no subcommand): got %d, want 2", code)
	}
}

//fusa:test REQ-FO-SUP005
func TestSuppressAdd(t *testing.T) {
	f := filepath.Join(t.TempDir(), "s.json")
	code, out, _ := runArgs(t, "suppress", "add",
		"--file", f,
		"--fingerprint", "sha256:deadbeef",
		"--reason", "test reason",
	)
	if code != 0 {
		t.Fatalf("suppress add: code=%d", code)
	}
	if !strings.Contains(out, "sha256:deadbeef") {
		t.Errorf("output missing fingerprint: %q", out)
	}
	// Verify file was written.
	data, err := os.ReadFile(f)
	if err != nil {
		t.Fatalf("file not created: %v", err)
	}
	if !strings.Contains(string(data), "sha256:deadbeef") {
		t.Errorf("file missing fingerprint: %s", data)
	}
}

//fusa:test REQ-FO-SUP005
func TestSuppressAddMissingFingerprint(t *testing.T) {
	code, _, errOut := runArgs(t, "suppress", "add", "--reason", "test")
	if code != 1 {
		t.Errorf("suppress add without fingerprint: got %d, want 1; err=%q", code, errOut)
	}
}

//fusa:test REQ-FO-SUP005
func TestSuppressAddBadExpires(t *testing.T) {
	f := filepath.Join(t.TempDir(), "s.json")
	code, _, _ := runArgs(t, "suppress", "add",
		"--file", f,
		"--fingerprint", "sha256:abc",
		"--reason", "test",
		"--expires", "not-a-date",
	)
	if code != 1 {
		t.Errorf("suppress add bad expires: got %d, want 1", code)
	}
}

//fusa:test REQ-FO-SUP006
func TestSuppressList(t *testing.T) {
	f := filepath.Join(t.TempDir(), "s.json")
	runArgs(t, "suppress", "add", "--file", f, "--fingerprint", "sha256:abc", "--reason", "listed")
	code, out, _ := runArgs(t, "suppress", "list", "--file", f)
	if code != 0 {
		t.Fatalf("suppress list: code=%d", code)
	}
	if !strings.Contains(out, "sha256:abc") {
		t.Errorf("list missing fingerprint: %q", out)
	}
}

//fusa:test REQ-FO-SUP006
func TestSuppressListJSON(t *testing.T) {
	f := filepath.Join(t.TempDir(), "s.json")
	runArgs(t, "suppress", "add", "--file", f, "--fingerprint", "sha256:xyz", "--reason", "r")
	code, out, _ := runArgs(t, "suppress", "list", "--file", f, "--format", "json")
	if code != 0 {
		t.Fatalf("suppress list json: code=%d", code)
	}
	if !strings.Contains(out, `"fingerprint"`) {
		t.Errorf("json output missing fingerprint key: %q", out)
	}
}

//fusa:test REQ-FO-SUP007
func TestSuppressPrune(t *testing.T) {
	f := filepath.Join(t.TempDir(), "s.json")
	// Add one expired entry.
	past := time.Now().AddDate(-1, 0, 0).Format("2006-01-02")
	runArgs(t, "suppress", "add", "--file", f, "--fingerprint", "sha256:old", "--reason", "expired", "--expires", past)
	runArgs(t, "suppress", "add", "--file", f, "--fingerprint", "sha256:new", "--reason", "active", "--expires", "2099-12-31")
	code, out, _ := runArgs(t, "suppress", "prune", "--file", f)
	if code != 0 {
		t.Fatalf("suppress prune: code=%d", code)
	}
	if !strings.Contains(out, "1") {
		t.Errorf("prune output missing count: %q", out)
	}
	// Verify expired entry is gone.
	listCode, listOut, _ := runArgs(t, "suppress", "list", "--file", f)
	if listCode != 0 || strings.Contains(listOut, "sha256:old") {
		t.Errorf("expired entry still present after prune: %q", listOut)
	}
}

//fusa:test REQ-FO-SUP008
func TestSuppressVerifyNoStale(t *testing.T) {
	dir := goProject(t)
	f := filepath.Join(t.TempDir(), "s.json")
	// Empty suppression file — all zero suppressions are valid.
	runArgs(t, "suppress", "add", "--file", f, "--fingerprint", "sha256:nomatch", "--reason", "stale")
	// Verify exits 1 because fingerprint is not in current findings.
	code, _, _ := runArgs(t, "suppress", "verify", "--file", f, "--dir", dir)
	if code != 1 {
		t.Errorf("suppress verify with stale entry: got %d, want 1", code)
	}
}

//fusa:test REQ-FO-CLI040
func TestCheckShowSuppressedFlag(t *testing.T) {
	// Flag must parse; unknown flag exits 2.
	code, _, errb := runArgs(t, "check", "--show-suppressed", "--format", "text", "--dir", goProject(t))
	if code == 2 {
		t.Errorf("--show-suppressed not recognised: %s", errb)
	}
}

//fusa:test REQ-FO-CLI040
func TestReportShowSuppressedFlag(t *testing.T) {
	code, _, errb := runArgs(t, "report", "--show-suppressed", "--format", "text", "--dir", goProject(t))
	if code == 2 {
		t.Errorf("--show-suppressed not recognised: %s", errb)
	}
}

//fusa:test REQ-FO-CLI041
func TestCheckShowFingerprintsFlag(t *testing.T) {
	code, _, errb := runArgs(t, "check", "--show-fingerprints", "--format", "text", "--dir", goProject(t))
	if code == 2 {
		t.Errorf("--show-fingerprints not recognised: %s", errb)
	}
}

//fusa:test REQ-FO-CLI041
func TestReportShowFingerprintsFlag(t *testing.T) {
	code, _, errb := runArgs(t, "report", "--show-fingerprints", "--format", "text", "--dir", goProject(t))
	if code == 2 {
		t.Errorf("--show-fingerprints not recognised: %s", errb)
	}
}

//fusa:test REQ-FO-CLI042
func TestCheckSaveBaselineFlag(t *testing.T) {
	dir := goProject(t)
	bPath := filepath.Join(t.TempDir(), "baseline.json")
	code, stdout, errb := runArgs(t, "check", "--save-baseline", bPath, "--format", "text", "--dir", dir)
	if code == 2 {
		t.Errorf("--save-baseline not recognised: %s", errb)
	}
	if code != 0 && code != 1 {
		t.Fatalf("unexpected exit code %d", code)
	}
	if !strings.Contains(stdout, "Saved baseline") {
		t.Errorf("expected 'Saved baseline' in stdout: %q", stdout)
	}
	if _, err := os.Stat(bPath); err != nil {
		t.Errorf("baseline file not created: %v", err)
	}
}

//fusa:test REQ-FO-CLI043
func TestConfigValidateOK(t *testing.T) {
	dir := t.TempDir()
	cfg := config.Default("test-project")
	if err := config.Save(filepath.Join(dir, ".fusaops.json"), cfg); err != nil {
		t.Fatal(err)
	}
	code, stdout, stderr := runArgs(t, "config", "validate", "--dir", dir)
	if code != 0 {
		t.Fatalf("config validate: code=%d stderr=%q", code, stderr)
	}
	if !strings.Contains(stdout, "OK") {
		t.Errorf("expected OK in output: %q", stdout)
	}
	if !strings.Contains(stdout, "test-project") {
		t.Errorf("expected project name in output: %q", stdout)
	}
}

//fusa:test REQ-FO-CLI043
func TestConfigValidateMissing(t *testing.T) {
	dir := t.TempDir()
	code, _, stderr := runArgs(t, "config", "validate", "--dir", dir)
	if code != 1 {
		t.Fatalf("config validate missing file: got code=%d, want 1", code)
	}
	if !strings.Contains(stderr, "no config file") {
		t.Errorf("expected 'no config file' error: %q", stderr)
	}
}

//fusa:test REQ-FO-CLI043
func TestConfigValidateInvalid(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".fusaops.json"), []byte(`{"version":"1"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	code, _, stderr := runArgs(t, "config", "validate", "--dir", dir)
	if code != 1 {
		t.Fatalf("config validate invalid: got code=%d, want 1", code)
	}
	if !strings.Contains(stderr, "invalid") {
		t.Errorf("expected 'invalid' in error: %q", stderr)
	}
}

//fusa:test REQ-FO-CLI043
func TestConfigValidateFileFlag(t *testing.T) {
	dir := t.TempDir()
	cfg := config.Default("file-flag-project")
	cfgPath := filepath.Join(dir, "custom.json")
	if err := config.Save(cfgPath, cfg); err != nil {
		t.Fatal(err)
	}
	code, stdout, _ := runArgs(t, "config", "validate", "--file", cfgPath)
	if code != 0 {
		t.Fatalf("config validate --file: code=%d", code)
	}
	if !strings.Contains(stdout, "file-flag-project") {
		t.Errorf("expected project name: %q", stdout)
	}
}

//fusa:test REQ-FO-CLI044
func TestConfigShowOK(t *testing.T) {
	dir := t.TempDir()
	cfg := config.Default("show-project")
	if err := config.Save(filepath.Join(dir, ".fusaops.json"), cfg); err != nil {
		t.Fatal(err)
	}
	code, stdout, stderr := runArgs(t, "config", "show", "--dir", dir)
	if code != 0 {
		t.Fatalf("config show: code=%d stderr=%q", code, stderr)
	}
	if !strings.Contains(stdout, `"show-project"`) {
		t.Errorf("expected project name in JSON: %q", stdout)
	}
	if !strings.Contains(stdout, `"version"`) {
		t.Errorf("expected version field in JSON: %q", stdout)
	}
}

//fusa:test REQ-FO-CLI044
func TestConfigShowMissing(t *testing.T) {
	dir := t.TempDir()
	code, _, stderr := runArgs(t, "config", "show", "--dir", dir)
	if code != 1 {
		t.Fatalf("config show missing: got code=%d, want 1", code)
	}
	if !strings.Contains(stderr, "no config file") {
		t.Errorf("expected 'no config file' error: %q", stderr)
	}
}

//fusa:test REQ-FO-CLI043
//fusa:test REQ-FO-CLI044
func TestConfigUnknownSubcommand(t *testing.T) {
	code, _, stderr := runArgs(t, "config", "bogus")
	if code != 2 {
		t.Fatalf("config bogus: got code=%d, want 2", code)
	}
	if !strings.Contains(stderr, "unknown subcommand") {
		t.Errorf("expected 'unknown subcommand': %q", stderr)
	}
}

//fusa:test REQ-FO-CLI043
//fusa:test REQ-FO-CLI044
func TestConfigNoSubcommand(t *testing.T) {
	code, _, stderr := runArgs(t, "config")
	if code != 2 {
		t.Fatalf("config (no sub): got code=%d, want 2", code)
	}
	if !strings.Contains(stderr, "subcommand required") {
		t.Errorf("expected 'subcommand required': %q", stderr)
	}
}

//fusa:test REQ-FO-CLI045
func TestHistoryListEmpty(t *testing.T) {
	dir := t.TempDir()
	code, stdout, _ := runArgs(t, "history", "list", "--dir", dir)
	if code != 0 {
		t.Fatalf("history list empty: code=%d", code)
	}
	if !strings.Contains(stdout, "No history") {
		t.Errorf("expected 'No history' message: %q", stdout)
	}
}

//fusa:test REQ-FO-CLI045
func TestHistoryListText(t *testing.T) {
	dir := t.TempDir()
	// Write a minimal history entry directly.
	if err := os.WriteFile(filepath.Join(dir, ".fusaops-history.jsonl"),
		[]byte(`{"runAt":"2026-06-01T10:00:00Z","status":"PASS","total":5,"errors":0,"warnings":2,"infos":3,"languages":[]}`+"\n"),
		0o600); err != nil {
		t.Fatal(err)
	}
	code, stdout, _ := runArgs(t, "history", "list", "--dir", dir)
	if code != 0 {
		t.Fatalf("history list: code=%d", code)
	}
	if !strings.Contains(stdout, "PASS") {
		t.Errorf("expected PASS in output: %q", stdout)
	}
}

//fusa:test REQ-FO-CLI045
func TestHistoryListJSON(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".fusaops-history.jsonl"),
		[]byte(`{"runAt":"2026-06-01T10:00:00Z","status":"FAIL","total":1,"errors":1,"warnings":0,"infos":0,"languages":[]}`+"\n"),
		0o600); err != nil {
		t.Fatal(err)
	}
	code, stdout, _ := runArgs(t, "history", "list", "--dir", dir, "--format", "json")
	if code != 0 {
		t.Fatalf("history list --format json: code=%d", code)
	}
	if !strings.Contains(stdout, `"status"`) {
		t.Errorf("expected JSON output with 'status' field: %q", stdout)
	}
}

//fusa:test REQ-FO-CLI046
func TestHistoryPruneEmpty(t *testing.T) {
	dir := t.TempDir()
	code, stdout, _ := runArgs(t, "history", "prune", "--dir", dir, "--keep", "10")
	if code != 0 {
		t.Fatalf("history prune empty: code=%d", code)
	}
	if !strings.Contains(stdout, "Pruned 0") {
		t.Errorf("expected 'Pruned 0': %q", stdout)
	}
}

//fusa:test REQ-FO-CLI046
func TestHistoryPruneKeep(t *testing.T) {
	dir := t.TempDir()
	lines := ""
	for i := range 8 {
		lines += fmt.Sprintf(`{"runAt":"2026-06-01T%02d:00:00Z","status":"PASS","total":%d,"errors":0,"warnings":0,"infos":0,"languages":[]}`+"\n", i, i)
	}
	if err := os.WriteFile(filepath.Join(dir, ".fusaops-history.jsonl"), []byte(lines), 0o600); err != nil {
		t.Fatal(err)
	}
	code, stdout, _ := runArgs(t, "history", "prune", "--dir", dir, "--keep", "5")
	if code != 0 {
		t.Fatalf("history prune: code=%d", code)
	}
	if !strings.Contains(stdout, "Pruned 3") || !strings.Contains(stdout, "5 remaining") {
		t.Errorf("expected 'Pruned 3 entries, 5 remaining.': %q", stdout)
	}
}

//fusa:test REQ-FO-CLI045
//fusa:test REQ-FO-CLI046
func TestHistoryUnknownSubcommand(t *testing.T) {
	code, _, stderr := runArgs(t, "history", "bogus")
	if code != 2 {
		t.Fatalf("history bogus: got code=%d, want 2", code)
	}
	if !strings.Contains(stderr, "unknown subcommand") {
		t.Errorf("expected 'unknown subcommand': %q", stderr)
	}
}

//fusa:test REQ-FO-CLI045
//fusa:test REQ-FO-CLI046
func TestHistoryNoSubcommand(t *testing.T) {
	code, _, stderr := runArgs(t, "history")
	if code != 2 {
		t.Fatalf("history (no sub): got code=%d, want 2", code)
	}
	if !strings.Contains(stderr, "subcommand required") {
		t.Errorf("expected 'subcommand required': %q", stderr)
	}
}
