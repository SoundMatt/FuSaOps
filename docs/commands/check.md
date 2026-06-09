# `fusaops check`

Run every applicable x-FuSa tool against the repository and print the aggregated
multi-language report. **The CI gate.**

```bash
fusaops check [--dir <path>] [--only <tools>] [--format text|json|html|sarif] [--strict]
```

## Flags

| Flag | Default | Description |
|---|---|---|
| `--dir` | `.` | Project root |
| `--only` | all applicable | Comma-separated tool names to run (e.g. `gofusa,cpfusa`) |
| `--format` | `text` | Output format |
| `--strict` | off | Exit non-zero on WARNING findings too |

## Exit codes

| Code | Meaning |
|---|---|
| 0 | No ERROR findings (and no WARNING under `--strict`) |
| 1 | One or more ERROR findings, no supported languages, or a run error |
| 2 | Usage error |

## Behaviour

- Detects languages, runs each applicable **installed** tool, and aggregates.
- Applicable-but-uninstalled or failed tools appear as **skipped** components —
  treat them as coverage gaps (they do not by themselves fail the check).
- Exit code reflects the aggregate across all languages.

## Example

```bash
fusaops check --dir . --format sarif > fusaops.sarif
```

Serves ISO 26262-6, IEC 61508-3, and DO-178C verification objectives by gating
on the union of every language's safety findings.
