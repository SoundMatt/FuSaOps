# x-FuSa Tool Specification

**Spec version:** 1.5.0 · **Status:** Normative · **Owner:** FuSaOps

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
  "configVersion": "1.0",                 // MUST. config-format version — own series (see below)
  "project": { "name": "FuSaOps", "version": "0.1.0" },  // MUST: project.name; version SHOULD default "0.1.0"
  "standard": "iso26262",                 // MUST. canonical standard id — see §2.4.1
  "asil": "ASIL-C",                       // SHOULD: include exactly ONE integrity field —
                                          //   asil (iso26262) | sil (iec61508) | dal (do178c).
                                          //   OMIT the other two entirely (do not emit explicit nulls).
  "sourceDirs": ["."],                    // MAY. see scope note
  "excludePatterns": ["vendor/**", "build/**"], // MAY. see scope note
  "strict": false                         // MAY. default gate strictness
}
```

A tool MUST read `project.name`, `standard`, and the integrity field. Unknown
keys MUST be ignored (forward compatibility). The integrity field is the **only
one of `asil`/`sil`/`dal` present** — the irrelevant keys are omitted, not set to
`null` (this keeps `omitempty`-style unmarshalling unambiguous). A tool MUST also
accept a legacy flat `"project": "name"` string and normalise it to the nested
form, but MUST emit the nested `{ "name", "version" }` from `init` (with
`version` defaulting to `"0.1.0"` absent `--project-version`).

**`configVersion` is its own series**, starting at `"1.0"`. It tracks the
**config-file** format only and advances **only when §1.2.1 changes** — it does
**not** follow the spec version. A tool implementing **any spec v1.x** still
writes `configVersion: "1.0"` from `init` because the config schema has not
changed since v1.0. (Distinct key from report `schemaVersion` §3 and tool
`specVersion` §9.1; the three series evolve independently.)

**`sourceDirs` / `excludePatterns` scope (MUST).** Every command that walks
source — `check`, `trace`, `fmea`, `coupling`, `coverage`, `boundary`, `cyber`,
`vuln` — MUST honour both. Metadata commands (`version`, `init`) ignore them.
Patterns are gitignore-style globs relative to `--dir`.

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

`file` (when used as a fallback key) MUST be **project-relative**, the same rule
as §4 `location.file`, so it matches. See §4.1 for how `check` matches a finding
to a disposition and how that affects the exit code.

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
scan single-line comments and, **in languages that support them**, MUST also scan
inside multi-line block comments (`/* … */`); languages without block comments
(Python, shell, Ruby) are exempt from that part. A tool MUST accept exactly one
ID per line. A single-ID tool MUST treat any token after the first as
**malformed** and surface a `WARNING` finding (category `requirement`); a tool
that opts into multi-ID lines treats every whitespace-separated token as an ID. A
malformed annotation (e.g. missing ID) MUST likewise be a `WARNING`, never
silently dropped.

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
| `--strict` | `check` `trace`; MAY on standards cmds | Promote warnings/gaps to a non-zero exit (standards semantics in §9.3) |
| `--gaps` | `trace` | Output only requirements with no test tag |
| `--no-color` | all | Disable ANSI colour (see §2.6) |

Long flags use `--kebab-case`. Any command whose JSON FuSaOps consumes MUST
support `--format json`. **Command-specific flags** (e.g. `init --force`,
`version --format json`, go-FuSa's `--no-summary`) are permitted but MUST be
additive and MUST NOT change the schemas here.

**`--output` redirects the report (MUST).** When `--output <file>` is given, a
tool MUST write the report to that file and MUST NOT also write it to stdout
(progress/warning lines on stderr are fine) — so FuSaOps gets a clean stream.

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

`iso26262` · `iec61508` · `do178c` · `iso21434` · `iec62443-4-1` ·
`iec62443-4-2` · `misra-c` · `misra-cpp` · `autosar-cpp14` · `cert-c` ·
`cert-cpp` · `unece-r155` · `unece-r156`.

A **command name maps to one or more standard ids** (the command name is not
itself an id): `do178` → `do178c`; `unece` → `unece-r155` and/or `unece-r156`;
`iec62443` → `iec62443-4-1` and/or `iec62443-4-2`; `misra` → `misra-c` (C
projects) and/or `misra-cpp` (C++), emitting `misra-c-gap-report.json` and/or
`misra-cpp-gap-report.json` respectively.

For a multi-part command, supporting **only one part is conformant** — the
emitted gap-report's `standard` field identifies which; a tool that supports both
SHOULD emit both gap-reports.

There is no `"ISO 26262"` form anywhere in the JSON. A clause reference is the
separate `clause` field (e.g. `"6.4.4"`). Consumers MUST treat an unrecognised id
verbatim (pass-through), never reject it.

### 2.5 JSON formatting

UTF-8, 2-space indented, RFC 3339 timestamps (`generatedAt`). Field names are
`lowerCamelCase` (`ruleId`, not `rule_id`). Every top-level JSON document **MUST**
carry `"schemaVersion"` — **except** the `version --format json` response, which
carries `specVersion` instead (see the §3 scope paragraph).

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
| `configVersion` | `.fusa.json` (§1.2.1) | config-file format version — own series, starts `1.0` |
| `schemaVersion` | report documents (§3) | the spec version the document conforms to |
| `specVersion` | `version --format json` (§9.1) | the spec version the tool implements |

The three series evolve independently and MUST NOT be conflated under one key
name. `schemaVersion` and `specVersion` track the spec version; `configVersion`
moves only when the config schema (§1.2.1) itself changes.

---

## 3. Common envelope

**Scope.** "Report documents" — the `--format json` outputs of `check` (§4),
`trace` (§5), `qualify` (§6), `report` (§9.1), and the standards gap-reports
(§9.3) — MUST carry the envelope below. The following are **not** report
documents and carry only their own documented fields (exempt from the envelope,
and from the §2.5 "every document MUST carry `schemaVersion`" rule where noted):
**file-format artefacts** `sbom.json` / `provenance.json` /
`artifact-manifest.json` (§7) and audit-pack `manifest.json` (§8) — these still
carry `schemaVersion`; and the **`version --format json`** response (§9.1) — a
command-status response that carries `specVersion` instead. This resolves the
apparent "every document MUST" contradiction.

A report document MUST carry these self-describing header fields so an aggregated
artefact stays attributable:

```jsonc
{
  "schemaVersion": "1.5",        // MUST. the spec version this document conforms to (MAJOR.MINOR)
  "tool":        "go-FuSa",      // MUST. human-readable tool name
  "toolVersion": "0.23.0",       // MUST. tool semver
  "language":    "go",           // MUST. go | c | cpp | …
  "generatedAt": "2026-06-10T13:54:40Z",  // MUST. RFC 3339
  "projectRoot": "/abs/path",    // MUST. the --dir value verbatim (see note)
  "project":     "my-project",   // SHOULD
  "standard":    "iso26262"      // SHOULD — canonical id (§2.4.1); FuSaOps routes/groups on this
  // "asil": "ASIL-C"            // MAY — exactly one of asil|sil|dal, same rule as §1.2.1 (omit the others)
  // "error": "…"                // present ONLY on a runtime error (with exit 3); omit otherwise
}
```

**`error` (MUST when runtime error, else absent).** It MUST be present (a
non-empty string) when a runtime error occurred while a partial document was
still emitted (paired with exit `3`); otherwise it MUST be **omitted** (do not
emit `"error": null` in normal documents).

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

`summary` counts are **by severity across all findings regardless of
disposition** — a disposition changes only the exit-code gate (§4.1), never a
finding's severity or its presence in the counts.

### Finding (canonical superset)

```jsonc
{
  "ruleId":   "LINT001",                 // MUST. lowerCamelCase key, stable rule id
  "severity": "ERROR",                   // MUST. ERROR|WARNING|INFO
  "message":  "function exceeds 60 lines",   // MUST
  "location": {                          // MUST be an object (not flat file/line)
    "file":      "src/foo.go",           // MUST. relative to projectRoot (--dir) — see below
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

**Path relativity (MUST).** `location.file` MUST be a path **relative to
`projectRoot`** (the `--dir` value), using `/` separators — never an absolute
path. This is what makes the §4.2 fingerprint identical across tools and machines
(host vs. container), and it makes the SARIF `uri` naturally project-relative.

**Region indexing.** `line`/`column`/`endLine`/`endColumn` are **1-indexed**; the
end position is **inclusive**, matching SARIF. A point location omits the `end*`
fields.

**`--format sarif`.** When a tool emits SARIF it MUST emit **SARIF 2.1.0** with a
`physicalLocation` on every result (this is what GitHub Code Scanning requires).

**`fingerprint` adoption.** It is `SHOULD` today. It is **expected to become
`MUST`** when FuSaOps begins consuming `diff` (a future MINOR bump, §13) — until
every tool emits it, cross-tool `diff` is unusable.

`summaryTable` (go-FuSa) MAY be present and is ignored by FuSaOps.

### 4.1 Dispositions & the exit code

Disposition support is **SHOULD** for a tool. When a tool does **not** implement
it, every finding is treated as open and FuSaOps reads the exit code at face
value. When a tool **does** implement it, the rules below are MUST.

A disposition entry records a **waiver decision** on a finding. `accepted` and
`deferred` are waivers that suppress the gate; `rejected` is **not** a waiver.

**Value semantics (MUST).**

| `disposition` | Meaning | Gates? |
|---|---|---|
| absent / `"open"` | no waiver decision | **yes** |
| `"accepted"` | waiver granted (e.g. false positive, risk accepted) | no |
| `"deferred"` | waiver granted for now (tracked to fix later) | no |
| `"rejected"` | a proposed waiver was **denied** — the finding remains actionable | **yes** |

> `rejected` does **not** mean "the finding was rejected as invalid." It means
> the *waiver request* was rejected, so the finding is still open and MUST still
> gate. (Recording it keeps the denied-waiver decision auditable.)

**Matching (MUST).** To decide a finding's disposition, a tool MUST match by
`fingerprint` (§4.2) when both the finding and a disposition entry carry one; it
MAY fall back to `ruleId` + `location.file` + `location.line`; and it MAY support
a rule-level accept (`ruleId` only) when an entry omits file/line. A rule-level
entry suppresses **every** finding for that rule project-wide, so it SHOULD carry
an explanatory `note` given its broad scope.

**Orphaned dispositions (SHOULD).** When an **`accepted` or `deferred`** entry
matches **no** finding in the current run (e.g. a fallback `ruleId+file+line`
entry stranded after a refactor moved the code), `check` SHOULD emit a `WARNING`
finding (category `config`) naming it, so a team does not silently lose a waiver.
An orphaned **`rejected`** entry SHOULD be **silent** — the denied finding was
resolved, which is success, not a lost waiver.

**Exit code (MUST).** `check` gates only on **open** findings (absent/`open`/
`rejected`). A finding with `disposition:"accepted"` or `"deferred"` MUST remain
in the JSON (marked via the `disposition` field, not removed) but MUST NOT by
itself cause exit `1`. FuSaOps therefore trusts the exit code, and MAY
additionally read `.fusa-dispositions.json` (§1.2.3) using the same matching
rule.

### 4.2 `fingerprint` (canonical algorithm, MUST when emitted)

`fingerprint = "sha256:" + lowercase_hex( SHA-256( utf8( canonical ) ) )` where

```
canonical = ruleId + "\x1f" + location.file + "\x1f" + normalizedMessage
```

- `ruleId` and `location.file` are used **verbatim** (no normalisation — the
  digits in `LINT001` and the path are significant). `location.file` is the
  project-relative path (§4) with `/` separators, so the same finding hashes
  identically regardless of where the repo is checked out.
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

`--gaps` selects only requirements with **no tag of kind `test` OR `sec-test`**.
In `--format json` it filters the `requirements[]` and `tags[]` arrays to those
untested requirements, but `coverage` MUST still report the **full** totals (so
the gap set is visible without distorting the percentage).

**Counting (MUST).** Per requirement:
- `tracedRequirements` — counts it if it has **≥1 tag of any kind** (`impl`,
  `test`, or `sec-test`).
- `testedRequirements` — counts it if it has a `test` **or** `sec-test` tag (a
  security test is a test).
- `secTestedRequirements` — counts it only if it has a `sec-test` tag.

So a requirement with only a `sec-test` tag counts toward all three; one with
only an `impl` tag counts toward `traced` but is a *tested* gap.

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
    // requirementId/file/kind MUST; line SHOULD (1-indexed). file MUST be
    // project-relative (same rule as §4 location.file).
    { "requirementId": "REQ-FO-CORE001", "file": "src/x.go", "line": 12, "kind": "impl" }
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
counts **every** case; `passed` counts `PASS`; `failed` counts **`FAIL` only**
(not `SKIP`/`ERROR`). So `passed + failed` need not equal `total` — the remainder
is skipped/errored, visible in `results[]`.

**`hash` (reproducible, MUST when emitted).** It MUST be independent of run time,
**of array ordering, and of JSON key ordering**, so it is computed over a
*canonical* serialisation, not the pretty-printed output:

1. **Sort `results[]` by `name` ascending** (so parallel/non-deterministic test
   order does not change the hash).
2. **Remove** the `hash` member and **set** `generatedAt` to `""` (both vary or
   are self-referential).
3. Serialise per **RFC 8785 (JSON Canonicalization Scheme)** — UTF-8, keys sorted
   lexicographically at every level, no insignificant whitespace, numbers in
   shortest round-trip form.
4. `hash = "sha256:" + lowercase_hex( SHA-256( canonical_bytes ) )`.

"Field order shown above" is **not** a valid substitute — Go, Python, and
`nlohmann::json` order keys differently. RFC 8785's lexicographic ordering is the
single rule that makes two tools agree. The same procedure applies to any other
self-integrity `hash` field in this spec.

---

## 7. `release` — SBOM, provenance, manifest

`<lang>fusa release [--dir <path>] [--output-dir <dir>] [--spdx-version 2.2|2.3|3.0.1] [--full]`

MUST write **`sbom.json`** (this exact name) into the output dir. `--output-dir`
**defaults to the project root** (the `--dir` value) when omitted, and a tool
MUST **create it** if it does not exist (do not fail). MAY *also* write an SPDX
document; `--spdx-version` **defaults to `2.3`**. `sbom.json` is a file-format
artefact (envelope-exempt per §3) and =

```jsonc
{
  "schemaVersion": "1.5",
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

**`provenance.json` / `artifact-manifest.json`.** File-format artefacts
(envelope-exempt, §3); **not consumed by FuSaOps in v1** (they ride along inside
the audit-pack as opaque evidence). Minimal bodies:

```jsonc
// provenance.json
{ "schemaVersion": "1.5", "format": "x-FuSa provenance v1", "generatedAt": "…",
  "module": "…", "builder": "github-actions", "vcsRevision": "f8127ea",
  "vcsModified": false, "os": "linux", "arch": "amd64" }

// artifact-manifest.json
{ "schemaVersion": "1.5", "format": "x-FuSa manifest v1", "generatedAt": "…",
  "artifacts": [ { "path": "sbom.json", "sha256": "<bare-hex>" } ] }
```

Fields beyond these are tool-defined.

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
  "schemaVersion": "1.5",
  "tool": "go-FuSa", "toolVersion": "0.23.0",   // toolVersion key, aligned with §3 (artefact is still envelope-exempt)
  "module": "…",                         // project/module identity
  "files": [ { "path": "sbom.json", "size": 1234, "sha256": "<bare-hex>" } ]  // paths relative to ZIP root
}
```

`files[].sha256` is **bare lowercase hex** (the key names the algorithm) per the
§2.7 hash convention — distinct from the SBOM's `algo:value` `hash`, which is
intentional, not an inconsistency.

**Contents (MUST).** The pack MUST include `manifest.json` plus every §1.2 input
file and every §1.3 generated file that exists at the project root — **except
`audit-pack.zip` itself**, which MUST be excluded from its own contents (it is in
the §1.3 list but cannot contain itself). `path` values in the manifest are the
entry names at the ZIP root.

> **Note.** `audit-pack` collects from the **project root**. `release --full`
> runs `audit-pack` last in the same directory, so the pack is complete. But
> `release --output-dir <non-root>` writes evidence elsewhere; a *separate*
> later `audit-pack` would then miss it. For a complete standalone pack, run
> `release` with the default output dir (the project root).

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
json` (exactly three fields, no envelope):

```json
{ "tool": "go-FuSa", "version": "0.23.0", "specVersion": "1.5" }
```

`specVersion` is the spec the tool implements — distinct from a document's
`schemaVersion` (§2.8).

**`init` (MUST).** Creates `.fusa.json` (§1.2.1, with `project.name`, `standard`,
and the integrity field populated) and `.fusa-reqs.json` containing
`{ "requirements": [] }`. It operates **per file**: it creates each target that
is missing and leaves an existing one untouched (a one-line stderr note), rather
than aborting the whole command — so a repo with `.fusa.json` but no
`.fusa-reqs.json` gets the missing file created. `--force` **overwrites
completely** (it does not merge) any target. A tool SHOULD source the config
values from flags (`--name`, `--standard`, and one of `--asil`/`--sil`/`--dal`)
and/or interactive prompts; if a required value is missing **and stdin is not a
TTY** (CI), `init` MUST exit `2` rather than prompt or write a placeholder
config. It MAY scaffold additional structure (`.github/`, hooks) and MAY offer
`--migrate` (§1.2).

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

`iso26262` · `iec61508` · `do178` · `iso21434` · `iec62443` · `misra` · `unece` ·
`slsa` · `sas` · `sci` · `badge` · `disposition` · `pr` · `hooks` · `sign` ·
`template` · `req` · `impact` · `metrics` · `fix` · `analyze` · `lint`.

Command → standard id mapping is in §2.4.1. Two of these commands have no
gap-report shape and are clarified here:

- **`slsa`** writes a SLSA build-provenance **attestation** in in-toto format to
  `provenance.intoto.jsonl` — distinct from `release`'s native `provenance.json`
  (§7): `release` records build metadata in the x-FuSa schema, `slsa` produces
  the SLSA/in-toto attestation for the supply-chain toolchain. It is not a
  gap-report command.
- **`disposition`** manages `.fusa-dispositions.json` (§1.2.3) — it adds, lists,
  and shows waiver decisions; it does not itself gate.

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
  status to `gap` (fail-safe). `summary` MUST satisfy the invariant
  `satisfied + partial + gaps = total`.
- **`objectives[].findings`** are **rule id strings** (e.g. `"LINT001"`), not
  §4.2 fingerprints — a gap-report is not bound to one `check` run, so it uses the
  run-stable rule id.
- **`--strict`** (MAY) on a standards command exits `1` when any objective is
  `gap`; `partial` does **not** trip `--strict` (it is a known-incomplete, not a
  failure).

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
| finding `fingerprint` algo (SHOULD; →MUST when `diff` lands) | ▫️ add | ▫️ add | ▫️ add |
| location `endLine/endColumn` (new) | ▫️ add | ▫️ | ▫️ |
| standards `<std>-gap-report.json` canonical | ⚠️ per-cmd shapes | ⚠️ | ⚠️ `objectives` vs other |

The `req`/`impact`/`metrics`/`lint`/`fix`/`iec62443`/`slsa` commands are §9.3
optional and **not consumed by FuSaOps v1** — intentionally absent from the
audited rows above.

> **Reference split.** go-FuSa is the **schema** reference; until it adopts exit
> codes `2`/`3`, **c-FuSa is the exit-code-semantics reference**. A conformant
> tool needs both — neither tool is fully conformant at this snapshot.

**Net change-set to reach conformance** (unchanged MUST count from 1.0, plus the
new shared MUSTs `schemaVersion`/envelope/exit-3/no-color which apply to all
three):

- **c-FuSa:** nest `location` + add `remediation` (check); adopt
  `requirements/tags/coverage` `trace` schema **incl. the new
  `secTestedRequirements` counter** (§5); `qualify --output` with
  `total/passed/failed`; emit `sbom.json` with `algo:value` hashes; single-ZIP
  `audit-pack` + `manifest.json`; `.fusa-reqs.json` (delete the stray no-dot
  file) **+ duplicate-id ERROR check** on load (§1.2.2); lowercase evidence
  filenames; **flip `--spdx-version` default to `2.3`** (currently `3.0.1`, §7);
  **change `release --output-dir` default to the project root** (currently
  `.cfusa_release/`, §7); **declare itself a multi-ID-annotation tool** so
  existing `req REQ-A REQ-B` lines don't become WARNINGs (§1.4); emit
  project-relative `location.file`/`tags[].file` (§4/§5). `qualify.hash` is MAY —
  fine to omit rather than add a JCS serialiser in C.
- **cpp-FuSa:** primary `check --format json` → `ruleId`, nested `location`,
  `remediation` (rename `fix`).
- **go-FuSa:** exit `2` for usage errors; the additive resolution fields
  (`category`, `standard`+`clause`, `fingerprint`, `location` regions); and the
  canonical standards gap-report key (`objectives`).
- **all three:** `schemaVersion` on every JSON doc; envelope
  `tool/toolVersion/language`; exit `3` for runtime errors; `--no-color`/
  `NO_COLOR`; `.fusa.json` per §1.2.1; `location.file` project-relative (§4).

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
| `coverage` | tool-defined | `{ lines:{covered,total,pct}, mutation:{score}, dal? }` — `pct`/`score` are **percentages 0–100** (e.g. `75.3` = 75.3%, **not** 0–1.0); `dal` is the string form (e.g. `"DAL-A"`) |
| `diff` | tool-defined; **blocked on fingerprint adoption** (§4.2 is SHOULD) — unusable cross-tool until all tools emit fingerprints | `{ added:[fingerprint], removed:[fingerprint], unchanged:N }`; baseline is a prior `check --format json`, given via `--baseline <file>` |
| `hara` → `.fusa-hara.json` | **input** file; the `hara` command validates/normalises it (and scaffolds a template if absent), output tool-defined | `{ hazards:[{id, hazard, severity, exposure, controllability, asil, safetyGoal}] }` |
| `sas` → `sas.json`/`sas.md` | tool-defined; **conflict**: go md-only vs cpp `sas.json`+`md` | `sas.json` (envelope + tool-defined body) plus `sas.md` |
| `sci` → `sci.json` | tool-defined; **conflict**: go stdout-only vs cpp `sci.json` | `sci.json` (envelope + tool-defined body) |
| `boundary` → `.dot`/`.mermaid` | tool-defined graph text | no JSON contract in v1 |
| `verify` → `.fusa-evidence.json` | tool-defined | `{ passed, failed, suites:[ {name, passed, failed, tests:[{name, result}]} ] }` (`result` per §6) |

---

## 14. Changelog

### 1.5.0 — 2026-06-10 (fifth review round: go-FuSa / c-FuSa / cpp-FuSa)

- **Path relativity completed:** `tags[].file` (§5) and disposition fallback
  `file` (§1.2.3) MUST be project-relative too — closes the last gap where the
  §4.2 fingerprint / disposition match could silently break.
- **qualify hash determinism (§6):** MUST sort `results[]` by `name` before
  hashing (parallel/non-deterministic test order no longer changes the hash);
  clarified `failed` = `FAIL` only, so `passed + failed` need not equal `total`.
- **audit-pack self-reference (§8):** `audit-pack.zip` MUST be excluded from its
  own contents.
- **`--output` (§2.2):** when given, a tool MUST NOT also write the report to
  stdout (clean stream for FuSaOps).
- **init in CI (§9.1):** missing required value + no TTY ⇒ exit `2` (no prompt,
  no placeholder config); added the explicit `version --format json` example.
- **Disposition refinements (§4.1):** orphaned `rejected` entry is **silent** (the
  denied finding was resolved); only orphaned `accepted`/`deferred` warn.
- **slsa / disposition commands described (§9.3):** `slsa` → `provenance.intoto.jsonl`
  (in-toto attestation, distinct from `release`'s `provenance.json`);
  `disposition` manages `.fusa-dispositions.json`.
- **Envelope (§3):** the one-of `asil`/`sil`/`dal` rule applies to the envelope too.
- **§2.4.1:** multi-part standards — supporting one part is conformant, both
  SHOULD emit both gap-reports.
- **§13:** `coverage` units clarified (`75.3` = 75.3%, not 0–1.0).
- **§11:** added c-FuSa `release --output-dir` default change
  (`.cfusa_release/` → project root) and project-relative paths.
- Stale "spec v1.3" reference in §1.2.1 → "any spec v1.x".

### 1.4.0 — 2026-06-10 (fourth review round: go-FuSa / c-FuSa / cpp-FuSa)

- **`location.file` MUST be project-relative (§4):** the missing rule that makes

- **`location.file` MUST be project-relative (§4):** the missing rule that makes
  the §4.2 fingerprint actually identical across tools/machines (and the SARIF
  `uri` project-relative).
- **Standard ids/commands (§2.4.1, §9.3):** added `iec62443-4-1`/`iec62443-4-2`
  + `iec62443` command, and the `slsa` command; documented the command→id map
  (`do178`→`do178c`, `misra`→`misra-c`/`misra-cpp` with per-language
  gap-reports, etc.).
- **release (§7):** `--output-dir` defaults to the project root and is created
  if absent; §8 notes the standalone-`audit-pack` caveat for non-root output.
- **Counting (§5):** defined `tracedRequirements` (≥1 tag of any kind).
- **`summary` (§4):** counts are by severity across **all** findings regardless
  of disposition.
- **`version --format json` (§3/§2.5):** explicitly envelope-exempt (carries
  `specVersion`, not `schemaVersion`) — resolves the §2.5 contradiction.
- **gap-report (§9.3):** `satisfied + partial + gaps = total` invariant; standards
  `--strict` added to the §2.2 flag table.
- **Dispositions (§4.1):** rule-level entries SHOULD carry a `note` (broad scope).
- **init (§9.1):** SHOULD source values from `--name`/`--standard`/`--asil|sil|dal`
  or prompts.
- **§13:** `diff` baseline = prior `check` json via `--baseline`; `coverage`
  units (percentages 0–100, `dal` string); `hara` command purpose described.
- **§11:** surfaced the c-FuSa v1.3+ deltas (secTested counter, duplicate-id
  ERROR, `--spdx-version` default flip to 2.3, multi-ID declaration,
  project-relative `location.file`).

### 1.3.0 — 2026-06-10 (third review round: go-FuSa / c-FuSa / cpp-FuSa)

- **qualify `hash` determinism (§6):** computed over an **RFC 8785 (JCS)**
  canonical serialisation (lexicographic keys), with `hash` removed and
  `generatedAt:""` — "field order shown above" is explicitly rejected.
- **Envelope scope (§3):** report documents carry the envelope; **file-format
  artefacts** (`sbom.json`, `provenance.json`, `artifact-manifest.json`,
  audit-pack `manifest.json`) are envelope-exempt — resolves the
  "every document MUST" contradiction. `error` is omitted unless a runtime error
  occurred (no `"error": null` default).
- **`rejected` disposition (§4.1):** clarified — it means a *waiver was denied*;
  the finding stays **open and gates**. Added a value-semantics table and an
  **orphaned-disposition WARNING** rule (stale waiver after a refactor).
- **sec-test counting (§5):** a `sec-test` tag counts toward **both**
  `testedRequirements` and `secTestedRequirements`; `--gaps` excludes
  requirements with a `test` **or** `sec-test` tag.
- **provenance/manifest bodies (§7):** minimal schemas added; marked not consumed
  by FuSaOps v1. `--spdx-version` default = `2.3`.
- **SARIF pinned (§4):** SARIF **2.1.0** with a location on every result.
- **`configVersion` own series (§1.2.1/§2.8):** starts `1.0`, advances only on
  config-schema change — decoupled from the spec version.
- **init (§9.1):** per-file create (missing only), `--force` overwrites;
  `project.version` defaults `0.1.0`.
- **Scope/edge fixes:** `sourceDirs`/`excludePatterns` apply to source-walking
  commands (§1.2.1); UNECE ids `unece-r155`/`unece-r156` (§2.4.1); block-comment
  scanning only in languages that have them, extra-token-after-ID is malformed
  (§1.4); `tags[].line` SHOULD (§5); standards `--strict` gates on `gap` not
  `partial` (§9.3); manifest uses `toolVersion` (§8); fingerprint shown as a real
  SHOULD gap in §11; `§2.8` typo fixed.

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
