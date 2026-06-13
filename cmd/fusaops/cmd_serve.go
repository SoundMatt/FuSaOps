package main

import (
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/SoundMatt/FuSaOps/orchestrator"
	"github.com/SoundMatt/FuSaOps/server"
)

// runServe launches the web reporting dashboard.
//
//fusa:req REQ-FO-CLI010
//fusa:req REQ-FO-CLI025
//fusa:req REQ-FO-CLI026
//fusa:req REQ-FO-CLI027
//fusa:req REQ-FO-CLI028
//fusa:req REQ-FO-CLI029
func runServe(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("fusaops serve", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dir := fs.String("dir", ".", "project root directory")
	only := fs.String("only", "", "comma-separated tool names to run (default: all applicable)")
	addr := fs.String("addr", ":8080", "listen address")
	auth := fs.String("auth", "", "enable HTTP Basic Auth read-write (user:pass)")
	authRO := fs.String("auth-ro", "", "read-only HTTP Basic Auth credentials (user:pass)")
	auditLog := fs.String("audit-log", "", "directory for .fusaops-audit.jsonl request audit log")
	fleetCfg := fs.String("fleet", "", "path to fleet.json — adds /fleet dashboard")
	tlsCert := fs.String("tls-cert", "", "TLS certificate file (PEM); enables HTTPS")
	tlsKey := fs.String("tls-key", "", "TLS key file (PEM); required with --tls-cert")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	root, opts, _, err := loadOptions(*dir, *only, stderr)
	if err != nil {
		fmt.Fprintf(stderr, "fusaops serve: %v\n", err)
		return 1
	}

	srv := server.New(root, orchestrator.New(nil), opts).WithHistoryDir(root)
	if *auth != "" {
		user, pass, ok := strings.Cut(*auth, ":")
		if !ok {
			fmt.Fprintln(stderr, "fusaops serve: --auth must be in user:pass format")
			return 1
		}
		srv = srv.WithAuth(user, pass)
	}
	if *authRO != "" {
		user, pass, ok := strings.Cut(*authRO, ":")
		if !ok {
			fmt.Fprintln(stderr, "fusaops serve: --auth-ro must be in user:pass format")
			return 1
		}
		srv = srv.WithAuthRO(user, pass)
	}
	if *auditLog != "" {
		srv = srv.WithAuditLog(*auditLog)
	}
	if *fleetCfg != "" {
		srv = srv.WithFleetConfig(*fleetCfg)
	}

	scheme := "http"
	if *tlsCert != "" {
		scheme = "https"
	}
	fmt.Fprintf(stdout, "FuSaOps dashboard for %s\n", root)
	fmt.Fprintf(stdout, "Listening on %s://localhost%s  (Ctrl-C to stop)\n", scheme, *addr)

	if *tlsCert != "" {
		if *tlsKey == "" {
			fmt.Fprintln(stderr, "fusaops serve: --tls-key is required with --tls-cert")
			return 1
		}
		if err := srv.ListenAndServeTLS(*addr, *tlsCert, *tlsKey); err != nil {
			fmt.Fprintf(stderr, "fusaops serve: %v\n", err)
			return 1
		}
		return 0
	}

	if err := srv.ListenAndServe(*addr); err != nil {
		fmt.Fprintf(stderr, "fusaops serve: %v\n", err)
		return 1
	}
	return 0
}
