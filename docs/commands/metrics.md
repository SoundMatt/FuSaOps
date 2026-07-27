# `fusaops metrics`

Track FuSaOps project safety metrics over time in `.fusaops-metrics.json`.

```bash
fusaops metrics <record|show> [flags]
```

## Subcommands

### `record`

Collects a metrics snapshot (error/warning/info counts, total requirements,
coverage percentage) and appends it to the time series.

### `show`

```bash
fusaops metrics show [--format text|json] [--output <file>]
```

Displays the full metrics time series.

## Flags (global)

| Flag | Default | Description |
|---|---|---|
| `--dir` | `.` | Project root |

## Exit codes

| Code | Meaning |
|---|---|
| 0 | Success |
| 1 | Load/collect/save error |
| 2 | Missing subcommand, or usage error |

## Example

```bash
fusaops metrics record                 # snapshot the current scan
fusaops metrics show --format json     # inspect the trend
```

Useful as a scheduled CI step (e.g. nightly) to build a history of safety
posture for dashboards and audits.
