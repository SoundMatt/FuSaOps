package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/SoundMatt/FuSaOps/vuln"
)

// runVuln scans dependency manifests for known vulnerabilities via osv-scanner.
//
//fusa:req REQ-FO-CLI071
func runVuln(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("fusaops vuln", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() {
		fmt.Fprintf(stderr, "Usage: fusaops vuln [flags]\n\n")
		fmt.Fprintf(stderr, "Scan dependency manifests (go.mod, Cargo.toml, requirements.txt,\n")
		fmt.Fprintf(stderr, "package.json, pom.xml) for known vulnerabilities.\n\n")
		fmt.Fprintf(stderr, "When osv-scanner is available on PATH, it is invoked to check each\n")
		fmt.Fprintf(stderr, "manifest against the OSV vulnerability database. Otherwise the manifests\n")
		fmt.Fprintf(stderr, "are discovered and reported without a vulnerability check.\n\n")
		fmt.Fprintf(stderr, "Exits 1 if any vulnerabilities are found.\n\n")
		fmt.Fprintf(stderr, "Flags:\n")
		fs.PrintDefaults()
	}

	var (
		dir    = fs.String("dir", "", "project root directory (default: current directory)")
		output = fs.String("output", "", "path for the vulnerability report (default: <dir>/.fusaops-vuln.json)")
		format = fs.String("format", "text", "output format: text, json")
	)
	if err := fs.Parse(args); err != nil {
		return 2
	}

	projectRoot := *dir
	if projectRoot == "" {
		var err error
		projectRoot, err = os.Getwd()
		if err != nil {
			fmt.Fprintf(stderr, "fusaops vuln: get working directory: %v\n", err)
			return 1
		}
	}

	r, err := vuln.Scan(projectRoot, nil) // nil → real osv-scanner on PATH
	if err != nil {
		fmt.Fprintf(stderr, "fusaops vuln: scan: %v\n", err)
		return 1
	}

	if renderErr := vuln.Render(stdout, r, *format); renderErr != nil {
		fmt.Fprintf(stderr, "fusaops vuln: render: %v\n", renderErr)
		return 2
	}

	outPath := *output
	if outPath == "" {
		outPath = filepath.Join(projectRoot, vuln.ReportFile)
	}
	if saveErr := vuln.Save(outPath, r); saveErr != nil {
		fmt.Fprintf(stderr, "fusaops vuln: save: %v\n", saveErr)
		return 1
	}
	fmt.Fprintf(stdout, "\nVulnerability report written to %s\n", outPath)
	fmt.Fprintf(stdout, "Integrity hash: %s\n", r.Hash)

	if r.HasFindings() {
		return 1
	}
	return 0
}
