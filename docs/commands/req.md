# `fusaops req`

Show, import, or export requirements from `.fusa-reqs.json`.

```bash
fusaops req [REQ-ID ...]                          # show (default)
fusaops req import --file <path> [--format csv|doors|polarion|codebeamer|jama]
fusaops req export [--output <path>] [--format csv|doors|polarion|codebeamer|jama]
```

## Subcommands

### (none) — show

Prints requirement metadata. With no `REQ-ID` arguments, shows every
requirement; otherwise filters to the given IDs (exits 1 if any ID is not
found).

### `import`

| Flag | Required | Default | Description |
|---|---|---|---|
| `--file` | yes | — | Input file path |
| `--format` | no | `csv` | `csv`, `doors`, `polarion`, `codebeamer`, or `jama` |

Skips requirements whose ID already exists in the registry (reports how many
were added vs. skipped as duplicates).

### `export`

| Flag | Default | Description |
|---|---|---|
| `--format` | `csv` | `csv`, `doors`, `polarion`, `codebeamer`, or `jama` |
| `--output` | stdout | Output file |

## Flags (global)

| Flag | Default | Description |
|---|---|---|
| `--dir` | `.` | Project root |

## Exit codes

| Code | Meaning |
|---|---|
| 0 | Success |
| 1 | Load/save error, requirement ID not found, or import/export parse error |
| 2 | Missing `--file` (import), or unknown format |

## Example

```bash
fusaops req                                   # list all requirements
fusaops req REQ-FO-CLI001                     # show one requirement
fusaops req import --file reqs.csv
fusaops req export --format doors --output reqs.doors.xml
```

Interoperates with external requirements-management tools (DOORS, Polarion,
Codebeamer, Jama) for organizations that maintain their master requirement set
outside `.fusa-reqs.json`.
