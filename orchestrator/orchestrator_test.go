package orchestrator

import (
	"context"
	"errors"
	"os"
	"testing"

	fusaops "github.com/SoundMatt/FuSaOps"
	"github.com/SoundMatt/FuSaOps/adapter"
)

// fakeAdapter implements adapter.Adapter for tests without real binaries.
type fakeAdapter struct {
	name, tool string
	lang       fusaops.Language
	detect     bool
	avail      bool
	findings   []fusaops.Finding
	checkErr   error
}

func (f *fakeAdapter) Name() string                { return f.name }
func (f *fakeAdapter) Language() fusaops.Language  { return f.lang }
func (f *fakeAdapter) Tool() string                { return f.tool }
func (f *fakeAdapter) Detect(string) (bool, error) { return f.detect, nil }
func (f *fakeAdapter) Available() bool             { return f.avail }
func (f *fakeAdapter) Check(context.Context, string) ([]fusaops.Finding, error) {
	return f.findings, f.checkErr
}

func regWith(as ...adapter.Adapter) *adapter.Registry {
	r := adapter.NewRegistry()
	for _, a := range as {
		r.MustRegister(a)
	}
	return r
}

//fusa:test REQ-FO-ORC003
func TestRunAggregatesInstalledAdapters(t *testing.T) {
	reg := regWith(
		&fakeAdapter{name: "go-FuSa", tool: "gofusa", lang: fusaops.LangGo, detect: true, avail: true,
			findings: []fusaops.Finding{{RuleID: "FUSA001", Severity: fusaops.SeverityError, Message: "x"}}},
		&fakeAdapter{name: "c-FuSa", tool: "cfusa", lang: fusaops.LangC, detect: false, avail: true},
	)
	rep, err := New(reg).Run(context.Background(), t.TempDir(), Options{Project: "p"})
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Components) != 1 {
		t.Fatalf("got %d components, want 1 (only detected)", len(rep.Components))
	}
	if !rep.HasErrors() {
		t.Error("expected aggregate errors")
	}
}

func TestRunRecordsSkippedWhenNotInstalled(t *testing.T) {
	reg := regWith(&fakeAdapter{name: "c-FuSa", tool: "cfusa", lang: fusaops.LangC, detect: true, avail: false})
	rep, err := New(reg).Run(context.Background(), t.TempDir(), Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Components) != 1 || rep.Components[0].Skipped == "" {
		t.Errorf("expected one skipped component, got %+v", rep.Components)
	}
}

func TestRunCheckErrorRecordedAsSkipped(t *testing.T) {
	reg := regWith(&fakeAdapter{name: "go-FuSa", tool: "gofusa", lang: fusaops.LangGo, detect: true, avail: true,
		checkErr: errors.New("boom")})
	rep, err := New(reg).Run(context.Background(), t.TempDir(), Options{})
	if err != nil {
		t.Fatal(err)
	}
	if rep.Components[0].Skipped == "" {
		t.Error("check error should be recorded as skipped reason")
	}
}

func TestRunNoApplicableReturnsErr(t *testing.T) {
	reg := regWith(&fakeAdapter{tool: "gofusa", lang: fusaops.LangGo, detect: false, avail: true})
	_, err := New(reg).Run(context.Background(), t.TempDir(), Options{})
	if !errors.Is(err, fusaops.ErrNoAdapters) {
		t.Errorf("got %v, want ErrNoAdapters", err)
	}
}

//fusa:test REQ-FO-ORC001
func TestRunOnlyFilter(t *testing.T) {
	reg := regWith(
		&fakeAdapter{tool: "gofusa", lang: fusaops.LangGo, detect: true, avail: true},
		&fakeAdapter{tool: "cfusa", lang: fusaops.LangC, detect: true, avail: true},
	)
	rep, err := New(reg).Run(context.Background(), t.TempDir(), Options{Only: []string{"cfusa"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Components) != 1 || rep.Components[0].Tool != "cfusa" {
		t.Errorf("only filter failed: %+v", rep.Components)
	}
}

func TestRunRequireAvailable(t *testing.T) {
	reg := regWith(&fakeAdapter{tool: "gofusa", lang: fusaops.LangGo, detect: true, avail: false})
	_, err := New(reg).Run(context.Background(), t.TempDir(), Options{RequireAvailable: true})
	if !errors.Is(err, fusaops.ErrNoAdapters) {
		t.Errorf("got %v, want ErrNoAdapters", err)
	}
}

//fusa:test REQ-FO-ORC002
func TestNewNilRegistryUsesDefault(t *testing.T) {
	if New(nil).Registry != adapter.Default {
		t.Error("nil registry should default to adapter.Default")
	}
}

//fusa:test REQ-FO-ORC010
func TestRunComponentPins(t *testing.T) {
	root := t.TempDir()
	reg := regWith(
		&fakeAdapter{tool: "gofusa", lang: fusaops.LangGo, detect: true, avail: true,
			findings: []fusaops.Finding{{RuleID: "G1", Severity: fusaops.SeverityWarning}}},
		&fakeAdapter{tool: "cfusa", lang: fusaops.LangC, detect: true, avail: true},
	)
	opts := Options{
		Components: []ComponentPin{
			{Path: ".", Adapter: "gofusa"},
		},
	}
	rep, err := New(reg).Run(context.Background(), root, opts)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Components) != 1 || rep.Components[0].Tool != "gofusa" {
		t.Errorf("component pin should restrict to gofusa: %+v", rep.Components)
	}
}

//fusa:test REQ-FO-ORC008
//fusa:test REQ-FO-ORC009
func TestRunWithWorkers(t *testing.T) {
	reg := regWith(
		&fakeAdapter{tool: "gofusa", lang: fusaops.LangGo, detect: true, avail: true,
			findings: []fusaops.Finding{{RuleID: "G1", Severity: fusaops.SeverityInfo}}},
		&fakeAdapter{tool: "cfusa", lang: fusaops.LangC, detect: true, avail: true},
	)
	rep, err := New(reg).Run(context.Background(), t.TempDir(), Options{Workers: 2})
	if err != nil {
		t.Fatal(err)
	}
	// Both adapters detected — both should appear regardless of concurrency.
	if len(rep.Components) != 2 {
		t.Errorf("expected 2 components with workers=2, got %d", len(rep.Components))
	}
}

// TestRunSuppressFile verifies suppressed findings are excluded from the summary.
//
//fusa:test REQ-FO-SUP004
func TestRunSuppressFile(t *testing.T) {
	const fp = "deadbeef"
	reg := regWith(
		&fakeAdapter{tool: "gofusa", lang: fusaops.LangGo, detect: true, avail: true,
			findings: []fusaops.Finding{
				{RuleID: "G1", Severity: fusaops.SeverityError, Fingerprint: fp},
				{RuleID: "G2", Severity: fusaops.SeverityWarning, Fingerprint: "other"},
			}},
	)
	// Write a suppression config.
	dir := t.TempDir()
	supPath := dir + "/.fusaops-suppress.json"
	content := `{"suppressions":[{"fingerprint":"` + fp + `","reason":"accepted risk"}]}`
	if err := os.WriteFile(supPath, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	opts := Options{SuppressFile: supPath}
	rep, err := New(reg).Run(context.Background(), dir, opts)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Suppressed != 1 {
		t.Errorf("expected 1 suppressed finding, got %d", rep.Suppressed)
	}
	if rep.Summary.Errors != 0 {
		t.Errorf("suppressed error should not appear in summary, got %d", rep.Summary.Errors)
	}
	if rep.Summary.Warnings != 1 {
		t.Errorf("non-suppressed warning should remain, got %d", rep.Summary.Warnings)
	}
}

// TestRunSuppressFileMissing returns an error when the suppress file is invalid.
//
//fusa:test REQ-FO-SUP004
func TestRunSuppressFileMissing(t *testing.T) {
	reg := regWith(
		&fakeAdapter{tool: "gofusa", lang: fusaops.LangGo, detect: true, avail: true},
	)
	opts := Options{SuppressFile: "/nonexistent/suppress.json"}
	_, err := New(reg).Run(context.Background(), t.TempDir(), opts)
	if err == nil {
		t.Error("expected error for missing suppress file")
	}
}
