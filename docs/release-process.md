# Release Process

FuSaOps follows [Semantic Versioning](https://semver.org). Releases are cut from
`main` once CI is green.

## Checklist

1. **Green build** — `make test cover vet lint` and the CI matrix pass.
2. **Traceability** — `gofusa trace --dir .` shows every requirement traced and
   tested; `gofusa check --dir .` reports 0 ERROR findings.
3. **Evidence refreshed** — regenerate and commit the evidence bundle:
   ```bash
   gofusa verify        # .fusa-evidence.json
   gofusa qualify       # qualify-report.json
   gofusa release --full # sbom.json, provenance.json, artifact-manifest.json, ...
   gofusa safety-case   # safety-case.json/.md/.mermaid
   gofusa tara          # tara.json/.md
   gofusa fmea          # fmea.json/.csv
   gofusa cyber vuln coupling
   ```
4. **Changelog** — move `## [Unreleased]` entries under the new version with the
   date; update version strings (`fusaops.go`, `docs/tool-safety-manual.md`).
5. **Tag** — `git tag -s vX.Y.Z && git push origin vX.Y.Z` (signed/annotated).

## What the tag triggers

- **`docker-publish.yml`** builds `ghcr.io/soundmatt/fusaops` for the version and
  `latest`, with `pull: true` so the newest bundled x-FuSa tools are included
  (`linux/amd64`).

## Keeping the image current between releases

The bundled image does not require a FuSaOps release to pick up new x-FuSa tool
versions. `tools-monitor.yml` rebuilds `fusaops:latest`:

- on `repository_dispatch` (`xfusa-released`) fired by a tool's release, and
- on a weekly schedule as a fallback.

See `docs/extending.md` for the notification snippet tool repos add.

## Versioning policy

- **MAJOR** — breaking CLI, config (`.fusaops.json`), or report-schema changes.
- **MINOR** — new adapters, commands, or report formats (backward compatible).
- **PATCH** — fixes and evidence/doc updates.

## Provenance

Each release image is built in GitHub Actions from a tagged commit. The
committed `provenance.json` and `sbom.json` record the build inputs; the
`artifact-manifest.json` lists SHA-256 hashes of the generated evidence.
