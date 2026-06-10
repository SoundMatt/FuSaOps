// Package conform runs the x-FuSa spec v1.8 conformance checks against a tool binary.
//
// It creates a temporary project, invokes each required command, and validates the
// output shapes and invariants defined in the spec.  No real binary is needed in unit
// tests — inject a runnerFunc via Options.RunFunc.
package conform

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Level is the RFC 2119 compliance level of a check.
//
//fusa:req REQ-FO-CNF001
type Level string

const (
	LevelMUST   Level = "MUST"
	LevelSHOULD Level = "SHOULD"
	LevelMAY    Level = "MAY"
)

// Status is the outcome of one conformance check.
//
//fusa:req REQ-FO-CNF001
type Status string

const (
	StatusPass Status = "PASS"
	StatusFail Status = "FAIL"
	StatusSkip Status = "SKIP"
)

// Result is the outcome of one conformance check.
//
//fusa:req REQ-FO-CNF002
type Result struct {
	ID      string `json:"id"`
	Section string `json:"section"`
	Level   Level  `json:"level"`
	Name    string `json:"name"`
	Status  Status `json:"status"`
	Detail  string `json:"detail,omitempty"`
}

// Report collects conformance results for one binary.
//
//fusa:req REQ-FO-CNF003
type Report struct {
	Binary      string    `json:"binary"`
	Tool        string    `json:"tool"`
	ToolVersion string    `json:"toolVersion"`
	Language    string    `json:"language"`
	SpecVersion string    `json:"specVersion,omitempty"`
	Results     []Result  `json:"results"`
	Generated   time.Time `json:"generated"`
}

// Summary returns counts of PASS/FAIL/SKIP results.
func (r *Report) Summary() (pass, fail, skip int) {
	for _, res := range r.Results {
		switch res.Status {
		case StatusPass:
			pass++
		case StatusFail:
			fail++
		case StatusSkip:
			skip++
		}
	}
	return
}

// HasFailures reports whether any MUST check failed.
//
//fusa:req REQ-FO-CNF003
func (r *Report) HasFailures() bool {
	for _, res := range r.Results {
		if res.Status == StatusFail && res.Level == LevelMUST {
			return true
		}
	}
	return false
}

// RunFunc is the signature of the shell-execution helper, injectable for testing.
//
//fusa:req REQ-FO-CNF004
type RunFunc func(dir, binary string, args ...string) (stdout, stderr []byte, exitCode int)

// Options configures a conformance run.
//
//fusa:req REQ-FO-CNF004
type Options struct {
	// TempDir is the project directory.  When empty, os.MkdirTemp is used and cleaned up.
	TempDir string
	// RunFunc overrides the default exec-based runner.  Set in tests.
	RunFunc RunFunc
}

// Run executes all conformance checks against binary and returns a report.
// When opts.RunFunc is set the binary is not looked up on PATH; the provided
// string is used as-is (useful for tests that inject a fake runner).
//
//fusa:req REQ-FO-CNF005
func Run(binary string, opts Options) (*Report, error) {
	run := opts.RunFunc
	binPath := binary
	if run == nil {
		run = execRun
		var lerr error
		binPath, lerr = exec.LookPath(binary)
		if lerr != nil {
			return nil, fmt.Errorf("conform: %q not found on PATH: %w", binary, lerr)
		}
	}

	tmpDir := opts.TempDir
	cleanup := false
	if tmpDir == "" {
		var terr error
		tmpDir, terr = os.MkdirTemp("", "fusaops-conform-*")
		if terr != nil {
			return nil, fmt.Errorf("conform: create temp dir: %w", terr)
		}
		cleanup = true
	}
	if cleanup {
		defer func() { _ = os.RemoveAll(tmpDir) }()
	}

	r := &runner{
		binary: binPath,
		dir:    tmpDir,
		run:    run,
		report: &Report{
			Binary:    binPath,
			Generated: time.Now().UTC(),
		},
	}

	r.checkVersion()
	if err := r.scaffold(); err != nil {
		return r.report, fmt.Errorf("conform: scaffold: %w", err)
	}
	r.checkInit()
	r.checkCheck()
	r.checkTrace()
	r.checkQualify()
	r.checkRelease()
	r.checkAuditPack()
	r.checkCapabilities()

	return r.report, nil
}

// Render writes the report to w in the requested format (text or json).
//
//fusa:req REQ-FO-CNF006
func Render(w io.Writer, rep *Report, format string) error {
	switch format {
	case "json", "":
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(rep)
	case "text":
		pass, fail, skip := rep.Summary()
		fmt.Fprintf(w, "x-FuSa conformance: %s %s\n", rep.Tool, rep.ToolVersion)
		fmt.Fprintf(w, "binary:  %s\n", rep.Binary)
		fmt.Fprintf(w, "lang:    %s\n", rep.Language)
		if rep.SpecVersion != "" {
			fmt.Fprintf(w, "spec:    %s\n", rep.SpecVersion)
		}
		fmt.Fprintf(w, "results: %d PASS  %d FAIL  %d SKIP\n\n", pass, fail, skip)
		for _, res := range rep.Results {
			sym := "✓"
			switch res.Status {
			case StatusFail:
				sym = "✗"
			case StatusSkip:
				sym = "–"
			}
			fmt.Fprintf(w, "  %s [%s] %s  %s\n", sym, res.Level, res.Section, res.Name)
			if res.Detail != "" {
				fmt.Fprintf(w, "      %s\n", res.Detail)
			}
		}
		if rep.HasFailures() {
			fmt.Fprintln(w, "\nRESULT: FAIL — one or more MUST checks failed")
		} else {
			fmt.Fprintln(w, "\nRESULT: PASS")
		}
		return nil
	default:
		return fmt.Errorf("conform: unsupported format %q", format)
	}
}

// runner holds the mutable state of a single conformance run.
type runner struct {
	binary string
	dir    string
	run    RunFunc
	report *Report
	lang   string
}

func (r *runner) addResult(res Result) {
	r.report.Results = append(r.report.Results, res)
}

func (r *runner) pass(id, section string, level Level, name string) {
	r.addResult(Result{ID: id, Section: section, Level: level, Name: name, Status: StatusPass})
}

func (r *runner) fail(id, section string, level Level, name, detail string) {
	r.addResult(Result{ID: id, Section: section, Level: level, Name: name, Status: StatusFail, Detail: detail})
}

func (r *runner) skip(id, section string, level Level, name, reason string) {
	r.addResult(Result{ID: id, Section: section, Level: level, Name: name, Status: StatusSkip, Detail: reason})
}

// scaffold writes the minimum project fixtures.
func (r *runner) scaffold() error {
	fusa := `{
  "configVersion": "1.0",
  "project": { "name": "conform-test", "version": "0.1.0" },
  "standard": "iso26262",
  "asil": "ASIL-C"
}
`
	reqs := `{
  "requirements": [
    { "id": "REQ-TEST-001", "title": "Test requirement", "text": "The tool shall analyse source code.", "standard": "iso26262", "level": "HLR" },
    { "id": "REQ-TEST-002", "title": "Test requirement 2", "text": "Analysis results shall be reported.", "standard": "iso26262", "level": "HLR" }
  ]
}
`
	if err := os.WriteFile(filepath.Join(r.dir, ".fusa.json"), []byte(fusa), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(r.dir, ".fusa-reqs.json"), []byte(reqs), 0o644); err != nil {
		return err
	}
	if err := r.writeSourceFiles(); err != nil {
		return err
	}
	return nil
}

// writeSourceFiles creates language-appropriate source stubs with annotations.
func (r *runner) writeSourceFiles() error {
	lang := r.lang
	if lang == "" {
		lang = langFromBinary(filepath.Base(r.binary))
	}
	var name, content string
	switch lang {
	case "go":
		name = "main.go"
		content = `package main

//fusa:req REQ-TEST-001
//fusa:test REQ-TEST-001

//fusa:req REQ-TEST-002

func main() {}
`
	case "c":
		name = "main.c"
		content = `#include <stdio.h>

//fusa:req REQ-TEST-001
//fusa:test REQ-TEST-001

//fusa:req REQ-TEST-002

int main(void) { return 0; }
`
	case "cpp":
		name = "main.cpp"
		content = `#include <iostream>

//fusa:req REQ-TEST-001
//fusa:test REQ-TEST-001

//fusa:req REQ-TEST-002

int main() { return 0; }
`
	default:
		return nil
	}
	return os.WriteFile(filepath.Join(r.dir, name), []byte(content), 0o644)
}

// langFromBinary maps a binary basename to a language id per §1.1.
//
//fusa:req REQ-FO-CNF007
func langFromBinary(base string) string {
	switch strings.ToLower(base) {
	case "gofusa":
		return "go"
	case "cfusa":
		return "c"
	case "cpfusa":
		return "cpp"
	default:
		return ""
	}
}

// execRun is the default RunFunc that invokes a real subprocess.
func execRun(dir, binary string, args ...string) (stdout, stderr []byte, exitCode int) {
	cmd := exec.CommandContext(context.Background(), binary, args...)
	cmd.Dir = dir
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return outBuf.Bytes(), errBuf.Bytes(), ee.ExitCode()
		}
		return outBuf.Bytes(), errBuf.Bytes(), -1
	}
	return outBuf.Bytes(), errBuf.Bytes(), 0
}

// decodeJSON unmarshals b into v, stripping leading/trailing non-JSON noise.
func decodeJSON(b []byte, v interface{}) error {
	start := bytes.IndexByte(b, '{')
	if start < 0 {
		start = bytes.IndexByte(b, '[')
	}
	end := bytes.LastIndexByte(b, '}')
	if end < 0 {
		end = bytes.LastIndexByte(b, ']')
	}
	if start < 0 || end < 0 || end < start {
		return fmt.Errorf("no JSON object found")
	}
	return json.Unmarshal(b[start:end+1], v)
}
