# `fusaops template`

Generate safety documentation templates for multi-language projects, written
as Markdown files. Available templates cover: Software Safety Plan, HARA, SRS,
Test Plan, TARA, SCI, SAS, and Problem Report.

```bash
fusaops template [--dir <path>] [--output-dir <path>] [--standards <list>]
                  [--output <file>] [--format text|json]
```

## Flags

| Flag | Default | Description |
|---|---|---|
| `--dir` | current directory | Project root |
| `--output-dir` | `<dir>/safety-docs` | Directory to write templates |
| `--standards` | all | Comma-separated list of standards to filter by (`ISO 26262`, `IEC 61508`, `DO-178C`, `ISO 21434`) |
| `--output` | `<dir>/.fusaops-templates.json` | Path for the generation report |
| `--format` | `text` | Output format: `text` or `json` |

## Exit codes

| Code | Meaning |
|---|---|
| 0 | Templates generated |
| 1 | Generation error |
| 2 | Render error |

## Example

```bash
fusaops template --standards "ISO 26262,DO-178C" --output-dir docs/safety
```

Bootstraps a project's compliance paperwork so the required documents exist
from day one, ready to be filled in.
