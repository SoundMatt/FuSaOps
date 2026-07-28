# `fusaops safety-case`

Assemble a structured safety argument from FuSaOps evidence artefacts. Each
claim in the safety case maps to a class of evidence (test bundle,
qualification report, SBOM, build provenance, etc.); a claim passes when all
required artefacts are present in the project root.

```bash
fusaops safety-case [--dir <path>] [--output <file>] [--format text|json]
                     [--standard "ISO 26262"|"DO-178C"|"IEC 61508"|"ISO 21434"]
```

## Flags

| Flag | Default | Description |
|---|---|---|
| `--dir` | current directory | Project root |
| `--output` | `<dir>/.fusaops-safety-case.json` | Report path |
| `--format` | `text` | Output format: `text` or `json` |
| `--standard` | `ISO 26262` | Target standard |

## Exit codes

| Code | Meaning |
|---|---|
| 0 | All claims satisfied |
| 1 | One or more claims have a gap (missing evidence), or build error |
| 2 | Unknown `--standard`, or render error |

## Behaviour

- Looks for each claim's required evidence file (e.g. `.fusaops-evidence.json`
  for `verify`, `sbom.json` for the SBOM claim) under the project root.
- The generated report includes a SHA-256 integrity hash (`sha256:`-prefixed).
- Alongside `claims`/`totalClaims`/`passedClaims`, the JSON also carries a GSN
  (Goal Structuring Notation) projection of the same data — `nodes`/`edges`/
  `completeness` — per x-FuSa spec §9.2: each claim becomes a `goal` node
  supported by a `strategy` node, itself supported by one `solution` node per
  present evidence file. A claim with no present evidence counts toward
  `completeness.undeveloped` rather than fabricating a solution.

## Example

```bash
fusaops safety-case --standard "DO-178C" --format json --output safety-case.json
```

Assembles the top-level safety argument that ties together every other
FuSaOps evidence artefact for audit review.
