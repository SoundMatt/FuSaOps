package orchestrator

import (
	"archive/zip"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	fusaops "github.com/SoundMatt/FuSaOps"
	"github.com/SoundMatt/FuSaOps/comp"
	"github.com/SoundMatt/FuSaOps/mcdc"
	"github.com/SoundMatt/FuSaOps/sbom"
	"github.com/SoundMatt/FuSaOps/standards"
	"github.com/SoundMatt/FuSaOps/trace"
)

// capFake is a fully capable fake adapter: it implements Adapter plus every
// capability interface, with injectable per-capability errors.
type capFake struct {
	tool         string
	lang         fusaops.Language
	detect       bool
	avail        bool
	matrix       *trace.Matrix
	qual         *trace.Qualification
	doc          *sbom.Document
	gapReport    *standards.GapReport
	compReport   *comp.Report
	mcdcReport   *mcdc.Report
	traceErr     error
	sbomErr      error
	packErr      error
	standardsErr error
	compErr      error
	mcdcErr      error
}

func (f *capFake) Name() string                                             { return f.tool }
func (f *capFake) Language() fusaops.Language                               { return f.lang }
func (f *capFake) Tool() string                                             { return f.tool }
func (f *capFake) Detect(string) (bool, error)                              { return f.detect, nil }
func (f *capFake) Available() bool                                          { return f.avail }
func (f *capFake) Check(context.Context, string) ([]fusaops.Finding, error) { return nil, nil }

func (f *capFake) Trace(context.Context, string) (*trace.Matrix, error) {
	return f.matrix, f.traceErr
}
func (f *capFake) Qualify(context.Context, string) (*trace.Qualification, error) {
	return f.qual, nil
}
func (f *capFake) SBOM(context.Context, string) (*sbom.Document, error) {
	return f.doc, f.sbomErr
}
func (f *capFake) AuditPack(_ context.Context, _, dest string) error {
	if f.packErr != nil {
		return f.packErr
	}
	return os.WriteFile(dest, []byte("PK\x03\x04 fake pack"), 0o600)
}

func (f *capFake) Standards(_ context.Context, _, _ string) (*standards.GapReport, error) {
	return f.gapReport, f.standardsErr
}

func (f *capFake) Comp(_ context.Context, _ string, _ int, _ string) (*comp.Report, error) {
	return f.compReport, f.compErr
}

func (f *capFake) MCDC(_ context.Context, _ string) (*mcdc.Report, error) {
	return f.mcdcReport, f.mcdcErr
}

func tracer(tool string) *capFake {
	return &capFake{
		tool: tool, lang: fusaops.LangGo, detect: true, avail: true,
		matrix: &trace.Matrix{Coverage: trace.Coverage{TotalRequirements: 4, TracedRequirements: 4, TestedRequirements: 3}},
		qual:   &trace.Qualification{Total: 2, Passed: 2},
		doc:    &sbom.Document{Module: "m-" + tool, Components: []sbom.Package{{Name: "dep", Version: "v1"}}},
		gapReport: &standards.GapReport{
			Standard: "iso26262",
			Summary:  standards.Summary{Total: 10, Satisfied: 9, Partial: 1, Gaps: 0},
		},
		compReport: &comp.Report{Threshold: 10, TotalFunctions: 8, Violations: 1,
			Results: []comp.Function{{File: "x.go", Name: "F", Complexity: 12, ExceedsThreshold: true}}},
		mcdcReport: &mcdc.Report{TotalConditions: 10, CoveredConditions: 8},
	}
}

//fusa:test REQ-FO-ORC004
func TestRunTrace(t *testing.T) {
	reg := regWith(
		tracer("gofusa"),
		&capFake{tool: "cfusa", lang: fusaops.LangC, detect: true, avail: false},     // skipped: not installed
		&fakeAdapter{tool: "nope", lang: fusaops.LangCpp, detect: true, avail: true}, // not a Tracer
	)
	agg, err := New(reg).RunTrace(context.Background(), t.TempDir(), Options{Project: "p"})
	if err != nil {
		t.Fatal(err)
	}
	if len(agg.Components) != 3 {
		t.Fatalf("want 3 components, got %d", len(agg.Components))
	}
	if agg.Coverage.TotalRequirements != 4 || agg.Coverage.TestedRequirements != 3 {
		t.Errorf("aggregate coverage wrong: %+v", agg.Coverage)
	}
	// gofusa carries qualification.
	var gofusa trace.ComponentTrace
	for _, c := range agg.Components {
		switch c.Tool {
		case "gofusa":
			gofusa = c
		case "cfusa":
			if c.Skipped == "" {
				t.Error("cfusa should be skipped (not installed)")
			}
		case "nope":
			if c.Skipped == "" {
				t.Error("nope should be skipped (not a Tracer)")
			}
		}
	}
	if gofusa.Qualification == nil || gofusa.Qualification.Passed != 2 {
		t.Errorf("gofusa qualification missing: %+v", gofusa)
	}
}

//fusa:test REQ-FO-ORC004
func TestRunTraceErrorsAndEmpty(t *testing.T) {
	// trace failure recorded as skipped, not fatal.
	bad := tracer("gofusa")
	bad.traceErr = errors.New("boom")
	agg, err := New(regWith(bad)).RunTrace(context.Background(), t.TempDir(), Options{})
	if err != nil {
		t.Fatal(err)
	}
	if agg.Components[0].Skipped == "" {
		t.Error("trace error should be recorded as skipped")
	}
	// No applicable adapters → ErrNoAdapters.
	none := regWith(&capFake{tool: "gofusa", detect: false, avail: true})
	if _, err := New(none).RunTrace(context.Background(), t.TempDir(), Options{}); !errors.Is(err, fusaops.ErrNoAdapters) {
		t.Errorf("got %v, want ErrNoAdapters", err)
	}
}

//fusa:test REQ-FO-ORC005
func TestRunSBOM(t *testing.T) {
	reg := regWith(
		tracer("gofusa"),
		&capFake{tool: "cfusa", lang: fusaops.LangC, detect: true, avail: false},
		&fakeAdapter{tool: "nope", lang: fusaops.LangCpp, detect: true, avail: true},
	)
	agg, err := New(reg).RunSBOM(context.Background(), t.TempDir(), Options{Project: "p"})
	if err != nil {
		t.Fatal(err)
	}
	if agg.TotalPackages != 1 || len(agg.Packages) != 1 {
		t.Errorf("want 1 merged package, got %+v", agg.Packages)
	}
	if len(agg.Components) != 3 {
		t.Errorf("want 3 components, got %d", len(agg.Components))
	}
}

//fusa:test REQ-FO-ORC005
func TestRunSBOMErrorsAndOnly(t *testing.T) {
	bad := tracer("gofusa")
	bad.sbomErr = errors.New("boom")
	reg := regWith(bad, tracer("cfusa"))
	// Only filter narrows to cfusa.
	agg, err := New(reg).RunSBOM(context.Background(), t.TempDir(), Options{Only: []string{"cfusa"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(agg.Components) != 1 || agg.Components[0].Tool != "cfusa" {
		t.Errorf("only filter failed: %+v", agg.Components)
	}
	// All applicable but sbom errors → component skipped, no packages.
	aggBad, _ := New(regWith(bad)).RunSBOM(context.Background(), t.TempDir(), Options{})
	if aggBad.Components[0].Skipped == "" {
		t.Error("sbom error should be recorded as skipped")
	}
}

//fusa:test REQ-FO-ORC006
func TestRunAuditPack(t *testing.T) {
	reg := regWith(
		tracer("gofusa"),
		&fakeAdapter{tool: "nope", lang: fusaops.LangCpp, detect: true, avail: true}, // not a Packer
	)
	dest := filepath.Join(t.TempDir(), "audit-pack.zip")
	res, err := New(reg).RunAuditPack(context.Background(), t.TempDir(), dest, Options{Project: "p"})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Packed) != 1 || res.Packed[0] != "gofusa" {
		t.Errorf("packed wrong: %+v", res.Packed)
	}
	if len(res.Skipped) != 1 {
		t.Errorf("nope should be skipped: %+v", res.Skipped)
	}

	// The bundle contains the per-tool pack and all FuSaOps-level evidence.
	names := zipNames(t, dest)
	for _, want := range []string{"components/gofusa/audit-pack.zip", "report.json", "trace.json", "sbom.json", "manifest.json"} {
		if !names[want] {
			t.Errorf("audit-pack missing %q (have %v)", want, names)
		}
	}
}

//fusa:test REQ-FO-ORC006
func TestRunAuditPackErrorsAndEmpty(t *testing.T) {
	// Packer that fails → recorded as skipped, FuSaOps evidence still bundled.
	bad := tracer("gofusa")
	bad.packErr = errors.New("pack boom")
	dest := filepath.Join(t.TempDir(), "p.zip")
	res, err := New(regWith(bad)).RunAuditPack(context.Background(), t.TempDir(), dest, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Packed) != 0 || len(res.Skipped) != 1 {
		t.Errorf("expected pack failure recorded: %+v", res)
	}
	if !zipNames(t, dest)["report.json"] {
		t.Error("FuSaOps evidence should still be bundled")
	}

	// No applicable adapters → ErrNoAdapters.
	none := regWith(&capFake{tool: "gofusa", detect: false, avail: true})
	if _, err := New(none).RunAuditPack(context.Background(), t.TempDir(), dest, Options{}); !errors.Is(err, fusaops.ErrNoAdapters) {
		t.Errorf("got %v, want ErrNoAdapters", err)
	}
}

//fusa:test REQ-FO-ORC007
func TestRunStandards(t *testing.T) {
	reg := regWith(
		tracer("gofusa"),
		&capFake{tool: "cfusa", lang: fusaops.LangC, detect: true, avail: false},     // skipped: not installed
		&fakeAdapter{tool: "nope", lang: fusaops.LangCpp, detect: true, avail: true}, // not a StandardsProvider
	)
	agg, err := New(reg).RunStandards(context.Background(), t.TempDir(), "iso26262", Options{Project: "p"})
	if err != nil {
		t.Fatal(err)
	}
	if len(agg.Components) != 3 {
		t.Fatalf("want 3 components, got %d", len(agg.Components))
	}
	if agg.Standard != "iso26262" {
		t.Errorf("standard wrong: %s", agg.Standard)
	}
	var gofusa standards.ComponentGap
	for _, c := range agg.Components {
		switch c.Tool {
		case "gofusa":
			gofusa = c
		case "cfusa":
			if c.Skipped == "" {
				t.Error("cfusa should be skipped (not installed)")
			}
		case "nope":
			if c.Skipped == "" {
				t.Error("nope should be skipped (not a StandardsProvider)")
			}
		}
	}
	if gofusa.Report == nil {
		t.Fatal("gofusa gap report missing")
	}
	if gofusa.Report.Summary.Total != 10 {
		t.Errorf("gofusa summary wrong: %+v", gofusa.Report.Summary)
	}
	if agg.HasGaps() {
		t.Error("no gaps expected in this aggregate")
	}
}

//fusa:test REQ-FO-ORC007
func TestRunStandardsErrorsAndEmpty(t *testing.T) {
	// standards failure recorded as skipped, not fatal.
	bad := tracer("gofusa")
	bad.standardsErr = errors.New("boom")
	agg, err := New(regWith(bad)).RunStandards(context.Background(), t.TempDir(), "iso26262", Options{})
	if err != nil {
		t.Fatal(err)
	}
	if agg.Components[0].Skipped == "" {
		t.Error("standards error should be recorded as skipped")
	}
	// No applicable adapters → ErrNoAdapters.
	none := regWith(&capFake{tool: "gofusa", detect: false, avail: true})
	if _, err := New(none).RunStandards(context.Background(), t.TempDir(), "iso26262", Options{}); !errors.Is(err, fusaops.ErrNoAdapters) {
		t.Errorf("got %v, want ErrNoAdapters", err)
	}
}

func zipNames(t *testing.T, path string) map[string]bool {
	t.Helper()
	zr, err := zip.OpenReader(path)
	if err != nil {
		t.Fatalf("open zip: %v", err)
	}
	defer func() { _ = zr.Close() }()
	out := map[string]bool{}
	for _, f := range zr.File {
		out[f.Name] = true
	}
	return out
}

//fusa:test REQ-FO-ORC008
func TestRunTraceWithWorkers(t *testing.T) {
	a := tracer("gofusa")
	b := tracer("cfusa")
	agg, err := New(regWith(a, b)).RunTrace(context.Background(), t.TempDir(), Options{Workers: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(agg.Components) != 2 {
		t.Errorf("expected 2 trace components with workers=2, got %d", len(agg.Components))
	}
}

//fusa:test REQ-FO-ORC008
func TestRunSBOMWithWorkers(t *testing.T) {
	a := tracer("gofusa")
	b := tracer("cfusa")
	agg, err := New(regWith(a, b)).RunSBOM(context.Background(), t.TempDir(), Options{Workers: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(agg.Components) != 2 {
		t.Errorf("expected 2 sbom components with workers=2, got %d", len(agg.Components))
	}
}

//fusa:test REQ-FO-ORC013
func TestRunComp(t *testing.T) {
	reg := regWith(
		tracer("gofusa"),
		&capFake{tool: "cfusa", lang: fusaops.LangC, detect: true, avail: false},     // skipped: not installed
		&fakeAdapter{tool: "nope", lang: fusaops.LangCpp, detect: true, avail: true}, // not a Compler
	)
	agg, err := New(reg).RunComp(context.Background(), t.TempDir(), Options{Project: "p"}, 0, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(agg.Components) != 3 {
		t.Fatalf("components = %d, want 3", len(agg.Components))
	}
	// Components are sorted by tool name: cfusa, gofusa, nope.
	byTool := make(map[string]comp.ComponentComp, 3)
	for _, c := range agg.Components {
		byTool[c.Tool] = c
	}
	// gofusa: has comp report
	if byTool["gofusa"].Report == nil {
		t.Error("gofusa component should have a report")
	}
	if agg.TotalFunctions != 8 || agg.Violations != 1 {
		t.Errorf("aggregate wrong: funcs=%d violations=%d", agg.TotalFunctions, agg.Violations)
	}
	// cfusa: skipped (not installed)
	if byTool["cfusa"].Skipped == "" {
		t.Error("cfusa should be skipped")
	}
	// nope: skipped (not a Compler)
	if byTool["nope"].Skipped == "" {
		t.Error("nope should be skipped (not a Compler)")
	}
}

//fusa:test REQ-FO-ORC013
func TestRunCompErrorRecordedAsSkipped(t *testing.T) {
	bad := &capFake{tool: "gofusa", lang: fusaops.LangGo, detect: true, avail: true, compErr: errors.New("comp boom")}
	agg, err := New(regWith(bad)).RunComp(context.Background(), t.TempDir(), Options{}, 0, "")
	if err != nil {
		t.Fatal(err)
	}
	if agg.Components[0].Skipped == "" {
		t.Error("expected comp error to be recorded as skipped")
	}
}

//fusa:test REQ-FO-ORC013
func TestRunCompNoAdapters(t *testing.T) {
	none := regWith(&capFake{tool: "gofusa", detect: false, avail: true})
	if _, err := New(none).RunComp(context.Background(), t.TempDir(), Options{}, 0, ""); !errors.Is(err, fusaops.ErrNoAdapters) {
		t.Errorf("expected ErrNoAdapters, got %v", err)
	}
}

//fusa:test REQ-FO-MCDC002
func TestRunMCDC(t *testing.T) {
	reg := regWith(
		tracer("gofusa"),
		&capFake{tool: "cfusa", lang: fusaops.LangC, detect: true, avail: false},     // skipped: not installed
		&fakeAdapter{tool: "nope", lang: fusaops.LangCpp, detect: true, avail: true}, // not a McdcRunner
	)
	agg, err := New(reg).RunMCDC(context.Background(), t.TempDir(), Options{Project: "p"})
	if err != nil {
		t.Fatal(err)
	}
	if len(agg.Components) != 3 {
		t.Fatalf("want 3 components, got %d", len(agg.Components))
	}
	byTool := make(map[string]mcdc.MCDCComponent, 3)
	for _, c := range agg.Components {
		byTool[c.Tool] = c
	}
	if byTool["gofusa"].Report == nil {
		t.Error("gofusa should have an MCDC report")
	}
	if byTool["cfusa"].Skipped == "" {
		t.Error("cfusa should be skipped (not installed)")
	}
	if byTool["nope"].Skipped == "" {
		t.Error("nope should be skipped (not a McdcRunner)")
	}
}

//fusa:test REQ-FO-MCDC002
func TestRunMCDCErrorRecordedAsSkipped(t *testing.T) {
	bad := &capFake{tool: "gofusa", lang: fusaops.LangGo, detect: true, avail: true, mcdcErr: errors.New("mcdc boom")}
	agg, err := New(regWith(bad)).RunMCDC(context.Background(), t.TempDir(), Options{})
	if err != nil {
		t.Fatal(err)
	}
	if agg.Components[0].Skipped == "" {
		t.Error("expected mcdc error to be recorded as skipped")
	}
}

//fusa:test REQ-FO-MCDC002
func TestRunMCDCNoAdapters(t *testing.T) {
	none := regWith(&capFake{tool: "gofusa", detect: false, avail: true})
	if _, err := New(none).RunMCDC(context.Background(), t.TempDir(), Options{}); !errors.Is(err, fusaops.ErrNoAdapters) {
		t.Errorf("expected ErrNoAdapters, got %v", err)
	}
}

// detectErrAdapter wraps capFake and injects an error from Detect so that
// selectAdapters propagates the error upward without any adapter being selected.
type detectErrAdapter struct{ *capFake }

func (d *detectErrAdapter) Detect(string) (bool, error) {
	return false, errors.New("scan: detect error injected by test")
}

//fusa:test REQ-FO-ORC004
func TestRunTraceSelectError(t *testing.T) {
	reg := regWith(&detectErrAdapter{&capFake{tool: "gofusa", lang: fusaops.LangGo}})
	if _, err := New(reg).RunTrace(context.Background(), t.TempDir(), Options{}); err == nil {
		t.Error("RunTrace: expected error from selectAdapters when Detect fails")
	}
}

//fusa:test REQ-FO-ORC005
func TestRunSBOMSelectError(t *testing.T) {
	reg := regWith(&detectErrAdapter{&capFake{tool: "gofusa", lang: fusaops.LangGo}})
	if _, err := New(reg).RunSBOM(context.Background(), t.TempDir(), Options{}); err == nil {
		t.Error("RunSBOM: expected error from selectAdapters when Detect fails")
	}
}

//fusa:test REQ-FO-ORC007
func TestRunStandardsSelectError(t *testing.T) {
	reg := regWith(&detectErrAdapter{&capFake{tool: "gofusa", lang: fusaops.LangGo}})
	if _, err := New(reg).RunStandards(context.Background(), t.TempDir(), "iso26262", Options{}); err == nil {
		t.Error("RunStandards: expected error from selectAdapters when Detect fails")
	}
}

//fusa:test REQ-FO-ORC013
func TestRunCompSelectError(t *testing.T) {
	reg := regWith(&detectErrAdapter{&capFake{tool: "gofusa", lang: fusaops.LangGo}})
	if _, err := New(reg).RunComp(context.Background(), t.TempDir(), Options{}, 0, ""); err == nil {
		t.Error("RunComp: expected error from selectAdapters when Detect fails")
	}
}

//fusa:test REQ-FO-MCDC002
func TestRunMCDCSelectError(t *testing.T) {
	reg := regWith(&detectErrAdapter{&capFake{tool: "gofusa", lang: fusaops.LangGo}})
	if _, err := New(reg).RunMCDC(context.Background(), t.TempDir(), Options{}); err == nil {
		t.Error("RunMCDC: expected error from selectAdapters when Detect fails")
	}
}

//fusa:test REQ-FO-ORC006
func TestRunAuditPackSelectError(t *testing.T) {
	reg := regWith(&detectErrAdapter{&capFake{tool: "gofusa", lang: fusaops.LangGo}})
	dest := filepath.Join(t.TempDir(), "out.zip")
	if _, err := New(reg).RunAuditPack(context.Background(), t.TempDir(), dest, Options{}); err == nil {
		t.Error("RunAuditPack: expected error from selectAdapters when Detect fails")
	}
}
