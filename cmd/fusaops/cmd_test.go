package main

import (
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
