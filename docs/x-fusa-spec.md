# x-FuSa Tool Specification

**Spec version:** 1.1.0 · **Status:** Normative · **Owner:** FuSaOps

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
> commands it implements. FuSaOps requires the MUSTs; it tolerates missing
> SHOULD/MAY fields.

The canonical reference implementation is **go-FuSa**. Where this spec adds a
field go-FuSa does not yet emit, that field is marked **(new)** and listed in
§11 as a go-FuSa adoption item.

> **What FuSaOps consumes in spec v1.** Only the §9.1 *required* commands
> (`version`, `init`, `check`, `trace`, `qualify`, `release`, `audit-pack`,
> `report`) and the §1.2 input files have FuSaOps-consumed schemas. The §9.2
> *recommended* and §9.3 *optional* commands are **tool-defined and not consumed
> by FuSaOps in v1** — see §13 for their status and the canonical direction.

---

## 1. Files & naming

### 1.1 Binary

`<lang>fusa` — `gofusa`, `cfusa`, `cpfusa`. A future tool for language `L` is
`Lfusa` (or the closest readable contraction).

### 1.2 Input / config files (dot-prefixed, un-tool-prefixed, with schema)

| File | Purpose | Schema |
|---|---|---|
| `.fusa.json` | Project config | §1.2.1 (MUST read) |
| `.fusa-reqs.json` | Requirements registry | §1.2.2 (MUST read) |
| `.fusa-hara.json` | Hazard analysis & risk assessment | tool-defined (see §13) |
| `.fusa-evidence.json` | Test-evidence bundle (`verify`) | tool-defined |
| `.fusa-dispositions.json` | Finding dispositions | §9.3 / disposition rules |
| `.fusa-problems.json` | Problem-report log | tool-defined |

Names are **un-prefixed** (`.fusa-…`, not `.cfusa-…`): one tool owns a repo, so
prefixing is redundant and breaks cross-tool tooling.

**Migration.** When the canonical `.fusa.json` / `.fusa-reqs.json` is absent, a
tool SHOULD fall back to a legacy prefixed name (`.gofusa.json`, `.cfusa.json`,
`.cpfusa.json`, …) and emit a one-line deprecation warning to **stderr**. The
un-prefixed name is canonical and MUST be preferred when both exist.

#### 1.2.1 `.fusa.json` schema (MUST)

```jsonc
{
  "schemaVersion": "1.1",                 // MUST
  "project": { "name": "FuSaOps", "version": "0.2.0" },  // MUST: project.name
  "standard": "iso26262",                 // MUST. iso26262|iec61508|do178c|iso21434|…
  "asil": "ASIL-C",                       // SHOULD: exactly one integrity field —
  "sil": null,                            //   asil (ISO 26262) | sil (IEC 61508) |
  "dal": null,                            //   dal (DO-178C). Others null/absent.
  "sourceDirs": ["."],                    // MAY
  "excludePatterns": ["vendor/**", "build/**"], // MAY
  "strict": false                         // MAY. default gate strictness
}
```

A tool MUST read `project.name`, `standard`, and the integrity field. Unknown
keys MUST be ignored (forward compatibility). A tool MAY accept a flat
`{"project":"name","version":"…"}` form, but MUST emit the nested form from
`init`.

#### 1.2.2 `.fusa-reqs.json` schema (MUST)

```jsonc
{
  "requirements": [                       // MUST. consumers read ONLY this key
    {
      "id":       "REQ-FO-CORE001",       // MUST. unique within the file
      "title":    "Severity classification", // SHOULD
      "text":     "The tool shall …",     // SHOULD (full requirement text)
      "standard": "ISO 26262",            // MAY
      "level":    "HLR",                  // MAY. see §5 for the recommended set
      "asil":     "ASIL-C",               // MAY
      "parent":   "REQ-FO-CORE000"        // MAY. parent requirement id (decomposition)
    }
  ]
}
```

A document MAY wrap additional top-level keys (`project`, `version`,
`generated`); consumers MUST read only `.requirements` and MUST ignore the rest.

### 1.3 Generated evidence (lowercase kebab-case, at project root)

`sbom.json` · `provenance.json` · `artifact-manifest.json` ·
`safety-case.{json,md,mermaid}` · `tara.{json,md}` · `fmea.{json,csv}` ·
`coupling-report.json` · `cyber-report.json` · `vuln.json` ·
`qualify-report.json` · `boundary.{dot,mermaid}` · `audit-pack.zip` ·
`<standard>-gap-report.json` (e.g. `iso26262-gap-report.json`).

Evidence filenames MUST be lowercase kebab-case and are case-sensitive. (No
`SAFETY_CASE.md`, `TARA.md`; no `<project>-<ver>.spdx.json` as the *primary*
SBOM — an SPDX document MAY be emitted additionally.)

### 1.4 Source annotations

```
//fusa:req  <REQ-ID>      // one requirement ID per line (canonical)
//fusa:test <REQ-ID>      // one requirement ID per line (canonical)
//fusa:sec-test <REQ-ID>  // security test (optional)
```

The comment leader follows the host language (`//`, `/* … */`, `#`). A tool MUST
accept exactly one ID per line. A tool MAY additionally accept space-separated
IDs on one line, but MUST NOT *require* it. A malformed annotation (e.g. missing
ID) MUST be surfaced by `check` as a `WARNING` finding (category `requirement`),
never silently dropped.

**Tool-defined annotations** such as `//fusa:unsafe`, `//fusa:nolint`, and
`//fusa:file-suppress <RULE>` are *not* part of this contract; they remain
tool-specific. FuSaOps does not consume rule suppressions in spec v1.

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
| `--format <fmt>` | reporting cmds | one of `text` `json` `html` `sarif` `md` (per-command subset; `json` is the machine contract) |
| `--output <file>` | `report` `check` `trace` `qualify` `audit-pack` | Write to a **file** (default: stdout, or the documented default path) |
| `--output-dir <dir>` | `release` | Directory for the generated bundle |
| `--strict` | `check` `trace` | Promote warnings/gaps to a non-zero exit |
| `--gaps` | `trace` | Output only requirements with no test tag |
| `--no-color` | all | Disable ANSI colour (see §2.6) |

Long flags use `--kebab-case`. Any command whose JSON FuSaOps consumes MUST
support `--format json`. **Tool-specific flags** (e.g. go-FuSa's `--no-summary`)
are permitted but MUST be additive and MUST NOT change the schemas here.

### 2.3 Exit codes (MUST)

| Code | Meaning |
|---|---|
| `0` | Success; no gate failure |
| `1` | **Gate failure** — ERROR findings, coverage gap under `--strict`, etc. The tool ran correctly and found problems. |
| `2` | **Usage error** — bad flag/args |
| `3` | **Runtime/internal error** — could not complete analysis (I/O, parse, crash) |

This separates "ran and found problems" (`1`) from "could not run" (`3`) so
FuSaOps can tell a real failure from a finding. A non-zero exit MUST NOT prevent
the requested `--format json`/`--output` artefact from being written when the
cause is a gate failure (`1`); FuSaOps reads the artefact regardless of a `1`.
When a partial document is emitted under a `3`, the envelope MUST set `error`
(§3).

### 2.4 Severity enum (MUST)

`"ERROR"` · `"WARNING"` · `"INFO"` — uppercase, exactly these three strings. An
unknown value MUST be treated by consumers as `INFO` (fail-safe, never dropped).
(Note: the lowercase `errors`/`warnings`/`infos` keys in `summary` (§4) are
*count fields*, distinct from the severity enum — do not conflate.)

### 2.5 JSON formatting

UTF-8, 2-space indented, RFC 3339 timestamps (`generatedAt`). Field names are
`lowerCamelCase` (`ruleId`, not `rule_id`). Every top-level JSON document **MUST**
carry `"schemaVersion"` (§3).

### 2.6 Colour (MUST)

When stdout is **not** a TTY, or `--format json` is set, or `--no-color` is
given, or the `NO_COLOR` environment variable is set, a tool MUST NOT emit ANSI
escape codes. This keeps captured subprocess streams clean for FuSaOps.

---

## 3. Common envelope

Every `--format json` report document MUST carry these self-describing header
fields so an aggregated artefact stays attributable:

```jsonc
{
  "schemaVersion": "1.1",        // MUST. the spec version this document conforms to (MAJOR.MINOR)
  "tool":        "go-FuSa",      // MUST. human-readable tool name
  "toolVersion": "0.23.0",       // MUST. tool semver
  "language":    "go",           // MUST. go | c | cpp | …
  "generatedAt": "2026-06-10T13:54:40Z",  // MUST. RFC 3339
  "projectRoot": "/abs/path",    // MUST
  "project":     "my-project",   // SHOULD
  "standard":    "iso26262",     // SHOULD — FuSaOps routes/groups on this
  "asil":        "ASIL-C",       // MAY
  "error":       null            // MAY. non-null string ⇒ a runtime error occurred (paired with exit 3)
}
```

`tool`, `toolVersion`, `language` are **(new)** — they let FuSaOps attribute a
raw artefact without inferring it from the adapter.

**`schemaVersion` semantics (MUST).** It is the **spec** version the document
conforms to, `MAJOR.MINOR`. A consumer MUST accept any document whose MAJOR
equals a MAJOR it supports; a MINOR bump is additive and **never** invalidates an
older document (a `1.0` document stays conformant under a `1.1` reader). A tool
emits the highest spec MINOR it fully implements. FuSaOps uses `schemaVersion` as
its parser discriminator.

---

## 4. `check` — finding report

`<lang>fusa check [--dir <path>] [--format text|json|html|sarif] [--output <file>] [--strict]`

The everyday gate. Exit `1` on any **open** ERROR finding (and on WARNING under
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
    "line":      42,                     // SHOULD. 1-indexed; 0/omitted = not line-scoped
    "column":    5,                      // MAY. 1-indexed
    "endLine":   48,                     // MAY (new). 1-indexed, inclusive (SARIF region semantics)
    "endColumn": 1                       // MAY (new). 1-indexed, inclusive
  },
  "category":    "lint",                 // SHOULD (new for go-FuSa). closed enum — see below
  "standard":    "ISO 26262",            // MAY (new for go-FuSa). normative standard the rule maps to
  "clause":      "6.4.4",                // MAY (new for go-FuSa). clause within that standard
  "remediation": "split into smaller functions",  // SHOULD. free text, one actionable sentence. (NOT "fix")
  "disposition": "accepted",             // MAY. open(absent)|accepted|deferred|rejected — see §4.1
  "fingerprint": "sha256:a1b2…"          // MAY (new). canonical hash — see §4.2
}
```

**`category` (closed, extensible enum).** One of: `lint` · `style` · `safety` ·
`security` · `coverage` · `requirement` · `concurrency` · `supply-chain` ·
`config` · `other`. Consumers MUST map any unrecognised value to `other`.

**Region indexing.** `line`/`column`/`endLine`/`endColumn` are **1-indexed**; the
end position is **inclusive**, matching SARIF. A point location omits the `end*`
fields.

`summaryTable` (go-FuSa) MAY be present and is ignored by FuSaOps.

### 4.1 Dispositions & the exit code (MUST, when `disposition` is implemented)

The `check` exit code (§2.3) gates only on **open** findings — those with no
`disposition` or `disposition:"rejected"`. A finding with
`disposition:"accepted"` or `"deferred"` MUST remain in the JSON (marked, not
removed) but MUST NOT by itself cause exit `1`. FuSaOps therefore trusts the exit
code and MAY additionally read `.fusa-dispositions.json`.

### 4.2 `fingerprint` (canonical algorithm, MUST when emitted)

`fingerprint = "sha256:" + lowercase_hex( SHA-256( utf8( canonical ) ) )` where

```
canonical = ruleId  + "\x1f" + location.file + "\x1f" + normalizedMessage
normalizedMessage = NFC(message), with every run of ASCII digits replaced by a
                    single "#", all whitespace runs collapsed to one space, trimmed
```

(`\x1f` is the ASCII Unit Separator.) Two conforming tools MUST produce identical
fingerprints for identical `(ruleId, file, normalised message)`, so
`diff`/baseline works cross-tool.

---

## 5. `trace` — requirement traceability matrix

`<lang>fusa trace [--dir <path>] [--format text|json|html|md] [--output <file>] [--gaps] [--req-coverage N] [--sec-tested N]`

`--req-coverage N` / `--sec-tested N` take **N as a percentage 0–100** and exit
`1` when coverage is below N. `--gaps` prints only untested requirements.

JSON document = §3 envelope plus the matrix. **This is the canonical shape —
flat `{total,traced,tested,matrix[]}` is NOT conformant.**

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
- `requirements[].level`: free string; tools SHOULD use one of
  `"HLR"` · `"LLR"` · `"SYS"` · `"SW"`. It is the requirement *tier* and MUST NOT
  be overloaded with `SHALL`/`SHOULD` (that belongs in prose, not `level`).
- A malformed annotation is handled per §1.4 (a `check` WARNING), and is simply
  not counted here.

---

## 6. `qualify` — tool qualification record

`<lang>fusa qualify [--dir <path>] [--format text|json] [--output <file>]`

MUST support `--output <file>`. Writes `qualify-report.json` by default. JSON =
§3 envelope plus:

```jsonc
{
  "...envelope": "...",
  "total":  44,                          // MUST (NOT only tests_passed/tests_failed)
  "passed": 44,                          // MUST
  "failed": 0,                           // MUST
  "results": [                           // SHOULD
    { "name": "rule-LINT001-known-answer", "result": "PASS" }   // PASS|FAIL|SKIP|ERROR
  ],
  "hash": "sha256:…"                     // MAY. "sha256:" + lowercase hex of the report (sans the hash field)
}
```

`results[].result` MUST be one of `PASS` · `FAIL` · `SKIP` · `ERROR`. `total`
counts every case including skipped/errored.

---

## 7. `release` — SBOM, provenance, manifest

`<lang>fusa release [--dir <path>] [--output-dir <dir>] [--spdx-version 2.2|2.3|3.0.1] [--full]`

MUST write **`sbom.json`** (this exact name) into the output dir. MAY *also* write
an SPDX document. `sbom.json` =

```jsonc
{
  "schemaVersion": "1.1",
  "format":      "x-FuSa SBOM v1",
  "generatedAt": "…",
  "module":      "github.com/SoundMatt/go-FuSa",   // MUST. identity — see below
  "language":    "go",                              // SHOULD
  "components": [                                    // MUST (deps; may be empty)
    { "name": "golang.org/x/sys", "version": "v0.1.0", "hash": "sha256:…" }
  ]
}
```

- **`module`**: the module/package identity. For ecosystems without a module path
  (C/C++), it MUST be the canonical repository URL, or `<project>@<version>` when
  no URL exists.
- **`components[].hash`**: `"<algo>:<value>"` with `algo` ∈ {`sha256`, `h1`}.
  Tools SHOULD use `sha256` for cross-language comparability; Go MAY carry the
  native `h1:…` from `go.sum`. (A bare hash with no `algo:` prefix is
  non-conformant.)
- A tool whose ecosystem has no external deps emits `"components": []` — valid.

**`--full` (MUST produce, when given):** in addition to `sbom.json`,
`provenance.json`, and `artifact-manifest.json`, a `--full` run MUST emit
`fmea.json`, `fmea.csv`, `boundary.dot`, `boundary.mermaid`, `vuln.json`, and
`audit-pack.zip`. `provenance.json` and `artifact-manifest.json` carry the §3
envelope.

---

## 8. `audit-pack` — evidence bundle

`<lang>fusa audit-pack [--dir <path>] [--output <file>]`

MUST produce a **single ZIP** at `--output` (default `audit-pack.zip`) — not a
directory tree.

**Internal layout (MUST).** Entries are **flat at the ZIP root** (no
`evidence/` subdirectory). A top-level **`manifest.json`** (lowercase;
case-sensitive — ZIP entry names are case-sensitive) lists every packed file with
its SHA-256:

```jsonc
{
  "schemaVersion": "1.1",
  "tool": "go-FuSa", "version": "0.23.0",
  "module": "…",                         // project/module identity
  "files": [ { "path": "sbom.json", "size": 1234, "sha256": "…" } ]  // paths relative to ZIP root
}
```

**Contents (MUST).** The pack MUST include `manifest.json` plus every §1.2 input
file and every §1.3 generated file that exists at the project root. `path` values
in the manifest are the entry names at the ZIP root.

FuSaOps nests each tool's `audit-pack.zip` under `components/<tool>/` in its own
unified pack, so the per-tool pack MUST be a self-contained, openable ZIP.

---

## 9. Command catalog

### 9.1 Required (FuSaOps-consumed — MUST)

`version` · `init` · `check` · `trace` · `qualify` · `release` · `audit-pack` ·
`report`.

**`version` (MUST).** Prints to stdout a single line matching the regex
`^(\S+) (\d+\.\d+\.\d+[0-9A-Za-z.+-]*)$` — tool token, one space, semver
(e.g. `go-FuSa 0.23.0`). FuSaOps extracts the version as the second capture
group of the first stdout line. A tool SHOULD also support `version --format
json` → `{ "tool": "go-FuSa", "version": "0.23.0", "spec": "1.1" }`.

**`init` (MUST).** Creates `.fusa.json` (§1.2.1, with `project.name`, `standard`,
and the integrity field populated) and `.fusa-reqs.json` containing
`{ "requirements": [] }`. It MUST refuse to overwrite an existing file without
`--force`. It MAY scaffold additional structure (`.github/`, hooks).

**`report` (MUST).** `report [--format text|json|html|sarif|md] [--output <file>]`
generates an aggregate report for a **single** run and MUST exit `0` regardless
of findings (it never gate-fails; only `2`/`3` apply). Its `--format json` shape
is identical to `check` (§4). It does not aggregate across multiple runs.

### 9.2 Recommended (safety evidence — SHOULD; not consumed by FuSaOps v1)

`verify` · `hara` · `tara` · `fmea` · `safety-case` · `coupling` · `cyber` ·
`vuln` · `boundary` · `coverage` · `diff`. Their JSON is **tool-defined** in spec
v1 — see §13. A command in this group MAY support `--format json`; if it does and
FuSaOps later consumes it, the canonical schema is added per §12/§13.

### 9.3 Optional (standards & workflow — MAY)

`iso26262` · `iec61508` · `do178` · `iso21434` · `misra` · `unece` · `sas` ·
`sci` · `badge` · `disposition` · `pr` · `hooks` · `sign` · `template` · `req` ·
`impact` · `metrics` · `fix` · `analyze` · `lint`.

A standards command (`iso26262`, `iec61508`, `do178`, …) that emits JSON MUST use
the canonical **gap-report** schema:

```jsonc
{
  "...envelope": "...",
  "standard": "iso26262",
  "objectives": [                        // canonical key (NOT "requirements")
    { "id": "6.4.4", "title": "…", "clause": "6.4.4",
      "status": "satisfied",             // satisfied | partial | gap
      "evidence": ["safety-case.json"],  // artefacts supporting it
      "findings": ["LINT001"] }          // rule ids / finding refs that block it
  ],
  "summary": { "total": 50, "satisfied": 48, "partial": 1, "gaps": 1 }
}
```

This is written to `<standard>-gap-report.json` (§1.3) and is the shape FuSaOps
v0.3 rolls up.

---

## 10. How FuSaOps consumes the contract

| FuSaOps capability | Tool command | Reads |
|---|---|---|
| aggregate report | `check --format json --output <f>` | §4 findings → `report.Finding` |
| `fusaops trace` | `trace --format json` | §5 matrix → `trace.Matrix` |
| trace qualification column | `qualify --output <f>` | §6 `{total,passed,failed}` |
| `fusaops sbom` | `release --output-dir <d>` → `sbom.json` | §7 `{module,components}` |
| `fusaops audit-pack` | `audit-pack --output <f>` | §8 ZIP, nested verbatim |
| version probe | `version` | §9.1 regex |
| project metadata | `.fusa.json` | §1.2.1 |

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
| `.fusa.json` schema (§1.2.1) | ▫️ subset | ⚠️ drifted keys | ⚠️ drifted keys |
| check finding `ruleId` (camel) | ✅ | ✅ | ⚠️ `rule_id` in primary JSON |
| check finding **nested `location`** | ✅ | ⚠️ flat `file`/`line` | ⚠️ flat in primary JSON |
| check `remediation` (not `fix`) | ✅ | ▫️ none | ⚠️ `fix` |
| trace **`requirements/tags/coverage`** schema | ✅ | ⚠️ `total/traced/tested/matrix` | ✅ |
| qualify `--output` + `total/passed/failed` | ✅ | ⚠️ stdout, `tests_passed/_failed` | ✅ |
| `sbom.json` name + `{module,components}` | ✅ | ⚠️ `<proj>.spdx.json` only | ✅ |
| `components[].hash` = `algo:value` | ▫️ `h1:` ok | ⚠️ | ▫️ |
| audit-pack = single **ZIP** + `manifest.json` | ✅ | ⚠️ directory + `MANIFEST.json` | ✅ |
| evidence filenames lowercase-kebab | ✅ | ⚠️ `SAFETY_CASE.md`,`TARA.md` | ✅ |
| exit `2` for usage errors | ⚠️ returns `1` | ⚠️ verify | ⚠️ verify |
| exit `3` for runtime errors (new) | ▫️ add | ▫️ add | ▫️ add |
| `--no-color`/`NO_COLOR` (new) | ▫️ add | ▫️ add | ▫️ add |
| envelope `tool/toolVersion/language` (new) | ▫️ add | ▫️ add | ▫️ add |
| `schemaVersion` MUST on every doc (new) | ▫️ add | ▫️ add | ▫️ add |
| finding `category` (new for go) | ▫️ add | ✅ has it | ✅ has it |
| finding `standard`+`clause` (new for go) | ▫️ add | ▫️ | ✅ has it |
| finding `fingerprint` algo (new) | ▫️ add | ▫️ | ▫️ |
| location `endLine/endColumn` (new) | ▫️ add | ▫️ | ▫️ |
| standards `<std>-gap-report.json` canonical | ⚠️ per-cmd shapes | ⚠️ | ⚠️ `objectives` vs other |

The `req`/`impact`/`metrics`/`lint`/`fix` commands are §9.3 optional and **not
consumed by FuSaOps v1** — intentionally absent from the audited rows above.

**Net change-set to reach conformance** (unchanged MUST count from 1.0, plus the
new shared MUSTs `schemaVersion`/envelope/exit-3/no-color which apply to all
three):

- **c-FuSa:** nest `location` + add `remediation` (check); adopt
  `requirements/tags/coverage` `trace` schema; `qualify --output` with
  `total/passed/failed`; emit `sbom.json` with `algo:value` hashes; single-ZIP
  `audit-pack` + `manifest.json`; `.fusa-reqs.json` (delete the stray no-dot
  file); lowercase evidence filenames.
- **cpp-FuSa:** primary `check --format json` → `ruleId`, nested `location`,
  `remediation` (rename `fix`).
- **go-FuSa:** exit `2` for usage errors; the additive resolution fields
  (`category`, `standard`+`clause`, `fingerprint`, `location` regions); and the
  canonical standards gap-report key (`objectives`).
- **all three:** `schemaVersion` on every JSON doc; envelope
  `tool/toolVersion/language`; exit `3` for runtime errors; `--no-color`/
  `NO_COLOR`; `.fusa.json` per §1.2.1.

---

## 12. Versioning

This spec is semver-versioned. `schemaVersion` in every document is the
`MAJOR.MINOR` it conforms to (§3). Additive, backward-compatible changes bump
MINOR and never invalidate existing documents; a breaking change to a MUST bumps
MAJOR and is coordinated across all tools and FuSaOps in lock-step. Tools and
FuSaOps MUST accept any equal-MAJOR document. When FuSaOps begins consuming a
§9.2/§9.3 command (adding its canonical schema, §13), that is an additive MINOR
bump.

---

## 13. SHOULD-command schema status

FuSaOps does **not** consume these in spec v1; their JSON is **tool-defined**.
This section records the known cross-tool conflicts and the *canonical
direction* the schema will take **when** FuSaOps adds consumption (a future MINOR
bump). Tools SHOULD NOT assume cross-tool compatibility for these until then.

| Command / file | Status in v1 | Canonical direction (future) |
|---|---|---|
| `tara` → `tara.json` | tool-defined; **conflict**: go `entries` vs cpp `scenarios` | top-level `"threats": [ {id, asset, threat, attackVector, impact, likelihood, risk, treatment} ]` |
| `fmea` → `fmea.json` | tool-defined; **conflict**: cpp has `rpn/occurrence/detectability` | superset entry `{id, item, failureMode, effect, cause, severity, occurrence, detection, rpn, mitigations[]}` |
| `safety-case` → `safety-case.json` | tool-defined; **conflict**: go `{clauses,gaps}` vs cpp `{nodes,edges}` | GSN graph `{ nodes:[{id,type,text}], edges:[{from,to,type}] }` (encodes clauses + gaps) |
| `vuln` → `vuln.json` | tool-defined | finding-list reusing §4 `Finding` shape |
| `cyber` → `cyber-report.json` | tool-defined | finding-list reusing §4 `Finding` shape |
| `coupling` → `coupling-report.json` | tool-defined | `{ modules:[…], edges:[{from,to,weight}], metrics:{…} }` |
| `coverage` | tool-defined | `{ lines:{covered,total,pct}, mutation:{score}, dal? }` |
| `diff` | tool-defined | `{ added:[fingerprint], removed:[fingerprint], unchanged:N }` (uses §4.2 fingerprints) |
| `hara` → `.fusa-hara.json` | input file; output tool-defined | hazard list `{ hazards:[{id,hazard,asil,safetyGoal}] }` |
| `boundary` → `.dot`/`.mermaid` | tool-defined graph text | no JSON contract in v1 |
| `verify` → `.fusa-evidence.json` | tool-defined | `{ passed, failed, suites:[…] }` |

---

## 14. Changelog

### 1.1.0 — 2026-06-10 (incorporates go-FuSa / c-FuSa / cpp-FuSa review)

- **Envelope/§3:** `schemaVersion` now MUST (resolves §2.5↔§3 conflict) with
  explicit "spec version, equal-MAJOR accepted, MINOR never invalidates"
  semantics; `standard` promoted to SHOULD; added optional `error` field.
- **§1.2:** added full schemas for `.fusa.json` (§1.2.1) and `.fusa-reqs.json`
  (§1.2.2); added a config migration/fallback rule.
- **§2.3 exit codes:** added `3` (runtime/internal error) to separate crashes
  from gate failures.
- **§2.2/§2.6:** added `--no-color`/`NO_COLOR` (MUST when non-TTY/json), `--gaps`
  on `trace`, `md` to the `--format` enum, `audit-pack` to the `--output` row.
- **§4 Finding:** `category` is now a closed enum; `fingerprint` has a canonical
  SHA-256 algorithm (§4.2); region fields defined 1-indexed/inclusive (SARIF);
  `remediation` format guidance; dispositions vs exit code defined (§4.1).
- **§5 trace:** `--req-coverage`/`--sec-tested` are percentages; `level` value
  set recommended; malformed-tag handling defined.
- **§6 qualify:** `result` enum extended with `SKIP`/`ERROR`; `hash` algorithm
  specified.
- **§7 release:** `--full` MUST-produce file list made normative; `module` for
  non-module ecosystems defined; `components[].hash` format (`algo:value`).
- **§8 audit-pack:** ZIP internal layout (flat root), MUST-include file set, and
  case-sensitivity specified.
- **§9.1:** exact `version` format + regex + JSON form; `init` file outputs;
  `report` vs `check` distinction (exit 0, same schema, single run).
- **§9.3:** canonical standards **gap-report** schema (`objectives`/`status`/
  `summary`) defined.
- **§11:** added exit-2 audit rows and the new shared-MUST rows; noted
  `req/impact/metrics` are out of FuSaOps v1 scope.
- **§13 (new):** explicit status + canonical direction for every §9.2/§9.3
  command, including the active `tara`/`fmea`/`safety-case` conflicts.

### 1.0.0 — 2026-06-10

Initial master contract.
