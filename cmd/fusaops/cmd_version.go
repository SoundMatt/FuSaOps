package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"

	fusaops "github.com/SoundMatt/FuSaOps"
)

// runVersion prints the FuSaOps version.
//
//fusa:req REQ-FO-CLI003
func runVersion(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("fusaops version", flag.ContinueOnError)
	fs.SetOutput(stderr)
	format := fs.String("format", "text", "output format: text or json")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	switch *format {
	case "text", "":
		fmt.Fprintf(stdout, "fusaops %s\n", fusaops.Version)
	case "json":
		v := struct {
			Tool        string `json:"tool"`
			Version     string `json:"version"`
			SpecVersion string `json:"specVersion"`
		}{
			Tool:        "fusaops",
			Version:     fusaops.Version,
			SpecVersion: fusaops.SpecVersion,
		}
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(v); err != nil {
			fmt.Fprintf(stderr, "fusaops version: %v\n", err)
			return 1
		}
	default:
		fmt.Fprintf(stderr, "fusaops version: unknown format %q (want text or json)\n", *format)
		return 2
	}
	return 0
}
