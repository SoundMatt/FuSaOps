package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/SoundMatt/FuSaOps/orchestrator"
	"github.com/SoundMatt/FuSaOps/report"
	"github.com/SoundMatt/FuSaOps/suppression"
)

const defaultSuppressFile = ".fusaops-suppress.json"

// runSuppress dispatches suppress subcommands: add, list, prune, verify, import.
//
//fusa:req REQ-FO-CLI039
//fusa:req REQ-FO-CLI048
//fusa:req REQ-FO-SUP005
//fusa:req REQ-FO-SUP006
//fusa:req REQ-FO-SUP007
//fusa:req REQ-FO-SUP008
//fusa:req REQ-FO-SUP009
func runSuppress(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "fusaops suppress: subcommand required: add|list|prune|verify|import")
		return 2
	}
	switch args[0] {
	case "add":
		return runSuppressAdd(args[1:], stdout, stderr)
	case "list":
		return runSuppressList(args[1:], stdout, stderr)
	case "prune":
		return runSuppressPrune(args[1:], stdout, stderr)
	case "verify":
		return runSuppressVerify(args[1:], stdout, stderr)
	case "import":
		return runSuppressImport(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "fusaops suppress: unknown subcommand %q\n", args[0])
		return 2
	}
}

// runSuppressAdd appends a new suppression entry.
//
//fusa:req REQ-FO-SUP005
func runSuppressAdd(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("fusaops suppress add", flag.ContinueOnError)
	fs.SetOutput(stderr)
	file := fs.String("file", defaultSuppressFile, "path to .fusaops-suppress.json")
	fingerprint := fs.String("fingerprint", "", "finding fingerprint to suppress (sha256:...)")
	reason := fs.String("reason", "", "human-readable rationale (required)")
	expires := fs.String("expires", "", "optional expiry date (YYYY-MM-DD)")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *fingerprint == "" {
		fmt.Fprintln(stderr, "fusaops suppress add: --fingerprint is required")
		return 1
	}
	if *reason == "" {
		fmt.Fprintln(stderr, "fusaops suppress add: --reason is required")
		return 1
	}
	if *expires != "" {
		if _, err := time.Parse("2006-01-02", *expires); err != nil {
			fmt.Fprintf(stderr, "fusaops suppress add: --expires must be YYYY-MM-DD: %v\n", err)
			return 1
		}
	}
	cfg, err := suppression.LoadConfig(*file)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		fmt.Fprintf(stderr, "fusaops suppress add: load %s: %v\n", *file, err)
		return 1
	}
	cfg.Suppressions = append(cfg.Suppressions, suppression.Suppression{
		Fingerprint: *fingerprint,
		Reason:      *reason,
		Expires:     *expires,
	})
	if err := suppression.SaveConfig(*file, cfg); err != nil {
		fmt.Fprintf(stderr, "fusaops suppress add: write %s: %v\n", *file, err)
		return 1
	}
	fmt.Fprintf(stdout, "Added suppression for %s → %s\n", *fingerprint, *file)
	return 0
}

// runSuppressList prints suppression entries.
//
//fusa:req REQ-FO-SUP006
func runSuppressList(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("fusaops suppress list", flag.ContinueOnError)
	fs.SetOutput(stderr)
	file := fs.String("file", defaultSuppressFile, "path to .fusaops-suppress.json")
	format := fs.String("format", "text", "output format: text|json")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	cfg, err := suppression.LoadConfig(*file)
	if err != nil {
		fmt.Fprintf(stderr, "fusaops suppress list: %v\n", err)
		return 1
	}
	if *format == "json" {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(cfg)
		return 0
	}
	now := time.Now()
	if len(cfg.Suppressions) == 0 {
		fmt.Fprintln(stdout, "No suppressions.")
		return 0
	}
	for i, s := range cfg.Suppressions {
		status := "active"
		if s.Expires != "" {
			exp, perr := time.Parse("2006-01-02", s.Expires)
			if perr != nil {
				status = "invalid-date"
			} else if !now.Before(exp.AddDate(0, 0, 1)) {
				status = "expired"
			}
		}
		exp := "never"
		if s.Expires != "" {
			exp = s.Expires
		}
		fmt.Fprintf(stdout, "%d. [%s] %s\n   reason: %s\n   expires: %s\n",
			i+1, status, s.Fingerprint, s.Reason, exp)
	}
	return 0
}

// runSuppressPrune removes expired suppression entries and rewrites the file.
//
//fusa:req REQ-FO-SUP007
func runSuppressPrune(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("fusaops suppress prune", flag.ContinueOnError)
	fs.SetOutput(stderr)
	file := fs.String("file", defaultSuppressFile, "path to .fusaops-suppress.json")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	cfg, err := suppression.LoadConfig(*file)
	if err != nil {
		fmt.Fprintf(stderr, "fusaops suppress prune: %v\n", err)
		return 1
	}
	pruned, removed := suppression.Prune(cfg, time.Now())
	if removed == 0 {
		fmt.Fprintln(stdout, "No expired suppressions to remove.")
		return 0
	}
	if err := suppression.SaveConfig(*file, pruned); err != nil {
		fmt.Fprintf(stderr, "fusaops suppress prune: write %s: %v\n", *file, err)
		return 1
	}
	fmt.Fprintf(stdout, "Removed %d expired suppression(s) → %s\n", removed, *file)
	return 0
}

// runSuppressVerify checks that every suppression fingerprint matches a
// current finding. Exit 1 if any suppression is stale (no matching finding).
//
//fusa:req REQ-FO-SUP008
func runSuppressVerify(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("fusaops suppress verify", flag.ContinueOnError)
	fs.SetOutput(stderr)
	file := fs.String("file", defaultSuppressFile, "path to .fusaops-suppress.json")
	dir := fs.String("dir", ".", "project root to scan")
	only := fs.String("only", "", "comma-separated tool names to run")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	cfg, err := suppression.LoadConfig(*file)
	if err != nil {
		fmt.Fprintf(stderr, "fusaops suppress verify: %v\n", err)
		return 1
	}
	root, opts, _, err := loadOptions(*dir, *only, stderr)
	if err != nil {
		fmt.Fprintf(stderr, "fusaops suppress verify: %v\n", err)
		return 1
	}
	rep, err := orchestrator.New(nil).Run(context.Background(), root, opts)
	if err != nil {
		fmt.Fprintf(stderr, "fusaops suppress verify: scan: %v\n", err)
		return 1
	}
	currentFPs := make(map[string]struct{})
	for _, c := range rep.Components {
		for _, f := range c.Findings {
			if f.Fingerprint != "" {
				currentFPs[f.Fingerprint] = struct{}{}
			}
		}
	}
	var stale []suppression.Suppression
	for _, s := range cfg.Suppressions {
		if s.Fingerprint == "" {
			continue
		}
		if _, ok := currentFPs[s.Fingerprint]; !ok {
			stale = append(stale, s)
		}
	}
	if len(stale) == 0 {
		fmt.Fprintf(stdout, "All %d suppression(s) match current findings — OK\n", len(cfg.Suppressions))
		return 0
	}
	fmt.Fprintf(stderr, "fusaops suppress verify: %d stale suppression(s) (fingerprint not in current findings):\n", len(stale))
	for _, s := range stale {
		fmt.Fprintf(stderr, "  %s  %s\n", s.Fingerprint, s.Reason)
	}
	return 1
}

// runSuppressImport bulk-adds fingerprints from a fusaops check JSON report.
//
//fusa:req REQ-FO-SUP009
//fusa:req REQ-FO-CLI048
func runSuppressImport(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("fusaops suppress import", flag.ContinueOnError)
	fs.SetOutput(stderr)
	file := fs.String("file", defaultSuppressFile, "path to .fusaops-suppress.json")
	from := fs.String("from", "", "path to a fusaops check --format json report")
	reason := fs.String("reason", "imported", "reason for all imported suppressions")
	expires := fs.String("expires", "", "expiry date for imported suppressions (YYYY-MM-DD; optional)")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *from == "" {
		fmt.Fprintln(stderr, "fusaops suppress import: --from is required")
		return 2
	}

	data, err := os.ReadFile(*from)
	if err != nil {
		fmt.Fprintf(stderr, "fusaops suppress import: read %s: %v\n", *from, err)
		return 1
	}
	var rep report.AggregateReport
	if err := json.Unmarshal(data, &rep); err != nil {
		fmt.Fprintf(stderr, "fusaops suppress import: parse %s: %v\n", *from, err)
		return 1
	}

	cfg, loadErr := suppression.LoadConfig(*file)
	if loadErr != nil && !os.IsNotExist(loadErr) {
		fmt.Fprintf(stderr, "fusaops suppress import: load suppress file: %v\n", loadErr)
		return 1
	}

	existing := make(map[string]struct{}, len(cfg.Suppressions))
	for _, s := range cfg.Suppressions {
		existing[s.Fingerprint] = struct{}{}
	}

	total, newCount := 0, 0
	for _, c := range rep.Components {
		for _, f := range c.Findings {
			if f.Fingerprint == "" {
				continue
			}
			total++
			if _, ok := existing[f.Fingerprint]; ok {
				continue
			}
			cfg.Suppressions = append(cfg.Suppressions, suppression.Suppression{
				Fingerprint: f.Fingerprint,
				Reason:      *reason,
				Expires:     *expires,
			})
			existing[f.Fingerprint] = struct{}{}
			newCount++
		}
	}

	if err := suppression.SaveConfig(*file, cfg); err != nil {
		fmt.Fprintf(stderr, "fusaops suppress import: save suppress file: %v\n", err)
		return 1
	}

	already := total - newCount
	fmt.Fprintf(stdout, "Imported %d findings (%d new, %d already present).\n", total, newCount, already)
	return 0
}
