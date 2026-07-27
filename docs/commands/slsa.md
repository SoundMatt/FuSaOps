# `fusaops slsa`

Generate a SLSA v1.0 (Supply-chain Levels for Software Artifacts) integrity
gap report for a project.

```bash
fusaops slsa [--dir <path>] [--level L1|L2|L3|L4]
             [--format text|json] [--output <file>]
```

## Flags

| Flag | Default | Description |
|---|---|---|
| `--dir` | `.` | Project root |
| `--level` | `L2` | Target SLSA level: `L1`, `L2`, `L3`, or `L4` |
| `--format` | `text` | Output format: `text` or `json` |
| `--output` | stdout | Write report to file |

## Exit codes

| Code | Meaning |
|---|---|
| 0 | No gaps against the target level |
| 1 | One or more gaps against the target level, or assessment error |

## Example

```bash
fusaops slsa --level L3 --format json --output slsa-report.json
```

Complements `fusaops release` (which produces the provenance and manifest
this report assesses) for supply-chain integrity claims under ISO 21434 and
SLSA itself.
