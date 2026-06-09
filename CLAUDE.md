# CLAUDE.md — working on FuSaOps

Guidance for Claude Code (and contributors) working in this repository.

## What FuSaOps is

FuSaOps is the **multi-language orchestration layer** over the x-FuSa toolchain
(go-FuSa, c-FuSa, cpp-FuSa, future tools). It does **not** implement
language-specific safety rules. It detects the languages in a repo, runs each
language's x-FuSa tool, normalises the machine-readable output, and aggregates
everything into one report and one web dashboard.

When extending FuSaOps, keep that boundary: language analysis belongs in the
per-language tools; FuSaOps does discovery, orchestration, aggregation, and
presentation.

## Architecture

| Package | Responsibility |
|---|---|
| `.` (`fusaops`) | Core types: `Severity`, `Language`, `Finding`, sentinel errors, `Version` |
| `config/` | `.fusaops.json` (optional; zero-config by default) |
| `adapter/` | `Adapter` interface, registry, generic `cmdAdapter`, go/c/cpp adapters |
| `scan/` | Language detection with file counts |
| `orchestrator/` | Runs applicable+installed adapters, records skipped components, aggregates |
| `report/` | `AggregateReport` + text/json/html/sarif renderers |
| `server/` | Web dashboard + JSON API (`fusaops serve`) |
| `cmd/fusaops/` | CLI dispatch + subcommands |

## Conventions (mirrors go-FuSa)

- **No external runtime dependencies** — standard library only. The web UI is
  server-rendered Go with inlined assets. Do not add third-party modules.
- **Requirement annotations** — every exported behaviour carries a
  `//fusa:req REQ-FO-...` comment; tests carry `//fusa:test REQ-FO-...`
  (one ID per line). Every requirement is registered in `.fusa-reqs.json`.
- **Coverage gate** — keep `go test` coverage at or above **80%**.
- **Adapters are testable without their tool** — inject a fake `runnerFunc`;
  never require a real binary on PATH in unit tests.
- **Determinism** — tests must pass whether or not the adapter tools are
  installed (an installed tool may legitimately change a check's exit code).

## Common commands

```bash
make build test cover vet lint   # dev loop
make selfcheck                   # gofusa check on FuSaOps's own Go source
gofusa trace --dir .             # requirement traceability matrix
```

## Standards

FuSaOps is itself developed as an ISO 26262 ASIL-C tool (`.fusa.json`,
`.fusa-hara.json`). See `docs/tool-safety-manual.md` and `docs/qualification.md`.
It aggregates evidence relevant to ISO 26262, IEC 61508, ISO 21434, and DO-178C
across the languages it orchestrates.

## Adding a language

See `docs/extending.md`: implement `adapter.Adapter`, register it, add the scan
extension and a fake-runner test, add the Dockerfile tool stage, and wire the
release notification. No FuSaOps recompile is needed for tool *version* updates —
the all-in-one image refreshes via `tools-monitor.yml`.
