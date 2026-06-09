package main

import (
	"fmt"
	"io"

	fusaops "github.com/SoundMatt/FuSaOps"
)

// runVersion prints the FuSaOps version.
//
//fusa:req REQ-FO-CLI003
func runVersion(_ []string, stdout, _ io.Writer) int {
	fmt.Fprintf(stdout, "fusaops %s\n", fusaops.Version)
	return 0
}
