# `fusaops badge`

Generate a Shields.io-compatible SVG status badge from a `fusaops check
--format json` report. Reads from a file argument, or from stdin if none is
given.

```bash
fusaops badge [--output <file>] [report.json]
```

## Flags

| Flag | Default | Description |
|---|---|---|
| `--output` | stdout | Write the SVG to a file |

## Exit codes

| Code | Meaning |
|---|---|
| 0 | Badge generated |
| 1 | Read/parse error |
| 2 | More than one positional argument |

## Example

```bash
fusaops check --format json | fusaops badge --output badge.svg
fusaops badge --output badge.svg check-report.json
```

Embed the resulting SVG in a README (`![FuSaOps](badge.svg)`) for an
at-a-glance error/warning count.
