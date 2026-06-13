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
[rust-FuSa](https://github.com/SoundMatt/rust-FuSa), [py-FuSa](https://github.com/SoundMatt/py-FuSa),
[java-FuSa](https://github.com/SoundMatt/java-FuSa)
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
fusaops serve --auth user:pass           # enable HTTP Basic Auth
fusaops serve --tls-cert c.pem --tls-key k.pem  # HTTPS
fusaops serve --fleet fleet.json         # add /fleet multi-repo dashboard
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

### Web dashboard (v0.1 / v0.6 / v0.7)

```bash
fusaops serve
# open http://localhost:8080
```

The dashboard shows an overall status badge, per-language summary cards, and a
filterable findings table. The page is fully self-contained (no external assets).

| Endpoint | Description |
|---|---|
| `/` | HTML dashboard — PASS/WARN/FAIL badge, per-language cards, findings table |
| `/history` | HTML trend page — PASS/FAIL badges, severity bars, per-language breakdown |
| `/refresh` | Re-runs the scan and updates the dashboard |
| `/api/report` | Full aggregate JSON report |
| `/api/history` | JSON array of run snapshots (persisted in `.fusaops-history.jsonl`) |
| `/api/v1/status` | Lightweight poll endpoint: `{"status":"PASS","errors":0,"warnings":1,"total":1}` |
| `/api/v1/findings` | Filtered findings: `?severity=ERROR&language=go&tool=gofusa` |
| `/api/v1/report` | Versioned alias for `/api/report` |
| `/api/v1/history` | Versioned alias for `/api/history` |

Run history is persisted to `.fusaops-history.jsonl` automatically; the `/history` trend page
and `/api/history` endpoint are available after the first run.

### Fleet view (v0.8)

Scan multiple repositories with one command:

```bash
fusaops fleet --config fleet.json              # columnar text output
fusaops fleet --config fleet.json --format json
fusaops fleet --config fleet.json --strict     # exit 1 on any WARNING
```

**Fleet config format:**

```json
{
  "project": "my-system",
  "repos": [
    { "name": "firmware", "dir": "/path/to/firmware", "adapter": "cfusa" },
    { "name": "app",      "dir": "/path/to/app" }
  ]
}
```

Each repo is scanned in parallel. `adapter` is optional — omitting it detects
languages automatically. The output is a columnar table (or JSON) with per-repo
PASS/WARN/FAIL status and finding counts.

### Policy engine (v0.9)

Codify org-wide safety gates in a JSON policy file:

```bash
fusaops policy --policy policy.json              # evaluate against current scan
fusaops policy --policy policy.json --dir ./src
fusaops policy --policy policy.json --format json
```

**Policy config format:**

```json
{
  "name": "ci-gate",
  "rules": [
    { "id": "no-errors",        "requireStatus": "WARN" },
    { "id": "go-strict",        "language": "go",  "requireStatus": "PASS" },
    { "id": "cpp-error-budget", "language": "cpp", "maxErrors": 5 }
  ]
}
```

Each rule can scope to a `language` and/or `tool`, and can enforce `maxFindings`,
`maxErrors`, `maxWarnings`, and/or `requireStatus` (`PASS` = zero errors + zero
warnings; `WARN` = zero errors, warnings allowed). `fusaops policy` exits 1 if
any rule fails.

### Enterprise features (v1.0)

`fusaops serve` can be hardened for team and enterprise use:

```bash
# Password-protect the dashboard (HTTP Basic Auth)
fusaops serve --auth admin:secret

# HTTPS (TLS 1.2+)
fusaops serve --tls-cert /etc/certs/server.pem --tls-key /etc/certs/server-key.pem

# Combined fleet + auth + HTTPS
fusaops serve \
  --fleet fleet.json \
  --auth admin:secret \
  --tls-cert server.pem --tls-key server.key
```

When `--fleet` is set, the server adds two routes to the existing dashboard:
- `/fleet` — HTML page: per-repo PASS/WARN/FAIL badge, error/warning counts
- `/api/fleet` — JSON: full `FleetReport` for CI polling

### Multi-project dashboard (v1.2)

Serve multiple repositories from a single process:

```bash
fusaops serve --projects projects.json
```

**projects.json format:**
```json
{
  "projects": [
    { "name": "firmware", "dir": "/path/to/firmware" },
    { "name": "app",      "dir": "/path/to/app", "adapter": "gofusa" }
  ]
}
```

All projects are scanned in parallel on startup and on `/refresh`. Routes:
- `/` — HTML grid of project status cards (badge, counts, detail link)
- `/api/projects` — JSON array of all project statuses
- `/p/{name}` — HTML findings table for a single project

All enterprise flags (`--auth`, `--auth-ro`, `--audit-log`, `--tls-cert`) compose with `--projects`.

## Badge service & webhooks

### SVG status badges

Both `Server` and `MultiServer` expose embeddable SVG badges:

| Route | Description |
|---|---|
| `/badge/status.svg` | Overall PASS/WARN/FAIL/pending badge |
| `/badge/{name}/status.svg` | Per-project badge (multi-project mode) |

Embed in a README:
```markdown
![FuSaOps status](http://localhost:8080/badge/status.svg)
```

Badge colours follow shields.io convention: green (PASS), yellow (WARN), red (FAIL/error), gray (pending). Responses carry `Cache-Control: no-cache`.

### Webhook notifications

```bash
fusaops serve --webhook https://hooks.example.com/fusaops
```

When the aggregate status transitions (e.g. PASS → FAIL), FuSaOps POSTs:

```json
{"status":"FAIL","prev":"PASS","errors":3}
```

The server retries once after 2 seconds on failure. Webhooks integrate with Slack, PagerDuty, or any HTTP receiver.

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

**Bundled tools.** The image currently bundles `gofusa` (Go, v0.30.0). All six
adapters are registered; the other tools' Docker images will be added to the
bundle as they publish to GHCR (cpp-FuSa v0.12.5, c-FuSa v0.5.16, rust-FuSa v0.2.6,
py-FuSa v0.1.4, java-FuSa v0.2.0 are each spec v1.10 aligned). The java-FuSa
image requires a JVM in the runtime stage; the Dockerfile will add `openjdk-21-jre`
when the image is wired in.

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

### GitHub Action (zero-install)

The reusable action wraps the all-in-one Docker image — no Go installation or tool
setup needed in your CI:

```yaml
steps:
  - uses: actions/checkout@v4
  - uses: SoundMatt/FuSaOps/.github/actions/fusaops@v0.9.0
    # Runs "fusaops check" by default; exit 1 on any ERROR finding, any language.

  # With options:
  - uses: SoundMatt/FuSaOps/.github/actions/fusaops@v0.9.0
    with:
      args: '--strict'        # also gate on WARNING findings
      upload-report: 'true'  # attach fusaops-report.html as a workflow artifact
```

**Action inputs:**

| Input | Default | Description |
|---|---|---|
| `command` | `check` | Any fusaops subcommand (`check`, `trace`, `report`, `sbom`, `audit-pack`, ...) |
| `args` | _(empty)_ | Extra flags appended after the command |
| `image` | `ghcr.io/soundmatt/fusaops:latest` | Docker image to pull |
| `upload-report` | `false` | When `"true"`, generates an HTML report and uploads it as `fusaops-report` artifact |

See [`.github/fusaops-example.yml`](.github/fusaops-example.yml) for more usage patterns.

### Direct install

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
| Go       | go-FuSa    | `gofusa` | ✅ (v0.30.0, spec v1.10) |
| C++      | cpp-FuSa   | `cpfusa` | ✅ (v0.12.5, spec v1.10) |
| C        | c-FuSa     | `cfusa`  | ✅ (v0.5.16, spec v1.10) |
| Rust     | rust-FuSa  | `rsfusa` | ✅ (v0.2.6, spec v1.10) |
| Python   | py-FuSa    | `pyfusa` | ✅ (v0.1.4, spec v1.10, alpha) |
| Java     | java-FuSa  | `jfusa`  | ✅ (v0.2.0, spec v1.10, alpha) |

All six adapters exist; an un-bundled tool reports as *not installed* until its
image publishes. New languages are added by implementing the `adapter.Adapter`
interface — see [docs/extending.md](docs/extending.md).

## Safety & standards

FuSaOps is itself developed as an ISO 26262 **ASIL-C** tool and carries the
go-FuSa-grade evidence set. It aggregates evidence relevant to
**ISO 26262, IEC 61508, ISO 21434, and DO-178C** across the languages it
orchestrates.

- **Requirements** — [`.fusa-reqs.json`](.fusa-reqs.json) (198 requirements);
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
