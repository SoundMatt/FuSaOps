# FuSaOps Tool Safety Manual

**Version:** 0.1.0
**Module:** `github.com/SoundMatt/FuSaOps`
**License:** Mozilla Public License 2.0
**Development ASIL:** ISO 26262 ASIL-C
**Standards addressed (via orchestrated tools):** ISO 26262, IEC 61508, ISO 21434, DO-178C

---

## 1. Purpose

This is the Tool Safety Manual for FuSaOps. It is intended for:

- Teams qualifying FuSaOps for use across mixed-language safety-critical repos
- Auditors assessing compliance with ISO 26262-8 (software tools) or equivalents
- CI architects integrating FuSaOps as a release gate

## 2. Tool overview

FuSaOps is a multi-language functional-safety **orchestration** tool. It is
**not** a certification product and does **not** implement language-specific
safety rules. It detects the languages in a repository, runs each language's
x-FuSa tool, normalises and aggregates their findings, and presents one report
and one dashboard.

## 3. Intended use

- Run `fusaops check` locally or in CI to gate a polyglot codebase on the
  aggregated ERROR findings of every applicable tool.
- Run `fusaops report` / `fusaops serve` to produce auditor-facing evidence.

## 4. Assumptions of use (AoU)

The safety argument depends on these assumptions. Violating them invalidates the
qualification.

- **AoU-1** — The adapter tools on `PATH` (or bundled in the image) are the
  qualified versions for your project. Verify with `fusaops adapters` and each
  tool's `version`.
- **AoU-2** — `PATH` is trusted. FuSaOps runs whatever binary resolves for each
  tool name; a shadowing binary would run with your privileges.
- **AoU-3** — A **skipped** component is treated as an unchecked language
  (a coverage gap), not as a pass.
- **AoU-4** — Gating decisions use a freshly computed report. In the dashboard,
  use `/refresh` before relying on the result (hazard H-005).
- **AoU-5** — Each underlying tool is qualified separately; FuSaOps does not
  compensate for a tool's false negatives.

## 5. Hazards and mitigations

See `.fusa-hara.json` for the full HARA. Summary:

| Hazard | Mitigation | Safety goal |
|---|---|---|
| H-001 Dropped finding | Normalisation preserves all findings; unknown severities → INFO | SG-001 |
| H-002 Masked coverage gap | Applicable-but-unrun tools recorded as skipped | SG-002 |
| H-003 Silent failure | Non-zero exit on any ERROR finding | SG-003 |
| H-004 Misattribution | Findings tagged with language + tool | SG-004 |
| H-005 Stale evidence | On-demand recompute / refresh | SG-005 |

## 6. Constraints and limitations

- Output fidelity is bounded by the underlying tools (§4 AoU-5).
- The all-in-one image is `linux/amd64` (tool images are amd64).
- The web dashboard has no authentication by default — `fusaops serve --auth
  user:pass` enables HTTP Basic Auth (with an optional `--auth-ro` read-only
  role), but until a deployment sets one of those flags, bind to localhost or
  place behind an authenticating proxy (see `SECURITY.md`).

## 7. Operating instructions

```bash
fusaops scan          # confirm expected languages/adapters are detected
fusaops adapters      # confirm the expected tools are installed/bundled
fusaops check         # gate (exit non-zero on ERROR; --strict adds WARNING)
fusaops report -o ... # evidence artefact
```

## 8. Response to tool malfunction

If FuSaOps behaves unexpectedly (a finding you can reproduce with the tool
directly does not appear in the aggregate, or a coverage gap is not surfaced),
stop relying on the gate and follow `INCIDENT-RESPONSE.md`.

## 9. Configuration management

FuSaOps is versioned in git; releases are tagged and the image is published to
`ghcr.io/soundmatt/fusaops`. Evidence artefacts (`.fusa-evidence.json`,
`qualify-report.json`, `safety-case.json`, etc.) are committed. See
`docs/release-process.md`.
