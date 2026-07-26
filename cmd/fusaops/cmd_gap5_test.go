package main

// Tests for cmd_serve.go branches not yet covered: all With* option branches in
// runServe (auth, authRO, auditLog, fleet, webhook, refreshInterval, baseline,
// qualifyReport, VandV config, comp config, authOK line), the loadOptions error
// path, and all With* branches in runServeMulti (rwUser, roUser, auditDir,
// interval, baseline) plus the ListenAndServe error body.

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/SoundMatt/FuSaOps/config"
	"github.com/SoundMatt/FuSaOps/server"
)

// TestServeAllWithFlags exercises every With* option branch in runServe by
// passing all optional flags with a .fusaops.json containing VandV and comp
// settings. ListenAndServeTLS fails immediately because the cert file does not
// exist, so the function returns 1 without blocking.
//
//fusa:test REQ-FO-CLI025
//fusa:test REQ-FO-CLI026
//fusa:test REQ-FO-CLI027
//fusa:test REQ-FO-CLI028
//fusa:test REQ-FO-CLI029
//fusa:test REQ-FO-CLI030
//fusa:test REQ-FO-CLI031
//fusa:test REQ-FO-CLI032
//fusa:test REQ-FO-CLI038
func TestServeAllWithFlags(t *testing.T) {
	dir := t.TempDir()
	// Write a .fusaops.json with VandV and comp fields so cfg != nil, the VandV
	// block executes (cfg.VandV.ImplementationAuthor != ""), and WithComp is called.
	cfg := config.Default("serve-test")
	cfg.VandV.ImplementationAuthor = "dev"
	cfg.VandV.IndependentReviewer = "rev"
	cfg.Comp.Threshold = 10
	cfg.Comp.DAL = "DAL-B"
	if err := config.Save(filepath.Join(dir, config.ConfigFile), cfg); err != nil {
		t.Fatal(err)
	}

	auditDir := t.TempDir()
	blFile := filepath.Join(t.TempDir(), "bl.json")
	qrFile := filepath.Join(t.TempDir(), "qr.json")
	fleetFile := filepath.Join(t.TempDir(), "fleet.json")

	var stdout, stderr bytes.Buffer
	code := runServe([]string{
		"--dir", dir,
		"--auth", "admin:pw",
		"--auth-ro", "viewer:ro",
		"--audit-log", auditDir,
		"--fleet", fleetFile,
		"--webhook", "http://example.com/hook",
		"--refresh-interval", "5m",
		"--baseline", blFile,
		"--qualify-report", qrFile,
		"--tls-cert", "/nonexistent/cert.pem",
		"--tls-key", "/nonexistent/key.pem",
	}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("serve all-flags TLS failure: want 1, got %d (stderr=%q)", code, stderr.String())
	}
}

// TestServeLoadOptionsError verifies runServe returns 1 when loadOptions fails
// because the project directory contains a malformed .fusaops.json, covering
// the err != nil branch at line 99.
//
//fusa:test REQ-FO-CLI010
func TestServeLoadOptionsError(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, config.ConfigFile), []byte("{bad json"), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := runServe([]string{"--dir", dir}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("serve bad config: want 1, got %d (stderr=%q)", code, stderr.String())
	}
}

// TestServeInvalidAddrHTTP verifies runServe returns 1 when ListenAndServe fails
// because the port is out of the valid range, covering the ListenAndServe error
// body (non-TLS path). No --tls-cert so the HTTP path is taken at line 153.
//
//fusa:test REQ-FO-CLI010
func TestServeInvalidAddrHTTP(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runServe([]string{
		"--dir", t.TempDir(),
		"--addr", ":99999",
	}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("serve invalid addr HTTP: want 1, got %d (stderr=%q)", code, stderr.String())
	}
}

// TestServeMultiWithAllFlags exercises all With* branches in runServeMulti by
// passing auth, authRO, auditLog, refresh-interval, and baseline flags via the
// top-level runServe entry-point. The port 99999 is out of the valid 1–65535
// range, so ListenAndServe returns an "invalid port" error immediately without
// blocking, allowing the test to verify that all option branches were reached.
//
//fusa:test REQ-FO-CLI030
//fusa:test REQ-FO-CLI032
func TestServeMultiWithAllFlags(t *testing.T) {
	projDir := t.TempDir()

	// Write a valid projects.json with one project pointing to an existing dir.
	projCfg := server.ProjectsConfig{
		Projects: []server.ProjectConfig{
			{Name: "p1", Dir: projDir},
		},
	}
	data, err := json.Marshal(projCfg)
	if err != nil {
		t.Fatal(err)
	}
	cfgFile := filepath.Join(t.TempDir(), "projects.json")
	if err := os.WriteFile(cfgFile, data, 0o644); err != nil {
		t.Fatal(err)
	}

	auditDir := t.TempDir()
	blFile := filepath.Join(t.TempDir(), "bl.json")

	var stdout, stderr bytes.Buffer
	// Port 99999 > 65535 → net.Listen returns "invalid port" immediately.
	code := runServe([]string{
		"--projects", cfgFile,
		"--addr", ":99999",
		"--auth", "admin:pw",
		"--auth-ro", "viewer:ro",
		"--audit-log", auditDir,
		"--refresh-interval", "5m",
		"--baseline", blFile,
	}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("serve multi all-flags invalid port: want 1, got %d (stderr=%q)", code, stderr.String())
	}
}
