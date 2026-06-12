package main

import (
	"bytes"
	"strings"
	"testing"
)

// TestStandardsCommandNoLanguages verifies exit 1 when no languages detected.
//
//fusa:test REQ-FO-CLI015
//fusa:test REQ-FO-CLI016
//fusa:test REQ-FO-CLI017
//fusa:test REQ-FO-CLI019
//fusa:test REQ-FO-CLI020
//fusa:test REQ-FO-CLI022
func TestStandardsCommandNoLanguages(t *testing.T) {
	for _, cmd := range []string{"iso26262", "iec61508", "do178", "iso21434", "unece", "iec62443"} {
		var stdout, stderr bytes.Buffer
		// TempDir has no source files → no adapters applicable → ErrNoAdapters.
		code := runStandards(cmd, []string{"--dir", t.TempDir()}, &stdout, &stderr)
		if code != 1 {
			t.Errorf("%s: want exit 1 for no languages, got %d (stderr: %s)", cmd, code, stderr.String())
		}
	}
}

// TestIec62443Command verifies iec62443 is dispatched and the help text mentions IEC 62443.
//
//fusa:test REQ-FO-CLI022
func TestIec62443Command(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runStandards("iec62443", []string{"--help"}, &stdout, &stderr)
	_ = code
	out := stdout.String() + stderr.String()
	if !strings.Contains(out, "IEC 62443") {
		t.Errorf("iec62443 usage should mention IEC 62443: %s", out)
	}
}

// TestStandardsCommandBadFormat verifies exit 2 for unsupported format.
//
//fusa:test REQ-FO-CLI015
func TestStandardsCommandBadFormat(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runStandards("iso26262", []string{"--format", "xml"}, &stdout, &stderr)
	if code != 2 {
		t.Errorf("want exit 2, got %d", code)
	}
}

// TestStandardsCommandBadFlag verifies exit 2 for unknown flag.
//
//fusa:test REQ-FO-CLI015
func TestStandardsCommandBadFlag(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runStandards("iso26262", []string{"--notaflag"}, &stdout, &stderr)
	if code != 2 {
		t.Errorf("want exit 2, got %d", code)
	}
}

// TestStandardsCommandDo178Alias verifies do178 maps to do178c.
//
//fusa:test REQ-FO-CLI017
func TestStandardsCommandDo178Alias(t *testing.T) {
	var stdout, stderr bytes.Buffer
	// Help output should mention DO-178C (not do178c literally) for the alias.
	code := runStandards("do178", []string{"--help"}, &stdout, &stderr)
	_ = code // flag.ContinueOnError returns 2 on --help, that's fine
	out := stdout.String() + stderr.String()
	if !strings.Contains(out, "DO-178C") {
		t.Errorf("do178 usage should mention DO-178C: %s", out)
	}
}
