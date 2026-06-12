package adapter

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	fusaops "github.com/SoundMatt/FuSaOps"
)

func pyFuSaTestAdapter(run runnerFunc) *cmdAdapter {
	a := newPyFuSa()
	a.run = run
	return a
}

// TestPyFuSaCheck verifies that pyfusa check output is decoded via the generic
// cmdAdapter path (conformant x-FuSa JSON schema: ruleId + location object).
//
//fusa:test REQ-FO-ADP027
func TestPyFuSaCheck(t *testing.T) {
	const report = `{"findings":[
		{"ruleId":"PY001","severity":"ERROR","message":"unreachable code",
		 "location":{"file":"src/module.py","line":12},"category":"safety"}
	]}`
	a := pyFuSaTestAdapter(func(_ context.Context, _, _ string, args ...string) ([]byte, error) {
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
	if f.RuleID != "PY001" {
		t.Errorf("ruleId: got %q, want PY001", f.RuleID)
	}
	if f.Language != fusaops.LangPython {
		t.Errorf("language: got %q, want %q", f.Language, fusaops.LangPython)
	}
	if f.Location.File != "src/module.py" || f.Location.Line != 12 {
		t.Errorf("location: %+v", f.Location)
	}
}

// TestPyFuSaTrace verifies that pyfusa trace --format json stdout is decoded
// via the generic cmdAdapter.Trace path.
//
//fusa:test REQ-FO-ADP027
func TestPyFuSaTrace(t *testing.T) {
	const traceJSON = `{
		"requirements":[{"id":"REQ-PY001","title":"Type safety","standard":"IEC 61508","level":"SIL2"}],
		"tags":[
			{"requirementId":"REQ-PY001","file":"src/engine.py","line":7,"kind":"req"},
			{"requirementId":"REQ-PY001","file":"tests/test_engine.py","line":14,"kind":"test"}
		],
		"coverage":{"totalRequirements":1,"tracedRequirements":1,"testedRequirements":1}
	}`
	a := pyFuSaTestAdapter(func(_ context.Context, _, _ string, _ ...string) ([]byte, error) {
		return []byte(traceJSON), nil
	})
	m, err := a.Trace(context.Background(), "/r")
	if err != nil {
		t.Fatal(err)
	}
	if m.Coverage.TotalRequirements != 1 || m.Coverage.TracedRequirements != 1 {
		t.Errorf("coverage wrong: %+v", m.Coverage)
	}
	if len(m.Requirements) != 1 || m.Requirements[0].ID != "REQ-PY001" {
		t.Errorf("requirements: %+v", m.Requirements)
	}
	if len(m.Tags) != 2 || m.Tags[0].Kind != "req" {
		t.Errorf("tags: %+v", m.Tags)
	}
}

// TestPyFuSaAdapterMetadata verifies adapter name, tool, and language.
//
//fusa:test REQ-FO-ADP027
func TestPyFuSaAdapterMetadata(t *testing.T) {
	a := newPyFuSa()
	if a.Name() != "py-FuSa" {
		t.Errorf("Name: got %q", a.Name())
	}
	if a.Tool() != "pyfusa" {
		t.Errorf("Tool: got %q", a.Tool())
	}
	if a.Language() != fusaops.LangPython {
		t.Errorf("Language: got %q, want %q", a.Language(), fusaops.LangPython)
	}
}

// TestPyFuSaDetect verifies that .py files trigger detection.
//
//fusa:test REQ-FO-ADP027
func TestPyFuSaDetect(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.py"), []byte("print('hello')"), 0o600); err != nil {
		t.Fatal(err)
	}
	a := newPyFuSa()
	ok, err := a.Detect(dir)
	if err != nil || !ok {
		t.Errorf("Detect python project: ok=%v err=%v", ok, err)
	}
	// Directory without .py files must not be detected.
	if ok2, _ := a.Detect(t.TempDir()); ok2 {
		t.Error("Detect empty dir: expected false")
	}
}
