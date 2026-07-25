package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/SoundMatt/FuSaOps/adapter"
	"github.com/SoundMatt/FuSaOps/config"
	"github.com/SoundMatt/FuSaOps/qualify"
)

// runQualify runs tool qualification for every available x-FuSa adapter.
//
//fusa:req REQ-FO-CLI064
//fusa:req REQ-FO-CLI078
func runQualify(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("fusaops qualify", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() {
		fmt.Fprintf(stderr, "Usage: fusaops qualify [flags]\n\n")
		fmt.Fprintf(stderr, "Run the tool qualification suite for each installed x-FuSa adapter.\n")
		fmt.Fprintf(stderr, "The report can be used as tool confidence evidence for regulated environments.\n\n")
		fmt.Fprintf(stderr, "Flags:\n")
		fs.PrintDefaults()
	}

	var (
		dir       = fs.String("dir", "", "project root directory (default: current directory)")
		output    = fs.String("output", "", "path for the JSON report (default: <dir>/.fusaops-qualify-report.json)")
		format    = fs.String("format", "text", "output format: text, json")
		qualType  = fs.String("type", "self", "qualification type: self or independent")
		recordURI = fs.String("record-uri", "", "URI of external TQL-5/DO-330 qualification certificate")
	)
	if err := fs.Parse(args); err != nil {
		return 2
	}

	projectRoot := *dir
	if projectRoot == "" {
		var err error
		projectRoot, err = os.Getwd()
		if err != nil {
			fmt.Fprintf(stderr, "fusaops qualify: get working directory: %v\n", err)
			return 1
		}
	}

	// Apply config-file defaults for --type and --record-uri when flags are at defaults.
	cfgPath := filepath.Join(projectRoot, config.ConfigFile)
	if cfg, err := config.Load(cfgPath); err == nil {
		if *qualType == "self" && cfg.Qualify.Type != "" {
			*qualType = cfg.Qualify.Type
		}
		if *recordURI == "" && cfg.Qualify.RecordUri != "" {
			*recordURI = cfg.Qualify.RecordUri
		}
	}

	adapters, err := adapter.Default.Applicable(projectRoot)
	if err != nil {
		fmt.Fprintf(stderr, "fusaops qualify: detect adapters: %v\n", err)
		return 1
	}
	if len(adapters) == 0 {
		fmt.Fprintf(stderr, "fusaops qualify: no applicable adapters found\n")
		return 1
	}

	fmt.Fprintf(stdout, "Running qualification for %d adapter(s)...\n", len(adapters))
	report, err := qualify.Run(context.Background(), adapters, projectRoot,
		qualify.RunOptions{
			Type:      qualify.QualificationType(*qualType),
			RecordUri: *recordURI,
		})
	if err != nil {
		fmt.Fprintf(stderr, "fusaops qualify: %v\n", err)
		return 1
	}

	if err := qualify.Render(stdout, report, *format); err != nil {
		fmt.Fprintf(stderr, "fusaops qualify: render: %v\n", err)
		return 2
	}

	outPath := *output
	if outPath == "" {
		outPath = filepath.Join(projectRoot, qualify.ReportFile)
	}
	if err := qualify.Save(outPath, report); err != nil {
		fmt.Fprintf(stderr, "fusaops qualify: save report: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "Qualification report written to %s\n", outPath)
	fmt.Fprintf(stdout, "Integrity hash: %s\n", report.Hash)
	if report.QualificationType != "" {
		fmt.Fprintf(stdout, "Qualification type: %s\n", report.QualificationType)
	}
	if report.QualificationRecordUri != "" {
		fmt.Fprintf(stdout, "Certificate URI: %s\n", report.QualificationRecordUri)
	}

	if report.HasFailures() {
		return 1
	}
	return 0
}
