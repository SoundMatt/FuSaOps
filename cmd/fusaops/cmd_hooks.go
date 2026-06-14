package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

const preCommitScript = `#!/bin/sh
# FuSaOps pre-commit hook — installed by: fusaops hooks install
set -e
if command -v fusaops >/dev/null 2>&1; then
  fusaops check --strict
else
  echo "fusaops: not found in PATH; skipping safety check" >&2
fi
`

// runHooks manages git hooks for FuSaOps integration.
//
//fusa:req REQ-FO-HOOKS001
//fusa:req REQ-FO-CLI058
func runHooks(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("fusaops hooks", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() {
		fmt.Fprintf(stderr, "Usage: fusaops hooks <subcommand> [flags]\n\n")
		fmt.Fprintf(stderr, "Manage git hooks for FuSaOps integration.\n\n")
		fmt.Fprintf(stderr, "Subcommands:\n")
		fmt.Fprintf(stderr, "  install   Install pre-commit hook into .git/hooks/\n")
		fmt.Fprintf(stderr, "  remove    Remove the FuSaOps pre-commit hook\n")
		fmt.Fprintf(stderr, "  show      Print the hook script to stdout\n\n")
		fmt.Fprintf(stderr, "Flags:\n")
		fs.PrintDefaults()
	}
	dir := fs.String("dir", ".", "project root directory")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() == 0 {
		fs.Usage()
		return 2
	}

	hookPath := filepath.Join(*dir, ".git", "hooks", "pre-commit")

	switch fs.Arg(0) {
	case "install":
		return hooksInstall(hookPath, stdout, stderr)
	case "remove":
		return hooksRemove(hookPath, stdout, stderr)
	case "show":
		fmt.Fprint(stdout, preCommitScript)
		return 0
	default:
		fmt.Fprintf(stderr, "fusaops hooks: unknown subcommand %q (want install, remove, or show)\n", fs.Arg(0))
		return 2
	}
}

//fusa:req REQ-FO-HOOKS001
func hooksInstall(hookPath string, stdout, stderr io.Writer) int {
	if _, err := os.Stat(hookPath); err == nil {
		fmt.Fprintf(stderr, "fusaops hooks: %s already exists; run 'fusaops hooks remove' first\n", hookPath)
		return 1
	}
	if err := os.MkdirAll(filepath.Dir(hookPath), 0o750); err != nil {
		fmt.Fprintf(stderr, "fusaops hooks: create hooks dir: %v\n", err)
		return 1
	}
	if err := os.WriteFile(hookPath, []byte(preCommitScript), 0o750); err != nil {
		fmt.Fprintf(stderr, "fusaops hooks: write hook: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "FuSaOps pre-commit hook installed: %s\n", hookPath)
	fmt.Fprintf(stdout, "Hook runs 'fusaops check --strict' on every commit.\n")
	return 0
}

//fusa:req REQ-FO-HOOKS001
func hooksRemove(hookPath string, stdout, stderr io.Writer) int {
	if err := os.Remove(hookPath); err != nil {
		if os.IsNotExist(err) {
			fmt.Fprintf(stderr, "fusaops hooks: hook not found: %s\n", hookPath)
			return 1
		}
		fmt.Fprintf(stderr, "fusaops hooks: remove hook: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "FuSaOps pre-commit hook removed: %s\n", hookPath)
	return 0
}
