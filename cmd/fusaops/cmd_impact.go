package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/SoundMatt/FuSaOps/impact"
)

// runImpact analyses the effect of source changes on requirements and safety artefacts.
//
//fusa:req REQ-FO-CLI059
func runImpact(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("fusaops impact", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() {
		fmt.Fprintf(stderr, "Usage: fusaops impact [flags]\n\n")
		fmt.Fprintf(stderr, "Analyse the impact of source changes on requirements and safety artefacts.\n")
		fmt.Fprintf(stderr, "Uses git diff to identify changed files, then cross-references fusa:req/fusa:test\n")
		fmt.Fprintf(stderr, "annotations to find impacted requirements and stale evidence artefacts.\n\n")
		fmt.Fprintf(stderr, "Flags:\n")
		fs.PrintDefaults()
	}
	dir := fs.String("dir", ".", "project root directory")
	from := fs.String("from", "", "from git ref (default: diff working tree vs HEAD)")
	to := fs.String("to", "", "to git ref (default: HEAD)")
	format := fs.String("format", "text", "output format: text or json")
	output := fs.String("output", "", "write report to file (default: stdout)")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	rep, err := impact.Analyse(*dir, *from, *to)
	if err != nil {
		fmt.Fprintf(stderr, "fusaops impact: %v\n", err)
		return 1
	}

	w := stdout
	if *output != "" {
		f, ferr := os.Create(*output)
		if ferr != nil {
			fmt.Fprintf(stderr, "fusaops impact: create %s: %v\n", *output, ferr)
			return 1
		}
		defer func() { _ = f.Close() }()
		w = f
	}

	if err := impact.Render(w, rep, *format); err != nil {
		fmt.Fprintf(stderr, "fusaops impact: render: %v\n", err)
		return 1
	}

	if *output != "" {
		stale := 0
		for _, a := range rep.StaleArtifacts {
			if a.Stale {
				stale++
			}
		}
		fmt.Fprintf(stdout, "Impact report written to %s (%d changed files, %d impacted reqs, %d stale artefacts)\n",
			filepath.Base(*output), len(rep.ChangedFiles), len(rep.ImpactedReqs), stale)
	}
	return 0
}
