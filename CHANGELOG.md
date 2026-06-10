# Changelog

All notable changes to FuSaOps are documented here.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.0.0/)
and the project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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

[Unreleased]: https://github.com/SoundMatt/FuSaOps/compare/v0.2.0...HEAD
[0.2.0]: https://github.com/SoundMatt/FuSaOps/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/SoundMatt/FuSaOps/releases/tag/v0.1.0
