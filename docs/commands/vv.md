# `fusaops vv`

Manage V&V (verification and validation) independence declarations and report
the achievable ASIL, stored in `.fusaops.json`.

```bash
fusaops vv [show|set] [flags]
```

## Subcommands

### `show` (default)

```bash
fusaops vv show [--format text|json] [--output <file>]
```

Displays the current independence declarations and the computed achievable
ASIL. Prints validation warnings to stderr (e.g. reviewer == author).

### `set`

```bash
fusaops vv set [--implementation-author <name>]
               [--independent-reviewer <name>]
               [--independent-test-executor <name>]
```

Updates only the flags supplied; omitted flags keep their existing value.
Requires an existing `.fusaops.json` (run `fusaops init` first).

## Flags (global)

| Flag | Default | Description |
|---|---|---|
| `--dir` | current directory | Project root |

## Exit codes

| Code | Meaning |
|---|---|
| 0 | Success |
| 1 | No config found (`set`), or load/save error |
| 2 | Unknown subcommand |

## Example

```bash
fusaops vv set --independent-reviewer "A. Chen" --independent-test-executor "B. Diallo"
fusaops vv show
```

Serves ISO 26262-8 §6 independence requirements — the achievable ASIL is
capped by which roles are (and aren't) held independently.
