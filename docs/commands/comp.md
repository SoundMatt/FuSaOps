# `fusaops comp`

Compute McCabe cyclomatic complexity (V(G)) across every applicable language by
delegating to each tool's own `comp` subcommand and rolling up the results.

```bash
fusaops comp [--dir <path>] [--only <tools>] [--format text|json]
             [--threshold <n>] [--dal DAL-A|DAL-B|DAL-C|DAL-D]
             [--workers <n>] [--timeout <duration>]
```

## Flags

| Flag | Default | Description |
|---|---|---|
| `--dir` | `.` | Project root |
| `--only` | all applicable | Comma-separated tool names to run |
| `--format` | `text` | Output format: `text` or `json` |
| `--output` | stdout | Write report to file |
| `--threshold` | tool default | Complexity threshold override (0 = use tool default) |
| `--dal` | (none) | DAL level override that sets the threshold: `DAL-A`\|`DAL-B`\|`DAL-C`\|`DAL-D` |
| `--workers` | unlimited | Max parallel adapters |
| `--timeout` | none | Per-adapter deadline, e.g. `30s`, `5m` |

## DAL-level thresholds (DO-178C §6.3.4)

| Level | Max V(G) |
|---|---|
| DAL-A | 4 |
| DAL-B | 10 |
| DAL-C | 15 |
| DAL-D | 20 |

`--threshold` and `--dal` are mutually reinforcing: setting `--dal` alone
applies its threshold; `--threshold` overrides it explicitly.

## Exit codes

| Code | Meaning |
|---|---|
| 0 | No function exceeds the threshold |
| 1 | One or more functions exceed the threshold, or no supported languages detected |
| 2 | Usage error |

## Example

```bash
fusaops comp --dal DAL-B --format json --output complexity.json
```

Serves ISO 26262-6 and DO-178C §6.3.4 structural complexity objectives across
every language in the repository.
