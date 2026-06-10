# `fusaops sbom`

Merge every applicable x-FuSa tool's Software Bill of Materials into one
**cross-language** SBOM, rendered as native JSON, plain text, or an SPDX 2.3
document.

```bash
fusaops sbom [--dir <path>] [--only <tools>] [--format json|text|spdx] [--output <file>]
```

## Flags

| Flag | Default | Description |
|---|---|---|
| `--dir` | `.` | Project root |
| `--only` | all applicable | Comma-separated tool names to roll up |
| `--format` | `json` | Output format (`json`, `text`, `spdx`) |
| `--output` | stdout | Write the SBOM to a file |

## Exit codes

| Code | Meaning |
|---|---|
| 0 | Rendered successfully |
| 1 | No supported languages, or a run error |
| 2 | Usage error |

## Behaviour

- Runs each installed tool's SBOM generation, decodes its package list, and
  **merges + de-duplicates** on `(name, version)` across languages.
- Each merged package keeps the language of the first component (in tool order)
  that contributed it; the list is deterministically sorted.
- A tool that is not installed or cannot produce an SBOM is recorded as a
  skipped component.

## SPDX

`--format spdx` emits a minimal, valid **SPDX 2.3** JSON document: one package
per dependency, each with a `DESCRIBES` relationship to the document and a
syntactically valid `SPDXRef-Package-N` identifier.

## Example

```bash
fusaops sbom --format spdx --output sbom.spdx.json
```

Serves the supply-chain / configuration-management objectives of ISO 26262-8,
ISO 21434, and DO-178C across every language in one bill of materials.
