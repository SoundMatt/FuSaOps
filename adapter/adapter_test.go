package adapter

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	fusaops "github.com/SoundMatt/FuSaOps"
)

const sampleReport = `{
  "generatedAt": "2026-06-09T00:00:00Z",
  "projectRoot": "/x",
  "findings": [
    {"ruleId":"LINT001","severity":"WARNING","message":"discard","location":{"file":"a.go","line":7,"column":3},"remediation":"fix it"},
    {"ruleId":"FUSA001","severity":"ERROR","message":"no config","location":{"file":".fusa.json"}},
    {"ruleId":"NOTE1","severity":"weird","message":"unknown sev"}
  ],
  "summary": {"total":3,"errors":1,"warnings":1,"infos":1}
}`

//fusa:test REQ-FO-ADP006
func TestParseToolReport(t *testing.T) {
	findings, _, err := parseToolReport([]byte(sampleReport), fusaops.LangGo, "go-FuSa")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(findings) != 3 {
		t.Fatalf("got %d findings, want 3", len(findings))
	}
	f := findings[0]
	if f.Language != fusaops.LangGo || f.Tool != "go-FuSa" {
		t.Errorf("tagging wrong: %+v", f)
	}
	if f.RuleID != "LINT001" || f.Severity != fusaops.SeverityWarning || f.Location.Line != 7 {
		t.Errorf("finding 0 wrong: %+v", f)
	}
	// Unknown severity must normalise to INFO, never be dropped.
	if findings[2].Severity != fusaops.SeverityInfo {
		t.Errorf("unknown severity: got %v, want INFO", findings[2].Severity)
	}
}

func TestParseToolReportInvalid(t *testing.T) {
	if _, _, err := parseToolReport([]byte("{not json"), fusaops.LangGo, "t"); err == nil {
		t.Error("expected error on invalid JSON")
	}
}

func TestNormaliseSeverity(t *testing.T) {
	cases := map[string]fusaops.Severity{
		"ERROR": fusaops.SeverityError, "WARNING": fusaops.SeverityWarning,
		"INFO": fusaops.SeverityInfo, "": fusaops.SeverityInfo, "junk": fusaops.SeverityInfo,
	}
	for in, want := range cases {
		if got := normaliseSeverity(in); got != want {
			t.Errorf("normaliseSeverity(%q): got %v, want %v", in, got, want)
		}
	}
}

//fusa:test REQ-FO-ADP006
func TestNormaliseCategory(t *testing.T) {
	cases := map[string]string{
		"lint": "lint", "security": "security", "supply-chain": "supply-chain",
		"other": "other", "": "other", "bogus": "other",
	}
	for in, want := range cases {
		if got := normaliseCategory(in); got != want {
			t.Errorf("normaliseCategory(%q): got %q, want %q", in, got, want)
		}
	}
}

//fusa:test REQ-FO-ADP005
func TestCheckExit3SurfacesAsError(t *testing.T) {
	reportJSON := `{"findings":[{"ruleId":"X","severity":"ERROR","message":"partial"}],"error":{"code":"internal","message":"crashed mid-scan"}}`
	a := &cmdAdapter{
		name: "test-tool", tool: "testtool", language: fusaops.LangGo,
		run: func(ctx context.Context, dir, name string, args ...string) ([]byte, error) {
			// Emulate the runner writing a partial report then reporting the
			// tool's exit 3, mirroring defaultRunner's behavior on exit 3.
			for i, a := range args {
				if a == "--output" && i+1 < len(args) {
					_ = os.WriteFile(args[i+1], []byte(reportJSON), 0o600)
				}
			}
			return nil, errors.New("testtool exited 3 (runtime/internal error)")
		},
	}
	_, err := a.Check(context.Background(), t.TempDir())
	if err == nil {
		t.Fatal("Check: want error on exit-3 runtime failure, got nil")
	}
}

//fusa:test REQ-FO-ADP005
func TestCheckErrorFieldWithoutRunError(t *testing.T) {
	reportJSON := `{"findings":[],"error":{"code":"unsupported","message":"nothing to analyze"}}`
	a := &cmdAdapter{
		name: "test-tool", tool: "testtool", language: fusaops.LangGo,
		run: func(ctx context.Context, dir, name string, args ...string) ([]byte, error) {
			for i, a := range args {
				if a == "--output" && i+1 < len(args) {
					_ = os.WriteFile(args[i+1], []byte(reportJSON), 0o600)
				}
			}
			return nil, nil
		},
	}
	_, err := a.Check(context.Background(), t.TempDir())
	if err == nil {
		t.Fatal("Check: want error when report's error field is present even with nil run error")
	}
}

//fusa:test REQ-FO-ADP004
func TestDetect(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main"), 0o600); err != nil {
		t.Fatal(err)
	}
	a := newGoFuSa()
	ok, err := a.Detect(dir)
	if err != nil || !ok {
		t.Errorf("go detect: ok=%v err=%v", ok, err)
	}
	cpp := newCppFuSa()
	ok, _ = cpp.Detect(dir)
	if ok {
		t.Error("cpp adapter should not detect a Go-only project")
	}
}

func TestDetectSkipsBuildDirs(t *testing.T) {
	dir := t.TempDir()
	buildDir := filepath.Join(dir, "build")
	if err := os.MkdirAll(buildDir, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(buildDir, "gen.go"), []byte("package x"), 0o600); err != nil {
		t.Fatal(err)
	}
	ok, err := newGoFuSa().Detect(dir)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Error("should not detect Go inside build/ dir")
	}
}

// fakeRunner writes fixture JSON to the --output path the adapter passes.
func fakeRunner(payload string) runnerFunc {
	return func(_ context.Context, _, _ string, args ...string) ([]byte, error) {
		out := args[len(args)-1] // Check passes --output <path> last
		return nil, os.WriteFile(out, []byte(payload), 0o600)
	}
}

//fusa:test REQ-FO-ADP002
//fusa:test REQ-FO-ADP005
func TestCheckWithFakeRunner(t *testing.T) {
	a := &cmdAdapter{
		name: "go-FuSa", language: fusaops.LangGo, tool: "gofusa",
		extensions: []string{".go"}, run: fakeRunner(sampleReport),
	}
	findings, err := a.Check(context.Background(), t.TempDir())
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if len(findings) != 3 {
		t.Errorf("got %d findings, want 3", len(findings))
	}
}

//fusa:test REQ-FO-ADP007
//fusa:test REQ-FO-ADP030
func TestRegistryRegisterDuplicate(t *testing.T) {
	r := NewRegistry()
	if err := r.Register(newGoFuSa()); err != nil {
		t.Fatal(err)
	}
	if err := r.Register(newGoFuSa()); err == nil {
		t.Error("expected duplicate registration error")
	}
	if err := r.Register(nil); err == nil {
		t.Error("expected nil registration error")
	}
}

// TestMustRegisterPanic verifies MustRegister panics when the underlying
// Register call returns an error (duplicate tool name).
//
//fusa:test REQ-FO-ADP007
func TestMustRegisterPanic(t *testing.T) {
	r := NewRegistry()
	r.MustRegister(newGoFuSa())
	defer func() {
		if recover() == nil {
			t.Error("MustRegister: expected panic on duplicate tool, got none")
		}
	}()
	r.MustRegister(newGoFuSa()) // must panic
}

//fusa:test REQ-FO-ADP008
func TestRegistryApplicable(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.c"), []byte("int main(){}"), 0o600); err != nil {
		t.Fatal(err)
	}
	r := NewRegistry()
	r.MustRegister(newGoFuSa())
	r.MustRegister(newCFuSa())
	app, err := r.Applicable(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(app) != 1 || app[0].Tool() != "cfusa" {
		t.Errorf("applicable: got %v, want [cfusa]", toolNames(app))
	}
}

//fusa:test REQ-FO-ADP009
//fusa:test REQ-FO-ADP010
//fusa:test REQ-FO-ADP011
//fusa:test REQ-FO-ADP012
//fusa:test REQ-FO-ADP026
//fusa:test REQ-FO-ADP027
//fusa:test REQ-FO-ADP028
//fusa:test REQ-FO-ADP031
func TestDefaultRegistryHasSevenAdapters(t *testing.T) {
	if len(Default.All()) != 7 {
		t.Errorf("default registry: got %d adapters, want 7", len(Default.All()))
	}
}

func toolNames(as []Adapter) []string {
	out := make([]string, len(as))
	for i, a := range as {
		out[i] = a.Tool()
	}
	return out
}

// TestCmdAdapterStandards verifies the Standards method decodes a gap report and
// recomputes the summary from raw objective statuses.
//
//fusa:test REQ-FO-ADP018
//fusa:test REQ-FO-ADP019
func TestCmdAdapterStandards(t *testing.T) {
	const payload = `{"standard":"iso26262","tool":"gofusa","language":"go",
		"objectives":[{"id":"S1","status":"satisfied"},{"id":"G1","status":"gap"}],
		"summary":{"total":2,"satisfied":1,"partial":0,"gaps":1}}`
	a := &cmdAdapter{
		name: "go-FuSa", language: fusaops.LangGo, tool: "gofusa",
		extensions: []string{".go"},
		run: func(_ context.Context, _, _ string, _ ...string) ([]byte, error) {
			return []byte(payload), nil
		},
	}
	r, err := a.Standards(context.Background(), t.TempDir(), "iso26262")
	if err != nil {
		t.Fatalf("Standards: %v", err)
	}
	if r.Standard != "iso26262" {
		t.Errorf("standard: got %q, want iso26262", r.Standard)
	}
	if r.Summary.Gaps != 1 {
		t.Errorf("gaps: got %d, want 1", r.Summary.Gaps)
	}
	if r.Summary.Satisfied != 1 {
		t.Errorf("satisfied: got %d, want 1", r.Summary.Satisfied)
	}
}

// TestCmdAdapterStandardsRunError verifies Standards returns an error when the
// runner itself fails (e.g., tool not installed).
//
//fusa:test REQ-FO-ADP019
func TestCmdAdapterStandardsRunError(t *testing.T) {
	a := &cmdAdapter{
		name: "go-FuSa", language: fusaops.LangGo, tool: "gofusa",
		extensions: []string{".go"},
		run: func(_ context.Context, _, _ string, _ ...string) ([]byte, error) {
			return nil, errors.New("tool not found")
		},
	}
	_, err := a.Standards(context.Background(), t.TempDir(), "iso26262")
	if err == nil {
		t.Error("Standards: expected error when runner fails")
	}
}

// TestCmdAdapterStandardsBadJSON verifies Standards returns an error when the
// runner returns output that cannot be decoded as a GapReport.
//
//fusa:test REQ-FO-ADP019
func TestCmdAdapterStandardsBadJSON(t *testing.T) {
	a := &cmdAdapter{
		name: "go-FuSa", language: fusaops.LangGo, tool: "gofusa",
		extensions: []string{".go"},
		run: func(_ context.Context, _, _ string, _ ...string) ([]byte, error) {
			return []byte("not-valid-json"), nil
		},
	}
	_, err := a.Standards(context.Background(), t.TempDir(), "iso26262")
	if err == nil {
		t.Error("Standards: expected error for malformed JSON output")
	}
}

// TestCheckBadJSONOutput verifies Check returns an error when the runner writes
// malformed JSON to the output file, covering the parseToolReport error branch.
//
//fusa:test REQ-FO-ADP005
func TestCheckBadJSONOutput(t *testing.T) {
	a := &cmdAdapter{
		name: "go-FuSa", language: fusaops.LangGo, tool: "gofusa",
		extensions: []string{".go"},
		run:        fakeRunner("{not valid json}"),
	}
	_, err := a.Check(context.Background(), t.TempDir())
	if err == nil {
		t.Error("Check: expected error for malformed JSON in output file")
	}
}
