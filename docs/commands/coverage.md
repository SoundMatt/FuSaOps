# `fusaops coverage`

Produce a DO-178C structural coverage report from a Go coverage profile, or
run the LLVM source-based MC/DC gate when `--mcdc` is set.

```bash
fusaops coverage [--dal DAL-A|DAL-B|DAL-C|DAL-D] [--format text|json|markdown]
                  [--output <file>] [--dir <path>] [coverage.out]

fusaops coverage --mcdc [--mcdc-file <path>] [--mcdc-threshold <pct>]
                  [--req-dir <path>] [--dal <level>]
```

## Flags

| Flag | Default | Description |
|---|---|---|
| `--dal` | `DAL-B` | Design assurance level: `DAL-A`, `DAL-B`, `DAL-C`, `DAL-D` |
| `--format` | `text` | Output format: `text`, `json`, or `markdown` |
| `--output` | stdout | Write report to file |
| `--dir` | — | Directory to search for `coverage.out` |

### MC/DC flags (LLVM source-based, DAL-A prerequisite)

| Flag | Default | Description |
|---|---|---|
| `--mcdc` | off | Enable the LLVM source-based MC/DC coverage gate |
| `--mcdc-file` | `mcdc.json` | Path to `llvm-cov export --format=json` output |
| `--mcdc-threshold` | `100.0` | Minimum condition coverage percentage for gate pass |
| `--req-dir` | `--dir` or cwd | Directory to scan for `//fusa:req`-annotated functions |

## Behaviour

- **Standard path**: reads a Go coverage profile (generate with
  `go test -coverprofile=coverage.out ./...`) and reports DO-178C statement
  coverage against the given DAL.
- **MC/DC path** (`--mcdc`): parses LLVM's per-condition coverage JSON,
  cross-references `//fusa:req`-annotated functions, and gates on the
  configured condition-coverage threshold — the DAL-A prerequisite DO-178C
  §6.4.4.2 objective that statement/branch coverage alone cannot satisfy.

## Exit codes

| Code | Meaning |
|---|---|
| 0 | Coverage/MC/DC gate passed |
| 1 | Gate failed, or profile/MC/DC file not found |
| 2 | Invalid `--dal`, or render error |

## Example

```bash
go test -coverprofile=coverage.out ./...
fusaops coverage --dal DAL-C coverage.out

# MC/DC (DAL-A):
llvm-cov export ./mybinary -instr-profile=default.profdata -format=json > mcdc.json
fusaops coverage --mcdc --mcdc-file mcdc.json --dal DAL-A
```
