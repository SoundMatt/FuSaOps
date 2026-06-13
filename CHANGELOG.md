# Changelog

All notable changes to FuSaOps are documented here.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.0.0/)
and the project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.23.0] — 2026-06-13

### `fusaops trace --format markdown` — GFM traceability matrix

- **`trace.Render(w, a, "markdown")` / `"md"` alias** — new format; produces a GitHub-Flavoured Markdown document with a status badge header (`🟢 **PASS**` / `🟡 **GAP**`), a per-component summary table (requirements/traced%/tested%/sec-tested%/qualification), a TOTAL row, and collapsible `<details>` gap lists. (REQ-FO-TRC018)
- **`fusaops trace --format markdown [--output REPORT.md]`** — expose the new format from the CLI.
- Mirrors the style of rust-FuSa's `--format md` gap-report.

### Safety
- Requirements registry at 247 requirements (added REQ-FO-TRC018)
- 3 new tests; 247/247 requirements traced+tested; combined coverage 81.3%

## [1.22.0] — 2026-06-13

### `fusaops suppress import` — bulk suppression from check report

- **`fusaops suppress import --from PATH [--file SUPPRESS_FILE] [--reason TEXT] [--expires DATE]`** — reads a `fusaops check --format json` report, extracts all finding fingerprints, and appends new entries to `.fusaops-suppress.json`. Fingerprints already present in the suppress file are skipped (no duplicates). Prints `Imported N findings (M new, K already present).` and exits 0. (REQ-FO-SUP009, REQ-FO-CLI048)
- `--from` is required; exits 2 if missing.
- `--reason` (default `"imported"`) and `--expires` (optional, `YYYY-MM-DD`) apply to all new entries.
- Typical workflow: `fusaops check --format json --output check.json && fusaops suppress import --from check.json --reason "acknowledged on YYYY-MM-DD"`

### Safety
- Requirements registry at 246 requirements (added REQ-FO-SUP009, REQ-FO-CLI048)
- 4 new tests; 246/246 requirements traced+tested; combined coverage 81.1%

## [1.21.0] — 2026-06-13

### `fusaops fleet --format html` — Fleet HTML report

- **`fleet.Render(w, fr, "html")`** — new format support; produces a self-contained HTML page with an overall PASS/WARN/FAIL badge, a per-repo status table (name, status badge, errors, warnings, infos, total) with colour-coded severity cells, and a summary footer. No external CSS or JS. (REQ-FO-FLT005)
- **`fusaops fleet --format html [--output fleet.html]`** — write a fleet HTML report from the CLI. (REQ-FO-FLT005)
- `fleet.Render` error message updated from `"want text or json"` → `"want text, json, or html"`.

### Safety
- Requirements registry at 244 requirements (added REQ-FO-FLT005)
- 4 new tests (3 fleet package + 1 cmd); 244/244 requirements traced+tested; combined coverage 81.2%

## [1.20.0] — 2026-06-13

### `--min-severity` severity filter

- **`Options.MinSeverity fusaops.Severity`** — new field on `orchestrator.Options`. When set, findings whose `Severity.Rank()` is below `MinSeverity.Rank()` are filtered out after fingerprint enrichment and before suppression. Empty string (the zero value) disables filtering. (REQ-FO-ORC012)
- **`fusaops check --min-severity INFO|WARNING|ERROR`** and **`fusaops report --min-severity INFO|WARNING|ERROR`** — new flag forwarding the value to `Options.MinSeverity`. An invalid value (not one of the three) exits 2 with a descriptive error. (REQ-FO-CLI047)
- Common use: `fusaops check --min-severity ERROR` to suppress INFO and WARNING findings in a blocking CI gate.

### Safety
- Requirements registry at 243 requirements (added REQ-FO-ORC012, REQ-FO-CLI047)
- 6 new tests (3 orchestrator, 3 cmd); 243/243 requirements traced+tested; combined coverage 81.1%

## [1.19.0] — 2026-06-13

### `fusaops history list|prune` subcommands & `history.Prune`

- **`history.Prune(dir string, keep int) (int, error)`** — new exported function in the `history` package; retains the `keep` most-recent snapshots, writes the file back, and returns the count removed. `keep <= 0` defaults to `MaxSnapshots`. A missing file returns `0, nil`. (REQ-FO-HST003)
- **`fusaops history list [--dir DIR] [--file PATH] [--format text|json] [--limit N]`** — lists check-run snapshots from `.fusaops-history.jsonl` newest-first in a human-readable table (text) or JSON array. Missing file prints `No history entries found.` and exits 0. (REQ-FO-CLI045)
- **`fusaops history prune [--dir DIR] [--file PATH] [--keep N]`** — removes old entries keeping at most `--keep` (default 100) most-recent snapshots; prints `Pruned N entries, M remaining.`; exits 0. (REQ-FO-CLI046)
- Both commands accept `--file PATH` to override the default `.fusaops-history.jsonl` search location.
- x-FuSa spec snapshot updated: rust-FuSa v0.2.6 → **v0.2.8** (added `--format md` for all 8 standards gap-report commands; 100% requirement coverage).

### Safety
- Requirements registry at 241 requirements (added REQ-FO-HST003, REQ-FO-CLI045, REQ-FO-CLI046)
- 11 new tests (8 cmd-level + 3 history package); 241/241 requirements traced+tested; combined coverage 81.1%

## [1.18.0] — 2026-06-13

### `fusaops config validate|show` subcommands

- **`fusaops config validate [--dir DIR] [--file PATH]`** — loads and validates `.fusaops.json`; on success prints `OK <path>` with a human-readable summary of `version`, `project`, `standard`, `adapters`, and `format`; exits 0. On a missing file exits 1 with `no config file`; on a malformed/invalid file exits 1 with the validation error. (REQ-FO-CLI043)
- **`fusaops config show [--dir DIR] [--file PATH]`** — loads and validates the config file then prints the effective configuration as indented JSON to stdout; exits 0. Errors to stderr and exits 1. (REQ-FO-CLI044)
- Both sub-subcommands accept `--dir` (default `.`) and `--file` (overrides `--dir`) to locate the config file.
- `fusaops config` with no subcommand or an unknown subcommand exits 2 with a descriptive error.
- `fusaops config validate` is designed for CI pre-flight — a `step: run: fusaops config validate` catches misconfigured `.fusaops.json` before the scan runs.

### Safety
- Requirements registry at 238 requirements (added REQ-FO-CLI043, REQ-FO-CLI044)
- 8 new tests; 238/238 requirements traced+tested; combined coverage 81.1%

## [1.17.0] — 2026-06-13

### Diff HTML/Markdown renderers & `fusaops check --save-baseline`

- **`diff.Render` now supports `"html"` and `"markdown"`/`"md"` formats** — the HTML renderer produces a self-contained dashboard with added/removed finding tables and a gate verdict badge; the Markdown renderer produces a GFM report with shield badge, summary table, and per-section finding tables. `fusaops diff --format html/markdown` works out of the box. (REQ-FO-DIF006)
- **`fusaops check --save-baseline PATH`** — after a successful scan, saves the current findings to PATH as a diff baseline (calls `diff.SaveBaseline`). Prints `Saved baseline to PATH (N findings)`. Allows CI pipelines to roll the baseline forward from one command instead of a separate `diff --update-baseline` step. (REQ-FO-CLI042)

### Safety
- Requirements registry at 236 requirements (added REQ-FO-DIF006, REQ-FO-CLI042)
- 9 new tests; 236/236 requirements traced+tested; combined coverage 81.0%

## [1.16.0] — 2026-06-13

### Finding fingerprint hints & quick-suppress scaffolding

- **Auto-compute fingerprints in orchestrator** — if an x-FuSa tool does not include a `fingerprint` field in a finding, the orchestrator now computes one via `fusaops.ComputeFingerprint` (§4.2 sha256 algorithm) before storing the finding. This ensures fingerprints are always available downstream for suppression matching and report display. (REQ-FO-ORC011)
- **`RenderOptions.ShowFingerprints bool`** — new field on the options struct passed to `RenderWithOptions`. (REQ-FO-RPT019)
- **Text renderer** — when `ShowFingerprints: true`, each finding is followed by two lines: `fingerprint: <fp>` and `$ fusaops suppress add --fingerprint <fp> --reason ""`. (REQ-FO-RPT019)
- **HTML renderer** — when `ShowFingerprints: true`, each finding in the findings table shows a monospace `.fp-chip` span below the message, with a tooltip holding the full `fusaops suppress add` command. (REQ-FO-RPT019)
- **Markdown renderer** — when `ShowFingerprints: true`, a `Fingerprint` column is added to the per-component GFM table. (REQ-FO-RPT019)
- **`fusaops check --show-fingerprints`** and **`fusaops report --show-fingerprints`** — new flag wiring `ShowFingerprints: true`. (REQ-FO-CLI041)

### Safety
- Requirements registry at 234 requirements (added REQ-FO-ORC011, REQ-FO-RPT019, REQ-FO-CLI041)
- 11 new tests; 234/234 requirements traced+tested; combined coverage 80.9%

## [1.15.0] — 2026-06-13

### Report annotation & inline suppression hints

- **`Component.SuppressedFindings []fusaops.Finding`** — the orchestrator now stores suppressed findings on each component rather than discarding them, preserving full auditability of what was filtered. (REQ-FO-RPT017)
- **`AggregateReport.SuppressedComponents []string`** — lists the tool names of every component that had at least one finding suppressed. (REQ-FO-RPT018)
- **`report.RenderWithOptions(w, r, format, RenderOptions{ShowSuppressed bool})`** — new public function; `Render` delegates to it with `ShowSuppressed: false`. (REQ-FO-RPT017)
- **`report.RenderToFileWithOptions`** — file-writing companion to `RenderWithOptions`. (REQ-FO-RPT017)
- **Text renderer** — when `ShowSuppressed: true`, suppressed findings are printed after active findings with a `[SUPPRESSED]` prefix; when false, a per-component hint line `(N suppressed — use --show-suppressed to view)` is shown. (REQ-FO-RPT017)
- **HTML renderer** — each component with suppressions gets a collapsible `<details>` section below the main table; `ShowSuppressed: true` expands it by default (`open` attribute). The per-component card also shows the suppressed count. (REQ-FO-RPT017)
- **Markdown renderer** — suppressed findings appear in a `<details>` block per component (collapsed by default, `open` when `ShowSuppressed: true`). (REQ-FO-RPT017)
- **JSON renderer** — `suppressedFindings` field serialised from the struct automatically; no format change needed. (REQ-FO-RPT017)
- **`fusaops check --show-suppressed`** and **`fusaops report --show-suppressed`** — new flag that passes `ShowSuppressed: true` to the renderer. (REQ-FO-CLI040)

### Safety
- Requirements registry at 231 requirements (added REQ-FO-RPT017, REQ-FO-RPT018, REQ-FO-CLI040)
- 14 new tests; 231/231 requirements traced+tested; combined coverage 80.8%

## [1.14.0] — 2026-06-13

### Suppression management CLI (`fusaops suppress`)

- **`fusaops suppress add --fingerprint sha256:<hex> --reason text [--expires YYYY-MM-DD] [--file path]`** — append a new suppression entry to `.fusaops-suppress.json` (creates the file if absent). (REQ-FO-SUP005)
- **`fusaops suppress list [--file path] [--format text|json]`** — print all suppression entries with active/expired status, fingerprint, reason, and expiry. (REQ-FO-SUP006)
- **`fusaops suppress prune [--file path]`** — remove expired suppression entries and print the count removed. (REQ-FO-SUP007)
- **`fusaops suppress verify [--file path] [--dir path]`** — run the orchestrator, compare suppression fingerprints against current findings, and exit 1 with a per-entry report for any stale (unmatched) suppressions. (REQ-FO-SUP008)
- `suppression.SaveConfig(path, cfg)` and `suppression.Prune(cfg, now)` added to the suppression package.

### Safety
- Requirements registry at 228 requirements (added REQ-FO-SUP005-008, REQ-FO-CLI039)
- 18 new tests; 228/228 requirements traced+tested; combined coverage ≥80%

## [1.13.0] — 2026-06-13

### Per-project suppression & config in multi-project mode

- **Per-project `"suppression"` field** in `projects.json` — set a path to a `.fusaops-suppress.json` file and it is applied to that project's scan independently of other projects. (REQ-FO-MPJ005)
- **Auto-load `.fusaops.json`** from each project's directory — `MultiServer` now mirrors the single-project behaviour: project name override, adapter filter, and future settings apply per-project without an extra flag. (REQ-FO-MPJ006)
- **Startup directory validation** — `fusaops serve --projects projects.json` now validates all project directories before binding a port and exits 1 with a descriptive error for each missing path. (REQ-FO-MPJ007)
- **`/api/v1/diff?project=name`** — diff a single named project in fleet mode instead of the merged fleet. Unknown project names return 503. (REQ-FO-SRV009)

### Safety
- Requirements registry at 223 requirements (added REQ-FO-MPJ005-007, REQ-FO-SRV009)
- 8 new tests; 223/223 requirements traced+tested; combined coverage ≥80%

## [1.12.0] — 2026-06-13

### Diff gate via REST API

- **`GET /api/v1/diff?baseline=PATH[&strict=true][&format=json|text]`** — compares the cached report against a baseline JSON file and returns the delta. When `strict=true` and new ERROR findings exist, the response is `409 Conflict`. Both `Server` and `MultiServer` supported. (REQ-FO-SRV007)
- **`POST /api/v1/baseline`** — saves the current cached findings to the configured baseline file and returns `{"saved":"path","findings":N}`. (REQ-FO-SRV008)
- **`fusaops serve --baseline path`** — sets a default baseline file for both endpoints without requiring a per-request query parameter. (REQ-FO-CLI038)
- Multi-project mode merges all project findings before diffing and saving.

### Safety
- Requirements registry at 219 requirements (added REQ-FO-SRV007, REQ-FO-SRV008, REQ-FO-CLI038)
- 14 new tests; 219/219 requirements traced+tested; combined coverage ≥80%

## [1.11.0] — 2026-06-13

### Server export endpoint

- **`GET /api/v1/export?format=FORMAT`** on both `Server` (single-project) and `MultiServer` (multi-project) — returns the cached aggregate report rendered in the requested format as a file download. (REQ-FO-SRV006)
- Supported formats: `json` (default), `text`, `html`, `sarif`, `junit`, `csv`, `markdown`/`md`.
- Response includes `Content-Type` and `Content-Disposition: attachment; filename="fusaops-report.<ext>"` so browsers and CI tools download the file directly.
- Multi-project mode merges all project components into a single fleet-level `AggregateReport` before rendering.

### Safety
- Requirements registry at 216 requirements (added REQ-FO-SRV006)
- 5 new tests; 216/216 requirements traced+tested; combined coverage ≥80%

## [1.10.0] — 2026-06-13

### HTML dashboard text search filter
- **Live search box** in the findings table — filters rows by matching text against rule ID, message, and category simultaneously as you type. (REQ-FO-RPT016)
- The severity filter buttons and the text search compose with AND logic; a result counter badge appears when the filter is active (e.g. "12 / 47 shown").

### `fusaops check --output` file flag
- `fusaops check --output path` writes the report to a file instead of stdout, with a format-specific extension (e.g. `--format junit --output results.xml`). A confirmation message is printed to stdout; the exit code still reflects the gate result. (REQ-FO-CLI037)

### Safety
- Requirements registry at 215 requirements (added REQ-FO-RPT016, REQ-FO-CLI037)
- 2 new tests; 215/215 requirements traced+tested; combined coverage ≥80%

## [1.9.0] — 2026-06-13

### Markdown report format

- **`--format markdown`** (alias `--format md`) on `fusaops check` and `fusaops report` —
  produces GitHub-Flavored Markdown suitable for PR comments, wiki pages, and Markdown-capable
  documents. (REQ-FO-RPT015, REQ-FO-CLI036)
- Output includes: a status badge (shields.io-style), summary table with emoji severity icons,
  per-component sections with GFM finding tables, skip reasons for unavailable tools, and
  "no findings" notes for clean components.
- Pipe characters in messages and rule IDs are escaped so they don't break GFM tables.

### Safety
- Requirements registry at 213 requirements (added REQ-FO-RPT015, REQ-FO-CLI036)
- 8 new tests (7 renderer, 1 CLI); 213/213 requirements traced+tested; combined coverage ≥80%

## [1.8.0] — 2026-06-13

### CSV report format

- **`--format csv`** on `fusaops check` and `fusaops report` — produces RFC 4180 CSV with
  columns: `language, tool, ruleId, severity, message, file, line, column, category, fingerprint`.
  One row per finding across all components; skipped/zero-finding components emit no data rows.
  (REQ-FO-RPT014, REQ-FO-CLI035)
- Useful for importing findings into Excel, Google Sheets, or any spreadsheet auditing workflow.
- Zero external dependencies — `encoding/csv` from stdlib.

### Safety
- Requirements registry at 211 requirements (added REQ-FO-RPT014, REQ-FO-CLI035)
- 5 new tests (4 renderer, 1 CLI); 211/211 requirements traced+tested; combined coverage ≥80%

## [1.7.0] — 2026-06-13

### JUnit XML report format

- **`--format junit`** on `fusaops check` and `fusaops report` — produces JUnit XML
  (`<?xml ...?><testsuites ...>`) for CI systems that natively consume JUnit test results
  (Jenkins, Azure DevOps, CircleCI, GitLab, GitHub Actions test summary). (REQ-FO-RPT013, REQ-FO-CLI034)
- Each component (language × tool) maps to a `<testsuite name="lang/tool">`.
- Each finding maps to a `<testcase>`; ERROR and WARNING findings carry a `<failure type="ERROR|WARNING">` element with file:line in the body.
- Components with zero findings emit a synthetic `<testcase name="(no findings)"/>` (passed).
- Skipped/unavailable components emit `<testcase name="(skipped)"><skipped/></testcase>`.
- Zero external dependencies — `encoding/xml` from stdlib.

### Safety
- Requirements registry at 209 requirements (added REQ-FO-RPT013, REQ-FO-CLI034)
- 8 new tests (7 renderer, 1 CLI); 207/207 requirements traced+tested; combined coverage ≥80%

## [1.6.0] — 2026-06-13

### Finding suppression

- **`suppression` package** — `Config` / `Suppression` types; `LoadConfig(path)` (empty path = no-op); `Filter(findings, cfg, now)` returns (kept, suppressed) slices. Suppressions match on spec §4.2 fingerprint. An optional `expires` date (YYYY-MM-DD) stops the suppression after that day. (REQ-FO-SUP001–003)
- **Orchestrator integration** — `Options.SuppressFile` wires suppression into `Runner.Run()`. Suppressed findings are removed from component and global summaries; `AggregateReport.Suppressed` holds the count. A missing or malformed file exits with an error. (REQ-FO-SUP004)
- **CLI** — `fusaops check --suppress-file .fusaops-suppress.json` and `fusaops report --suppress-file ...` forward the file path to the orchestrator. Text renderer appends `(N suppressed)` to the TOTAL line when suppressions are active. (REQ-FO-CLI033)

### Safety
- Requirements registry at 208 requirements (added REQ-FO-SUP001–004, REQ-FO-CLI033)
- 10 new tests (7 suppression, 2 orchestrator, 1 CLI); combined coverage 80.8%

## [1.5.0] — 2026-06-13

### OpenMetrics / Prometheus endpoint

- **`GET /metrics`** — both `Server` and `MultiServer` now expose an OpenMetrics text endpoint (Content-Type: `text/plain; version=0.0.4`). Metrics emitted:
  - `fusaops_findings_total{severity="error"|"warning"|"info"}` — finding counts as gauges
  - `fusaops_status` — 1=PASS, 2=WARN, 3=FAIL, 0=pending/error
- **Per-project labels** — in multi-project mode, all series carry `project="<name>"` labels so a single Prometheus scrape target covers the entire monorepo.
- No external dependencies — full OpenMetrics text format in stdlib. (REQ-FO-MTR001, REQ-FO-MTR002)

### Safety
- Requirements registry at 202 requirements (added REQ-FO-MTR001–002)
- 3 new tests; combined coverage 80.4%

## [1.4.0] — 2026-06-13

### Scheduled scans

- **`WithRefreshInterval(d time.Duration)`** — `Server` and `MultiServer` accept a background rescan interval. A goroutine runs `compute()` at every tick after the initial startup scan. Zero or negative disables scheduling. (REQ-FO-SCHD001)
- **`fusaops serve --refresh-interval 5m`** — CLI flag wiring for both single-project and multi-project modes. A non-positive or unparseable value exits 1. (REQ-FO-CLI032)

### Safety
- Requirements registry at 200 requirements (added REQ-FO-SCHD001, REQ-FO-CLI032)
- 4 new tests (2 CLI, 2 server); combined coverage maintained ≥80%

## [1.3.0] — 2026-06-13

### Badge service & webhooks

- **SVG status badge** — `GET /badge/status.svg` returns a shields.io-style flat SVG badge: label `fusaops`, message `pass`/`warn`/`fail`/`pending`/`error`, colored green/yellow/red/gray respectively. `Content-Type: image/svg+xml` with `Cache-Control: no-cache`. (REQ-FO-BADGE001)
- **Per-project badge** — `MultiServer` exposes `GET /badge/{name}/status.svg` for each configured project, labeled with the project name; `GET /badge/status.svg` shows the aggregate status across all projects. (REQ-FO-BADGE002)
- **Webhook notifications** — `fusaops serve --webhook url` POSTs `{"status":"…","prev":"…","errors":N}` whenever the aggregate status transitions between runs. The first compute never fires (no prior state). Retries once after 2 seconds on failure. (REQ-FO-HOOK001/002, REQ-FO-CLI031)

### Safety
- Requirements registry at 198 requirements (added REQ-FO-BADGE001–002, REQ-FO-HOOK001–002, REQ-FO-CLI031)
- 10 new tests in `server` package; combined coverage maintained ≥80%

## [1.2.0] — 2026-06-13

### Multi-project dashboard

- **`MultiServer`** — new type in `server` package; holds one `projectEntry` per project; `compute(ctx)` scans all in parallel (one goroutine per project).
- **`fusaops serve --projects projects.json`** — switches to multi-project mode. Projects config format: `{"projects":[{"name":"…","dir":"…","adapter":"…"}]}`. Unknown/missing file exits 1.
- **Overview page `/`** — HTML status grid: one card per project with PASS/WARN/FAIL badge, error/warning counts, and a link to the detail page.
- **`/api/projects`** — JSON array of all project statuses (`name`, `dir`, `status`, `total`, `errors`, `warnings`, optional `error`).
- **Per-project detail `/p/{name}`** — HTML findings table (rule, severity, language, file, message). Unknown names return 404.
- **Auth + audit compose** — `--auth`, `--auth-ro`, and `--audit-log` flags apply equally to MultiServer; role-gating and audit logging work identically.

### Safety
- Requirements registry at 193 requirements (added REQ-FO-MPJ001–004, REQ-FO-CLI030)
- 9 new tests in `server` package; 1 new CLI test; combined coverage 80.1%

## [1.1.0] — 2026-06-13

### Role-based access & audit log

- **Read-only credentials** — `fusaops serve --auth-ro viewer:pass` sets a second credential pair with read-only access via `Server.WithAuthRO(user, pass)`. Users authenticating with ro credentials can view all dashboards and API endpoints but receive `403 Forbidden` on mutating routes (`/refresh`). Full rw credentials set via `--auth` retain unrestricted access (REQ-FO-RBAC001/002).
- **Request audit log** — `fusaops serve --audit-log dir` enables access logging via `Server.WithAuditLog(dir)`. Every authenticated request is appended to `.fusaops-audit.jsonl` as a JSON record: `{timestamp, method, path, user, status}`. File is created with mode `0600`; never truncated (REQ-FO-AUDIT001/002).

### Safety
- Requirements registry at 188 requirements (added REQ-FO-RBAC001–002, REQ-FO-AUDIT001–002, REQ-FO-CLI028–029)
- 7 new tests in `server` package; 1 new CLI test; combined coverage 81.2%

## [1.0.0] — 2026-06-13

### Enterprise readiness

- **HTTP Basic Auth** — `fusaops serve --auth user:pass` enables authentication on all routes (dashboard, API, history). Unauthenticated requests receive `401 Unauthorized` with a `WWW-Authenticate: Basic realm="fusaops"` challenge. Implemented via `Server.WithAuth(user, pass string)` (REQ-FO-AUTH001/002).
- **TLS / HTTPS** — `fusaops serve --tls-cert cert.pem --tls-key key.pem` switches the server to HTTPS using `Server.ListenAndServeTLS` (TLS 1.2+); `--tls-key` is required when `--tls-cert` is set (REQ-FO-TLS001).
- **Fleet web dashboard** — `fusaops serve --fleet fleet.json` adds a `/fleet` HTML status page and `/api/fleet` JSON endpoint to the dashboard. The fleet scan runs in parallel (reusing the existing `fleet.Run()`) on every `compute()` call. Shows overall PASS/WARN/FAIL badge, per-repo status table with error/warning counts, and links back to the main dashboard (REQ-FO-FLT005/006).

### Docs
- README: added fleet view, policy engine, and REST API sections with usage examples; updated action reference to `@v0.9.0`; c-FuSa version corrected to v0.5.16; requirements count updated to 183
- CHANGELOG: footer links corrected for all releases v0.1.0–v0.9.0
- ROADMAP: v1.0 marked ✅; added v1.1/v1.2 placeholders

### Safety
- Requirements registry at 183 requirements (added REQ-FO-AUTH001–002, REQ-FO-FLT005–006, REQ-FO-TLS001, REQ-FO-CLI025–027)
- 13 new tests in `server` package; 2 new CLI tests; combined coverage 81.3%

## [0.9.0] — 2026-06-13

### Policy engine — org-wide safety rules

- **`policy` package** — evaluates rules over an `AggregateReport`
  - `Policy` + `Rule` JSON config: `maxFindings`, `maxErrors`, `maxWarnings`, `requireStatus` (PASS/WARN), optional `language` and `tool` scope filters
  - `Evaluate(policy, report)` → `PolicyReport` with per-rule `RuleResult` (passed, message)
  - `PolicyReport.Status()` → PASS/FAIL; `HasFailures()`
  - `Render(text|json)` + `RenderToFile`
- **`fusaops policy --policy policy.json [--dir dir] [--format text|json] [--output file]`** — evaluates the policy after running the orchestrator; exits 1 on any rule failure

### Example policy config

```json
{
  "name": "ci-gate",
  "rules": [
    { "id": "no-errors",         "requireStatus": "WARN" },
    { "id": "go-strict",         "language": "go", "requireStatus": "PASS" },
    { "id": "cpp-error-budget",  "language": "cpp", "maxErrors": 5 }
  ]
}
```

### Safety
- Requirements registry at 175 requirements (added REQ-FO-POL001–004, REQ-FO-CLI024)
- 12 new tests in `policy` package; 2 new CLI tests; combined coverage 81.4%

## [0.8.0] — 2026-06-13

### Fleet view — multi-repo scanning

- **`fleet` package** — multi-repository check orchestration
  - `Config` + `Repo` — JSON fleet config (`{"project":"p","repos":[{"name":"svc","dir":"/path","adapter":"gofusa"}]}`)
  - `FleetReport` + `RepoResult` — per-repo status (PASS/WARN/FAIL/ERROR), counts, optional scan error
  - `FleetReport.Status()` — FAIL if any repo fails; WARN if any warns; PASS if all clean
  - `Run(ctx, cfg, runner)` — parallel check across all repos (one goroutine per repo)
  - `Render(w, fr, "text"|"json")` + `RenderToFile` — columnar text table or JSON output
- **`fusaops fleet --config fleet.json [--format text|json] [--output file] [--strict]`** — first-class CLI command; exit 1 on FAIL (or on WARN under `--strict`)
- x-FuSa spec v1.10.8: c-FuSa v0.5.16 exit-code fixes noted; version snapshot updated

### Safety
- Requirements registry at 170 requirements (added REQ-FO-FLT001–004, REQ-FO-CLI023)
- 10 new tests in `fleet` package; 2 new CLI tests; combined coverage 81.3%

## [0.7.0] — 2026-06-13

### REST API v1 + dashboard nav

- **`GET /api/v1/status`** — lightweight JSON poll endpoint: `{"status":"PASS","errors":0,"warnings":2,"total":2}`; returns `PENDING` (503) when no report available yet
- **`GET /api/v1/findings`** — filtered findings array; supports `?severity=ERROR`, `?language=go`, `?tool=gofusa` (combinable); always returns a JSON array (never null)
- **`GET /api/v1/report`** — versioned alias for `/api/report` (full aggregate JSON)
- **`GET /api/v1/history`** — versioned alias for `/api/history`
- **Dashboard nav** — main HTML dashboard now has _History_, _JSON_, and _Refresh_ links in the header for quick navigation in `fusaops serve`
- x-FuSa spec v1.10.7: cpp-FuSa v0.12.5 `location.file` project-relative fix; c-FuSa v0.5.14 `hara show` (non-spec feature); version snapshot updated

### Safety
- Requirements registry at 165 requirements (added REQ-FO-API001, REQ-FO-API002)
- 6 new tests in `server` package

## [0.6.0] — 2026-06-13

### Run-history trend

- **`history` package** — persists check-run outcomes to `.fusaops-history.jsonl` (JSONL, one object per line) in the project root
  - `Snapshot` records: run time, PASS/FAIL status, total/error/warning/info counts, per-language summary
  - `FromReport(rep)` builds a Snapshot from an `AggregateReport`
  - `Store(dir, snap)` appends and trims to `MaxSnapshots = 100` entries
  - `Load(dir, limit)` returns the most-recent entries oldest-first; missing file returns empty slice
- **`server.WithHistoryDir(dir)`** — opt-in fluent method; when set, each successful `/refresh` or startup compute appends a Snapshot
- **`/api/history`** — JSON array of all stored Snapshots
- **`/history`** — self-contained HTML trend page: PASS/FAIL badges, per-severity counts, proportional severity bar, per-language breakdown; links back to the main dashboard
- **`fusaops serve`** — now calls `WithHistoryDir(root)` automatically; history accumulates with every dashboard refresh

### Safety
- Requirements registry at 163 requirements (added REQ-FO-HST001 – HST004)
- 7 new tests in `history` package; 5 new tests in `server` package

## [0.5.1] — 2026-06-13

### GitHub Action

- **`.github/actions/fusaops/action.yml`** — reusable composite action wrapping the `ghcr.io/soundmatt/fusaops` Docker image; zero-install CI integration for any GitHub-hosted repository
  - `command` input (default: `check`): any fusaops subcommand
  - `args` input: extra flags appended after the command (e.g. `--strict`)
  - `image` input: pin to a digest or tag for reproducibility
  - `upload-report` input: when `"true"`, also runs `fusaops report --format html` and uploads the output as a `fusaops-report` workflow artifact
- **`.github/fusaops-example.yml`** updated to use `uses: SoundMatt/FuSaOps/.github/actions/fusaops@v0.5.1` with four patterns: minimal check, strict+report, trace gate, audit-pack archive
- **README.md** CI integration section updated to lead with the action; direct-install path retained as fallback

## [0.5.0] — 2026-06-13

### All-language support

- **rust-FuSa adapter** (`rsfusa`, `LangRust`): detects `.rs` files; runs `rsfusa check --format json`; full `Tracer`/`Qualifier`/`SBOMer`/`Packer` capability interfaces via `cmdAdapter`
- **py-FuSa adapter** (`pyfusa`, `LangPython`): detects `.py` files; same generic path
- **java-FuSa adapter** (`jfusa`, `LangJava`): detects `.java` files; same generic path
- All six adapters registered and active: go-FuSa · c-FuSa · cpp-FuSa · rust-FuSa · py-FuSa · java-FuSa
- **`fusaops conform`** extended to all six languages: `langFromBinary` maps `rsfusa`/`pyfusa`/`jfusa`; `writeSourceFiles` scaffolds `src/main.rs`+`Cargo.toml`, `main.py`, `Main.java` with `//fusa:req`/`#fusa:req` annotations
- Package comment updated from spec v1.8 → v1.10
- x-FuSa spec v1.10.6: all 6 tools audited against spec §11; c-FuSa fully conformant as of v0.5.10

### Safety
- Requirements registry at 159 requirements, all traced and tested
- Spec §11 conformance table updated to v1.10.6 (c-FuSa v0.5.10 · cpp-FuSa v0.12.4 · java-FuSa v0.2.0)

## [0.4.0] — 2026-06-10

### Monorepo & component model

- Per-directory component pinning in `.fusaops.json` (`scan.components[].adapter`, `scan.components[].timeout`, `scan.components[].path`)
- Parallel adapter execution with per-component and global timeouts (`run.workers`, `run.timeout`, `Options.Timeout`, `Options.Workers`)
- Baseline + diff gating (`fusaops diff --baseline <file> --strict`): fingerprint-matched across all tools; exit 1 on new errors; `--strict` also gates on new warnings
- `Finding.Category` and `Finding.Fingerprint` fields; `ComputeFingerprint` per spec §4.2 (sha256 over normalised `ruleId:file:message`)
- x-FuSa spec promoted to v1.9: `category`, `fingerprint`, `remediation`, `capabilities` → MUST; `fusaops conform` updated to check all four

### Safety
- Requirements registry at 146 requirements (added DIFF, ORC, CNF, ADP requirements)
- `fusaops conform` conformance gate per spec §16 step 7; 37 checks covering version, init, check, trace, qualify, release, audit-pack, capabilities

## [0.3.0] — 2026-06-10

### Standards roll-up & spec conformance

- **Standards subcommands**: `fusaops iso26262`, `fusaops iec61508`, `fusaops do178`, `fusaops iso21434`, `fusaops unece`, `fusaops iec62443` — roll up each language tool's gap reports into a cross-language PASS/GAP matrix; `--strict` exits 1 on any gap; skipped components stay visible
- **`fusaops conform <binary>`**: runs 37 checks against any x-FuSa tool binary (version, init, check, trace, qualify, release, audit-pack, capabilities); validates JSON schemas, exit codes, and key-naming invariants; exit 1 on any MUST failure; conformance gate per spec §16 step 7
- JSON Schemas for all 9 document kinds (`spec/schemas/`)
- Golden reference vectors with pre-computed fingerprints (`spec/vectors/`)
- x-FuSa spec v1.8 published; `schemaVersion` field required across all documents

### Safety
- Requirements registry at 101 + CNF/STD group (added `fusaops conform` and standards requirements)
- Docs: `docs/conformance.md`, per-standard references (`docs/standards/`)

## [0.2.0] — 2026-06-10

### Evidence aggregation
- `trace` package: rolls every tool's requirement traceability matrix and
  qualification summary into one cross-language `Aggregate`; skipped components
  stay visible and are excluded from coverage totals; text / JSON / HTML
  renderers and a PASS/GAP status
- `sbom` package: merges and de-duplicates every tool's SBOM on `(name,
  version)`; native JSON, plain-text, and **SPDX 2.3** renderers
- `auditpack` package: bundles each tool's own audit-pack plus the FuSaOps
  aggregate report, trace matrix, and SBOM into one deterministic ZIP with a
  `manifest.json` recording each file's size and SHA-256
- `adapter` capability interfaces (`Tracer`, `Qualifier`, `SBOMer`, `Packer`):
  optional, type-asserted by the orchestrator so a tool contributes evidence
  only where it supports it; `cmdAdapter` implements all four via the tool's
  `trace` / `qualify` / `release` / `audit-pack` subcommands
- `orchestrator`: `RunTrace`, `RunSBOM`, `RunAuditPack` roll-ups
- CLI: `fusaops trace` (with `--strict` polyglot coverage gate), `fusaops sbom`,
  `fusaops audit-pack`

### Safety
- Requirements registry grown to 101 (added TRC, SBM, PCK and new ADP/ORC/CLI
  requirements); 100% traced and tested; `gofusa check` passes with 0 errors
- Docs: per-command references for `trace`, `sbom`, `audit-pack`

## [0.1.0] — 2026-06-09

### Multi-language orchestration core
- `adapter` package: `Adapter` interface, registry, and `cmdAdapter` generic
  implementation parsing the common `<tool> check --format json` schema
- Built-in adapters: go-FuSa (`gofusa`), c-FuSa (`cfusa`), cpp-FuSa (`cpfusa`)
- `scan` package: language detection with file counts
- `orchestrator` package: runs applicable + installed tools, records skipped
  components so coverage gaps are explicit
- `report` package: `AggregateReport` with text / JSON / HTML / SARIF renderers
- `server` package: self-contained web dashboard + JSON API (`fusaops serve`)
- `config` package: optional `.fusaops.json` (zero-config by default)
- CLI: `init`, `scan`, `adapters`, `check`, `report`, `serve`, `version`
- `fusaops check` exits non-zero on any ERROR finding across languages;
  `--strict` also gates on WARNING findings

### Docker quickstart
- All-in-one image (`ghcr.io/soundmatt/fusaops`) bundling the x-FuSa tools by
  copying each binary from its published image
  (`COPY --from=ghcr.io/soundmatt/go-fusa:latest`) — `docker run fusaops check`
  scans with no local installs and no Docker socket
- `tools-monitor.yml`: `repository_dispatch` (`xfusa-released`) + weekly
  scheduled rebuild of `fusaops:latest` with `pull: true`, refreshing bundled
  tools with no manual rebuild
- `docs/extending.md`: one-line process for bundling a future x-FuSa
- `docker-compose.yml` defaults to the published image

### Safety artifacts (go-FuSa parity)
- `.fusa-reqs.json`: 61 requirements (ISO 26262 / ASIL-C); `gofusa trace`
  reports 61/61 traced **and** tested via `//fusa:req` / `//fusa:test`
- `.fusa-hara.json`: tool-failure HARA (dropped finding, masked coverage gap,
  silent failure, misattribution, stale evidence) with safety goals
- Committed evidence: safety-case, TARA, dFMEA, SBOM, provenance, artifact
  manifest, coupling, cyber, vuln, qualify report, test-evidence bundle
- Docs: `tool-safety-manual`, `qualification`, `release-process`, per-command
  (`docs/commands/`) and per-standard (`docs/standards/`) references,
  `INCIDENT-RESPONSE.md`, `CLAUDE.md`
- `.github/`: issue templates, PR template, CODEOWNERS, `fusaops-example.yml`

### CI / quality
- go-FuSa self-check job (FuSaOps gates its own Go source with `gofusa check`)
- Go 1.22/1.23 × Ubuntu/macOS/Windows matrix, 80% coverage gate (84% actual),
  golangci-lint v2.1.6, DCO sign-off, CodeQL, SARIF upload, Docker build
  smoke-test, concurrency cancellation

[Unreleased]: https://github.com/SoundMatt/FuSaOps/compare/v1.10.0...HEAD
[1.10.0]: https://github.com/SoundMatt/FuSaOps/compare/v1.9.0...v1.10.0
[1.9.0]: https://github.com/SoundMatt/FuSaOps/compare/v1.8.0...v1.9.0
[1.8.0]: https://github.com/SoundMatt/FuSaOps/compare/v1.7.0...v1.8.0
[1.7.0]: https://github.com/SoundMatt/FuSaOps/compare/v1.6.0...v1.7.0
[1.6.0]: https://github.com/SoundMatt/FuSaOps/compare/v1.5.0...v1.6.0
[1.5.0]: https://github.com/SoundMatt/FuSaOps/compare/v1.4.0...v1.5.0
[1.4.0]: https://github.com/SoundMatt/FuSaOps/compare/v1.3.0...v1.4.0
[1.3.0]: https://github.com/SoundMatt/FuSaOps/compare/v1.2.0...v1.3.0
[1.2.0]: https://github.com/SoundMatt/FuSaOps/compare/v1.1.0...v1.2.0
[1.1.0]: https://github.com/SoundMatt/FuSaOps/compare/v1.0.0...v1.1.0
[1.0.0]: https://github.com/SoundMatt/FuSaOps/compare/v0.9.0...v1.0.0
[0.9.0]: https://github.com/SoundMatt/FuSaOps/compare/v0.8.0...v0.9.0
[0.8.0]: https://github.com/SoundMatt/FuSaOps/compare/v0.7.0...v0.8.0
[0.7.0]: https://github.com/SoundMatt/FuSaOps/compare/v0.6.0...v0.7.0
[0.6.0]: https://github.com/SoundMatt/FuSaOps/compare/v0.5.1...v0.6.0
[0.5.1]: https://github.com/SoundMatt/FuSaOps/compare/v0.5.0...v0.5.1
[0.5.0]: https://github.com/SoundMatt/FuSaOps/compare/v0.4.0...v0.5.0
[0.4.0]: https://github.com/SoundMatt/FuSaOps/compare/v0.3.0...v0.4.0
[0.3.0]: https://github.com/SoundMatt/FuSaOps/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/SoundMatt/FuSaOps/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/SoundMatt/FuSaOps/releases/tag/v0.1.0
