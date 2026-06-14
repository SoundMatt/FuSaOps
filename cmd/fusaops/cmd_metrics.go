package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/SoundMatt/FuSaOps/metrics"
)

// runMetrics dispatches fusaops metrics subcommands (record, show).
//
//fusa:req REQ-FO-CLI055
func runMetrics(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("fusaops metrics", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() {
		fmt.Fprintf(stderr, "Usage: fusaops metrics <subcommand> [flags]\n\n")
		fmt.Fprintf(stderr, "Track FuSaOps project safety metrics over time.\n\n")
		fmt.Fprintf(stderr, "Subcommands:\n")
		fmt.Fprintf(stderr, "  record  Collect and append a metrics snapshot to .fusaops-metrics.json\n")
		fmt.Fprintf(stderr, "  show    Display the full metrics time series\n\n")
		fmt.Fprintf(stderr, "Global flags:\n")
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
	case "record":
		return runMetricsRecord(*dir, stdout, stderr)
	case "show":
		return runMetricsShow(sub[1:], *dir, stdout, stderr)
	default:
		fmt.Fprintf(stderr, "fusaops metrics: unknown subcommand %q (want record or show)\n", sub[0])
		return 2
	}
}

//fusa:req REQ-FO-CLI055
func runMetricsRecord(dir string, stdout, stderr io.Writer) int {
	ts, err := metrics.Load(dir)
	if err != nil {
		fmt.Fprintf(stderr, "fusaops metrics record: load: %v\n", err)
		return 1
	}

	snap, err := metrics.Collect(dir)
	if err != nil {
		fmt.Fprintf(stderr, "fusaops metrics record: collect: %v\n", err)
		return 1
	}

	ts = metrics.Append(ts, snap)

	if err := metrics.Save(dir, ts); err != nil {
		fmt.Fprintf(stderr, "fusaops metrics record: save: %v\n", err)
		return 1
	}

	covStr := "n/a"
	if snap.CoveragePct > 0 {
		covStr = fmt.Sprintf("%.1f%%", snap.CoveragePct)
	}
	fmt.Fprintf(stdout, "Metrics recorded: errors=%d warnings=%d infos=%d reqs=%d coverage=%s\n",
		snap.ErrorCount, snap.WarningCount, snap.InfoCount,
		snap.TotalRequirements, covStr,
	)
	path := filepath.Join(dir, metrics.MetricsFile)
	fmt.Fprintf(stdout, "Time series saved to %s (%d snapshots)\n", path, len(ts.Snapshots))
	return 0
}

//fusa:req REQ-FO-CLI055
func runMetricsShow(args []string, dir string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("fusaops metrics show", flag.ContinueOnError)
	fs.SetOutput(stderr)
	format := fs.String("format", "text", "output format: text or json")
	output := fs.String("output", "", "write to file (default: stdout)")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	ts, err := metrics.Load(dir)
	if err != nil {
		fmt.Fprintf(stderr, "fusaops metrics show: load: %v\n", err)
		return 1
	}

	w := stdout
	if *output != "" {
		f, createErr := os.Create(*output)
		if createErr != nil {
			fmt.Fprintf(stderr, "fusaops metrics show: create %s: %v\n", *output, createErr)
			return 1
		}
		defer func() { _ = f.Close() }()
		w = f
	}

	if err := metrics.Render(w, ts, *format); err != nil {
		fmt.Fprintf(stderr, "fusaops metrics show: render: %v\n", err)
		return 1
	}
	return 0
}
