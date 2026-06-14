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

## v1.13 — Per-project suppression & config override ✅

**Goal:** Give multi-project setups independent suppression lists and config without a separate server per project.

- ✅ Per-project `"suppression"` key in `projects.json` → applied as independent `SuppressFile` per project
- ✅ Auto-load `.fusaops.json` from each project's directory (project name override, adapter filter)
- ✅ `/api/v1/diff?project=name` — diff a single named project in fleet mode; 503 for unknown names
- ✅ Startup path validation: `fusaops serve --projects` exits 1 with a descriptive error for missing dirs

---

## v1.14 — Suppression management CLI ✅

**Goal:** Make suppression list management ergonomic from the command line.

- ✅ `fusaops suppress add --fingerprint sha256:<hex> --reason text [--expires YYYY-MM-DD]` — append to `.fusaops-suppress.json`
- ✅ `fusaops suppress list [--format text|json]` — show all suppressions with active/expired status, reason, expiry
- ✅ `fusaops suppress prune` — remove expired entries; print count removed
- ✅ `fusaops suppress verify [--dir .]` — run orchestrator, exit 1 with a per-entry report for stale (unmatched) suppressions

---

## v1.41 — `fusaops badge` — SVG status badge ✅

**Goal:** Mirror `gofusa badge` for colour-coded health indicators in README files and CI artefacts.

- `badge` package — `Badge`/`Status` types; `New(errors, warnings, version)` (REQ-FO-BADGE001); `Render(w, badge)` Shields.io-style SVG (REQ-FO-BADGE002)
- `fusaops badge [--output FILE] [report.json]` — reads aggregate report JSON, emits SVG (REQ-FO-CLI056)
- `fusaops capabilities` updated with badge command and svg format

---

## v1.40 — `fusaops metrics` — safety metrics time series ✅

**Goal:** Mirror `gofusa metrics` for CI-level safety KPI tracking across releases.

- `metrics` package — `Snapshot` and `TimeSeries` types; `Load`, `Save`, `Append`, `Collect`, `Render` (text/json) (REQ-FO-MET001, REQ-FO-MET002, REQ-FO-MET003)
- `fusaops metrics record` — collect snapshot from project artefacts, append to `.fusaops-metrics.json` (REQ-FO-CLI055)
- `fusaops metrics show [--format text|json] [--output FILE]` — display full time series (REQ-FO-CLI055)

---

## v1.39 — `fusaops coverage --format markdown` ✅

**Goal:** Complete the coverage command format matrix for PR-comment embedding.

- `coverage.Render` gains `"markdown"` / `"md"` format: 🟢/🟡/🔴 badge, statement/decision/MC/DC table with required flags, coverage gaps file table (REQ-FO-COV003)
- `fusaops coverage --format markdown [--output FILE]`
- `fusaops capabilities` format map updated

---

## v1.38 — `fusaops capabilities` — §9.1 discovery document ✅

**Goal:** Mirror `gofusa capabilities` for machine-readable discovery.

- `fusaops capabilities` — emits §9.1 `kind: "capabilities"` JSON document with tool, version, spec version, commands, per-command formats, and standards (REQ-FO-CLI054)
- JSON-only (per spec §9.1); `--format text` returns exit 2

---

## v1.37 — `fusaops version --format json` and `SpecVersion` ✅

**Goal:** Mirror `gofusa version --format json` for machine-readable version introspection.

- `fusaops.SpecVersion = "1.10.4"` — exported constant for the x-FuSa spec version FuSaOps targets (REQ-FO-CORE007)
- `fusaops version --format json` — emits `{"tool":"fusaops","version":"...","specVersion":"..."}` (REQ-FO-CLI053)
- Default `text` format unchanged

---

## v1.36 — `fusaops req` — requirement show/import/export ✅

**Goal:** Mirror `gofusa req` for requirement management at the FuSaOps orchestration level.

- `req` package: `Entry` struct, `LoadRegistry`/`SaveRegistry`, CSV import/export, DOORS ReqIF/Polarion/Codebeamer/Jama XML import/export (REQ-FO-REQ001/002/003)
- `fusaops req [REQ-ID...]` — show requirements from `.fusa-reqs.json` with optional ID filter (REQ-FO-CLI052)
- `fusaops req import --format csv|doors|polarion|codebeamer|jama --file FILE` — add new entries, skip duplicates
- `fusaops req export --format csv|doors|polarion|codebeamer|jama [--output FILE]` — export to file or stdout

---

## v1.35 — `fusaops coverage` — DO-178C structural coverage ✅

**Goal:** Give FuSaOps the same DO-178C structural coverage reporting that `gofusa coverage` provides, applied to FuSaOps's own Go source.

- `coverage` package: `Parse` (Go profile reader), `Analyse` (DO-178C report: statement%, decision%, MC/DC flag, gaps), `BuildFromFile`, `Render` (text/json) (REQ-FO-COV001/002/003)
- `fusaops coverage [flags] [coverage.out]` — `--dal DAL-A|B|C|D`, `--format text|json`, `--output`, `--dir` (REQ-FO-CLI051)
- Mirrors `gofusa coverage`; enables ASIL-C DO-178C evidence from `go test -coverprofile=coverage.out ./...`

---

## v1.34 — `fusaops adapters --format json` ✅

**Goal:** Let CI scripts and tooling introspect installed adapters without parsing human-readable text.

- `fusaops adapters` gains `--format json` flag (REQ-FO-CLI050)
- Emits a JSON array: `[{"name": "gofusa", "tool": "gofusa", "language": "go", "available": true}, ...]`
- Default format remains `text`

---

## v1.33 — `fusaops sbom --format markdown` ✅

**Goal:** Let CI pipelines embed SBOM summaries in PR comments and GitHub Actions job summaries.

- `sbom.Render` gains `"markdown"` / `"md"` format support (REQ-FO-SBM011)
- GFM output: project/SBOM header, metadata, Components table (Tool/Language/Module/Packages with skipped-inline), Packages table (Name/Version/Language) with pipe-escaped names
- `fusaops sbom --format markdown [--output FILE]`

---

## v1.32 — `fusaops <standard> --format markdown` ✅

**Goal:** Complete the standards command format set so gap reports can be posted directly into PR comments and job summaries.

- `standards.Render` gains `"markdown"` / `"md"` format support (REQ-FO-STD012)
- GFM output: standard header with 🟢/🔴 badge, per-component sections (tool/language heading, counts, objective table ID/Status/Title+Clause/Evidence), skipped/nil fallbacks
- Applies to all six commands: `iso26262`, `iec61508`, `do178`, `iso21434`, `unece`, `iec62443`

---

## v1.31 — `fusaops conform --format markdown` ✅

**Goal:** Let CI pipelines post x-FuSa tool conformance summaries directly into PR comments and job summaries.

- `conform.Render` gains `"markdown"` / `"md"` format support (REQ-FO-CNF019)
- GFM output: tool header, 🟢/🔴 badge, pass/fail/skip counts, per-check table (Result/Level/Section/Check+detail) with pipe-escaping
- `fusaops conform <binary> --format markdown [--output FILE]`

---

## v1.30 — `fusaops policy --format markdown` ✅

**Goal:** Let CI pipelines post policy gate results directly into pull request comments and GitHub Actions job summaries.

- `policy.Render` gains `"markdown"` / `"md"` format support (REQ-FO-POL006)
- GFM output: policy-name header, 🟢/🔴 badge, passed/failed counts, per-rule table (Result/Rule/Message) with pipe-escaping in messages
- `fusaops policy --format markdown [--output FILE]`

---

## v1.29 — `fusaops fleet --format markdown` ✅

**Goal:** Let CI pipelines post multi-repo safety summaries directly into pull request comments and GitHub Actions job summaries.

- `fleet.Render` gains `"markdown"` / `"md"` format support (REQ-FO-FLT007)
- GFM output: project header, 🟢/🟡/🔴 badge, per-repo table (Repository/Status/Errors/Warnings/Infos/Total), TOTAL summary row, inline scan errors
- `fusaops fleet --format markdown [--output FILE]`

---

## v1.28 — `--workers` and `--timeout` CLI flags ✅

**Goal:** Let CI pipelines cap adapter concurrency and set per-adapter timeouts without editing `.fusaops.json`.

- `fusaops check`, `report`, `trace`, `sbom`, and `audit-pack` now accept `--workers N` and `--timeout DURATION` (REQ-FO-CLI049)
- Flags override the equivalent `run.workers` / `run.timeout` values from config when non-zero/non-empty
- Invalid `--timeout` values exit 2 with a descriptive error

---

## v1.27 — `fusaops <standard> --format html` ✅

**Goal:** Complete the standards command format set so gap reports are as shareable as other report types.

- `standards.Render` gains `"html"` format support (REQ-FO-STD011)
- Self-contained HTML: per-component section with tool/language header, satisfied/partial/gap counts, and colour-coded objective table; fallback for skipped/nil components; no external CSS/JS
- Applies to all six commands: `iso26262`, `iec61508`, `do178`, `iso21434`, `unece`, `iec62443`

---

## v1.26 — `fusaops conform --format html` ✅

**Goal:** Complete the conform format set so conformance reports are as shareable as other report types.

- `conform.Render` gains `"html"` format support (REQ-FO-CNF018)
- Self-contained HTML: PASS/FAIL badge, pass/fail/skip counts, per-check results table (result/level/section/name+detail), no external CSS/JS
- `fusaops conform <binary> --format html [--output conform.html]`

---

## v1.25 — `fusaops policy --format html` ✅

**Goal:** Complete the policy report format set so policy gates are as shareable as other report formats.

- `policy.Render` gains `"html"` format support (REQ-FO-POL005)
- Self-contained HTML: PASS/FAIL badge, passed/failed count summary, per-rule results table (result/ID/message), no external CSS/JS
- `fusaops policy --format html [--output policy.html]`

---

## v1.24 — `fusaops sbom --format html` ✅

**Goal:** Complete the SBOM format set so the merged bill of materials is as shareable as the findings report.

- `sbom.Render` gains `"html"` format support (REQ-FO-SBM010)
- Self-contained HTML: component summary table + full de-duplicated package table, no external CSS/JS
- `fusaops sbom --format html [--output sbom.html]`
- Matches the design language of the single-project HTML report and trace HTML renderer

---

## v1.23 — `fusaops trace --format markdown` ✅

**Goal:** Let teams embed the cross-language traceability matrix in GitHub wikis, PR descriptions, and documentation sites without converting from HTML.

- `trace.Render` gains `"markdown"` (and `"md"` alias) format support (REQ-FO-TRC018)
- GFM output: status badge header, per-component summary table (requirements/traced/tested/sec-tested/qualification), TOTAL row, and collapsible `<details>` gap lists
- `fusaops trace --format markdown [--output REPORT.md]`
- Mirrors rust-FuSa's `--format md` gap-report output style

---

## v1.22 — Suppress import ✅

**Goal:** Let teams acknowledge all existing findings in one command when onboarding FuSaOps.

- `fusaops suppress import --from check.json [--file suppress.json] [--reason TEXT] [--expires DATE]` (REQ-FO-CLI048)
- Reads fingerprints from a `fusaops check --format json` report; de-duplicates against existing entries
- Prints `Imported N findings (M new, K already present).` (REQ-FO-SUP009)
- `--from` required; exits 2 if missing

---

## v1.21 — Fleet HTML report ✅

**Goal:** Complete the fleet report format set to match single-project report parity.

- `fleet.Render` gains `"html"` format support (REQ-FO-FLT005)
- Self-contained HTML: overall badge, per-repo status table, summary footer, colour-coded severity cells
- `fusaops fleet --format html [--output fleet.html]`
- No external CSS/JS — compatible with single-project HTML report design language

---

## v1.20 — Severity filter ✅

**Goal:** Let CI gates and report consumers focus on only the severity levels they care about.

- `Options.MinSeverity` — orchestrator filters findings below the threshold before storing them (REQ-FO-ORC012)
- `fusaops check --min-severity ERROR|WARNING|INFO` (REQ-FO-CLI047)
- `fusaops report --min-severity ERROR|WARNING|INFO` (REQ-FO-CLI047)
- Invalid value exits 2; empty value (default) disables filtering

---

## v1.19 — History CLI ✅

**Goal:** Make check-run history accessible from the CLI without a running dashboard server.

- `fusaops history list [--dir|--file] [--format text|json] [--limit N]` — lists snapshots from `.fusaops-history.jsonl` newest-first (REQ-FO-CLI045)
- `fusaops history prune [--dir|--file] [--keep N]` — trims old entries to `--keep` most-recent (REQ-FO-CLI046)
- `history.Prune(dir, keep)` — exported function for programmatic access (REQ-FO-HST003)
- Spec snapshot: rust-FuSa v0.2.8 (gap-report `--format md` support)

---

## v1.18 — Config validation CLI ✅

**Goal:** Surface config errors early and make the effective config inspectable from the command line.

- `fusaops config validate [--dir|--file]` — loads and validates `.fusaops.json`; CI-friendly pre-flight check (REQ-FO-CLI043)
- `fusaops config show [--dir|--file]` — prints effective config as formatted JSON (REQ-FO-CLI044)
- Both commands support `--file PATH` to override the default `.fusaops.json` search location

---

## v1.17 — Diff HTML/Markdown & `check --save-baseline` ✅

**Goal:** Complete the diff workflow loop and add richer diff report output formats.

- `diff.Render` gains `"html"` and `"markdown"`/`"md"` format support (REQ-FO-DIF006)
- HTML: self-contained dashboard with added/removed tables and gate badge
- Markdown: GFM shield badge, summary table, per-section finding tables
- `fusaops check --save-baseline PATH` — save findings as diff baseline in one command (REQ-FO-CLI042)

---

## v1.16 — Finding fingerprint hints & quick-suppress scaffolding ✅

**Goal:** Close the loop between seeing a finding and suppressing it — show the fingerprint and a ready-to-run scaffold in all rendered formats.

- Orchestrator auto-computes `sha256:` fingerprint for any finding without one (REQ-FO-ORC011)
- `RenderOptions.ShowFingerprints` added; `--show-fingerprints` flag on `check` and `report` (REQ-FO-CLI041)
- Text: `fingerprint: <fp>` + `$ fusaops suppress add --fingerprint <fp> --reason ""` per finding
- HTML: monospace `.fp-chip` chip with tooltip in message cell
- Markdown: `Fingerprint` column added to the per-component GFM table

---

## v1.15 — Report annotation & inline suppression hints ✅

**Goal:** Surface suppression opportunities and explain suppressed findings in rendered reports.

- `Component.SuppressedFindings` stores suppressed findings per component for audit traceability
- `AggregateReport.SuppressedComponents` tracks which components had suppressions applied
- `RenderWithOptions(ShowSuppressed bool)` — text renderer shows `[SUPPRESSED]` prefix or count hint; HTML shows collapsible `<details>` per component; Markdown uses `<details open>` toggle; JSON always serialises suppressed findings
- `fusaops check --show-suppressed` / `fusaops report --show-suppressed` expands suppressed findings in output

---

## Adding a language adapter

1. Implement `adapter.Adapter` (Name, Language, Tool, Detect, Available, Check).
2. For tools emitting the standard `check --format json` schema, embed
   `cmdAdapter` and supply the binary name plus source extensions.
3. Register it in an `init()` with `Default.MustRegister`.
4. Add detection extensions to `scan.langExtensions`.
