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
	"path/filepath"
	"sync"
	"time"

	fusaops "github.com/SoundMatt/FuSaOps"
	"github.com/SoundMatt/FuSaOps/adapter"
	"github.com/SoundMatt/FuSaOps/report"
)

// ComponentPin targets one sub-directory at a specific adapter for a Run.
// When Options.Components is non-empty, only pinned paths are scanned.
//
//fusa:req REQ-FO-ORC010
type ComponentPin struct {
	Path    string        // relative to repo root
	Adapter string        // tool name; empty means auto-detect all applicable
	Timeout time.Duration // overrides Options.Timeout for this component; zero means use global
}

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
	// Timeout is the per-adapter execution deadline. Zero means no limit.
	//
	//fusa:req REQ-FO-ORC008
	Timeout time.Duration
	// Workers caps the number of adapters running concurrently. Zero means unlimited.
	//
	//fusa:req REQ-FO-ORC008
	Workers int
	// Components pins specific sub-directories to specific adapters. Empty means
	// scan the whole repo root with all applicable adapters.
	//
	//fusa:req REQ-FO-ORC010
	Components []ComponentPin
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

// Run detects applicable adapters under root, executes the installed ones in
// parallel (governed by Options.Workers), and returns the aggregated report.
// Adapters that apply but whose binary is not installed are recorded as skipped
// components rather than silently dropped. When Options.Components is non-empty,
// each pin is scanned independently so the same adapter may run multiple times.
//
//fusa:req REQ-FO-ORC003
//fusa:req REQ-FO-ORC008
//fusa:req REQ-FO-ORC009
//fusa:req REQ-FO-ORC010
func (rn *Runner) Run(ctx context.Context, root string, opts Options) (*report.AggregateReport, error) {
	type job struct {
		root    string
		dir     string // component-relative path (for report.Component.Dir)
		a       adapter.Adapter
		timeout time.Duration
	}

	only := make(map[string]struct{}, len(opts.Only))
	for _, t := range opts.Only {
		only[t] = struct{}{}
	}

	var jobs []job

	if len(opts.Components) == 0 {
		applicable, err := rn.Registry.Applicable(root)
		if err != nil {
			return nil, fmt.Errorf("orchestrator: detect adapters: %w", err)
		}
		for _, a := range applicable {
			if len(only) > 0 {
				if _, ok := only[a.Tool()]; !ok {
					continue
				}
			}
			jobs = append(jobs, job{root: root, dir: "", a: a, timeout: opts.Timeout})
		}
	} else {
		for _, pin := range opts.Components {
			absPath := filepath.Join(root, pin.Path)
			applicable, err := rn.Registry.Applicable(absPath)
			if err != nil {
				return nil, fmt.Errorf("orchestrator: detect adapters for %s: %w", pin.Path, err)
			}
			for _, a := range applicable {
				if pin.Adapter != "" && a.Tool() != pin.Adapter {
					continue
				}
				if len(only) > 0 {
					if _, ok := only[a.Tool()]; !ok {
						continue
					}
				}
				timeout := opts.Timeout
				if pin.Timeout > 0 {
					timeout = pin.Timeout
				}
				jobs = append(jobs, job{root: absPath, dir: pin.Path, a: a, timeout: timeout})
			}
		}
	}

	if len(jobs) == 0 {
		return nil, fusaops.ErrNoAdapters
	}

	results := make([]report.Component, len(jobs))
	var wg sync.WaitGroup
	var sem chan struct{}
	if opts.Workers > 0 {
		sem = make(chan struct{}, opts.Workers)
	}

	for i, j := range jobs {
		wg.Add(1)
		go func(i int, j job) {
			defer wg.Done()
			if sem != nil {
				sem <- struct{}{}
				defer func() { <-sem }()
			}

			comp := report.Component{
				Language:  j.a.Language(),
				Tool:      j.a.Tool(),
				Dir:       j.dir,
				Available: j.a.Available(),
			}
			if !comp.Available {
				comp.Skipped = fmt.Sprintf("%s binary not found on PATH", j.a.Tool())
				results[i] = comp
				return
			}

			tctx := ctx
			var cancel context.CancelFunc
			if j.timeout > 0 {
				tctx, cancel = context.WithTimeout(ctx, j.timeout)
				defer cancel()
			}

			findings, err := j.a.Check(tctx, j.root)
			if err != nil {
				comp.Skipped = fmt.Sprintf("check failed: %v", err)
			} else {
				comp.Findings = findings
			}
			results[i] = comp
		}(i, j)
	}
	wg.Wait()

	ran := 0
	for _, c := range results {
		if c.Skipped == "" && c.Available {
			ran++
		}
	}

	if opts.RequireAvailable && ran == 0 {
		return nil, fmt.Errorf("%w: applicable tools detected but none installed", fusaops.ErrNoAdapters)
	}

	return report.New(root, opts.Project, results), nil
}
