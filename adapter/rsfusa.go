package adapter

import fusaops "github.com/SoundMatt/FuSaOps"

// newRustFuSa returns the adapter for rust-FuSa (Rust projects).
// The generic cmdAdapter is sufficient: rsfusa check/trace/qualify/release/audit-pack
// all conform to the x-FuSa spec v1.9 common CLI surface.
//
//fusa:req REQ-FO-ADP026
func newRustFuSa() *cmdAdapter {
	return &cmdAdapter{
		name:       "rust-FuSa",
		language:   fusaops.LangRust,
		tool:       "rsfusa",
		extensions: []string{".rs"},
		run:        defaultRunner,
	}
}

func init() { Default.MustRegister(newRustFuSa()) }
