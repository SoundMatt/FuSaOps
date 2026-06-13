# FuSaOps Roadmap

## Vision

FuSaOps is the multi-language orchestration layer for the x-FuSa functional
safety toolchain. A polyglot safety-critical repository should be able to run a
single command and receive one unified, auditor-ready evidence view spanning
every language it contains.

It is **NOT** a certification product. It is an engineering accelerator that
removes the per-language friction of producing functional safety evidence.

---

## v0.1 — Foundation ✅

**Goal:** Orchestrate go-FuSa, c-FuSa and cpp-FuSa behind one CLI and web UI.

- Adapter interface + registry; go-FuSa, c-FuSa, cpp-FuSa adapters
- Language detection (`fusaops scan`)
- Orchestrator: run applicable+installed tools, record skipped components
- Aggregate report with text / JSON / HTML / SARIF renderers
- Web dashboard (`fusaops serve`) with status badge, per-language cards,
  filterable findings table, and JSON API
- `.fusaops.json` configuration (optional; zero-config by default)
- go-FuSa self-check in CI (dogfooding)
- Docker image bundling gofusa; multi-platform CI; 80%+ coverage gate

Deliverables: `fusaops init|scan|adapters|check|report|serve|version`

---

## v0.1.1 — Docker Quickstart ✅

**Goal:** Zero-install, all-in-one image that stays current with the tools.

- All-in-one image (`ghcr.io/soundmatt/fusaops`) bundling the x-FuSa tools by
  copying each binary from its own published image (`COPY --from`) — no
  build-from-source, no Docker socket at runtime
- `tools-monitor.yml`: `repository_dispatch` (instant) + weekly schedule rebuild
  so a tool release refreshes the image **without a FuSaOps release or manual
  rebuild**
- `docker compose up` dashboard; CI builds + smoke-tests the image and the
  bundled tools
- One-line extension path for future tools, documented in `docs/extending.md`

Bundled today: `gofusa`. `cpfusa`/`cfusa` activate as their images publish.

---

## v0.1.2 — Safety Evidence & Standards Docs ✅

**Goal:** Bring FuSaOps to go-FuSa-grade safety-artifact parity.

- Requirements registry (`.fusa-reqs.json`, 61 reqs) with full `gofusa trace`
  traceability — every requirement traced **and** tested
- Tool-failure HARA (`.fusa-hara.json`) with safety goals; FuSaOps developed as
  an ISO 26262 ASIL-C tool
- Generated, committed evidence: safety case, TARA, dFMEA, SBOM, provenance,
  coupling, cyber, vuln, qualification report, test-evidence bundle
- Docs: Tool Safety Manual, Qualification (TCL2), Release Process, per-command
  and per-standard references, Incident Response, CLAUDE.md
- CI parity: CodeQL, SARIF upload, DCO; README badges (CI, CodeQL, Go Reference,
  Go Report Card, MPL, standards, image)

---

## v0.2 — Evidence Aggregation ✅

**Goal:** Aggregate more than findings.

- Merge per-language traceability (`trace`) into a cross-language matrix, with
  per-component coverage and qualification status; skipped tools stay visible as
  gaps rather than inflating coverage
- Unified audit-pack: bundle every component's own audit-pack plus the FuSaOps
  aggregate report, trace matrix, and SBOM into one ZIP with a hashed manifest
- Aggregate SBOM across languages — merged, de-duplicated, with SPDX 2.3 output
- Optional adapter capability interfaces (`Tracer`, `Qualifier`, `SBOMer`,
  `Packer`) so a tool contributes evidence only where it supports it

Deliverables: `fusaops trace`, `fusaops audit-pack`, `fusaops sbom`

---

## The x-FuSa contract

[`docs/x-fusa-spec.md`](docs/x-fusa-spec.md) — the master specification all
x-FuSa tools conform to (CLI, JSON schemas, naming, exit codes). Superset of
go/c/cpp-FuSa; go-FuSa is the canonical reference. The standards roll-up below
relies on every tool emitting the spec's `<standard>-gap-report.json`.

---

## v0.3 — Standards Roll-up & Spec Conformance ✅

**Goal:** One compliance view across languages, plus a machine-checkable spec.

- Roll up ISO 26262 / IEC 61508 / DO-178C gap reports from each component
- Per-standard PASS/GAP matrix spanning all languages with skipped-component
  visibility (gaps are never silently dropped)
- x-FuSa spec v1.8 → machine-checkable conformance kit:
  - JSON Schemas for all 9 document kinds (`spec/schemas/`)
  - Golden reference vectors with pre-computed fingerprints (`spec/vectors/`)
  - `fusaops conform <binary>` runner — 37 checks, 8 subcommands, stdlib-only
  - Conformance gate per spec §16 step 7 for onboarding any language tool

Deliverables: `fusaops iso26262`, `fusaops iec61508`, `fusaops do178`,
`fusaops conform`

---

## v0.4 — Monorepo & Component Model ✅

**Goal:** First-class support for large mixed-language monorepos.

- ✅ Per-directory component pinning in `.fusaops.json` (`scan.components[].adapter`, `scan.components[].timeout`)
- ✅ Parallel adapter execution with per-component timeouts (`run.workers`, `run.timeout`, `Options.Timeout/Workers`)
- ✅ Baseline + diff gating (`fusaops diff --baseline --strict`) — fingerprint-matched across all tools
- ✅ `Finding.Category` and `Finding.Fingerprint` fields; `ComputeFingerprint` per spec §4.2
- ✅ x-FuSa spec promoted to v1.9: `category`, `fingerprint`, `remediation`, `capabilities` → MUST

Deliverables: `fusaops diff`, component-scoped scans, 80%+ coverage, 146 requirements

---

## v0.5 — All-Language Support ✅

**Goal:** Complete the x-FuSa adapter set; extend `fusaops conform` to all 6 languages.

- rust-FuSa adapter (`rsfusa`, `LangRust`) — detects `.rs` files
- py-FuSa adapter (`pyfusa`, `LangPython`) — detects `.py` files
- java-FuSa adapter (`jfusa`, `LangJava`) — detects `.java` files
- All six adapters active: go-FuSa · c-FuSa · cpp-FuSa · rust-FuSa · py-FuSa · java-FuSa
- `fusaops conform` extended to all six languages: `langFromBinary` + `writeSourceFiles` for rust/python/java
- x-FuSa spec v1.10.6: full §11 audit of all 6 tools; c-FuSa v0.5.10 now fully conformant

---

## v0.6 — Distribution & Dashboards ✅

**Goal:** Make FuSaOps easy to adopt in any CI and give teams a persistent view.

- ✅ **GitHub Action** (`.github/actions/fusaops/action.yml`) — shipped in v0.5.1
- ✅ Historical metrics trend — `history` package persists Snapshots to `.fusaops-history.jsonl`; `fusaops serve` appends on every refresh
- ✅ Persisted dashboard — `/history` HTML trend page (PASS/FAIL badges, severity bars, per-language breakdown); `/api/history` JSON endpoint
- Pre-built language-bundle images (Go+C, Go+C+C++, all-6) — deferred to v0.7

---

## v0.7 — REST API ✅

**Goal:** Machine-readable API endpoints for CI polling and IDE integration.

- ✅ `GET /api/v1/status` — lightweight PASS/WARN/FAIL/PENDING JSON for CI polling
- ✅ `GET /api/v1/findings?severity=ERROR&language=go&tool=gofusa` — filtered findings
- ✅ `GET /api/v1/report`, `/api/v1/history` — versioned aliases
- ✅ Dashboard nav links (History · JSON · Refresh) in main header

---

## v0.8 — Fleet View ✅

**Goal:** One command to scan multiple repositories and get a combined status.

- ✅ `fleet` package: `Config`, `Repo`, `FleetReport`, `RepoResult`; parallel `Run()`
- ✅ `fusaops fleet --config fleet.json [--format text|json] [--strict]`
- ✅ Columnar text table and JSON output; per-repo PASS/WARN/FAIL/ERROR with error detail

---

## v0.9 — Policy Engine ✅

**Goal:** Let orgs codify safety gates in a machine-checkable policy file.

- ✅ `policy` package: `Policy`, `Rule`, `PolicyReport`, `RuleResult`; `Evaluate()`
- ✅ Rule constraints: `maxFindings`, `maxErrors`, `maxWarnings`, `requireStatus` (PASS/WARN)
- ✅ Scope filters: `language`, `tool`
- ✅ `fusaops policy --policy policy.json [--dir] [--format text|json]`

---

## v1.0 — Enterprise readiness ✅

**Goal:** Secure, enterprise-grade `fusaops serve` with authentication, HTTPS, and a fleet dashboard.

- ✅ **HTTP Basic Auth** — `fusaops serve --auth user:pass` protects all routes with RFC 7617 Basic Auth; unauthenticated requests get 401 + `WWW-Authenticate` challenge
- ✅ **TLS / HTTPS** — `fusaops serve --tls-cert cert.pem --tls-key key.pem` switches to HTTPS (TLS 1.2+)
- ✅ **Fleet web dashboard** — `fusaops serve --fleet fleet.json` adds `/fleet` HTML page and `/api/fleet` JSON endpoint to the serve dashboard; fleet scan runs in parallel on each refresh
- README and CHANGELOG footer links updated for all v0.6–v0.9 releases

---

## v1.1 — Role-based access & audit log ✅

**Goal:** Give operators fine-grained access control and a compliance-ready audit trail.

- ✅ **Read-only credentials** — `fusaops serve --auth-ro viewer:pass` adds a second credential that can read dashboards but is blocked from /refresh (403 Forbidden). Implemented via `Server.WithAuthRO(user, pass)` (REQ-FO-RBAC001/002).
- ✅ **Request audit log** — `fusaops serve --audit-log /var/log` appends one JSON record per authenticated request to `.fusaops-audit.jsonl`: timestamp, method, path, user, HTTP status. Implemented via `Server.WithAuditLog(dir)` (REQ-FO-AUDIT001/002).

---

## v1.2 — Multi-project dashboard ✅

**Goal:** Serve multiple repositories from a single `fusaops serve` process.

- ✅ **`MultiServer`** — holds multiple project entries, runs all scans in parallel on compute
- ✅ **`fusaops serve --projects projects.json`** — switches to multi-project mode
- ✅ **Overview page `/`** — HTML grid of project status cards (name, PASS/WARN/FAIL badge, error/warning counts, link to detail)
- ✅ **`/api/projects`** — JSON array of all project statuses for CI polling
- ✅ **Per-project page `/p/{name}`** — HTML findings table for a single project
- ✅ All v1.0/v1.1 features (auth, ro-auth, audit log) compose with MultiServer

**projects.json format:**
```json
{"projects": [{"name": "firmware", "dir": "/firmware"}, {"name": "app", "dir": "/app"}]}
```

---

## v1.3 — Badge service & webhooks ✅

**Goal:** Make FuSaOps status embeddable and observable.

- ✅ SVG status badge at `/badge/status.svg` (PASS=green, WARN=yellow, FAIL=red, PENDING=gray) in shields.io style; multi-project badge at `/badge/{name}/status.svg`
- ✅ Webhook notifications: `fusaops serve --webhook url` POSTs `{"status":"FAIL","prev":"PASS","errors":3}` when the status transitions; retries once on failure

---

## v1.4 — Scheduled scans ✅

**Goal:** Keep the dashboard current without manual intervention in unattended deployments.

- ✅ `fusaops serve --refresh-interval 5m` — background goroutine rescans every tick after startup (both single-project `Server` and `MultiServer`)
- ✅ Invalid or non-positive interval exits 1 with a descriptive error

---

## v1.5 — OpenMetrics / Prometheus endpoint ✅

**Goal:** First-class Prometheus integration with zero external dependencies.

- ✅ `GET /metrics` — OpenMetrics text exposition (`text/plain; version=0.0.4`) with `fusaops_findings_total{severity}` gauges and `fusaops_status` (1=PASS, 2=WARN, 3=FAIL, 0=pending/error)
- ✅ Multi-project mode emits per-project labels: `fusaops_findings_total{project="firmware",severity="error"}`

---

## v1.6 — Finding suppression ✅

**Goal:** Let teams acknowledge known findings with documented rationale and optional expiry.

- ✅ `suppression` package — `LoadConfig`, `Filter` (fingerprint-based, expiry-aware)
- ✅ `orchestrator.Options.SuppressFile` — suppression applied in `Runner.Run()` before report assembly; `AggregateReport.Suppressed` counter
- ✅ `fusaops check/report --suppress-file .fusaops-suppress.json`
- ✅ Text renderer appends `(N suppressed)` to TOTAL line when suppressions are active

---

## v1.7 — JUnit XML report format ✅

**Goal:** Surface FuSaOps findings natively in CI test result dashboards.

- ✅ `--format junit` on `fusaops check` and `fusaops report` — produces JUnit XML
- ✅ Each component (language × tool) maps to a `<testsuite>`; each finding maps to a `<testcase>` with `<failure>` for ERROR/WARNING
- ✅ Components with zero findings emit a synthetic passing testcase; skipped components emit `<skipped/>`
- ✅ Zero external dependencies — `encoding/xml` stdlib only

---

## v1.8 — CSV report format ✅

**Goal:** Enable auditors and project managers to import findings into spreadsheet tools.

- ✅ `--format csv` on `fusaops check` and `fusaops report` — produces RFC 4180 CSV
- ✅ Columns: language, tool, ruleId, severity, message, file, line, column, category, fingerprint
- ✅ Skipped/unavailable components emit no data rows
- ✅ Zero external dependencies — `encoding/csv` stdlib only

---

## v1.9 — Markdown report format ✅

**Goal:** Enable posting FuSaOps reports directly into PR comments and wiki pages.

- ✅ `--format markdown` (alias `--format md`) on `fusaops check` and `fusaops report`
- ✅ Output: status badge, summary table with emoji severity icons, per-component GFM tables
- ✅ Skipped components show skip reason; clean components show "no findings" note
- ✅ Pipe characters escaped to avoid GFM table breakage

---

## v1.10 — Dashboard search & check --output ✅

**Goal:** Improve CI ergonomics and dashboard usability.

- ✅ **HTML dashboard text search** — live `<input type="search">` filters findings by rule ID, message, and category; composes with severity buttons (AND logic); shows result count badge
- ✅ **`fusaops check --output`** — write check report to a file (like `fusaops report --output`); exit code still reflects gate result

---

## v1.11 — Server export endpoint ✅

**Goal:** Allow CI and IDE integrations to pull the cached report directly from the dashboard server.

- ✅ **`GET /api/v1/export?format=FORMAT`** on both `Server` and `MultiServer` — returns the cached report in the requested format as a file download
- ✅ Supported formats: json (default), text, html, sarif, junit, csv, markdown/md
- ✅ `Content-Disposition: attachment` with format-appropriate filename and MIME type
- ✅ Multi-project mode merges all project components into a fleet-level aggregate before rendering

---

## v1.12 — Diff gate in CI ✅

**Goal:** Expose the baseline-diff feature through the REST API and server, enabling zero-config CI gating based on new-findings-only.

- ✅ `GET /api/v1/diff?baseline=PATH[&strict=true][&format=json|text]` — compare cached report against a baseline file; 409 Conflict when `strict=true` and new ERRORs exist
- ✅ `POST /api/v1/baseline` — save the current cached findings as a baseline file; returns `{"saved":"path","findings":N}`
- ✅ `fusaops serve --baseline baseline.json` — set a default baseline path without per-request query parameters
- ✅ Multi-project mode merges all project findings before diff and save

---

## v1.13 — Per-project suppression & config override

**Goal:** Give multi-project setups independent suppression lists and config without a separate server per project.

- `fusaops serve --projects projects.json` now supports per-project `suppression` and `config` keys in the projects config
- `MultiServer` loads each project's suppression file before computing its report
- `/api/v1/diff` supports `?project=name` to diff a single project in fleet mode
- `fusaops serve --projects projects.json` validates project paths at startup and exits 1 with a descriptive error for missing directories

---

## Adding a language adapter

1. Implement `adapter.Adapter` (Name, Language, Tool, Detect, Available, Check).
2. For tools emitting the standard `check --format json` schema, embed
   `cmdAdapter` and supply the binary name plus source extensions.
3. Register it in an `init()` with `Default.MustRegister`.
4. Add detection extensions to `scan.langExtensions`.
