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
