package adapter

import fusaops "github.com/SoundMatt/FuSaOps"

// newGoFuSa returns the adapter for go-FuSa (Go projects).
//
//fusa:req REQ-FO-ADP010
func newGoFuSa() *cmdAdapter {
	return &cmdAdapter{
		name:       "go-FuSa",
		language:   fusaops.LangGo,
		tool:       "gofusa",
		extensions: []string{".go"},
		run:        defaultRunner,
	}
}

func init() { Default.MustRegister(newGoFuSa()) }
