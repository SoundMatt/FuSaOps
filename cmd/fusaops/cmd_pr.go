package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/SoundMatt/FuSaOps/pr"
)

// runPR dispatches fusaops pr subcommands.
//
//fusa:req REQ-FO-CLI061
func runPR(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("fusaops pr", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() {
		fmt.Fprintf(stderr, "Usage: fusaops pr <subcommand> [flags]\n\n")
		fmt.Fprintf(stderr, "Manage software problem reports (DO-178C §11.17).\n\n")
		fmt.Fprintf(stderr, "Subcommands:\n")
		fmt.Fprintf(stderr, "  init   Create an empty .fusaops-problems.json\n")
		fmt.Fprintf(stderr, "  add    Add a new problem report\n")
		fmt.Fprintf(stderr, "  list   List all problem reports\n")
		fmt.Fprintf(stderr, "  close  Close a problem report\n\n")
		fmt.Fprintf(stderr, "Flags:\n")
		fs.PrintDefaults()
	}
	dir := fs.String("dir", ".", "project root directory")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	sub := fs.Args()
	if len(sub) == 0 {
		fs.Usage()
		return 2
	}
	switch sub[0] {
	case "init":
		return prInit(*dir, stdout, stderr)
	case "add":
		return prAdd(sub[1:], *dir, stdout, stderr)
	case "list":
		return prList(sub[1:], *dir, stdout, stderr)
	case "close":
		return prClose(sub[1:], *dir, stdout, stderr)
	default:
		fmt.Fprintf(stderr, "fusaops pr: unknown subcommand %q (want init, add, list, or close)\n", sub[0])
		return 2
	}
}

//fusa:req REQ-FO-CLI061
func prInit(dir string, stdout, stderr io.Writer) int {
	path := filepath.Join(dir, pr.ProblemsFile)
	if _, err := os.Stat(path); err == nil {
		fmt.Fprintf(stderr, "fusaops pr init: %s already exists\n", pr.ProblemsFile)
		return 2
	}
	project := filepath.Base(dir)
	if dir == "." {
		if cwd, err := os.Getwd(); err == nil {
			project = filepath.Base(cwd)
		}
	}
	log := &pr.Log{Project: project}
	if err := pr.Save(dir, log); err != nil {
		fmt.Fprintf(stderr, "fusaops pr init: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "Problem report log created: %s\n", path)
	return 0
}

//fusa:req REQ-FO-CLI061
func prAdd(args []string, dir string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("fusaops pr add", flag.ContinueOnError)
	fs.SetOutput(stderr)
	id := fs.String("id", "", "problem report ID (required)")
	title := fs.String("title", "", "short description (required)")
	desc := fs.String("desc", "", "detailed description")
	phase := fs.String("phase", string(pr.PhaseDevelopment), "phase found: planning/development/verification/integration/operation")
	severity := fs.String("severity", string(pr.PRSeverityMinor), "severity: critical/major/minor")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *id == "" {
		fmt.Fprintln(stderr, "fusaops pr add: --id is required")
		return 2
	}
	if *title == "" {
		fmt.Fprintln(stderr, "fusaops pr add: --title is required")
		return 2
	}

	log, err := pr.Load(dir)
	if err != nil {
		fmt.Fprintf(stderr, "fusaops pr add: load: %v\n", err)
		return 1
	}

	report := pr.ProblemReport{
		ID:          *id,
		Title:       *title,
		Description: *desc,
		PhaseFound:  pr.Phase(*phase),
		Severity:    pr.PRSeverity(*severity),
		Status:      pr.StatusOpen,
	}
	log = pr.Add(log, report)

	if err := pr.Save(dir, log); err != nil {
		fmt.Fprintf(stderr, "fusaops pr add: save: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "Problem report %s added to %s\n", *id, pr.ProblemsFile)
	return 0
}

//fusa:req REQ-FO-CLI061
func prList(args []string, dir string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("fusaops pr list", flag.ContinueOnError)
	fs.SetOutput(stderr)
	format := fs.String("format", "text", "output format: text or json")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	log, err := pr.Load(dir)
	if err != nil {
		fmt.Fprintf(stderr, "fusaops pr list: %v\n", err)
		return 1
	}
	if err := pr.Render(stdout, log, *format); err != nil {
		fmt.Fprintf(stderr, "fusaops pr list: %v\n", err)
		return 1
	}
	return 0
}

//fusa:req REQ-FO-CLI061
func prClose(args []string, dir string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("fusaops pr close", flag.ContinueOnError)
	fs.SetOutput(stderr)
	id := fs.String("id", "", "problem report ID to close (required)")
	resolution := fs.String("resolution", "", "resolution description")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *id == "" {
		fmt.Fprintln(stderr, "fusaops pr close: --id is required")
		return 2
	}

	log, err := pr.Load(dir)
	if err != nil {
		fmt.Fprintf(stderr, "fusaops pr close: load: %v\n", err)
		return 1
	}
	if err := pr.Close(log, *id, *resolution); err != nil {
		fmt.Fprintf(stderr, "fusaops pr close: %v\n", err)
		return 1
	}
	if err := pr.Save(dir, log); err != nil {
		fmt.Fprintf(stderr, "fusaops pr close: save: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "Problem report %s closed\n", *id)
	return 0
}
