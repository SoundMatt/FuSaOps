package main

import (
	"flag"
	"fmt"
	"io"
	"time"

	"github.com/SoundMatt/FuSaOps/disposition"
)

// runDisposition dispatches fusaops disposition subcommands.
//
//fusa:req REQ-FO-CLI060
func runDisposition(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("fusaops disposition", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() {
		fmt.Fprintf(stderr, "Usage: fusaops disposition <subcommand> [flags]\n\n")
		fmt.Fprintf(stderr, "Manage finding disposition entries in .fusaops-dispositions.json.\n\n")
		fmt.Fprintf(stderr, "Subcommands:\n")
		fmt.Fprintf(stderr, "  add   Record a disposition decision for a finding\n")
		fmt.Fprintf(stderr, "  list  List all disposition entries\n")
		fmt.Fprintf(stderr, "  show  Show the disposition entry for a specific rule\n\n")
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
	case "add":
		return runDispositionAdd(sub[1:], *dir, stdout, stderr)
	case "list":
		return runDispositionList(*dir, stdout, stderr)
	case "show":
		return runDispositionShow(sub[1:], *dir, stdout, stderr)
	default:
		fmt.Fprintf(stderr, "fusaops disposition: unknown subcommand %q (want add, list, or show)\n", sub[0])
		return 2
	}
}

//fusa:req REQ-FO-CLI060
func runDispositionAdd(args []string, dir string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("fusaops disposition add", flag.ContinueOnError)
	fs.SetOutput(stderr)
	ruleID := fs.String("rule", "", "rule ID to disposition (required)")
	language := fs.String("lang", "", "language (e.g. go, rust, python)")
	action := fs.String("action", "accept", "action: accept or fix")
	reviewer := fs.String("reviewer", "", "reviewer name (required)")
	rationale := fs.String("rationale", "", "rationale for disposition (required)")
	ref := fs.String("ref", "", "optional reference (issue, ticket, etc)")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *ruleID == "" {
		fmt.Fprintln(stderr, "fusaops disposition add: --rule is required")
		return 2
	}
	if *reviewer == "" {
		fmt.Fprintln(stderr, "fusaops disposition add: --reviewer is required")
		return 2
	}
	if *rationale == "" {
		fmt.Fprintln(stderr, "fusaops disposition add: --rationale is required")
		return 2
	}
	if *action != "accept" && *action != "fix" {
		fmt.Fprintf(stderr, "fusaops disposition add: --action must be 'accept' or 'fix'\n")
		return 2
	}

	log, err := disposition.Load(dir)
	if err != nil {
		fmt.Fprintf(stderr, "fusaops disposition add: load: %v\n", err)
		return 1
	}

	entry := disposition.Entry{
		RuleID:    *ruleID,
		Language:  *language,
		Rationale: *rationale,
		Reviewer:  *reviewer,
		Date:      time.Now().UTC(),
		Action:    disposition.Action(*action),
		Reference: *ref,
	}
	log = disposition.Add(log, entry)

	if err := disposition.Save(dir, log); err != nil {
		fmt.Fprintf(stderr, "fusaops disposition add: save: %v\n", err)
		return 1
	}

	fmt.Fprintf(stdout, "Disposition added: rule=%s action=%s reviewer=%s\n",
		entry.RuleID, entry.Action, entry.Reviewer)
	return 0
}

//fusa:req REQ-FO-CLI060
func runDispositionList(dir string, stdout, stderr io.Writer) int {
	log, err := disposition.Load(dir)
	if err != nil {
		fmt.Fprintf(stderr, "fusaops disposition list: %v\n", err)
		return 1
	}
	if err := disposition.RenderEntries(stdout, log); err != nil {
		fmt.Fprintf(stderr, "fusaops disposition list: render: %v\n", err)
		return 1
	}
	return 0
}

//fusa:req REQ-FO-CLI060
func runDispositionShow(args []string, dir string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("fusaops disposition show", flag.ContinueOnError)
	fs.SetOutput(stderr)
	ruleID := fs.String("rule", "", "rule ID to show (required)")
	language := fs.String("lang", "", "language to filter by")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *ruleID == "" {
		fmt.Fprintln(stderr, "fusaops disposition show: --rule is required")
		return 2
	}

	log, err := disposition.Load(dir)
	if err != nil {
		fmt.Fprintf(stderr, "fusaops disposition show: %v\n", err)
		return 1
	}

	e := disposition.Find(log, *ruleID, *language)
	if e == nil {
		fmt.Fprintf(stderr, "fusaops disposition show: no disposition found for rule %q\n", *ruleID)
		return 1
	}

	fmt.Fprintf(stdout, "Rule:      %s\n", e.RuleID)
	if e.Language != "" {
		fmt.Fprintf(stdout, "Language:  %s\n", e.Language)
	}
	fmt.Fprintf(stdout, "Action:    %s\n", e.Action)
	fmt.Fprintf(stdout, "Reviewer:  %s\n", e.Reviewer)
	fmt.Fprintf(stdout, "Date:      %s\n", e.Date.Format("2006-01-02"))
	fmt.Fprintf(stdout, "Rationale: %s\n", e.Rationale)
	if e.Reference != "" {
		fmt.Fprintf(stdout, "Reference: %s\n", e.Reference)
	}
	return 0
}
