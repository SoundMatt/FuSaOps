# Changelog

All notable changes to FuSaOps are documented here.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.0.0/)
and the project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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

[Unreleased]: https://github.com/SoundMatt/FuSaOps/compare/v0.5.0...HEAD
[0.5.0]: https://github.com/SoundMatt/FuSaOps/compare/v0.4.0...v0.5.0
[0.4.0]: https://github.com/SoundMatt/FuSaOps/compare/v0.3.0...v0.4.0
[0.3.0]: https://github.com/SoundMatt/FuSaOps/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/SoundMatt/FuSaOps/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/SoundMatt/FuSaOps/releases/tag/v0.1.0
