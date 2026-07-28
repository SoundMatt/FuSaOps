package adapter

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	fusaops "github.com/SoundMatt/FuSaOps"
)

func adaFuSaTestAdapter(run runnerFunc) *cmdAdapter {
	a := newAdaFuSa()
	a.run = run
	return a
}

// TestAdaFuSaCheck verifies that adafusa check output is decoded via the
// generic cmdAdapter path (conformant x-FuSa JSON schema: ruleId + location
// object).
//
//fusa:test REQ-FO-ADP031
func TestAdaFuSaCheck(t *testing.T) {
	const report = `{"findings":[
		{"ruleId":"ADA001","severity":"ERROR","message":"pragma Suppress used without a fusa:unsafe justification",
		 "location":{"file":"src/engine.adb","line":12},"category":"safety"}
	]}`
	a := adaFuSaTestAdapter(func(_ context.Context, _, _ string, args ...string) ([]byte, error) {
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
	if f.RuleID != "ADA001" {
		t.Errorf("ruleId: got %q, want ADA001", f.RuleID)
	}
	if f.Language != fusaops.LangAda {
		t.Errorf("language: got %q, want %q", f.Language, fusaops.LangAda)
	}
	if f.Location.File != "src/engine.adb" || f.Location.Line != 12 {
		t.Errorf("location: %+v", f.Location)
	}
}

// TestAdaFuSaTrace verifies that adafusa trace --format json stdout is
// decoded via the generic cmdAdapter.Trace path.
//
//fusa:test REQ-FO-ADP031
func TestAdaFuSaTrace(t *testing.T) {
	const traceJSON = `{
		"requirements":[{"id":"REQ-A001","title":"Bounds checking","standard":"iso26262","level":"ASIL-B"}],
		"tags":[
			{"requirementId":"REQ-A001","file":"src/engine.adb","line":10,"kind":"impl"},
			{"requirementId":"REQ-A001","file":"tests/engine_tests.adb","line":20,"kind":"test"}
		],
		"coverage":{"totalRequirements":1,"tracedRequirements":1,"testedRequirements":1}
	}`
	a := adaFuSaTestAdapter(func(_ context.Context, _, _ string, _ ...string) ([]byte, error) {
		return []byte(traceJSON), nil
	})
	m, err := a.Trace(context.Background(), "/r")
	if err != nil {
		t.Fatal(err)
	}
	if m.Coverage.TotalRequirements != 1 || m.Coverage.TracedRequirements != 1 {
		t.Errorf("coverage wrong: %+v", m.Coverage)
	}
	if len(m.Requirements) != 1 || m.Requirements[0].ID != "REQ-A001" {
		t.Errorf("requirements: %+v", m.Requirements)
	}
	if len(m.Tags) != 2 || m.Tags[0].Kind != "impl" {
		t.Errorf("tags: %+v", m.Tags)
	}
}

// TestAdaFuSaAdapterMetadata verifies adapter name, tool, and language.
//
//fusa:test REQ-FO-ADP031
func TestAdaFuSaAdapterMetadata(t *testing.T) {
	a := newAdaFuSa()
	if a.Name() != "ada-FuSa" {
		t.Errorf("Name: got %q", a.Name())
	}
	if a.Tool() != "adafusa" {
		t.Errorf("Tool: got %q", a.Tool())
	}
	if a.Language() != fusaops.LangAda {
		t.Errorf("Language: got %q, want %q", a.Language(), fusaops.LangAda)
	}
}

// TestAdaFuSaDetect verifies that .ads/.adb files trigger detection.
//
//fusa:test REQ-FO-ADP031
func TestAdaFuSaDetect(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "engine.ads"), []byte("package Engine is\nend Engine;"), 0o600); err != nil {
		t.Fatal(err)
	}
	a := newAdaFuSa()
	ok, err := a.Detect(dir)
	if err != nil || !ok {
		t.Errorf("Detect ada project: ok=%v err=%v", ok, err)
	}
	if ok2, _ := a.Detect(t.TempDir()); ok2 {
		t.Error("Detect empty dir: expected false")
	}
}
