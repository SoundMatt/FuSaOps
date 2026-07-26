package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

//fusa:test REQ-FO-CLI082
func TestCompNoLanguages(t *testing.T) {
	code, _, errb := runArgs(t, "comp", "--dir", t.TempDir())
	if code != 1 || !strings.Contains(errb, "no supported languages") {
		t.Errorf("comp empty dir: code=%d err=%q", code, errb)
	}
}

//fusa:test REQ-FO-CLI082
func TestCompTextFormat(t *testing.T) {
	dir := goProject(t)
	code, _, errb := runArgs(t, "comp", "--dir", dir, "--format", "text")
	// 0 = pass, 1 = violations or tool skipped — both valid.
	if code > 1 {
		t.Fatalf("comp text: unexpected code=%d err=%q", code, errb)
	}
}

//fusa:test REQ-FO-CLI082
func TestCompJSONFormat(t *testing.T) {
	dir := goProject(t)
	code, out, errb := runArgs(t, "comp", "--dir", dir, "--format", "json")
	if code > 1 {
		t.Fatalf("comp json: unexpected code=%d err=%q", code, errb)
	}
	if !strings.Contains(out, `"totalFunctions"`) || !strings.Contains(out, `"components"`) {
		t.Errorf("comp json: missing expected fields: %q", out)
	}
}

//fusa:test REQ-FO-CLI082
func TestCompInvalidDAL(t *testing.T) {
	code, _, errb := runArgs(t, "comp", "--dal", "DAL-Z")
	if code != 2 || !strings.Contains(errb, "DAL") {
		t.Errorf("comp --dal DAL-Z: code=%d err=%q", code, errb)
	}
}

//fusa:test REQ-FO-CLI082
func TestCompInvalidTimeout(t *testing.T) {
	dir := goProject(t)
	code, _, errb := runArgs(t, "comp", "--timeout", "notaduration", "--dir", dir)
	if code != 2 || !strings.Contains(errb, "--timeout") {
		t.Errorf("comp --timeout notaduration: code=%d err=%q", code, errb)
	}
}

//fusa:test REQ-FO-CLI082
func TestCompOutputFile(t *testing.T) {
	dir := goProject(t)
	out := filepath.Join(t.TempDir(), "comp-report.json")
	code, stdout, errb := runArgs(t, "comp", "--dir", dir, "--format", "json", "--output", out)
	if code > 1 {
		t.Fatalf("comp --output: unexpected code=%d err=%q", code, errb)
	}
	if stdout != "" {
		t.Errorf("comp stdout must be empty when --output given: %q", stdout)
	}
	if _, err := os.Stat(out); err != nil {
		t.Errorf("comp --output file not created: %v", err)
	}
	if !strings.Contains(errb, "comp-report.json") {
		t.Errorf("expected filename in stderr confirmation: %q", errb)
	}
}

//fusa:test REQ-FO-CLI082
func TestCompWorkersFlag(t *testing.T) {
	code, _, errb := runArgs(t, "comp", "--workers", "2", "--dir", goProject(t))
	if code == 2 {
		t.Errorf("--workers not recognised: %s", errb)
	}
}

//fusa:test REQ-FO-CLI082
func TestCompDALB(t *testing.T) {
	dir := goProject(t)
	code, _, errb := runArgs(t, "comp", "--dal", "DAL-B", "--dir", dir)
	if code == 2 {
		t.Errorf("comp --dal DAL-B should be valid: code=%d err=%q", code, errb)
	}
}

//fusa:test REQ-FO-CLI082
func TestCompInvalidFormat(t *testing.T) {
	dir := goProject(t)
	code, _, errb := runArgs(t, "comp", "--format", "xml", "--dir", dir)
	if code != 1 || !strings.Contains(errb, "comp") {
		t.Errorf("comp --format xml: code=%d err=%q", code, errb)
	}
}

//fusa:test REQ-FO-CLI082
func TestCompBadOutputPath(t *testing.T) {
	dir := goProject(t)
	code, _, errb := runArgs(t, "comp", "--output", "/nonexistent/dir/comp.json", "--dir", dir)
	if code != 1 || !strings.Contains(errb, "comp") {
		t.Errorf("comp bad --output: code=%d err=%q", code, errb)
	}
}
