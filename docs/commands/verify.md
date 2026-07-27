# `fusaops verify`

Run `go test -json -count=1 ./...` over FuSaOps's own Go source and save a test
evidence bundle to `.fusaops-evidence.json`.

```bash
fusaops verify [--dir <path>] [--output <file>] [--format text|json]
```

## Flags

| Flag | Default | Description |
|---|---|---|
| `--dir` | current directory | Project root |
| `--output` | `<dir>/.fusaops-evidence.json` | Evidence bundle path |
| `--format` | `text` | Output format: `text` or `json` |

## Exit codes

| Code | Meaning |
|---|---|
| 0 | All tests passed |
| 1 | One or more tests failed, or `go test` could not run |
| 2 | Render error |

## Behaviour

- Runs the Go test suite via `go test -json`, capturing pass/fail per package
  and test.
- Saves a signed-hashable evidence bundle summarizing pass/fail counts,
  suitable as DO-178C §11.14 / ISO 26262-6 test-execution evidence.

## Example

```bash
fusaops verify --output test-evidence.json
```
