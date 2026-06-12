# FuSaOps

**Multi-language functional safety orchestration.**

[![CI](https://github.com/SoundMatt/FuSaOps/actions/workflows/ci.yml/badge.svg)](https://github.com/SoundMatt/FuSaOps/actions/workflows/ci.yml)
[![CodeQL](https://github.com/SoundMatt/FuSaOps/actions/workflows/codeql.yml/badge.svg)](https://github.com/SoundMatt/FuSaOps/actions/workflows/codeql.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/SoundMatt/FuSaOps.svg)](https://pkg.go.dev/github.com/SoundMatt/FuSaOps)
[![Go Report Card](https://goreportcard.com/badge/github.com/SoundMatt/FuSaOps)](https://goreportcard.com/report/github.com/SoundMatt/FuSaOps)
[![Go 1.22](https://img.shields.io/badge/Go-1.22-00ADD8?logo=go&logoColor=white)](https://go.dev/dl/)
[![License: MPL 2.0](https://img.shields.io/badge/License-MPL_2.0-brightgreen.svg)](https://www.mozilla.org/en-US/MPL/2.0/)
[![Standards](https://img.shields.io/badge/ISO_26262_·_IEC_61508_·_ISO_21434_·_DO--178C-informational)](docs/standards/)
[![image](https://img.shields.io/badge/ghcr.io-soundmatt%2Ffusaops-blue?logo=docker&logoColor=white)](https://github.com/SoundMatt/FuSaOps/pkgs/container/fusaops)

FuSaOps sits on top of the x-FuSa toolchain — [go-FuSa](https://github.com/SoundMatt/go-FuSa),
[c-FuSa](https://github.com/SoundMatt/c-FuSa), [cpp-FuSa](https://github.com/SoundMatt/cpp-FuSa),
[rust-FuSa](https://github.com/SoundMatt/rust-FuSa), [py-FuSa](https://github.com/SoundMatt/py-FuSa)
and future language tools — and gives mixed-language repositories a single,
intuitive way to **scan**, **aggregate**, and **report** functional safety
evidence. One command runs the right tool for every language present and merges
their results into one report and one web dashboard.

FuSaOps does **not** reimplement language-specific safety rules. It detects the
languages in a repo, delegates to each language's x-FuSa tool, normalises the
machine-readable output, and presents a unified PASS/WARN/FAIL view for ISO 26262,
IEC 61508, ISO 21434, DO-178C and related safety cases.

> It is **NOT** a certification product. It is an engineering accelerator that
> reduces the cost of producing functional safety evidence across a polyglot
> codebase.

---

## How it works

```
            ┌────────────────────────────────── FuSaOps ──────────────────────────────────┐
   repo ──▶ │  scan (detect languages) ──▶ orchestrator ──▶ aggregate report            │ ──▶ text / json / html / sarif
            │                                   │                                        │ ──▶ web dashboard (fusaops serve)
            └───────────────────────────────────┼────────────────────────────────────────┘
                                                 │
          ┌──────────────┬──────────────┬────────┴─────────┬──────────────┐
          ▼              ▼              ▼                   ▼              ▼
       gofusa (Go)   cfusa (C)    cpfusa (C++)        rsfusa (Rust)  pyfusa (Python)
   check --format json  …         …                   …              …
```

Each adapter runs `<tool> check --format json`, FuSaOps decodes the common
report schema, tags every finding with its language and tool, and merges them.
Adapters whose language is present but whose binary is not installed are
recorded as **skipped** components — coverage gaps are never silently dropped.

## Install

```bash
go install github.com/SoundMatt/FuSaOps/cmd/fusaops@latest
```

The adapter tools must be on `PATH` for the languages you want scanned
(`gofusa`, `cfusa`, `cpfusa`, `rsfusa`, `pyfusa`). The Docker image bundles all five.

## Usage

```bash
fusaops scan                 # detect languages and applicable adapters
fusaops adapters             # list adapters and whether each tool is installed
fusaops check                # run every applicable tool; exit 1 on ERROR findings
fusaops check --strict       # also exit 1 on WARNING findings
fusaops report --format html --output fusaops-report.html
fusaops diff --baseline check-report.json   # compare baseline; exit 1 on new errors
fusaops diff --strict        # exit 1 on any new finding (not just errors)
fusaops trace                # cross-language requirement traceability + qualification
fusaops trace --strict       # CI gate: fail on any untraced/untested requirement
fusaops sbom --format spdx   # merged cross-language SBOM (SPDX 2.3)
fusaops audit-pack           # bundle every language's evidence into audit-pack.zip
fusaops iso26262             # roll up ISO 26262 gap reports across all languages
fusaops iec61508             # roll up IEC 61508 gap reports across all languages
fusaops do178                # roll up DO-178C gap reports across all languages
fusaops iso21434             # roll up ISO 21434 gap reports across all languages
fusaops unece                # roll up UNECE R155/R156 gap reports across all languages
fusaops iec62443             # roll up IEC 62443 gap reports across all languages
fusaops trace --gaps         # show only untraced/untested requirements
fusaops trace --req-coverage 80 --sec-tested 60  # threshold-based coverage gate
fusaops conform gofusa       # check a binary against the x-FuSa spec
fusaops serve --addr :8080   # launch the web dashboard
fusaops init                 # write a starter .fusaops.json
```

### Evidence aggregation (v0.2)

Beyond aggregating *findings*, FuSaOps rolls up the evidence each tool already
produces into one cross-language view:

- **`fusaops trace`** — merges every tool's requirement traceability matrix and
  qualification status; `--strict` is a polyglot coverage gate.
- **`fusaops sbom`** — merges and de-duplicates every tool's SBOM, with an
  SPDX 2.3 output.
- **`fusaops audit-pack`** — bundles each tool's own audit-pack plus the
  FuSaOps aggregate report, trace matrix, and SBOM into one ZIP with a hashed
  manifest.

### Standards roll-up (v0.3)

- **`fusaops iso26262`** — rolls up ISO 26262 gap reports from each language
  tool into one cross-language PASS/GAP matrix; `--strict` exits 1 on any gap.
- **`fusaops iec61508`** — same for IEC 61508.
- **`fusaops do178`** — same for DO-178C (maps to the `do178c` canonical id).
- **`fusaops iso21434`** — same for ISO 21434 (automotive cybersecurity).
- **`fusaops unece`** — same for UNECE R155/R156 (vehicle cybersecurity management).
- **`fusaops iec62443`** — same for IEC 62443 (industrial cybersecurity).

### Diff gating (v0.4)

- **`fusaops diff --baseline <file>`** — compares a stored baseline
  `check-report.json` with the findings from a fresh scan, matching by
  fingerprint (§4.2).  Exit 0 when no new errors; exit 1 when new errors appear.
  `--strict` widens the gate to any new finding.  Ideal CI step after storing a
  clean-baseline artifact.

### Monorepo & component pinning (v0.4)

In `.fusaops.json`, pin specific sub-directories to specific adapters and run
everything in parallel:

```json
{
  "run": { "timeout": "60s", "workers": 4 },
  "scan": {
    "components": [
      { "path": "services/auth", "adapter": "gofusa", "timeout": "30s" },
      { "path": "drivers/safety", "adapter": "cfusa" }
    ]
  }
}
```

### Spec conformance (v0.3 / updated v0.4)

- **`fusaops conform <binary>`** — validates any x-FuSa tool binary against the
  spec v1.9 schema and behavioural invariants.  Per spec §16 step 7, this is a
  **MUST** gate for onboarding a new language tool.  See
  [`docs/conformance.md`](docs/conformance.md).

See [`docs/commands/`](docs/commands/) for each command.

### Web dashboard

```bash
fusaops serve
# open http://localhost:8080
```

The dashboard shows an overall status badge, per-language summary cards, and a
filterable findings table. JSON is available at `/api/report`; `/refresh`
re-runs the scan. The page is fully self-contained (no external assets).

## Docker quickstart

The published image is **all-in-one**: it bundles the x-FuSa tools, so there is
nothing else to install. Mount your repo at `/project`.

```bash
# Scan / check a repo (no local Go, no tool installs)
docker run --rm -v "$(pwd)":/project ghcr.io/soundmatt/fusaops scan
docker run --rm -v "$(pwd)":/project ghcr.io/soundmatt/fusaops check

# Web dashboard
docker run --rm -p 8080:8080 -v "$(pwd)":/project \
  ghcr.io/soundmatt/fusaops serve --addr :8080
# or: docker compose up   →  http://localhost:8080
```

**How the image stays current.** Each tool binary is copied from that tool's own
published image (`ghcr.io/soundmatt/<x>-fusa`). When an x-FuSa releases, a
`repository_dispatch` rebuilds `ghcr.io/soundmatt/fusaops:latest` with the fresh
tool — **no manual rebuild, and FuSaOps itself does not need a new release**. A
weekly scheduled rebuild is the safety net. See
[`docs/extending.md`](docs/extending.md).

**Bundled tools.** The image bundles `gofusa` (Go, v0.25), `cpfusa` (C++,
v0.9.2), and `rsfusa` (Rust, v0.2.0). `cfusa` and `pyfusa` are registered
adapters but not yet bundled — those tools do not yet publish a Docker image.

> The image is `linux/amd64` (the tool images are amd64). On Apple Silicon it
> runs under emulation; add `--platform linux/amd64` if your client needs it.

## Configuration

`.fusaops.json` is optional — FuSaOps works zero-config by detecting languages
directly. When present it sets the project name, restricts which adapters run,
and sets report defaults:

```json
{
  "version": "1",
  "project": { "name": "my-system", "standard": "ISO26262" },
  "scan": { "adapters": ["gofusa", "cpfusa"], "exclude": ["third_party"] },
  "report": { "format": "html", "output": "fusaops-report.html" }
}
```

Per-language configuration stays in each component's own x-FuSa config (e.g. a
Go module's `.fusa.json`); FuSaOps does not manage it.

## CI integration

```yaml
- run: go install github.com/SoundMatt/FuSaOps/cmd/fusaops@latest
- run: fusaops check        # fails the build on any ERROR finding, any language
```

## Self-hosting

FuSaOps is written in Go, so it eats its own dog food: the **`go-FuSa self-check`**
CI job installs `gofusa` and runs `gofusa check` against the FuSaOps source on
every run, gating on ERROR findings (see `.github/workflows/ci.yml`). CodeQL and
a SARIF upload of those findings run alongside it. The toolchain FuSaOps
orchestrates also gates FuSaOps itself.

## Supported languages

| Language | Adapter    | Tool     | Bundled in image |
|----------|------------|----------|------------------|
| Go       | go-FuSa    | `gofusa` | ✅ (v0.25, spec v1.9) |
| C++      | cpp-FuSa   | `cpfusa` | ✅ (v0.9.2, spec v1.9) |
| C        | c-FuSa     | `cfusa`  | ✅ (v0.5.1, spec v1.9) |
| Rust     | rust-FuSa  | `rsfusa` | ✅ (v0.2.0, spec v1.9) |
| Python   | py-FuSa    | `pyfusa` | ✅ (v0.1.1, spec v1.9, alpha) |

All five adapters exist; an un-bundled tool reports as *not installed* until its
image publishes. New languages are added by implementing the `adapter.Adapter`
interface — see [docs/extending.md](docs/extending.md).

## Safety & standards

FuSaOps is itself developed as an ISO 26262 **ASIL-C** tool and carries the
go-FuSa-grade evidence set. It aggregates evidence relevant to
**ISO 26262, IEC 61508, ISO 21434, and DO-178C** across the languages it
orchestrates.

- **Requirements** — [`.fusa-reqs.json`](.fusa-reqs.json) (146 requirements);
  `gofusa trace` reports them all traced **and** tested.
- **HARA** — [`.fusa-hara.json`](.fusa-hara.json) (tool-failure hazards + safety goals).
- **Tool Safety Manual** — [docs/tool-safety-manual.md](docs/tool-safety-manual.md)
  (intended use, assumptions, hazards, mitigations).
- **Qualification** — [docs/qualification.md](docs/qualification.md) (TCL2).
- **Standards** — [docs/standards/](docs/standards/) ·
  **Commands** — [docs/commands/](docs/commands/) ·
  **Release** — [docs/release-process.md](docs/release-process.md) ·
  **Incident response** — [INCIDENT-RESPONSE.md](INCIDENT-RESPONSE.md).
- **Generated evidence** (committed) — safety case, TARA, dFMEA, SBOM,
  provenance, coupling, cyber, vuln, qualification report, test-evidence bundle.

## License

[Mozilla Public License 2.0](LICENSE).
