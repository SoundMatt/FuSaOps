package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"

	fusaops "github.com/SoundMatt/FuSaOps"
	"github.com/SoundMatt/FuSaOps/orchestrator"
	"github.com/SoundMatt/FuSaOps/standards"
)

// runStandards implements `fusaops iso26262`, `fusaops iec61508`, `fusaops do178`,
// `fusaops iso21434`, `fusaops unece`, and `fusaops iec62443`. cmd is the CLI
// name; standard is the canonical §2.4.1 id returned by CommandStandard.
//
//fusa:req REQ-FO-CLI015
//fusa:req REQ-FO-CLI016
//fusa:req REQ-FO-CLI017
//fusa:req REQ-FO-CLI019
//fusa:req REQ-FO-CLI020
//fusa:req REQ-FO-CLI022
func runStandards(cmd string, args []string, stdout, stderr io.Writer) int {
	standard := standards.CommandStandard(cmd)

	fs := flag.NewFlagSet(cmd, flag.ContinueOnError)
	fs.SetOutput(stderr)
	dir := fs.String("dir", ".", "Project root")
	only := fs.String("only", "", "Comma-separated tool names to run (default: all applicable)")
	format := fs.String("format", "text", "Output format: text|json|html")
	output := fs.String("output", "", "Write report to file (default: stdout)")
	strict := fs.Bool("strict", false, "Exit 1 if any objective has gap status")
	fs.Usage = func() {
		fmt.Fprintf(stderr, `Usage: fusaops %s [flags]

Roll up %s gap reports from every applicable installed x-FuSa tool and
print the cross-language compliance matrix.

Flags:
  --dir <path>              Project root (default: .)
  --only <tools>            Comma-separated tool names to run
  --format text|json|html   Output format (default: text)
  --output <file>           Write report to file (default: stdout)
  --strict                  Exit 1 if any objective has gap status

Exit codes:
  0  all components satisfy or partially satisfy objectives (or are skipped)
  1  one or more objectives have gap status under --strict; or a run error
  2  usage error

`, cmd, standards.DisplayName(standard))
	}
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *format != "text" && *format != "json" && *format != "html" {
		fmt.Fprintf(stderr, "fusaops %s: unsupported format %q\n", cmd, *format)
		return 2
	}

	root, opts, _, err := loadOptions(*dir, *only, stderr)
	if err != nil {
		fmt.Fprintf(stderr, "fusaops %s: %v\n", cmd, err)
		return 1
	}

	agg, err := orchestrator.New(nil).RunStandards(context.Background(), root, standard, opts)
	if err != nil {
		if errors.Is(err, fusaops.ErrNoAdapters) {
			fmt.Fprintf(stderr, "fusaops %s: no supported languages detected\n", cmd)
			return 1
		}
		fmt.Fprintf(stderr, "fusaops %s: %v\n", cmd, err)
		return 1
	}

	if *output == "" {
		if err := standards.Render(stdout, agg, *format); err != nil {
			fmt.Fprintf(stderr, "fusaops %s: %v\n", cmd, err)
			return 1
		}
	} else {
		if err := standards.RenderToFile(agg, *format, *output); err != nil {
			fmt.Fprintf(stderr, "fusaops %s: %v\n", cmd, err)
			return 1
		}
		fmt.Fprintf(stdout, "Wrote %s %s report to %s\n", *format, standards.DisplayName(standard), *output)
	}

	if *strict && agg.HasGaps() {
		return 1
	}
	return 0
}
