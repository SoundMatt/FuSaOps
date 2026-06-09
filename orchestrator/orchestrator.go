// Package orchestrator runs the applicable x-FuSa adapters against a repository
// and assembles their output into a single AggregateReport.
//
// It is the engine of FuSaOps: it decides which adapters apply, runs the ones
// whose tool binaries are installed, records why any were skipped, and merges
// the per-language findings into the aggregate view consumed by the CLI and
// web UI.
package orchestrator

import (
	"context"
	"fmt"

	fusaops "github.com/SoundMatt/FuSaOps"
	"github.com/SoundMatt/FuSaOps/adapter"
	"github.com/SoundMatt/FuSaOps/report"
)

// Options configures a Run.
//
//fusa:req REQ-FO-ORC001
type Options struct {
	// Project is the human-readable project name recorded in the report.
	Project string
	// Only restricts execution to adapters whose tool name appears here.
	// Empty means "every applicable adapter".
	Only []string
	// RequireAvailable, when true, makes Run return ErrNoAdapters if no
	// applicable adapter's binary is installed.
	RequireAvailable bool
}

// Runner executes adapters against a project root. It wraps a registry so the
// set of adapters can be swapped in tests.
//
//fusa:req REQ-FO-ORC002
type Runner struct {
	Registry *adapter.Registry
}

// New returns a Runner backed by the given registry, defaulting to the
// package-level adapter.Default when reg is nil.
func New(reg *adapter.Registry) *Runner {
	if reg == nil {
		reg = adapter.Default
	}
	return &Runner{Registry: reg}
}

// Run detects applicable adapters under root, executes the installed ones, and
// returns the aggregated report. Adapters that apply but whose binary is not
// installed are recorded as skipped components rather than silently dropped, so
// the report makes coverage gaps explicit.
//
//fusa:req REQ-FO-ORC003
func (rn *Runner) Run(ctx context.Context, root string, opts Options) (*report.AggregateReport, error) {
	applicable, err := rn.Registry.Applicable(root)
	if err != nil {
		return nil, fmt.Errorf("orchestrator: detect adapters: %w", err)
	}

	only := make(map[string]struct{}, len(opts.Only))
	for _, t := range opts.Only {
		only[t] = struct{}{}
	}

	var components []report.Component
	ran := 0
	for _, a := range applicable {
		if len(only) > 0 {
			if _, ok := only[a.Tool()]; !ok {
				continue
			}
		}
		comp := report.Component{
			Language:  a.Language(),
			Tool:      a.Tool(),
			Available: a.Available(),
		}
		if !comp.Available {
			comp.Skipped = fmt.Sprintf("%s binary not found on PATH", a.Tool())
			components = append(components, comp)
			continue
		}
		findings, err := a.Check(ctx, root)
		if err != nil {
			comp.Skipped = fmt.Sprintf("check failed: %v", err)
			components = append(components, comp)
			continue
		}
		comp.Findings = findings
		components = append(components, comp)
		ran++
	}

	if len(components) == 0 {
		return nil, fusaops.ErrNoAdapters
	}
	if opts.RequireAvailable && ran == 0 {
		return nil, fmt.Errorf("%w: applicable tools detected but none installed", fusaops.ErrNoAdapters)
	}

	return report.New(root, opts.Project, components), nil
}
