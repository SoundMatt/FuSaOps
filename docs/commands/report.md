# `fusaops report`

Generate the aggregated multi-language report as an evidence artefact. Unlike
`check`, it never fails on findings.

```bash
fusaops report [--dir <path>] [--only <tools>] [--format text|json|html|sarif] [--output <file>]
```

## Flags

| Flag | Default | Description |
|---|---|---|
| `--dir` | `.` | Project root |
| `--only` | all applicable | Restrict to specific tools |
| `--format` | `json` | Output format |
| `--output` | stdout | Output file |

## Formats

- **json** — the canonical `AggregateReport` schema (for tooling).
- **html** — a self-contained dashboard (auditor-facing).
- **sarif** — SARIF 2.1.0, one run per component, for GitHub Code Scanning.
- **text** — human-readable summary.

## Example

```bash
fusaops report --format html --output fusaops-report.html
```
