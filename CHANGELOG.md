# Changelog

All notable changes to FuSaOps are documented here.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.0.0/)
and the project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.0] — 2026-06-09

### Added
- Multi-language orchestration core
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
- go-FuSa self-check job in CI (FuSaOps gates its own Go source with go-FuSa)
- Docker image bundling `gofusa`; `docker-compose.yml` for the dashboard
- CI: Go 1.22/1.23 × Ubuntu/macOS/Windows matrix, 80% coverage gate,
  golangci-lint, DCO sign-off, Docker build smoke test
- `//fusa:req` requirement annotations throughout the source

[Unreleased]: https://github.com/SoundMatt/FuSaOps/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/SoundMatt/FuSaOps/releases/tag/v0.1.0
