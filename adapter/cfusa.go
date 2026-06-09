package adapter

import fusaops "github.com/SoundMatt/FuSaOps"

// newCFuSa returns the adapter for c-FuSa (C projects).
//
// Only C translation units (.c) mark the language; headers (.h) are ambiguous
// with C++ and are deliberately excluded from detection so a header-only C++
// project is not misclassified as C.
//
//fusa:req REQ-FO-ADP011
func newCFuSa() *cmdAdapter {
	return &cmdAdapter{
		name:       "c-FuSa",
		language:   fusaops.LangC,
		tool:       "cfusa",
		extensions: []string{".c"},
		run:        defaultRunner,
	}
}

func init() { Default.MustRegister(newCFuSa()) }
