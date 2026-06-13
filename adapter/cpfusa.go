package adapter

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	fusaops "github.com/SoundMatt/FuSaOps"
	"github.com/SoundMatt/FuSaOps/standards"
)

// cppFuSaAdapter wraps cmdAdapter with cpp-FuSa-specific normalizations.
// Trace uses the generic cmdAdapter path: cpp-FuSa v0.12.1+ writes canonical
// trace JSON to stdout. Standards still requires a temp-file override because
// cpp-FuSa writes gap reports to --output rather than stdout.
type cppFuSaAdapter struct {
	*cmdAdapter
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
