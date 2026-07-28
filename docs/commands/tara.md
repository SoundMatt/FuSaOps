# `fusaops tara`

Generate a Threat Analysis and Risk Assessment (TARA) per ISO 21434:2021
Chapter 9. Produces a structured report of cybersecurity threat scenarios for
the multi-language safety-analysis toolchain, each with impact, feasibility,
computed risk level, and recommended treatment controls.

```bash
fusaops tara [--dir <path>] [--output <file>] [--format text|json]
```

## Flags

| Flag | Default | Description |
|---|---|---|
| `--dir` | current directory | Project root |
| `--output` | `<dir>/.fusaops-tara.json` | Report path |
| `--format` | `text` | Output format: `text` or `json` |

## Exit codes

| Code | Meaning |
|---|---|
| 0 | No critical-risk scenarios |
| 1 | One or more scenarios carry a critical risk level, or build error |
| 2 | Render error |

## Example

```bash
fusaops tara --format json --output tara.json
```

Serves ISO 21434 threat analysis objectives across the FuSaOps orchestration
pipeline and the tools it invokes.

## x-FuSa spec conformance

The JSON shape follows x-FuSa spec §9.2's `tara` schema: threat scenarios are
under `threats[]` (each carrying `threat`/`attackFeasibility`/`mitigations`),
and `impact` is an SFOP object (`safety`/`financial`/`operational`/`privacy`)
per ISO 21434 Clause 15.7 rather than a single generic severity, since a
threat can rate differently on each axis.
