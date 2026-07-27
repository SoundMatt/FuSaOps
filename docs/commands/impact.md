# `fusaops impact`

Analyse the effect of source changes on requirements and safety artefacts.
Uses `git diff` to identify changed files, then cross-references `fusa:req` /
`fusa:test` annotations to find impacted requirements and stale evidence.

```bash
fusaops impact [--dir <path>] [--from <ref>] [--to <ref>]
               [--format text|json] [--output <file>]
```

## Flags

| Flag | Default | Description |
|---|---|---|
| `--dir` | `.` | Project root |
| `--from` | working tree vs `HEAD` | From git ref |
| `--to` | `HEAD` | To git ref |
| `--format` | `text` | Output format: `text` or `json` |
| `--output` | stdout | Write report to file |

## Behaviour

- Requires a git repository; changed files are diffed between `--from` and
  `--to`.
- Requirements whose `fusa:req`/`fusa:test`-annotated source changed are
  reported as impacted.
- Evidence artefacts (SBOM, qualification report, safety case, etc.) whose
  inputs changed since they were generated are flagged **stale**.

## Example

```bash
fusaops impact --from main --to HEAD --format json --output impact.json
```

Supports ISO 26262-8 / DO-178C §7.2.5 change-impact analysis by pinpointing
exactly which requirements and evidence a code change touches.
