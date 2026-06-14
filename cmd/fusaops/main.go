// Command fusaops is the FuSaOps multi-language functional safety
// orchestration CLI.
//
// Usage:
//
//	fusaops <command> [flags]
//
// Commands:
//
//	init        Initialise a .fusaops.json project configuration
//	config      Validate or display the effective .fusaops.json configuration
//	history     List or prune the .fusaops-history.jsonl check run history
//	scan        Detect languages and applicable x-FuSa adapters in a repo
//	adapters    List registered adapters and their availability on PATH
//	check       Run every applicable x-FuSa tool and print the aggregate report
//	report      Generate an aggregate report (text/json/html/sarif) to a file
//	trace       Roll up cross-language requirement traceability and qualification
//	sbom        Merge every language's SBOM into one (json/text/spdx)
//	audit-pack  Bundle every component's evidence into one ZIP
//	diff        Compare a baseline check-report with the current scan
//	suppress    Manage the .fusaops-suppress.json suppression list
//	conform     Run x-FuSa spec conformance checks against a tool binary
//	iso26262    Roll up ISO 26262 gap reports across all languages
//	iec61508    Roll up IEC 61508 gap reports across all languages
//	do178       Roll up DO-178C gap reports across all languages
//	iso21434    Roll up ISO 21434 gap reports across all languages
//	unece       Roll up UNECE R155/R156 gap reports across all languages
//	iec62443    Roll up IEC 62443 gap reports across all languages
//	policy      Evaluate org-wide safety rules over the aggregated report
//	fleet       Run check across all repos in a fleet config file
//	coverage      DO-178C structural coverage report from a Go coverage profile
//	req           Show, import, or export requirements from .fusa-reqs.json
//	capabilities  Report FuSaOps's supported commands, formats, and standards
//	metrics       Track project safety metrics over time
//	badge         Generate an SVG status badge from an aggregate check report
//	slsa          Generate a SLSA supply-chain integrity gap report
//	serve       Launch the web reporting dashboard
//	version     Print the FuSaOps version
//
// Run 'fusaops <command> --help' for per-command flags.
package main

import (
	"fmt"
	"io"
	"os"

	// Blank imports activate the built-in adapters via their init functions.
	_ "github.com/SoundMatt/FuSaOps/adapter"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run dispatches a single CLI invocation and returns the process exit code.
//
//fusa:req REQ-FO-CLI001
func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		usage(stdout)
		return 1
	}
	switch args[0] {
	case "init":
		return runInit(args[1:], stdout, stderr)
	case "scan":
		return runScan(args[1:], stdout, stderr)
	case "adapters":
		return runAdapters(args[1:], stdout, stderr)
	case "check":
		return runCheck(args[1:], stdout, stderr)
	case "report":
		return runReport(args[1:], stdout, stderr)
	case "trace":
		return runTrace(args[1:], stdout, stderr)
	case "sbom":
		return runSBOM(args[1:], stdout, stderr)
	case "audit-pack":
		return runAuditPack(args[1:], stdout, stderr)
	case "diff":
		return runDiff(args[1:], stdout, stderr)
	case "suppress":
		return runSuppress(args[1:], stdout, stderr)
	case "conform":
		return runConform(args[1:], stdout, stderr)
	case "iso26262", "iec61508", "do178", "iso21434", "unece", "iec62443":
		return runStandards(args[0], args[1:], stdout, stderr)
	case "policy":
		return runPolicy(args[1:], stdout, stderr)
	case "fleet":
		return runFleet(args[1:], stdout, stderr)
	case "coverage":
		return runCoverage(args[1:], stdout, stderr)
	case "req":
		return runReq(args[1:], stdout, stderr)
	case "capabilities":
		return runCapabilities(args[1:], stdout, stderr)
	case "metrics":
		return runMetrics(args[1:], stdout, stderr)
	case "badge":
		return runBadge(args[1:], stdout, stderr)
	case "slsa":
		return runSLSA(args[1:], stdout, stderr)
	case "serve":
		return runServe(args[1:], stdout, stderr)
	case "config":
		return runConfig(args[1:], stdout, stderr)
	case "history":
		return runHistory(args[1:], stdout, stderr)
	case "version":
		return runVersion(args[1:], stdout, stderr)
	case "help", "-h", "--help":
		usage(stdout)
		return 0
	default:
		fmt.Fprintf(stderr, "fusaops: unknown command %q\n\n", args[0])
		usage(stderr)
		return 1
	}
}

// usage prints the top-level help text.
//
//fusa:req REQ-FO-CLI002
func usage(w io.Writer) {
	fmt.Fprint(w, `FuSaOps — multi-language functional safety orchestration

Usage:
  fusaops <command> [flags]

Commands:
  init      Initialise a .fusaops.json project configuration
  config    Validate or display the effective .fusaops.json configuration
  history   List or prune the .fusaops-history.jsonl check run history
  scan      Detect languages and applicable x-FuSa adapters in a repo
  adapters  List registered adapters and their availability on PATH
  check      Run every applicable x-FuSa tool and print the aggregate report
  report     Generate an aggregate report (text/json/html/sarif) to a file
  trace      Roll up cross-language requirement traceability and qualification
  sbom       Merge every language's SBOM into one (json/text/spdx)
  audit-pack Bundle every component's evidence into one ZIP
  diff       Compare a baseline check-report with the current scan
  suppress   Manage the .fusaops-suppress.json suppression list
  conform    Run x-FuSa spec conformance checks against a tool binary
  iso26262   Roll up ISO 26262 gap reports across all languages
  iec61508   Roll up IEC 61508 gap reports across all languages
  do178      Roll up DO-178C gap reports across all languages
  iso21434   Roll up ISO 21434 gap reports across all languages
  unece      Roll up UNECE R155/R156 gap reports across all languages
  iec62443   Roll up IEC 62443 gap reports across all languages
  policy     Evaluate org-wide safety rules over the aggregated report
  fleet      Run check across all repos in a fleet config file
  coverage   DO-178C structural coverage report from a Go coverage profile
  req           Show, import, or export requirements from .fusa-reqs.json
  capabilities  Report FuSaOps's supported commands, formats, and standards
  metrics       Track project safety metrics over time
  badge         Generate an SVG status badge from an aggregate check report
  slsa          Generate a SLSA supply-chain integrity gap report
  serve         Launch the web reporting dashboard
  version    Print the FuSaOps version

Run 'fusaops <command> --help' for per-command flags.
`)
}
