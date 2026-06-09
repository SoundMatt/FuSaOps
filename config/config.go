// Package config manages FuSaOps project configuration.
//
// A multi-language project is configured via a .fusaops.json file at the repo
// root. Use Load to read an existing file, Default to build a starter config,
// and Save to write it to disk. The configuration is intentionally small: it
// records project identity, which adapters to enable, and report preferences.
// All language-specific configuration lives in each component's own x-FuSa
// config file (e.g. a Go module's .fusa.json), which FuSaOps does not manage.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	fusaops "github.com/SoundMatt/FuSaOps"
)

// ConfigFile is the conventional name for the FuSaOps configuration file.
const ConfigFile = ".fusaops.json"

// Config is the top-level FuSaOps project configuration.
//
//fusa:req REQ-FO-CFG001
type Config struct {
	Version string        `json:"version"`
	Project ProjectConfig `json:"project"`
	Scan    ScanConfig    `json:"scan"`
	Report  ReportConfig  `json:"report"`
}

// ProjectConfig holds project identity and safety context.
type ProjectConfig struct {
	Name     string `json:"name"`
	Standard string `json:"standard,omitempty"` // ISO26262, IEC61508, DO178C, ...
}

// ScanConfig controls discovery and adapter selection.
type ScanConfig struct {
	// Adapters optionally restricts which adapters run. Empty means "all
	// adapters whose language is detected in the project".
	//
	//fusa:req REQ-FO-CFG002
	Adapters []string `json:"adapters,omitempty"`
	// Exclude lists directory names skipped during language detection
	// (in addition to the always-skipped VCS and build directories).
	Exclude []string `json:"exclude,omitempty"`
	// Components optionally pins explicit sub-directories to scan, each rooted
	// at a path relative to the repo root. Empty means "scan the repo root".
	Components []ComponentConfig `json:"components,omitempty"`
}

// ComponentConfig pins one sub-directory to a specific language/adapter.
type ComponentConfig struct {
	Path     string `json:"path"`
	Language string `json:"language,omitempty"`
}

// ReportConfig controls aggregate report output.
type ReportConfig struct {
	Format string `json:"format"`           // text | json | html | sarif
	Output string `json:"output,omitempty"` // file path; stdout if empty
}

// Default returns a starter Config for the given project name.
//
//fusa:req REQ-FO-CFG003
func Default(name string) *Config {
	return &Config{
		Version: "1",
		Project: ProjectConfig{Name: name, Standard: "generic"},
		Scan:    ScanConfig{},
		Report:  ReportConfig{Format: "text"},
	}
}

// Load reads and validates a Config from the JSON file at path.
//
//fusa:req REQ-FO-CFG004
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("%w: %s", fusaops.ErrNoConfig, path)
		}
		return nil, fmt.Errorf("config: read %s: %w", path, err)
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("%w: parse %s: %v", fusaops.ErrInvalidConfig, path, err)
	}
	if err := Validate(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// Save marshals cfg to indented JSON and writes it to path.
//
//fusa:req REQ-FO-CFG005
func Save(path string, cfg *Config) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("config: marshal: %w", err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0o640); err != nil {
		return fmt.Errorf("config: write %s: %w", path, err)
	}
	return nil
}

// Validate returns an error if cfg contains inconsistencies.
//
//fusa:req REQ-FO-CFG006
func Validate(cfg *Config) error {
	if cfg == nil {
		return fmt.Errorf("%w: nil config", fusaops.ErrInvalidConfig)
	}
	if cfg.Version == "" {
		return fmt.Errorf("%w: missing version", fusaops.ErrInvalidConfig)
	}
	if cfg.Project.Name == "" {
		return fmt.Errorf("%w: missing project.name", fusaops.ErrInvalidConfig)
	}
	switch cfg.Report.Format {
	case "", "text", "json", "html", "sarif":
	default:
		return fmt.Errorf("%w: unsupported report.format %q", fusaops.ErrInvalidConfig, cfg.Report.Format)
	}
	return nil
}
