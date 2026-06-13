package adapter

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/SoundMatt/FuSaOps/standards"
)

func cppFuSaTestAdapter(run runnerFunc) *cppFuSaAdapter {
	a := newCppFuSa()
	a.run = run
	return a
}

// TestCppFuSaTrace verifies cpp-FuSa trace uses the generic cmdAdapter path
// (stdout) with the v0.12.4 canonical format: kind "trace-matrix", flat tags[],
// coverage block, and requirements[].standard key (not "standardRef").
//
//fusa:test REQ-FO-ADP014
func TestCppFuSaTrace(t *testing.T) {
	const traceJSON = `{
		"kind": "trace-matrix",
		"requirements": [
			{"id": "REQ-001", "title": "Req one", "standard": "iso26262"},
			{"id": "REQ-002", "title": "Req two", "standard": ""}
		],
		"tags": [
			{"requirementId": "REQ-001", "file": "src/a.cpp",       "line": 10, "kind": "impl"},
			{"requirementId": "REQ-001", "file": "test/a_test.cpp", "line": 20, "kind": "test"},
			{"requirementId": "REQ-002", "file": "src/b.cpp",       "line":  5, "kind": "impl"},
			{"requirementId": "REQ-002", "file": "src/c.cpp",       "line":  7, "kind": "impl"}
		],
		"coverage": {"totalRequirements": 2, "tracedRequirements": 2, "testedRequirements": 1, "secTestedRequirements": 0}
	}`

	a := cppFuSaTestAdapter(func(_ context.Context, _, _ string, _ ...string) ([]byte, error) {
		return []byte(traceJSON), nil
	})

	m, err := a.Trace(context.Background(), "/r")
	if err != nil {
		t.Fatal(err)
	}
	if m.Coverage.TotalRequirements != 2 {
		t.Errorf("TotalRequirements: got %d, want 2", m.Coverage.TotalRequirements)
	}
	if m.Coverage.TracedRequirements != 2 {
		t.Errorf("TracedRequirements: got %d, want 2", m.Coverage.TracedRequirements)
	}
	if m.Coverage.TestedRequirements != 1 {
		t.Errorf("TestedRequirements: got %d, want 1", m.Coverage.TestedRequirements)
	}
	if len(m.Requirements) != 2 {
		t.Fatalf("requirements: got %d, want 2", len(m.Requirements))
	}
	if m.Requirements[0].ID != "REQ-001" || m.Requirements[0].Title != "Req one" || m.Requirements[0].Standard != "iso26262" {
		t.Errorf("req[0]: %+v", m.Requirements[0])
	}
	if len(m.Tags) != 4 {
		t.Fatalf("tags: got %d, want 4", len(m.Tags))
	}
	if m.Tags[0].Kind != "impl" || m.Tags[0].File != "src/a.cpp" || m.Tags[0].Line != 10 {
		t.Errorf("tag[0]: %+v", m.Tags[0])
	}
	if m.Tags[1].Kind != "test" {
		t.Errorf("tag[1].Kind = %q, want test", m.Tags[1].Kind)
	}
}

// TestCppFuSaStandards verifies cpp-FuSa standards normalization: JSON written
// to --output file rather than stdout; RecomputeSummary called.
//
//fusa:test REQ-FO-ADP025
func TestCppFuSaStandards(t *testing.T) {
	report := standards.GapReport{
		Standard: "iso26262",
		Objectives: []standards.Objective{
			{ID: "7-1", Title: "Software safety lifecycle", Status: "satisfied"},
			{ID: "7-4", Title: "Software architectural design", Status: "gap"},
		},
		Summary: standards.Summary{Total: 2, Satisfied: 1, Gaps: 1},
	}
	data, _ := json.Marshal(report)

	a := cppFuSaTestAdapter(func(_ context.Context, _, _ string, args ...string) ([]byte, error) {
		out := argVal(args, "--output")
		return nil, os.WriteFile(out, data, 0o600)
	})

	r, err := a.Standards(context.Background(), "/r", "iso26262")
	if err != nil {
		t.Fatal(err)
	}
	if r.Standard != "iso26262" {
		t.Errorf("standard: got %q, want iso26262", r.Standard)
	}
	if len(r.Objectives) != 2 {
		t.Errorf("objectives: got %d, want 2", len(r.Objectives))
	}
}

// TestCppFuSaStandardsDo178c verifies do178c → do178 subcommand mapping.
//
//fusa:test REQ-FO-ADP025
func TestCppFuSaStandardsDo178c(t *testing.T) {
	var capturedArgs []string
	a := cppFuSaTestAdapter(func(_ context.Context, _, _ string, args ...string) ([]byte, error) {
		capturedArgs = args
		out := argVal(args, "--output")
		return nil, os.WriteFile(out, []byte(`{"standard":"do178c"}`), 0o600)
	})

	if _, err := a.Standards(context.Background(), "/r", "do178c"); err != nil {
		t.Fatal(err)
	}
	if len(capturedArgs) == 0 || capturedArgs[0] != "do178" {
		t.Errorf("expected subcommand do178, got args %v", capturedArgs)
	}
}

// TestCppStandardCmd verifies the canonical→cpfusa subcommand name mapping.
//
//fusa:test REQ-FO-ADP025
func TestCppStandardCmd(t *testing.T) {
	cases := map[string]string{
		"do178c":   "do178",
		"iso26262": "iso26262",
		"iec61508": "iec61508",
		"iec62443": "iec62443",
		"slsa":     "slsa",
		"iso21434": "iso21434",
		"unece":    "unece",
	}
	for in, want := range cases {
		if got := cppStandardCmd(in); got != want {
			t.Errorf("cppStandardCmd(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestCppFuSaStandardsRunError verifies that a runner failure is propagated.
//
//fusa:test REQ-FO-ADP025
func TestCppFuSaStandardsRunError(t *testing.T) {
	a := cppFuSaTestAdapter(func(_ context.Context, _, _ string, _ ...string) ([]byte, error) {
		return nil, context.DeadlineExceeded
	})
	if _, err := a.Standards(context.Background(), "/r", "iso26262"); err == nil {
		t.Error("expected error when runner fails")
	}
}

// TestCppFuSaStandardsFileNotWritten verifies error when runner succeeds but
// writes nothing to --output (tool bug scenario).
//
//fusa:test REQ-FO-ADP025
func TestCppFuSaStandardsFileNotWritten(t *testing.T) {
	a := cppFuSaTestAdapter(func(_ context.Context, _, _ string, _ ...string) ([]byte, error) {
		return nil, nil // runner succeeds but never writes the output file
	})
	if _, err := a.Standards(context.Background(), "/r", "iso26262"); err == nil {
		t.Error("expected error when output file is not written")
	}
}

// TestCppFuSaAdapterRegistration verifies newCppFuSa returns a valid adapter.
func TestCppFuSaAdapterRegistration(t *testing.T) {
	a := newCppFuSa()
	if a.Name() != "cpp-FuSa" {
		t.Errorf("Name: got %q", a.Name())
	}
	if a.Tool() != "cpfusa" {
		t.Errorf("Tool: got %q", a.Tool())
	}
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "main.cpp"), []byte("int main(){}"), 0o600)
	ok, err := a.Detect(dir)
	if err != nil || !ok {
		t.Errorf("Detect cpp project: ok=%v err=%v", ok, err)
	}
}
