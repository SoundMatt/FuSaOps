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

func TestSplitCSV(t *testing.T) {
	got := splitCSV("a,b,,c")
	if len(got) != 3 || got[0] != "a" || got[2] != "c" {
		t.Errorf("splitCSV: %v", got)
	}
	if len(splitCSV("")) != 0 {
		t.Error("splitCSV empty should be empty")
	}
}
