package orchestrator

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	fusaops "github.com/SoundMatt/FuSaOps"
	"github.com/SoundMatt/FuSaOps/adapter"
	"github.com/SoundMatt/FuSaOps/auditpack"
	"github.com/SoundMatt/FuSaOps/sbom"
	"github.com/SoundMatt/FuSaOps/standards"
	"github.com/SoundMatt/FuSaOps/trace"
)

// selectAdapters returns the applicable adapters under root, narrowed by
// opts.Only. It is the shared front half of every roll-up: each command runs a
// different capability against the same selected set.
func (rn *Runner) selectAdapters(root string, opts Options) ([]adapter.Adapter, error) {
	applicable, err := rn.Registry.Applicable(root)
	if err != nil {
		return nil, fmt.Errorf("orchestrator: detect adapters: %w", err)
	}
	if len(opts.Only) == 0 {
		return applicable, nil
	}
	only := make(map[string]struct{}, len(opts.Only))
	for _, t := range opts.Only {
		only[t] = struct{}{}
	}
	var out []adapter.Adapter
	for _, a := range applicable {
		if _, ok := only[a.Tool()]; ok {
			out = append(out, a)
		}
	}
	return out, nil
}

// semaphore returns a buffered channel for limiting concurrency, or nil when
// workers is zero (unlimited).
func newSem(workers int) chan struct{} {
	if workers > 0 {
		return make(chan struct{}, workers)
	}
	return nil
}

// RunTrace rolls every applicable tool's requirement traceability matrix and
// qualification summary up into one cross-language Aggregate. A component whose
// binary is missing, whose tool cannot trace, or whose trace fails is recorded
// as skipped so coverage gaps stay visible rather than inflating the totals.
// Adapters run in parallel, governed by Options.Workers and Options.Timeout.
//
//fusa:req REQ-FO-ORC004
func (rn *Runner) RunTrace(ctx context.Context, root string, opts Options) (*trace.Aggregate, error) {
	adapters, err := rn.selectAdapters(root, opts)
	if err != nil {
		return nil, err
	}
	if len(adapters) == 0 {
		return nil, fusaops.ErrNoAdapters
	}

	results := make([]trace.ComponentTrace, len(adapters))
	var wg sync.WaitGroup
	sem := newSem(opts.Workers)

	for i, a := range adapters {
		wg.Add(1)
		go func(i int, a adapter.Adapter) {
			defer wg.Done()
			if sem != nil {
				sem <- struct{}{}
				defer func() { <-sem }()
			}

			ct := trace.ComponentTrace{Language: a.Language().String(), Tool: a.Tool(), Available: a.Available()}

			tctx := ctx
			var cancel context.CancelFunc
			if opts.Timeout > 0 {
				tctx, cancel = context.WithTimeout(ctx, opts.Timeout)
				defer cancel()
			}

			switch {
			case !ct.Available:
				ct.Skipped = fmt.Sprintf("%s binary not found on PATH", a.Tool())
			default:
				tr, ok := a.(adapter.Tracer)
				if !ok {
					ct.Skipped = fmt.Sprintf("%s does not support trace", a.Tool())
					break
				}
				m, err := tr.Trace(tctx, root)
				if err != nil {
					ct.Skipped = fmt.Sprintf("trace failed: %v", err)
					break
				}
				ct.Coverage = m.Coverage
				ct.Requirements = m.Requirements
				ct.Tags = m.Tags
				// Qualification is best-effort: its absence must not drop the row.
				if q, ok := a.(adapter.Qualifier); ok {
					if qr, qerr := q.Qualify(tctx, root); qerr == nil {
						ct.Qualification = qr
					}
				}
			}
			results[i] = ct
		}(i, a)
	}
	wg.Wait()

	return trace.New(root, opts.Project, results), nil
}

// RunSBOM rolls every applicable tool's SBOM up into one merged, de-duplicated
// cross-language Aggregate, recording skipped components as for RunTrace.
// Adapters run in parallel, governed by Options.Workers and Options.Timeout.
//
//fusa:req REQ-FO-ORC005
func (rn *Runner) RunSBOM(ctx context.Context, root string, opts Options) (*sbom.Aggregate, error) {
	adapters, err := rn.selectAdapters(root, opts)
	if err != nil {
		return nil, err
	}
	if len(adapters) == 0 {
		return nil, fusaops.ErrNoAdapters
	}

	results := make([]sbom.ComponentSBOM, len(adapters))
	var wg sync.WaitGroup
	sem := newSem(opts.Workers)

	for i, a := range adapters {
		wg.Add(1)
		go func(i int, a adapter.Adapter) {
			defer wg.Done()
			if sem != nil {
				sem <- struct{}{}
				defer func() { <-sem }()
			}

			cs := sbom.ComponentSBOM{Language: a.Language().String(), Tool: a.Tool(), Available: a.Available()}

			tctx := ctx
			var cancel context.CancelFunc
			if opts.Timeout > 0 {
				tctx, cancel = context.WithTimeout(ctx, opts.Timeout)
				defer cancel()
			}

			switch {
			case !cs.Available:
				cs.Skipped = fmt.Sprintf("%s binary not found on PATH", a.Tool())
			default:
				s, ok := a.(adapter.SBOMer)
				if !ok {
					cs.Skipped = fmt.Sprintf("%s does not support SBOM", a.Tool())
					break
				}
				doc, err := s.SBOM(tctx, root)
				if err != nil {
					cs.Skipped = fmt.Sprintf("sbom failed: %v", err)
					break
				}
				cs.Module = doc.Module
				cs.Packages = doc.Components
			}
			results[i] = cs
		}(i, a)
	}
	wg.Wait()

	return sbom.New(root, opts.Project, results), nil
}

// RunStandards rolls every applicable tool's §9.3 gap report for standard up
// into one cross-language Aggregate. A component whose binary is missing,
// whose tool cannot produce a gap report, or whose command fails is recorded as
// skipped so coverage gaps remain visible.
// Adapters run in parallel, governed by Options.Workers and Options.Timeout.
//
//fusa:req REQ-FO-ORC007
func (rn *Runner) RunStandards(ctx context.Context, root, standard string, opts Options) (*standards.Aggregate, error) {
	adapters, err := rn.selectAdapters(root, opts)
	if err != nil {
		return nil, err
	}
	if len(adapters) == 0 {
		return nil, fusaops.ErrNoAdapters
	}

	results := make([]standards.ComponentGap, len(adapters))
	var wg sync.WaitGroup
	sem := newSem(opts.Workers)

	for i, a := range adapters {
		wg.Add(1)
		go func(i int, a adapter.Adapter) {
			defer wg.Done()
			if sem != nil {
				sem <- struct{}{}
				defer func() { <-sem }()
			}

			cg := standards.ComponentGap{Language: a.Language().String(), Tool: a.Tool()}

			tctx := ctx
			var cancel context.CancelFunc
			if opts.Timeout > 0 {
				tctx, cancel = context.WithTimeout(ctx, opts.Timeout)
				defer cancel()
			}

			switch {
			case !a.Available():
				cg.Skipped = fmt.Sprintf("%s binary not found on PATH", a.Tool())
			default:
				sp, ok := a.(adapter.StandardsProvider)
				if !ok {
					cg.Skipped = fmt.Sprintf("%s does not support standards", a.Tool())
					break
				}
				r, serr := sp.Standards(tctx, root, standard)
				if serr != nil {
					cg.Skipped = fmt.Sprintf("standards failed: %v", serr)
					break
				}
				cg.Report = r
			}
			results[i] = cg
		}(i, a)
	}
	wg.Wait()

	return standards.New(opts.Project, standard, results), nil
}

// AuditPackResult reports what a unified audit-pack run produced.
//
//fusa:req REQ-FO-ORC006
type AuditPackResult struct {
	Manifest *auditpack.Manifest
	Packed   []string // tools whose per-tool pack was bundled
	Skipped  []string // "tool: reason" for components that contributed no pack
}

// RunAuditPack bundles every applicable tool's audit-pack ZIP together with the
// FuSaOps cross-language artefacts (aggregate report, trace matrix, SBOM) into a
// single audit-pack.zip at dest. Per-tool packs land under components/<tool>/.
// A tool that cannot produce a pack is recorded in Skipped; the FuSaOps-level
// evidence is always included so the bundle is never empty.
// Per-tool packing runs in parallel, governed by Options.Workers and Options.Timeout.
//
//fusa:req REQ-FO-ORC006
func (rn *Runner) RunAuditPack(ctx context.Context, root, dest string, opts Options) (*AuditPackResult, error) {
	adapters, err := rn.selectAdapters(root, opts)
	if err != nil {
		return nil, err
	}
	if len(adapters) == 0 {
		return nil, fusaops.ErrNoAdapters
	}

	tmp, err := os.MkdirTemp("", "fusaops-auditpack-*")
	if err != nil {
		return nil, fmt.Errorf("orchestrator: temp dir: %w", err)
	}
	defer func() { _ = os.RemoveAll(tmp) }()

	type packResult struct {
		tool    string
		zipPath string
		skipped string
	}

	packResults := make([]packResult, len(adapters))
	var wg sync.WaitGroup
	sem := newSem(opts.Workers)

	for i, a := range adapters {
		wg.Add(1)
		go func(i int, a adapter.Adapter) {
			defer wg.Done()
			if sem != nil {
				sem <- struct{}{}
				defer func() { <-sem }()
			}

			pr := packResult{tool: a.Tool()}

			if !a.Available() {
				pr.skipped = fmt.Sprintf("%s: binary not found on PATH", a.Tool())
				packResults[i] = pr
				return
			}
			p, ok := a.(adapter.Packer)
			if !ok {
				pr.skipped = fmt.Sprintf("%s: does not support audit-pack", a.Tool())
				packResults[i] = pr
				return
			}

			tctx := ctx
			var cancel context.CancelFunc
			if opts.Timeout > 0 {
				tctx, cancel = context.WithTimeout(ctx, opts.Timeout)
				defer cancel()
			}

			zipPath := filepath.Join(tmp, a.Tool()+"-audit-pack.zip")
			if perr := p.AuditPack(tctx, root, zipPath); perr != nil {
				pr.skipped = fmt.Sprintf("%s: %v", a.Tool(), perr)
				packResults[i] = pr
				return
			}
			pr.zipPath = zipPath
			packResults[i] = pr
		}(i, a)
	}
	wg.Wait()

	res := &AuditPackResult{}
	var sources []auditpack.Source
	for _, pr := range packResults {
		if pr.skipped != "" {
			res.Skipped = append(res.Skipped, pr.skipped)
			continue
		}
		sources = append(sources, auditpack.Source{
			ArchivePath: "components/" + pr.tool + "/audit-pack.zip",
			FilePath:    pr.zipPath,
		})
		res.Packed = append(res.Packed, pr.tool)
	}

	// Always include the FuSaOps cross-language evidence.
	addJSON := func(name string, v any) {
		if s, jerr := auditpack.JSONSource(name, v); jerr == nil {
			sources = append(sources, s)
		}
	}
	if rep, rerr := rn.Run(ctx, root, opts); rerr == nil {
		addJSON("report.json", rep)
	}
	if tr, rerr := rn.RunTrace(ctx, root, opts); rerr == nil {
		addJSON("trace.json", tr)
	}
	if sb, rerr := rn.RunSBOM(ctx, root, opts); rerr == nil {
		addJSON("sbom.json", sb)
	}

	manifest, err := auditpack.Pack(dest, opts.Project, sources)
	if err != nil {
		return nil, err
	}
	res.Manifest = manifest
	return res, nil
}
