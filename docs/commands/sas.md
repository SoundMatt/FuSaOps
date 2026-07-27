# `fusaops sas`

Generate the Software Accomplishment Summary (SAS) per DO-178C §11.20. The SAS
attests that all software lifecycle activities have been completed and their
outputs verified — each activity maps to a FuSaOps evidence artefact; a
missing artefact marks the activity as incomplete.

```bash
fusaops sas [--dir <path>] [--output <file>] [--format text|json]
            [--level DAL-A|DAL-B|DAL-C|DAL-D|DAL-E]
```

## Flags

| Flag | Default | Description |
|---|---|---|
| `--dir` | current directory | Project root |
| `--output` | `<dir>/.fusaops-sas.json` | Report path |
| `--format` | `text` | Output format: `text` or `json` |
| `--level` | `DAL-C` | Software level, DAL-A through DAL-E |

## Exit codes

| Code | Meaning |
|---|---|
| 0 | No gaps — every lifecycle activity has its required evidence |
| 1 | One or more activities incomplete, or build error |
| 2 | Render error |

## Example

```bash
fusaops sas --level DAL-B --format json --output sas.json
```
