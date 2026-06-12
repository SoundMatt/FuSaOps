package adapter

import fusaops "github.com/SoundMatt/FuSaOps"

// newPyFuSa returns the adapter for py-FuSa (Python projects).
// The generic cmdAdapter is sufficient: pyfusa check/trace/qualify/release/audit-pack
// all conform to the x-FuSa spec v1.9 common CLI surface.
//
//fusa:req REQ-FO-ADP027
func newPyFuSa() *cmdAdapter {
	return &cmdAdapter{
		name:       "py-FuSa",
		language:   fusaops.LangPython,
		tool:       "pyfusa",
		extensions: []string{".py"},
		run:        defaultRunner,
	}
}

func init() { Default.MustRegister(newPyFuSa()) }
