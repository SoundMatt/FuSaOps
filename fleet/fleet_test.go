package fleet

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	fusaops "github.com/SoundMatt/FuSaOps"
	"github.com/SoundMatt/FuSaOps/adapter"
	"github.com/SoundMatt/FuSaOps/orchestrator"
)

type fakeAdapter struct {
	lang     fusaops.Language
	tool     string
	findings []fusaops.Finding
}

func (f *fakeAdapter) Name() string                { return f.tool }
func (f *fakeAdapter) Language() fusaops.Language  { return f.lang }
func (f *fakeAdapter) Tool() string                { return f.tool }
func (f *fakeAdapter) Detect(string) (bool, error) { return true, nil }
func (f *fakeAdapter) Available() bool             { return true }
func (f *fakeAdapter) Check(_ context.Context, _ string) ([]fusaops.Finding, error) {
	return f.findings, nil
}

func newRunner(findings ...fusaops.Finding) *orchestrator.Runner {
	reg := adapter.NewRegistry()
	reg.MustRegister(&fakeAdapter{
		lang:     fusaops.LangGo,
		tool:     "gofusa",
		findings: findings,
	})
	return orchestrator.New(reg)
}

// TestLoadConfig verifies JSON config parsing.
//
//fusa:test REQ-FO-FLT001
func TestLoadConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "fleet.json")
	content := `{"project":"myfleet","repos":[{"name":"svc","dir":"/tmp/svc"}]}`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.Project != "myfleet" || len(cfg.Repos) != 1 || cfg.Repos[0].Name != "svc" {
		t.Errorf("unexpected config: %+v", cfg)
	}
}

// TestLoadConfigMissing verifies a missing config file returns an error.
//
//fusa:test REQ-FO-FLT001
func TestLoadConfigMissing(t *testing.T) {
	_, err := LoadConfig("/nonexistent/fleet.json")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

// TestLoadConfigInvalidJSON verifies malformed config returns an error.
//
//fusa:test REQ-FO-FLT001
func TestLoadConfigInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "fleet.json")
	_ = os.WriteFile(path, []byte("{bad json}"), 0o644)
	_, err := LoadConfig(path)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

// TestRunFleet verifies Run produces correct per-repo results.
//
//fusa:test REQ-FO-FLT003
func TestRunFleet(t *testing.T) {
	dir := t.TempDir()
	cfg := Config{
		Project: "testfleet",
		Repos: []Repo{
			{Name: "clean", Dir: dir},
			{Name: "warn", Dir: dir},
		},
	}
	runner := newRunner(fusaops.Finding{
		Language: fusaops.LangGo, Tool: "gofusa",
		Severity: fusaops.SeverityWarning, RuleID: "W001", Message: "warn",
	})
	fr := Run(context.Background(), cfg, runner)
	if len(fr.Repos) != 2 {
		t.Fatalf("want 2 repo results, got %d", len(fr.Repos))
	}
	if fr.Repos[0].Name != "clean" {
		t.Errorf("repo[0].Name: got %q", fr.Repos[0].Name)
	}
	if fr.Status() != "WARN" {
		t.Errorf("fleet status: got %q, want WARN", fr.Status())
	}
	if fr.Warnings != 2 {
		t.Errorf("fleet warnings: got %d, want 2", fr.Warnings)
	}
}

// TestFleetStatusFail verifies FAIL propagates to fleet status.
//
//fusa:test REQ-FO-FLT002
func TestFleetStatusFail(t *testing.T) {
	dir := t.TempDir()
	cfg := Config{
		Project: "fail-fleet",
		Repos:   []Repo{{Name: "r", Dir: dir}},
	}
	runner := newRunner(fusaops.Finding{
		Language: fusaops.LangGo, Tool: "gofusa",
		Severity: fusaops.SeverityError, RuleID: "E001", Message: "err",
	})
	fr := Run(context.Background(), cfg, runner)
	if fr.Status() != "FAIL" {
		t.Errorf("want FAIL, got %s", fr.Status())
	}
	if !fr.HasFailures() {
		t.Error("HasFailures should be true")
	}
}

// TestFleetStatusPass verifies a clean fleet is PASS.
//
//fusa:test REQ-FO-FLT002
func TestFleetStatusPass(t *testing.T) {
	dir := t.TempDir()
	cfg := Config{Project: "p", Repos: []Repo{{Name: "r", Dir: dir}}}
	fr := Run(context.Background(), cfg, newRunner())
	if fr.Status() != "PASS" {
		t.Errorf("want PASS, got %s", fr.Status())
	}
	if fr.HasFailures() || fr.HasWarnings() {
		t.Error("clean fleet should not have failures or warnings")
	}
}

// TestRenderText verifies text rendering produces expected fields.
//
//fusa:test REQ-FO-FLT004
func TestRenderText(t *testing.T) {
	dir := t.TempDir()
	cfg := Config{Project: "p", Repos: []Repo{{Name: "svc", Dir: dir}}}
	fr := Run(context.Background(), cfg, newRunner())
	var sb strings.Builder
	if err := Render(&sb, fr, "text"); err != nil {
		t.Fatalf("Render text: %v", err)
	}
	out := sb.String()
	if !strings.Contains(out, "Fleet:") {
		t.Error("missing Fleet: header")
	}
	if !strings.Contains(out, "svc") {
		t.Error("missing repo name in output")
	}
	if !strings.Contains(out, "PASS") {
		t.Error("missing PASS status")
	}
}

// TestRenderJSON verifies JSON rendering is parseable.
//
//fusa:test REQ-FO-FLT004
func TestRenderJSON(t *testing.T) {
	dir := t.TempDir()
	cfg := Config{Project: "p", Repos: []Repo{{Name: "svc", Dir: dir}}}
	fr := Run(context.Background(), cfg, newRunner())
	var sb strings.Builder
	if err := Render(&sb, fr, "json"); err != nil {
		t.Fatalf("Render json: %v", err)
	}
	var out FleetReport
	if err := json.Unmarshal([]byte(sb.String()), &out); err != nil {
		t.Fatalf("not valid JSON: %v", err)
	}
	if out.Project != "p" {
		t.Errorf("project: got %q", out.Project)
	}
}

// TestRenderUnsupportedFormat verifies an error is returned for unknown formats.
//
//fusa:test REQ-FO-FLT004
func TestRenderUnsupportedFormat(t *testing.T) {
	fr := &FleetReport{}
	if err := Render(nil, fr, "bogus"); err == nil {
		t.Error("expected error for unsupported format")
	}
}

// TestRenderHTMLBasic verifies HTML output contains key fleet information.
//
//fusa:test REQ-FO-FLT005
func TestRenderHTMLBasic(t *testing.T) {
	fr := &FleetReport{
		Project: "test-fleet",
		Repos: []RepoResult{
			{Name: "repo-a", Status: "PASS", Total: 0, Errors: 0, Warnings: 0},
			{Name: "repo-b", Status: "FAIL", Total: 3, Errors: 2, Warnings: 1},
		},
		Total:    3,
		Errors:   2,
		Warnings: 1,
	}
	var buf strings.Builder
	if err := Render(&buf, fr, "html"); err != nil {
		t.Fatalf("Render html: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "test-fleet") {
		t.Errorf("expected project name in HTML: %q", out[:200])
	}
	if !strings.Contains(out, "repo-a") || !strings.Contains(out, "repo-b") {
		t.Errorf("expected repo names in HTML")
	}
	if !strings.Contains(out, "PASS") || !strings.Contains(out, "FAIL") {
		t.Errorf("expected status badges in HTML")
	}
	if !strings.Contains(out, "badge-fail") {
		t.Errorf("expected badge-fail CSS class in HTML")
	}
}

//fusa:test REQ-FO-FLT005
func TestRenderHTMLScanError(t *testing.T) {
	fr := &FleetReport{
		Project: "err-fleet",
		Repos: []RepoResult{
			{Name: "broken", Status: "ERROR", ScanErr: "tool not found"},
		},
	}
	var buf strings.Builder
	if err := Render(&buf, fr, "html"); err != nil {
		t.Fatalf("Render html with scan error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "tool not found") {
		t.Errorf("expected scan error in HTML: %q", out[:200])
	}
}

//fusa:test REQ-FO-FLT005
func TestRenderHTMLPass(t *testing.T) {
	fr := &FleetReport{
		Project: "clean-fleet",
		Repos: []RepoResult{
			{Name: "repo-ok", Status: "PASS"},
		},
	}
	var buf strings.Builder
	if err := Render(&buf, fr, "html"); err != nil {
		t.Fatalf("Render html pass: %v", err)
	}
	if !strings.Contains(buf.String(), "badge-pass") {
		t.Errorf("expected badge-pass in clean fleet HTML")
	}
}

// TestRenderMarkdown verifies markdown rendering produces a GFM table with fleet data.
//
//fusa:test REQ-FO-FLT007
func TestRenderMarkdown(t *testing.T) {
	fr := &FleetReport{
		Project: "my-fleet",
		Repos: []RepoResult{
			{Name: "repo-a", Status: "PASS", Total: 0, Errors: 0, Warnings: 0, Infos: 0},
			{Name: "repo-b", Status: "FAIL", Total: 5, Errors: 3, Warnings: 2, Infos: 0},
		},
		Total:    5,
		Errors:   3,
		Warnings: 2,
	}
	var buf strings.Builder
	if err := Render(&buf, fr, "markdown"); err != nil {
		t.Fatalf("Render markdown: %v", err)
	}
	out := buf.String()
	for _, want := range []string{"# FuSaOps Fleet", "my-fleet", "**FAIL**", "| Repository |", "repo-a", "repo-b", "TOTAL"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q in markdown output:\n%s", want, out)
		}
	}
}

// TestRenderMarkdownAlias verifies "md" is an alias for "markdown".
//
//fusa:test REQ-FO-FLT007
func TestRenderMarkdownAlias(t *testing.T) {
	fr := &FleetReport{Project: "p", Repos: []RepoResult{{Name: "r", Status: "PASS"}}}
	var buf strings.Builder
	if err := Render(&buf, fr, "md"); err != nil {
		t.Fatalf("Render md alias: %v", err)
	}
	if !strings.Contains(buf.String(), "# FuSaOps Fleet") {
		t.Error("expected markdown header from md alias")
	}
}

// TestRenderMarkdownScanError verifies scan errors appear in the markdown table.
//
//fusa:test REQ-FO-FLT007
func TestRenderMarkdownScanError(t *testing.T) {
	fr := &FleetReport{
		Project: "broken-fleet",
		Repos:   []RepoResult{{Name: "broken", Status: "ERROR", ScanErr: "adapter crashed"}},
	}
	var buf strings.Builder
	if err := Render(&buf, fr, "markdown"); err != nil {
		t.Fatalf("Render markdown scan error: %v", err)
	}
	if !strings.Contains(buf.String(), "adapter crashed") {
		t.Errorf("expected scan error in markdown output:\n%s", buf.String())
	}
}

// TestRenderMarkdownPass verifies green badge for all-PASS fleet.
//
//fusa:test REQ-FO-FLT007
func TestRenderMarkdownPass(t *testing.T) {
	fr := &FleetReport{
		Project: "clean",
		Repos:   []RepoResult{{Name: "ok", Status: "PASS"}},
	}
	var buf strings.Builder
	if err := Render(&buf, fr, "markdown"); err != nil {
		t.Fatalf("Render markdown pass: %v", err)
	}
	if !strings.Contains(buf.String(), "🟢") {
		t.Error("expected green badge for PASS fleet")
	}
}

// TestRenderToFile verifies RenderToFile writes to a file.
//
//fusa:test REQ-FO-FLT004
func TestRenderToFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "fleet.json")
	fr := &FleetReport{Project: "p"}
	if err := RenderToFile(nil, fr, "json", path); err != nil {
		t.Fatalf("RenderToFile: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	if !strings.Contains(string(data), `"project"`) {
		t.Error("output missing project field")
	}
}
