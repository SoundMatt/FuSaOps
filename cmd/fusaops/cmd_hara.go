package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/SoundMatt/FuSaOps/hara"
)

// runHara dispatches the hara subcommand (show|init|asil).
//
//fusa:req REQ-FO-CLI073
func runHara(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("fusaops hara", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() {
		fmt.Fprintf(stderr, "Usage: fusaops hara <subcommand> [flags]\n\n")
		fmt.Fprintf(stderr, "Manage the Hazard Analysis and Risk Assessment (.fusa-hara.json).\n\n")
		fmt.Fprintf(stderr, "Subcommands:\n")
		fmt.Fprintf(stderr, "  show    Display HARA as text, json, or markdown\n")
		fmt.Fprintf(stderr, "  init    Create a starter .fusa-hara.json\n")
		fmt.Fprintf(stderr, "  asil    Derive ASIL from S/E/C per ISO 26262-3:2018 Table 4\n")
		fmt.Fprintf(stderr, "\nFlags:\n")
		fs.PrintDefaults()
	}
	dir := fs.String("dir", "", "project root directory (default: current directory)")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	projectRoot := *dir
	if projectRoot == "" {
		var err error
		projectRoot, err = os.Getwd()
		if err != nil {
			fmt.Fprintf(stderr, "fusaops hara: get working directory: %v\n", err)
			return 1
		}
	}

	sub := fs.Arg(0)
	subArgs := fs.Args()
	if len(subArgs) > 0 {
		subArgs = subArgs[1:]
	}

	switch sub {
	case "", "show":
		return runHaraShow(subArgs, projectRoot, stdout, stderr)
	case "init":
		return runHaraInit(subArgs, projectRoot, stdout, stderr)
	case "asil":
		return runHaraASIL(subArgs, stdout, stderr)
	default:
		fmt.Fprintf(stderr, "fusaops hara: unknown subcommand %q\n", sub)
		fmt.Fprintf(stderr, "Run 'fusaops hara --help' for usage.\n")
		return 2
	}
}

func runHaraShow(args []string, projectRoot string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("fusaops hara show", flag.ContinueOnError)
	fs.SetOutput(stderr)
	format := fs.String("format", "text", "output format: text, json, markdown")
	output := fs.String("output", "", "write output to file (default: stdout)")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	h, err := hara.Load(projectRoot)
	if err != nil {
		fmt.Fprintf(stderr, "fusaops hara show: %v\n", err)
		return 1
	}

	w := stdout
	if *output != "" {
		f, ferr := os.Create(*output)
		if ferr != nil {
			fmt.Fprintf(stderr, "fusaops hara show: create %s: %v\n", *output, ferr)
			return 1
		}
		defer func() { _ = f.Close() }()
		w = f
	}

	if err := hara.Render(w, h, *format); err != nil {
		fmt.Fprintf(stderr, "fusaops hara show: render: %v\n", err)
		return 2
	}

	findings := hara.Validate(h)
	if len(findings) > 0 && *output != "" {
		fmt.Fprintf(stderr, "fusaops hara: %d gap(s) found — run 'fusaops hara show' for details\n", len(findings))
	}
	return 0
}

func runHaraInit(args []string, projectRoot string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("fusaops hara init", flag.ContinueOnError)
	fs.SetOutput(stderr)
	project := fs.String("project", "", "project name (default: directory name)")
	standard := fs.String("standard", "ISO 26262", "safety standard (e.g. 'ISO 26262', 'IEC 61508')")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	path := filepath.Join(projectRoot, hara.HARAFile)
	if _, err := os.Stat(path); err == nil {
		fmt.Fprintf(stderr, "fusaops hara init: %s already exists — delete it first to reinitialise\n", hara.HARAFile)
		return 2
	}

	name := *project
	if name == "" {
		name = filepath.Base(projectRoot)
	}

	// Per x-FuSa spec §1.2.5/§1.6 rule 1: a scaffold MUST emit empty
	// collections, never a dummy row — placeholder text asserts a false
	// completeness that a real "empty, honestly incomplete" state does not.
	h := &hara.HARA{
		Project:     name,
		Standard:    *standard,
		CreatedAt:   time.Now().UTC(),
		Situations:  []hara.OperationalSituation{},
		Hazards:     []hara.Hazard{},
		SafetyGoals: []hara.SafetyGoal{},
	}

	if err := hara.Save(path, h); err != nil {
		fmt.Fprintf(stderr, "fusaops hara init: %v\n", err)
		return 1
	}

	fmt.Fprintf(stdout, "Created %s (project=%s standard=%q)\n", path, name, *standard)
	fmt.Fprintf(stdout, "Edit %s to document project hazards and safety goals.\n", hara.HARAFile)
	return 0
}

func runHaraASIL(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("fusaops hara asil", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() {
		fmt.Fprintf(stderr, "Usage: fusaops hara asil [flags]\n\n")
		fmt.Fprintf(stderr, "Derive ASIL from ISO 26262-3:2018 Table 4.\n\n")
		fmt.Fprintf(stderr, "Flags:\n")
		fs.PrintDefaults()
		fmt.Fprintf(stderr, "\nExample: fusaops hara asil -s S2 -e E3 -c C2\n")
	}
	s := fs.String("s", "", "Severity: S0, S1, S2, S3 (required)")
	e := fs.String("e", "", "Exposure: E0, E1, E2, E3, E4 (required)")
	c := fs.String("c", "", "Controllability: C0, C1, C2, C3 (required)")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	if *s == "" || *e == "" || *c == "" {
		fmt.Fprintf(stderr, "fusaops hara asil: -s, -e, and -c are required\n")
		fs.Usage()
		return 2
	}

	asil := hara.DetermineASIL(hara.Severity(*s), hara.Exposure(*e), hara.Controllability(*c))
	fmt.Fprintf(stdout, "S=%s  E=%s  C=%s  →  %s\n", *s, *e, *c, asil)
	return 0
}
