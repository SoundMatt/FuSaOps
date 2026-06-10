# x-FuSa Tool Specification

**Spec version:** 1.2.0 · **Status:** Normative · **Owner:** FuSaOps

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
| `.fusa-dispositions.json` | Finding dispositions | §1.2.3 (read by FuSaOps) |
| `.fusa-problems.json` | Problem-report log | tool-defined |

Names are **un-prefixed** (`.fusa-…`, not `.cfusa-…`): one tool owns a repo, so
prefixing is redundant and breaks cross-tool tooling.

**Migration.** When the canonical `.fusa.json` / `.fusa-reqs.json` is absent, a
tool SHOULD fall back to a legacy prefixed name (`.gofusa.json`, `.cfusa.json`,
`.cpfusa.json`, …) and emit a one-line deprecation warning to **stderr**. The
un-prefixed name is canonical and MUST be preferred when both exist. A tool MAY
provide `init --migrate` to rewrite a legacy prefixed config (and flat
`project` string, §1.2.1) into the canonical file and key shape.

#### 1.2.1 `.fusa.json` schema (MUST)

```jsonc
{
  "configVersion": "1.2",                 // MUST. config-format version (distinct from report schemaVersion)
  "project": { "name": "FuSaOps", "version": "0.2.0" },  // MUST: project.name
  "standard": "iso26262",                 // MUST. canonical standard id — see §2.4.1
  "asil": "ASIL-C",                       // SHOULD: include exactly ONE integrity field —
                                          //   asil (iso26262) | sil (iec61508) | dal (do178c).
                                          //   OMIT the other two entirely (do not emit explicit nulls).
  "sourceDirs": ["."],                    // MAY
  "excludePatterns": ["vendor/**", "build/**"], // MAY
  "strict": false                         // MAY. default gate strictness
}
```

A tool MUST read `project.name`, `standard`, and the integrity field. Unknown
keys MUST be ignored (forward compatibility). The integrity field is the **only
one of `asil`/`sil`/`dal` present** — the irrelevant keys are omitted, not set to
`null` (this keeps `omitempty`-style unmarshalling unambiguous). A tool MUST also
accept a legacy flat `"project": "name"` string and normalise it to the nested
form, but MUST emit the nested `{ "name", "version" }` from `init`.

`configVersion` is the **config-file** format version; it is intentionally a
*different key* from the report `schemaVersion` (§3) and the tool `specVersion`
(§9.1), because the three version independently.

#### 1.2.2 `.fusa-reqs.json` schema (MUST)

```jsonc
{
  "requirements": [                       // MUST. consumers read ONLY this key
    {
      "id":       "REQ-FO-CORE001",       // MUST. unique within the file
      "title":    "Severity classification", // SHOULD
      "text":     "The tool shall …",     // SHOULD (full requirement text)
      "standard": "iso26262",             // MAY. canonical standard id — §2.4.1
      "level":    "HLR",                  // MAY. see §5 for the recommended set
      "asil":     "ASIL-C",               // MAY
      "parent":   "REQ-FO-CORE000"        // MAY. parent requirement id (decomposition)
    }
  ]
}
```

A document MAY wrap additional top-level keys (`project`, `version`,
`generated`); consumers MUST read only `.requirements` and MUST ignore the rest.
A duplicate `id` MUST be reported by `check` as an `ERROR` finding (category
`requirement`); a tool MUST NOT silently merge or drop duplicates.

#### 1.2.3 `.fusa-dispositions.json` schema (read by FuSaOps when present)

```jsonc
{
  "dispositions": [
    {
      "fingerprint": "sha256:a1b2…",      // SHOULD. §4.2 fingerprint (primary match key)
      "ruleId":      "LINT001",           // MAY. fallback match key
      "file":        "src/foo.go",        // MAY. fallback match key
      "line":        42,                  // MAY. fallback match key
      "status":      "accepted",          // MUST. accepted | deferred | rejected
      "note":        "false positive — generated code",  // SHOULD
      "by":          "matt@jellybaby.com",// SHOULD
      "at":          "2026-06-10T12:00:00Z"  // SHOULD. RFC 3339
    }
  ]
}
```

See §4.1 for how `check` matches a finding to a disposition and how that affects
the exit code.

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
scan single-line comments and MUST also scan inside multi-line block comments
(`/* … */`) for annotations. A tool MUST accept exactly one ID per line. A tool
MAY additionally accept space-separated IDs on one line, but MUST NOT *require*
it. A malformed annotation (e.g. missing ID) MUST be surfaced by `check` as a
`WARNING` finding (category `requirement`), never silently dropped.

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
support `--format json`. **Command-specific flags** (e.g. `init --force`,
`version --format json`, go-FuSa's `--no-summary`) are permitted but MUST be
additive and MUST NOT change the schemas here.

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

#### 2.4.1 Standard id (MUST — one representation everywhere)

The `standard` value is a **canonical lowercase id**, used identically in
`.fusa.json`, the envelope, `Finding.standard`, requirements, and gap-reports —
it is an **enum value, never a display string**. Defined ids:

`iso26262` · `iec61508` · `do178c` · `iso21434` · `misra-c` · `misra-cpp` ·
`autosar-cpp14` · `cert-c` · `cert-cpp`.

There is no `"ISO 26262"` form anywhere in the JSON. A clause reference is the
separate `clause` field (e.g. `"6.4.4"`). Consumers MUST treat an unrecognised id
verbatim (pass-through), never reject it.

### 2.5 JSON formatting

UTF-8, 2-space indented, RFC 3339 timestamps (`generatedAt`). Field names are
`lowerCamelCase` (`ruleId`, not `rule_id`). Every top-level JSON document **MUST**
carry `"schemaVersion"` (§3).

### 2.6 Colour (MUST)

When stdout is **not** a TTY, or `--format json` is set, or `--no-color` is
given, or the `NO_COLOR` environment variable is set, a tool MUST NOT emit ANSI
escape codes. This keeps captured subprocess streams clean for FuSaOps.

### 2.7 Hash conventions (MUST)

Two hash field conventions are used, deliberately:

- A field **named for its algorithm** (`sha256`) carries **bare lowercase hex**
  — the key already states the algorithm. Used by `audit-pack` `manifest.json`
  (§8) and any other fixed-SHA-256 integrity field.
- A field **named `hash`** carries `"<algo>:<value>"` because its algorithm
  varies. Used by SBOM `components[].hash` (§7), where `algo` ∈ {`sha256`, `h1`},
  and by the `fingerprint` / `qualify.hash` fields (always `sha256:`).

This is the single rule; the two shapes are not an inconsistency.

### 2.8 Versioning keys (MUST — three distinct keys)

| Key | Where | Means |
|---|---|---|
| `configVersion` | `.fusa.json` (§1.2.1) | config-file format version |
| `schemaVersion` | every report document (§3) | the spec version the document conforms to |
| `specVersion` | `version --format json` (§9.1) | the spec version the tool implements |

They version independently and MUST NOT be conflated under one key name.

---

## 3. Common envelope

Every `--format json` report document MUST carry these self-describing header
fields so an aggregated artefact stays attributable:

```jsonc
{
  "schemaVersion": "1.2",        // MUST. the spec version this document conforms to (MAJOR.MINOR)
  "tool":        "go-FuSa",      // MUST. human-readable tool name
  "toolVersion": "0.23.0",       // MUST. tool semver
  "language":    "go",           // MUST. go | c | cpp | …
  "generatedAt": "2026-06-10T13:54:40Z",  // MUST. RFC 3339
  "projectRoot": "/abs/path",    // MUST. the --dir value verbatim (see note)
  "project":     "my-project",   // SHOULD
  "standard":    "iso26262",     // SHOULD — canonical id (§2.4.1); FuSaOps routes/groups on this
  "asil":        "ASIL-C",       // MAY
  "error":       null            // MAY. non-null string ⇒ a runtime error occurred (paired with exit 3)
}
```

**`projectRoot` across boundaries.** It is informational. The same source tree
has different absolute paths on host vs. in a container (`/Users/x/p` vs.
`/project`), so FuSaOps MUST NOT use `projectRoot` to correlate findings across
components — cross-component identity is the `fingerprint` (§4.2). A tool SHOULD
emit the `--dir` value verbatim (resolved to absolute).

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
  "standard":    "iso26262",             // MAY (new for go-FuSa). canonical standard id (§2.4.1), NOT a display string
  "clause":      "6.4.4",                // MAY (new for go-FuSa). clause within that standard
  "remediation": "split into smaller functions",  // SHOULD. free text, one actionable sentence. (NOT "fix")
  "disposition": "accepted",             // omit when open; accepted|deferred|rejected|open — see §4.1
  "fingerprint": "sha256:a1b2…"          // SHOULD (new). canonical hash — see §4.2
}
```

**`category` (closed, extensible enum).** One of: `lint` · `style` · `safety` ·
`security` · `coverage` · `requirement` · `concurrency` · `supply-chain` ·
`config` · `other`. Consumers MUST map any unrecognised value to `other`.

**Region indexing.** `line`/`column`/`endLine`/`endColumn` are **1-indexed**; the
end position is **inclusive**, matching SARIF. A point location omits the `end*`
fields.

`summaryTable` (go-FuSa) MAY be present and is ignored by FuSaOps.

### 4.1 Dispositions & the exit code

Disposition support is **SHOULD** for a tool. When a tool does **not** implement
it, every finding is treated as open and FuSaOps reads the exit code at face
value. When a tool **does** implement it, the rules below are MUST.

**Open vs. dispositioned.** An open finding omits the `disposition` field (the
explicit string `"open"` is also accepted, and is equivalent to absent). A
`"rejected"` disposition is also open (the rejection was overruled).

**Matching (MUST).** To decide a finding's disposition, a tool MUST match by
`fingerprint` (§4.2) when both the finding and a disposition entry carry one; it
MAY fall back to `ruleId` + `location.file` + `location.line`; and it MAY support
a rule-level accept (`ruleId` only) when an entry omits file/line.

**Exit code (MUST).** `check` gates only on **open** findings. A finding with
`disposition:"accepted"` or `"deferred"` MUST remain in the JSON (marked via the
`disposition` field, not removed) but MUST NOT by itself cause exit `1`. FuSaOps
therefore trusts the exit code, and MAY additionally read
`.fusa-dispositions.json` (§1.2.3) using the same matching rule.

### 4.2 `fingerprint` (canonical algorithm, MUST when emitted)

`fingerprint = "sha256:" + lowercase_hex( SHA-256( utf8( canonical ) ) )` where

```
canonical = ruleId + "\x1f" + location.file + "\x1f" + normalizedMessage
```

- `ruleId` and `location.file` are used **verbatim** (no normalisation — the
  digits in `LINT001` and the path are significant).
- `normalizedMessage` = `message` with: every run of ASCII digits replaced by a
  single `"#"`, all whitespace runs collapsed to one space, then trimmed.
- Apply **Unicode NFC** to the message first **only when it contains non-ASCII
  codepoints**; a pure-ASCII message needs no NFC (so an ASCII-only tool needs no
  Unicode dependency).

(`\x1f` is the ASCII Unit Separator.) Two conforming tools MUST produce identical
fingerprints for identical `(ruleId, file, normalised message)`, so
`diff`/baseline works cross-tool.

---

## 5. `trace` — requirement traceability matrix

`<lang>fusa trace [--dir <path>] [--format text|json|html|md] [--output <file>] [--gaps] [--req-coverage N] [--sec-tested N]`

`--req-coverage N` / `--sec-tested N` take **N as a percentage 0–100** and exit
`1` when the respective coverage is below N. `N = 0` **disables** that gate (it
always passes). On `trace`, `--strict` with no explicit threshold is equivalent
to `--req-coverage 100 --sec-tested 100` (any gap → exit `1`); an explicit
`--req-coverage N` / `--sec-tested N` overrides the implied 100.

`--gaps` selects only requirements with no `test` tag. In `--format json` it
filters the `requirements[]` and `tags[]` arrays to those untested requirements,
but `coverage` MUST still report the **full** totals (so the gap set is visible
without distorting the percentage).

JSON document = §3 envelope plus the matrix. **This is the canonical shape —
flat `{total,traced,tested,matrix[]}` is NOT conformant.**

```jsonc
{
  "...envelope": "...",
  "requirements": [
    { "id": "REQ-FO-CORE001", "title": "…", "text": "…",
      "standard": "iso26262", "level": "HLR", "asil": "ASIL-C" }
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
  "hash": "sha256:…"                     // MAY. reproducible integrity hash — see below
}
```

`results[].result` MUST be one of `PASS` · `FAIL` · `SKIP` · `ERROR`. `total`
counts every case including skipped/errored.

**`hash` (reproducible, MUST when emitted).** It MUST be independent of run time.
Compute it over the document with `hash` set to `null` **and** `generatedAt` set
to `""`, serialised as 2-space-indented UTF-8 JSON in the field order shown
above; then `hash = "sha256:" + lowercase_hex( SHA-256( that_bytes ) )`. Two runs
of identical code on identical inputs MUST therefore yield the same `hash`.

---

## 7. `release` — SBOM, provenance, manifest

`<lang>fusa release [--dir <path>] [--output-dir <dir>] [--spdx-version 2.2|2.3|3.0.1] [--full]`

MUST write **`sbom.json`** (this exact name) into the output dir. MAY *also* write
an SPDX document. `sbom.json` =

```jsonc
{
  "schemaVersion": "1.2",
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

**`--full` (when given).** The base outputs `sbom.json`, `provenance.json`, and
`artifact-manifest.json` are MUST (they do not depend on a §9.2 command).
Additionally, a `--full` run MUST emit the output of **each of the following that
the tool implements**, SHOULD attempt all of them, and MUST emit a one-line
stderr warning for each it skips because the command is not implemented (it MUST
NOT emit an empty/stub file and MUST NOT exit `3` for a *deliberately*
unimplemented component): `fmea.json`, `fmea.csv`, `boundary.dot`,
`boundary.mermaid`, `vuln.json`, and finally `audit-pack.zip`. Because
`audit-pack` bundles the other artefacts, it MUST run **last**.
`provenance.json` and `artifact-manifest.json` carry the §3 envelope.

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
  "schemaVersion": "1.2",
  "tool": "go-FuSa", "version": "0.23.0",
  "module": "…",                         // project/module identity
  "files": [ { "path": "sbom.json", "size": 1234, "sha256": "<bare-hex>" } ]  // paths relative to ZIP root
}
```

`files[].sha256` is **bare lowercase hex** (the key names the algorithm) per the
§2.7 hash convention — distinct from the SBOM's `algo:value` `hash`, which is
intentional, not an inconsistency.

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
json` → `{ "tool": "go-FuSa", "version": "0.23.0", "specVersion": "1.2" }`. This
JSON form does **not** carry the §3 envelope, and uses `specVersion` (the spec
the tool implements) — distinct from a document's `schemaVersion` (§2.8).

**`init` (MUST).** Creates `.fusa.json` (§1.2.1, with `project.name`, `standard`,
and the integrity field populated) and `.fusa-reqs.json` containing
`{ "requirements": [] }`. It MUST refuse to overwrite an existing file without
`--force`; `--force` **overwrites the file completely** (it does not merge). It
MAY scaffold additional structure (`.github/`, hooks) and MAY offer `--migrate`
(§1.2).

**`report` (MUST).** `report [--format text|json|html|sarif|md] [--output <file>]`
**re-runs analysis** on the project root — it does not read a cached
`check-report.json` and has no `--input` flag. It is effectively `check` that
always exits `0` regardless of findings (only `2`/`3` apply); its `--format json`
shape is identical to `check` (§4). It produces one report for one run and does
not aggregate across runs.

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
  "standard": "iso26262",                // canonical id (§2.4.1)
  "objectives": [                        // canonical key (NOT "requirements")
    { "id": "6.4.4", "title": "…", "clause": "6.4.4",
      "status": "satisfied",             // satisfied | partial | gap
      "evidence": ["safety-case.json"],  // artefacts supporting it
      "findings": ["LINT001"] }          // blocking rule ids (stable strings, NOT fingerprints)
  ],
  "summary": { "total": 50, "satisfied": 48, "partial": 1, "gaps": 1 }
}
```

- **`status`** ∈ `satisfied` | `partial` | `gap`. `satisfied` = all required
  evidence present and all clauses met; `partial` = some evidence present but not
  all clauses met; `gap` = no evidence. A consumer MUST map any unrecognised
  status to `gap` (fail-safe).
- **`objectives[].findings`** are **rule id strings** (e.g. `"LINT001"`), not
  §4.2 fingerprints — a gap-report is not bound to one `check` run, so it uses the
  run-stable rule id.

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

> **Reference split.** go-FuSa is the **schema** reference; until it adopts exit
> codes `2`/`3`, **c-FuSa is the exit-code-semantics reference**. A conformant
> tool needs both — neither tool is fully conformant at this snapshot.

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
| `tara` → `tara.json` | tool-defined; **conflict**: go `entries` vs cpp `scenarios` | `"threats": [ {id, asset, threat, attackVector, impact, likelihood, risk, treatment, mitigations:[]} ]` |
| `fmea` → `fmea.json` | tool-defined; **conflict**: cpp has `rpn/occurrence/detectability` | superset entry `{id, item, failureMode, effect, cause, severity, occurrence, detection, rpn, mitigations[]}` |
| `safety-case` → `safety-case.json` | tool-defined; **conflict**: go `{clauses,gaps}` vs cpp `{nodes,edges}` | GSN graph `{ nodes:[{id,type,text}], edges:[{from,to,type}] }` (encodes clauses + gaps) |
| `vuln` → `vuln.json` | tool-defined | finding-list reusing §4 `Finding` shape |
| `cyber` → `cyber-report.json` | tool-defined | finding-list reusing §4 `Finding` shape |
| `coupling` → `coupling-report.json` | tool-defined; **c-FuSa ships a finding-list today** | graph `{ modules:[…], edges:[{from,to,weight}], metrics:{…} }` — ⚠️ a change from the finding-list; do not deepen investment in the list shape |
| `coverage` | tool-defined | `{ lines:{covered,total,pct}, mutation:{score}, dal? }` |
| `diff` | tool-defined; **blocked on fingerprint adoption** (§4.2 is SHOULD) — unusable cross-tool until all tools emit fingerprints | `{ added:[fingerprint], removed:[fingerprint], unchanged:N }` |
| `hara` → `.fusa-hara.json` | input file; output tool-defined | `{ hazards:[{id, hazard, severity, exposure, controllability, asil, safetyGoal}] }` |
| `sas` → `sas.json`/`sas.md` | tool-defined; **conflict**: go md-only vs cpp `sas.json`+`md` | `sas.json` (envelope + tool-defined body) plus `sas.md` |
| `sci` → `sci.json` | tool-defined; **conflict**: go stdout-only vs cpp `sci.json` | `sci.json` (envelope + tool-defined body) |
| `boundary` → `.dot`/`.mermaid` | tool-defined graph text | no JSON contract in v1 |
| `verify` → `.fusa-evidence.json` | tool-defined | `{ passed, failed, suites:[ {name, passed, failed, tests:[{name, result}]} ] }` (`result` per §6) |

---

## 14. Changelog

### 1.2.0 — 2026-06-10 (second review round: go-FuSa / c-FuSa / cpp-FuSa)

- **`standard` casing unified (§2.4.1):** one canonical lowercase id
  (`iso26262`, …) everywhere — config, envelope, `Finding.standard`,
  requirements, gap-reports. Removed all `"ISO 26262"` display forms.
- **Versioning keys split (§2.8):** `configVersion` (config), `schemaVersion`
  (report doc), `specVersion` (`version` command) — three distinct keys.
- **`--full` MUST/SHOULD contradiction fixed (§7):** MUST emit each listed
  artefact the tool *implements*, SHOULD attempt all, MUST stderr-warn per skip;
  `audit-pack` runs last.
- **qualify `hash` made reproducible (§6):** exact serialisation with
  `hash:null` and `generatedAt:""`, 2-space UTF-8, then SHA-256.
- **Dispositions (§4.1):** support is SHOULD (FuSaOps falls back to all-open when
  absent); **matching algorithm** defined (fingerprint, else ruleId+file+line);
  open = field omitted or `"open"`. `.fusa-dispositions.json` schema added
  (§1.2.3).
- **fingerprint scope (§4.2):** digit-normalisation applies to the message only;
  `ruleId`/`file` verbatim; NFC required only for non-ASCII messages.
- **trace (§5):** `--strict` ⇒ `--req-coverage 100 --sec-tested 100`; `N=0`
  disables a gate; `--gaps` JSON behaviour (filters `requirements`/`tags`,
  `coverage` keeps full totals) defined.
- **release (§7):** SBOM/manifest `schemaVersion` → 1.2; hash conventions
  consolidated (§2.7: bare-hex `sha256` field vs `algo:value` `hash`).
- **version JSON (§9.1):** `spec` → `specVersion`; no envelope.
- **report (§9.1):** clarified — re-runs analysis, no `--input`, `check` with
  exit 0.
- **gap-report (§9.3):** `status` unknown→`gap`; `partial` defined;
  `objectives[].findings` are rule ids (not fingerprints).
- **§1.2.1:** `asil`/`sil`/`dal` — include only the relevant key (no explicit
  nulls); duplicate requirement id ⇒ `check` ERROR (§1.2.2); block-comment
  annotation scanning (§1.4); `init --force` overwrites, `init --migrate` added.
- **projectRoot (§3):** informational; FuSaOps correlates by `fingerprint`, not
  path (container/host boundary).
- **§13:** added `sas`/`sci` rows; `tara` gains `mitigations`; `hara` gains
  severity/exposure/controllability; `verify` suite sketch; `diff` flagged
  blocked-on-fingerprint; `coupling` flagged as a shape change from c-FuSa's
  current finding-list.
- **§11:** reference split note (go-FuSa = schema ref, c-FuSa = exit-code ref).

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
