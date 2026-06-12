package adapter

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	fusaops "github.com/SoundMatt/FuSaOps"
	"github.com/SoundMatt/FuSaOps/standards"
	"github.com/SoundMatt/FuSaOps/trace"
)

// cppFuSaAdapter wraps cmdAdapter with cpp-FuSa-specific normalizations.
// Check uses the generic cmdAdapter path unchanged.
type cppFuSaAdapter struct {
	*cmdAdapter
}

// Trace runs "cpfusa trace --format json --output <tmp>" and normalizes
// cpp-FuSa's non-conformant output to trace.Matrix (cpp-FuSa issue #3).
// cpp-FuSa writes trace JSON to a file rather than stdout, uses per-requirement
// nested tags[] rather than the spec's flat top-level tags[], and emits tag
// kind "req" instead of "impl".
//
//fusa:req REQ-FO-ADP024
func (a *cppFuSaAdapter) Trace(ctx context.Context, root string) (*trace.Matrix, error) {
	tmp, err := os.CreateTemp("", "fusaops-cpfusa-trace-*.json")
	if err != nil {
		return nil, fmt.Errorf("adapter %s: temp file: %w", a.name, err)
	}
	out := tmp.Name()
	_ = tmp.Close()
	defer func() { _ = os.Remove(out) }()

	if _, err = a.run(ctx, root, a.tool, "trace", "--format", "json", "--output", out); err != nil {
		return nil, fmt.Errorf("adapter %s: trace: %w", a.name, err)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		return nil, fmt.Errorf("adapter %s: read trace: %w", a.name, err)
	}
	return parseCppFuSaTrace(data, a.name)
}

// parseCppFuSaTrace decodes cpp-FuSa's trace JSON (v0.10.0+ format) into
// trace.Matrix. cpp-FuSa nests tags[] inside each requirements entry instead of
// at the top level, and uses kind "req" where the spec requires "impl". Both are
// normalised here pending the fix in cpp-FuSa issue #3.
func parseCppFuSaTrace(data []byte, tool string) (*trace.Matrix, error) {
	var raw struct {
		Requirements []struct {
			ID          string `json:"id"`
			Title       string `json:"title"`
			StandardRef string `json:"standardRef"`
			Tags        []struct {
				RequirementID string `json:"requirementId"`
				File          string `json:"file"`
				Line          int    `json:"line"`
				Kind          string `json:"kind"`
			} `json:"tags"`
		} `json:"requirements"`
		Summary struct {
			Total     int `json:"total"`
			Annotated int `json:"annotated"`
			Tested    int `json:"tested"`
		} `json:"summary"`
	}
	if err := json.Unmarshal(extractJSON(data), &raw); err != nil {
		return nil, fmt.Errorf("adapter %s: decode trace: %w", tool, err)
	}
	m := &trace.Matrix{
		Coverage: trace.Coverage{
			TotalRequirements:  raw.Summary.Total,
			TracedRequirements: raw.Summary.Annotated,
			TestedRequirements: raw.Summary.Tested,
		},
	}
	for _, req := range raw.Requirements {
		m.Requirements = append(m.Requirements, trace.Requirement{
			ID:       req.ID,
			Title:    req.Title,
			Standard: req.StandardRef,
		})
		for _, tag := range req.Tags {
			kind := tag.Kind
			if kind == "req" {
				kind = "impl" // cpp-FuSa issue #3: "req" → "impl"
			}
			m.Tags = append(m.Tags, trace.Tag{
				RequirementID: req.ID,
				File:          tag.File,
				Line:          tag.Line,
				Kind:          kind,
			})
		}
	}
	return m, nil
}

// Standards runs "cpfusa <standard> --output <tmp>" and reads the JSON report.
// cpp-FuSa writes gap reports to a file rather than stdout; it does not accept
// --format json. cppStandardCmd maps canonical ids to cpp-FuSa subcommand names.
//
//fusa:req REQ-FO-ADP025
func (a *cppFuSaAdapter) Standards(ctx context.Context, root, standard string) (*standards.GapReport, error) {
	tmp, err := os.CreateTemp("", "fusaops-cpfusa-"+standard+"-*.json")
	if err != nil {
		return nil, fmt.Errorf("adapter %s: temp file: %w", a.name, err)
	}
	out := tmp.Name()
	_ = tmp.Close()
	defer func() { _ = os.Remove(out) }()

	cmd := cppStandardCmd(standard)
	if _, err = a.run(ctx, root, a.tool, cmd, "--output", out); err != nil {
		return nil, fmt.Errorf("adapter %s: %s: %w", a.name, standard, err)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		return nil, fmt.Errorf("adapter %s: read %s report: %w", a.name, standard, err)
	}
	var r standards.GapReport
	if err := json.Unmarshal(extractJSON(data), &r); err != nil {
		return nil, fmt.Errorf("adapter %s: decode %s report: %w", a.name, standard, err)
	}
	r.RecomputeSummary()
	return &r, nil
}

// cppStandardCmd maps a canonical §2.4.1 standard id to cpp-FuSa's subcommand
// name where they differ.
func cppStandardCmd(standard string) string {
	if standard == "do178c" {
		return "do178"
	}
	return standard
}

// newCppFuSa returns the adapter for cpp-FuSa (C++ projects).
//
//fusa:req REQ-FO-ADP012
func newCppFuSa() *cppFuSaAdapter {
	return &cppFuSaAdapter{&cmdAdapter{
		name:       "cpp-FuSa",
		language:   fusaops.LangCpp,
		tool:       "cpfusa",
		extensions: []string{".cpp", ".cc", ".cxx", ".hpp", ".hh"},
		run:        defaultRunner,
	}}
}

func init() { Default.MustRegister(newCppFuSa()) }
