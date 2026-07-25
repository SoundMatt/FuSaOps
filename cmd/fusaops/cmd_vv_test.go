package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SoundMatt/FuSaOps/config"
)

// fusaopsProject creates a temp dir with a .fusaops.json for vv tests.
func fusaopsVVProject(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	cfg := config.Default("vv-test-project")
	cfgPath := filepath.Join(dir, config.ConfigFile)
	if err := config.Save(cfgPath, cfg); err != nil {
		t.Fatal(err)
	}
	return dir
}

// TestVVHelp verifies that fusaops vv --help exits cleanly.
//
//fusa:test REQ-FO-CLI074
func TestVVHelp(t *testing.T) {
	code, _, _ := runArgs(t, "vv", "--help")
	if code != 0 && code != 2 {
		t.Errorf("vv --help: got code %d, want 0 or 2", code)
	}
}

// TestVVUnknownSubcommand verifies that an unknown subcommand exits non-zero.
//
//fusa:test REQ-FO-CLI074
func TestVVUnknownSubcommand(t *testing.T) {
	dir := fusaopsVVProject(t)
	code, _, stderr := runArgs(t, "vv", "--dir", dir, "bogus")
	if code == 0 {
		t.Error("expected non-zero exit for unknown subcommand")
	}
	if !strings.Contains(stderr, "unknown subcommand") {
		t.Errorf("expected unknown subcommand in stderr: %q", stderr)
	}
}

// TestVVShowDefaultsToText verifies that fusaops vv show (no format) produces text.
//
//fusa:test REQ-FO-CLI075
func TestVVShowDefaultsToText(t *testing.T) {
	dir := fusaopsVVProject(t)
	code, stdout, _ := runArgs(t, "vv", "--dir", dir, "show")
	if code != 0 {
		t.Fatalf("vv show: got code %d", code)
	}
	if !strings.Contains(stdout, "ASIL-B") {
		t.Errorf("expected ASIL-B in text output: %q", stdout)
	}
}

// TestVVShowNoConfig verifies that fusaops vv show tolerates a missing
// .fusaops.json and shows an empty declaration.
//
//fusa:test REQ-FO-CLI075
func TestVVShowNoConfig(t *testing.T) {
	dir := t.TempDir() // no .fusaops.json
	code, stdout, _ := runArgs(t, "vv", "--dir", dir, "show")
	if code != 0 {
		t.Fatalf("vv show without config: got code %d", code)
	}
	if !strings.Contains(stdout, "ASIL-B") {
		t.Errorf("expected ASIL-B in output for empty declaration: %q", stdout)
	}
}

// TestVVShowJSON verifies that fusaops vv show --format json produces valid JSON.
//
//fusa:test REQ-FO-CLI075
func TestVVShowJSON(t *testing.T) {
	dir := fusaopsVVProject(t)
	code, stdout, _ := runArgs(t, "vv", "--dir", dir, "show", "--format", "json")
	if code != 0 {
		t.Fatalf("vv show --format json: got code %d", code)
	}
	var got map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("not valid JSON: %v\noutput: %s", err, stdout)
	}
	if got["achievableAsil"] != "ASIL-B" {
		t.Errorf("achievableAsil: got %v, want ASIL-B", got["achievableAsil"])
	}
}

// TestVVShowOutputFile verifies that --output writes the result to a file.
//
//fusa:test REQ-FO-CLI075
func TestVVShowOutputFile(t *testing.T) {
	dir := fusaopsVVProject(t)
	out := filepath.Join(dir, "vv-out.txt")
	code, stdout, _ := runArgs(t, "vv", "--dir", dir, "show", "--output", out)
	if code != 0 {
		t.Fatalf("vv show --output: got code %d", code)
	}
	if stdout != "" {
		t.Errorf("stdout must be empty when --output given: %q", stdout)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("output file not created: %v", err)
	}
	if !strings.Contains(string(data), "ASIL-B") {
		t.Errorf("output file missing ASIL-B: %s", data)
	}
}

// TestVVSetNoConfig verifies that fusaops vv set without .fusaops.json prints
// a helpful message and exits non-zero.
//
//fusa:test REQ-FO-CLI076
func TestVVSetNoConfig(t *testing.T) {
	dir := t.TempDir()
	code, _, stderr := runArgs(t, "vv", "--dir", dir, "set", "--implementation-author", "Alice")
	if code == 0 {
		t.Error("expected non-zero exit when no config file")
	}
	if !strings.Contains(stderr, "fusaops init") {
		t.Errorf("expected helpful message mentioning 'fusaops init': %q", stderr)
	}
}

// TestVVSetUpdatesFields verifies that fusaops vv set writes the supplied fields
// to .fusaops.json and reports the achievable ASIL.
//
//fusa:test REQ-FO-CLI076
func TestVVSetUpdatesFields(t *testing.T) {
	dir := fusaopsVVProject(t)
	code, stdout, _ := runArgs(t, "vv", "--dir", dir, "set",
		"--implementation-author", "Alice",
		"--independent-reviewer", "Bob",
	)
	if code != 0 {
		t.Fatalf("vv set: got code %d", code)
	}
	if !strings.Contains(stdout, "ASIL-C") {
		t.Errorf("expected ASIL-C in output: %q", stdout)
	}

	// Verify config was actually saved.
	cfg, err := config.Load(filepath.Join(dir, config.ConfigFile))
	if err != nil {
		t.Fatalf("load config after set: %v", err)
	}
	if cfg.VandV.ImplementationAuthor != "Alice" {
		t.Errorf("ImplementationAuthor: got %q, want Alice", cfg.VandV.ImplementationAuthor)
	}
	if cfg.VandV.IndependentReviewer != "Bob" {
		t.Errorf("IndependentReviewer: got %q, want Bob", cfg.VandV.IndependentReviewer)
	}
}

// TestVVSetFullIndependence verifies that setting all three fields reports ASIL-D.
//
//fusa:test REQ-FO-CLI076
func TestVVSetFullIndependence(t *testing.T) {
	dir := fusaopsVVProject(t)
	code, stdout, _ := runArgs(t, "vv", "--dir", dir, "set",
		"--implementation-author", "Alice",
		"--independent-reviewer", "Bob",
		"--independent-test-executor", "Carol",
	)
	if code != 0 {
		t.Fatalf("vv set: got code %d", code)
	}
	if !strings.Contains(stdout, "ASIL-D") {
		t.Errorf("expected ASIL-D in output: %q", stdout)
	}
}

// TestVVSetOnlyUpdatesSuppliedFlags verifies that omitted flags are not zeroed out.
//
//fusa:test REQ-FO-CLI076
func TestVVSetOnlyUpdatesSuppliedFlags(t *testing.T) {
	dir := fusaopsVVProject(t)
	// First set the author.
	if code, _, _ := runArgs(t, "vv", "--dir", dir, "set", "--implementation-author", "Alice"); code != 0 {
		t.Fatal("first set failed")
	}
	// Then set only the reviewer (author should be preserved).
	if code, _, _ := runArgs(t, "vv", "--dir", dir, "set", "--independent-reviewer", "Bob"); code != 0 {
		t.Fatal("second set failed")
	}
	cfg, err := config.Load(filepath.Join(dir, config.ConfigFile))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.VandV.ImplementationAuthor != "Alice" {
		t.Errorf("ImplementationAuthor wiped out: got %q, want Alice", cfg.VandV.ImplementationAuthor)
	}
	if cfg.VandV.IndependentReviewer != "Bob" {
		t.Errorf("IndependentReviewer: got %q, want Bob", cfg.VandV.IndependentReviewer)
	}
}

// TestVVValidationWarningsToStderr verifies that consistency warnings appear on
// stderr (e.g. reviewer same as author).
//
//fusa:test REQ-FO-CLI076
func TestVVValidationWarningsToStderr(t *testing.T) {
	dir := fusaopsVVProject(t)
	code, _, stderr := runArgs(t, "vv", "--dir", dir, "set",
		"--implementation-author", "Alice",
		"--independent-reviewer", "Alice", // same as author — should warn
	)
	if code != 0 {
		t.Fatalf("vv set: got code %d", code)
	}
	if !strings.Contains(stderr, "same person") {
		t.Errorf("expected same-person warning in stderr: %q", stderr)
	}
}

// TestVVInMainDispatch verifies that "vv" is wired into the main dispatcher.
//
//fusa:test REQ-FO-CLI074
func TestVVInMainDispatch(t *testing.T) {
	dir := fusaopsVVProject(t)
	code, _, _ := runArgs(t, "vv", "--dir", dir)
	// Default subcommand (show) should succeed.
	if code != 0 {
		t.Errorf("vv (no subcommand): got code %d, want 0", code)
	}
}
