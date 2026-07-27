# `fusaops hara`

Manage the Hazard Analysis and Risk Assessment (`.fusa-hara.json`) per
ISO 26262-3:2018.

```bash
fusaops hara [show|init|asil] [flags]
```

## Subcommands

### `show` (default)

```bash
fusaops hara show [--format text|json|markdown] [--output <file>]
```

Displays the HARA. Also runs validation and, when `--output` is a file,
prints a gap-count warning to stderr if issues are found.

### `init`

```bash
fusaops hara init [--project <name>] [--standard "ISO 26262"]
```

Creates a starter `.fusa-hara.json` with one example operational situation,
hazard, and safety goal. Refuses to overwrite an existing file.

### `asil`

```bash
fusaops hara asil -s <S0-S3> -e <E0-E4> -c <C0-C3>
```

Derives an ASIL from Severity/Exposure/Controllability per ISO 26262-3:2018
Table 4.

## Flags (global)

| Flag | Default | Description |
|---|---|---|
| `--dir` | current directory | Project root |

## Exit codes

| Code | Meaning |
|---|---|
| 0 | Success |
| 1 | Load/save error |
| 2 | Unknown subcommand, missing required flags, or file already exists (`init`) |

## Example

```bash
fusaops hara init --project my-ecu
fusaops hara asil -s S2 -e E3 -c C2   # → ASIL B
fusaops hara show --format markdown --output HARA.md
```
