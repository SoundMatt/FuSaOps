# `fusaops fmea`

Generate a Design Failure Mode and Effects Analysis (dFMEA) per
IEC 61508:2010 / ISO 26262:2018 Part 8-7. Analyses failure modes in the
FuSaOps orchestration pipeline itself, each with Severity, Occurrence, and
Detection ratings (1–10); RPN = S × O × D.

```bash
fusaops fmea [--dir <path>] [--output <file>] [--format text|json]
```

## Flags

| Flag | Default | Description |
|---|---|---|
| `--dir` | current directory | Project root |
| `--output` | `<dir>/.fusaops-fmea.json` | Report path |
| `--format` | `text` | Output format: `text` or `json` |

## Exit codes

| Code | Meaning |
|---|---|
| 0 | No high-RPN failure modes |
| 1 | One or more failure modes exceed the high-RPN threshold, or build error |
| 2 | Render error |

## Example

```bash
fusaops fmea --format json --output fmea.json
```

Serves IEC 61508 / ISO 26262 Part 8 dFMEA objectives for the orchestration
tool itself.

## x-FuSa spec conformance

The JSON shape follows x-FuSa spec §9.2's `fmea` schema: failure modes are
under `entries[]` (each carrying `failureMode`/`item`/`actionPriority`/
`requirementIds`), and `ratingScale`/`summary` are top-level fields. This
package's 8 entries are a fixed, curated review of key architectural risk
areas in the FuSaOps orchestration pipeline, not an exhaustive per-function
scan.
