package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/SoundMatt/FuSaOps/slsa"
)

// runSLSA generates a SLSA supply-chain gap report for a FuSaOps project.
//
//fusa:req REQ-FO-CLI057
func runSLSA(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("fusaops slsa", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() {
		fmt.Fprintf(stderr, "Usage: fusaops slsa [flags]\n\n")
		fmt.Fprintf(stderr, "Generate a SLSA v1.0 supply-chain integrity gap report.\n\n")
		fmt.Fprintf(stderr, "Flags:\n")
		fs.PrintDefaults()
	}
	dir := fs.String("dir", ".", "project root directory")
	level := fs.String("level", "L2", "SLSA level: L1, L2, L3, L4")
	format := fs.String("format", "text", "output format: text or json")
	output := fs.String("output", "", "write report to file (default: stdout)")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	validLevels := map[string]bool{"L1": true, "L2": true, "L3": true, "L4": true}
	if !validLevels[*level] {
		fmt.Fprintf(stderr, "fusaops slsa: unknown level %q (must be L1, L2, L3, or L4)\n", *level)
		return 2
	}

	project := filepath.Base(*dir)

	rep, err := slsa.Assess(*dir, project, slsa.Level(*level))
	if err != nil {
		fmt.Fprintf(stderr, "fusaops slsa: assess: %v\n", err)
		return 1
	}

	w := stdout
	if *output != "" {
		f, ferr := os.Create(*output)
		if ferr != nil {
			fmt.Fprintf(stderr, "fusaops slsa: create %s: %v\n", *output, ferr)
			return 1
		}
		defer func() { _ = f.Close() }()
		w = f
	}

	if err := slsa.Render(w, rep, *format); err != nil {
		fmt.Fprintf(stderr, "fusaops slsa: render: %v\n", err)
		return 1
	}

	if *output != "" {
		fmt.Fprintf(stderr, "SLSA gap report written to %s (%d gap(s))\n", *output, rep.Gap)
	}

	if rep.Gap > 0 {
		return 1
	}
	return 0
}
