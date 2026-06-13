package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"path/filepath"

	fusaops "github.com/SoundMatt/FuSaOps"
	"github.com/SoundMatt/FuSaOps/config"
)

// runConfig dispatches fusaops config <subcommand>.
//
//fusa:req REQ-FO-CLI043
//fusa:req REQ-FO-CLI044
func runConfig(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "fusaops config: subcommand required (validate|show)")
		return 2
	}
	switch args[0] {
	case "validate":
		return runConfigValidate(args[1:], stdout, stderr)
	case "show":
		return runConfigShow(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "fusaops config: unknown subcommand %q\n", args[0])
		return 2
	}
}

// runConfigValidate validates the .fusaops.json config file.
//
//fusa:req REQ-FO-CLI043
func runConfigValidate(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("fusaops config validate", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dir := fs.String("dir", ".", "directory containing .fusaops.json")
	file := fs.String("file", "", "path to config file (overrides --dir)")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	path := *file
	if path == "" {
		path = filepath.Join(*dir, config.ConfigFile)
	}

	cfg, err := config.Load(path)
	if err != nil {
		if errors.Is(err, fusaops.ErrNoConfig) {
			fmt.Fprintf(stderr, "fusaops config validate: no config file at %s\n", path)
			return 1
		}
		fmt.Fprintf(stderr, "fusaops config validate: %v\n", err)
		return 1
	}

	fmt.Fprintf(stdout, "OK  %s\n", path)
	fmt.Fprintf(stdout, "    version:  %s\n", cfg.Version)
	fmt.Fprintf(stdout, "    project:  %s\n", cfg.Project.Name)
	if cfg.Project.Standard != "" {
		fmt.Fprintf(stdout, "    standard: %s\n", cfg.Project.Standard)
	}
	if len(cfg.Scan.Adapters) > 0 {
		fmt.Fprintf(stdout, "    adapters: %v\n", cfg.Scan.Adapters)
	}
	if cfg.Report.Format != "" {
		fmt.Fprintf(stdout, "    format:   %s\n", cfg.Report.Format)
	}
	return 0
}

// runConfigShow prints the effective configuration as formatted JSON.
//
//fusa:req REQ-FO-CLI044
func runConfigShow(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("fusaops config show", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dir := fs.String("dir", ".", "directory containing .fusaops.json")
	file := fs.String("file", "", "path to config file (overrides --dir)")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	path := *file
	if path == "" {
		path = filepath.Join(*dir, config.ConfigFile)
	}

	cfg, err := config.Load(path)
	if err != nil {
		if errors.Is(err, fusaops.ErrNoConfig) {
			fmt.Fprintf(stderr, "fusaops config show: no config file at %s\n", path)
			return 1
		}
		fmt.Fprintf(stderr, "fusaops config show: %v\n", err)
		return 1
	}

	enc := json.NewEncoder(stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(cfg); err != nil {
		fmt.Fprintf(stderr, "fusaops config show: %v\n", err)
		return 1
	}
	return 0
}
