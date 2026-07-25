package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	fusaops "github.com/SoundMatt/FuSaOps"
	"github.com/SoundMatt/FuSaOps/config"
	"github.com/SoundMatt/FuSaOps/vv"
)

// runVV dispatches the vv subcommand (show|set).
//
//fusa:req REQ-FO-CLI074
func runVV(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("fusaops vv", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() {
		fmt.Fprintf(stderr, "Usage: fusaops vv <subcommand> [flags]\n\n")
		fmt.Fprintf(stderr, "Manage V&V independence declarations and report achievable ASIL.\n\n")
		fmt.Fprintf(stderr, "Subcommands:\n")
		fmt.Fprintf(stderr, "  show  Display V&V declarations and computed achievable ASIL\n")
		fmt.Fprintf(stderr, "  set   Update one or more independence declaration fields\n")
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
			fmt.Fprintf(stderr, "fusaops vv: get working directory: %v\n", err)
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
		return runVVShow(subArgs, projectRoot, stdout, stderr)
	case "set":
		return runVVSet(subArgs, projectRoot, stdout, stderr)
	default:
		fmt.Fprintf(stderr, "fusaops vv: unknown subcommand %q\n", sub)
		fmt.Fprintf(stderr, "Run 'fusaops vv --help' for usage.\n")
		return 1
	}
}

// runVVShow displays V&V declarations and the computed achievable ASIL.
//
//fusa:req REQ-FO-CLI075
func runVVShow(args []string, projectRoot string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("fusaops vv show", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() {
		fmt.Fprintf(stderr, "Usage: fusaops vv show [flags]\n\n")
		fmt.Fprintf(stderr, "Display V&V independence declarations and the computed achievable ASIL.\n\n")
		fmt.Fprintf(stderr, "Flags:\n")
		fs.PrintDefaults()
	}
	format := fs.String("format", "text", "output format: text or json")
	output := fs.String("output", "", "write output to file instead of stdout")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	// Load config; tolerate ErrNoConfig (show empty Declaration).
	cfgPath := filepath.Join(projectRoot, config.ConfigFile)
	cfg, err := config.Load(cfgPath)
	if err != nil && !errors.Is(err, fusaops.ErrNoConfig) {
		fmt.Fprintf(stderr, "fusaops vv show: %v\n", err)
		return 1
	}

	var decl vv.Declaration
	if cfg != nil {
		decl = vv.Declaration{
			Project:                 cfg.Project.Name,
			ImplementationAuthor:    cfg.VandV.ImplementationAuthor,
			IndependentReviewer:     cfg.VandV.IndependentReviewer,
			IndependentTestExecutor: cfg.VandV.IndependentTestExecutor,
		}
	}

	// Print any validation warnings to stderr.
	for _, iss := range vv.Validate(decl) {
		fmt.Fprintf(stderr, "vv warning: %s\n", iss)
	}

	// Determine output writer.
	w := stdout
	if *output != "" {
		f, err := os.Create(*output)
		if err != nil {
			fmt.Fprintf(stderr, "fusaops vv show: create %s: %v\n", *output, err)
			return 1
		}
		defer f.Close()
		w = f
	}

	if err := vv.Render(w, decl, *format); err != nil {
		fmt.Fprintf(stderr, "fusaops vv show: %v\n", err)
		return 1
	}
	return 0
}

// runVVSet updates one or more independence declaration fields in .fusaops.json.
//
//fusa:req REQ-FO-CLI076
func runVVSet(args []string, projectRoot string, stdout, stderr io.Writer) int {
	const sentinel = "\x00"
	fs := flag.NewFlagSet("fusaops vv set", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() {
		fmt.Fprintf(stderr, "Usage: fusaops vv set [flags]\n\n")
		fmt.Fprintf(stderr, "Update V&V independence declaration fields in .fusaops.json.\n\n")
		fmt.Fprintf(stderr, "Only supplied flags are updated; omitted flags keep their existing values.\n\n")
		fmt.Fprintf(stderr, "Flags:\n")
		fs.PrintDefaults()
	}
	author := fs.String("implementation-author", sentinel, "person or team that wrote the implementation")
	reviewer := fs.String("independent-reviewer", sentinel, "person who performed independent design review (must differ from author)")
	executor := fs.String("independent-test-executor", sentinel, "person who executed tests independently (must differ from author)")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	cfgPath := filepath.Join(projectRoot, config.ConfigFile)
	cfg, err := config.Load(cfgPath)
	if err != nil {
		if errors.Is(err, fusaops.ErrNoConfig) {
			fmt.Fprintf(stderr, "fusaops vv set: no .fusaops.json found — run 'fusaops init' first\n")
		} else {
			fmt.Fprintf(stderr, "fusaops vv set: %v\n", err)
		}
		return 1
	}

	// Apply only the flags that were explicitly provided.
	if *author != sentinel {
		cfg.VandV.ImplementationAuthor = *author
	}
	if *reviewer != sentinel {
		cfg.VandV.IndependentReviewer = *reviewer
	}
	if *executor != sentinel {
		cfg.VandV.IndependentTestExecutor = *executor
	}

	// Validate the updated declaration and print warnings.
	decl := vv.Declaration{
		Project:                 cfg.Project.Name,
		ImplementationAuthor:    cfg.VandV.ImplementationAuthor,
		IndependentReviewer:     cfg.VandV.IndependentReviewer,
		IndependentTestExecutor: cfg.VandV.IndependentTestExecutor,
	}
	for _, iss := range vv.Validate(decl) {
		fmt.Fprintf(stderr, "vv warning: %s\n", iss)
	}

	if err := config.Save(cfgPath, cfg); err != nil {
		fmt.Fprintf(stderr, "fusaops vv set: save config: %v\n", err)
		return 1
	}

	fmt.Fprintf(stdout, "Updated vv declarations. Achievable ASIL: %s\n", vv.AchievableASIL(decl))
	return 0
}
