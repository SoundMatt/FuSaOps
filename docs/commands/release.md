# `fusaops release`

Generate the cross-language SBOM, build provenance, and artifact manifest in
one pass — the release-evidence bundle for a build.

```bash
fusaops release [--dir <path>] [--output-dir <path>] [--builder <name>]
```

## Flags

| Flag | Default | Description |
|---|---|---|
| `--dir` | current directory | Project root |
| `--output-dir` | project root | Output directory for generated files |
| `--builder` | auto-detected from CI env vars | Builder identifier (e.g. `github-actions`) |

## Behaviour

Runs three steps in order, each writing a file into `--output-dir`:

1. **SBOM** — merged cross-language SBOM (`sbom.json`). Skipped with a notice
   if no adapters are detected.
2. **Provenance** — build provenance record (`provenance.json`), including the
   detected or supplied builder identity.
3. **Artifact manifest** — a manifest (`manifest.json`) referencing the SBOM
   and provenance files with their hashes.

## Exit codes

| Code | Meaning |
|---|---|
| 0 | All steps completed |
| 1 | Any step failed (SBOM build, provenance build, or file I/O) |

## Example

```bash
fusaops release --output-dir dist/ --builder github-actions
```

Produces the SLSA/ISO 21434 supply-chain evidence trio in one command for CI
release jobs.
