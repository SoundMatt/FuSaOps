package main

import (
	"context"
	"flag"
	"fmt"
	"io"

	"github.com/SoundMatt/FuSaOps/orchestrator"
	"github.com/SoundMatt/FuSaOps/policy"
)

// runPolicy evaluates a policy against the current scan report.
//
//fusa:req REQ-FO-CLI024
func runPolicy(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("fusaops policy", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dir := fs.String("dir", ".", "project root directory")
	only := fs.String("only", "", "comma-separated tool names to run (default: all applicable)")
	policyFile := fs.String("policy", "policy.json", "policy configuration file")
	format := fs.String("format", "text", "output format: text or json")
	output := fs.String("output", "", "write report to file (default: stdout)")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	p, err := policy.LoadPolicy(*policyFile)
	if err != nil {
		fmt.Fprintf(stderr, "fusaops policy: %v\n", err)
		return 1
	}

	root, opts, _, err := loadOptions(*dir, *only, stderr)
	if err != nil {
		fmt.Fprintf(stderr, "fusaops policy: %v\n", err)
		return 1
	}

	rep, err := orchestrator.New(nil).Run(context.Background(), root, opts)
	if err != nil {
		fmt.Fprintf(stderr, "fusaops policy: scan failed: %v\n", err)
		return 1
	}

	pr := policy.Evaluate(p, rep)

	if err := policy.RenderToFile(stdout, pr, *format, *output); err != nil {
		fmt.Fprintf(stderr, "fusaops policy: %v\n", err)
		return 1
	}

	if pr.HasFailures() {
		return 1
	}
	return 0
}
