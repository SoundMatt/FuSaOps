package adapter

import fusaops "github.com/SoundMatt/FuSaOps"

// newJFuSa returns the adapter for java-FuSa (Java projects).
// The generic cmdAdapter is sufficient: jfusa check/trace/qualify/release/audit-pack
// all conform to the x-FuSa spec v1.10 common CLI surface.
//
//fusa:req REQ-FO-ADP028
func newJFuSa() *cmdAdapter {
	return &cmdAdapter{
		name:       "java-FuSa",
		language:   fusaops.LangJava,
		tool:       "jfusa",
		extensions: []string{".java"},
		run:        defaultRunner,
	}
}

func init() { Default.MustRegister(newJFuSa()) }
