package adapter

import (
	"context"
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

func TestParseToolReport(t *testing.T) {
	findings, err := parseToolReport([]byte(sampleReport), fusaops.LangGo, "go-FuSa")
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
	if _, err := parseToolReport([]byte("{not json"), fusaops.LangGo, "t"); err == nil {
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
	if err := os.MkdirAll(buildDir, 0o755); err != nil {
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

func TestDefaultRegistryHasThreeAdapters(t *testing.T) {
	if len(Default.All()) != 3 {
		t.Errorf("default registry: got %d adapters, want 3", len(Default.All()))
	}
}

func toolNames(as []Adapter) []string {
	out := make([]string, len(as))
	for i, a := range as {
		out[i] = a.Tool()
	}
	return out
}
