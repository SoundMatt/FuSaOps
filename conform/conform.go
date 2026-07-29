// Package conform runs the x-FuSa spec v1.10 conformance checks against a tool binary.
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
	"html/template"
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
//
//fusa:req REQ-FO-CNF020
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

// Render writes the report to w in the requested format (text, json, html, or markdown).
//
//fusa:req REQ-FO-CNF006
//fusa:req REQ-FO-CNF018
//fusa:req REQ-FO-CNF019
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
	case "html":
		pass, fail, skip := rep.Summary()
		data := struct {
			*Report
			Pass, Fail, Skip int
		}{rep, pass, fail, skip}
		if err := conformTemplate.Execute(w, data); err != nil {
			return fmt.Errorf("conform: html render: %w", err)
		}
		return nil
	case "markdown", "md":
		return renderMarkdown(w, rep)
	default:
		return fmt.Errorf("conform: unsupported format %q", format)
	}
}

// renderMarkdown writes a GFM markdown conformance report to w.
//
//fusa:req REQ-FO-CNF019
func renderMarkdown(w io.Writer, rep *Report) error {
	pass, fail, skip := rep.Summary()
	badge := "🟢"
	if rep.HasFailures() {
		badge = "🔴"
	}
	status := "PASS"
	if rep.HasFailures() {
		status = "FAIL"
	}
	fmt.Fprintf(w, "# FuSaOps — Conformance: %s\n\n", rep.Tool)
	fmt.Fprintf(w, "%s **%s** · %s %s · spec %s\n\n", badge, status, rep.Tool, rep.ToolVersion, rep.SpecVersion)
	fmt.Fprintf(w, "%d pass · %d fail · %d skip\n\n", pass, fail, skip)
	fmt.Fprintln(w, "| Result | Level | Section | Check |")
	fmt.Fprintln(w, "|---|---|---|---|")
	for _, res := range rep.Results {
		sym := "✅"
		switch res.Status {
		case StatusFail:
			sym = "❌"
		case StatusSkip:
			sym = "⏭"
		}
		name := strings.ReplaceAll(res.Name, "|", "\\|")
		if res.Detail != "" {
			name += " — " + strings.ReplaceAll(res.Detail, "|", "\\|")
		}
		fmt.Fprintf(w, "| %s | %s | %s | %s |\n", sym, res.Level, res.Section, name)
	}
	return nil
}

// conformTemplate is a self-contained HTML conformance report.
//
//fusa:req REQ-FO-CNF018
var conformTemplate = template.Must(template.New("conform").Parse(`<!doctype html>
<html lang="en"><head><meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>FuSaOps — Conformance: {{.Tool}}</title>
<style>
 body{font:15px/1.5 system-ui,sans-serif;margin:0;background:#0f1115;color:#e6e6e6}
 header{padding:1.2rem 1.6rem;background:#171a21;border-bottom:1px solid #272b34}
 h1{margin:0;font-size:1.25rem} .meta{color:#9aa3b2;font-size:.85rem;margin-top:.3rem}
 main{padding:1.6rem;max-width:960px;margin:0 auto}
 .badge{display:inline-block;padding:.15rem .6rem;border-radius:.5rem;font-weight:600;font-size:.85rem}
 .PASS{background:#16361f;color:#7ee2a0} .FAIL{background:#3a1212;color:#f07070}
 table{width:100%;border-collapse:collapse;background:#171a21;border-radius:.6rem;overflow:hidden;margin-top:1rem}
 th,td{padding:.55rem .8rem;text-align:left;border-bottom:1px solid #272b34;font-size:.9rem}
 th{background:#1d2129;color:#9aa3b2;font-weight:600}
 .p{color:#7ee2a0} .f{color:#f07070} .s{color:#9aa3b2}
 .detail{color:#9aa3b2;font-size:.85rem;font-style:italic}
 .level{display:inline-block;padding:.1rem .4rem;border-radius:.3rem;font-size:.8rem;background:#1d2129;color:#9aa3b2}
</style></head><body>
<header>
 <h1>FuSaOps — x-FuSa Conformance: {{.Tool}} {{.ToolVersion}}</h1>
 <div class="meta">Binary: {{.Binary}} · Language: {{.Language}}{{if .SpecVersion}} · Spec: {{.SpecVersion}}{{end}} ·
  Generated {{.Generated.Format "2006-01-02 15:04 MST"}} ·
  {{.Pass}} passed · {{.Fail}} failed · {{.Skip}} skipped ·
  <span class="badge {{if .HasFailures}}FAIL{{else}}PASS{{end}}">{{if .HasFailures}}FAIL{{else}}PASS{{end}}</span></div>
</header>
<main>
 <table>
  <thead><tr><th>Result</th><th>Level</th><th>§ Section</th><th>Check</th></tr></thead>
  <tbody>
  {{range .Results}}
   <tr>
    <td>{{if eq .Status "PASS"}}<span class="p">PASS</span>{{else if eq .Status "FAIL"}}<span class="f">FAIL</span>{{else}}<span class="s">SKIP</span>{{end}}</td>
    <td><span class="level">{{.Level}}</span></td>
    <td>{{.Section}}</td>
    <td>{{.Name}}{{if .Detail}}<br><span class="detail">{{.Detail}}</span>{{end}}</td>
   </tr>
  {{end}}
  </tbody>
 </table>
</main></body></html>
`))

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
	// Build annotation markers at runtime so the scanner does not treat this
	// source file itself as carrying REQ-TEST-* FuSaOps requirements.
	slashReq := "//" + "fusa:req"
	slashTest := "//" + "fusa:test"
	hashReq := "#" + "fusa:req"
	hashTest := "#" + "fusa:test"
	dashReq := "--" + "fusa:req"
	dashTest := "--" + "fusa:test"
	switch lang {
	case "go":
		src := fmt.Sprintf("package main\n\n%s REQ-TEST-001\n%s REQ-TEST-001\n\n%s REQ-TEST-002\n\nfunc main() {}\n",
			slashReq, slashTest, slashReq)
		return os.WriteFile(filepath.Join(r.dir, "main.go"), []byte(src), 0o644)
	case "c":
		src := fmt.Sprintf("#include <stdio.h>\n\n%s REQ-TEST-001\n%s REQ-TEST-001\n\n%s REQ-TEST-002\n\nint main(void) { return 0; }\n",
			slashReq, slashTest, slashReq)
		return os.WriteFile(filepath.Join(r.dir, "main.c"), []byte(src), 0o644)
	case "cpp":
		src := fmt.Sprintf("#include <iostream>\n\n%s REQ-TEST-001\n%s REQ-TEST-001\n\n%s REQ-TEST-002\n\nint main() { return 0; }\n",
			slashReq, slashTest, slashReq)
		return os.WriteFile(filepath.Join(r.dir, "main.cpp"), []byte(src), 0o644)
	case "rust":
		if err := os.MkdirAll(filepath.Join(r.dir, "src"), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(r.dir, "Cargo.toml"), []byte(`[package]
name = "conform-test"
version = "0.1.0"
edition = "2021"
`), 0o644); err != nil {
			return err
		}
		src := fmt.Sprintf("%s REQ-TEST-001\n%s REQ-TEST-001\n\n%s REQ-TEST-002\n\nfn main() {}\n",
			slashReq, slashTest, slashReq)
		return os.WriteFile(filepath.Join(r.dir, "src", "main.rs"), []byte(src), 0o644)
	case "python":
		src := fmt.Sprintf("%s REQ-TEST-001\n%s REQ-TEST-001\n\n%s REQ-TEST-002\n",
			hashReq, hashTest, hashReq)
		return os.WriteFile(filepath.Join(r.dir, "main.py"), []byte(src), 0o644)
	case "java":
		src := fmt.Sprintf("%s REQ-TEST-001\n%s REQ-TEST-001\n\n%s REQ-TEST-002\n\npublic class Main {\n    public static void main(String[] args) {}\n}\n",
			slashReq, slashTest, slashReq)
		return os.WriteFile(filepath.Join(r.dir, "Main.java"), []byte(src), 0o644)
	case "ada":
		if err := os.WriteFile(filepath.Join(r.dir, "conform_test.gpr"), []byte(`project Conform_Test is
   for Main use ("main.adb");
   for Source_Dirs use (".");
   for Object_Dir use "obj";
end Conform_Test;
`), 0o644); err != nil {
			return err
		}
		src := fmt.Sprintf("%s REQ-TEST-001\n%s REQ-TEST-001\n\n%s REQ-TEST-002\n\nprocedure Main is\nbegin\n   null;\nend Main;\n",
			dashReq, dashTest, dashReq)
		return os.WriteFile(filepath.Join(r.dir, "main.adb"), []byte(src), 0o644)
	default:
		return nil
	}
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
	case "rsfusa":
		return "rust"
	case "pyfusa":
		return "python"
	case "jfusa":
		return "java"
	case "adafusa":
		return "ada"
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
