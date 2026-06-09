package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

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

func TestCheckReturnsReport(t *testing.T) {
	dir := goProject(t)
	// gofusa is not assumed installed; the component is skipped, so no ERROR
	// findings are produced and check exits 0.
	code, stdout, _ := runArgs(t, "check", "--dir", dir, "--format", "text")
	if code != 0 {
		t.Fatalf("check: code=%d", code)
	}
	if !strings.Contains(stdout, "FuSaOps") {
		t.Errorf("check stdout: %q", stdout)
	}
}

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
	if code != 0 {
		t.Fatalf("check: code=%d", code)
	}
	if !strings.Contains(stdout, "configured-project") {
		t.Errorf("check did not use config project name: %q", stdout)
	}
}

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
