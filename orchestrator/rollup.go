package orchestrator

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

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

// RunTrace rolls every applicable tool's requirement traceability matrix and
// qualification summary up into one cross-language Aggregate. A component whose
// binary is missing, whose tool cannot trace, or whose trace fails is recorded
// as skipped so coverage gaps stay visible rather than inflating the totals.
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
	var components []trace.ComponentTrace
	for _, a := range adapters {
		ct := trace.ComponentTrace{Language: a.Language().String(), Tool: a.Tool(), Available: a.Available()}
		switch {
		case !ct.Available:
			ct.Skipped = fmt.Sprintf("%s binary not found on PATH", a.Tool())
		default:
			tr, ok := a.(adapter.Tracer)
			if !ok {
				ct.Skipped = fmt.Sprintf("%s does not support trace", a.Tool())
				break
			}
			m, err := tr.Trace(ctx, root)
			if err != nil {
				ct.Skipped = fmt.Sprintf("trace failed: %v", err)
				break
			}
			ct.Coverage = m.Coverage
			ct.Requirements = m.Requirements
			// Qualification is best-effort: its absence must not drop the row.
			if q, ok := a.(adapter.Qualifier); ok {
				if qr, qerr := q.Qualify(ctx, root); qerr == nil {
					ct.Qualification = qr
				}
			}
		}
		components = append(components, ct)
	}
	return trace.New(root, opts.Project, components), nil
}

// RunSBOM rolls every applicable tool's SBOM up into one merged, de-duplicated
// cross-language Aggregate, recording skipped components as for RunTrace.
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
	var components []sbom.ComponentSBOM
	for _, a := range adapters {
		cs := sbom.ComponentSBOM{Language: a.Language().String(), Tool: a.Tool(), Available: a.Available()}
		switch {
		case !cs.Available:
			cs.Skipped = fmt.Sprintf("%s binary not found on PATH", a.Tool())
		default:
			s, ok := a.(adapter.SBOMer)
			if !ok {
				cs.Skipped = fmt.Sprintf("%s does not support SBOM", a.Tool())
				break
			}
			doc, err := s.SBOM(ctx, root)
			if err != nil {
				cs.Skipped = fmt.Sprintf("sbom failed: %v", err)
				break
			}
			cs.Module = doc.Module
			cs.Packages = doc.Components
		}
		components = append(components, cs)
	}
	return sbom.New(root, opts.Project, components), nil
}

// RunStandards rolls every applicable tool's §9.3 gap report for standard up
// into one cross-language Aggregate.  A component whose binary is missing,
// whose tool cannot produce a gap report, or whose command fails is recorded as
// skipped so coverage gaps remain visible.
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
	var components []standards.ComponentGap
	for _, a := range adapters {
		cg := standards.ComponentGap{Language: a.Language().String(), Tool: a.Tool()}
		switch {
		case !a.Available():
			cg.Skipped = fmt.Sprintf("%s binary not found on PATH", a.Tool())
		default:
			sp, ok := a.(adapter.StandardsProvider)
			if !ok {
				cg.Skipped = fmt.Sprintf("%s does not support standards", a.Tool())
				break
			}
			r, serr := sp.Standards(ctx, root, standard)
			if serr != nil {
				cg.Skipped = fmt.Sprintf("standards failed: %v", serr)
				break
			}
			cg.Report = r
		}
		components = append(components, cg)
	}
	return standards.New(opts.Project, standard, components), nil
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

	res := &AuditPackResult{}
	var sources []auditpack.Source

	for _, a := range adapters {
		if !a.Available() {
			res.Skipped = append(res.Skipped, fmt.Sprintf("%s: binary not found on PATH", a.Tool()))
			continue
		}
		p, ok := a.(adapter.Packer)
		if !ok {
			res.Skipped = append(res.Skipped, fmt.Sprintf("%s: does not support audit-pack", a.Tool()))
			continue
		}
		zipPath := filepath.Join(tmp, a.Tool()+"-audit-pack.zip")
		if perr := p.AuditPack(ctx, root, zipPath); perr != nil {
			res.Skipped = append(res.Skipped, fmt.Sprintf("%s: %v", a.Tool(), perr))
			continue
		}
		sources = append(sources, auditpack.Source{
			ArchivePath: "components/" + a.Tool() + "/audit-pack.zip",
			FilePath:    zipPath,
		})
		res.Packed = append(res.Packed, a.Tool())
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
