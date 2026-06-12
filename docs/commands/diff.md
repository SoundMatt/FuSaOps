# `fusaops diff`

Compare a stored baseline check-report against a fresh scan and **gate on new
findings**. The rolling-baseline CI step: store a passing run, then fail any
build that introduces new safety issues.

```bash
fusaops diff [--dir <path>] [--baseline <file>] [--only <tools>]
             [--format text|json] [--output <file>]
             [--strict] [--update-baseline]
```

## Flags

| Flag | Default | Description |
|---|---|---|
| `--dir` | `.` | Project root to scan |
| `--baseline` | `check-report.json` | Path to baseline (relative to `--dir` if not absolute) |
| `--only` | all applicable | Comma-separated tool names to run |
| `--format` | `text` | Output format (`text` or `json`) |
| `--output` | stdout | Write the diff result to a file |
| `--strict` | off | Fail on any new finding, not just new errors |
| `--update-baseline` | off | Overwrite `--baseline` with the current run's findings after comparing |

## Exit codes

| Code | Meaning |
|---|---|
| 0 | Gate PASS — no new ERROR findings (or no new findings at all under `--strict`) |
| 1 | Gate FAIL, missing baseline, scan error, or usage error |

## Behaviour

- Findings are matched by **fingerprint** (§4.2 of the x-FuSa spec). If a
  finding in either side lacks a fingerprint, `ComputeFingerprint` is called on
  the fly so baselines produced before fingerprint adoption still work.
- The baseline can be a FuSaOps aggregate report (`components[].findings`) or a
  flat single-tool report (`findings[]`).
- Added findings are sorted by file → line → ruleId for deterministic output.
- Text output includes a per-severity breakdown (`2 added (1 error, 1 warning)`).
- JSON output includes `generatedAt` and a `summary` object with per-severity
  counts alongside `added`, `removed`, `unchanged`, and `gate`.

## Rolling-baseline workflow

```bash
# First run: store the baseline from a clean check
fusaops check --format json --output check-report.json

# CI: gate on regressions and roll the baseline forward on success
fusaops diff --baseline check-report.json --update-baseline
```

`--update-baseline` writes the current run's findings back to the baseline file
**before** the exit code is set, so a green CI job leaves an up-to-date baseline
committed alongside the code.

## Strict mode

```bash
fusaops diff --strict    # fail on any new finding, including warnings
```

Tighter gate for release branches where no new issue of any severity is acceptable.

## Example output (text)

```
FuSaOps Diff — 1 added (1 error), 0 removed, 42 unchanged
──── Added ────
+ [ERROR] SAFETY001 [safety] unguarded memory access (drivers/can.c:87)
    → Replace with safe_memcpy from the safety HAL
Gate: FAIL
```

Serves ISO 26262-6 regression prevention objectives by ensuring every
commit is compared against a known-good safety baseline across all languages.
