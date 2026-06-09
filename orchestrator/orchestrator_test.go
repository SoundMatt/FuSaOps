package orchestrator

import (
	"context"
	"errors"
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

func TestNewNilRegistryUsesDefault(t *testing.T) {
	if New(nil).Registry != adapter.Default {
		t.Error("nil registry should default to adapter.Default")
	}
}
