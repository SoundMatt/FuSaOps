package adapter

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	fusaops "github.com/SoundMatt/FuSaOps"
)

func jFuSaTestAdapter(run runnerFunc) *cmdAdapter {
	a := newJFuSa()
	a.run = run
	return a
}

// TestJFuSaCheck verifies that jfusa check output is decoded via the generic
// cmdAdapter path (conformant x-FuSa JSON schema: ruleId + location object).
//
//fusa:test REQ-FO-ADP028
func TestJFuSaCheck(t *testing.T) {
	const report = `{"findings":[
		{"ruleId":"LINT001","severity":"WARNING","message":"return null without //fusa:unsafe",
		 "location":{"file":"src/Main.java","line":42},"category":"safety"}
	]}`
	a := jFuSaTestAdapter(func(_ context.Context, _, _ string, args ...string) ([]byte, error) {
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
	if f.Language != fusaops.LangJava {
		t.Errorf("language: got %q, want %q", f.Language, fusaops.LangJava)
	}
	if f.Location.File != "src/Main.java" || f.Location.Line != 42 {
		t.Errorf("location: %+v", f.Location)
	}
}

// TestJFuSaTrace verifies that jfusa trace --format json stdout is decoded
// via the generic cmdAdapter.Trace path.
//
//fusa:test REQ-FO-ADP028
func TestJFuSaTrace(t *testing.T) {
	const traceJSON = `{
		"requirements":[{"id":"REQ-J001","title":"Null safety","standard":"iso26262","level":"ASIL-B"}],
		"tags":[
			{"requirementId":"REQ-J001","file":"src/Engine.java","line":10,"kind":"impl"},
			{"requirementId":"REQ-J001","file":"src/EngineTest.java","line":20,"kind":"test"}
		],
		"coverage":{"totalRequirements":1,"tracedRequirements":1,"testedRequirements":1}
	}`
	a := jFuSaTestAdapter(func(_ context.Context, _, _ string, _ ...string) ([]byte, error) {
		return []byte(traceJSON), nil
	})
	m, err := a.Trace(context.Background(), "/r")
	if err != nil {
		t.Fatal(err)
	}
	if m.Coverage.TotalRequirements != 1 || m.Coverage.TracedRequirements != 1 {
		t.Errorf("coverage wrong: %+v", m.Coverage)
	}
	if len(m.Requirements) != 1 || m.Requirements[0].ID != "REQ-J001" {
		t.Errorf("requirements: %+v", m.Requirements)
	}
	if len(m.Tags) != 2 || m.Tags[0].Kind != "impl" {
		t.Errorf("tags: %+v", m.Tags)
	}
}

// TestJFuSaAdapterMetadata verifies adapter name, tool, and language.
//
//fusa:test REQ-FO-ADP028
func TestJFuSaAdapterMetadata(t *testing.T) {
	a := newJFuSa()
	if a.Name() != "java-FuSa" {
		t.Errorf("Name: got %q", a.Name())
	}
	if a.Tool() != "jfusa" {
		t.Errorf("Tool: got %q", a.Tool())
	}
	if a.Language() != fusaops.LangJava {
		t.Errorf("Language: got %q, want %q", a.Language(), fusaops.LangJava)
	}
}

// TestJFuSaDetect verifies that .java files trigger detection.
//
//fusa:test REQ-FO-ADP028
func TestJFuSaDetect(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "Main.java"), []byte("class Main {}"), 0o600); err != nil {
		t.Fatal(err)
	}
	a := newJFuSa()
	ok, err := a.Detect(dir)
	if err != nil || !ok {
		t.Errorf("Detect java project: ok=%v err=%v", ok, err)
	}
	if ok2, _ := a.Detect(t.TempDir()); ok2 {
		t.Error("Detect empty dir: expected false")
	}
}
