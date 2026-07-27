# CLAUDE.md — working on FuSaOps

Guidance for Claude Code (and contributors) working in this repository.

## What FuSaOps is

FuSaOps is the **multi-language orchestration layer** over the x-FuSa toolchain
(go-FuSa, c-FuSa, cpp-FuSa, rust-FuSa, py-FuSa, java-FuSa — six languages today,
with more addable via the adapter interface). It does **not** implement
language-specific safety rules. It detects the languages in a repo, runs each
language's x-FuSa tool, normalises the machine-readable output, and aggregates
everything into one report and one web dashboard.

When extending FuSaOps, keep that boundary: language analysis belongs in the
per-language tools; FuSaOps does discovery, orchestration, aggregation, and
presentation.

## The x-FuSa contract

[`docs/x-fusa-spec.md`](docs/x-fusa-spec.md) is the **master specification** every
x-FuSa tool builds on — CLI surface, JSON output schemas, file/naming
conventions, exit codes. It is a superset of the three tools, with go-FuSa as the
canonical reference. **FuSaOps' decoders in `report/`, `trace/`, `sbom/`,
`auditpack/` are the authoritative implementation of that spec — keep the spec
and those structs in lock-step.** When a tool's output disagrees with the spec,
the tool is wrong, not FuSaOps.

## Architecture

Core orchestration pipeline:

| Package | Responsibility |
|---|---|
| `.` (`fusaops`) | Core types: `Severity`, `Language`, `Finding`, sentinel errors, `Version` |
| `config/` | `.fusaops.json` (optional; zero-config by default) |
| `adapter/` | `Adapter` interface, registry, generic `cmdAdapter`, per-language adapters (go/c/cpp/rust/py/java) |
| `scan/` | Language detection with file counts |
| `orchestrator/` | Runs applicable+installed adapters, records skipped components, aggregates; `RunTrace`/`RunSBOM`/`RunAuditPack` roll-ups |
| `report/` | `AggregateReport` + text/json/html/sarif renderers |
| `trace/` | Cross-language requirement traceability + qualification roll-up (text/json/html) |
| `sbom/` | Cross-language SBOM merge + native JSON / SPDX 2.3 renderers |
| `auditpack/` | Unified evidence ZIP bundler with hashed manifest |
| `server/` | Web dashboard + JSON API (`fusaops serve`) |
| `cmd/fusaops/` | CLI dispatch + subcommands |

Compliance, evidence, and workflow packages layered on top of the pipeline:

| Package | Responsibility |
|---|---|
| `badge/` | Shields.io-compatible SVG status badges from an aggregate report |
| `comp/` | McCabe cyclomatic complexity (V(G)) decode + aggregate |
| `conform/` | x-FuSa spec conformance checks against a tool binary |
| `coverage/` | Go coverage profiles → DO-178C-style structural coverage report |
| `diff/` | Compares two `fusaops check` runs to detect new/resolved findings |
| `disposition/` | Finding disposition entries (accept/defer/waive) per project |
| `doctemplate/` | Safety documentation template generation |
| `fleet/` | Multi-repository safety analysis orchestration |
| `fmea/` | Design Failure Mode and Effects Analysis (dFMEA) generation |
| `hara/` | Hazard Analysis and Risk Assessment (HARA) data management |
| `history/` | Persists/retrieves run snapshots so the dashboard can show trends |
| `impact/` | Change-impact analysis of source edits on requirements/tests |
| `mcdc/` | Modified Condition/Decision Coverage (MC/DC) decode + aggregate |
| `metrics/` | Project safety metrics tracked over time |
| `policy/` | Org-wide safety rule evaluation over an aggregated report |
| `pr/` | DO-178C §11.17 Software Problem Report workflow |
| `qualify/` | Per-tool qualification report aggregation |
| `release/` | Build provenance and artifact manifest generation |
| `req/` | Requirement registry (`.fusa-reqs.json`) |
| `safetycase/` | Structured safety argument assembly |
| `sas/` | Software Accomplishment Summary (DO-178C §11.20) |
| `sci/` | Software Configuration Index (DO-178C §11.16) |
| `sign/` | HMAC-SHA256 file signing for FuSaOps artifacts |
| `slsa/` | SLSA (Supply-chain Levels for Software Artifacts) provenance |
| `standards/` | §9.3 gap-report roll-up across standards (ISO 26262, IEC 61508, ISO 21434, DO-178C) |
| `suppression/` | Filters findings acknowledged in a project's suppression list |
| `tara/` | Threat Analysis and Risk Assessment (TARA) per ISO 21434 |
| `verify/` | Test evidence collection |
| `vuln/` | Dependency manifest discovery + vulnerability findings across languages |
| `vv/` | V&V (verification and validation) independence declarations |

Adapter **capability interfaces** (`adapter/capabilities.go`): `Tracer`,
`Qualifier`, `SBOMer`, `Packer` are optional — the orchestrator type-asserts for
them, so a tool that cannot produce an artefact is recorded as skipped, never
fatal. Every `cmdAdapter` implements all four by shelling out to its tool's
matching subcommand (`trace`, `qualify`, `release`, `audit-pack`).

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
