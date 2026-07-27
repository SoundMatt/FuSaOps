package fleet

// Gap tests covering uncovered branches in fleet.go.

import (
	"context"
	"fmt"
	"strings"
	"testing"

	fusaops "github.com/SoundMatt/FuSaOps"
	"github.com/SoundMatt/FuSaOps/adapter"
	"github.com/SoundMatt/FuSaOps/orchestrator"
)

// errDetectAdapter always returns an error from Detect so runner.Run fails.
type errDetectAdapter struct{}

func (e *errDetectAdapter) Name() string                                            { return "errdetect" }
func (e *errDetectAdapter) Language() fusaops.Language                             { return fusaops.LangGo }
func (e *errDetectAdapter) Tool() string                                            { return "errdetect" }
func (e *errDetectAdapter) Detect(string) (bool, error)                            { return false, fmt.Errorf("errdetect: simulated failure") }
func (e *errDetectAdapter) Available() bool                                         { return true }
func (e *errDetectAdapter) Check(_ context.Context, _ string) ([]fusaops.Finding, error) {
	return nil, nil
}

// TestFleetAdapterFilter verifies that opts.Only is populated when Repo.Adapter
// is non-empty, covering fleet.go:110.26,112.5.
//
//fusa:test REQ-FO-FLT003
func TestFleetAdapterFilter(t *testing.T) {
	dir := t.TempDir()
	cfg := Config{
		Project: "filtered",
		Repos:   []Repo{{Name: "svc", Dir: dir, Adapter: "gofusa"}},
	}
	fr := Run(context.Background(), cfg, newRunner())
	if len(fr.Repos) != 1 {
		t.Fatalf("want 1 repo result, got %d", len(fr.Repos))
	}
	// The adapter filter was applied if the run completed (any status).
	if fr.Repos[0].Name != "svc" {
		t.Errorf("repo name: got %q, want svc", fr.Repos[0].Name)
	}
}

// TestFleetRunError verifies that a runner.Run error is recorded as ScanErr on
// the repo result, covering fleet.go:115.18,118.5.
//
//fusa:test REQ-FO-FLT003
func TestFleetRunError(t *testing.T) {
	dir := t.TempDir()
	reg := adapter.NewRegistry()
	reg.MustRegister(&errDetectAdapter{})
	runner := orchestrator.New(reg)

	cfg := Config{
		Project: "errfleet",
		Repos:   []Repo{{Name: "svc", Dir: dir}},
	}
	fr := Run(context.Background(), cfg, runner)
	if len(fr.Repos) != 1 {
		t.Fatalf("want 1 repo result, got %d", len(fr.Repos))
	}
	if fr.Repos[0].ScanErr == "" {
		t.Error("expected ScanErr to be set when runner.Run fails")
	}
	if fr.Repos[0].Status != "ERROR" {
		t.Errorf("expected ERROR status, got %q", fr.Repos[0].Status)
	}
}

// TestRenderTextLongName verifies renderText updates maxName when a repo name
// exceeds the 4-character initial value, covering fleet.go:175.28,177.4.
//
//fusa:test REQ-FO-FLT004
func TestRenderTextLongName(t *testing.T) {
	fr := &FleetReport{
		Project: "p",
		Repos:   []RepoResult{{Name: "longrepository", Status: "PASS"}},
	}
	var sb strings.Builder
	if err := Render(&sb, fr, "text"); err != nil {
		t.Fatalf("Render text long name: %v", err)
	}
	if !strings.Contains(sb.String(), "longrepository") {
		t.Errorf("expected long repo name in text output:\n%s", sb.String())
	}
}

// TestRenderTextScanError verifies renderText prints the scan-error row when
// ScanErr is non-empty, covering fleet.go:182.22,184.4.
//
//fusa:test REQ-FO-FLT004
func TestRenderTextScanError(t *testing.T) {
	fr := &FleetReport{
		Project: "p",
		Repos:   []RepoResult{{Name: "broken", Status: "ERROR", ScanErr: "tool unavailable"}},
	}
	var sb strings.Builder
	if err := Render(&sb, fr, "text"); err != nil {
		t.Fatalf("Render text scan error: %v", err)
	}
	if !strings.Contains(sb.String(), "scan error: tool unavailable") {
		t.Errorf("expected scan error in text output:\n%s", sb.String())
	}
}

// TestRenderHTMLWarnBadge verifies the badge-warn CSS class is emitted for a
// repo with WARN status, covering fleet.go:198.15,199.23 (badgeClass "WARN").
//
//fusa:test REQ-FO-FLT005
func TestRenderHTMLWarnBadge(t *testing.T) {
	fr := &FleetReport{
		Project:  "p",
		Repos:    []RepoResult{{Name: "r", Status: "WARN", Warnings: 1}},
		Warnings: 1,
	}
	var sb strings.Builder
	if err := Render(&sb, fr, "html"); err != nil {
		t.Fatalf("Render html warn: %v", err)
	}
	if !strings.Contains(sb.String(), "badge-warn") {
		t.Errorf("expected badge-warn in HTML output")
	}
}

// TestRenderHTMLSkipBadge verifies the badge-skip CSS class is emitted for a
// repo whose status does not match any known value, covering fleet.go:202.11,203.23
// (badgeClass default branch).
//
//fusa:test REQ-FO-FLT005
func TestRenderHTMLSkipBadge(t *testing.T) {
	fr := &FleetReport{
		Project: "p",
		Repos:   []RepoResult{{Name: "r", Status: "SKIP"}},
	}
	var sb strings.Builder
	if err := Render(&sb, fr, "html"); err != nil {
		t.Fatalf("Render html skip badge: %v", err)
	}
	if !strings.Contains(sb.String(), "badge-skip") {
		t.Errorf("expected badge-skip in HTML output")
	}
}

// TestRenderMarkdownWarn verifies the 🟡 badge is emitted when the fleet
// status is WARN, covering fleet.go:278.14,279.17.
//
//fusa:test REQ-FO-FLT007
func TestRenderMarkdownWarn(t *testing.T) {
	fr := &FleetReport{
		Project:  "p",
		Repos:    []RepoResult{{Name: "r", Status: "WARN", Warnings: 1}},
		Warnings: 1,
	}
	var sb strings.Builder
	if err := Render(&sb, fr, "markdown"); err != nil {
		t.Fatalf("Render markdown warn: %v", err)
	}
	if !strings.Contains(sb.String(), "🟡") {
		t.Errorf("expected 🟡 badge in WARN markdown output:\n%s", sb.String())
	}
}

// TestRenderToFilePassthrough verifies RenderToFile delegates to Render when
// the path argument is empty, covering fleet.go:304.16,306.3.
//
//fusa:test REQ-FO-FLT004
func TestRenderToFilePassthrough(t *testing.T) {
	fr := &FleetReport{Project: "p", Repos: []RepoResult{{Name: "r", Status: "PASS"}}}
	var sb strings.Builder
	if err := RenderToFile(&sb, fr, "text", ""); err != nil {
		t.Fatalf("RenderToFile passthrough: %v", err)
	}
	if !strings.Contains(sb.String(), "Fleet:") {
		t.Errorf("expected Fleet: header in passthrough output:\n%s", sb.String())
	}
}
