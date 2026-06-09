# Changelog

All notable changes to FuSaOps are documented here.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.0.0/)
and the project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Docker quickstart: all-in-one image (`ghcr.io/soundmatt/fusaops`) that bundles
  the x-FuSa tools by copying each binary from its published image
  (`COPY --from=ghcr.io/soundmatt/go-fusa:latest`), so `docker run fusaops check`
  scans without any local installs and without a Docker socket
- `tools-monitor.yml` workflow: `repository_dispatch` (`xfusa-released`) +
  weekly schedule rebuild `fusaops:latest` with `pull: true`, refreshing bundled
  tools with no manual rebuild or FuSaOps release
- `docs/extending.md`: one-line process for bundling a future x-FuSa, including
  the release-notification snippet for tool repos
- CI Docker job logs in to GHCR and smoke-tests both `fusaops` and bundled
  `gofusa`; docker-publish/build use `pull: true` to bundle the newest tools

### Changed
- Dockerfile no longer `go install`s gofusa; it copies the binary from the
  go-FuSa image (alpine base, `linux/amd64`). cpp/c tool stages are present as
  commented one-liners pending their published images
- `docker-compose.yml` defaults to the published image

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
