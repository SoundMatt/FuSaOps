package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"time"

	fusaops "github.com/SoundMatt/FuSaOps"
	"github.com/SoundMatt/FuSaOps/orchestrator"
)

// runAuditPack bundles every applicable tool's audit-pack together with the
// FuSaOps cross-language evidence into one ZIP for auditors.
//
//fusa:req REQ-FO-CLI013
//fusa:req REQ-FO-CLI049
func runAuditPack(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("fusaops audit-pack", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dir := fs.String("dir", ".", "project root directory")
	only := fs.String("only", "", "comma-separated tool names to run (default: all applicable)")
	output := fs.String("output", "audit-pack.zip", "output path for the unified audit-pack ZIP")
	workers := fs.Int("workers", 0, "max parallel adapters (0 = unlimited; overrides config)")
	timeout := fs.String("timeout", "", "per-adapter deadline e.g. 30s, 5m (overrides config)")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	root, opts, _, err := loadOptions(*dir, *only, stderr)
	if err != nil {
		fmt.Fprintf(stderr, "fusaops audit-pack: %v\n", err)
		return 1
	}
	if *workers > 0 {
		opts.Workers = *workers
	}
	if *timeout != "" {
		d, perr := time.ParseDuration(*timeout)
		if perr != nil {
			fmt.Fprintf(stderr, "fusaops audit-pack: --timeout %q: %v\n", *timeout, perr)
			return 2
		}
		opts.Timeout = d
	}

	res, err := orchestrator.New(nil).RunAuditPack(context.Background(), root, *output, opts)
	if err != nil {
		if errors.Is(err, fusaops.ErrNoAdapters) {
			fmt.Fprintln(stderr, "fusaops audit-pack: no supported languages detected")
			return 1
		}
		fmt.Fprintf(stderr, "fusaops audit-pack: %v\n", err)
		return 1
	}

	fmt.Fprintf(stdout, "Wrote audit-pack to %s (%d files)\n", *output, len(res.Manifest.Files))
	if len(res.Packed) > 0 {
		fmt.Fprintf(stdout, "Bundled tool packs: %v\n", res.Packed)
	}
	for _, s := range res.Skipped {
		fmt.Fprintf(stdout, "  skipped: %s\n", s)
	}
	return 0
}
