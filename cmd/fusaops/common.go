package main

import (
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"time"

	fusaops "github.com/SoundMatt/FuSaOps"
	"github.com/SoundMatt/FuSaOps/config"
	"github.com/SoundMatt/FuSaOps/orchestrator"
	"github.com/SoundMatt/FuSaOps/report"
)

// loadOptions resolves the absolute root and builds orchestrator options,
// merging any .fusaops.json found at the root. A missing config is not an
// error: FuSaOps works zero-config by detecting languages directly.
//
//fusa:req REQ-FO-CLI007
func loadOptions(dir, only string, stderr io.Writer) (string, orchestrator.Options, *config.Config, error) {
	root, err := filepath.Abs(dir)
	if err != nil {
		return "", orchestrator.Options{}, nil, err
	}
	opts := orchestrator.Options{Project: filepath.Base(root)}

	cfg, err := config.Load(filepath.Join(root, config.ConfigFile))
	if err != nil && !errors.Is(err, fusaops.ErrNoConfig) {
		return "", orchestrator.Options{}, nil, err
	}
	if cfg != nil {
		if cfg.Project.Name != "" {
			opts.Project = cfg.Project.Name
		}
		opts.Only = cfg.Scan.Adapters
		if cfg.Run.Timeout != "" {
			d, perr := time.ParseDuration(cfg.Run.Timeout)
			if perr != nil {
				fmt.Fprintf(stderr, "fusaops: invalid run.timeout %q: %v (ignored)\n", cfg.Run.Timeout, perr)
			} else {
				opts.Timeout = d
			}
		}
		opts.Workers = cfg.Run.Workers
		for _, c := range cfg.Scan.Components {
			pin := orchestrator.ComponentPin{Path: c.Path, Adapter: c.Adapter}
			if c.Timeout != "" {
				d, perr := time.ParseDuration(c.Timeout)
				if perr != nil {
					fmt.Fprintf(stderr, "fusaops: invalid component timeout %q for %s: %v (ignored)\n", c.Timeout, c.Path, perr)
				} else {
					pin.Timeout = d
				}
			}
			opts.Components = append(opts.Components, pin)
		}
	}
	if only != "" {
		opts.Only = splitCSV(only)
	}
	return root, opts, cfg, nil
}

// splitCSV splits a comma-separated flag value, trimming empties.
func splitCSV(s string) []string {
	var out []string
	start := 0
	for i := 0; i <= len(s); i++ {
		if i == len(s) || s[i] == ',' {
			if i > start {
				out = append(out, s[start:i])
			}
			start = i + 1
		}
	}
	return out
}

// applyIntegrityLevel populates Standard/ASIL/SIL/DAL on rep from cfg.
// When cfg is nil (zero-config mode) the report fields remain empty.
//
//fusa:req REQ-FO-RPT020
func applyIntegrityLevel(rep *report.AggregateReport, cfg *config.Config) {
	if cfg == nil {
		return
	}
	rep.Standard = cfg.Project.Standard
	switch cfg.Project.Standard {
	case "IEC61508":
		rep.SIL = cfg.Project.SIL
	case "DO178C":
		rep.DAL = cfg.Project.DAL
	default:
		rep.ASIL = cfg.Project.ASIL
	}
}
