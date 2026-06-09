# Tool Qualification Guide

## Overview

ISO 26262 Part 8 §11 and IEC 61508 Part 3 require that software tools used in
safety-related development be qualified. Qualification establishes confidence
that a tool performs its intended function correctly. It does not certify the
tool; it provides documented evidence supporting a safety assessor's judgement.

FuSaOps is an **orchestration tool**: it runs the per-language x-FuSa tools and
aggregates their output. Its qualification therefore has two parts:

1. **FuSaOps itself** — does it faithfully run the right tools and aggregate
   their findings without loss, miscount, or misattribution?
2. **The orchestrated tools** — each x-FuSa tool carries its own qualification
   evidence (`<tool> qualify`); FuSaOps does not re-qualify them.

This guide covers part 1. For part 2, see each tool's qualification guide.

## Tool classification

Per ISO 26262-8 Table 4, tool confidence derives from **Tool Impact (TI)** and
**Tool error Detection (TD)**:

- **TI2** — a FuSaOps malfunction (a dropped or masked finding) *can* introduce
  or fail to detect an error in the safety-related item.
- **TD2** — there is medium confidence that such a malfunction is detected,
  because each tool's findings are independently reproducible by running the
  tool directly, and FuSaOps records skipped components explicitly.

TI2 × TD2 ⇒ **TCL2**, consistent with FuSaOps's ISO 26262 **ASIL-C** development
posture (see `.fusa.json`, `.fusa-hara.json`).

## Confidence measures

| Measure | Evidence |
|---|---|
| Faithful aggregation | Unit tests assert every tool finding survives normalisation; unknown severities normalise to INFO, never dropped (SG-001) |
| Visible coverage gaps | Applicable-but-uninstalled or failed tools are recorded as skipped components, never omitted (SG-002) |
| Correct gating | `fusaops check` exits non-zero on any ERROR finding; tested for exit-code invariance (SG-003) |
| Correct attribution | Findings carry originating language + tool through aggregation (SG-004) |
| Requirement traceability | `gofusa trace` reports every requirement traced **and** tested |
| Self-check | go-FuSa runs `gofusa check` on FuSaOps's own Go source in CI |

## Validating FuSaOps in your environment

```bash
# 1. Reproduce the test evidence (full suite, 0 failures expected)
go test -race -count=1 ./...

# 2. Confirm full requirement traceability
gofusa trace --dir .

# 3. Confirm the self-check passes (0 ERROR findings)
gofusa check --dir .

# 4. Confirm the orchestrator surfaces a known finding end-to-end
fusaops check --dir <a repo with a known violation>
```

Record the outputs as qualification evidence alongside the committed
`.fusa-evidence.json` and `qualify-report.json`.

## Known limitations

- FuSaOps reports only what the underlying tools report; a false negative in a
  bundled tool is a false negative in FuSaOps. Qualify each tool separately.
- A language with no installed adapter is reported as a **skipped** component,
  not a failure — treat skipped components as coverage gaps in your assessment.
- The bundled image pins tool versions at build time; verify the bundled
  versions (`fusaops adapters`, each tool's `version`) match your qualified set.
