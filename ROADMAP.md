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

## v0.2 — Evidence Aggregation

**Goal:** Aggregate more than findings.

- Merge per-language traceability (`trace`) into a cross-language matrix
- Aggregate coverage and qualification status per component
- Unified audit-pack: bundle every component's evidence into one ZIP
- Aggregate SBOM across languages (SPDX)

Deliverables: `fusaops trace`, `fusaops audit-pack`, `fusaops sbom`

---

## v0.3 — Standards Roll-up

**Goal:** One compliance view across languages.

- Roll up ISO 26262 / IEC 61508 / DO-178C gap reports from each component
- Per-standard PASS/GAP matrix spanning all languages
- Project-level safety case assembling component safety cases (GSN)

Deliverables: `fusaops iso26262`, `fusaops do178`, `fusaops safety-case`

---

## v0.4 — Monorepo & Component Model

**Goal:** First-class support for large mixed-language monorepos.

- Per-directory component pinning in `.fusaops.json`
- Parallel adapter execution with per-component timeouts
- Baseline + diff gating across the whole repo (regression CI gate)

Deliverables: `fusaops diff`, component-scoped scans

---

## v0.5 — Distribution & Dashboards

**Goal:** Team-facing reporting.

- Historical metrics trend across runs
- Persisted dashboard (store reports; compare over time)
- GitHub Action and prebuilt language-bundle images (Go+C+C++)

---

## Future / Advanced

| Version | Capability |
|---|---|
| v0.6 | Additional language adapters (Rust, Ada/SPARK, Python) |
| v0.7 | REST API + multi-repo aggregation (fleet view) |
| v0.8 | Policy engine — org-wide rules over aggregated findings |
| v1.0 | Enterprise: auth, persistence, multi-tenant dashboards |

---

## Adding a language adapter

1. Implement `adapter.Adapter` (Name, Language, Tool, Detect, Available, Check).
2. For tools emitting the standard `check --format json` schema, embed
   `cmdAdapter` and supply the binary name plus source extensions.
3. Register it in an `init()` with `Default.MustRegister`.
4. Add detection extensions to `scan.langExtensions`.
