package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/SoundMatt/FuSaOps/req"
)

// runReq dispatches fusaops req subcommands (show, import, export).
//
//fusa:req REQ-FO-CLI052
func runReq(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("fusaops req", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() {
		fmt.Fprintf(stderr, "Usage: fusaops req [subcommand] [flags] [REQ-ID ...]\n\n")
		fmt.Fprintf(stderr, "Show, import, or export requirements from .fusa-reqs.json.\n\n")
		fmt.Fprintf(stderr, "Subcommands:\n")
		fmt.Fprintf(stderr, "  (none)  Show requirements and their metadata\n")
		fmt.Fprintf(stderr, "  import  Import requirements from CSV, DOORS, Polarion, Codebeamer, or Jama\n")
		fmt.Fprintf(stderr, "  export  Export requirements to CSV, DOORS, Polarion, Codebeamer, or Jama\n\n")
		fmt.Fprintf(stderr, "Flags:\n")
		fs.PrintDefaults()
	}
	dir := fs.String("dir", ".", "project root directory")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	remaining := fs.Args()
	if len(remaining) > 0 {
		switch remaining[0] {
		case "import":
			return runReqImport(remaining[1:], *dir, stdout, stderr)
		case "export":
			return runReqExport(remaining[1:], *dir, stdout, stderr)
		}
	}
	return runReqShow(remaining, *dir, stdout, stderr)
}

//fusa:req REQ-FO-CLI052
func runReqShow(ids []string, dir string, stdout, stderr io.Writer) int {
	entries, err := req.LoadRegistry(dir)
	if err != nil {
		fmt.Fprintf(stderr, "fusaops req: %v\n", err)
		return 1
	}
	filter := make(map[string]bool)
	for _, id := range ids {
		filter[id] = true
	}
	printed := 0
	for _, e := range entries {
		if len(filter) > 0 && !filter[e.ID] {
			continue
		}
		printed++
		fmt.Fprintf(stdout, "%s  %s\n", e.ID, e.Title)
		if e.Text != "" {
			fmt.Fprintf(stdout, "  %s\n", e.Text)
		} else if e.Description != "" {
			fmt.Fprintf(stdout, "  %s\n", e.Description)
		}
		if e.Standard != "" {
			fmt.Fprintf(stdout, "  Standard: %s", e.Standard)
			if e.Level != "" {
				fmt.Fprintf(stdout, "  Level: %s", e.Level)
			}
			fmt.Fprintln(stdout)
		}
		if e.Priority != "" {
			fmt.Fprintf(stdout, "  Priority: %s\n", e.Priority)
		}
		fmt.Fprintln(stdout)
	}
	if len(filter) > 0 && printed == 0 {
		for id := range filter {
			fmt.Fprintf(stderr, "fusaops req: requirement %q not found\n", id)
		}
		return 1
	}
	return 0
}

//fusa:req REQ-FO-CLI052
func runReqImport(args []string, dir string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("fusaops req import", flag.ContinueOnError)
	fs.SetOutput(stderr)
	format := fs.String("format", "csv", "import format: csv, doors, polarion, codebeamer, jama")
	file := fs.String("file", "", "input file path (required)")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *file == "" {
		fmt.Fprintf(stderr, "fusaops req import: --file is required\n")
		return 2
	}

	existing, err := req.LoadRegistry(dir)
	if err != nil {
		existing = nil // no registry yet is fine
	}
	existingIDs := make(map[string]bool)
	for _, e := range existing {
		existingIDs[e.ID] = true
	}

	data, readErr := os.ReadFile(*file)
	if readErr != nil && *format != "csv" {
		fmt.Fprintf(stderr, "fusaops req import: read %s: %v\n", *file, readErr)
		return 1
	}

	var imported []req.Entry
	switch *format {
	case "csv":
		f, openErr := os.Open(*file)
		if openErr != nil {
			fmt.Fprintf(stderr, "fusaops req import: open %s: %v\n", *file, openErr)
			return 1
		}
		defer func() { _ = f.Close() }()
		imported, err = req.ParseCSV(f)
	case "doors":
		imported, err = req.ParseDOORS(data)
	case "polarion":
		imported, err = req.ParsePolarion(data)
	case "codebeamer":
		imported, err = req.ParseCodebeamer(data)
	case "jama":
		imported, err = req.ParseJama(data)
	default:
		fmt.Fprintf(stderr, "fusaops req import: unknown format %q (want csv, doors, polarion, codebeamer, jama)\n", *format)
		return 2
	}
	if err != nil {
		fmt.Fprintf(stderr, "fusaops req import: parse: %v\n", err)
		return 1
	}

	added, skipped := 0, 0
	for _, e := range imported {
		if existingIDs[e.ID] {
			skipped++
			continue
		}
		existing = append(existing, e)
		existingIDs[e.ID] = true
		added++
	}

	if err := req.SaveRegistry(dir, existing); err != nil {
		fmt.Fprintf(stderr, "fusaops req import: save: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "Imported %d requirements (%d skipped as duplicates)\n", added, skipped)
	return 0
}

//fusa:req REQ-FO-CLI052
func runReqExport(args []string, dir string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("fusaops req export", flag.ContinueOnError)
	fs.SetOutput(stderr)
	format := fs.String("format", "csv", "export format: csv, doors, polarion, codebeamer, jama")
	output := fs.String("output", "", "output file (default: stdout)")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	entries, err := req.LoadRegistry(dir)
	if err != nil {
		fmt.Fprintf(stderr, "fusaops req export: load registry: %v\n", err)
		return 1
	}

	w := stdout
	if *output != "" {
		f, createErr := os.Create(*output)
		if createErr != nil {
			fmt.Fprintf(stderr, "fusaops req export: create %s: %v\n", *output, createErr)
			return 1
		}
		defer func() { _ = f.Close() }()
		w = f
	}

	switch *format {
	case "csv":
		if err := req.RenderCSV(w, entries); err != nil {
			fmt.Fprintf(stderr, "fusaops req export: %v\n", err)
			return 1
		}
	default:
		var (
			data      []byte
			exportErr error
		)
		switch *format {
		case "doors":
			data, exportErr = req.ExportDOORS(entries)
		case "polarion":
			data, exportErr = req.ExportPolarion(entries)
		case "codebeamer":
			data, exportErr = req.ExportCodebeamer(entries)
		case "jama":
			data, exportErr = req.ExportJama(entries)
		default:
			fmt.Fprintf(stderr, "fusaops req export: unknown format %q (want csv, doors, polarion, codebeamer, jama)\n", *format)
			return 2
		}
		if exportErr != nil {
			fmt.Fprintf(stderr, "fusaops req export: %v\n", exportErr)
			return 1
		}
		if _, werr := w.Write(data); werr != nil {
			fmt.Fprintf(stderr, "fusaops req export: write: %v\n", werr)
			return 1
		}
	}
	return 0
}
