# `fusaops capabilities`

Report FuSaOps's supported commands, output formats, and safety standards as
the x-FuSa spec §9.1 discovery document (`kind: "capabilities"`).

```bash
fusaops capabilities [--format json]
```

## Flags

| Flag | Default | Description |
|---|---|---|
| `--format` | `json` | Output format (only `json` is supported) |

## Exit codes

| Code | Meaning |
|---|---|
| 0 | Success |
| 1 | Encode error |
| 2 | Unsupported `--format` |

## Behaviour

Emits a JSON document with:

- `schemaVersion` / `specVersion` — the x-FuSa spec version FuSaOps conforms to
- `toolVersion` — FuSaOps's own version
- `commands` — every registered subcommand
- `formats` — supported output formats per subcommand
- `standards` — safety/security standards FuSaOps can report against

## Example

```bash
fusaops capabilities | jq '.commands'
```

Useful for scripts and orchestrators that need to discover what a given
FuSaOps build supports without hardcoding a command list.
