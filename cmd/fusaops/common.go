package main

import (
	"errors"
	"io"
	"path/filepath"

	fusaops "github.com/SoundMatt/FuSaOps"
	"github.com/SoundMatt/FuSaOps/config"
	"github.com/SoundMatt/FuSaOps/orchestrator"
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
