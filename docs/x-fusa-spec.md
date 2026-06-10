# x-FuSa Tool Specification

**Spec version:** 1.0.0 · **Status:** Normative · **Owner:** FuSaOps

This is the **master contract** every x-FuSa tool (go-FuSa, c-FuSa, cpp-FuSa, and
future tools) implements. It defines the CLI surface, the machine-readable output
schemas, the file/naming conventions, and the exit-code semantics that let
FuSaOps orchestrate any conforming tool **without tool-specific code**.

It is a **superset**: it unions the features the three tools have today, picks
one canonical shape wherever they currently disagree, and adds a small number of
fields that improve resolution. Tool-specific *analysis rules* are out of
scope — those belong to each tool. This document governs only the *interface*.

> **Conformance language.** "MUST" / "MUST NOT" / "SHOULD" / "MAY" follow
> RFC 2119. A tool is **conformant** when it satisfies every MUST for the
> commands it implements. FuSaOps only requires the MUSTs; it tolerates missing
> SHOULD/MAY fields.

The canonical reference implementation is **go-FuSa**. Where this spec adds a
field go-FuSa does not yet emit, that field is marked **(new)** and listed in
§11 as a go-FuSa adoption item.

---

## 1. Naming conventions

### 1.1 Binary

`<lang>fusa` — `gofusa`, `cfusa`, `cpfusa`. A future tool for language `L` is
`Lfusa` (or the closest readable contraction).

### 1.2 Input / config files (dot-prefixed, un-tool-prefixed)

| File | Purpose |
|---|---|
| `.fusa.json` | Project config (standard, ASIL/SIL, project name) |
| `.fusa-reqs.json` | Requirements registry |
| `.fusa-hara.json` | Hazard analysis & risk assessment |
| `.fusa-evidence.json` | Test-evidence bundle (`verify`) |
| `.fusa-dispositions.json` | Finding dispositions (optional) |
| `.fusa-problems.json` | Problem-report log (optional) |

These are **un-prefixed** (`.fusa-…`, not `.cfusa-…`): one tool owns a repo, so
prefixing is redundant and breaks cross-tool tooling. A tool MUST read
`.fusa-reqs.json`.

### 1.3 Generated evidence (lowercase kebab-case, at project root)

`sbom.json` · `provenance.json` · `artifact-manifest.json` ·
`safety-case.{json,md,mermaid}` · `tara.{json,md}` · `fmea.{json,csv}` ·
`coupling-report.json` · `cyber-report.json` · `vuln.json` ·
`qualify-report.json` · `boundary.{dot,mermaid}` · `audit-pack.zip` ·
`<standard>-gap-report.json` (e.g. `iso26262-gap-report.json`).

Evidence filenames MUST be lowercase kebab-case. (No `SAFETY_CASE.md`,
`TARA.md`, no `<project>-<ver>.spdx.json` as the primary SBOM.)

### 1.4 Source annotations

```
//fusa:req  <REQ-ID>      // one requirement ID per line (canonical)
//fusa:test <REQ-ID>      // one requirement ID per line (canonical)
//fusa:sec-test <REQ-ID>  // security test (optional)
```

The comment leader follows the host language (`//`, `/* … */`, `#`). A tool
MUST accept exactly one ID per line. A tool MAY additionally accept
space-separated IDs on one line, but MUST NOT *require* it.

---

## 2. Common CLI conventions

### 2.1 Invocation

```
<lang>fusa <command> [flags]
```

### 2.2 Shared flags

| Flag | Applies to | Meaning |
|---|---|---|
| `--dir <path>` | all | Project root (default: cwd) |
| `--format <fmt>` | reporting cmds | `text` \| `json` \| `html` \| `sarif` (subset per command) |
| `--output <file>` | reporting/qualify/audit-pack | Write to a **file** (default: stdout, or a documented default path) |
| `--output-dir <dir>` | `release` | Directory for the generated bundle |
| `--strict` | `check`, `trace` | Promote warnings/gaps to a non-zero exit |
| `--only <tools>` | (FuSaOps only) | Not a tool flag |

Long flags use `--kebab-case`. `--format json` MUST emit the schema in §§4–8.

### 2.3 Exit codes (MUST)

| Code | Meaning |
|---|---|
| `0` | Success; no gate failure |
| `1` | Gate failure (ERROR findings, coverage gap under `--strict`, etc.) **or** a runtime error |
| `2` | Usage error (bad flag/args) |

A non-zero exit MUST NOT prevent the requested `--format json`/`--output`
artefact from being written when findings merely indicate failure (FuSaOps reads
the artefact regardless of exit code).

### 2.4 Severity enum (MUST)

`"ERROR"` · `"WARNING"` · `"INFO"` — uppercase, exactly these three strings. An
unknown value MUST be treated by consumers as `INFO` (fail-safe, never dropped).

### 2.5 JSON formatting

UTF-8, 2-space indented, RFC 3339 timestamps (`generatedAt`). Field names are
`lowerCamelCase` (`ruleId`, not `rule_id`). Every top-level document SHOULD carry
`"schemaVersion": "1.0"`.

---

## 3. Common envelope (new)

Every `--format json` report document MUST carry these self-describing header
fields so an aggregated artefact stays attributable:

```jsonc
{
  "schemaVersion": "1.0",
  "tool":        "go-FuSa",      // human-readable tool name
  "toolVersion": "0.23.0",       // tool semver
  "language":    "go",           // go | c | cpp | …
  "generatedAt": "2026-06-10T13:54:40Z",
  "projectRoot": "/abs/path",
  "project":     "my-project",   // optional
  "standard":    "iso26262",     // optional default standard
  "asil":        "ASIL-C"        // optional
}
```

`tool`, `toolVersion`, `language` are **(new)** — they let FuSaOps attribute
findings even from a raw artefact, instead of inferring it from the adapter.

---

## 4. `check` — finding report

`<lang>fusa check [--dir <path>] [--format text|json|html|sarif] [--output <file>] [--strict]`

The everyday gate. Exit `1` on any ERROR finding (and on WARNING under
`--strict`). JSON document = §3 envelope plus:

```jsonc
{
  "...envelope": "...",
  "findings": [ Finding, ... ],
  "summary": { "total": 3, "errors": 1, "warnings": 1, "infos": 1 }
}
```

### Finding (canonical superset)

```jsonc
{
  "ruleId":   "LINT001",                 // MUST. lowerCamelCase key, stable rule id
  "severity": "ERROR",                   // MUST. ERROR|WARNING|INFO
  "message":  "function exceeds 60 lines",   // MUST
  "location": {                          // MUST be an object (not flat file/line)
    "file":      "src/foo.go",           // MUST
    "line":      42,                     // SHOULD (omit/0 if not line-scoped)
    "column":    5,                      // MAY
    "endLine":   48,                     // MAY (new — region for editors/SARIF)
    "endColumn": 1                       // MAY (new)
  },
  "category":    "lint",                 // SHOULD (new for go-FuSa). lint|safety|security|coverage|style|…
  "standard":    "ISO 26262",            // MAY (new for go-FuSa). normative standard the rule maps to
  "clause":      "6.4.4",                // MAY (new for go-FuSa). clause within that standard
  "remediation": "split into smaller functions",  // SHOULD. (NOT "fix")
  "fingerprint": "a1b2c3…"               // MAY (new). stable hash of {ruleId,file,normalisedMessage} for baseline/diff
}
```

**Rationale for the additions (better resolution):** `category` groups findings
in the aggregate UI; `standard`+`clause` map each finding to a normative
requirement (huge traceability win, already in cpp-FuSa); `endLine`/`endColumn`
give SARIF/editors a real region; `fingerprint` gives `diff`/baseline a stable
identity across runs. All are additive and backward-compatible.

`summaryTable` (go-FuSa) MAY be present and is ignored by FuSaOps.

---

## 5. `trace` — requirement traceability matrix

`<lang>fusa trace [--dir <path>] [--format text|json|html] [--output <file>] [--req-coverage N] [--sec-tested N]`

JSON document = §3 envelope plus the matrix. **This is the canonical go-FuSa
shape — flat `{total,traced,tested,matrix[]}` is NOT conformant.**

```jsonc
{
  "...envelope": "...",
  "requirements": [
    { "id": "REQ-FO-CORE001", "title": "…", "text": "…",
      "standard": "ISO 26262", "level": "HLR", "asil": "ASIL-C" }
  ],
  "tags": [
    { "requirementId": "REQ-FO-CORE001", "file": "x.go", "line": 12, "kind": "impl" }
  ],
  "coverage": {
    "totalRequirements":     101,
    "tracedRequirements":    101,
    "testedRequirements":    101,
    "secTestedRequirements": 0
  }
}
```

- `tags[].kind` MUST be one of `"impl"` · `"test"` · `"sec-test"`.
- `requirements[].level` is the requirement tier (`HLR`/`LLR`); `asil`/`sil` is
  separate. (Do not overload `level` with `SHALL`/`SHOULD`.)
- `--req-coverage N` / `--sec-tested N` exit `1` below the threshold.

---

## 6. `qualify` — tool qualification record

`<lang>fusa qualify [--dir <path>] [--format text|json] [--output <file>]`

MUST support `--output <file>`. Writes `qualify-report.json` by default. JSON =
§3 envelope plus:

```jsonc
{
  "...envelope": "...",
  "total":  44,                          // MUST (NOT "tests_passed"+"tests_failed" only)
  "passed": 44,                          // MUST
  "failed": 0,                           // MUST
  "results": [                           // SHOULD
    { "name": "rule-LINT001-known-answer", "result": "PASS" }   // result: PASS|FAIL
  ],
  "hash": "sha256:…"                     // MAY (integrity of the record)
}
```

---

## 7. `release` — SBOM, provenance, manifest

`<lang>fusa release [--dir <path>] [--output-dir <dir>] [--spdx-version 2.2|2.3|3.0.1] [--full]`

MUST write **`sbom.json`** (this exact name) into the output dir. MAY *also*
write an SPDX document. `sbom.json` =

```jsonc
{
  "schemaVersion": "1.0",
  "format":      "x-FuSa SBOM v1",
  "generatedAt": "…",
  "module":      "github.com/SoundMatt/go-FuSa",   // MUST (module/package identity)
  "language":    "go",                              // SHOULD
  "components": [                                    // MUST (deps; may be empty)
    { "name": "golang.org/x/sys", "version": "v0.1.0", "hash": "h1:…" }
  ]
}
```

`provenance.json` and `artifact-manifest.json` follow the same envelope; `--full`
additionally runs fmea/boundary/vuln/audit-pack. (A tool whose ecosystem has no
external deps emits `"components": []` — still valid.)

---

## 8. `audit-pack` — evidence bundle

`<lang>fusa audit-pack [--dir <path>] [--output <file>]`

MUST produce a **single ZIP** at `--output` (default `audit-pack.zip`) — not a
directory tree. The ZIP MUST contain a top-level **`manifest.json`** (lowercase)
listing every packed file with its SHA-256:

```jsonc
{
  "schemaVersion": "1.0",
  "tool": "go-FuSa", "version": "0.23.0",
  "module": "…",                         // project/module identity
  "files": [ { "path": "sbom.json", "size": 1234, "sha256": "…" } ]
}
```

FuSaOps nests each tool's `audit-pack.zip` under `components/<tool>/` in its own
unified pack, so the per-tool pack MUST be a self-contained, openable ZIP.

---

## 9. Command catalog

### 9.1 Required (FuSaOps-consumed — MUST)

`version` · `init` · `check` · `trace` · `qualify` · `release` · `audit-pack` ·
`report`.

- `version` MUST print a parseable `<tool> <semver>` line.
- `report [--format text|json|html|sarif] [--output <file>]` generates an
  aggregate report **without** failing on findings (evidence generation).

### 9.2 Recommended (safety evidence — SHOULD)

`verify` · `hara` · `tara` · `fmea` · `safety-case` · `coupling` · `cyber` ·
`vuln` · `boundary` · `coverage` · `diff`.

### 9.3 Optional (standards roll-ups — MAY)

`iso26262` · `iec61508` · `do178` · `iso21434` · `misra` · `unece` · `sas` ·
`sci` · `badge` · `disposition` · `pr` · `hooks` · `sign` · `template`.

A standards command (e.g. `iso26262`) SHOULD emit `<standard>-gap-report.json`
with a `{ requirements/objectives: [...], summary: { satisfied, gaps } }` shape
so FuSaOps v0.3 can roll standards up.

---

## 10. How FuSaOps consumes the contract

| FuSaOps capability | Tool command | Reads |
|---|---|---|
| aggregate report | `check --format json --output <f>` | §4 findings → `report.Finding` |
| `fusaops trace` | `trace --format json` | §5 matrix → `trace.Matrix` |
| trace qualification column | `qualify --output <f>` | §6 `{total,passed,failed}` |
| `fusaops sbom` | `release --output-dir <d>` → `sbom.json` | §7 `{module,components}` |
| `fusaops audit-pack` | `audit-pack --output <f>` | §8 ZIP, nested verbatim |

FuSaOps' Go types in `report/`, `trace/`, `sbom/`, `auditpack/` are the
authoritative decoders; keep this spec and those structs in lock-step.

---

## 11. Current conformance & change-set

Snapshot 2026-06-10. ✅ conforms · ⚠️ gap (MUST) · ▫️ nice-to-have (SHOULD/MAY).

| Item | go-FuSa | c-FuSa | cpp-FuSa |
|---|---|---|---|
| severity enum `ERROR/WARNING/INFO` | ✅ | ✅ | ✅ |
| tag kinds `impl/test/sec-test` | ✅ | ✅ | ✅ |
| `.fusa-reqs.json` (un-prefixed) | ✅ | ⚠️ `.cfusa-reqs.json` (+ stray `cfusa-reqs.json`) | ✅ |
| check finding `ruleId` (camel) | ✅ | ✅ | ⚠️ `rule_id` in `check-report.json` |
| check finding **nested `location`** | ✅ | ⚠️ flat `file`/`line` | ⚠️ flat in primary JSON |
| check `remediation` (not `fix`) | ✅ | ▫️ none | ⚠️ `fix` |
| trace **`requirements/tags/coverage`** schema | ✅ | ⚠️ `total/traced/tested/matrix` | ✅ |
| qualify `--output` + `total/passed/failed` | ✅ | ⚠️ stdout, `tests_passed/_failed` | ✅ |
| `sbom.json` name + `{module,components}` | ✅ | ⚠️ `<proj>.spdx.json` only | ✅ |
| audit-pack = single **ZIP** + `manifest.json` | ✅ | ⚠️ directory + `MANIFEST.json` | ✅ |
| evidence filenames lowercase-kebab | ✅ | ⚠️ `SAFETY_CASE.md`,`TARA.md` | ✅ |
| envelope `tool/toolVersion/language` (new) | ▫️ add | ▫️ add | ▫️ add |
| finding `category` (new for go) | ▫️ add | ✅ has it | ✅ has it |
| finding `standard`+`clause` (new for go) | ▫️ add | ▫️ | ✅ has it |
| finding `fingerprint` (new) | ▫️ add | ▫️ | ▫️ |
| location `endLine/endColumn` (new) | ▫️ add | ▫️ | ▫️ |

**Net change-set to reach conformance:**

- **c-FuSa (5 MUSTs):** nest `location` + add `remediation` in `check`; adopt the
  `requirements/tags/coverage` `trace` schema; `qualify --output` with
  `total/passed/failed`; emit `sbom.json`; make `audit-pack` a single ZIP with
  `manifest.json`. Plus naming nits: standardise `.fusa-reqs.json` (delete the
  stray no-dot file) and lowercase the evidence filenames.
- **cpp-FuSa (2 MUSTs):** make the primary `check --format json` emit `ruleId`
  (camel), nested `location`, and `remediation` (it already has the richer
  fields — just align the key casing and the `fix`→`remediation` rename in the
  main report path; its SARIF path already uses `ruleId`).
- **go-FuSa (better resolution — all additive, no MUSTs):** add `category`,
  `standard`+`clause`, `fingerprint`, `location.endLine/endColumn`, and the
  envelope `tool/toolVersion/language`. These make go-FuSa the true superset
  master and give the aggregate far finer resolution.

---

## 12. Versioning

This spec is semver-versioned (`schemaVersion` "MAJOR.MINOR"). Additive,
backward-compatible fields bump MINOR; a breaking change to a MUST bumps MAJOR
and is coordinated across all tools and FuSaOps in lock-step. Tools and FuSaOps
SHOULD emit and accept any equal-MAJOR schema.
