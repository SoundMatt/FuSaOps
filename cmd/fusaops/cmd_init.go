package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/SoundMatt/FuSaOps/config"
)

// runInit writes a starter .fusaops.json to the project directory.
//
//fusa:req REQ-FO-CLI004
func runInit(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("fusaops init", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dir := fs.String("dir", ".", "project root directory")
	name := fs.String("name", "", "project name (default: directory name)")
	force := fs.Bool("force", false, "overwrite an existing .fusaops.json")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	root, err := filepath.Abs(*dir)
	if err != nil {
		fmt.Fprintf(stderr, "fusaops init: %v\n", err)
		return 1
	}
	projName := *name
	if projName == "" {
		projName = filepath.Base(root)
	}
	path := filepath.Join(root, config.ConfigFile)
	if _, err := os.Stat(path); err == nil && !*force {
		fmt.Fprintf(stderr, "fusaops init: %s already exists (use --force to overwrite)\n", path)
		return 1
	}

	cfg := config.Default(projName)
	if err := config.Save(path, cfg); err != nil {
		fmt.Fprintf(stderr, "fusaops init: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "Wrote %s\n", path)
	return 0
}
