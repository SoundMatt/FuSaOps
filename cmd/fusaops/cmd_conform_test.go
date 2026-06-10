package main

import (
	"bytes"
	"strings"
	"testing"
)

// TestConformCommandNoArg verifies usage error with no binary argument.
//
//fusa:test REQ-FO-CLI014
func TestConformCommandNoArg(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runConform([]string{}, &stdout, &stderr)
	if code != 2 {
		t.Errorf("want exit 2, got %d", code)
	}
	if !strings.Contains(stderr.String(), "binary") {
		t.Errorf("expected 'binary' in error output, got: %s", stderr.String())
	}
}

// TestConformCommandBadFormat verifies usage error for invalid format.
//
//fusa:test REQ-FO-CLI014
func TestConformCommandBadFormat(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runConform([]string{"--format", "xml", "somebinary"}, &stdout, &stderr)
	if code != 2 {
		t.Errorf("want exit 2, got %d", code)
	}
}

// TestConformCommandBadFlag verifies usage error for invalid flag.
//
//fusa:test REQ-FO-CLI014
func TestConformCommandBadFlag(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runConform([]string{"--notaflag"}, &stdout, &stderr)
	if code != 2 {
		t.Errorf("want exit 2, got %d", code)
	}
}

// TestConformCommandBinaryNotFound verifies exit 1 when binary is missing.
//
//fusa:test REQ-FO-CLI014
func TestConformCommandBinaryNotFound(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runConform([]string{"no-such-xfusa-binary-xyz"}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("want exit 1 for missing binary, got %d", code)
	}
	if !strings.Contains(stderr.String(), "not found") {
		t.Errorf("expected 'not found' in error, got: %s", stderr.String())
	}
}
