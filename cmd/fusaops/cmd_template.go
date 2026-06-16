package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/SoundMatt/FuSaOps/doctemplate"
)

// runTemplate generates safety documentation templates for the project.
//
//fusa:req REQ-FO-CLI072
func runTemplate(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("fusaops template", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() {
		fmt.Fprintf(stderr, "Usage: fusaops template [flags]\n\n")
		fmt.Fprintf(stderr, "Generate safety documentation templates for multi-language projects.\n\n")
		fmt.Fprintf(stderr, "Templates are written as Markdown files to the output directory.\n")
		fmt.Fprintf(stderr, "Available templates cover: Software Safety Plan, HARA, SRS, Test Plan,\n")
		fmt.Fprintf(stderr, "TARA, SCI, SAS, and Problem Report.\n\n")
		fmt.Fprintf(stderr, "Use --standards to filter by target standard(s).\n")
		fmt.Fprintf(stderr, "Supported standards: ISO 26262, IEC 61508, DO-178C, ISO 21434\n\n")
		fmt.Fprintf(stderr, "Flags:\n")
		fs.PrintDefaults()
	}

	var (
		dir       = fs.String("dir", "", "project root directory (default: current directory)")
		outputDir = fs.String("output-dir", "", "directory to write templates (default: <dir>/safety-docs)")
		standards = fs.String("standards", "", "comma-separated list of standards to filter by (default: all)")
		output    = fs.String("output", "", "path for the template generation report (default: <dir>/.fusaops-templates.json)")
		format    = fs.String("format", "text", "output format: text, json")
	)
	if err := fs.Parse(args); err != nil {
		return 2
	}

	projectRoot := *dir
	if projectRoot == "" {
		var err error
		projectRoot, err = os.Getwd()
		if err != nil {
			fmt.Fprintf(stderr, "fusaops template: get working directory: %v\n", err)
			return 1
		}
	}

	outDir := *outputDir
	if outDir == "" {
		outDir = filepath.Join(projectRoot, doctemplate.DefaultOutputDir)
	}

	var stdList []string
	if *standards != "" {
		for _, s := range strings.Split(*standards, ",") {
			s = strings.TrimSpace(s)
			if s != "" {
				stdList = append(stdList, s)
			}
		}
	}

	r, err := doctemplate.Generate(projectRoot, outDir, stdList)
	if err != nil {
		fmt.Fprintf(stderr, "fusaops template: generate: %v\n", err)
		return 1
	}

	if renderErr := doctemplate.Render(stdout, r, *format); renderErr != nil {
		fmt.Fprintf(stderr, "fusaops template: render: %v\n", renderErr)
		return 2
	}

	reportPath := *output
	if reportPath == "" {
		reportPath = filepath.Join(projectRoot, doctemplate.ReportFile)
	}
	if saveErr := doctemplate.Save(reportPath, r); saveErr != nil {
		fmt.Fprintf(stderr, "fusaops template: save: %v\n", saveErr)
		return 1
	}
	fmt.Fprintf(stdout, "\nTemplates written to %s\n", outDir)
	fmt.Fprintf(stdout, "Report: %s\n", reportPath)
	fmt.Fprintf(stdout, "Integrity hash: %s\n", r.Hash)
	return 0
}
