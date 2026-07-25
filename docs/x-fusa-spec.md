# x-FuSa Tool Specification

**Spec version:** 1.10.10 · **Status:** Normative · **Owner:** FuSaOps

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

> **What FuSaOps consumes in spec v1.** The §9.1 *required* commands
> (`version`, `init`, `check`, `trace`, `qualify`, `release`, `audit-pack`,
> `report`) and the §1.2 input files have FuSaOps-consumed schemas. The §9.2
> `comp` command is also consumed (v1.70.0+, §10). The remaining §9.2
> *recommended* and §9.3 *optional* commands are **tool-defined and not consumed
> by FuSaOps** — see §13 for their status and the canonical direction.

---

## 1. Files & naming

### 1.1 Tool, language & binary registry

Three identifiers travel together and MUST come from this registry. A new tool
adds a row (one PR against this spec) before release, so there is exactly one
canonical spelling of each.

| `language` id | binary | human name (`tool`) | image |
|---|---|---|---|
| `go`  | `gofusa` | `go-FuSa`  | `ghcr.io/soundmatt/go-fusa`  |
| `c`   | `cfusa`  | `c-FuSa`   | `ghcr.io/soundmatt/c-fusa`   |
| `cpp` | `cpfusa` | `cpp-FuSa` | `ghcr.io/soundmatt/cpp-fusa` |
| `rust` *(reserved)* | `rsfusa` | `rust-FuSa` | `ghcr.io/soundmatt/rust-fusa` |
| `python` *(reserved)* | `pyfusa` | `py-FuSa` | `ghcr.io/soundmatt/py-fusa` |
| `java` *(reserved)* | `jfusa` | `java-FuSa` | `ghcr.io/soundmatt/java-fusa` |
| `ada` *(reserved)* | `adafusa` | `ada-FuSa` | `ghcr.io/soundmatt/ada-fusa` |

- **`language`** is the lowercase id emitted in every document header (§3.1) and
  used as the rule-id namespace (§1.5). It MUST be unique and stable.
- **binary** = `<contraction>fusa`, lowercase, on `PATH`.
- **human name** (`tool`) = `<Language>-FuSa`.
- **image** = `ghcr.io/soundmatt/<language>-fusa` (note: full language id, not the
  binary contraction — §15).

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
A tool MUST validate `.fusa-reqs.json` for duplicate `id`s **whenever it reads
the file** (e.g. in `check` or `trace`); a duplicate MUST surface as an `ERROR`
finding (category `requirement`) — a tool MUST NOT silently merge or drop
duplicates.

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
`coupling-report.json` · `cyber-report.json` · `vuln.json` · `qualify-report.json` ·
`comp-report.json` (from `comp`, §9.2) · `boundary.{dot,mermaid}` · `audit-pack.zip` ·
`<standard>-gap-report.json` (e.g. `iso26262-gap-report.json`, `slsa-gap-report.json`).

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

### 1.5 Identifier naming (rules & requirements)

The taxonomies (severity, category, tag kind, standard id, document kind) are
already common enums. This section makes the **leaf identifiers** — rule/lint
codes and requirement ids — common too, so a cross-language aggregate is
unambiguous and a new tool inherits a ready-made scheme.

#### 1.5.1 Rule ids (`ruleId`)

- **Local form (MUST).** A `ruleId` MUST match
  `^[A-Z][A-Z0-9]*(-[A-Z0-9.]+)*$` — uppercase, no spaces, stable across runs.
  This covers `LINT001`, `FUSA004`, `MISRA-15.5`, `AUTOSAR-A7-1-1`,
  `CERT-INT30-C`, `CWE-787`. A `ruleId` MUST be unique **within a tool**.
- **Qualified form (MUST cross-tool).** `ruleId` is only tool-local, so the
  canonical **cross-language** identity is `"<language>/<ruleId>"`
  (`go/LINT001`, `cpp/MISRA-15.5`). Any reference that can span languages — the
  FuSaOps aggregate, dispositions applied across components, gap-report
  `findings[]` in a multi-language project — MUST use the qualified form. Within
  a single tool's own document, the bare `ruleId` is sufficient (its
  `language` header supplies the namespace).
- **Prefix → category registry (SHOULD).** A rule id's leading token SHOULD come
  from this shared set, and MUST set `category` (§4) consistently with it,
  so the same prefix means the same thing everywhere:

  | Prefix | `category` | Meaning |
  |---|---|---|
  | `LINT` | `lint` | general correctness / lint |
  | `STYLE` | `style` | formatting / style |
  | `FUSA` | `safety` | the tool's own functional-safety rules |
  | `SEC`, `CWE-<n>` | `security` | security weakness |
  | `COV` | `coverage` | coverage / test gap |
  | `REQ` | `requirement` | requirement-traceability defect |
  | `CONC`, `RACE` | `concurrency` | concurrency / data race |
  | `SBOM`, `SLSA`, `VULN` | `supply-chain` | dependency / supply-chain |
  | `CFG` | `config` | configuration |
  | `MISRA-*`, `AUTOSAR-*`, `CERT-*` | per the rule's nature (usually `safety`) | standard-defined rule (keep the standard's own numbering verbatim) |

  Standard-defined rules keep their official id (`MISRA-15.5`, not a re-coding)
  and still set `category` + the relevant `standard`/`clause` fields (§4).

#### 1.5.2 Requirement ids (`id`)

A requirement `id` SHOULD match `^REQ-[A-Z0-9]+(-[A-Z0-9]+)*$`
(`REQ-FO-CORE001`, `REQ-LINT001`). It is unique within the project's
`.fusa-reqs.json` (§1.2.2). References to it in `trace` tags, gap-reports, and
markdown MUST use the id **verbatim** (no re-casing). Requirement ids are
project-scoped, so they are **not** language-qualified.

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
(§3), and the tool SHOULD still honour `--output` (write the partial document
rather than nothing).

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
`cert-cpp` · `unece-r155` · `unece-r156` · `slsa`.

A **command name maps to one or more standard ids** (the command name is not
itself an id): `do178` → `do178c`; `unece` → `unece-r155` and/or `unece-r156`;
`iec62443` → `iec62443-4-1` and/or `iec62443-4-2`; `misra` → `misra-c` (C
projects) and/or `misra-cpp` (C++), emitting `misra-c-gap-report.json` and/or
`misra-cpp-gap-report.json` respectively; `slsa` → `slsa`.

For a multi-part command, supporting **only one part is conformant** — the
emitted gap-report's `standard` field identifies which; a tool that supports both
SHOULD emit both gap-reports.

There is no `"ISO 26262"` form anywhere in the JSON. A clause reference is the
separate `clause` field (e.g. `"6.4.4"`). Consumers MUST treat an unrecognised id
verbatim (pass-through), never reject it.

### 2.5 JSON formatting

UTF-8, 2-space indented, RFC 3339 timestamps (`generatedAt`). Field names are
`lowerCamelCase` (`ruleId`, not `rule_id`). Every top-level JSON document **MUST**
carry the §3.1 common header (`schemaVersion` + `kind` + attribution) — **except**
the `version --format json` response, which is a command-status reply carrying
`specVersion` (§9.1).

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

### 2.9 Identifiers are format-invariant (MUST)

The **same identifier** appears byte-identical in every output format a command
supports — `json`, `sarif`, `text`, `html`, `md`. A `ruleId` is the same string
in the text line, the JSON `ruleId`, and the SARIF `result.ruleId`; a requirement
`id` is the same in the trace table, JSON, and markdown. Formats differ only in
*presentation*, never in the value of an id, `severity`, `category`, `standard`,
or `kind`.

**SARIF mapping (MUST, when `--format sarif`).** `tool.driver.name` = the `tool`
name (§1.1); each `result.ruleId` = the finding's `ruleId`; `tool.driver.rules[]`
declares each rule once with `id` = `ruleId`; `result.level` maps `ERROR`→`error`,
`WARNING`→`warning`, `INFO`→`note`; `physicalLocation.artifactLocation.uri` = the
project-relative `location.file` (§4). A rule's `category`/`standard`/`clause`
SHOULD ride in `result.properties`.

---

## 3. Common header & envelope

Every JSON document a tool emits is **self-identifying** and carries a uniform
header, so FuSaOps (or anything) can read attribution and route decoding off
*any* artefact without knowing which command produced it.

### 3.1 Common header (every document MUST carry)

```jsonc
{
  "schemaVersion": "1.9",        // MUST. spec version the document conforms to (MAJOR.MINOR)
  "kind":          "check-report", // MUST. document-type discriminator — see below
  "tool":          "go-FuSa",    // MUST. human-readable tool name
  "toolVersion":   "0.24.0",     // MUST. tool semver
  "language":      "go",         // MUST. go | c | cpp | …
  "generatedAt":   "2026-06-10T13:54:40Z"  // MUST. RFC 3339
}
```

**`kind` (MUST — closed, extensible enum).** Identifies the document type so a
consumer routes generically: `check-report` (also `report`) · `trace-matrix` ·
`qualification` · `sbom` · `provenance` · `artifact-manifest` · `audit-manifest`
· `gap-report` · `capabilities`. A consumer MUST treat an unknown `kind` as
opaque (read the common header, skip the payload) — never reject it.

The common header applies to **every** document, including the file-format
artefacts (`sbom.json`, `provenance.json`, `artifact-manifest.json`, audit-pack
`manifest.json`). (The `version --format json` response (§9.1) is the one
exception — a command-status reply, not a document; it carries `specVersion`
instead and is described there.)

### 3.2 Report extension (report documents add these)

The `--format json` outputs of `check` (§4), `trace` (§5), `qualify` (§6),
`report` (§9.1), and gap-reports (§9.3) are **report documents**: they carry the
§3.1 header **plus**:

```jsonc
{
  "projectRoot": "/abs/path",    // MUST. the --dir value verbatim (see note)
  "project":     "my-project",   // SHOULD
  "standard":    "iso26262",     // SHOULD — canonical id (§2.4.1); FuSaOps routes/groups on this
  "asil":        "ASIL-C",       // MAY — exactly one of asil|sil|dal, same rule as §1.2.1 (omit the others)
  "error":       { "code": "internal", "message": "…" }  // present ONLY on a runtime error (exit 3); omit otherwise
}
```

**`error` (structured; MUST when runtime error, else absent).** When a runtime
error occurred but a partial document was still emitted (paired with exit `3`),
`error` MUST be an object `{ "code", "message" }` where `code` is one of
`no-config` · `invalid-config` · `unsupported` · `internal` (consumers map an
unknown code to `internal`). Otherwise `error` MUST be **omitted** (do not emit
`"error": null`). The structured form lets FuSaOps react to error *categories*
generically instead of parsing free text.

**`projectRoot` across boundaries.** It is informational. The same source tree
has different absolute paths on host vs. in a container (`/Users/x/p` vs.
`/project`), so FuSaOps MUST NOT use `projectRoot` to correlate findings across
components — cross-component identity is the `fingerprint` (§4.2). A tool SHOULD
emit the `--dir` value verbatim (resolved to absolute).

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
  "category":    "lint",                 // MUST. closed enum — see below
  "standard":    "iso26262",             // SHOULD. canonical standard id (§2.4.1), NOT a display string
  "clause":      "6.4.4",                // SHOULD. clause within that standard
  "remediation": "split into smaller functions",  // MUST. free text, one actionable sentence. (NOT "fix")
  "disposition": "accepted",             // omit when open; accepted|deferred|rejected|open — see §4.1
  "fingerprint": "sha256:a1b2…"          // MUST. canonical hash — see §4.2
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

**`fingerprint` is `MUST` from spec v1.9.** Every conformant tool MUST emit it.
This unblocks cross-tool `diff` (§13) and enables stable finding suppression via
dispositions (§4.1) without relying on mutable line-number matching.

**`Finding` is the canonical finding atom.** Any command that emits a list of
findings (`vuln`, `cyber`, and `diff`'s `added`/`removed`, §13) SHOULD reuse this
exact shape, so FuSaOps has **one** finding decoder for every finding-bearing
document rather than one per command.

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
   order does not change the hash). A tool MUST ensure `results[].name` is
   **unique** within the document, so the sort is total and the hash stable.
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
document; `--spdx-version` **defaults to `2.3`**. `sbom.json` carries the §3.1
common header (`kind: "sbom"`) plus its payload — it does not add the §3.2 report
fields:

```jsonc
{
  "schemaVersion": "1.8", "kind": "sbom",           // §3.1 common header (+ tool/toolVersion/language/generatedAt)
  "tool": "go-FuSa", "toolVersion": "0.23.0", "language": "go", "generatedAt": "…",
  "format":      "x-FuSa SBOM v1",
  "module":      "github.com/SoundMatt/go-FuSa",   // MUST. identity — see below
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
`audit-pack` bundles the other artefacts, it MUST run **last**. `--full` does
**not** run `slsa` (the SLSA attestation is a separate supply-chain step); but if
`provenance.intoto.jsonl` already exists from a prior `slsa` run, `audit-pack`
includes it as a §1.3 file.

**`provenance.json` / `artifact-manifest.json`.** Carry the §3.1 common header
(`kind: "provenance"` / `"artifact-manifest"`); **not consumed by FuSaOps in v1**
(they ride along inside the audit-pack as opaque evidence). Minimal bodies (common
header abbreviated):

```jsonc
// provenance.json
{ "schemaVersion": "1.8", "kind": "provenance", "tool": "go-FuSa",
  "toolVersion": "0.23.0", "language": "go", "generatedAt": "…",
  "format": "x-FuSa provenance v1", "module": "…", "builder": "github-actions",
  "vcsRevision": "f8127ea", "vcsModified": false, "os": "linux", "arch": "amd64" }

// artifact-manifest.json
{ "schemaVersion": "1.8", "kind": "artifact-manifest", "tool": "go-FuSa",
  "toolVersion": "0.23.0", "language": "go", "generatedAt": "…",
  "format": "x-FuSa manifest v1",
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
  "schemaVersion": "1.8", "kind": "audit-manifest",   // §3.1 common header
  "tool": "go-FuSa", "toolVersion": "0.23.0", "language": "go", "generatedAt": "…",
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
`report` · `capabilities` (all MUST).

**`version` (MUST).** Prints to stdout a single line matching the regex
`^(\S+) (\d+\.\d+\.\d+[0-9A-Za-z.+-]*)$` — tool token, one space, semver
(e.g. `go-FuSa 0.24.0`). FuSaOps extracts the version as the second capture
group of the first stdout line. A tool SHOULD also support `version --format
json` (exactly three fields, no envelope):

```json
{ "tool": "go-FuSa", "version": "0.24.0", "specVersion": "1.9" }
```

`specVersion` is the spec the tool implements — distinct from a document's
`schemaVersion` (§2.8). `version --format text` is the same as the default line.

**`capabilities` (MUST — generic discovery).** `capabilities --format json`
emits a §3.1-header document (`kind: "capabilities"`) declaring what the tool
supports, so FuSaOps can orchestrate it **without trial-and-error or per-tool
branching**:

```jsonc
{
  "schemaVersion": "1.9", "kind": "capabilities", "tool": "go-FuSa",
  "toolVersion": "0.24.0", "language": "go", "generatedAt": "…",
  "specVersion": "1.9",                          // spec implemented
  "commands":  ["check","trace","qualify","release","audit-pack","report","fmea"],
  "formats":   { "check": ["text","json","sarif"], "trace": ["text","json","html"] },
  "standards": ["iso26262"]                       // canonical ids (§2.4.1) it can gap-report
}
```

FuSaOps calls `capabilities` first; when a non-conformant tool does not support
it, FuSaOps falls back to probing by running commands and handling results. This
is the keystone that keeps the FuSaOps↔tool exchange generic as commands and
tools grow.

**`init` (MUST).** Creates `.fusa.json` (§1.2.1, with `project.name`, `standard`,
and the integrity field populated) and `.fusa-reqs.json` containing
`{ "requirements": [] }`. It operates **per file**: it creates each target that
is missing and leaves an existing one untouched (a one-line stderr note), rather
than aborting the whole command — so a repo with `.fusa.json` but no
`.fusa-reqs.json` gets the missing file created. `--force` **overwrites
completely** (it does not merge) any target. A tool SHOULD source the config
values from flags (`--name`, `--standard`, one of `--asil`/`--sil`/`--dal`, and
`--project-version`) and/or interactive prompts. The **required** values are
`project.name` and `standard` (the integrity field is SHOULD); if a required
value is missing **and stdin is not a TTY** (CI), `init` MUST exit `2` rather
than prompt or write a placeholder config. It MAY scaffold additional structure
(`.github/`, hooks) and MAY offer `--migrate` (§1.2).

**`report` (MUST).** `report [--format text|json|html|sarif|md] [--output <file>]`
**re-runs analysis** on the project root — it does not read a cached
`check-report.json` and has no `--input` flag. It is effectively `check` that
always exits `0` regardless of findings (only `2`/`3` apply); its `--format json`
shape is identical to `check` (§4). It produces one report for one run and does
not aggregate across runs. Because `report` never gate-fails, a tool SHOULD treat
`--strict` on `report` as a usage error (exit `2`).

### 9.2 Recommended (safety evidence — SHOULD)

`verify` · `hara` · `tara` · `fmea` · `safety-case` · `coupling` · `cyber` ·
`vuln` · `boundary` · `coverage` · `diff`. Their JSON is **tool-defined** — see §13.
**`comp` is consumed by FuSaOps v1.70.0+** (see below and §10).
A command in this group MAY support `--format json`; if it does it
**SHOULD carry the §3 envelope** so that future consumption — added per §12/§13 —
does not force a breaking change to add the envelope later.

**`comp` (SHOULD).** `comp [--dir <path>] [--threshold <N>] [--dal DAL-A|B|C|D] [--format text|json] [--output <file>]`

Computes McCabe cyclomatic complexity V(G) per function (DO-178C §6.3.4). Exits
`1` when any function exceeds the threshold; exits `0` otherwise. DAL-level
thresholds: A ≤ 4, B ≤ 10 (default), C ≤ 15, D ≤ 20. `--dal` overrides
`--threshold`. `--format json` writes `comp-report.json`:

```jsonc
{
  "...header": "...",          // §3.1 common header, kind: "comp-report"
  "threshold": 10,
  "dal": "DAL-B",              // MAY — omit when threshold was set directly
  "totalFunctions": 42,
  "violations": 3,
  "results": [
    { "file": "src/main.c", "line": 12, "name": "process",
      "complexity": 15, "exceedsThreshold": true }
  ]
}
```

All six tools implement `comp`. FuSaOps rolls up comp-reports into a cross-language aggregate via `fusaops comp` (v1.70.0+). See §10.

### 9.3 Optional (standards & workflow — MAY)

`iso26262` · `iec61508` · `do178` · `iso21434` · `iec62443` · `misra` · `unece` ·
`slsa` · `sas` · `sci` · `badge` · `disposition` · `pr` · `hooks` · `sign` ·
`template` · `req` · `impact` · `metrics` · `fix` · `analyze` · `lint`.

Command → standard id mapping is in §2.4.1. Two of these commands have no
gap-report shape and are clarified here:

- **`slsa`** IS a gap-report command — `standard: "slsa"`, `kind: "gap-report"`,
  `--level L1|L2|L3|L4` (default L2). Assesses SLSA v1.0 supply-chain compliance
  across 10 objectives (provenance, SBOM, CODEOWNERS, SHA256SUMS, audit-pack).
  Written to `slsa-gap-report.json`. Use the §9.3 canonical gap-report schema.
  (Pre-v1.10 spec described `slsa` as writing `provenance.intoto.jsonl`; no tool
  implements that — the in-toto attestation path is deferred to a future `provenance`
  command. `slsa` is and was a gap-report.)
- **`disposition`** manages `.fusa-dispositions.json` (§1.2.3) — it adds, lists,
  and shows waiver decisions (and MAY support remove/update for entries that
  change over time); it does not itself gate.

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
| capability discovery | `capabilities --format json` | §9.1 (when present) |
| `fusaops comp` | `comp --format json` | §9.2 / §13 `{threshold,totalFunctions,violations,results[]}` |
| project metadata | `.fusa.json` | §1.2.1 |

**Generic routing.** Every document carries the §3.1 header, so FuSaOps reads the
same attribution off any artefact and dispatches on `kind` — one header decoder
plus a `kind`→payload-decoder map, rather than command-specific parsing. FuSaOps'
Go types in `report/`, `trace/`, `sbom/`, `auditpack/` are the authoritative
payload decoders; keep this spec and those structs in lock-step.

---

## 11. Current conformance & change-set

Snapshot 2026-07-25 (go-FuSa v0.31.0 · cpp-FuSa v0.12.6 · c-FuSa v0.5.34 · rust-FuSa v0.2.9 · py-FuSa v0.1.9 · java-FuSa v0.3.1). **All tools fully conformant.** ✅ conforms · ⚠️ gap (MUST) · ▫️ nice-to-have (SHOULD/MAY).

| Item | go-FuSa | c-FuSa | cpp-FuSa | rust-FuSa | py-FuSa | java-FuSa |
|---|---|---|---|---|---|---|
| severity enum `ERROR/WARNING/INFO` | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ (v0.2.0) |
| tag kinds `impl/test/sec-test` | ✅ | ✅ | ✅ (v0.12.2) | ✅ | ✅ | ✅ (v0.2.0) |
| `.fusa-reqs.json` (un-prefixed) | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| `.fusa.json` schema (§1.2.1) | ▫️ subset | ✅ | ✅ | ✅ | ▫️ subset | ▫️ subset |
| check finding `ruleId` (camel) | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ (v0.2.0) |
| check finding **nested `location`** | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ (v0.2.0) |
| check finding `remediation` (not `fix`) | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ (v0.2.0) |
| trace **`requirements/tags/coverage`** schema | ✅ | ✅ | ✅ (v0.12.4) | ✅ | ✅ | ✅ (v0.2.0) |
| trace `--format json` | ✅ | ✅ | ✅ (v0.12.1+) | ✅ | ✅ | ✅ (v0.2.0) |
| qualify `--output` + `total/passed/failed` | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| `sbom.json` name + `{module,components}` | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| `components[].hash` = `sha256:hex` (§2.7) | ✅ | ✅ | ✅ | ✅ | ✅ (v0.1.3) | ✅ (v0.2.0) |
| audit-pack = single **ZIP** + `manifest.json` | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| evidence filenames lowercase-kebab | ✅ | ✅ | ✅ | ✅ | ✅ (v0.1.3) | ✅ (v0.2.0) |
| exit `2` for usage errors | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ (v0.2.0) |
| exit `3` for runtime errors | ✅ | ✅ | ✅ | ✅ | ✅ (v0.1.3) | ✅ (v0.2.0) |
| `--no-color`/`NO_COLOR` | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ (v0.3.0, issue #6 fixed) |
| `--output` ⇒ no stdout copy (§2.2) | ✅ (v0.29.0) | ✅ (v0.5.7) | ✅ (v0.12.1+) | ✅ (v0.2.7) | ✅ (v0.1.3) | ✅ (v0.2.0) |
| `location.file`/`tags[].file` project-relative | ✅ (v0.30.0) | ✅ | ✅ (v0.12.5) | ✅ | ✅ | ✅ (v0.2.0) |
| envelope `tool/toolVersion/language` on check + gap | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| `kind` + common header on check + gap docs (§3.1) | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| `kind` + common header on trace/qualify/sbom/pack (§3.1) | ✅ | ✅ | ✅ (v0.12.4) | ✅ | ✅ | ✅ |
| gap-report `kind` = `"gap-report"` (§3.1) | ✅ | ✅ (v0.5.7) | ✅ | ✅ (v0.2.6) | ✅ (v0.1.5, issue #1 fixed) | ✅ (v0.2.0) |
| structured `error {code,message}` on check (§3.2) | ✅ | ✅ | ✅ | ✅ | ✅ (v0.1.3) | ✅ (v0.2.0) |
| check report `projectRoot` (MUST, §3.2) | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ (v0.3.0, issue #5 fixed) |
| `capabilities` command (MUST, §9.1) | ✅ | ✅ (v0.5.10) | ✅ (v0.12.3) | ✅ | ✅ | ✅ (v0.3.0, issue #3 fixed) |
| `schemaVersion` on check + gap docs | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| `schemaVersion` on trace/qualify/sbom/pack | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| finding `category` | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ (v0.2.0) |
| finding `standard`+`clause` | ✅ | ✅ | ✅ | ✅ | ✅ (v0.1.3) | ✅ (v0.2.0) |
| finding `fingerprint` (MUST, §4.2) | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ (v0.2.0) |
| location `endLine/endColumn` (MAY, §4) | ✅ (v0.30.0) | ✅ (v0.5.9) | ✅ (v0.12.2) | ✅ (v0.2.5) | ✅ (v0.1.4) | ✅ (v0.2.0) |
| `ruleId` regex + qualified `lang/ruleId` (§1.5) | ▫️ verify | ✅ (pattern `^[A-Z][A-Z0-9]+$` confirmed) | ▫️ verify | ▫️ verify | ▫️ verify | ▫️ verify |
| ids format-invariant across formats (§2.9) | ✅ (v0.29.0) | ✅ (v0.5.6) | ✅ (v0.12.4) | ✅ (v0.2.4) | ✅ (v0.1.3) | ✅ (v0.2.0) |
| image: **alpine/musl base** + `/usr/local/bin/<bin>` (§15) | ✅ | ✅ | ✅ | ✅ | ▫️ verify | ▫️ verify (JVM base) |
| image: OCI + `io.x-fusa.*` labels (§15) | ✅ | ✅ | ✅ (v0.12.3) | ✅ | ✅ | ✅ (v0.2.0) |
| standards `iso21434`+`unece`+`iec62443`+`slsa` subcommands (§9.3) | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| standards canonical §9.3 shape (`satisfied`/`gaps`) | ✅ | ✅ (v0.5.7) | ✅ (v0.12.3) | ✅ (v0.2.6) | ✅ | ✅ (v0.2.0) |
| `slsa` standard id = `"slsa"`, kind = `"gap-report"` (§2.4.1) | ✅ | ✅ | ✅ (v0.12.3) | ✅ | ✅ | ✅ (v0.2.0) |
| `comp` command (§9.2 SHOULD) | ✅ (engine rule) | ✅ | ✅ | ✅ | ✅ | ✅ |
| `fusaops comp` roll-up (§9.2 / §10) | — | — | — | — | — | — |
| trace `secTestedRequirements` (MUST, §5) | ✅ | ✅ | ✅ | ✅ (v0.2.6) | ✅ | ✅ (v0.2.0) |
| `fusaops trace --gaps/--req-coverage/--sec-tested` (§5) | — | — | — | — | — | — |

The `req`/`impact`/`metrics`/`lint`/`fix` commands are §9.3 optional and
**not consumed by FuSaOps v1** — intentionally absent from the audited rows above.

> **Reference split.** go-FuSa is the **schema** reference and the **exit-code** reference.
> A conformant tool MUST match both.

**All tools are fully conformant as of 2026-06-18.** No open MUST gaps remain.

Previously tracked MUST gaps now closed:
- **py-FuSa v0.1.5**: gap-report `kind` fixed from `"<std>-gap-report"` → `"gap-report"` (issue #1 ✅)
- **java-FuSa v0.3.0**: `--no-color` flag added (issue #6 ✅); `projectRoot` added to JSON reports (issue #5 ✅); `capabilities.standards[]` now emits `"iec62443"` consistently (issue #3 ✅)
- **rust-FuSa v0.2.7**: `--format text --output <file>` no longer copies text to stdout (§2.2 ✅)

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
| `diff` | tool-defined; fingerprint is **MUST** from v1.9 — cross-tool diff is now enabled for conformant tools | `{ added:[fingerprint], removed:[fingerprint], unchanged:N }`; baseline is a prior `check --format json`, given via `--baseline <file>`. **Exit `1`** when `added[]` contains any open ERROR (or any severity under `--strict`), else `0` |
| `hara` → `.fusa-hara.json` | **input** file; the `hara` command validates/normalises it (and scaffolds a template if absent), output tool-defined | `{ hazards:[{id, hazard, severity, exposure, controllability, asil, safetyGoal}] }` |
| `sas` → `sas.json`/`sas.md` | tool-defined; **conflict**: go md-only vs cpp `sas.json`+`md` | `sas.json` (envelope + tool-defined body) plus `sas.md` |
| `sci` → `sci.json` | tool-defined; **conflict**: go stdout-only vs cpp `sci.json` | `sci.json` (envelope + tool-defined body) |
| `boundary` → `.dot`/`.mermaid` | tool-defined graph text | no JSON contract in v1 |
| `verify` → `.fusa-evidence.json` | tool-defined | `{ passed, failed, suites:[ {name, passed, failed, tests:[{name, result}]} ] }` (`result` per §6) |
| `comp` → `comp-report.json` | **canonical — all six tools** (§9.2 SHOULD); **consumed by FuSaOps v1.70.0+** | `{ …header(kind:"comp-report"), threshold:N, dal?:"DAL-B", totalFunctions:N, violations:N, results:[{file,line,name,complexity,exceedsThreshold}] }` |

---

## 14. Changelog

### 1.10.10 — 2026-07-25 (comp consumed by FuSaOps v1.70.0+; all-tool spec-version sync)

- **Spec header corrected:** header bumped from `1.10.4` to `1.10.10`, retroactively incorporating the §11/§14 entries written in 1.10.5–1.10.9 that were each additive updates without contract changes.
- **`comp` consumed by FuSaOps v1.70.0+:** §9.2 heading updated to remove `comp` from the "not consumed" list; intro note (§0) updated accordingly; §13 row already noted v1.70.0+ consumption. This is the MINOR bump per §12 for FuSaOps beginning to consume a §9.2 command.
- **Version snapshot updated (§11):** go-FuSa v0.31.0 · cpp-FuSa v0.12.6 · c-FuSa v0.5.34 · rust-FuSa v0.2.9 · py-FuSa v0.1.9.
- **go-FuSa v0.31.0:** `SpecVersion` constant fixed to `"1.10.4"` ✅; `release` command auto-detects CI builder from `GITHUB_ACTIONS`/`CI`/`GITHUB_WORKFLOW_REF` env vars; `--builder` flag added for explicit override. No new MUST gaps.
- **c-FuSa v0.5.34:** `CFUSA_SCHEMA_VERSION` and `CFUSA_SPEC_VERSION` constants corrected to `"1.10.4"` ✅; `docker-publish.yml` CI added.
- **cpp-FuSa v0.12.6:** `SpecVersion` constant corrected to `"1.10.4"` ✅; MSVC C2338 compile error in test wrapped.
- **rust-FuSa v0.2.9:** `SPEC_VERSION` constant corrected to `"1.10.4"` ✅; `docker-publish.yml` CI added.
- **py-FuSa v0.1.9:** `SPEC_VERSION` corrected to `"1.10.4"` ✅; `docker-publish.yml` CI added (first tagged image release).
- No new MUST gaps; no §11 table row changes.

### 1.10.9 — 2026-06-13 (rust-FuSa v0.2.8 gap-report --format md; 100% coverage)

- **Version snapshot updated (§11):** rust-FuSa v0.2.8.
- **rust-FuSa v0.2.8:** all 8 standards gap-report commands (`iso26262`, `iec61508`, `do178c`, `iso21434`, `unece`, `misra`, `iec62443`, `slsa`) now support `--format md` output. 10 new requirements added to `.fusa-reqs.json`; 100% traced and tested (174 requirements). DCO enforcement added to CI. No new or closed MUST conformance gaps; no §11 table changes.
- **rust-FuSa open SHOULD gap:** `--format text --output <file>` still echoes text to stdout while writing to file (§2.2 nuance). JSON-only `--output` already conformant.

### 1.10.8 — 2026-06-13 (c-FuSa v0.5.16 exit-code fixes)

- **Version snapshot updated (§11):** c-FuSa v0.5.16.
- **c-FuSa v0.5.16:** `slsa`/`iec62443` now return exit 3 on runtime errors; `comp` now returns exit 2 for invalid `--format`. Fixes edge cases in exit-code conformance (§2.1). No §11 table changes (cells already ✅). No FuSaOps adapter changes.

### 1.10.7 — 2026-06-13 (cpp-FuSa v0.12.5 location.file fix; c-FuSa v0.5.14 hara show)

- **Version snapshot updated (§11):** cpp-FuSa v0.12.5 · c-FuSa v0.5.14.
- **cpp-FuSa v0.12.5:** all `analyze` own-pass findings (ANAL003–012) now emit project-relative `location.file` instead of absolute OS paths (§4 MUST). `location.file` project-relative: ▫️ partial → ✅ (v0.12.5). No FuSaOps adapter changes needed.
- **c-FuSa v0.5.14:** adds `hara show --format json|markdown` — a c-FuSa-specific hazard analysis command, not a spec requirement. No §11 changes; no FuSaOps adapter changes.
- No new MUST gaps; no issues closed or opened.

### 1.10.6 — 2026-06-13 (c-FuSa v0.5.10 closes both MUST bugs; §11 table accuracy)

- **Version snapshot updated (§11):** c-FuSa v0.5.10.
- **c-FuSa v0.5.9:** `location.endLine`/`endColumn` added to all check findings (§4 MAY) ✅ — closes c-FuSa issue #16.
- **c-FuSa v0.5.10:** `capabilities` now lists `"slsa"` in `commands[]` and `standards[]` (§9.1 MUST) ✅ — closes c-FuSa issue #18. c-FuSa is now fully conformant.
- **§11 table accuracy (java-FuSa v0.2.0, confirmed from source):**
  - `--output` no stdout copy: ▫️ verify → ✅
  - `structured error {code,message}`: ▫️ verify → ✅
  - `finding standard+clause`: ▫️ verify → ✅
  - `image OCI + io.x-fusa.* labels`: ▫️ verify → ✅
  - ⚠️ New MUST gap: `--no-color` CLI flag missing; only `NO_COLOR` env var honoured (§2.6). Filed java-FuSa issue #6.
  - ⚠️ New MUST gap: `projectRoot` absent from check/gap/trace/qualify JSON (§3.2). Filed java-FuSa issue #5. Added §11 row.
- **§11 table accuracy (cpp-FuSa v0.12.4, confirmed from source):**
  - `ids format-invariant`: ▫️ verify → ✅
- **Net change-set:** removed c-FuSa entries (both fixed); added java-FuSa issue #5 and #6.

### 1.10.5 — 2026-06-12 (cpp-FuSa v0.12.4 trace fix; c-FuSa v0.5.8; java-FuSa v0.2.0 conformance)

- **Version snapshot updated (§11):** cpp-FuSa v0.12.4 · c-FuSa v0.5.8 · java-FuSa v0.2.0.
- **cpp-FuSa v0.12.4 — issues #11+#12 fixed:**
  - `trace --format json` now emits `kind: "trace-matrix"` ✅ (was `"trace-report"`).
  - `requirements[].standard` key used ✅ (was `"standardRef"`).
  - §11 row `kind + common header on trace/…`: ⚠️ → ✅. Row `trace requirements/tags/coverage schema`: ⚠️ → ✅.
  - FuSaOps `adapter/cpfusa_test.go` updated to v0.12.4 format; `Standard` field now checked.
- **c-FuSa v0.5.8:** feature release — `coverage --dal` + metrics auto-collect. No new MUST gaps.
  - Filed c-FuSa issue #18: `capabilities` omits `"slsa"` from `commands[]` + `standards[]` despite `cmd_slsa.c` present (§9.1 MUST).
- **java-FuSa v0.2.0 — major spec conformance sprint:**
  - Confirmed ✅ (from `Spec11ConformanceTest.java` + source): severity uppercase, ruleId camelCase, nested location, remediation, fingerprint sha256, category, gap-report kind="gap-report" all 7 standards, slsa standard="slsa", secTestedRequirements, trace kind="trace-matrix", endLine/endColumn, exit codes, SBOM shape, evidence filenames, ids format-invariant, location.file project-relative.
  - ⚠️ New MUST gap: `capabilities.standards[]` uses `"iec62443-4-1"` but `iec62443` gap-report emits `"standard":"iec62443"` — inconsistent with all other tools; filed java-FuSa issue #3.
  - Updated ▫️ verify rows to ✅ in §11 table.

### 1.10.4 — 2026-06-12 (multi-tool conformance sprint — go-FuSa v0.30.0, c-FuSa v0.5.7, cpp-FuSa v0.12.3, rust-FuSa v0.2.6, py-FuSa v0.1.4)

- **Version snapshot updated (§11):** go-FuSa v0.30.0 · cpp-FuSa v0.12.3 · c-FuSa v0.5.7 · rust-FuSa v0.2.6 · py-FuSa v0.1.4.
- **go-FuSa v0.30.0:** `location.file` now project-relative ✅; capabilities `"slsa"` canonical ✅; `endLine`/`endColumn` added ✅.
- **c-FuSa v0.5.7:** gap-report `kind: "gap-report"`, canonical `standard`, `status: "satisfied"/"gap"`, `findings[]` array ✅; audit-pack success message to stderr ✅.
- **cpp-FuSa v0.12.3:** iec62443/slsa/iso26262 objectives `"objectives"` key with `"title"` field ✅; capabilities `"slsa"` ✅; Dockerfile spec-version label `"1.10"` ✅. **Deferred:** trace `kind: "trace-report"` and `requirements[].standardRef` (issues #11/#12 closed by maintainer, not yet fixed in v0.12.3).
- **rust-FuSa v0.2.5/v0.2.6:** `endLine`/`endColumn` populated (v0.2.5) ✅; gap-report canonical shape `"objectives"`/`"satisfied"` (v0.2.6) ✅; sec-tested gate uses `sec_tested_requirements` ✅.
- **py-FuSa v0.1.4:** `endLine`/`endColumn` added to all AST findings ✅; evidence filenames confirmed lowercase-kebab ✅; `ruleId`/ids format-invariant confirmed ✅.
- **c-FuSa feat: filed issue #16** for `endLine`/`endColumn` in findings (§4 MAY — confirmed absent in v0.5.7 source).
- **§11 table updates:** c-FuSa `--output` confirmed ✅ (all commands, not just qualify); `ids format-invariant` confirmed ✅ for c-FuSa and py-FuSa; `evidence filenames` confirmed ✅ for py-FuSa; `--output` for rust-FuSa JSON ✅ (text+--output nuance noted).

### 1.10.3 — 2026-06-12 (go-FuSa v0.29.1 · cpp-FuSa v0.12.2 · c-FuSa v0.5.6 conformance updates)

- **Version snapshot updated (§11):** go-FuSa v0.29.1 (housekeeping) · cpp-FuSa v0.12.2 · c-FuSa v0.5.6.
- **cpp-FuSa v0.12.1/v0.12.2 fixes (§11 cells updated):**
  - `trace --format json` now writes to stdout by default (§2.2): ▫️ WIP → ✅.
  - `trace tags[]` now top-level flat array per §5: ⚠️ → ✅ (v0.12.2).
  - Tag `kind` values now `impl`/`test`/`sec-test` per §5: ⚠️ → ✅ (v0.12.2).
  - `coverage` block now uses canonical field names (`totalRequirements` etc.): ✅.
  - `secTestedRequirements` added to coverage: ✅ (v0.12.1+).
  - `location.endLine`/`endColumn` emitted when available (§4 MAY): ▫️ → ✅ (v0.12.2).
  - **Still open:** trace `kind: "trace-report"` → `"trace-matrix"` (§3.1 MUST); filed cpp-FuSa issue #11.
  - **New:** `requirements[].standardRef` should be `requirements[].standard` (§5 SHOULD); filed cpp-FuSa issue #12.
    FuSaOps test updated — `Standard` field remains empty for cpp-FuSa until fixed.
- **c-FuSa v0.5.6 fix:** `init --name` alias for `--project` added (§9.1); dirname default when no name given.
- **FuSaOps `adapter/cpfusa.go` updated:** `cppFuSaAdapter.Trace` override and `parseCppFuSaTrace`
  removed — cpp-FuSa v0.12.2 uses canonical stdout JSON; the generic `cmdAdapter.Trace` path is used.
  REQ-FO-ADP024 retired. cpp-FuSa trace tests updated to v0.12.2 format.

### 1.10.2 — 2026-06-12 (version snapshot updated; §11 corrected for cpp-FuSa trace format and py-FuSa v0.1.3 fixes)

- **Version snapshot updated (§11):** go-FuSa v0.29.0 · cpp-FuSa v0.12.0 · c-FuSa v0.5.5 · rust-FuSa v0.2.4 · py-FuSa v0.1.3 · java-FuSa v0.1.0.
- **§11 conformance corrections:**
  - `--output` no stdout copy (§2.2): go-FuSa ▫️ → ✅ (v0.29.0 §2.2+§2.9 test suite); c-FuSa and rust-FuSa updated to ▫️ partial (qualify only); cpp-FuSa ▫️ v0.12.1 WIP.
  - `ids format-invariant (§2.9)`: go-FuSa ▫️ → ✅ (v0.29.0); rust-FuSa ▫️ → ✅ (v0.2.4 §2.9 suite).
  - `components[].hash sha256:hex`: py-FuSa ▫️ → ✅ (v0.1.3).
  - `exit 3 for runtime errors`: py-FuSa ▫️ → ✅ (v0.1.3).
  - `structured error {code,message}`: py-FuSa ▫️ add → ✅ (v0.1.3 §3.2 JSON error envelope).
  - `finding standard+clause`: py-FuSa ▫️ add → ✅ (v0.1.3 engine injection).
  - `trace requirements/tags/coverage schema`: cpp-FuSa footnote corrected — v0.10.0 moved from `implementedBy/testedBy` to per-requirement nested `tags[]`, not top-level flat; the original bug in issue #3 (flat top-level tags[]) is still open. FuSaOps normalises during ingestion.
- **FuSaOps `adapter/cpfusa.go` updated:** `parseCppFuSaTrace` updated to parse cpp-FuSa v0.10.0+ per-requirement nested `tags[]` format; maps tag kind `"req"` → `"impl"` pending cpp-FuSa issue #3 fix.
- **Issues filed on tool repos:** go-FuSa #25 (endLine/endColumn feat), rust-FuSa #6 (end_line always 0), cpp-FuSa #5 (endLine/endColumn feat), py-FuSa #2 (endLine/endColumn feat), py-FuSa #3 (structured error feat), java-FuSa #1 (§11 conformance tracking). Closed: c-FuSa #9, py-FuSa #4 (incorrectly filed — both tools already support `--no-color`).

### 1.10.1 — 2026-06-12 (java-FuSa added; §11 table corrected after code review)

- **java-FuSa v0.1.0 registered (§1.1):** `language: "java"`, binary `jfusa`,
  image `ghcr.io/soundmatt/java-fusa`. FuSaOps adapter `adapter/jfusa.go` added
  (REQ-FO-ADP028). 45 commands, spec v1.10 conformant. ▫️ verify rows to be
  confirmed in v0.2.0.
- **§11 table corrected:** Code review of working trees revealed several filed
  issues were based on C-string-escape grep errors and stale pre-v0.10.0/v0.1.2
  data. Corrected cells:
  - c-FuSa `schemaVersion`/`kind` on trace/qualify/sbom/pack: ▫️ → ✅ (confirmed in source)
  - rust-FuSa `schemaVersion`/`kind` on trace/qualify/sbom/pack: ▫️ → ✅ (confirmed via serde `rename_all = "camelCase"`)
  - py-FuSa `remediation`, `sbom.json`, audit-pack, `--no-color`, `schemaVersion`, `kind` on check+gap+trace+qualify+sbom+pack: ▫️ → ✅ (confirmed in v0.1.2)
  - cpp-FuSa `secTestedRequirements`, flat `tags[]`: ▫️ → ✅ (confirmed in v0.10.0)
  - cpp-FuSa tag kind `impl/test/sec-test`: ✅ → ⚠️ (emits `"req"` not `"impl"` for impl tags)
  - cpp-FuSa trace kind: ▫️ → ⚠️ (emits `"trace-report"` not `"trace-matrix"`)
  - go-FuSa `slsa` standard id: ⚠️ → ✅ (confirmed `"slsa"` in working tree; v0.27 is a §2.2 fix)
  - py-FuSa gap-report kind: new ⚠️ row (emits `"<std>-gap-report"` not `"gap-report"`)
- Closed/corrected GitHub issues: go-FuSa #22 (resolved), c-FuSa #7 (incorrect),
  rust-FuSa #4 (incorrect), cpp-FuSa #3 (corrected scope), py-FuSa #1 (corrected scope).

### 1.10.0 — 2026-06-12 (comp command; slsa clarification; go-FuSa §3.1 common-header resolved)

- **`comp` command (§9.2 SHOULD):** Added to spec. Implemented by all five tools;
  output `comp-report.json`, `kind: "comp-report"`. DAL-level thresholds A≤4/B≤10/C≤15/D≤20.
  Flags: `--threshold <N>`, `--dal DAL-A|B|C|D`, `--format text|json`, `--output <file>`.
  Added to §1.3 evidence list and §13 canonical shape.
- **`slsa` is a gap-report command (§9.3):** Corrects the v1.4.0 description that
  called `slsa` "not a gap-report command" writing `provenance.intoto.jsonl`. All five
  tools implement `slsa` as a §9.3 gap-report (standard `"slsa"`, `kind: "gap-report"`,
  `--level L1|L2|L3|L4`, output `slsa-gap-report.json`). `provenance.intoto.jsonl`
  (in-toto attestation) is a **deferred** planned feature; no tool produces it — removed
  from §1.3 evidence list. Spec was wrong; tools were correct.
- **`slsa` standard id canonical = `"slsa"` (§2.4.1):** Added `slsa` to the enum.
  Command→id mapping: `slsa` → `slsa`. go-FuSa v0.26 currently emits `"slsa-v1.0"` — ⚠️
  non-conformant; fix targeted for v0.27.
- **go-FuSa §3.1 open item resolved (§11):** go-FuSa v0.26 added `schemaVersion`/`kind`/
  common header to trace-matrix, qualify-report, sbom.json, and audit-pack manifest.
  Hash format moved from `h1:base64` to canonical `sha256:hex` (§2.7). §11 rows updated.
- **go-FuSa `slsa`+`iec62443` (§11):** go-FuSa v0.26 added both gap-report commands,
  closing the parity gap vs c-FuSa/rust-FuSa/py-FuSa. §11 standards row updated.
- **Version snapshot (§11):** go-FuSa v0.26, cpp-FuSa v0.11.0, c-FuSa v0.5.2,
  rust-FuSa v0.2.3, py-FuSa v0.1.2.
- **`schemaVersion` in spec header updated** from `1.9` to `1.10`.

### 1.9.0 — 2026-06-10 (MUST promotion: category, remediation, fingerprint, capabilities)

Promotes four SHOULD/MAY fields to MUST, targeting items go-FuSa and cpp-FuSa
already implement so no current conformant tool is blocked. c-FuSa gains new ⚠️
items. Also corrects §11 table accuracy for c-FuSa: prior ▫️ entries on items
already MUST in the spec body are now correctly shown as ⚠️.

- **`category` (§4, MUST):** promoted from SHOULD. Every finding MUST carry a
  category from the closed enum (`lint`/`style`/`safety`/…). Enables
  FuSaOps to filter/group by category without tool-specific branching.
- **`remediation` (§4, MUST):** promoted from SHOULD. One actionable free-text
  sentence. Safety tooling that reports a finding without telling the engineer
  how to address it is insufficient evidence.
- **`fingerprint` (§4.2, MUST):** promoted from SHOULD; removes the deferred
  "→MUST when diff lands" qualifier. Every conformant tool MUST emit it now.
  This unblocks cross-tool `diff` (§13) and makes disposition matching via
  `fingerprint` stable and machine-readable rather than fragile line-number-based.
- **`capabilities` (§9.1, MUST):** promoted from SHOULD. FuSaOps calls it first
  and falls back to probing for non-conformant tools. The fallback exists but
  conformance requires the command.
- **`standard`+`clause` (§4, SHOULD):** promoted from MAY. Not MUST because not
  every rule maps to a specific standards clause; SHOULD gives a strong signal
  without over-constraining generic tools.
- **§11 accuracy:** c-FuSa ▫️ → ⚠️ for exit `3`, `--no-color`, envelope/kind/
  schemaVersion on check+gap, structured error — these were already MUST in the
  spec body; the table was incorrectly lenient.
- **§13 `diff`:** updated status note; fingerprint is now MUST so diff is
  enabled for conformant tools.
- **`(new)` annotations cleared** from fields now well-established (go-FuSa v0.24
  implements them all): exit `3`, `--no-color`, envelope, kind, schemaVersion,
  structured error, capabilities, fingerprint.

### 1.8.0 — 2026-06-10 (naming commonisation + container standard + onboarding)

Commonises **all** naming — values, not just shapes — and standardises the
container images, so a new x-FuSa starts at a high baseline.

- **Tool/language/binary registry (§1.1):** one canonical row per tool
  (`language` id · binary · human name · image), with reserved rows for future
  languages.
- **Identifier naming (§1.5):** `ruleId` regex + the qualified cross-language
  form `"<language>/<ruleId>"`; a shared **prefix→category registry** so the same
  prefix means the same thing across tools; requirement-id convention.
- **Format-invariant identifiers (§2.9):** the same id/severity/category/standard
  appears byte-identical in `json`/`sarif`/`text`/`html`/`md`; normative SARIF
  field mapping.
- **Container images (§15):** alpine/musl base + static binary at
  `/usr/local/bin/<binary>`, `ghcr.io/soundmatt/<language>-fusa` naming/tags, OCI
  + `io.x-fusa.*` labels, the one-line `COPY --from` bundling shape.
- **Onboarding ramp (§16):** the MUST baseline (1–7) that makes a new tool
  orchestrable by FuSaOps on day one with zero FuSaOps changes.
- **§11:** rows for rule-id naming, format-invariance, and the image base/labels;
  c-FuSa change-set gains the ubuntu→alpine base move.

### 1.7.0 — 2026-06-10 (generic data-exchange uplift)

Makes the FuSaOps↔tool exchange uniform so a consumer reads any artefact the
same way and routes generically:

- **Self-identifying documents (§3.1):** new **`kind`** discriminator
  (`check-report`/`trace-matrix`/`sbom`/`gap-report`/…) on **every** document.
- **Universal common header (§3.1):** `{schemaVersion, kind, tool, toolVersion,
  language, generatedAt}` now MUST appear on **every** document, including the
  file-format artefacts (`sbom.json`/`provenance.json`/`manifest.json`) — tightens
  the v1.5 artefact exemption so FuSaOps reads one header off anything. Report
  documents add the §3.2 fields (`projectRoot`/`project`/`standard`/`asil`/`error`).
- **Structured `error` (§3.2):** `{code, message}` with a code enum
  (`no-config`/`invalid-config`/`unsupported`/`internal`) so FuSaOps reacts to
  error categories generically instead of parsing free text.
- **`capabilities` command (§9.1, was SHOULD → MUST in v1.9):** `capabilities --format json` →
  `{commands, formats, standards, specVersion}` — the discovery handshake that
  lets FuSaOps orchestrate without per-tool trial-and-error.
- **One finding decoder (§4/§13):** `vuln`/`cyber`/`diff` SHOULD reuse the §4
  `Finding` atom verbatim.
- **§10/§11:** documented generic `kind`-based routing; added conformance rows
  for `kind`, the universal header, structured `error`, and `capabilities`.

### 1.6.0 — 2026-06-10 (sixth review round: go-FuSa / c-FuSa / cpp-FuSa)

- Fixed a duplicated bullet in the v1.4.0 changelog entry.
- **`provenance.intoto.jsonl`** (from `slsa`) added to the §1.3 evidence list so
  `audit-pack` packs it; §7 states `--full` does **not** run `slsa` but packs the
  attestation if present.
- **qualify hash:** `results[].name` MUST be **unique** (total sort order, stable
  hash on duplicate names).
- **`diff` exit codes (§13):** `1` when `added[]` has an open ERROR (or any under
  `--strict`), else `0`.
- **exit 3 + `--output` (§2.3):** SHOULD still write the partial document.
- **duplicate-id rule (§1.2.2):** rebased onto "whenever a tool reads
  `.fusa-reqs.json`" instead of binding it to `check`.
- **§9.2 envelope (SHOULD):** a §9.2 `--format json` SHOULD carry the §3 envelope
  to avoid a breaking change when FuSaOps later consumes it.
- **init (§9.1):** named the required values (`project.name`, `standard`); added
  `--project-version`; `report --strict` ⇒ exit `2`; `version --format text` =
  default line.
- **`disposition` command:** MAY support remove/update (§9.3).
- **§11:** added shared-MUST rows for `--output` no-stdout and project-relative
  paths.

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

---

## 15. Container images (MUST for a published tool)

So FuSaOps can bundle every tool uniformly (`COPY --from`) and a new tool drops
in with one `FROM`+`COPY` pair, published images follow one shape.

- **Base.** The runtime image MUST be **musl/alpine-compatible** and the tool
  binary MUST be **statically linked or musl-linked**, so binaries from different
  tools co-reside in one image without a libc clash. Use `alpine:<pinned>` (or a
  shared `ghcr.io/soundmatt/x-fusa-base:<ver>` once published) as the runtime
  base. (This is why `c-FuSa` must move its base ubuntu→alpine.)
- **Binary location (MUST).** `/usr/local/bin/<binary>` (e.g.
  `/usr/local/bin/gofusa`), on `PATH`, executable.
- **Image name & tags (MUST).** `ghcr.io/soundmatt/<language>-fusa` with tags
  `:latest`, `:<major.minor.patch>`, and `:<major.minor>`.
- **Entrypoint (SHOULD).** `WORKDIR /project`, `ENTRYPOINT ["<binary>"]`,
  `CMD ["help"]`, `EXPOSE` only if it serves.
- **Platforms (SHOULD).** `linux/amd64` MUST; `linux/arm64` SHOULD.
- **Labels (MUST).** OCI labels plus the x-FuSa set, so an image is
  self-describing and FuSaOps can introspect a tool image without running it:

  ```dockerfile
  LABEL org.opencontainers.image.title="go-FuSa" \
        org.opencontainers.image.version="0.23.0" \
        org.opencontainers.image.source="https://github.com/SoundMatt/go-FuSa" \
        org.opencontainers.image.licenses="MPL-2.0" \
        io.x-fusa.tool="go-FuSa" \
        io.x-fusa.language="go" \
        io.x-fusa.binary="gofusa" \
        io.x-fusa.spec-version="1.8"
  ```

- **Refresh (SHOULD).** A tool release SHOULD fire `repository_dispatch`
  (`xfusa-released`) so the FuSaOps all-in-one image rebuilds without a FuSaOps
  release (see `tools-monitor.yml`).

FuSaOps then bundles any tool with exactly:

```dockerfile
FROM ghcr.io/soundmatt/<language>-fusa:latest AS <language>
COPY --from=<language> /usr/local/bin/<binary> /usr/local/bin/<binary>
```

---

## 16. Onboarding a new x-FuSa tool

The high-water mark for a new language `L`. A conforming tool is orchestrable by
FuSaOps with **zero FuSaOps code changes** once these are done:

1. **Register** the row in §1.1 (`language` id, binary, human name, image) — one
   PR against this spec.
2. **Required commands (§9.1)** emitting §3 documents: `version` (+ `--format
   json`), `capabilities`, `init`, `check`, `trace`, `qualify`, `release`,
   `audit-pack`, `report`.
3. **Read** the §1.2 input files (`.fusa.json`, `.fusa-reqs.json`,
   `.fusa-dispositions.json`).
4. **Naming:** common header + `kind` on every document (§3.1); rule ids and
   requirement ids per §1.5; severity/category/standard/tag-kind enums; the same
   id in every format (§2.9).
5. **Exit codes** `0/1/2/3` (§2.3); `--no-color` (§2.6); `--output` redirection
   (§2.2); project-relative paths (§4).
6. **Image** per §15 (alpine base, `/usr/local/bin/<binary>`, labels, tags).
7. **Self-check:** validate output against the published JSON Schemas, and (when
   available) run the conformance kit `fusaops conform <binary>` in CI.
8. **Evidence (SHOULD):** the §9.2/§9.3 commands, following the §13 canonical
   directions so the new tool doesn't recreate the existing divergences.

Steps 1–7 are the **MUST** baseline for "on board"; step 8 deepens coverage. A
tool that does 1–7 interoperates on day one.
