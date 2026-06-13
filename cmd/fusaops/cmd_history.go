package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"path/filepath"

	"github.com/SoundMatt/FuSaOps/history"
)

// runHistory dispatches fusaops history <subcommand>.
//
//fusa:req REQ-FO-CLI045
//fusa:req REQ-FO-CLI046
func runHistory(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "fusaops history: subcommand required (list|prune)")
		return 2
	}
	switch args[0] {
	case "list":
		return runHistoryList(args[1:], stdout, stderr)
	case "prune":
		return runHistoryPrune(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "fusaops history: unknown subcommand %q\n", args[0])
		return 2
	}
}

// runHistoryList prints snapshots from .fusaops-history.jsonl.
//
//fusa:req REQ-FO-CLI045
func runHistoryList(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("fusaops history list", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dir := fs.String("dir", ".", "directory containing .fusaops-history.jsonl")
	file := fs.String("file", "", "path to history file (overrides --dir)")
	format := fs.String("format", "text", "output format: text or json")
	limit := fs.Int("limit", 0, "max snapshots to show (0 = all)")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	hdir := *dir
	if *file != "" {
		hdir = filepath.Dir(*file)
	}

	snaps, err := history.Load(hdir, *limit)
	if err != nil {
		fmt.Fprintf(stderr, "fusaops history list: %v\n", err)
		return 1
	}

	switch *format {
	case "json":
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(snaps); err != nil {
			fmt.Fprintf(stderr, "fusaops history list: %v\n", err)
			return 1
		}
	default:
		if len(snaps) == 0 {
			fmt.Fprintln(stdout, "No history entries found.")
			return 0
		}
		fmt.Fprintf(stdout, "%-25s  %-4s  %6s  %7s  %7s  %5s\n",
			"RUN AT", "STAT", "ERRORS", "WARNINGS", "INFOS", "TOTAL")
		fmt.Fprintf(stdout, "%-25s  %-4s  %6s  %7s  %7s  %5s\n",
			"-------------------------", "----", "------", "-------", "-------", "-----")
		for i := len(snaps) - 1; i >= 0; i-- {
			s := snaps[i]
			stat := "PASS"
			if s.Status == "FAIL" {
				stat = "FAIL"
			}
			fmt.Fprintf(stdout, "%-25s  %-4s  %6d  %7d  %7d  %5d\n",
				s.RunAt.Local().Format("2006-01-02 15:04:05 MST"),
				stat, s.Errors, s.Warnings, s.Infos, s.Total)
		}
	}
	return 0
}

// runHistoryPrune removes old entries from .fusaops-history.jsonl.
//
//fusa:req REQ-FO-CLI046
func runHistoryPrune(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("fusaops history prune", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dir := fs.String("dir", ".", "directory containing .fusaops-history.jsonl")
	file := fs.String("file", "", "path to history file (overrides --dir)")
	keep := fs.Int("keep", history.MaxSnapshots, "number of most-recent entries to keep")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	hdir := *dir
	if *file != "" {
		hdir = filepath.Dir(*file)
	}

	removed, err := history.Prune(hdir, *keep)
	if err != nil {
		fmt.Fprintf(stderr, "fusaops history prune: %v\n", err)
		return 1
	}

	all, _ := history.Load(hdir, 0)
	remaining := len(all)

	fmt.Fprintf(stdout, "Pruned %d entries, %d remaining.\n", removed, remaining)
	return 0
}
