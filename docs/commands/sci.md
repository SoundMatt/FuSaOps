# `fusaops sci`

Generate the Software Configuration Index (SCI) per DO-178C §11.16 — a list of
every software configuration item (tools, evidence artefacts, and language
components) with SHA-256 hashes and availability status.

```bash
fusaops sci [--dir <path>] [--output <file>] [--format text|json]
```

## Flags

| Flag | Default | Description |
|---|---|---|
| `--dir` | current directory | Project root |
| `--output` | `<dir>/.fusaops-sci.json` | Report path |
| `--format` | `text` | Output format: `text` or `json` |

## Behaviour

- Detects applicable adapters (availability is best-effort — an
  uninstalled tool is still listed, marked unavailable).
- Every configuration item is hashed where a file exists, so the SCI can be
  used to detect drift between what was assessed and what shipped.

## Exit codes

| Code | Meaning |
|---|---|
| 0 | Success |
| 1 | Adapter detection or build error |
| 2 | Render error |

## Example

```bash
fusaops sci --format json --output sci.json
```
