package main

import (
	"context"
	"flag"
	"fmt"
	"io"

	"github.com/SoundMatt/FuSaOps/fleet"
	"github.com/SoundMatt/FuSaOps/orchestrator"
)

// runFleet runs fusaops check across all repos listed in a fleet config.
//
//fusa:req REQ-FO-CLI023
func runFleet(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("fusaops fleet", flag.ContinueOnError)
	fs.SetOutput(stderr)
	config := fs.String("config", "fleet.json", "fleet configuration file")
	format := fs.String("format", "text", "output format: text or json")
	output := fs.String("output", "", "write report to file (default: stdout)")
	strict := fs.Bool("strict", false, "exit 1 on warnings in addition to errors")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	cfg, err := fleet.LoadConfig(*config)
	if err != nil {
		fmt.Fprintf(stderr, "fusaops fleet: %v\n", err)
		return 1
	}
	if len(cfg.Repos) == 0 {
		fmt.Fprintf(stderr, "fusaops fleet: no repos defined in %s\n", *config)
		return 1
	}

	fr := fleet.Run(context.Background(), cfg, orchestrator.New(nil))

	if err := fleet.RenderToFile(stdout, fr, *format, *output); err != nil {
		fmt.Fprintf(stderr, "fusaops fleet: %v\n", err)
		return 1
	}

	if fr.HasFailures() {
		return 1
	}
	if *strict && fr.HasWarnings() {
		return 1
	}
	return 0
}
