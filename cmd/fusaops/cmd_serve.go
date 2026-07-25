package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/SoundMatt/FuSaOps/orchestrator"
	"github.com/SoundMatt/FuSaOps/server"
	"github.com/SoundMatt/FuSaOps/vv"
)

// runServe launches the web reporting dashboard.
//
//fusa:req REQ-FO-CLI010
//fusa:req REQ-FO-CLI025
//fusa:req REQ-FO-CLI026
//fusa:req REQ-FO-CLI027
//fusa:req REQ-FO-CLI028
//fusa:req REQ-FO-CLI029
//fusa:req REQ-FO-CLI030
//fusa:req REQ-FO-CLI031
//fusa:req REQ-FO-CLI032
//fusa:req REQ-FO-CLI038
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
	projectsCfg := fs.String("projects", "", "path to projects.json — multi-project dashboard mode")
	webhook := fs.String("webhook", "", "URL to POST status-change notifications to")
	tlsCert := fs.String("tls-cert", "", "TLS certificate file (PEM); enables HTTPS")
	tlsKey := fs.String("tls-key", "", "TLS key file (PEM); required with --tls-cert")
	refreshInterval := fs.String("refresh-interval", "", "automatic rescan interval (e.g. 5m, 1h); default disabled")
	baseline := fs.String("baseline", "", "path to baseline JSON file for /api/v1/diff and POST /api/v1/baseline")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	// Validate shared flags.
	rwUser, rwPass, authOK := "", "", true
	if *auth != "" {
		var ok bool
		rwUser, rwPass, ok = strings.Cut(*auth, ":")
		if !ok {
			fmt.Fprintln(stderr, "fusaops serve: --auth must be in user:pass format")
			return 1
		}
		authOK = true
	}
	roUser, roPass := "", ""
	if *authRO != "" {
		var ok bool
		roUser, roPass, ok = strings.Cut(*authRO, ":")
		if !ok {
			fmt.Fprintln(stderr, "fusaops serve: --auth-ro must be in user:pass format")
			return 1
		}
	}
	if *tlsCert != "" && *tlsKey == "" {
		fmt.Fprintln(stderr, "fusaops serve: --tls-key is required with --tls-cert")
		return 1
	}
	var interval time.Duration
	if *refreshInterval != "" {
		var err error
		interval, err = time.ParseDuration(*refreshInterval)
		if err != nil || interval <= 0 {
			fmt.Fprintln(stderr, "fusaops serve: --refresh-interval must be a positive duration (e.g. 5m)")
			return 1
		}
	}
	_ = authOK

	scheme := "http"
	if *tlsCert != "" {
		scheme = "https"
	}

	// Multi-project mode.
	if *projectsCfg != "" {
		return runServeMulti(*projectsCfg, *addr, scheme, *tlsCert, *tlsKey,
			rwUser, rwPass, roUser, roPass, *auditLog, *baseline, interval, stdout, stderr)
	}

	// Single-project mode.
	root, opts, cfg, err := loadOptions(*dir, *only, stderr)
	if err != nil {
		fmt.Fprintf(stderr, "fusaops serve: %v\n", err)
		return 1
	}
	srv := server.New(root, orchestrator.New(nil), opts).WithHistoryDir(root)
	if cfg != nil && (cfg.VandV.ImplementationAuthor != "" ||
		cfg.VandV.IndependentReviewer != "" ||
		cfg.VandV.IndependentTestExecutor != "") {
		decl := vv.Declaration{
			Project:                 cfg.Project.Name,
			ImplementationAuthor:    cfg.VandV.ImplementationAuthor,
			IndependentReviewer:     cfg.VandV.IndependentReviewer,
			IndependentTestExecutor: cfg.VandV.IndependentTestExecutor,
		}
		srv = srv.WithVandV(decl)
	}
	if rwUser != "" {
		srv = srv.WithAuth(rwUser, rwPass)
	}
	if roUser != "" {
		srv = srv.WithAuthRO(roUser, roPass)
	}
	if *auditLog != "" {
		srv = srv.WithAuditLog(*auditLog)
	}
	if *fleetCfg != "" {
		srv = srv.WithFleetConfig(*fleetCfg)
	}
	if *webhook != "" {
		srv = srv.WithWebhook(*webhook)
	}
	if interval > 0 {
		srv = srv.WithRefreshInterval(interval)
	}
	if *baseline != "" {
		srv = srv.WithBaseline(*baseline)
	}

	fmt.Fprintf(stdout, "FuSaOps dashboard for %s\n", root)
	fmt.Fprintf(stdout, "Listening on %s://localhost%s  (Ctrl-C to stop)\n", scheme, *addr)

	if *tlsCert != "" {
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

// runServeMulti handles fusaops serve --projects.
//
//fusa:req REQ-FO-CLI030
//fusa:req REQ-FO-CLI032
//fusa:req REQ-FO-CLI038
//fusa:req REQ-FO-MPJ007
func runServeMulti(cfgPath, addr, scheme, tlsCert, tlsKey,
	rwUser, rwPass, roUser, roPass, auditDir, baselineFile string, interval time.Duration,
	stdout, stderr io.Writer) int {
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		fmt.Fprintf(stderr, "fusaops serve: projects config: %v\n", err)
		return 1
	}
	var cfg server.ProjectsConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		fmt.Fprintf(stderr, "fusaops serve: projects config: %v\n", err)
		return 1
	}
	ms := server.NewMulti(cfg, orchestrator.New(nil))
	if errs := ms.ValidateProjectDirs(); len(errs) > 0 {
		for _, e := range errs {
			fmt.Fprintf(stderr, "fusaops serve: %v\n", e)
		}
		return 1
	}
	if rwUser != "" {
		ms = ms.WithAuth(rwUser, rwPass)
	}
	if roUser != "" {
		ms = ms.WithAuthRO(roUser, roPass)
	}
	if auditDir != "" {
		ms = ms.WithAuditLog(auditDir)
	}
	if interval > 0 {
		ms = ms.WithRefreshInterval(interval)
	}
	if baselineFile != "" {
		ms = ms.WithBaseline(baselineFile)
	}
	fmt.Fprintf(stdout, "FuSaOps multi-project dashboard (%d projects)\n", len(cfg.Projects))
	fmt.Fprintf(stdout, "Listening on %s://localhost%s  (Ctrl-C to stop)\n", scheme, addr)
	if err := ms.ListenAndServe(addr); err != nil {
		fmt.Fprintf(stderr, "fusaops serve: %v\n", err)
		return 1
	}
	return 0
}
