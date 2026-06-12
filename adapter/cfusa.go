package adapter

import fusaops "github.com/SoundMatt/FuSaOps"

// cFuSaAdapter wraps cmdAdapter for c-FuSa. c-FuSa v0.3.0+ emits a spec-conformant
// sbom.json and a single-ZIP audit-pack, so no output normalisation is required;
// all capability methods (Trace, Qualify, SBOM, AuditPack, Standards) use the
// generic cmdAdapter path.
type cFuSaAdapter struct{ *cmdAdapter }

// newCFuSa returns the adapter for c-FuSa (C projects).
//
// Only C translation units (.c) mark the language; headers (.h) are ambiguous
// with C++ and are deliberately excluded from detection so a header-only C++
// project is not misclassified as C.
//
//fusa:req REQ-FO-ADP011
func newCFuSa() *cFuSaAdapter {
	return &cFuSaAdapter{&cmdAdapter{
		name:       "c-FuSa",
		language:   fusaops.LangC,
		tool:       "cfusa",
		extensions: []string{".c"},
		run:        defaultRunner,
	}}
}

func init() { Default.MustRegister(newCFuSa()) }
