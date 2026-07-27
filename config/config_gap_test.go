package config

// Gap tests for config.go:
// - Validate returns error when cfg.Version is empty (config.go:219.23,221.3).
// - Load returns error when Validate fails (config.go:191.39,193.3).
// The Save MarshalIndent error (config.go:202.16,204.3) is unreachable for Config.

import (
	"os"
	"path/filepath"
	"testing"
)

// TestValidateEmptyVersion verifies that Validate returns an ErrInvalidConfig
// error when cfg.Version is empty, covering config.go:219.23,221.3.
//
//fusa:test REQ-FO-CFG006
func TestValidateEmptyVersion(t *testing.T) {
	cfg := &Config{
		Project: ProjectConfig{Name: "testproject"},
		// Version is intentionally left empty
	}
	if err := Validate(cfg); err == nil {
		t.Fatal("Validate: expected error for empty Version, got nil")
	}
}

// TestLoadValidationFailure verifies that Load returns a non-nil error when the
// config file parses successfully but fails Validate (empty version), covering
// config.go:191.39,193.3. This also exercises Validate's empty-version branch.
//
//fusa:test REQ-FO-CFG004
//fusa:test REQ-FO-CFG006
func TestLoadValidationFailure(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, ConfigFile)
	// Valid JSON but empty version → Validate fails.
	if err := os.WriteFile(cfgPath, []byte(`{"project":{"name":"testproject"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Load(cfgPath)
	if err == nil {
		t.Fatal("Load: expected error for config with empty version, got nil")
	}
}
