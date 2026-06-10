# `fusaops conform`

Run x-FuSa spec v1.8 conformance checks against a tool binary.

```bash
fusaops conform <binary> [flags]
```

## Flags

| Flag | Default | Description |
|---|---|---|
| `--format` | `text` | Output format: `text` or `json` |
| `--output` | stdout | Write report to file |

## Exit codes

| Code | Meaning |
|---|---|
| 0 | All MUST checks passed |
| 1 | One or more MUST checks failed; or binary not found on PATH |
| 2 | Usage error |

## Behaviour

- Creates a temporary project with `.fusa.json`, `.fusa-reqs.json`, and
  language-appropriate source stubs derived from the binary name.
- Invokes each required x-FuSa subcommand (`version`, `init`, `check`, `trace`,
  `qualify`, `release`, `audit-pack`, `capabilities`) and validates the output
  against the normative shapes in the spec.
- Reports each check with its stable ID, RFC 2119 level (MUST/SHOULD/MAY),
  spec section reference, and PASS/FAIL/SKIP status.
- SHOULD/MAY failures are reported but do not set exit code 1 — only MUST
  failures trip the exit code.

## Example

```bash
# Check and print text report
fusaops conform gofusa

# JSON report to file for CI archiving
fusaops conform cfusa --format json --output cfusa-conform.json
```

## Full conformance guide

See [`docs/conformance.md`](../conformance.md) for the complete list of checks,
spec section cross-references, JSON Schema artifacts, and golden test vectors.
