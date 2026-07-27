# `fusaops qualify`

Run the tool qualification suite for each installed x-FuSa adapter, producing
tool-confidence evidence for regulated environments (DO-330 / ISO 26262-8 §11).

```bash
fusaops qualify [--dir <path>] [--output <file>] [--format text|json]
                [--type self|independent] [--record-uri <uri>]
```

## Flags

| Flag | Default | Description |
|---|---|---|
| `--dir` | current directory | Project root |
| `--output` | `<dir>/.fusaops-qualify-report.json` | JSON report path |
| `--format` | `text` | Output format: `text` or `json` |
| `--type` | `self` | Qualification type: `self` or `independent` |
| `--record-uri` | — | URI of an external TQL-5/DO-330 qualification certificate |

`--type` and `--record-uri` fall back to `qualify.type` / `qualify.recordUri`
in `.fusaops.json` when the flags are left at their defaults.

## Exit codes

| Code | Meaning |
|---|---|
| 0 | Qualification passed for every adapter |
| 1 | One or more adapters failed qualification, or no applicable adapters found |
| 2 | Render error |

## Behaviour

- Detects applicable adapters and runs each tool's own qualification checks.
- The report includes a SHA-256 integrity hash and, when set, the
  qualification type and certificate URI.

## Example

```bash
fusaops qualify --type independent --record-uri https://example.com/cert/gofusa-v0.33.4
```
