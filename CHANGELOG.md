# Changelog

All notable changes to FuSaOps are documented here.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.0.0/)
and the project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.89.0] — 2026-07-26

### feat: coverage expansion — report markdown helpers, policy scopeLabel, pr.Save, hara.MaxASIL

- **`report`** — `TestMarkdownBadgePending` covers the `default` branch of `markdownBadge`
  (unknown status → PENDING badge); `TestMarkdownSeverityIconInfo` covers the `default`
  branch of `markdownSeverityIcon` (SeverityInfo → ℹ️ INFO); `TestMarkdownLocFileOnly`
  covers the file-only path in `markdownLoc` (file present, line == 0).
- **`policy`** — `TestScopeLabelToolOnly` covers the tool-only branch of `scopeLabel`;
  `TestScopeLabelNeither` covers the empty-return branch; `TestRenderToFileCreateError`
  covers `RenderToFile` `os.Create` error when parent directory does not exist.
- **`pr`** — `TestSaveWriteError` covers `Save` `os.WriteFile` error when project root
  does not exist.
- **`hara`** — `TestMaxASILUnknownASIL` covers `asilRank` default return (-1) by calling
  `MaxASIL` with an unrecognised ASIL string.

Total coverage: 88.6% (↑ from 88.4%).

## [1.88.0] — 2026-07-26

### feat: coverage expansion — coverage.Parse, conform.scaffold, slsa assessment branches

- **`coverage`** — `TestParseMalformedLines` covers the three `continue` branches in
  `Parse` (no-colon, wrong-field-count, no-dash-in-range) previously unreachable by
  well-formed profile input.
- **`conform`** — `TestScaffoldWriteError` covers the `scaffold()` `os.WriteFile` error
  return when the project directory does not exist.
- **`slsa`** — `TestAssessProvenanceMalformedJSON` covers the `json.Unmarshal` error
  branch in `assessProvenanceField`; `TestAssessArtifactIntegrityDotSha256` covers the
  `.sha256` file loop branch in `assessArtifactIntegrity`.

Total coverage: 88.4% (↑ from 88.3%).

## [1.87.0] — 2026-07-26

### feat: coverage expansion — server, standards, suppression, disposition, vuln, adapter, comp, fleet, metrics, doctemplate

- **`server`** — `TestWriteHeaderViaAuth` covers `statusRecorder.WriteHeader` (was 0%).
- **`standards`** — `TestRenderToFileCreateError` covers `RenderToFile` `os.Create` error.
- **`suppression`** — `TestSaveConfigWriteError` covers `SaveConfig` write-failure path.
- **`disposition`** — `TestRenderEntriesEmptyProject` covers empty-project branch in `RenderEntries`.
- **`vuln`** — `TestScanSkipsVendorDir` covers `discoverManifests` `filepath.SkipDir` branch.
- **`adapter`** — `TestMustRegisterPanic` covers `MustRegister` panic path (was 50%).
- **`comp`** — `TestRenderToFileCreateError` covers `RenderToFile` `os.Create` error.
- **`fleet`** — `TestRenderToFileCreateError` covers `RenderToFile` `os.Create` error.
- **`metrics`** — `TestSaveWriteError` covers `Save` `os.WriteFile` error path.
- **`doctemplate`** — `TestSaveWriteError` covers `Save` `os.WriteFile` error path.

Total coverage: 88.3% → 88.5% (estimated).

## [1.86.0] — 2026-07-26

### feat: coverage expansion — tara, vuln, config, disposition

- **`tara`** — added internal `TestRiskMatrixUncoveredBranches` covering all
  `riskMatrix` combinations not exercised by the standard scenarios
  (`ImpactMajor×default`, `ImpactModerate×High/default`, outer `default`).
  `riskMatrix` now 100%.
- **`vuln`** — added internal `runner_test.go` with `TestRunCommandGoVersion`,
  `TestDefaultRunnerEmptyArgs`, `TestDefaultRunnerGoVersion` covering the real
  `runCommand` and `defaultRunner` execution paths.
- **`config`** — added `TestLoadDirectoryNotExist` covering the
  non-`ErrNotExist` read-error branch in `Load` (os.ReadFile on a directory
  → EISDIR).
- **`disposition`** — added `TestLoadReadError` covering the non-`ErrNotExist`
  read-error branch in `Load` (projectRoot is a regular file → ENOTDIR).
- **Total coverage: 88.0% → 88.2%** (tara: 89.0%→95.9%; vuln: 88.5%→91.8%;
  config: 88.2%→91.2%; disposition: 87.5%→90.0%).

## [1.85.0] — 2026-07-26

### fix + feat: gofusa blank-identifier fix; coverage expansion across 5 packages

- **gofusa selfcheck fix** — resolved blank-identifier finding in `history/history_test.go`
  (`TestPruneWriteAllError`): changed `_, err := Prune(...)` to named variable `n` so the
  discard is explicit.
- **`orchestrator`** — added `detectErrAdapter` + 6 `*SelectError` tests covering the
  `selectAdapters` error propagation path in `RunTrace`, `RunSBOM`, `RunStandards`,
  `RunComp`, `RunMCDC`, and `RunAuditPack`.
- **`diff`** — added 7 tests covering: sort-by-line (Added/Removed), sort-by-ruleID
  (Added/Removed), `Summary` default (RemovedInfos), `renderText` with removals, and
  `SaveBaseline` write-error path.
- **`impact`** — added `TestAppendUniqDuplicate` (duplicate-prevention branch),
  `TestAnalyseWithFromRef` and `TestAnalyseWithBothRefs` (changedFiles arg-selection branches).
- **`verify`** — added `TestRunNonExistentDir` covering the non-ExitError path in `Run`.
- **Total coverage: 87.6% → 88.0%** (all packages ≥ 80%; diff: 87.9% → 97.1%,
  verify: 84.1% → 95.2%, orchestrator: 88.2% → 90.1%, impact: 86.5% → 89.7%).

## [1.84.0] — 2026-07-26

### feat: coverage expansion — write-error paths across 6 more packages

- **`config`** — `TestSaveWriteError` covers the `os.WriteFile` failure branch in `Save`.
- **`disposition`** — `TestSaveWriteError` covers the `os.WriteFile` failure branch in `Save`.
- **`tara`** — `TestSaveWriteError` covers the `os.WriteFile` failure branch in `Save`.
- **`history`** — `TestPruneWriteAllError` covers the `os.Create` failure branch in
  `writeAll` by replacing the history file with a directory before calling `Prune`.
- **`sign`** — `TestKeygenWriteError` covers the `os.WriteFile` failure branch in `Keygen`.
- **Total coverage: 87.5% → 87.6%** (all packages ≥ 80%).

## [1.83.0] — 2026-07-26

### fix: resolve gofusa selfcheck findings; maximize coverage across 12 packages

- **gofusa selfcheck fix** — resolved 5 blank-identifier-in-call-assignment findings
  flagged by the SARIF upload after v1.82.0 merge:
  - `req/req_test.go`: `TestParseCodebeamerInvalidXML`, `TestParseJamaInvalidXML`,
    `TestParsePolarionInvalidXML` now use named variables (`entries, err :=`) rather
    than `_, err :=` so the discard of the result is explicit.
  - `cmd/fusaops/cmd_gap2_test.go`: `os.Stat` results assigned to named `st` and mode
    validated (3 occurrences: baseline, provenance.json, hook file).
- **Coverage improvements across 12 packages** (total **87.5%**, up from 87.3%):
  - **`scan`** — `TestScanTraversesNonSkippedSubdir` exercises the `skipDir` false branch
    (non-skipped subdirectory). scan: 92.1% → **92.9%**.
  - **`standards`** — `TestDisplayNameExported` covers the exported `DisplayName` wrapper.
    standards: 92.0% → **92.8%**.
  - **`slsa`** — `TestRenderTextUnknownStatus` exercises `statusIcon` default branch.
    slsa: 91.8% → **92.9%**.
  - **`hara`** — `TestSaveWriteError` covers the `os.WriteFile` error path in `Save`.
  - **`fmea`** — `TestSaveWriteError` + `TestRenderTextLowRPN` (RPN ≤ 50 → "LOW" label).
  - **`sas`** — `TestSaveWriteError` covers the `os.WriteFile` error path in `Save`.
  - **`qualify`** — `TestSaveWriteError` covers the `os.WriteFile` error path in `Save`.
  - **`safetycase`** — `TestSaveWriteError` + `TestRenderTextNonPrefixPath` covers
    the `shortPath` fallback (evidence path outside project root).
  - **`sci`** — `TestRenderTextWithComponentItem` exercises `kindLabel(KindComponent)`;
    `TestSaveWriteError` covers the `os.WriteFile` error path.
  - **`vuln`** — `TestSaveWriteError` covers the `os.WriteFile` error path. vuln: 87.7% → **88.5%**.
  - **`release`** — `TestSaveJSONWriteError` covers the `os.WriteFile` error path in `SaveJSON`.

## [1.82.0] — 2026-07-26

### feat: maximize test coverage in req, diff, report, cmd packages

- **`req/req_test.go`** — 12 new tests covering error paths and edge cases in
  `ParseCodebeamer`, `ParseJama`, and `ParsePolarion` (invalid XML, empty-ID
  fallback to name, skip-empty, and custom level field); `SaveRegistry` read-only
  dir error path. req package coverage 82.0% → **91.2%**.
- **`diff/diff_test.go`** — 2 new tests for plural severity labels ("2 errors",
  "2 warnings", "2 infos") and info-severity detail in `severityDetail`.
  diff package coverage 84.5% → **87.9%**.
- **`report/report_test.go`** — 3 new tests for `renderHTML`, `renderCSV`, and
  `renderJSON` write-error paths via an `errWriter` that returns an error on every
  Write call. report package coverage 91.2% → **91.9%**.
- **`cmd/fusaops/cmd_gap2_test.go`** — 3 new tests: release without `--dir` (exercises
  `os.Getwd()` path), `hooksInstall` happy path with a fresh hook path.
  cmd package coverage 81.4% → **81.5%**.
- **Total coverage: 87.3%** (up from 86.9%; all packages ≥ 80%).

## [1.81.0] — 2026-07-26

### docs: spec v1.10.12 snapshot — rust-FuSa v0.3.4 Dockerfile fix

- **`SpecVersion` bumped to `"1.10.12"`** — captures rust-FuSa v0.3.4 (Dockerfile
  multi-platform musl fix; no behavior change).
- **`docs/x-fusa-spec.md` §11 snapshot updated**: rust-FuSa v0.3.3 → v0.3.4.
- **`docs/x-fusa-spec.md` §14 changelog entry 1.10.12 added**.
- **`README.md` bundled tool version updated**: rust-FuSa v0.3.4 in prose and table.

## [1.80.0] — 2026-07-26

### feat: maximize test coverage across orchestrator, slsa, trace, server, verify, cmd

- **`orchestrator/rollup_test.go`** — additional `RunMCDC` tests covering edge cases and
  error paths; orchestrator coverage 88.2%.
- **`slsa/slsa_test.go`** — `assessSBOMHashes` edge-case tests; slsa coverage 91.8%.
- **`trace/trace_test.go`** — HLR/LLR decomposition, `renderHTML`, `renderMarkdown`
  tests; trace coverage 93.6%.
- **`cmd/fusaops/cmd_gap2_test.go`** — 14 new CLI subcommand tests; cmd coverage 81.4%.
- **`server/server_gap_test.go`** — 10 new HTTP endpoint tests; server coverage 84.8%.
- **`verify/verify_gap_test.go`** — additional verify path test; verify coverage 84.1%.
- **Total coverage: 86.9%** (up from previous release; all packages ≥ 80%).
- **`docs/x-fusa-spec.md` §11 tool versions updated**: c-FuSa v0.5.38 · rust-FuSa v0.3.3.

## [1.79.0] — 2026-07-26

### docs: spec v1.10.11 snapshot — all-tool safety features, latest versions

- **`SpecVersion` bumped to `"1.10.11"`** — captures the four safety features
  (HLR/LLR decomposition, tool qualification display, MC/DC coverage, V&V
  independence) now implemented across all six x-FuSa tools.
- **`docs/x-fusa-spec.md` updated to v1.10.11:**
  - Header: `1.10.10` → `1.10.11`.
  - §11 snapshot updated to 2026-07-26: go-FuSa v0.33.0 · cpp-FuSa v0.14.0 · c-FuSa v0.5.37 · rust-FuSa v0.3.2 · py-FuSa v0.2.1 · java-FuSa v0.4.1.
  - §14: added 1.10.11 entry documenting all four safety features and their FuSaOps consumption.
- **`README.md` updated**: bundled-tool version table updated to latest releases;
  bundled-tools prose updated to reflect all six tools active in the image.

## [1.78.0] — 2026-07-26

### fix: maximize test coverage and requirement traceability

Audit-driven improvements targeting P0 correctness bugs and P1 coverage gaps.

#### P0 — Correctness

- **`trace.Qualification`**: add `IndependentReviewer` and `QualificationMethod`
  fields so the struct fully mirrors the go-FuSa v0.32.0 qualify JSON output
  (REQ-FO-QLF010). These fields were previously silently dropped on decode.
- **`qualify.Run()`**: copy `qr.IndependentReviewer` and `qr.QualificationMethod`
  into each `ComponentResult` after the total/passed/failed assignments.
  `IsIndependent()` now correctly returns `true` for tools that declare an
  independent reviewer.
- **`/api/v1/qualify`**: new JSON endpoint on `server.Server` exposing the
  cached qualification report programmatically. Mirrors how `/api/v1/comp` and
  `/api/v1/mcdc` serve machine-readable data (REQ-FO-SRV014).

#### P1 — Test coverage

- **adapter**: `TestAdapterMCDC` and `TestAdapterMCDCErrors` — new fake-runner
  unit tests for the `McdcRunner` capability path (REQ-FO-MCDC001).
- **report**: `TestQualifyInfoIsIndependent` — exercises `QualifyInfo.IsIndependent()`
  with nil, empty, and populated receiver (REQ-FO-QLF011).
  `TestMCDCInfoCoveragePct` — exercises `MCDCInfo.CoveragePct()` for nil, zero,
  and non-zero condition counts (REQ-FO-MCDC003).
- **qualify**: `TestIndependentReviewerFields` (REQ-FO-QLF010) and `TestIsIndependent`
  (REQ-FO-QLF011) — full round-trip tests for V&V independence propagation.
  `TestRenderTextShowsIndependentReviewer` — text renderer output verified.
- **server**: `TestAPIMCDCBeforeCompute`, `TestAPIMCDCWithData`, `TestMCDCPageBeforeCompute`,
  `TestMCDCPageWithData`, `TestMCDCPageSkippedComponent`, `TestDashboardShowsMCDCSection`
  — new MCDC endpoint and page tests (REQ-FO-MCDC002, REQ-FO-MCDC003).
  `TestAPIQualifyEndpointPending`, `TestAPIQualifyEndpointWithReport` — qualify
  API endpoint tests (REQ-FO-SRV014). `TestIndexNilReport`, `TestAPIReportNilReport`
  — nil-report 503 paths.
- **cmd/fusaops**: `TestDispositionListWithEntries`, `TestReqExportToFile`,
  `TestReqExportBadFormat`, `TestReqExportPolarion`, `TestReqExportCodebeamer`,
  `TestReqExportJama`, `TestReqExportBadFlag`, `TestServeMultiBadJSON`,
  `TestServeMultiInvalidProjectDirs` — targeted tests for the highest-impact
  uncovered dispatch paths (REQ-FO-CLI052, REQ-FO-CLI060, REQ-FO-CLI030).

#### P2 — Tool version references

- **README.md**: updated bundled-tool version table and prose to reflect
  go-FuSa v0.32.0, cpp-FuSa v0.13.0, c-FuSa v0.5.35, rust-FuSa v0.3.0,
  py-FuSa v0.2.0, java-FuSa v0.4.0.
- **Dockerfile**: updated COPY --from stages to match the new tool versions.

## [1.77.0] — 2026-07-26

### feat: HLR/LLR traceability, tool qualification display, MC/DC, V&V independence

Four new x-FuSa cross-language features tracking the tool PRs across all six adapters
(go-FuSa #39, c-FuSa #55, cpp-FuSa #26, rust-FuSa #26, py-FuSa #14, java-FuSa #16).

#### HLR/LLR Decomposition (REQ-FO-TRC030)

- **`trace.HLRLLRSummary`**: new struct with `HLRCount`, `LLRCount`, `Orphaned`, and
  `Uncovered` counts decoded from each tool's trace matrix JSON.
- **`trace.Matrix.HLRLLRSummary`**: per-tool matrix now carries the optional
  HLR/LLR summary alongside its requirements and coverage.
- **`trace.ComponentTrace.HLRLLRSummary`**: per-component aggregate carries the
  per-tool summary for roll-up.
- **`trace.Aggregate.HLRLLRSummary`**: cross-language roll-up sums all non-skipped
  component HLR/LLR counts into a single aggregate summary.
- **Text/HTML renderers** show HLR→LLR hierarchy per component and in the totals
  row; the HTML page gains a new HLR/LLR section below Decomposition.
- **Requirement gaps** in the HTML trace page now show `[parent: <id>]` for LLRs.

#### Tool Qualification V&V Independence (REQ-FO-QLF010, REQ-FO-QLF011)

- **`qualify.ComponentResult`**: six new fields — `QualificationMethod`,
  `QualifierIdentity`, `QualificationRecordUri`, `ImplementationAuthor`,
  `IndependentReviewer`, `AchievableASIL` — decoded from each tool's qualify JSON.
- **`qualify.Report`**: same six fields at the aggregate level; `IsIndependent()`
  helper returns true when `IndependentReviewer` is set.
- **`report.QualifyInfo`**: five new fields propagated from the qualify report into
  the HTML renderer; `IsIndependent()` helper drives the dashboard badge.
- **HTML dashboard**: qualification section now shows an
  **independently-qualified** (green) or **self-qualified** (muted) badge;
  reviewer name and achievable ASIL are displayed when present.
- **`qualify.renderText`**: text output shows all new fields when populated.

#### MC/DC Coverage (REQ-FO-MCDC001, REQ-FO-MCDC002, REQ-FO-MCDC003)

- **`mcdc/` package**: new package with `Report`, `MCDCComponent`, and
  `MCDCAggregate` types for per-tool and cross-language MC/DC coverage tracking.
- **`adapter.McdcRunner`**: new capability interface; `cmdAdapter.MCDC()` shells out
  to `<tool> comp --mcdc --format json`.
- **`orchestrator.RunMCDC()`**: parallel roll-up analogous to `RunComp()`, recording
  skipped components for tools that do not implement `McdcRunner`.
- **`report.MCDCInfo` / `report.MCDCComponent`**: carry MC/DC data into HTML
  renderer without a `report→mcdc` import cycle; wired through `RenderOptions.MCDCInfo`.
- **`/api/v1/mcdc`**: new JSON endpoint serving the cached `MCDCAggregate`.
- **`/mcdc`**: new HTML page with per-component MC/DC coverage table and gate
  status, following the `/comp` page pattern.
- **Dashboard nav**: MC/DC nav link appears when data is available; dashboard
  section shows gate badge and condition coverage.
- **`server.mcdcInfoFromAggregate`**: converts `mcdc.MCDCAggregate` to
  `report.MCDCInfo` without import cycle.

## [1.76.0] — 2026-07-26

### feat: CLI tests for `fusaops comp` (REQ-FO-CLI082)

- **`cmd_comp_test.go`**: adds 10 focused CLI-level tests for `runComp` covering
  no-language detection (`TestCompNoLanguages`), text and JSON output formats,
  invalid `--dal`, invalid `--timeout`, `--output` file write with stderr
  confirmation, `--workers` flag parsing, valid `DAL-B` threshold, invalid format
  error path, and bad output path error path.
- Coverage on `cmd/fusaops/cmd_comp.go:runComp` rises from **0% → 71.4%**.
  Overall project coverage increases from **82.0% → 82.5%**.

## [1.75.0] — 2026-07-25

### feat: cyclomatic complexity metrics in /metrics endpoint

- **`/metrics` comp gauges**: when the server has a cached comp aggregate with at
  least one analysed function, the OpenMetrics endpoint now includes three additional
  gauges: `fusaops_comp_functions_total`, `fusaops_comp_violations_total`, and
  `fusaops_comp_status` (1=PASS, 2=FAIL). The metrics are omitted when no comp data
  is available (no adapters implement `Compler` or all were skipped). (REQ-FO-MTR003)

## [1.74.0] — 2026-07-25

### feat: /comp HTML page with function-level complexity detail

- **`/comp` page**: the `fusaops serve` web server now exposes a `/comp` HTML page
  showing the cached comp aggregate in full detail — per-component sections listing
  every function that exceeds the configured cyclomatic complexity threshold, with
  name, file, line number, and V(G) score. When no comp data is available, the page
  shows a helpful "no data" message. (REQ-FO-SRV013)
- **Dashboard nav link**: when comp data is available, the dashboard header nav
  gains a "Complexity" link pointing to `/comp`. The comp section heading is also
  hyperlinked to `/comp`. (REQ-FO-SRV013)

## [1.73.0] — 2026-07-25

### feat: HTML dashboard cyclomatic complexity section

- **Dashboard comp section**: the `fusaops serve` web dashboard now renders a
  **Cyclomatic Complexity** section below the findings table when the server has a
  cached comp aggregate. The section shows a pass/fail badge, total function count,
  and a per-component table with language, tool, function count, violation count, and
  threshold (DAL-labelled when applicable). The section is hidden when no comp data
  is available. (REQ-FO-RPT021)
- **`report.CompInfo` / `report.CompComponent`**: new types in the `report` package
  carry comp aggregate data into the HTML renderer without a `report→comp` import
  cycle. `RenderOptions.CompInfo` wires them through `RenderWithOptions`. (REQ-FO-RPT021)
- **`server.compInfoFromAggregate`**: helper converts `comp.Aggregate` to
  `report.CompInfo` and is called by `handleIndex` on every dashboard request. (REQ-FO-RPT021)

## [1.72.0] — 2026-07-25

### feat: comp capabilities registration, config support, and /api/v1/comp endpoint

- **`capabilities` fix**: `comp` is now listed in the `commands[]` array and the
  `formats` map (`["text","json"]`) of the `fusaops capabilities` discovery document.
  (REQ-FO-CLI083)
- **`config.CompConfig`**: new `.fusaops.json` section `"comp": { "threshold": N, "dal": "DAL-X" }`
  lets projects set the cyclomatic complexity threshold once for all tools. The server
  and `fusaops comp` CLI both read this config. (REQ-FO-CFG014)
- **`server`: `/api/v1/comp`**: the web server now computes a cross-language comp
  aggregate (`RunComp`) during each refresh cycle and exposes it at `GET /api/v1/comp`
  as JSON. `WithComp(threshold, dal)` is the builder method. (REQ-FO-SRV012)
- New requirements: REQ-FO-CLI083, REQ-FO-CFG014, REQ-FO-SRV012

## [1.71.0] — 2026-07-25

### feat: x-FuSa spec v1.10.10 sync — comp consumption and all-tool snapshot

- **`SpecVersion` bumped to `"1.10.10"`** — corrects the spec header (which had stalled
  at `1.10.4` while §14 accumulated 1.10.5–1.10.9 entries) and adds the MINOR bump for
  FuSaOps beginning to consume the §9.2 `comp` command (v1.70.0+).
- **`docs/x-fusa-spec.md` updated to v1.10.10:**
  - Header: `1.10.4` → `1.10.10`.
  - §9.2 heading: removed `comp` from "not consumed" list; heading updated to reflect
    that comp is now consumed.
  - Intro note (§0): clarified that §9.2 `comp` is consumed since v1.70.0+.
  - §11 snapshot: updated to 2026-07-25 with go-FuSa v0.31.0, cpp-FuSa v0.12.6,
    c-FuSa v0.5.34, rust-FuSa v0.2.9, py-FuSa v0.1.9.
  - §14: added 1.10.10 entry documenting all-tool spec-constant fixes and docker CI additions.
- New tool releases (all SpecVersion constant fixes + docker-publish.yml):
  - go-FuSa v0.31.0: CI builder auto-detect + `--builder` flag on `release`
  - c-FuSa v0.5.34, rust-FuSa v0.2.9, py-FuSa v0.1.9: SpecVersion fix + docker-publish.yml
  - cpp-FuSa v0.12.6: SpecVersion fix + MSVC compile fix

## [1.70.0] — 2026-07-25

### feat: fusaops comp — cross-language cyclomatic complexity roll-up (§9.2)

All six x-FuSa tools implement the `comp` command (McCabe V(G) per function,
DO-178C §6.3.4). FuSaOps now consumes and aggregates those results.

- `comp` package — `Report`, `Function`, `ComponentComp`, `Aggregate` types;
  `New` aggregator; `DALThreshold` / `ValidateDAL` helpers; `Render` / `RenderToFile`
  in `text` and `json` formats. Canonical schema from x-FuSa spec §13.
- `adapter.Compler` interface — `Comp(ctx, root, threshold, dal)` capability
  implemented by `cmdAdapter`: calls `<tool> comp [--threshold N] [--dal DAL-X] --format json`
  and decodes the comp-report.
- `orchestrator.RunComp` — parallel execution across all applicable adapters;
  tools without the binary, capability, or whose comp fails are recorded as skipped.
- `fusaops comp` CLI — `--threshold`, `--dal` (DAL-A|B|C|D), `--dir`, `--format`,
  `--output`, `--only`, `--workers`, `--timeout`; exits 1 on violations.
  DAL-level thresholds: A ≤ 4, B ≤ 10, C ≤ 15, D ≤ 20.
- New requirements: REQ-FO-COMP001–003, REQ-FO-ADP029, REQ-FO-ORC013, REQ-FO-CLI082.

## [1.69.0] — 2026-07-25

### feat: enable py-FuSa and java-FuSa in the all-in-one container image

- `Dockerfile` — uncomment and activate `FROM ghcr.io/soundmatt/py-fusa:latest AS pyfusa`
  and `FROM ghcr.io/soundmatt/java-fusa:latest AS jfusa` now that both tools have
  published Docker images to GHCR (py-FuSa v0.1.9, java-FuSa v0.3.1).
- Runtime base image changed from `alpine:3.20` to `python:3.12-alpine` (= alpine:3.20
  + Python 3.12) so the pyfusa entry-point shebang (`#!/usr/local/bin/python3.12`) is
  satisfied without a separate Python install step.
- `openjdk21-jre-headless` added to `apk add` to satisfy the `java` requirement of the
  jfusa shell wrapper.
- py-FuSa: COPY `/usr/local/bin/pyfusa` + `/usr/local/lib/python3.12/site-packages`
  from the pyfusa stage (same python:3.12-alpine base → ABI-compatible).
- java-FuSa: COPY `/usr/local/lib/jfusa.jar` + `/usr/local/bin/jfusa` from the jfusa stage.
- New requirements: REQ-FO-IMG001 (py-FuSa in image), REQ-FO-IMG002 (java-FuSa in image).

## [1.68.0] — 2026-07-25

### feat: provenance builder field with CI auto-detection (go-FuSa v0.31.0 parity)

- `release.Provenance.Builder` — new `builder` field (JSON `"builder"`) on the
  provenance record, matching the x-FuSa spec §7 provenance schema.
- `release.DetectBuilder(override)` — auto-detects the CI/CD system from env
  vars: `GITHUB_ACTIONS`+`GITHUB_WORKFLOW_REF` → `"github-actions/<ref>"` or
  `"github-actions"`; `GITLAB_CI` → `"gitlab-ci"`; `JENKINS_URL` → `"jenkins"`;
  any `CI` → `"ci"`; local/unknown → `""`. An explicit non-empty override is
  returned verbatim.
- `release.BuildProvenance` — gains a `builder` parameter; passes it through
  `DetectBuilder` so auto-detection applies when the caller passes `""`.
- `fusaops release --builder <uri>` — new flag for explicit builder override;
  auto-detection applies when the flag is omitted.
- Text render (`fusaops release`) shows `Builder:` line when non-empty.
- New requirements: REQ-FO-REL005, REQ-FO-CLI081.

## [1.67.0] — 2026-07-25

### Added — MC/DC Coverage Gate (DO-178C DAL-A prerequisite)

- `fusaops coverage --mcdc` — LLVM source-based MC/DC coverage gate.
  When set, parses the LLVM `llvm-cov export --format=json` output
  (`--mcdc-file`, default `mcdc.json`), scans Go source for `//fusa:req`-annotated
  functions (`--req-dir`), and fails the gate (exit 1) if any annotated function
  has uncovered conditions or if overall condition coverage falls below
  `--mcdc-threshold` (default 100.0%).
- `coverage.ParseMCDC` — decodes LLVM MC/DC JSON into `[]McdcFunction`.
- `coverage.AnalyseMCDC` — merges coverage data with annotated function set and
  computes `McdcReport` with `GatePassed`, `CondPct`, `UncoveredReqs`.
- `coverage.GateMCDC` — gate accessor over `McdcReport.GatePassed`.
- `coverage.RenderMCDC` — renders `McdcReport` as text, JSON, or markdown.
- `coverage.FindAnnotatedFunctions` — stdlib `go/parser` walk returning
  `//fusa:req`-annotated function names from `.go` source files.
- `coverage.McdcCondition`, `McdcDecision`, `McdcFunction`, `McdcReport` — new
  exported types for structured DO-178C MC/DC evidence.
- `config.CoverageConfig` and `config.McdcConfig` — optional `"coverage"` section
  in `.fusaops.json` with `mcdc.enabled` and `mcdc.threshold`; `Validate` rejects
  threshold values outside `[0, 100]`.
- 8 new requirements: REQ-FO-COV004–COV009, REQ-FO-CLI080, REQ-FO-CFG013.

## [1.66.0] — 2026-07-25

### Feature: Tool Qualification Record Display (issue #29)

New requirements REQ-FO-QUAL005–007, REQ-FO-CFG012, REQ-FO-CLI078–079,
REQ-FO-SRV011 across four layers: the qualify schema, config, CLI, and web
dashboard.

- `qualify.QualificationType` — typed enum: `"self"` (default) or
  `"independent"` (TQL-5/DO-330 externally certified).
- `qualify.RunOptions` — passes type and certificate URI into `qualify.Run()`;
  both fields are covered by the tamper-evident SHA-256 hash.
- `qualify.Report` — gains `qualificationType` and `qualificationRecordUri`
  JSON fields; text output (`fusaops qualify`) shows `Type:` and `Record:` lines.
- `config.QualifyConfig` — new `qualify` section in `.fusaops.json` for
  project-level defaults; `Validate()` rejects unknown types.
- `fusaops qualify --type self|independent --record-uri <uri>` — per-run
  override; config-file defaults are respected when flags are at their defaults.
- `fusaops serve --qualify-report <path>` — wires an on-disk qualification
  report into the dashboard.
- `GET /badge/qualify.svg` — SVG badge: `pending` / `<type> / pass` /
  `<type> / failing`; proper `Cache-Control` and `Content-Type` headers.
- HTML dashboard — qualification gap section in the header when a qualify
  report is available: badge, type, passed/total checks, and certificate link.

## [1.65.0] — 2026-07-25

### feat: HLR/LLR requirement decomposition gate

- **`fusaops trace --decomp`** runs the new cross-language HLR/LLR
  decomposition gate after collecting the aggregate traceability matrix.
  Every LLR must reference a known HLR parent; every HLR must have at least
  one LLR child. Requirements with no `level` field are silently ignored, so
  legacy matrices remain unaffected.
- **`fusaops trace --decomp-enforce warn|error|off|auto`** overrides the
  per-project severity. `auto` (the default) derives severity from the
  project integrity level: DAL-A/B or ASIL-C/D → `error`; all others →
  `warn`. The gate never exits non-zero in `warn` mode.
- **`Config.Trace.ReqDecompositionConfig`** — new `.fusaops.json` section
  (`trace.reqDecomposition.enforce` / `trace.reqDecomposition.minLevel`).
  Validated by `config.Validate`; `enforce` accepts `off|warn|error|auto`.
- **`trace.Requirement.Parent`** — new `parent` field (JSON `"parent"`)
  carrying the ID of the HLR a given LLR refines. Decoded from tool trace
  JSON output; also round-trips through `req.Entry.Parent` in CSV import/export.
- **`trace.CheckDecomposition`** / **`trace.SeverityForDecomposition`** —
  new exported functions implementing the gate logic.
- **`trace.DecompositionReport`** / **`trace.DecompositionViolation`** —
  new types attached to `trace.Aggregate.Decomposition`; automatically
  included in JSON, text, markdown, and HTML render outputs.
- New requirement IDs: REQ-FO-TRC019–TRC022, REQ-FO-CFG010, REQ-FO-CLI074.

## [1.64.0] — 2026-07-25

### feat: V&V independence tracking (GAP-003)

New `fusaops vv` command and supporting infrastructure for per-repo V&V
independence declarations, achievable ASIL computation, a web JSON API endpoint,
and an SVG badge.

**New package `vv/`**
- `vv.Declaration` — holds `implementationAuthor`, `independentReviewer`, and
  `independentTestExecutor` fields per ISO 26262-2:2018 §6.4.
- `vv.IndependenceLevel` — returns 0 (none), 1 (reviewer only), or 2 (reviewer + executor).
- `vv.AchievableASIL` — maps independence level to achievable ASIL: ASIL-B / ASIL-C / ASIL-D.
- `vv.Validate` — returns human-readable warnings for consistency problems.
- `vv.Render` — renders declarations in `text` or `json` format.

**New CLI commands**
- `fusaops vv show [--format text|json] [--output file]` — display current
  declarations and computed achievable ASIL; prints validation warnings to stderr.
- `fusaops vv set [--implementation-author X] [--independent-reviewer X]
  [--independent-test-executor X]` — update fields in `.fusaops.json`; only
  supplied flags are modified.

**Config schema** (`config.VandVConfig`)
- New optional `vv` section in `.fusaops.json` with `implementationAuthor`,
  `independentReviewer`, and `independentTestExecutor` fields.

**Server endpoints** (`fusaops serve`)
- `GET /api/v1/vv` — JSON response with declarations and `independenceLevel` /
  `achievableAsil` derived fields.
- `GET /badge/vv.svg` — SVG badge showing achievable ASIL (green ASIL-D,
  yellow-green ASIL-C, yellow ASIL-B, grey if unset).

**New requirements:** REQ-FO-VV001–004, REQ-FO-CFG010, REQ-FO-CLI074–076,
REQ-FO-SRV010, REQ-FO-BADGE003.

## [1.63.0] — 2026-07-25

### Container: enable c-FuSa and rust-FuSa binaries in all-in-one image

- `ghcr.io/soundmatt/fusaops` now bundles `cfusa` (c-FuSa v0.5.34) and `rsfusa`
  (rust-FuSa v0.2.9). Both tools now publish GHCR images triggered by tag push.
- C and Rust projects scanned with the container image produce cfusa/rsfusa
  findings in the aggregate report — no config change needed.
- `pyfusa` (Python) and `jfusa` (Java) are deferred: both require additional
  runtime layers (Python 3.12 and eclipse-temurin JRE respectively). Tracked in
  py-FuSa#13 and java-FuSa#15.

### Subproject releases

- go-FuSa v0.31.0 — SpecVersion `"1.10.4"` fix + CI builder auto-detection
- c-FuSa v0.5.34 — SpecVersion `"1.10.4"` fix + docker-publish.yml
- cpp-FuSa v0.12.6 — SpecVersion `"1.10.4"` fix + MSVC C2338 fix
- rust-FuSa v0.2.9 — SpecVersion `"1.10.4"` fix + docker-publish.yml
- py-FuSa v0.1.9 — SpecVersion `"1.10.4"` fix + docker-publish.yml (first tagged release)
- java-FuSa v0.3.1 — docker-publish.yml (first tagged release)

## [1.62.0] — 2026-07-25

### Container: enable cpp-FuSa binary in all-in-one image

- `ghcr.io/soundmatt/fusaops` now bundles `cpfusa` (cpp-FuSa v0.12.5). The
  `FROM ghcr.io/soundmatt/cpp-fusa:latest AS cpfusa` stage was previously
  commented out pending the image being published; `ghcr.io/soundmatt/cpp-fusa`
  is now confirmed published and the stage is active.
- C++ projects scanned with the container image now produce cpfusa findings in
  the aggregate report — no config change needed.
- Remaining four tools (c-FuSa, rust-FuSa, py-FuSa, java-FuSa) are tracked in
  c-FuSa#44, rust-FuSa#15, py-FuSa#5, java-FuSa#9.

### Tickets raised on x-FuSa subprojects

Container image gaps:
- c-FuSa#44 — add docker-publish.yml (no ghcr.io/soundmatt/c-fusa image)
- rust-FuSa#15 — add docker-publish.yml (no ghcr.io/soundmatt/rust-fusa image)
- py-FuSa#5 — add docker-publish.yml (no ghcr.io/soundmatt/py-fusa image)
- java-FuSa#9 — add Dockerfile + docker-publish.yml (no image; needs openjdk21-jre)

Stale specVersion constants:
- go-FuSa#31 — SpecVersion "1.9" should be "1.10.4"
- c-FuSa#45 — CFUSA_SPEC_VERSION "1.9" should be "1.10.4"
- cpp-FuSa#15 — SpecVersion "1.10" missing patch; should be "1.10.4"
- rust-FuSa#16 — SPEC_VERSION "1.10" missing patch; should be "1.10.4"
- py-FuSa#6 — SPEC_VERSION "1.10.8" diverged from canonical spec (1.10.4)

## [1.61.0] — 2026-06-16

### §2.4.1 capabilities `Standards` — add canonical `"slsa"` identifier

- `fusaops capabilities` now includes `"slsa"` in the `Standards` JSON array, matching the §2.4.1 canonical identifier. (REQ-FO-SPEC002)
- Mirrors `gofusa` v0.30.0 §2.4.1 fix.
- New test `TestCapabilitiesStandardsSLSA` verifies the canonical entry is present.

### Safety
- Requirements registry at 359 requirements (added REQ-FO-SPEC002)

## [1.60.0] — 2026-06-16

### §2.2 `--output` no-stdout invariant fix

- **`fusaops check`**, **`fusaops report`**, **`fusaops trace`**, **`fusaops sbom`**, **`fusaops iso26262`** (and all `standards` sub-commands), and **`fusaops conform`** now write the `Wrote ... to <file>` confirmation line to **stderr** rather than stdout when `--output <file>` is given. stdout is empty when output goes to a file, making all six commands safe for pipeline use (`fusaops check --output report.json | jq ...`). (REQ-FO-SPEC001)
- Mirrors `gofusa` v0.29.0 §2.2 stdout clean invariant.
- 4 existing tests updated to assert the confirmation on stderr and stdout emptiness.

### Safety
- Requirements registry at 358 requirements (added REQ-FO-SPEC001)

## [1.59.0] — 2026-06-16

### Aggregate report — standard and integrity level fields

- **`report.AggregateReport`** gains `Standard`, `ASIL`, `SIL`, `DAL` fields (`omitempty`). These propagate the project's safety standard and integrity level into the report JSON and every renderer. (REQ-FO-RPT020)
- **`config.ProjectConfig`** gains `ASIL`, `SIL`, `DAL` optional string fields in `.fusaops.json`, mirroring the `.fusa.json` pattern. `Standard` was already present.
- **`fusaops check` and `fusaops report`** now populate `Standard`/`ASIL`/`SIL`/`DAL` on the report from the project config (via `applyIntegrityLevel`). Switch: `IEC61508` → `SIL`, `DO178C` → `DAL`, default → `ASIL`.
- **Text renderer** shows `Standard: <std> (<level>)` when standard and level are both set; shows `Standard: <std>` when only standard is present.
- **Markdown renderer** shows `**Standard:** <std> (<level>)` in the report header.
- Mirrors `gofusa` v0.27.0 `sil`/`dal` JSON fields feature.

### Safety
- Requirements registry at 357 requirements (added REQ-FO-RPT020)
- 7 new report package tests; combined coverage ≥ 80%

## [1.58.0] — 2026-06-16

### `fusaops hara` — Hazard Analysis and Risk Assessment (ISO 26262-3:2018)

- **`hara` package** — `Severity`, `Exposure`, `Controllability`, `ASIL`, `RiskRating`, `OperationalSituation`, `Hazard`, `SafetyGoal`, `HARA`, `ValidationFinding` types; `DetermineASIL` (full ISO 26262-3:2018 Table 4 — 48 S×E×C combinations); `MaxASIL`; `Load`, `Save`, `Validate`, `Render` (text/json/markdown). Reads `.fusa-hara.json` if present; returns an empty HARA if absent. `Validate` detects HARA002–HARA004 gaps (incomplete risk rating, missing safety goal link, missing ASIL on safety goal). (REQ-FO-HARA001, REQ-FO-HARA002, REQ-FO-HARA003, REQ-FO-HARA004)
- **`fusaops hara show [--format text|json|markdown] [--output PATH]`** — display HARA as table or JSON.
- **`fusaops hara init [--project NAME] [--standard STD]`** — create a starter `.fusa-hara.json` with one example hazard and safety goal.
- **`fusaops hara asil -s S# -e E# -c C#`** — derive ASIL from S/E/C per ISO 26262-3:2018 Table 4. (REQ-FO-CLI073)
- `fusaops capabilities` updated with `hara` command and `text`/`json`/`markdown` formats.
- Mirrors `gofusa hara` adapted for the FuSaOps multi-language orchestration context.

### Safety
- Requirements registry at 356 requirements (added REQ-FO-HARA001–004, REQ-FO-CLI073)
- 18 package tests + 7 CLI tests; combined coverage ≥ 80%

## [1.57.0] — 2026-06-16

### `fusaops template` — Safety documentation template generator

- **`doctemplate` package** — `DocKind`, `DocTemplate`, `GeneratedDoc`, `Report` types; `Generate`, `Save`, `Load`, `Render` (text/json). Built-in library of 8 safety document templates: Software Safety Plan, HARA, SRS, Test Plan, TARA, SCI, SAS, and Problem Report. Templates are filtered by target standard(s) and written as Markdown files to an output directory. SHA-256 integrity hash over the generation report. (REQ-FO-TMPL001, REQ-FO-TMPL002, REQ-FO-TMPL003, REQ-FO-TMPL004)
- **`fusaops template [--dir DIR] [--output-dir DIR] [--standards ISO 26262,...] [--format text|json] [--output PATH]`** — generate and report on templates; always exits 0. (REQ-FO-CLI072)
- `fusaops capabilities` updated with `template` command and `text`/`json` formats.
- Mirrors `gofusa template` adapted for the FuSaOps multi-language orchestration context.

### Safety
- Requirements registry at 351 requirements (added REQ-FO-TMPL001–004, REQ-FO-CLI072)
- 15 package tests + 6 CLI tests; combined coverage ≥ 80%

## [1.56.0] — 2026-06-16

### `fusaops vuln` — Cross-language dependency vulnerability scan (OSV)

- **`vuln` package** — `ManifestKind`, `ScanStatus`, `Manifest`, `VulnFinding`, `VulnReport` types; `Scan`, `Save`, `Load`, `Render` (text/json). Discovers dependency manifests (go.mod, Cargo.toml, requirements.txt, package.json, pom.xml) by walking the project tree. When `osv-scanner` is available on PATH it is invoked via an injectable `RunnerFunc`; when absent each manifest is recorded with status `skipped`. Findings are classified by CRITICAL/HIGH severity. SHA-256 integrity hash over the assembled report. (REQ-FO-VULN001, REQ-FO-VULN002, REQ-FO-VULN003, REQ-FO-VULN004)
- **`fusaops vuln [--dir DIR] [--format text|json] [--output PATH]`** — discover manifests and run vulnerability scan; persists to `.fusaops-vuln.json`; exits 1 when any vulnerability is found. (REQ-FO-CLI071)
- `fusaops capabilities` updated with `vuln` command and `text`/`json` formats.
- Mirrors `gofusa vuln` extended to all supported manifest types across languages.

### Safety
- Requirements registry at 346 requirements (added REQ-FO-VULN001–004, REQ-FO-CLI071)
- 16 package tests + 5 CLI tests; combined coverage ≥ 80%

## [1.55.0] — 2026-06-16

### `fusaops fmea` — Design Failure Mode and Effects Analysis (IEC 61508 / ISO 26262 Part 8)

- **`fmea` package** — `FailureMode`, `FMEA` types; `Build`, `Save`, `Load`, `Render` (text/json). Produces 8 failure modes covering the FuSaOps orchestration pipeline (adapter not installed, adapter crash, schema mismatch, trace annotation loss, SBOM omission, evidence tampering, adapter timeout, suppression abuse). RPN = Severity × Occurrence × Detection (1–10). Items with RPN > 100 are classified as high-priority. SHA-256 integrity hash over the document. (REQ-FO-FMEA001, REQ-FO-FMEA002, REQ-FO-FMEA003, REQ-FO-FMEA004)
- **`fusaops fmea [--dir DIR] [--format text|json] [--output PATH]`** — build and render the FMEA; persists to `.fusaops-fmea.json`; exits 1 when any failure mode has high RPN. (REQ-FO-CLI070)
- `fusaops capabilities` updated with `fmea` command and `text`/`json` formats.
- Mirrors `gofusa fmea` adapted for the FuSaOps multi-language orchestration context.

### Safety
- Requirements registry at 341 requirements (added REQ-FO-FMEA001–004, REQ-FO-CLI070)
- 15 package tests + 5 CLI tests; combined coverage ≥ 80%

## [1.54.0] — 2026-06-16

### `fusaops tara` — Threat Analysis and Risk Assessment (ISO 21434:2021 Ch. 9)

- **`tara` package** — `Impact`, `Feasibility`, `RiskLevel`, `TreatmentDecision`, `ThreatScenario`, `TARA` types; `Build`, `Save`, `Load`, `Render` (text/json). Produces 8 threat scenarios covering the cybersecurity threats to a multi-language safety-analysis toolchain (SBOM tampering, evidence spoofing, adapter supply-chain compromise, report manipulation, signing key theft, traceability integrity, pipeline availability disruption, configuration tampering). Each scenario's risk level is computed from an Impact × Feasibility matrix per ISO 21434 Table 1. SHA-256 integrity hash over the assembled document. (REQ-FO-TARA001, REQ-FO-TARA002, REQ-FO-TARA003, REQ-FO-TARA004)
- **`fusaops tara [--dir DIR] [--format text|json] [--output PATH]`** — build and render the TARA; persists to `.fusaops-tara.json`; exits 1 when any scenario carries a critical risk level. (REQ-FO-CLI069)
- `fusaops capabilities` updated with `tara` command and `text`/`json` formats.
- Mirrors `gofusa tara` adapted for the FuSaOps multi-language orchestration context.

### Safety
- Requirements registry at 336 requirements (added REQ-FO-TARA001–004, REQ-FO-CLI069)
- 15 package tests + 5 CLI tests; combined coverage ≥ 80%

## [1.53.0] — 2026-06-16

### `fusaops sas` — Software Accomplishment Summary (DO-178C §11.20)

- **`sas` package** — `ActivityStatus`, `Activity`, `SAS` types; `Build`, `Save`, `Load`, `Render` (text/json). Maps 12 DO-178C lifecycle activities to FuSaOps evidence artefacts. An activity is complete when its evidence file exists; N/A for activities with no associated artefact. SHA-256 integrity hash over the assembled document. (REQ-FO-SAS001, REQ-FO-SAS002, REQ-FO-SAS003, REQ-FO-SAS004)
- **`fusaops sas [--dir DIR] [--level DAL-A|...|DAL-E] [--format text|json] [--output PATH]`** — build and render the SAS; exits 1 when incomplete activities remain. (REQ-FO-CLI068)
- `fusaops capabilities` updated with `sas` command and `text`/`json` formats.
- Mirrors `gofusa sas` adapted for the FuSaOps multi-language orchestration context.

### Safety
- Requirements registry at 331 requirements (added REQ-FO-SAS001–004, REQ-FO-CLI068)
- 15 package tests + 6 CLI tests; combined coverage 82.2%

## [1.52.0] — 2026-06-16

### `fusaops sci` — Software Configuration Index (DO-178C §11.16)

- **`sci` package** — `ItemKind`, `ConfigItem`, `SCI` types; `Build`, `Save`, `Load`, `Render` (text/json). Inventories software configuration items: the FuSaOps tool itself, each detected x-FuSa adapter (with availability), and 10 known evidence artefacts (with SHA-256 hash and size when present). A SHA-256 integrity hash covers the assembled document. (REQ-FO-SCI001, REQ-FO-SCI002, REQ-FO-SCI003, REQ-FO-SCI004)
- **`fusaops sci [--dir DIR] [--format text|json] [--output PATH]`** — build and render the SCI; persists to `.fusaops-sci.json`; always exits 0. (REQ-FO-CLI067)
- `fusaops capabilities` updated with `sci` command and `text`/`json` formats.
- Mirrors `gofusa sci` adapted for the FuSaOps multi-language orchestration context.

### Safety
- Requirements registry at 326 requirements (added REQ-FO-SCI001–004, REQ-FO-CLI067)
- 14 package tests + 6 CLI tests; combined coverage 82.1%

## [1.51.0] — 2026-06-16

### `fusaops safety-case` — structured safety argument assembly

- **`safetycase` package** — `Standard`, `EvidenceStatus`, `EvidenceRef`, `Claim`, `SafetyCase` types; `Build`, `Save`, `Load`, `Render` (text/json). Assembles a hierarchical safety argument by discovering known evidence artefacts (test bundle, qualify report, SBOM, provenance, manifest, problem log, audit pack) in the project root. Each claim passes when all required evidence is present; a SHA-256 integrity hash is computed over the assembled document. (REQ-FO-SC001, REQ-FO-SC002, REQ-FO-SC003, REQ-FO-SC004)
- **`fusaops safety-case [--dir DIR] [--standard ISO 26262|DO-178C|IEC 61508|ISO 21434] [--format text|json] [--output PATH]`** — build and render the safety case; exits 1 when evidence gaps remain. (REQ-FO-CLI066)
- `fusaops capabilities` updated with `safety-case` command and `text`/`json` formats.
- Mirrors `gofusa safety-case` adapted for the FuSaOps multi-language orchestration context.

### Safety
- Requirements registry at 321 requirements (added REQ-FO-SC001–004, REQ-FO-CLI066)
- 15 package tests; combined coverage 81.9%

## [1.50.0] — 2026-06-16

### `fusaops release` — cross-language SBOM, provenance, and artifact manifest

- **`release` package** — `Provenance`, `Artifact`, `Manifest` types; `BuildProvenance` (captures tool version, Go runtime, git VCS state), `BuildManifest` (SHA-256 hashes of all output files), `SaveJSON`, `RenderProvenance`, `RenderManifest` (text/json). File constants: `sbom.json`, `provenance.json`, `artifact-manifest.json`. (REQ-FO-REL001, REQ-FO-REL002, REQ-FO-REL003, REQ-FO-REL004)
- **`fusaops release [--dir DIR] [--output-dir DIR]`** — run `RunSBOM` across all adapters → `sbom.json`, build provenance → `provenance.json`, hash all outputs → `artifact-manifest.json`. (REQ-FO-CLI065)
- `fusaops capabilities` updated with `release` command and `text` format.
- Mirrors `gofusa release` adapted for the FuSaOps multi-language orchestration context.

### Safety
- Requirements registry at 321 requirements (added REQ-FO-REL001–004, REQ-FO-CLI065)
- 15 package tests; combined coverage 81.4%

## [1.49.0] — 2026-06-16

### `fusaops qualify` — cross-language tool qualification roll-up

- **`qualify` package** — `ComponentResult`, `Report` types; `Run`, `Save`, `Load`, `Render` (text/json). Calls each adapter's `Qualify()` method, aggregates per-component pass/fail counts, and computes a SHA-256 integrity hash. Adapters that are unavailable or do not implement `Qualifier` are recorded as skipped. (REQ-FO-QUAL001, REQ-FO-QUAL002, REQ-FO-QUAL003, REQ-FO-QUAL004)
- **`fusaops qualify [--dir DIR] [--output PATH] [--format text|json]`** — run tool qualification across all applicable adapters, save `.fusaops-qualify-report.json`, print per-component results; exits 1 if any adapter fails. (REQ-FO-CLI064)
- `fusaops capabilities` updated with `qualify` command and `text`/`json` formats.
- Mirrors `gofusa qualify` adapted for the FuSaOps multi-language orchestration context.

### Safety
- Requirements registry at 315 requirements (added REQ-FO-QUAL001–004, REQ-FO-CLI064)
- 15 package tests; combined coverage 81.4%

## [1.48.0] — 2026-06-16

### `fusaops sign` — HMAC-SHA256 artifact signing

- **`sign` package** — `Keygen`, `LoadKey`, `Sign`, `Verify` functions; `SigExt = ".sig"`. Generates random 32-byte keys, computes HMAC-SHA256 over artifact files, writes `.sig` files, and verifies signatures. Stdlib-only, no external tools. (REQ-FO-SIGN001, REQ-FO-SIGN002, REQ-FO-SIGN003, REQ-FO-SIGN004)
- **`fusaops sign --keygen <path>`** — generate a new HMAC key. (REQ-FO-CLI063)
- **`fusaops sign --key <keyfile> <file>`** — sign artifact, write `<file>.sig`. (REQ-FO-CLI063)
- **`fusaops sign --verify --key <keyfile> <file>`** — verify `<file>.sig`. (REQ-FO-CLI063)
- `fusaops capabilities` updated with `sign` command and `text` format.
- Mirrors `gofusa sign` adapted for the FuSaOps multi-language orchestration context.

### Safety
- Requirements registry at 309 requirements (added REQ-FO-SIGN001–004, REQ-FO-CLI063)
- 11 package tests; combined coverage 81.3%

## [1.47.0] — 2026-06-16

### `fusaops verify` — Go test evidence bundle

- **`verify` package** — `TestStatus`, `TestResult`, `Summary`, `Bundle` types; `Parse`, `Summarise`, `Run`, `New`, `Save`, `Load`, `Render` (text/json). Shells out to `go test -json -count=1 ./...`, parses the event stream, and persists an auditable evidence bundle to `.fusaops-evidence.json`. (REQ-FO-VER001, REQ-FO-VER002, REQ-FO-VER003, REQ-FO-VER004, REQ-FO-VER005)
- **`fusaops verify [--dir DIR] [--output PATH] [--format text|json]`** — run the Go test suite and save the evidence bundle; exits 1 if any tests fail. (REQ-FO-CLI062)
- `fusaops capabilities` updated with `verify` command and `text`/`json` formats.
- Mirrors `gofusa verify` adapted for the FuSaOps multi-language orchestration context.

### Safety
- Requirements registry at 303 requirements (added REQ-FO-VER001–005, REQ-FO-CLI062)
- 15 package tests; combined coverage 81.3%

## [1.46.0] — 2026-06-13

### `fusaops pr` — DO-178C §11.17 software problem reports

- **`pr` package** — `ProblemReport`, `Log`, `Phase`, `Status`, `PRSeverity` types; `Load`, `Save`, `Add`, `Close`, `Find`, `Render` (text/json). Manages `.fusaops-problems.json` for multi-language projects. (REQ-FO-PR001, REQ-FO-PR002, REQ-FO-PR003, REQ-FO-PR004)
- **`fusaops pr init [--dir DIR]`** — create an empty `.fusaops-problems.json`. (REQ-FO-CLI061)
- **`fusaops pr add --id ID --title TEXT [--desc TEXT] [--phase development] [--severity minor]`** — add a problem report. (REQ-FO-CLI061)
- **`fusaops pr list [--format text|json]`** — list all problem reports with open/closed counts. (REQ-FO-CLI061)
- **`fusaops pr close --id ID [--resolution TEXT]`** — close a problem report with optional resolution. (REQ-FO-CLI061)
- `fusaops capabilities` updated with `pr` command and `text`/`json` formats.
- Mirrors `gofusa pr` adapted for the FuSaOps multi-language orchestration context.

### Safety
- Requirements registry at 297 requirements (added REQ-FO-PR001, REQ-FO-PR002, REQ-FO-PR003, REQ-FO-PR004, REQ-FO-CLI061)
- 14 package tests + 8 CLI tests; combined coverage 81.9%

## [1.45.0] — 2026-06-13

### `fusaops disposition` — finding disposition management

- **`disposition` package** — `Action`, `Entry`, `Log` types; `Load`, `Save`, `Add`, `Find`, `RenderEntries`. Records accept/fix decisions for findings by rule ID and language in `.fusaops-dispositions.json`. (REQ-FO-DISP001, REQ-FO-DISP002, REQ-FO-DISP003)
- **`fusaops disposition add --rule RULE --reviewer NAME --rationale TEXT [--lang LANG] [--action accept|fix] [--ref REF]`** — record a disposition. (REQ-FO-CLI060)
- **`fusaops disposition list [--dir DIR]`** — list all disposition entries. (REQ-FO-CLI060)
- **`fusaops disposition show --rule RULE [--lang LANG]`** — show a specific disposition. (REQ-FO-CLI060)
- Distinct from `fusaops suppress` which hides findings — dispositions acknowledge and track conscious decisions.
- Mirrors `gofusa disposition` adapted for the multi-language FuSaOps context.

### Safety
- Requirements registry at 292 requirements (added REQ-FO-DISP001, REQ-FO-DISP002, REQ-FO-DISP003, REQ-FO-CLI060)
- 10 package tests + 7 CLI tests; combined coverage 81.7%

## [1.44.0] — 2026-06-13

### `fusaops impact` — cross-language change impact analysis

- **`impact` package** — `FileChange`, `RequirementImpact`, `ArtifactStatus`, `Report` types; `Analyse(projectRoot, fromRef, toRef)` runs git diff, scans `//fusa:req`/`//fusa:test`/`#fusa:req`/`#fusa:test` annotations across Go, C, C++, Rust, Python, Java sources, identifies impacted requirements and stale evidence artefacts; `Render` (text/json). (REQ-FO-IMP001, REQ-FO-IMP002, REQ-FO-IMP003)
- **`fusaops impact [--dir DIR] [--from REF] [--to REF] [--format text|json] [--output FILE]`** — cross-language change impact report. (REQ-FO-CLI059)
- `fusaops capabilities` updated with impact command and text/json formats.
- Mirrors `gofusa impact` adapted for multi-language annotation scanning and FuSaOps evidence artefact staleness checking.

### Safety
- Requirements registry at 288 requirements (added REQ-FO-IMP001, REQ-FO-IMP002, REQ-FO-IMP003, REQ-FO-CLI059)
- 13 package tests + 4 CLI tests; combined coverage 81.8%

## [1.43.0] — 2026-06-13

### `fusaops hooks` — git pre-commit hook management

- **`fusaops hooks install [--dir DIR]`** — writes a pre-commit script running `fusaops check --strict` to `.git/hooks/pre-commit`. Fails if a hook already exists. (REQ-FO-HOOKS001, REQ-FO-CLI058)
- **`fusaops hooks remove [--dir DIR]`** — removes the FuSaOps pre-commit hook. (REQ-FO-HOOKS001)
- **`fusaops hooks show`** — prints the hook script to stdout. (REQ-FO-HOOKS001)
- `fusaops capabilities` updated with hooks command.
- Mirrors `gofusa hooks` for gating commits on the multi-language safety check.

### Safety
- Requirements registry at 283 requirements (added REQ-FO-HOOKS001, REQ-FO-CLI058)
- 5 CLI tests; combined coverage 81.7%

## [1.42.0] — 2026-06-13

### `fusaops slsa` — SLSA supply-chain gap report

- **`slsa` package** — `Level`, `Objective`, `Report` types; `Assess(projectRoot, project, level)` evaluates 10 SLSA v1.0 objectives across L1–L4 (version control, module file, provenance, SBOM, CODEOWNERS, artifact integrity, evidence bundle); `Render` (text/json). (REQ-FO-SLSA001, REQ-FO-SLSA002, REQ-FO-SLSA003)
- **`fusaops slsa [--dir DIR] [--level L1|L2|L3|L4] [--format text|json] [--output FILE]`** — multi-language SLSA gap report; exits 1 when gaps remain. (REQ-FO-CLI057)
- `fusaops capabilities` updated with slsa command and text/json formats.
- Mirrors `gofusa slsa` adapted for the FuSaOps multi-language project context.

### Safety
- Requirements registry at 281 requirements (added REQ-FO-SLSA001, REQ-FO-SLSA002, REQ-FO-SLSA003, REQ-FO-CLI057)
- 11 package tests + 6 CLI tests; combined coverage 81.7%

## [1.41.0] — 2026-06-13

### `fusaops badge` — SVG status badge from aggregate report

- **`badge` package** — `Badge` and `Status` types (Pass/Warn/Fail); `New(errors, warnings, version)` derives status; `Render(w, badge)` writes a Shields.io-style SVG with label `fusaops`. (REQ-FO-BADGE001, REQ-FO-BADGE002)
- **`fusaops badge [--output FILE] [report.json]`** — reads an aggregate check report JSON from a file or stdin and writes the SVG badge to stdout or a file. (REQ-FO-CLI056)
- `fusaops capabilities` commands and formats maps updated to include `badge`.
- Mirrors `gofusa badge` for embedding a colour-coded health indicator in README files and CI artefacts.

### Safety
- Requirements registry at 276 requirements (added REQ-FO-BADGE001, REQ-FO-BADGE002, REQ-FO-CLI056)
- 5 package tests + 5 CLI tests; combined coverage 81.7%

## [1.40.0] — 2026-06-13

### `fusaops metrics` — safety metrics time series

- **`metrics` package** — `Snapshot` and `TimeSeries` types; `Load`, `Save`, `Append`, `Collect`, and `Render` (text/json) functions. `Collect` reads `check-report.json`, `.fusa-reqs.json`, and `coverage-report.json` to populate error/warning/info counts, requirement count, and coverage percent. (REQ-FO-MET001, REQ-FO-MET002, REQ-FO-MET003)
- **`fusaops metrics record`** — collect a snapshot from project artefacts and append to `.fusaops-metrics.json`. (REQ-FO-CLI055)
- **`fusaops metrics show [--format text|json] [--output FILE]`** — display the full metrics time series. (REQ-FO-CLI055)
- Mirrors `gofusa metrics` for CI-level safety KPI tracking across releases.

### Safety
- Requirements registry at 273 requirements (added REQ-FO-MET001, REQ-FO-MET002, REQ-FO-MET003, REQ-FO-CLI055)
- 12 package tests + 6 CLI tests; combined coverage 81.7%

## [1.39.0] — 2026-06-13

### `fusaops coverage --format markdown`

- **`coverage.Render(w, rep, "markdown")`** (alias `"md"`) — new GFM format: header with 🟢/🟡/🔴 badge and statement %, summary table (Statement / Decision / MC/DC with required/YES indicators), coverage gaps table for files below 100%, decision note in italic. (REQ-FO-COV003)
- **`fusaops coverage --format markdown [--output FILE]`** — expose from CLI; `fusaops capabilities` format map updated to include `"markdown"`.
- Enables embedding DO-178C coverage reports directly in PR comments and GitHub Actions job summaries.

### Safety
- Requirements registry at 269 requirements (REQ-FO-COV003 description updated)
- 3 new tests; combined coverage 82.0%

## [1.38.0] — 2026-06-13

### `fusaops capabilities` — §9.1 discovery document

- **`fusaops capabilities`** — new subcommand emitting a §9.1 x-FuSa discovery document: `kind: "capabilities"` with `tool`, `toolVersion`, `specVersion`, `commands` list, per-command `formats` map, and `standards` list. (REQ-FO-CLI054)
- Only JSON format supported (per spec §9.1); `--format text` returns exit 2.
- Mirrors `gofusa capabilities` for machine-readable discovery by CI tooling, IDEs, and the FuSaOps conform checker.

### Safety
- Requirements registry at 269 requirements (added REQ-FO-CLI054)
- 2 new tests; combined coverage 81.6%

## [1.37.0] — 2026-06-13

### `fusaops version --format json` and `SpecVersion` constant

- **`fusaops.SpecVersion`** — new exported constant `"1.10.4"` identifying the x-FuSa spec version FuSaOps targets. (REQ-FO-CORE007)
- **`fusaops version --format json`** — emits `{"tool":"fusaops","version":"1.37.0","specVersion":"1.10.4"}`; default format remains `text`. (REQ-FO-CLI053)
- Mirrors `gofusa version --format json` for machine-readable version introspection in CI and scripts.

### Safety
- Requirements registry at 268 requirements (added REQ-FO-CORE007, REQ-FO-CLI053)
- 2 new tests; combined coverage 81.6%

## [1.36.0] — 2026-06-13

### `fusaops req` — requirement show/import/export

- **`req` package** — `Entry` struct, `LoadRegistry`, `SaveRegistry`; CSV import/export (`ParseCSV`, `RenderCSV`); XML import/export for DOORS ReqIF, Polarion, Codebeamer, and Jama Connect. (REQ-FO-REQ001, REQ-FO-REQ002, REQ-FO-REQ003)
- **`fusaops req [REQ-ID ...]`** — show requirements from `.fusa-reqs.json`; optional ID filter; displays title, text/description, standard, level, priority. (REQ-FO-CLI052)
- **`fusaops req import --format csv|doors|polarion|codebeamer|jama --file FILE`** — import requirements, skipping duplicates; prints `Imported N requirements (M skipped as duplicates)`.
- **`fusaops req export --format csv|doors|polarion|codebeamer|jama [--output FILE]`** — export requirements to the requested format, defaulting to stdout.
- Mirrors `gofusa req show/import/export`; enables bidirectional sync with safety tool databases.

### Safety
- Requirements registry at 266 requirements (added REQ-FO-REQ001, REQ-FO-REQ002, REQ-FO-REQ003, REQ-FO-CLI052)
- 23 new tests; combined coverage 81.6%

## [1.35.0] — 2026-06-13

### `fusaops coverage` — DO-178C structural coverage report

- **`coverage` package** — new `Parse`, `Analyse`, `BuildFromFile`, and `Render` functions that read a standard Go `coverage.out` profile and compute a DO-178C structural coverage report: statement coverage %, decision coverage % (approximated from block hit ratio), MC/DC requirement flag (DAL-A only), per-file breakdown, and gap list. (REQ-FO-COV001, REQ-FO-COV002, REQ-FO-COV003)
- **`fusaops coverage [flags] [coverage.out]`** — new CLI subcommand mirroring `gofusa coverage`; accepts `--dal DAL-A|DAL-B|DAL-C|DAL-D` (default DAL-B), `--format text|json`, `--output FILE`, and `--dir DIR`. (REQ-FO-CLI051)
- Enables FuSaOps' own ASIL-C qualification evidence; generate a profile with `go test -coverprofile=coverage.out ./...` then run `fusaops coverage`.

### Safety
- Requirements registry at 262 requirements (added REQ-FO-COV001, REQ-FO-COV002, REQ-FO-COV003, REQ-FO-CLI051)
- 18 new tests; combined coverage 82.1%

## [1.34.0] — 2026-06-13

### `fusaops adapters --format json` — Machine-readable adapter list

- **`fusaops adapters --format json`** — new format flag; emits a JSON array of objects with `name`, `tool`, `language`, and `available` fields. Default format remains `text`. (REQ-FO-CLI050)
- Lets CI scripts and tooling introspect installed adapters programmatically without parsing human-readable text.

### Safety
- Requirements registry at 258 requirements (added REQ-FO-CLI050)
- 1 new test; combined coverage 81.6%

## [1.33.0] — 2026-06-13

### `fusaops sbom --format markdown` — SBOM markdown report

- **`sbom.Render(w, a, "markdown")`** (alias `"md"`) — new format support; produces a GFM markdown page with project metadata, a Components table (Tool / Language / Module / Packages, with skipped-component inline note), and a de-duplicated Packages table (Name / Version / Language). Pipe characters in package names are escaped. (REQ-FO-SBM011)
- **`fusaops sbom --format markdown [--output FILE]`** — expose the new format from the CLI.
- Useful for embedding SBOM summaries in pull request comments and GitHub Actions job summaries.

### Safety
- Requirements registry at 257 requirements (added REQ-FO-SBM011)
- 3 new tests; combined coverage 81.6%

## [1.32.0] — 2026-06-13

### `fusaops <standard> --format markdown` — Standards markdown report

- **`standards.Render(w, agg, "markdown")`** (alias `"md"`) — new format support; produces a GFM markdown page with a status badge (🟢 PASS / 🔴 GAP), per-component sections (tool/language heading, satisfied/partial/gap counts, objective table with ID / Status emoji / Title+Clause / Evidence), skipped/nil-report fallbacks, and pipe-escaped titles. (REQ-FO-STD012)
- **All six standards commands now accept `--format markdown [--output FILE]`**: `fusaops iso26262`, `fusaops iec61508`, `fusaops do178`, `fusaops iso21434`, `fusaops unece`, `fusaops iec62443`.
- Useful for embedding standards gap-report summaries in pull request comments and GitHub Actions job summaries.

### Safety
- Requirements registry at 256 requirements (added REQ-FO-STD012)
- 4 new tests; combined coverage 81.5%

## [1.31.0] — 2026-06-13

### `fusaops conform --format markdown` — Conformance markdown report

- **`conform.Render(w, rep, "markdown")`** (alias `"md"`) — new format support; produces a GFM markdown page with a status badge (🟢 PASS / 🔴 FAIL), pass/fail/skip counts, and a per-check GFM table (Result / Level / Section / Check with detail). Pipe characters are escaped to avoid breaking GFM table syntax. (REQ-FO-CNF019)
- **`fusaops conform <binary> --format markdown [--output FILE]`** — expose the new format from the CLI.
- Useful for posting x-FuSa tool conformance check summaries into pull request comments and GitHub Actions job summaries.

### Safety
- Requirements registry at 255 requirements (added REQ-FO-CNF019)
- 3 new tests; combined coverage 81.4%

## [1.30.0] — 2026-06-13

### `fusaops policy --format markdown` — Policy markdown report

- **`policy.Render(w, pr, "markdown")`** (alias `"md"`) — new format support; produces a GFM markdown page with a status badge (🟢 PASS / 🔴 FAIL), passed/failed counts, and a per-rule GFM table (Result / Rule / Message). Pipe characters in messages are escaped as `\|` to avoid breaking GFM table syntax. (REQ-FO-POL006)
- **`fusaops policy --format markdown [--output FILE]`** — expose the new format from the CLI.
- Useful for posting policy gate results directly into pull request comments and GitHub Actions job summaries.

### Safety
- Requirements registry at 254 requirements (added REQ-FO-POL006)
- 4 new tests; combined coverage 81.3%

## [1.29.0] — 2026-06-13

### `fusaops fleet --format markdown` — Fleet markdown report

- **`fleet.Render(w, fr, "markdown")`** (alias `"md"`) — new format support; produces a GFM markdown page with a status badge (🟢 PASS / 🟡 WARN / 🔴 FAIL), a per-repo table (Repository / Status / Errors / Warnings / Infos / Total), and a TOTAL summary row. Scan errors appear inline in the table. (REQ-FO-FLT007)
- **`fusaops fleet --format markdown [--output FILE]`** — expose the new format from the CLI.
- Useful for posting multi-repo safety status summaries directly into pull request comments and GitHub Actions job summaries.

### Safety
- Requirements registry at 253 requirements (added REQ-FO-FLT007)
- 4 new tests; combined coverage 81.2%

## [1.28.0] — 2026-06-13

### `--workers` and `--timeout` CLI flags on check/report/trace/sbom/audit-pack

- **`fusaops check|report|trace|sbom|audit-pack --workers N`** — cap the number of concurrent adapters. `0` means unlimited (the default). Overrides `run.workers` in `.fusaops.json`. (REQ-FO-CLI049)
- **`fusaops check|report|trace|sbom|audit-pack --timeout DURATION`** — set a per-adapter deadline (e.g. `30s`, `5m`). Overrides `run.timeout` in `.fusaops.json`. An invalid duration exits 2 with a descriptive error. (REQ-FO-CLI049)
- Useful in CI where adapter time limits differ per environment without needing a committed config change.

### Safety
- Requirements registry at 252 requirements (added REQ-FO-CLI049)
- 9 new tests covering workers/timeout acceptance and invalid-timeout rejection on all five commands; combined coverage 81.2%

## [1.27.0] — 2026-06-13

### `fusaops iso26262|iec61508|do178|iso21434|unece|iec62443 --format html` — Standards HTML report

- **`standards.Render(w, agg, "html")`** — new format support; produces a self-contained HTML page with a per-component section (header showing tool/language, satisfied/partial/gap counts, and an objective table with colour-coded status), fallback text for skipped/nil components. No external CSS or JS. (REQ-FO-STD011)
- **All six standards commands now accept `--format html [--output FILE]`**.
- Applies to: `fusaops iso26262`, `fusaops iec61508`, `fusaops do178`, `fusaops iso21434`, `fusaops unece`, `fusaops iec62443`.

### Safety
- Requirements registry at 251 requirements (added REQ-FO-STD011)
- 2 new tests; 251/251 requirements traced+tested; combined coverage 81.3%

## [1.26.0] — 2026-06-13

### `fusaops conform --format html` — Conformance HTML report

- **`conform.Render(w, rep, "html")`** — new format support; produces a self-contained HTML page with a PASS/FAIL badge, pass/fail/skip counts in the header, and a per-check results table (result/level/section/name with detail inline). No external CSS or JS. (REQ-FO-CNF018)
- **`fusaops conform <binary> --format html [--output conform.html]`** — expose the new format from the CLI.
- Format validation in cmd_conform.go updated to allow `"html"`.

### Safety
- Requirements registry at 250 requirements (added REQ-FO-CNF018)
- 2 new tests; 250/250 requirements traced+tested; combined coverage 81.3%

## [1.25.0] — 2026-06-13

### `fusaops policy --format html` — Policy HTML report

- **`policy.Render(w, pr, "html")`** — new format support; produces a self-contained HTML page with an overall PASS/FAIL badge, passed/failed counts, and a per-rule results table (result/rule-ID/message). No external CSS or JS. (REQ-FO-POL005)
- **`fusaops policy --format html [--output policy.html]`** — expose the new format from the CLI.
- Error message for unknown format updated from `"want text or json"` → `"want text, json, or html"`.

### Safety
- Requirements registry at 249 requirements (added REQ-FO-POL005)
- 2 new tests; 249/249 requirements traced+tested; combined coverage 81.3%

## [1.24.0] — 2026-06-13

### `fusaops sbom --format html` — SBOM HTML viewer

- **`sbom.Render(w, a, "html")`** — new format support; produces a self-contained HTML page with a component summary table (tool/language/module/package count, skipped rows) and a full de-duplicated package table (name/version/language). No external CSS or JS. (REQ-FO-SBM010)
- **`fusaops sbom --format html [--output sbom.html]`** — expose the new format from the CLI.
- Completes the SBOM format set to match the single-project report and trace HTML renderers.

### Safety
- Requirements registry at 248 requirements (added REQ-FO-SBM010)
- 2 new tests; 248/248 requirements traced+tested; combined coverage 81.3%

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

[Unreleased]: https://github.com/SoundMatt/FuSaOps/compare/v1.59.0...HEAD
[1.59.0]: https://github.com/SoundMatt/FuSaOps/compare/v1.58.0...v1.59.0
[1.58.0]: https://github.com/SoundMatt/FuSaOps/compare/v1.57.0...v1.58.0
[1.57.0]: https://github.com/SoundMatt/FuSaOps/compare/v1.56.0...v1.57.0
[1.56.0]: https://github.com/SoundMatt/FuSaOps/compare/v1.55.0...v1.56.0
[1.55.0]: https://github.com/SoundMatt/FuSaOps/compare/v1.54.0...v1.55.0
[1.54.0]: https://github.com/SoundMatt/FuSaOps/compare/v1.53.0...v1.54.0
[1.53.0]: https://github.com/SoundMatt/FuSaOps/compare/v1.52.0...v1.53.0
[1.52.0]: https://github.com/SoundMatt/FuSaOps/compare/v1.51.0...v1.52.0
[1.51.0]: https://github.com/SoundMatt/FuSaOps/compare/v1.50.0...v1.51.0
[1.50.0]: https://github.com/SoundMatt/FuSaOps/compare/v1.49.0...v1.50.0
[1.49.0]: https://github.com/SoundMatt/FuSaOps/compare/v1.48.0...v1.49.0
[1.48.0]: https://github.com/SoundMatt/FuSaOps/compare/v1.47.0...v1.48.0
[1.47.0]: https://github.com/SoundMatt/FuSaOps/compare/v1.46.0...v1.47.0
[1.46.0]: https://github.com/SoundMatt/FuSaOps/compare/v1.45.0...v1.46.0
[1.45.0]: https://github.com/SoundMatt/FuSaOps/compare/v1.44.0...v1.45.0
[1.44.0]: https://github.com/SoundMatt/FuSaOps/compare/v1.43.0...v1.44.0
[1.43.0]: https://github.com/SoundMatt/FuSaOps/compare/v1.42.0...v1.43.0
[1.42.0]: https://github.com/SoundMatt/FuSaOps/compare/v1.41.0...v1.42.0
[1.41.0]: https://github.com/SoundMatt/FuSaOps/compare/v1.40.0...v1.41.0
[1.40.0]: https://github.com/SoundMatt/FuSaOps/compare/v1.39.0...v1.40.0
[1.39.0]: https://github.com/SoundMatt/FuSaOps/compare/v1.38.0...v1.39.0
[1.38.0]: https://github.com/SoundMatt/FuSaOps/compare/v1.37.0...v1.38.0
[1.37.0]: https://github.com/SoundMatt/FuSaOps/compare/v1.36.0...v1.37.0
[1.36.0]: https://github.com/SoundMatt/FuSaOps/compare/v1.35.0...v1.36.0
[1.35.0]: https://github.com/SoundMatt/FuSaOps/compare/v1.34.0...v1.35.0
[1.34.0]: https://github.com/SoundMatt/FuSaOps/compare/v1.33.0...v1.34.0
[1.33.0]: https://github.com/SoundMatt/FuSaOps/compare/v1.32.0...v1.33.0
[1.32.0]: https://github.com/SoundMatt/FuSaOps/compare/v1.31.0...v1.32.0
[1.31.0]: https://github.com/SoundMatt/FuSaOps/compare/v1.30.0...v1.31.0
[1.30.0]: https://github.com/SoundMatt/FuSaOps/compare/v1.29.0...v1.30.0
[1.29.0]: https://github.com/SoundMatt/FuSaOps/compare/v1.28.0...v1.29.0
[1.28.0]: https://github.com/SoundMatt/FuSaOps/compare/v1.27.0...v1.28.0
[1.27.0]: https://github.com/SoundMatt/FuSaOps/compare/v1.26.0...v1.27.0
[1.26.0]: https://github.com/SoundMatt/FuSaOps/compare/v1.25.0...v1.26.0
[1.25.0]: https://github.com/SoundMatt/FuSaOps/compare/v1.24.0...v1.25.0
[1.24.0]: https://github.com/SoundMatt/FuSaOps/compare/v1.23.0...v1.24.0
[1.23.0]: https://github.com/SoundMatt/FuSaOps/compare/v1.22.0...v1.23.0
[1.22.0]: https://github.com/SoundMatt/FuSaOps/compare/v1.21.0...v1.22.0
[1.21.0]: https://github.com/SoundMatt/FuSaOps/compare/v1.20.0...v1.21.0
[1.20.0]: https://github.com/SoundMatt/FuSaOps/compare/v1.19.0...v1.20.0
[1.19.0]: https://github.com/SoundMatt/FuSaOps/compare/v1.18.0...v1.19.0
[1.18.0]: https://github.com/SoundMatt/FuSaOps/compare/v1.17.0...v1.18.0
[1.17.0]: https://github.com/SoundMatt/FuSaOps/compare/v1.16.0...v1.17.0
[1.16.0]: https://github.com/SoundMatt/FuSaOps/compare/v1.15.0...v1.16.0
[1.15.0]: https://github.com/SoundMatt/FuSaOps/compare/v1.14.0...v1.15.0
[1.14.0]: https://github.com/SoundMatt/FuSaOps/compare/v1.13.0...v1.14.0
[1.13.0]: https://github.com/SoundMatt/FuSaOps/compare/v1.12.0...v1.13.0
[1.12.0]: https://github.com/SoundMatt/FuSaOps/compare/v1.11.0...v1.12.0
[1.11.0]: https://github.com/SoundMatt/FuSaOps/compare/v1.10.0...v1.11.0
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
