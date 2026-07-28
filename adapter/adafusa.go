package adapter

import fusaops "github.com/SoundMatt/FuSaOps"

// newAdaFuSa returns the adapter for ada-FuSa (Ada/SPARK projects).
// The generic cmdAdapter is sufficient: adafusa check/trace conform to the
// x-FuSa spec v1.11 common CLI surface.
//
//fusa:req REQ-FO-ADP031
func newAdaFuSa() *cmdAdapter {
	return &cmdAdapter{
		name:       "ada-FuSa",
		language:   fusaops.LangAda,
		tool:       "adafusa",
		extensions: []string{".ads", ".adb"},
		run:        defaultRunner,
	}
}

func init() { Default.MustRegister(newAdaFuSa()) }
