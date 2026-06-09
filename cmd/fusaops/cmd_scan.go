package main

import (
	"flag"
	"fmt"
	"io"
	"path/filepath"

	"github.com/SoundMatt/FuSaOps/adapter"
	"github.com/SoundMatt/FuSaOps/scan"
)

// runScan detects languages in a repo and lists which adapters apply.
//
//fusa:req REQ-FO-CLI005
func runScan(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("fusaops scan", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dir := fs.String("dir", ".", "project root directory")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	root, err := filepath.Abs(*dir)
	if err != nil {
		fmt.Fprintf(stderr, "fusaops scan: %v\n", err)
		return 1
	}

	res, err := scan.Scan(root)
	if err != nil {
		fmt.Fprintf(stderr, "fusaops scan: %v\n", err)
		return 1
	}

	fmt.Fprintf(stdout, "Scan of %s\n\n", root)
	if len(res.Stats) == 0 {
		fmt.Fprintln(stdout, "No supported languages detected.")
		return 0
	}
	fmt.Fprintln(stdout, "Detected languages:")
	for _, s := range res.Stats {
		fmt.Fprintf(stdout, "  %-5s %d source file(s)\n", s.Language, s.Files)
	}

	applicable, err := adapter.Default.Applicable(root)
	if err != nil {
		fmt.Fprintf(stderr, "fusaops scan: %v\n", err)
		return 1
	}
	fmt.Fprintln(stdout, "\nApplicable adapters:")
	for _, a := range applicable {
		status := "installed"
		if !a.Available() {
			status = "NOT installed"
		}
		fmt.Fprintf(stdout, "  %-10s %-8s (%s)\n", a.Name(), a.Tool(), status)
	}
	return 0
}
