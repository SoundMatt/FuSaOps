package adapter

import fusaops "github.com/SoundMatt/FuSaOps"

// newCppFuSa returns the adapter for cpp-FuSa (C++ projects).
//
//fusa:req REQ-FO-ADP012
func newCppFuSa() *cmdAdapter {
	return &cmdAdapter{
		name:       "cpp-FuSa",
		language:   fusaops.LangCpp,
		tool:       "cpfusa",
		extensions: []string{".cpp", ".cc", ".cxx", ".hpp", ".hh"},
		run:        defaultRunner,
	}
}

func init() { Default.MustRegister(newCppFuSa()) }
