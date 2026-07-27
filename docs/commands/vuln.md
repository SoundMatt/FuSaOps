# `fusaops vuln`

Scan dependency manifests (`go.mod`, `Cargo.toml`, `requirements.txt`,
`package.json`, `pom.xml`) for known vulnerabilities.

```bash
fusaops vuln [--dir <path>] [--output <file>] [--format text|json]
```

## Flags

| Flag | Default | Description |
|---|---|---|
| `--dir` | current directory | Project root |
| `--output` | `<dir>/.fusaops-vuln.json` | Report path |
| `--format` | `text` | Output format: `text` or `json` |

## Behaviour

- When `osv-scanner` is available on `PATH`, it is invoked to check each
  discovered manifest against the OSV vulnerability database.
- Otherwise manifests are discovered and reported without a vulnerability
  check (no `osv-scanner` finding data, but still useful as a dependency
  inventory).

## Exit codes

| Code | Meaning |
|---|---|
| 0 | No vulnerabilities found |
| 1 | One or more vulnerabilities found, or scan error |
| 2 | Render error |

## Example

```bash
fusaops vuln --format json --output vuln.json
```

Serves ISO 21434 / IEC 62443 supply-chain vulnerability-management objectives
across every language's dependency manifests.
