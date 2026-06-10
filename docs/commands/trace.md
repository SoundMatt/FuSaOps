# `fusaops trace`

Roll every applicable x-FuSa tool's requirement traceability matrix and
qualification summary up into one **cross-language** view. The polyglot coverage
gate.

```bash
fusaops trace [--dir <path>] [--only <tools>] [--format text|json|html] [--output <file>] [--strict]
```

## Flags

| Flag | Default | Description |
|---|---|---|
| `--dir` | `.` | Project root |
| `--only` | all applicable | Comma-separated tool names to roll up |
| `--format` | `text` | Output format (`text`, `json`, `html`) |
| `--output` | stdout | Write the matrix to a file |
| `--strict` | off | Exit non-zero when any requirement is untraced or untested |

## Exit codes

| Code | Meaning |
|---|---|
| 0 | Rendered successfully (and, under `--strict`, no coverage gaps) |
| 1 | Coverage gaps under `--strict`, no supported languages, or a run error |
| 2 | Usage error |

## Behaviour

- Runs each installed tool's `trace --format json` and `qualify`, decoding the
  per-language requirement counts and tool-confidence figures.
- **Skipped components** (tool not installed, or it does not support trace) are
  shown but excluded from the aggregate totals, so a missing tool surfaces as a
  visible gap rather than silently inflating the percentage.
- Aggregate coverage is the sum across every component that produced a matrix.

## Example

```bash
fusaops trace --format html --output trace.html
fusaops trace --strict        # CI: fail when any language has an untested requirement
```

Serves ISO 26262-8 §6 (requirement traceability) and the equivalent IEC 61508,
DO-178C, and ISO 21434 traceability objectives across every language at once.
