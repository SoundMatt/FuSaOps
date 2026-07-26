package adapter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/SoundMatt/FuSaOps/comp"
	"github.com/SoundMatt/FuSaOps/mcdc"
	"github.com/SoundMatt/FuSaOps/sbom"
	"github.com/SoundMatt/FuSaOps/standards"
	"github.com/SoundMatt/FuSaOps/trace"
)

// The capability interfaces below are optional: an Adapter that also implements
// one can contribute the corresponding evidence to a FuSaOps roll-up. The
// orchestrator type-asserts for them, so an adapter that cannot (or whose tool
// does not) produce a given artefact is simply recorded as skipped rather than
// breaking the run. Every cmdAdapter implements all of them by shelling out to
// its tool's matching subcommand.

// Tracer can produce a requirement traceability matrix.
//
//fusa:req REQ-FO-ADP013
type Tracer interface {
	Trace(ctx context.Context, root string) (*trace.Matrix, error)
}

// Qualifier can produce a tool-qualification summary.
//
//fusa:req REQ-FO-ADP013
type Qualifier interface {
	Qualify(ctx context.Context, root string) (*trace.Qualification, error)
}

// SBOMer can produce a Software Bill of Materials.
//
//fusa:req REQ-FO-ADP013
type SBOMer interface {
	SBOM(ctx context.Context, root string) (*sbom.Document, error)
}

// Packer can produce a per-tool audit-pack ZIP at dest.
//
//fusa:req REQ-FO-ADP013
type Packer interface {
	AuditPack(ctx context.Context, root, dest string) error
}

// Compler can produce a cyclomatic complexity (V(G)) comp-report per §9.2.
//
//fusa:req REQ-FO-ADP029
type Compler interface {
	Comp(ctx context.Context, root string, threshold int, dal string) (*comp.Report, error)
}

// StandardsProvider can produce a §9.3 gap report for a given standard id.
//
//fusa:req REQ-FO-ADP018
type StandardsProvider interface {
	Standards(ctx context.Context, root, standard string) (*standards.GapReport, error)
}

// McdcRunner can produce a §9.4 MC/DC coverage report via the --mcdc flag.
//
//fusa:req REQ-FO-MCDC001
type McdcRunner interface {
	MCDC(ctx context.Context, root string) (*mcdc.Report, error)
}

// Trace runs "<tool> trace --format json" and decodes the matrix from stdout.
//
//fusa:req REQ-FO-ADP014
func (a *cmdAdapter) Trace(ctx context.Context, root string) (*trace.Matrix, error) {
	out, err := a.run(ctx, root, a.tool, "trace", "--format", "json")
	if err != nil {
		return nil, fmt.Errorf("adapter %s: trace: %w", a.name, err)
	}
	var m trace.Matrix
	if err := json.Unmarshal(extractJSON(out), &m); err != nil {
		return nil, fmt.Errorf("adapter %s: decode trace matrix: %w", a.name, err)
	}
	return &m, nil
}

// Qualify runs "<tool> qualify --output <tmp>" and decodes the report file. The
// x-FuSa qualify command writes its JSON report to a file rather than stdout.
//
//fusa:req REQ-FO-ADP015
func (a *cmdAdapter) Qualify(ctx context.Context, root string) (*trace.Qualification, error) {
	tmp, err := os.CreateTemp("", "fusaops-"+a.tool+"-qualify-*.json")
	if err != nil {
		return nil, fmt.Errorf("adapter %s: temp file: %w", a.name, err)
	}
	out := tmp.Name()
	_ = tmp.Close()
	defer func() { _ = os.Remove(out) }()

	if _, err = a.run(ctx, root, a.tool, "qualify", "--output", out); err != nil {
		return nil, fmt.Errorf("adapter %s: qualify: %w", a.name, err)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		return nil, fmt.Errorf("adapter %s: read qualify report: %w", a.name, err)
	}
	var q trace.Qualification
	if err := json.Unmarshal(extractJSON(data), &q); err != nil {
		return nil, fmt.Errorf("adapter %s: decode qualify report: %w", a.name, err)
	}
	return &q, nil
}

// SBOM runs "<tool> release --output-dir <tmp>" and decodes the generated
// sbom.json. release is the x-FuSa command that emits an SBOM; the rest of its
// output is written into the throwaway directory and discarded.
//
//fusa:req REQ-FO-ADP016
func (a *cmdAdapter) SBOM(ctx context.Context, root string) (*sbom.Document, error) {
	dir, err := os.MkdirTemp("", "fusaops-"+a.tool+"-sbom-*")
	if err != nil {
		return nil, fmt.Errorf("adapter %s: temp dir: %w", a.name, err)
	}
	defer func() { _ = os.RemoveAll(dir) }()

	if _, err = a.run(ctx, root, a.tool, "release", "--output-dir", dir); err != nil {
		return nil, fmt.Errorf("adapter %s: release: %w", a.name, err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "sbom.json"))
	if err != nil {
		return nil, fmt.Errorf("adapter %s: read sbom.json: %w", a.name, err)
	}
	var doc sbom.Document
	if err := json.Unmarshal(extractJSON(data), &doc); err != nil {
		return nil, fmt.Errorf("adapter %s: decode sbom: %w", a.name, err)
	}
	return &doc, nil
}

// AuditPack runs "<tool> audit-pack --output <dest>" and confirms the bundle was
// written.
//
//fusa:req REQ-FO-ADP017
func (a *cmdAdapter) AuditPack(ctx context.Context, root, dest string) error {
	if _, err := a.run(ctx, root, a.tool, "audit-pack", "--output", dest); err != nil {
		return fmt.Errorf("adapter %s: audit-pack: %w", a.name, err)
	}
	if _, err := os.Stat(dest); err != nil {
		return fmt.Errorf("adapter %s: audit-pack produced no file: %w", a.name, err)
	}
	return nil
}

// Comp runs "<tool> comp --format json" with optional threshold/DAL overrides and
// decodes the resulting comp-report.
//
//fusa:req REQ-FO-ADP029
func (a *cmdAdapter) Comp(ctx context.Context, root string, threshold int, dal string) (*comp.Report, error) {
	args := []string{"comp", "--format", "json"}
	if threshold > 0 {
		args = append(args, "--threshold", fmt.Sprintf("%d", threshold))
	}
	if dal != "" {
		args = append(args, "--dal", dal)
	}
	out, err := a.run(ctx, root, a.tool, args...)
	if err != nil {
		return nil, fmt.Errorf("adapter %s: comp: %w", a.name, err)
	}
	var r comp.Report
	if err := json.Unmarshal(extractJSON(out), &r); err != nil {
		return nil, fmt.Errorf("adapter %s: decode comp report: %w", a.name, err)
	}
	return &r, nil
}

// MCDC runs "<tool> comp --mcdc --format json" and decodes the MC/DC report.
// The --mcdc flag signals the tool to emit a structured MC/DC coverage report
// rather than a standard cyclomatic complexity report.
//
//fusa:req REQ-FO-MCDC001
func (a *cmdAdapter) MCDC(ctx context.Context, root string) (*mcdc.Report, error) {
	out, err := a.run(ctx, root, a.tool, "comp", "--mcdc", "--format", "json")
	if err != nil {
		return nil, fmt.Errorf("adapter %s: mcdc: %w", a.name, err)
	}
	var r mcdc.Report
	if err := json.Unmarshal(extractJSON(out), &r); err != nil {
		return nil, fmt.Errorf("adapter %s: decode mcdc report: %w", a.name, err)
	}
	return &r, nil
}

// Standards runs "<tool> <standard> --format json" and decodes the gap report.
// RecomputeSummary is called after decode so that tools using non-canonical
// summary key names (e.g. "addressed"/"gap") still produce correct aggregates.
//
//fusa:req REQ-FO-ADP019
func (a *cmdAdapter) Standards(ctx context.Context, root, standard string) (*standards.GapReport, error) {
	out, err := a.run(ctx, root, a.tool, standard, "--format", "json")
	if err != nil {
		return nil, fmt.Errorf("adapter %s: %s: %w", a.name, standard, err)
	}
	var r standards.GapReport
	if err := json.Unmarshal(extractJSON(out), &r); err != nil {
		return nil, fmt.Errorf("adapter %s: decode gap report: %w", a.name, err)
	}
	r.RecomputeSummary()
	return &r, nil
}

// extractJSON returns the JSON object spanning the first '{' to the last '}' in
// b. The runner returns a command's combined output, so this tolerates a tool
// that prints an incidental line to stderr around its JSON document.
func extractJSON(b []byte) []byte {
	start := bytes.IndexByte(b, '{')
	end := bytes.LastIndexByte(b, '}')
	if start < 0 || end < start {
		return b
	}
	return b[start : end+1]
}
