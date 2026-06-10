// Command fusaops is the FuSaOps multi-language functional safety
// orchestration CLI.
//
// Usage:
//
//	fusaops <command> [flags]
//
// Commands:
//
//	init      Initialise a .fusaops.json project configuration
//	scan      Detect languages and applicable x-FuSa adapters in a repo
//	adapters  List registered adapters and their availability on PATH
//	check      Run every applicable x-FuSa tool and print the aggregate report
//	report     Generate an aggregate report (text/json/html/sarif) to a file
//	trace      Roll up cross-language requirement traceability and qualification
//	sbom       Merge every language's SBOM into one (json/text/spdx)
//	audit-pack Bundle every component's evidence into one ZIP
//	serve      Launch the web reporting dashboard
//	version    Print the FuSaOps version
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
	case "serve":
		return runServe(args[1:], stdout, stderr)
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
  scan      Detect languages and applicable x-FuSa adapters in a repo
  adapters  List registered adapters and their availability on PATH
  check      Run every applicable x-FuSa tool and print the aggregate report
  report     Generate an aggregate report (text/json/html/sarif) to a file
  trace      Roll up cross-language requirement traceability and qualification
  sbom       Merge every language's SBOM into one (json/text/spdx)
  audit-pack Bundle every component's evidence into one ZIP
  serve      Launch the web reporting dashboard
  version    Print the FuSaOps version

Run 'fusaops <command> --help' for per-command flags.
`)
}
