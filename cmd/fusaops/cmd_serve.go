package main

import (
	"flag"
	"fmt"
	"io"

	"github.com/SoundMatt/FuSaOps/orchestrator"
	"github.com/SoundMatt/FuSaOps/server"
)

// runServe launches the web reporting dashboard.
//
//fusa:req REQ-FO-CLI010
func runServe(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("fusaops serve", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dir := fs.String("dir", ".", "project root directory")
	only := fs.String("only", "", "comma-separated tool names to run (default: all applicable)")
	addr := fs.String("addr", ":8080", "listen address")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	root, opts, _, err := loadOptions(*dir, *only, stderr)
	if err != nil {
		fmt.Fprintf(stderr, "fusaops serve: %v\n", err)
		return 1
	}

	srv := server.New(root, orchestrator.New(nil), opts).WithHistoryDir(root)
	fmt.Fprintf(stdout, "FuSaOps dashboard for %s\n", root)
	fmt.Fprintf(stdout, "Listening on http://localhost%s  (Ctrl-C to stop)\n", *addr)
	if err := srv.ListenAndServe(*addr); err != nil {
		fmt.Fprintf(stderr, "fusaops serve: %v\n", err)
		return 1
	}
	return 0
}
