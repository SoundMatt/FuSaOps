package config

import (
	"errors"
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
