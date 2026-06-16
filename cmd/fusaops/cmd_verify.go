package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/SoundMatt/FuSaOps/verify"
)

// runVerify executes go test -json and saves a test evidence bundle.
//
//fusa:req REQ-FO-CLI062
func runVerify(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("fusaops verify", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() {
		fmt.Fprintf(stderr, "Usage: fusaops verify [flags]\n\n")
		fmt.Fprintf(stderr, "Run go test and save a test evidence bundle to .fusaops-evidence.json.\n\n")
		fmt.Fprintf(stderr, "Flags:\n")
		fs.PrintDefaults()
	}

	var (
		dir    = fs.String("dir", "", "project root directory (default: current directory)")
		output = fs.String("output", "", "evidence bundle path (default: <dir>/.fusaops-evidence.json)")
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
			fmt.Fprintf(stderr, "fusaops verify: get working directory: %v\n", err)
			return 1
		}
	}

	fmt.Fprintf(stdout, "Running go test -json -count=1 ./...\n")
	results, err := verify.Run(context.Background(), projectRoot)
	if err != nil {
		fmt.Fprintf(stderr, "fusaops verify: run tests: %v\n", err)
		return 1
	}

	bundle := verify.New(projectRoot, results)
	outPath := *output
	if outPath == "" {
		outPath = filepath.Join(projectRoot, verify.BundleFile)
	}
	if err := verify.Save(outPath, bundle); err != nil {
		fmt.Fprintf(stderr, "fusaops verify: save bundle: %v\n", err)
		return 1
	}

	if err := verify.Render(stdout, bundle, *format); err != nil {
		fmt.Fprintf(stderr, "fusaops verify: render: %v\n", err)
		return 2
	}
	fmt.Fprintf(stdout, "Evidence bundle written to %s\n", outPath)

	if bundle.Summary.Failed > 0 {
		return 1
	}
	return 0
}
