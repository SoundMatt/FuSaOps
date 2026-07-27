package coverage

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// --- ParseMCDC tests ---

const emptyLLVM = `{"data":[],"type":"llvm.coverage.json.export","version":"2.0.1"}`

const basicLLVM = `{
  "data": [
    {
      "functions": [
        {
          "name": "myFunc",
          "filenames": ["pkg/myfunc.go"],
          "mcdc_records": [
            {
              "decision_region": [10, 0, 0, 0, 0, 0, 0, 0],
              "conditions": [
                {"id": 0, "covered_true_count": 1, "covered_false_count": 1},
                {"id": 1, "covered_true_count": 1, "covered_false_count": 0}
              ]
            }
          ]
        }
      ]
    }
  ],
  "type": "llvm.coverage.json.export",
  "version": "2.0.1"
}`

const multiFuncLLVM = `{
  "data": [
    {
      "functions": [
        {
          "name": "funcA",
          "filenames": ["pkg/a.go"],
          "mcdc_records": [
            {
              "decision_region": [5, 0, 0, 0, 0, 0, 0, 0],
              "conditions": [
                {"id": 0, "covered_true_count": 1, "covered_false_count": 1}
              ]
            }
          ]
        },
        {
          "name": "funcB",
          "filenames": ["pkg/b.go"],
          "mcdc_records": []
        }
      ]
    }
  ],
  "type": "llvm.coverage.json.export",
  "version": "2.0.1"
}`

//fusa:test REQ-FO-COV005
func TestParseMCDCEmpty(t *testing.T) {
	funcs, err := ParseMCDC(strings.NewReader(emptyLLVM))
	if err != nil {
		t.Fatalf("ParseMCDC empty: %v", err)
	}
	if len(funcs) != 0 {
		t.Errorf("want 0 functions, got %d", len(funcs))
	}
}

//fusa:test REQ-FO-COV005
func TestParseMCDCBasic(t *testing.T) {
	funcs, err := ParseMCDC(strings.NewReader(basicLLVM))
	if err != nil {
		t.Fatalf("ParseMCDC basic: %v", err)
	}
	if len(funcs) != 1 {
		t.Fatalf("want 1 function, got %d", len(funcs))
	}
	fn := funcs[0]
	if fn.Name != "myFunc" {
		t.Errorf("name = %q, want myFunc", fn.Name)
	}
	if fn.File != "pkg/myfunc.go" {
		t.Errorf("file = %q, want pkg/myfunc.go", fn.File)
	}
	if len(fn.Decisions) != 1 {
		t.Fatalf("want 1 decision, got %d", len(fn.Decisions))
	}
	d := fn.Decisions[0]
	if d.Line != 10 {
		t.Errorf("decision line = %d, want 10", d.Line)
	}
	if len(d.Conditions) != 2 {
		t.Fatalf("want 2 conditions, got %d", len(d.Conditions))
	}
	// condition 0: both covered
	c0 := d.Conditions[0]
	if !c0.CoveredT || !c0.CoveredF || !c0.Covered {
		t.Errorf("condition 0: CoveredT=%v CoveredF=%v Covered=%v, want all true", c0.CoveredT, c0.CoveredF, c0.Covered)
	}
	// condition 1: only true covered
	c1 := d.Conditions[1]
	if !c1.CoveredT {
		t.Errorf("condition 1: CoveredT should be true")
	}
	if c1.CoveredF {
		t.Errorf("condition 1: CoveredF should be false (count=0)")
	}
	if c1.Covered {
		t.Errorf("condition 1: Covered should be false")
	}
	// decision not fully covered (condition 1 is not covered)
	if d.Covered {
		t.Error("decision should not be covered (condition 1 uncovered)")
	}
	if fn.Covered {
		t.Error("function should not be covered")
	}
}

//fusa:test REQ-FO-COV005
func TestParseMCDCInvalidJSON(t *testing.T) {
	_, err := ParseMCDC(strings.NewReader("{invalid json"))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

//fusa:test REQ-FO-COV005
func TestParseMCDCMultiFunctions(t *testing.T) {
	funcs, err := ParseMCDC(strings.NewReader(multiFuncLLVM))
	if err != nil {
		t.Fatalf("ParseMCDC multi: %v", err)
	}
	if len(funcs) != 2 {
		t.Errorf("want 2 functions, got %d", len(funcs))
	}
}

// --- FindAnnotatedFunctions tests ---

//fusa:test REQ-FO-COV009
func TestFindAnnotatedFunctionsNone(t *testing.T) {
	dir := t.TempDir()
	src := "package main\n\nfunc Unannotated() {}\n"
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(src), 0o600); err != nil {
		t.Fatal(err)
	}
	m, err := FindAnnotatedFunctions(dir)
	if err != nil {
		t.Fatalf("FindAnnotatedFunctions: %v", err)
	}
	if len(m) != 0 {
		t.Errorf("want empty map, got %v", m)
	}
}

//fusa:test REQ-FO-COV009
func TestFindAnnotatedFunctionsFound(t *testing.T) {
	dir := t.TempDir()
	src := "package main\n\n// MyFunc does something.\n//\n//fusa:req REQ-X\nfunc MyFunc() {}\n"
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(src), 0o600); err != nil {
		t.Fatal(err)
	}
	m, err := FindAnnotatedFunctions(dir)
	if err != nil {
		t.Fatalf("FindAnnotatedFunctions: %v", err)
	}
	if !m["MyFunc"] {
		t.Errorf("MyFunc should be in annotated set; got %v", m)
	}
}

//fusa:test REQ-FO-COV009
func TestFindAnnotatedFunctionsEmptyDir(t *testing.T) {
	dir := t.TempDir()
	m, err := FindAnnotatedFunctions(dir)
	if err != nil {
		t.Fatalf("FindAnnotatedFunctions empty dir: %v", err)
	}
	if len(m) != 0 {
		t.Errorf("want empty map for empty dir, got %v", m)
	}
}

//fusa:test REQ-FO-COV009
func TestFindAnnotatedFunctionsSkipsTests(t *testing.T) {
	dir := t.TempDir()
	// A _test.go file with //fusa:req should be skipped.
	testSrc := "package main\n\n//fusa:req REQ-X\nfunc TestHelper() {}\n"
	if err := os.WriteFile(filepath.Join(dir, "helper_test.go"), []byte(testSrc), 0o600); err != nil {
		t.Fatal(err)
	}
	m, err := FindAnnotatedFunctions(dir)
	if err != nil {
		t.Fatalf("FindAnnotatedFunctions: %v", err)
	}
	if m["TestHelper"] {
		t.Error("TestHelper from _test.go should not be in annotated set")
	}
}

// --- AnalyseMCDC tests ---

//fusa:test REQ-FO-COV006
func TestAnalyseMCDCEmpty(t *testing.T) {
	rep := AnalyseMCDC(nil, nil, DALA, DefaultMcdcThreshold)
	if rep.TotalConds != 0 {
		t.Errorf("TotalConds = %d, want 0", rep.TotalConds)
	}
	if rep.CondPct != 0 {
		t.Errorf("CondPct = %.1f, want 0", rep.CondPct)
	}
	if !rep.GatePassed {
		t.Error("GatePassed should be true for empty (vacuous pass)")
	}
}

//fusa:test REQ-FO-COV006
func TestAnalyseMCDCGatePass(t *testing.T) {
	funcs := []McdcFunction{
		{
			Name:    "SafeFunc",
			File:    "pkg/safe.go",
			Covered: true,
			Decisions: []McdcDecision{
				{
					File:    "pkg/safe.go",
					Line:    10,
					Covered: true,
					Conditions: []McdcCondition{
						{ID: 0, CoveredT: true, CoveredF: true, Covered: true},
					},
				},
			},
		},
	}
	annotated := map[string]bool{"SafeFunc": true}
	rep := AnalyseMCDC(funcs, annotated, DALA, DefaultMcdcThreshold)
	if !rep.GatePassed {
		t.Error("GatePassed should be true: all conditions covered")
	}
	if len(rep.UncoveredReqs) != 0 {
		t.Errorf("UncoveredReqs should be empty, got %v", rep.UncoveredReqs)
	}
	if rep.CondPct != 100.0 {
		t.Errorf("CondPct = %.1f, want 100.0", rep.CondPct)
	}
}

//fusa:test REQ-FO-COV006
func TestAnalyseMCDCGateFailAnnotated(t *testing.T) {
	funcs := []McdcFunction{
		{
			Name:    "AnnotatedUncovered",
			File:    "pkg/foo.go",
			Covered: false,
			Decisions: []McdcDecision{
				{
					File:    "pkg/foo.go",
					Line:    5,
					Covered: false,
					Conditions: []McdcCondition{
						{ID: 0, CoveredT: true, CoveredF: false, Covered: false},
					},
				},
			},
		},
	}
	annotated := map[string]bool{"AnnotatedUncovered": true}
	rep := AnalyseMCDC(funcs, annotated, DALA, DefaultMcdcThreshold)
	if rep.GatePassed {
		t.Error("GatePassed should be false: annotated function has uncovered condition")
	}
	if len(rep.UncoveredReqs) != 1 {
		t.Errorf("want 1 UncoveredReq, got %d: %v", len(rep.UncoveredReqs), rep.UncoveredReqs)
	}
}

//fusa:test REQ-FO-COV006
func TestAnalyseMCDCGatePassNonAnnotated(t *testing.T) {
	// Non-annotated function with uncovered condition — gate only fails on threshold.
	funcs := []McdcFunction{
		{
			Name:    "NonAnnotated",
			File:    "pkg/noann.go",
			Covered: false,
			Decisions: []McdcDecision{
				{
					File:    "pkg/noann.go",
					Line:    3,
					Covered: false,
					Conditions: []McdcCondition{
						{ID: 0, CoveredT: true, CoveredF: true, Covered: true},
						{ID: 1, CoveredT: false, CoveredF: false, Covered: false},
					},
				},
			},
		},
	}
	// Not annotated → UncoveredReqs stays empty.
	// CondPct = 1/2 = 50% < 100% threshold → GatePassed = false.
	annotated := map[string]bool{}
	rep := AnalyseMCDC(funcs, annotated, DALA, DefaultMcdcThreshold)
	if len(rep.UncoveredReqs) != 0 {
		t.Errorf("UncoveredReqs should be empty for non-annotated, got %v", rep.UncoveredReqs)
	}
	if rep.GatePassed {
		t.Error("GatePassed should be false: CondPct < threshold")
	}
}

//fusa:test REQ-FO-COV006
func TestAnalyseMCDCThreshold(t *testing.T) {
	// 90% coverage but threshold is 95% — gate should fail on threshold alone.
	funcs := []McdcFunction{
		{
			Name:    "SomeFunc",
			File:    "pkg/some.go",
			Covered: false,
			Decisions: []McdcDecision{
				{
					File:    "pkg/some.go",
					Line:    1,
					Covered: false,
					Conditions: []McdcCondition{
						{ID: 0, CoveredT: true, CoveredF: true, Covered: true},
						{ID: 1, CoveredT: true, CoveredF: true, Covered: true},
						{ID: 2, CoveredT: true, CoveredF: true, Covered: true},
						{ID: 3, CoveredT: true, CoveredF: true, Covered: true},
						{ID: 4, CoveredT: true, CoveredF: true, Covered: true},
						{ID: 5, CoveredT: true, CoveredF: true, Covered: true},
						{ID: 6, CoveredT: true, CoveredF: true, Covered: true},
						{ID: 7, CoveredT: true, CoveredF: true, Covered: true},
						{ID: 8, CoveredT: true, CoveredF: true, Covered: true},
						{ID: 9, CoveredT: false, CoveredF: false, Covered: false}, // 9/10 = 90%
					},
				},
			},
		},
	}
	annotated := map[string]bool{}
	rep := AnalyseMCDC(funcs, annotated, DALB, 95.0)
	if rep.GatePassed {
		t.Errorf("GatePassed should be false: CondPct=%.1f < threshold=95.0", rep.CondPct)
	}
}

// --- GateMCDC tests ---

//fusa:test REQ-FO-COV007
func TestGateMCDCPass(t *testing.T) {
	rep := &McdcReport{GatePassed: true}
	if !GateMCDC(rep) {
		t.Error("GateMCDC: should return true for GatePassed=true")
	}
}

//fusa:test REQ-FO-COV007
func TestGateMCDCFail(t *testing.T) {
	rep := &McdcReport{GatePassed: false}
	if GateMCDC(rep) {
		t.Error("GateMCDC: should return false for GatePassed=false")
	}
}

// --- RenderMCDC tests ---

func makeTestReport(gatePassed bool) *McdcReport {
	return &McdcReport{
		Generated:    mustParseTime("2026-07-25T12:00:00Z"),
		DAL:          DALA,
		ProfileMode:  "llvm-mcdc",
		Threshold:    100.0,
		TotalConds:   2,
		CoveredConds: 1,
		CondPct:      50.0,
		GatePassed:   gatePassed,
		Functions: []McdcFunction{
			{Name: "foo", File: "pkg/foo.go", HasReqTag: true, Covered: false},
			{Name: "bar", File: "pkg/bar.go", HasReqTag: false, Covered: true},
		},
		UncoveredReqs: []string{"foo (pkg/foo.go): 1 uncovered condition(s)"},
	}
}

func mustParseTime(s string) time.Time {
	// Use a fixed time for deterministic test output.
	t, _ := time.Parse(time.RFC3339, s)
	return t
}

//fusa:test REQ-FO-COV008
func TestRenderMCDCText(t *testing.T) {
	rep := makeTestReport(false)
	var buf bytes.Buffer
	if err := RenderMCDC(&buf, rep, "text"); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "DO-178C MC/DC Coverage Report") {
		t.Error("text missing DO-178C MC/DC Coverage Report header")
	}
	if !strings.Contains(out, "DAL-A") {
		t.Error("text missing DAL-A")
	}
	if !strings.Contains(out, "50.0%") {
		t.Errorf("text missing condition pct: %q", out)
	}
	if !strings.Contains(out, "FAIL") {
		t.Error("text missing FAIL gate status")
	}
}

//fusa:test REQ-FO-COV008
//fusa:test REQ-FO-COV004
func TestRenderMCDCJSON(t *testing.T) {
	rep := makeTestReport(true)
	var buf bytes.Buffer
	if err := RenderMCDC(&buf, rep, "json"); err != nil {
		t.Fatal(err)
	}
	var decoded McdcReport
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if !strings.Contains(buf.String(), `"gatePassed"`) {
		t.Error("JSON missing gatePassed key")
	}
}

//fusa:test REQ-FO-COV008
func TestRenderMCDCMarkdown(t *testing.T) {
	rep := makeTestReport(false)
	var buf bytes.Buffer
	if err := RenderMCDC(&buf, rep, "markdown"); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "# FuSaOps") {
		t.Error("markdown missing # FuSaOps heading")
	}
	if !strings.Contains(out, "| Function |") {
		t.Error("markdown missing table header row")
	}
}

//fusa:test REQ-FO-COV008
func TestRenderMCDCMarkdownAlias(t *testing.T) {
	rep := makeTestReport(true)
	var buf bytes.Buffer
	if err := RenderMCDC(&buf, rep, "md"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "DO-178C") {
		t.Error("md alias missing DO-178C content")
	}
}

//fusa:test REQ-FO-COV008
func TestRenderMCDCUnknownFormat(t *testing.T) {
	rep := makeTestReport(true)
	err := RenderMCDC(&bytes.Buffer{}, rep, "xml")
	if err == nil {
		t.Error("expected error for unknown format")
	}
}

//fusa:test REQ-FO-COV008
func TestRenderMCDCJSONDefault(t *testing.T) {
	rep := makeTestReport(true)
	var buf bytes.Buffer
	if err := RenderMCDC(&buf, rep, ""); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "{") {
		t.Error("empty format should default to JSON")
	}
}
