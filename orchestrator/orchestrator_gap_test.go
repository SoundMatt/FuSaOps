package orchestrator

// Gap tests for orchestrator: covers timeout branches in Run/RunTrace/RunSBOM/
// RunStandards/RunComp/RunMCDC/RunAuditPack, the ErrNoAdapters paths in RunSBOM
// and RunAuditPack, and the Detect-error and component-pin branches in Run.

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	fusaops "github.com/SoundMatt/FuSaOps"
)

// detectErrFake is a fake adapter whose Detect returns a configurable error.
type detectErrFake struct {
	fakeAdapter
	detectErr error
}

func (f *detectErrFake) Detect(string) (bool, error) { return false, f.detectErr }

// ── orchestrator.go timeout + error branches ─────────────────────────────────

// TestRunWithTimeout verifies that Run enters the per-job timeout branch when
// Options.Timeout > 0, covering orchestrator.go:187–190.
//
//fusa:test REQ-FO-ORC008
func TestRunWithTimeout(t *testing.T) {
	a := &fakeAdapter{tool: "gofusa", lang: fusaops.LangGo, detect: true, avail: true}
	_, err := New(regWith(a)).Run(context.Background(), t.TempDir(), Options{Timeout: time.Millisecond})
	if err != nil {
		t.Fatalf("Run with timeout: %v", err)
	}
}

// TestRunDetectError verifies Run returns an error when Registry.Applicable
// fails due to an adapter Detect error, covering orchestrator.go:117–119.
//
//fusa:test REQ-FO-ORC003
func TestRunDetectError(t *testing.T) {
	a := &detectErrFake{
		fakeAdapter: fakeAdapter{tool: "gofusa", lang: fusaops.LangGo},
		detectErr:   errors.New("detect failed"),
	}
	_, err := New(regWith(a)).Run(context.Background(), t.TempDir(), Options{})
	if err == nil || !errors.Is(err, a.detectErr) {
		t.Errorf("Run detect error: want error wrapping %q, got %v", a.detectErr, err)
	}
}

// TestRunComponentDetectError verifies Run returns an error when Registry.Applicable
// fails while resolving component pins, covering orchestrator.go:132–134.
//
//fusa:test REQ-FO-ORC010
func TestRunComponentDetectError(t *testing.T) {
	a := &detectErrFake{
		fakeAdapter: fakeAdapter{tool: "gofusa", lang: fusaops.LangGo},
		detectErr:   errors.New("detect component failed"),
	}
	opts := Options{
		Components: []ComponentPin{{Path: "."}},
	}
	_, err := New(regWith(a)).Run(context.Background(), t.TempDir(), opts)
	if err == nil || !errors.Is(err, a.detectErr) {
		t.Errorf("Run component detect error: want error wrapping %q, got %v", a.detectErr, err)
	}
}

// TestRunComponentOnlyFilter verifies that when both Components and Only are
// set, adapters not in Only are skipped from the component pin scan, covering
// orchestrator.go:139–141.
//
//fusa:test REQ-FO-ORC010
func TestRunComponentOnlyFilter(t *testing.T) {
	a := &fakeAdapter{tool: "gofusa", lang: fusaops.LangGo, detect: true, avail: true}
	opts := Options{
		Components: []ComponentPin{{Path: "."}},
		Only:       []string{"cfusa"}, // gofusa not in Only → filtered out
	}
	_, err := New(regWith(a)).Run(context.Background(), t.TempDir(), opts)
	if !errors.Is(err, fusaops.ErrNoAdapters) {
		t.Errorf("Run component only-filter: want ErrNoAdapters, got %v", err)
	}
}

// TestRunComponentPinTimeout verifies that a ComponentPin with Timeout > 0
// overrides the global timeout and enters the per-job timeout branch, covering
// orchestrator.go:145–147 and 187–190.
//
//fusa:test REQ-FO-ORC008
//fusa:test REQ-FO-ORC010
func TestRunComponentPinTimeout(t *testing.T) {
	a := &fakeAdapter{tool: "gofusa", lang: fusaops.LangGo, detect: true, avail: true}
	opts := Options{
		Components: []ComponentPin{{Path: ".", Timeout: time.Millisecond}},
	}
	_, err := New(regWith(a)).Run(context.Background(), t.TempDir(), opts)
	if err != nil {
		t.Fatalf("Run component pin timeout: %v", err)
	}
}

// ── rollup.go timeout + ErrNoAdapters branches ───────────────────────────────

// TestRunTraceWithTimeout verifies RunTrace enters the per-adapter timeout
// branch when Options.Timeout > 0, covering rollup.go:86–89.
//
//fusa:test REQ-FO-ORC004
//fusa:test REQ-FO-ORC008
func TestRunTraceWithTimeout(t *testing.T) {
	agg, err := New(regWith(tracer("gofusa"))).RunTrace(
		context.Background(), t.TempDir(), Options{Timeout: time.Millisecond})
	if err != nil {
		t.Fatalf("RunTrace with timeout: %v", err)
	}
	if len(agg.Components) == 0 {
		t.Error("expected at least one trace component")
	}
}

// TestRunSBOMNoAdapters verifies RunSBOM returns ErrNoAdapters when no adapter
// detects the project root, covering rollup.go:133–135.
//
//fusa:test REQ-FO-ORC005
func TestRunSBOMNoAdapters(t *testing.T) {
	none := regWith(&capFake{tool: "gofusa", detect: false, avail: true})
	_, err := New(none).RunSBOM(context.Background(), t.TempDir(), Options{})
	if !errors.Is(err, fusaops.ErrNoAdapters) {
		t.Errorf("RunSBOM no adapters: want ErrNoAdapters, got %v", err)
	}
}

// TestRunSBOMWithTimeout verifies RunSBOM enters the per-adapter timeout branch
// when Options.Timeout > 0, covering rollup.go:154–157.
//
//fusa:test REQ-FO-ORC005
//fusa:test REQ-FO-ORC008
func TestRunSBOMWithTimeout(t *testing.T) {
	agg, err := New(regWith(tracer("gofusa"))).RunSBOM(
		context.Background(), t.TempDir(), Options{Timeout: time.Millisecond})
	if err != nil {
		t.Fatalf("RunSBOM with timeout: %v", err)
	}
	if len(agg.Components) == 0 {
		t.Error("expected at least one sbom component")
	}
}

// TestRunStandardsWithWorkersAndTimeout verifies RunStandards enters both the
// semaphore block (Workers > 0) and the per-adapter timeout branch (Timeout >
// 0), covering rollup.go:208–220.
//
//fusa:test REQ-FO-ORC006
//fusa:test REQ-FO-ORC008
func TestRunStandardsWithWorkersAndTimeout(t *testing.T) {
	a := tracer("gofusa")
	b := tracer("cfusa")
	agg, err := New(regWith(a, b)).RunStandards(
		context.Background(), t.TempDir(), "iso26262",
		Options{Workers: 2, Timeout: time.Millisecond})
	if err != nil {
		t.Fatalf("RunStandards workers+timeout: %v", err)
	}
	if len(agg.Components) < 2 {
		t.Errorf("expected 2 standards components, got %d", len(agg.Components))
	}
}

// TestRunCompWithWorkersAndTimeout verifies RunComp enters both the semaphore
// block (Workers > 0) and the per-adapter timeout branch (Timeout > 0),
// covering rollup.go:281–293.
//
//fusa:test REQ-FO-ORC013
//fusa:test REQ-FO-ORC008
func TestRunCompWithWorkersAndTimeout(t *testing.T) {
	a := tracer("gofusa")
	b := tracer("cfusa")
	agg, err := New(regWith(a, b)).RunComp(
		context.Background(), t.TempDir(),
		Options{Workers: 2, Timeout: time.Millisecond}, 0, "")
	if err != nil {
		t.Fatalf("RunComp workers+timeout: %v", err)
	}
	if len(agg.Components) < 2 {
		t.Errorf("expected 2 comp components, got %d", len(agg.Components))
	}
}

// TestRunMCDCWithWorkersAndTimeout verifies RunMCDC enters both the semaphore
// block (Workers > 0) and the per-adapter timeout branch (Timeout > 0),
// covering rollup.go:343–355.
//
//fusa:test REQ-FO-ORC014
//fusa:test REQ-FO-ORC008
func TestRunMCDCWithWorkersAndTimeout(t *testing.T) {
	a := tracer("gofusa")
	b := tracer("cfusa")
	agg, err := New(regWith(a, b)).RunMCDC(
		context.Background(), t.TempDir(),
		Options{Workers: 2, Timeout: time.Millisecond})
	if err != nil {
		t.Fatalf("RunMCDC workers+timeout: %v", err)
	}
	if len(agg.Components) < 2 {
		t.Errorf("expected 2 mcdc components, got %d", len(agg.Components))
	}
}

// TestRunAuditPackUnavailableAdapter verifies RunAuditPack records the adapter
// as skipped when its binary is not installed, covering rollup.go:425–429.
//
//fusa:test REQ-FO-ORC009
func TestRunAuditPackUnavailableAdapter(t *testing.T) {
	unavail := &capFake{tool: "gofusa", lang: fusaops.LangGo, detect: true, avail: false}
	dest := filepath.Join(t.TempDir(), "out.zip")
	res, err := New(regWith(unavail)).RunAuditPack(context.Background(), t.TempDir(), dest, Options{})
	if err != nil {
		t.Fatalf("RunAuditPack unavailable: %v", err)
	}
	if len(res.Skipped) != 1 {
		t.Errorf("expected 1 skipped, got %d: %+v", len(res.Skipped), res)
	}
}

// TestRunAuditPackWithWorkersAndTimeout verifies RunAuditPack enters both the
// semaphore block (Workers > 0) and the per-adapter timeout branch (Timeout >
// 0), covering rollup.go:418–442.
//
//fusa:test REQ-FO-ORC009
//fusa:test REQ-FO-ORC008
func TestRunAuditPackWithWorkersAndTimeout(t *testing.T) {
	a := tracer("gofusa")
	b := tracer("cfusa")
	dest := filepath.Join(t.TempDir(), "out.zip")
	_, err := New(regWith(a, b)).RunAuditPack(
		context.Background(), t.TempDir(), dest,
		Options{Workers: 2, Timeout: time.Millisecond})
	if err != nil {
		t.Fatalf("RunAuditPack workers+timeout: %v", err)
	}
	if _, serr := os.Stat(dest); serr != nil {
		t.Errorf("expected output zip at %s: %v", dest, serr)
	}
}
