package main

import (
	"flag"
	"fmt"
	"io"

	"github.com/SoundMatt/FuSaOps/adapter"
)

// runAdapters lists every registered adapter and whether its tool is on PATH.
//
//fusa:req REQ-FO-CLI006
func runAdapters(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("fusaops adapters", flag.ContinueOnError)
	fs.SetOutput(stderr)
	if err := fs.Parse(args); err != nil {
		return 2
	}
	fmt.Fprintln(stdout, "Registered adapters:")
	fmt.Fprintf(stdout, "  %-10s %-8s %-6s %s\n", "NAME", "TOOL", "LANG", "AVAILABLE")
	for _, a := range adapter.Default.All() {
		avail := "no"
		if a.Available() {
			avail = "yes"
		}
		fmt.Fprintf(stdout, "  %-10s %-8s %-6s %s\n", a.Name(), a.Tool(), a.Language(), avail)
	}
	return 0
}
