package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"time"

	fusaops "github.com/SoundMatt/FuSaOps"
)

// capabilities is the §9.1 x-FuSa discovery document (kind: "capabilities").
type capabilities struct {
	SchemaVersion string              `json:"schemaVersion"`
	Kind          string              `json:"kind"`
	Tool          string              `json:"tool"`
	ToolVersion   string              `json:"toolVersion"`
	GeneratedAt   time.Time           `json:"generatedAt"`
	SpecVersion   string              `json:"specVersion"`
	Commands      []string            `json:"commands"`
	Formats       map[string][]string `json:"formats"`
	Standards     []string            `json:"standards"`
}

// runCapabilities reports FuSaOps's supported commands, formats, and standards.
//
//fusa:req REQ-FO-CLI054
//fusa:req REQ-FO-SPEC002
//fusa:req REQ-FO-CLI083
func runCapabilities(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("fusaops capabilities", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() {
		fmt.Fprintf(stderr, "Usage: fusaops capabilities [--format json]\n\n")
		fmt.Fprintf(stderr, "Report FuSaOps's supported commands, output formats, and safety standards.\n\n")
		fmt.Fprintf(stderr, "Flags:\n")
		fs.PrintDefaults()
	}
	format := fs.String("format", "json", "output format: json")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *format != "json" && *format != "" {
		fmt.Fprintf(stderr, "fusaops capabilities: unsupported format %q (only json)\n", *format)
		return 2
	}

	cap := &capabilities{
		SchemaVersion: fusaops.SpecVersion,
		Kind:          "capabilities",
		Tool:          "fusaops",
		ToolVersion:   fusaops.Version,
		GeneratedAt:   time.Now().UTC(),
		SpecVersion:   fusaops.SpecVersion,
		Commands: []string{
			"version", "capabilities", "init", "config", "scan", "adapters",
			"check", "report", "trace", "sbom", "audit-pack", "comp", "diff",
			"suppress", "conform", "coverage", "req", "metrics", "badge", "slsa", "hooks", "impact", "disposition", "pr", "verify", "sign", "qualify", "release", "safety-case", "sci", "sas", "tara", "fmea", "vuln", "template", "hara", "vv",
			"iso26262", "iec61508", "do178", "iso21434", "unece", "iec62443",
			"policy", "fleet", "history", "serve",
		},
		Formats: map[string][]string{
			"comp":        {"text", "json"},
			"check":       {"text", "json", "html", "sarif", "junit", "csv", "markdown"},
			"report":      {"text", "json", "html", "sarif", "junit", "csv", "markdown"},
			"trace":       {"text", "json", "html", "markdown"},
			"sbom":        {"text", "json", "spdx", "html", "markdown"},
			"diff":        {"text", "json", "html", "markdown"},
			"fleet":       {"text", "json", "html", "markdown"},
			"policy":      {"text", "json", "html", "markdown"},
			"conform":     {"text", "json", "html", "markdown"},
			"coverage":    {"text", "json", "markdown"},
			"iso26262":    {"text", "json", "html", "markdown"},
			"iec61508":    {"text", "json", "html", "markdown"},
			"do178":       {"text", "json", "html", "markdown"},
			"iso21434":    {"text", "json", "html", "markdown"},
			"unece":       {"text", "json", "html", "markdown"},
			"iec62443":    {"text", "json", "html", "markdown"},
			"adapters":    {"text", "json"},
			"history":     {"text", "json"},
			"suppress":    {"text", "json"},
			"req":         {"csv", "doors", "polarion", "codebeamer", "jama"},
			"version":     {"text", "json"},
			"metrics":     {"text", "json"},
			"badge":       {"svg"},
			"slsa":        {"text", "json"},
			"impact":      {"text", "json"},
			"pr":          {"text", "json"},
			"verify":      {"text", "json"},
			"sign":        {"text"},
			"qualify":     {"text", "json"},
			"release":     {"text"},
			"safety-case": {"text", "json"},
			"sci":         {"text", "json"},
			"sas":         {"text", "json"},
			"tara":        {"text", "json"},
			"fmea":        {"text", "json"},
			"vuln":        {"text", "json"},
			"template":    {"text", "json"},
			"hara":        {"text", "json", "markdown"},
			"vv":          {"text", "json"},
		},
		Standards: []string{
			"iso26262", "iec61508", "do178c", "iso21434", "unece-r155", "iec62443", "slsa",
		},
	}

	enc := json.NewEncoder(stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(cap); err != nil {
		fmt.Fprintf(stderr, "fusaops capabilities: %v\n", err)
		return 1
	}
	return 0
}
