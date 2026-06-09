package main

import (
	"bytes"
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

func TestNoArgsShowsUsage(t *testing.T) {
	code, out, _ := runArgs(t)
	if code != 1 || !strings.Contains(out, "Usage:") {
		t.Errorf("no-args: code=%d out=%q", code, out)
	}
}

func TestUnknownCommand(t *testing.T) {
	code, _, errb := runArgs(t, "frobnicate")
	if code != 1 || !strings.Contains(errb, "unknown command") {
		t.Errorf("unknown cmd: code=%d err=%q", code, errb)
	}
}

func TestVersion(t *testing.T) {
	code, out, _ := runArgs(t, "version")
	if code != 0 || !strings.Contains(out, "fusaops") {
		t.Errorf("version: code=%d out=%q", code, out)
	}
}

func TestHelp(t *testing.T) {
	code, out, _ := runArgs(t, "help")
	if code != 0 || !strings.Contains(out, "Commands:") {
		t.Errorf("help: code=%d out=%q", code, out)
	}
}

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

func TestCheckNoLanguages(t *testing.T) {
	// Empty dir → no adapters → exit 1 with message.
	code, _, errb := runArgs(t, "check", "--dir", t.TempDir())
	if code != 1 || !strings.Contains(errb, "no supported languages") {
		t.Errorf("check empty: code=%d err=%q", code, errb)
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
