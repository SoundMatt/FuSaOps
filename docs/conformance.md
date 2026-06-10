# x-FuSa Conformance Kit

FuSaOps ships a **conformance kit** that turns the prose x-FuSa spec (v1.8,
frozen) into machine-checkable assertions.  Any x-FuSa tool binary — whether
an existing language tool, a work-in-progress port, or a third-party
implementation — can be validated against the spec in CI with a single command.

---

## Quick start

```bash
# Check go-FuSa's installed binary
fusaops conform gofusa

# Save a JSON report
fusaops conform gofusa --format json --output conform-report.json

# CI gate — exits 1 on any MUST failure
fusaops conform gofusa && echo "conformant"
```

---

## What `fusaops conform` checks

The runner creates a temporary project, invokes each of the tool's required
subcommands, and validates the output against the normative shapes in
`docs/x-fusa-spec.md` (v1.8).

| Category | Section | Level | What is verified |
|---|---|---|---|
| **version** | §9.1 | MUST | Exit 0; line matches `<tool> <semver>`; `--format json` emits `tool`/`version` keys |
| **init** | §9.1 | MUST | Exit 0; `.fusa.json` and `.fusa-reqs.json` written |
| **check** | §4 | MUST | Exit 0 or 1; valid §3.1 header; findings have nested `location`; `ruleId` format; `fingerprint` format; `category` enum; project-relative paths |
| **check** | §4 | SHOULD | `fingerprint` present; `standard`/`clause` fields |
| **trace** | §5 | MUST | Valid header; `requirements`/`tags`/`coverage` keys (not flat); coverage counters present |
| **qualify** | §6 | MUST | Valid header; `total`/`passed`/`failed` keys (not `tests_passed`/`tests_failed`) |
| **release** | §7 | MUST | Valid header; `sbom.json` written; SBOM has `module`/`components` |
| **audit-pack** | §8 | MUST | Single ZIP; `manifest.json` inside; SHA-256 values are bare lowercase hex (no `sha256:` prefix) |
| **capabilities** | §9.1 | SHOULD | Exit 0; `specVersion` and `commands` keys present |

Every check has a stable ID (e.g. `check/finding-location-nested`) and an RFC
2119 level.  `HasFailures()` is true only when a **MUST** check fails; SHOULD
failures are reported but do not fail the exit code.

---

## Exit codes

| Code | Meaning |
|---|---|
| 0 | All MUST checks passed |
| 1 | One or more MUST checks failed; or binary not found on PATH |
| 2 | Usage error |

---

## Output formats

### Text (default)

```
x-FuSa conformance: gofusa 0.24.0
binary:  /usr/local/bin/gofusa
lang:    go
spec:    1.8
results: 37 PASS  0 FAIL  1 SKIP

  ✓ [MUST] §9.1  version line format
  ✓ [MUST] §9.1  version json
  ✓ [MUST] §9.1  init creates config files
  ...
  – [SHOULD] §9.1  capabilities specVersion present   (not implemented)

RESULT: PASS
```

### JSON

```bash
fusaops conform gofusa --format json
```

```json
{
  "binary": "/usr/local/bin/gofusa",
  "tool": "gofusa",
  "toolVersion": "0.24.0",
  "language": "go",
  "specVersion": "1.8",
  "generated": "2026-06-10T12:00:00Z",
  "results": [
    {
      "id": "version/line-format",
      "section": "§9.1",
      "level": "MUST",
      "name": "version line format",
      "status": "PASS"
    }
  ]
}
```

---

## Machine-readable artifacts

The kit ships two sets of companion artifacts:

### `spec/schemas/`

Nine JSON Schema (draft-07) files — one per document kind — extracted from the
spec prose.  External validators and IDE tooling can reference these directly.

| File | Spec section | Document kind |
|---|---|---|
| `common-header.schema.json` | §3.1 | common envelope |
| `check-report.schema.json` | §4 | `check --format json` output |
| `trace-matrix.schema.json` | §5 | `trace --format json` output |
| `qualify-report.schema.json` | §6 | `qualify --output` file |
| `sbom.schema.json` | §7 | `sbom.json` |
| `audit-manifest.schema.json` | §8 | `manifest.json` |
| `capabilities.schema.json` | §9.1 | `capabilities --format json` |
| `version.schema.json` | §9.1 | `version --format json` |
| `gap-report.schema.json` | §9.3 | `<standard> --format json` |

### `spec/vectors/`

Seven golden reference JSON documents plus `fingerprint-cases.json` with
pre-computed SHA-256 fingerprint values for the §4.2 algorithm.  Tools can
compare their output structure against these to verify conformance without
running FuSaOps.

---

## Using in CI

```yaml
# .github/workflows/conformance.yml
- name: Conformance check
  run: |
    go install github.com/SoundMatt/FuSaOps/cmd/fusaops@latest
    fusaops conform gofusa   # exits 1 if any MUST check fails
```

Per §16 step 7 of `docs/x-fusa-spec.md`, passing `fusaops conform <binary>` is
a **MUST** baseline for onboarding a new x-FuSa tool.

---

## Programmatic API

The `conform` package is importable for embedding in tool CI:

```go
import "github.com/SoundMatt/FuSaOps/conform"

rep, err := conform.Run("gofusa", conform.Options{})
if err != nil {
    log.Fatal(err)
}
if rep.HasFailures() {
    // one or more MUST checks failed
    os.Exit(1)
}
pass, fail, skip := rep.Summary()
```

Inject a `RunFunc` to test without a real binary on PATH:

```go
rep, err := conform.Run("fake-fusa", conform.Options{
    TempDir: t.TempDir(),
    RunFunc: func(dir, binary string, args ...string) ([]byte, []byte, int) {
        // return fake outputs
    },
})
```

---

## See also

- [`docs/x-fusa-spec.md`](x-fusa-spec.md) — master specification (v1.8, frozen)
- [`docs/commands/conform.md`](commands/conform.md) — full flag reference
- [`docs/extending.md`](extending.md) — onboarding a new language tool
- [`spec/schemas/`](../spec/schemas/) — JSON Schema files
- [`spec/vectors/`](../spec/vectors/) — golden reference documents
