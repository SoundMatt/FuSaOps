# Getting started: applying FuSaOps to your project

A step-by-step path for adopting FuSaOps in a real repository — from a first
zero-config scan to a CI-gated, dashboarded safety-evidence pipeline. Each step
works standalone; stop wherever your project needs.

## Prerequisites

FuSaOps orchestrates per-language tools; it does not implement safety rules
itself. Pick one:

- **Docker (recommended for a first try — nothing to install locally).** The
  published image bundles all six x-FuSa tools. See [Step 0](#step-0-try-it-with-zero-setup-docker).
- **Native binaries.** Install `fusaops` (`go install
  github.com/SoundMatt/FuSaOps/cmd/fusaops@latest`) plus the x-FuSa tool for
  each language in your repo (`gofusa`, `cfusa`, `cpfusa`, `rsfusa`, `pyfusa`,
  `jfusa`) so they're on `PATH`. A language with no matching tool installed is
  reported as **skipped**, not fatal — you get partial coverage, not an error.

You do not need a config file to start. FuSaOps works zero-config by
detecting languages directly from the files on disk.

## Step 0: Try it with zero setup (Docker)

```bash
docker run --rm -v "$(pwd)":/project ghcr.io/soundmatt/fusaops scan
docker run --rm -v "$(pwd)":/project ghcr.io/soundmatt/fusaops check
```

`scan` tells you what FuSaOps *would* run (languages detected, adapters
matched, tool installed y/n). `check` actually runs it and prints findings.
Exit code is `1` if any ERROR-severity finding exists — that's the signal a CI
gate keys off later. If this is your first look at FuSaOps on this repo, stop
here and read the output before going further.

## Step 1: Confirm what FuSaOps sees

```bash
fusaops scan       # languages detected + file counts
fusaops adapters   # which tools are installed vs skipped
```

If a language you expected is missing from `scan`, check its file extensions
are recognised (see the [Supported languages](../README.md#supported-languages)
table). If a tool shows as skipped in `adapters`, install it or switch to the
Docker image, which bundles all six.

## Step 2: Run a full check

```bash
fusaops check                      # exit 1 on any ERROR finding
fusaops check --strict             # also exit 1 on WARNING findings
fusaops report --format html --output fusaops-report.html   # human-readable report
```

This is the core loop: `check` for CI (fast, exit-code driven), `report` for a
document a person reads. Both accept `--format text|json|html|sarif|junit|csv|markdown`.

## Step 3: Add a config file (optional, but recommended past a first look)

```bash
fusaops init        # writes a starter .fusaops.json
```

Edit it to name the project, restrict which adapters run, exclude vendored
directories, or pin specific sub-directories of a monorepo to specific
adapters:

```json
{
  "version": "1",
  "project": { "name": "my-system", "standard": "ISO26262" },
  "scan": { "adapters": ["gofusa", "cpfusa"], "exclude": ["third_party"] },
  "report": { "format": "html", "output": "fusaops-report.html" }
}
```

Validate it any time with `fusaops config validate` (exit 1 on error — good as
a CI pre-flight step before `check`). Per-language rule configuration (e.g. a
Go module's own `.fusa.json`) stays with that tool; FuSaOps doesn't manage it.

## Step 4: Wire into CI

Zero-install, using the reusable GitHub Action (wraps the Docker image):

```yaml
steps:
  - uses: actions/checkout@v4
  - uses: SoundMatt/FuSaOps/.github/actions/fusaops@v0.9.0
    # runs "fusaops check" by default; exit 1 on any ERROR finding, any language
```

Or install the binary directly in the runner:

```yaml
- run: go install github.com/SoundMatt/FuSaOps/cmd/fusaops@latest
- run: fusaops check
```

See [`.github/fusaops-example.yml`](../.github/fusaops-example.yml) for more
patterns (uploading the HTML report as an artifact, `--strict`, etc.).

## Step 5: Gate on regressions, not the whole backlog (optional)

If a repo has pre-existing findings you're not fixing today, don't block CI on
all of them — gate on *new* ones instead:

```bash
# once, to establish where you are today:
fusaops check --save-baseline baseline.json

# in CI thereafter:
fusaops diff --baseline baseline.json --strict
```

`diff` matches findings by a stable fingerprint (spec §4.2), so it survives
line-number churn. For a stricter, still-explicit alternative to a moving
baseline, use `.fusaops-suppress.json` to acknowledge specific findings by
fingerprint with a reason (and optional expiry) — see the
[Finding suppression](../README.md#finding-suppression) section of the README.

## Step 6: Add the dashboard (optional)

```bash
fusaops serve --addr :8080
# open http://localhost:8080
```

Add `--webhook <url>` to POST a status change (e.g. PASS → FAIL) to Slack or
PagerDuty, `--refresh-interval 5m` for automatic background rescans, or
`--fleet fleet.json` / `--projects projects.json` to dashboard multiple
repositories from one process. See the README's
[Web dashboard](../README.md#web-dashboard-v01--v06--v07) section for the
full endpoint list, including a Prometheus `/metrics` exporter and SVG status
badges for embedding in a README.

## Step 7: Deeper safety evidence (optional — for an actual safety case)

If you're assembling evidence for ISO 26262 / IEC 61508 / ISO 21434 / DO-178C,
not just running lint-style checks:

```bash
fusaops trace                # cross-language requirement traceability + qualification
fusaops trace --strict       # CI gate: fail on any untraced/untested requirement
fusaops sbom --format spdx   # merged cross-language SBOM (SPDX 2.3)
fusaops audit-pack           # bundle every language's evidence into one ZIP for auditors
fusaops iso26262             # cross-language ISO 26262 gap roll-up (also: iec61508, do178, iso21434, unece, iec62443)
```

Each of these depends on the per-language tools implementing the matching
`x-FuSa` command (`trace`, `release`, the standards gap reports) — the
`fusaops adapters` output and `docs/conformance.md` tell you which tools
support which evidence.

## Troubleshooting

- **A language shows 0 findings but you expected some.** Run `fusaops
  adapters` — the tool for that language may be skipped (not installed).
  Switch to the Docker image or install the tool.
- **`fusaops check` passes locally but fails in CI (or vice versa).** Check
  `.fusaops.json`'s `scan.adapters`/`scan.exclude` aren't scoped differently
  than your local run, and that the same tool versions are in play (the
  Docker image pins bundled-tool versions; a local install may be newer or
  older).
- **You need one specific tool's raw output, not the aggregate.** Every
  adapter runs `<tool> check --format json` under the hood — run that tool
  directly if you need its untouched output before FuSaOps normalises it.

## Where to go next

- Full command reference: the [README](../README.md#usage)'s `Usage` section.
- Per-command deep dives: [`docs/commands/`](commands/).
- Adding a brand-new language's tool to FuSaOps itself (not applying FuSaOps
  to a project): [`docs/extending.md`](extending.md).
- The wire contract every x-FuSa tool implements: [`docs/x-fusa-spec.md`](x-fusa-spec.md).
