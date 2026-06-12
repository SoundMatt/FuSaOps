package adapter

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	fusaops "github.com/SoundMatt/FuSaOps"
)

func rustFuSaTestAdapter(run runnerFunc) *cmdAdapter {
	a := newRustFuSa()
	a.run = run
	return a
}

// TestRustFuSaCheck verifies that rsfusa check output is decoded via the generic
// cmdAdapter path (conformant x-FuSa JSON schema: ruleId + location object).
//
//fusa:test REQ-FO-ADP026
func TestRustFuSaCheck(t *testing.T) {
	const report = `{"findings":[
		{"ruleId":"LINT001","severity":"WARNING","message":"unused variable",
		 "location":{"file":"src/lib.rs","line":5},"category":"lint"}
	]}`
	a := rustFuSaTestAdapter(func(_ context.Context, _, _ string, args ...string) ([]byte, error) {
		out := argVal(args, "--output")
		return nil, os.WriteFile(out, []byte(report), 0o600)
	})
	findings, err := a.Check(context.Background(), "/r")
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
	f := findings[0]
	if f.RuleID != "LINT001" {
		t.Errorf("ruleId: got %q, want LINT001", f.RuleID)
	}
	if f.Language != fusaops.LangRust {
		t.Errorf("language: got %q, want %q", f.Language, fusaops.LangRust)
	}
	if f.Location.File != "src/lib.rs" || f.Location.Line != 5 {
		t.Errorf("location: %+v", f.Location)
	}
}

// TestRustFuSaTrace verifies that rsfusa trace --format json stdout is decoded
// via the generic cmdAdapter.Trace path.
//
//fusa:test REQ-FO-ADP026
func TestRustFuSaTrace(t *testing.T) {
	const traceJSON = `{
		"requirements":[{"id":"REQ-001","title":"Safety init","standard":"ISO 26262","level":"HLR","asil":"ASIL-B"}],
		"tags":[
			{"requirementId":"REQ-001","file":"src/init.rs","line":3,"kind":"req"},
			{"requirementId":"REQ-001","file":"tests/init_test.rs","line":9,"kind":"test"}
		],
		"coverage":{"totalRequirements":1,"tracedRequirements":1,"testedRequirements":1}
	}`
	a := rustFuSaTestAdapter(func(_ context.Context, _, _ string, _ ...string) ([]byte, error) {
		return []byte(traceJSON), nil
	})
	m, err := a.Trace(context.Background(), "/r")
	if err != nil {
		t.Fatal(err)
	}
	if m.Coverage.TotalRequirements != 1 || m.Coverage.TracedRequirements != 1 || m.Coverage.TestedRequirements != 1 {
		t.Errorf("coverage wrong: %+v", m.Coverage)
	}
	if len(m.Requirements) != 1 || m.Requirements[0].ID != "REQ-001" {
		t.Errorf("requirements: %+v", m.Requirements)
	}
	if len(m.Tags) != 2 || m.Tags[1].Kind != "test" {
		t.Errorf("tags: %+v", m.Tags)
	}
}

// TestRustFuSaAdapterMetadata verifies adapter name, tool, and language.
//
//fusa:test REQ-FO-ADP026
func TestRustFuSaAdapterMetadata(t *testing.T) {
	a := newRustFuSa()
	if a.Name() != "rust-FuSa" {
		t.Errorf("Name: got %q", a.Name())
	}
	if a.Tool() != "rsfusa" {
		t.Errorf("Tool: got %q", a.Tool())
	}
	if a.Language() != fusaops.LangRust {
		t.Errorf("Language: got %q", a.Language())
	}
}

// TestRustFuSaDetect verifies that .rs files trigger detection.
//
//fusa:test REQ-FO-ADP026
func TestRustFuSaDetect(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.rs"), []byte("fn main(){}"), 0o600); err != nil {
		t.Fatal(err)
	}
	a := newRustFuSa()
	ok, err := a.Detect(dir)
	if err != nil || !ok {
		t.Errorf("Detect rust project: ok=%v err=%v", ok, err)
	}
	// Directory without .rs files must not be detected.
	if ok2, _ := a.Detect(t.TempDir()); ok2 {
		t.Error("Detect empty dir: expected false")
	}
}
