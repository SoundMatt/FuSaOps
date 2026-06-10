# `fusaops audit-pack`

Bundle every applicable x-FuSa tool's audit-pack together with the FuSaOps
**cross-language** evidence (aggregate report, traceability matrix, SBOM) into a
single ZIP an auditor can open once for the whole polyglot project.

```bash
fusaops audit-pack [--dir <path>] [--only <tools>] [--output <file>]
```

## Flags

| Flag | Default | Description |
|---|---|---|
| `--dir` | `.` | Project root |
| `--only` | all applicable | Comma-separated tool names to include |
| `--output` | `audit-pack.zip` | Output path for the unified ZIP |

## Exit codes

| Code | Meaning |
|---|---|
| 0 | Bundle written |
| 1 | No supported languages, or a packing error |
| 2 | Usage error |

## Layout

```
audit-pack.zip
├── manifest.json                       # FuSaOps index: tool, version, files + SHA-256
├── report.json                         # aggregate multi-language findings
├── trace.json                          # cross-language traceability + qualification
├── sbom.json                           # merged cross-language SBOM
└── components/
    └── <tool>/audit-pack.zip           # each language's own evidence pack, verbatim
```

## Behaviour

- Each installed tool's `audit-pack` is nested under `components/<tool>/`.
- The FuSaOps-level artefacts are **always** included, so the bundle is never
  empty even if no per-tool pack could be produced.
- A tool that is not installed or cannot pack is reported as skipped on stdout;
  it never aborts the bundle.
- `manifest.json` records the path, size, and SHA-256 of every packed file for
  integrity verification.

## Example

```bash
fusaops audit-pack --output evidence/audit-pack.zip
```

Assembles ISO 26262, IEC 61508, ISO 21434, and DO-178C evidence from every
language into one auditor-facing archive.
