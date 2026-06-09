# FuSaOps

**Multi-language functional safety orchestration.**

FuSaOps sits on top of the x-FuSa toolchain — [go-FuSa](https://github.com/SoundMatt/go-FuSa),
[c-FuSa](https://github.com/SoundMatt/c-FuSa), [cpp-FuSa](https://github.com/SoundMatt/cpp-FuSa)
and future language tools — and gives mixed-language repositories a single,
intuitive way to **scan**, **aggregate**, and **report** functional safety
evidence. One command runs the right tool for every language present and merges
their results into one report and one web dashboard.

FuSaOps does **not** reimplement language-specific safety rules. It detects the
languages in a repo, delegates to each language's x-FuSa tool, normalises the
machine-readable output, and presents a unified PASS/WARN/FAIL view for ISO 26262,
IEC 61508, ISO 21434, DO-178C and related safety cases.

> It is **NOT** a certification product. It is an engineering accelerator that
> reduces the cost of producing functional safety evidence across a polyglot
> codebase.

---

## How it works

```
            ┌──────────────────────────── FuSaOps ────────────────────────────┐
   repo ──▶ │  scan (detect languages) ──▶ orchestrator ──▶ aggregate report  │ ──▶ text / json / html / sarif
            │                                   │                              │ ──▶ web dashboard (fusaops serve)
            └───────────────────────────────────┼──────────────────────────────┘
                                                 │
                    ┌────────────────────────────┼────────────────────────────┐
                    ▼                            ▼                            ▼
                 gofusa (Go)                 cfusa (C)                  cpfusa (C++)
            check --format json         check --format json         check --format json
```

Each adapter runs `<tool> check --format json`, FuSaOps decodes the common
report schema, tags every finding with its language and tool, and merges them.
Adapters whose language is present but whose binary is not installed are
recorded as **skipped** components — coverage gaps are never silently dropped.

## Install

```bash
go install github.com/SoundMatt/FuSaOps/cmd/fusaops@latest
```

The adapter tools must be on `PATH` for the languages you want scanned
(`gofusa`, `cfusa`, `cpfusa`). The Docker image bundles `gofusa`.

## Usage

```bash
fusaops scan                 # detect languages and applicable adapters
fusaops adapters             # list adapters and whether each tool is installed
fusaops check                # run every applicable tool; exit 1 on ERROR findings
fusaops check --strict       # also exit 1 on WARNING findings
fusaops report --format html --output fusaops-report.html
fusaops serve --addr :8080   # launch the web dashboard
fusaops init                 # write a starter .fusaops.json
```

### Web dashboard

```bash
fusaops serve
# open http://localhost:8080
```

The dashboard shows an overall status badge, per-language summary cards, and a
filterable findings table. JSON is available at `/api/report`; `/refresh`
re-runs the scan. The page is fully self-contained (no external assets).

### Docker

```bash
docker run --rm -v "$(pwd)":/project ghcr.io/soundmatt/fusaops check
docker run --rm -p 8080:8080 -v "$(pwd)":/project ghcr.io/soundmatt/fusaops serve --addr :8080
```

## Configuration

`.fusaops.json` is optional — FuSaOps works zero-config by detecting languages
directly. When present it sets the project name, restricts which adapters run,
and sets report defaults:

```json
{
  "version": "1",
  "project": { "name": "my-system", "standard": "ISO26262" },
  "scan": { "adapters": ["gofusa", "cpfusa"], "exclude": ["third_party"] },
  "report": { "format": "html", "output": "fusaops-report.html" }
}
```

Per-language configuration stays in each component's own x-FuSa config (e.g. a
Go module's `.fusa.json`); FuSaOps does not manage it.

## CI integration

```yaml
- run: go install github.com/SoundMatt/FuSaOps/cmd/fusaops@latest
- run: fusaops check        # fails the build on any ERROR finding, any language
```

## Self-hosting

FuSaOps is written in Go, so it eats its own dog food: go-FuSa runs `gofusa check`
against the FuSaOps source on every CI run (see `.github/workflows/ci.yml`). The
toolchain that FuSaOps orchestrates also gates FuSaOps itself.

## Supported languages

| Language | Adapter  | Tool     |
|----------|----------|----------|
| Go       | go-FuSa  | `gofusa` |
| C        | c-FuSa   | `cfusa`  |
| C++      | cpp-FuSa | `cpfusa` |

New languages are added by implementing the `adapter.Adapter` interface and
registering it — see [ROADMAP.md](ROADMAP.md).

## License

[Mozilla Public License 2.0](LICENSE).
