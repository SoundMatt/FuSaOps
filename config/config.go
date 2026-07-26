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

// ReqDecompositionConfig configures the HLR/LLR decomposition gate run by
// "fusaops trace --decomp".
//
//fusa:req REQ-FO-CFG011
type ReqDecompositionConfig struct {
	// Enforce is the gate severity: "off" | "warn" | "error" | "auto".
	// Empty string is treated as "auto": derive severity from the project
	// integrity level (DAL / ASIL).
	Enforce string `json:"enforce,omitempty"`
	// MinLevel is the minimum level a leaf requirement must declare
	// (default "LLR"). Reserved for future use.
	MinLevel string `json:"minLevel,omitempty"`
}

// TraceConfig configures the fusaops trace command.
//
//fusa:req REQ-FO-CFG011
type TraceConfig struct {
	ReqDecomposition ReqDecompositionConfig `json:"reqDecomposition,omitempty"`
}

// McdcConfig controls MC/DC coverage gate behaviour.
//
//fusa:req REQ-FO-CFG013
type McdcConfig struct {
	Enabled   bool    `json:"enabled"`
	Threshold float64 `json:"threshold"` // 0–100; 0 means use default (100.0 for DAL-A)
}

// CoverageConfig controls coverage subcommand defaults from .fusaops.json.
//
//fusa:req REQ-FO-CFG013
type CoverageConfig struct {
	DAL  string     `json:"dal,omitempty"` // default DAL if --dal not given
	Mcdc McdcConfig `json:"mcdc,omitempty"`
}

// CompConfig configures the fusaops comp command and /api/v1/comp endpoint.
//
//fusa:req REQ-FO-CFG014
type CompConfig struct {
	// Threshold is the McCabe cyclomatic complexity ceiling per function.
	// 0 means use the DAL default (DAL-B → 10 when DAL is empty).
	Threshold int `json:"threshold,omitempty"`
	// DAL is the DO-178C design assurance level used to derive the threshold
	// when Threshold is 0 (e.g. "DAL-A" → 4, "DAL-B" → 10).
	DAL string `json:"dal,omitempty"`
}

// Config is the top-level FuSaOps project configuration.
//
//fusa:req REQ-FO-CFG001
type Config struct {
	Version  string         `json:"version"`
	Project  ProjectConfig  `json:"project"`
	Scan     ScanConfig     `json:"scan"`
	Report   ReportConfig   `json:"report"`
	Run      RunConfig      `json:"run,omitempty"`
	VandV    VandVConfig    `json:"vv,omitempty"`
	Trace    TraceConfig    `json:"trace,omitempty"`
	Qualify  QualifyConfig  `json:"qualify,omitempty"`  //fusa:req REQ-FO-CFG012
	Coverage CoverageConfig `json:"coverage,omitempty"` //fusa:req REQ-FO-CFG013
	Comp     CompConfig     `json:"comp,omitempty"`     //fusa:req REQ-FO-CFG014
}

// VandVConfig holds per-repo V&V independence declarations embedded in .fusaops.json.
// Independence determines the achievable ASIL per ISO 26262-2:2018 §6.4.
//
//fusa:req REQ-FO-CFG010
type VandVConfig struct {
	// ImplementationAuthor is the person or team that wrote the implementation.
	ImplementationAuthor string `json:"implementationAuthor,omitempty"`
	// IndependentReviewer is the person (distinct from author) who performed
	// independent design review, satisfying ISO 26262-2:2018 independence for ASIL-C.
	IndependentReviewer string `json:"independentReviewer,omitempty"`
	// IndependentTestExecutor is the person (distinct from author) who executed
	// tests independently, satisfying the additional independence for ASIL-D.
	IndependentTestExecutor string `json:"independentTestExecutor,omitempty"`
}

// ProjectConfig holds project identity and safety context.
type ProjectConfig struct {
	Name     string `json:"name"`
	Standard string `json:"standard,omitempty"` // ISO26262, IEC61508, DO178C, ...
	ASIL     string `json:"asil,omitempty"`     // ISO 26262 integrity level: ASIL-A … ASIL-D
	SIL      string `json:"sil,omitempty"`      // IEC 61508 integrity level: SIL-1 … SIL-4
	DAL      string `json:"dal,omitempty"`      // DO-178C integrity level: DAL-A … DAL-E
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

// ComponentConfig pins one sub-directory to a specific adapter.
//
//fusa:req REQ-FO-CFG009
type ComponentConfig struct {
	Path    string `json:"path"`
	Adapter string `json:"adapter,omitempty"` // tool name, e.g. "gofusa"; empty = auto-detect
	Timeout string `json:"timeout,omitempty"` // e.g. "30s"; overrides run.timeout
}

// RunConfig controls adapter execution concurrency and per-adapter timeouts.
type RunConfig struct {
	// Timeout is the per-adapter execution timeout (e.g. "60s"). Empty means no limit.
	//
	//fusa:req REQ-FO-CFG007
	Timeout string `json:"timeout,omitempty"`
	// Workers caps the number of adapters running concurrently. Zero means unlimited.
	//
	//fusa:req REQ-FO-CFG008
	Workers int `json:"workers,omitempty"`
}

// ReportConfig controls aggregate report output.
type ReportConfig struct {
	Format string `json:"format"`           // text | json | html | sarif
	Output string `json:"output,omitempty"` // file path; stdout if empty
}

// QualifyConfig controls tool-qualification report settings.
//
//fusa:req REQ-FO-CFG012
type QualifyConfig struct {
	// Type is the qualification approach: "self" (default) or "independent".
	Type string `json:"type,omitempty"`
	// RecordUri is the URI of an external TQL-5/DO-330 qualification certificate.
	RecordUri string `json:"recordUri,omitempty"`
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
	switch cfg.Trace.ReqDecomposition.Enforce {
	case "", "off", "warn", "error", "auto":
	default:
		return fmt.Errorf("%w: unsupported trace.reqDecomposition.enforce %q (want: off|warn|error|auto)",
			fusaops.ErrInvalidConfig, cfg.Trace.ReqDecomposition.Enforce)
	}
	switch cfg.Qualify.Type {
	case "", "self", "independent":
	default:
		return fmt.Errorf("%w: unsupported qualify.type %q", fusaops.ErrInvalidConfig, cfg.Qualify.Type)
	}
	if t := cfg.Coverage.Mcdc.Threshold; t != 0 && (t < 0 || t > 100) {
		return fmt.Errorf("%w: coverage.mcdc.threshold must be in range [0, 100]", fusaops.ErrInvalidConfig)
	}
	return nil
}
