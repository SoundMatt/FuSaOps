package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	fusaops "github.com/SoundMatt/FuSaOps"
)

//fusa:test REQ-FO-CFG003
func TestDefaultIsValid(t *testing.T) {
	cfg := Default("demo")
	if err := Validate(cfg); err != nil {
		t.Fatalf("default config invalid: %v", err)
	}
	if cfg.Project.Name != "demo" {
		t.Errorf("project name: got %q", cfg.Project.Name)
	}
}

//fusa:test REQ-FO-CFG001
//fusa:test REQ-FO-CFG002
//fusa:test REQ-FO-CFG004
//fusa:test REQ-FO-CFG005
func TestSaveLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ConfigFile)
	cfg := Default("round")
	cfg.Scan.Adapters = []string{"gofusa"}
	if err := Save(path, cfg); err != nil {
		t.Fatalf("save: %v", err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if got.Project.Name != "round" || len(got.Scan.Adapters) != 1 {
		t.Errorf("round trip mismatch: %+v", got)
	}
}

func TestLoadMissingReturnsErrNoConfig(t *testing.T) {
	_, err := Load(filepath.Join(t.TempDir(), "nope.json"))
	if !errors.Is(err, fusaops.ErrNoConfig) {
		t.Errorf("got %v, want ErrNoConfig", err)
	}
}

// TestLoadDirectoryNotExist verifies Load returns a non-ErrNoConfig error when
// the path is a directory (os.ReadFile returns EISDIR, not ErrNotExist).
//
//fusa:test REQ-FO-CFG004
func TestLoadDirectoryNotExist(t *testing.T) {
	_, err := Load(t.TempDir())
	if err == nil {
		t.Error("Load: expected error when path is a directory")
	}
	if errors.Is(err, fusaops.ErrNoConfig) {
		t.Errorf("Load: got ErrNoConfig but expected read-error for directory path")
	}
}

//fusa:test REQ-FO-CFG006
func TestValidateRejectsBadFormat(t *testing.T) {
	cfg := Default("x")
	cfg.Report.Format = "xml"
	if err := Validate(cfg); !errors.Is(err, fusaops.ErrInvalidConfig) {
		t.Errorf("got %v, want ErrInvalidConfig", err)
	}
}

func TestValidateRejectsMissingName(t *testing.T) {
	cfg := Default("x")
	cfg.Project.Name = ""
	if err := Validate(cfg); !errors.Is(err, fusaops.ErrInvalidConfig) {
		t.Errorf("got %v, want ErrInvalidConfig", err)
	}
}

func TestValidateNilConfig(t *testing.T) {
	if err := Validate(nil); !errors.Is(err, fusaops.ErrInvalidConfig) {
		t.Errorf("got %v, want ErrInvalidConfig for nil config", err)
	}
}

//fusa:test REQ-FO-CFG007
//fusa:test REQ-FO-CFG008
//fusa:test REQ-FO-CFG009
func TestRunConfigAndComponentsSaveLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ConfigFile)
	cfg := Default("comp-test")
	cfg.Run = RunConfig{Timeout: "30s", Workers: 4}
	cfg.Scan.Components = []ComponentConfig{
		{Path: "services/auth", Adapter: "gofusa", Timeout: "10s"},
		{Path: "lib/safety", Adapter: "cfusa"},
	}
	if err := Save(path, cfg); err != nil {
		t.Fatalf("save: %v", err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if got.Run.Timeout != "30s" {
		t.Errorf("run.timeout: got %q, want 30s", got.Run.Timeout)
	}
	if got.Run.Workers != 4 {
		t.Errorf("run.workers: got %d, want 4", got.Run.Workers)
	}
	if len(got.Scan.Components) != 2 {
		t.Fatalf("want 2 components, got %d", len(got.Scan.Components))
	}
	if got.Scan.Components[0].Adapter != "gofusa" {
		t.Errorf("component adapter: got %q, want gofusa", got.Scan.Components[0].Adapter)
	}
	if got.Scan.Components[0].Timeout != "10s" {
		t.Errorf("component timeout: got %q, want 10s", got.Scan.Components[0].Timeout)
	}
}

//fusa:test REQ-FO-CFG010
func TestVandVConfigRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ConfigFile)
	cfg := Default("vv-test")
	cfg.VandV = VandVConfig{
		ImplementationAuthor:    "Alice",
		IndependentReviewer:     "Bob",
		IndependentTestExecutor: "Carol",
	}
	if err := Save(path, cfg); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.VandV.ImplementationAuthor != "Alice" {
		t.Errorf("got author %q, want Alice", got.VandV.ImplementationAuthor)
	}
	if got.VandV.IndependentReviewer != "Bob" {
		t.Errorf("got reviewer %q, want Bob", got.VandV.IndependentReviewer)
	}
	if got.VandV.IndependentTestExecutor != "Carol" {
		t.Errorf("got executor %q, want Carol", got.VandV.IndependentTestExecutor)
	}
}

func TestLoadInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ConfigFile)
	if err := os.WriteFile(path, []byte("{invalid"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Error("expected error for invalid JSON")
	}
}

// TestTraceConfigRoundTrip verifies that TraceConfig.ReqDecomposition.Enforce
// survives a save/load round-trip.
//
//fusa:test REQ-FO-CFG011
func TestTraceConfigRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ConfigFile)
	cfg := Default("trace-rt")
	cfg.Trace = TraceConfig{
		ReqDecomposition: ReqDecompositionConfig{Enforce: "warn", MinLevel: "LLR"},
	}
	if err := Save(path, cfg); err != nil {
		t.Fatalf("save: %v", err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if got.Trace.ReqDecomposition.Enforce != "warn" {
		t.Errorf("enforce: got %q, want %q", got.Trace.ReqDecomposition.Enforce, "warn")
	}
	if got.Trace.ReqDecomposition.MinLevel != "LLR" {
		t.Errorf("minLevel: got %q, want %q", got.Trace.ReqDecomposition.MinLevel, "LLR")
	}
}

// TestValidateRejectsUnknownDecompEnforce verifies that Validate rejects an
// unknown enforce value.
//
//fusa:test REQ-FO-CFG011
func TestValidateRejectsUnknownDecompEnforce(t *testing.T) {
	cfg := Default("x")
	cfg.Trace.ReqDecomposition.Enforce = "strict"
	if err := Validate(cfg); !errors.Is(err, fusaops.ErrInvalidConfig) {
		t.Errorf("got %v, want ErrInvalidConfig for unknown enforce value", err)
	}
}

//fusa:test REQ-FO-CFG012
func TestQualifyConfigRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ConfigFile)
	cfg := Default("qual-test")
	cfg.Qualify = QualifyConfig{
		Type:      "independent",
		RecordUri: "https://cert",
	}
	if err := Save(path, cfg); err != nil {
		t.Fatalf("save: %v", err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if got.Qualify.Type != "independent" {
		t.Errorf("Qualify.Type = %q, want \"independent\"", got.Qualify.Type)
	}
	if got.Qualify.RecordUri != "https://cert" {
		t.Errorf("Qualify.RecordUri = %q, want \"https://cert\"", got.Qualify.RecordUri)
	}

	// Validate accepts "self" and "independent".
	for _, typ := range []string{"", "self", "independent"} {
		c := Default("x")
		c.Qualify.Type = typ
		if err := Validate(c); err != nil {
			t.Errorf("Validate with type %q: unexpected error: %v", typ, err)
		}
	}

	// Validate rejects unknown types.
	bad := Default("x")
	bad.Qualify.Type = "tql5"
	if err := Validate(bad); !errors.Is(err, fusaops.ErrInvalidConfig) {
		t.Errorf("Validate with type \"tql5\": got %v, want ErrInvalidConfig", err)
	}
}

//fusa:test REQ-FO-CFG013
func TestCoverageConfigSaveLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ConfigFile)
	cfg := Default("mcdc-test")
	cfg.Coverage = CoverageConfig{
		DAL: "DAL-A",
		Mcdc: McdcConfig{
			Enabled:   true,
			Threshold: 95.0,
		},
	}
	if err := Save(path, cfg); err != nil {
		t.Fatalf("save: %v", err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if !got.Coverage.Mcdc.Enabled {
		t.Error("coverage.mcdc.enabled should round-trip to true")
	}
	if got.Coverage.Mcdc.Threshold != 95.0 {
		t.Errorf("coverage.mcdc.threshold: got %.1f, want 95.0", got.Coverage.Mcdc.Threshold)
	}
	if got.Coverage.DAL != "DAL-A" {
		t.Errorf("coverage.dal: got %q, want DAL-A", got.Coverage.DAL)
	}
}

//fusa:test REQ-FO-CFG013
func TestCoverageConfigInvalidThreshold(t *testing.T) {
	cfg := Default("mcdc-bad")
	cfg.Coverage.Mcdc.Threshold = 150.0
	if err := Validate(cfg); !errors.Is(err, fusaops.ErrInvalidConfig) {
		t.Errorf("got %v, want ErrInvalidConfig for threshold=150", err)
	}
}

//fusa:test REQ-FO-CFG014
func TestCompConfigSaveLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ConfigFile)
	cfg := Default("comp-test")
	cfg.Comp = CompConfig{Threshold: 15, DAL: "DAL-C"}
	if err := Save(path, cfg); err != nil {
		t.Fatalf("save: %v", err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if got.Comp.Threshold != 15 {
		t.Errorf("comp.threshold: got %d, want 15", got.Comp.Threshold)
	}
	if got.Comp.DAL != "DAL-C" {
		t.Errorf("comp.dal: got %q, want DAL-C", got.Comp.DAL)
	}
}

// TestSaveWriteError verifies Save returns an error when the parent directory
// does not exist.
//
//fusa:test REQ-FO-CFG005
func TestSaveWriteError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing", "out.json")
	if err := Save(path, Default("testproject")); err == nil {
		t.Error("Save: expected error for non-existent parent directory")
	}
}
