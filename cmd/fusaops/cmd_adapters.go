package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"

	"github.com/SoundMatt/FuSaOps/adapter"
)

// runAdapters lists every registered adapter and whether its tool is on PATH.
//
//fusa:req REQ-FO-CLI006
//fusa:req REQ-FO-CLI050
func runAdapters(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("fusaops adapters", flag.ContinueOnError)
	fs.SetOutput(stderr)
	format := fs.String("format", "text", "output format: text or json")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	all := adapter.Default.All()

	if *format == "json" {
		type adapterInfo struct {
			Name      string `json:"name"`
			Tool      string `json:"tool"`
			Language  string `json:"language"`
			Available bool   `json:"available"`
		}
		infos := make([]adapterInfo, 0, len(all))
		for _, a := range all {
			infos = append(infos, adapterInfo{
				Name:      a.Name(),
				Tool:      a.Tool(),
				Language:  string(a.Language()),
				Available: a.Available(),
			})
		}
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(infos); err != nil {
			fmt.Fprintf(stderr, "fusaops adapters: json encode: %v\n", err)
			return 1
		}
		return 0
	}

	fmt.Fprintln(stdout, "Registered adapters:")
	fmt.Fprintf(stdout, "  %-10s %-8s %-6s %s\n", "NAME", "TOOL", "LANG", "AVAILABLE")
	for _, a := range all {
		avail := "no"
		if a.Available() {
			avail = "yes"
		}
		fmt.Fprintf(stdout, "  %-10s %-8s %-6s %s\n", a.Name(), a.Tool(), a.Language(), avail)
	}
	return 0
}
