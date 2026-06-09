# Extending FuSaOps with a new x-FuSa tool

FuSaOps is designed so that adding a new language tool is a small, mechanical
change. This guide covers both halves: the **adapter** (so `fusaops` knows how
to run the tool) and the **image** (so the tool ships in the all-in-one
container and stays fresh automatically).

Assume a hypothetical `rust-FuSa` whose CLI is `rustfusa`, published as
`ghcr.io/soundmatt/rust-fusa`.

## 1. Register the adapter (Go)

Add `adapter/rustfusa.go`, mirroring `adapter/gofusa.go`:

```go
package adapter

import fusaops "github.com/SoundMatt/FuSaOps"

func newRustFuSa() *cmdAdapter {
	return &cmdAdapter{
		name:       "rust-FuSa",
		language:   fusaops.LangRust, // add the constant in fusaops.go
		tool:       "rustfusa",
		extensions: []string{".rs"},
		run:        defaultRunner,
	}
}

func init() { Default.MustRegister(newRustFuSa()) }
```

Then:

- add `LangRust Language = "rust"` in `fusaops.go`;
- add `fusaops.LangRust: {".rs"}` to `langExtensions` in `scan/scan.go`;
- add a test with a fake runner (see `adapter/adapter_test.go`).

That is all the orchestrator needs — `check`, `report`, and `serve` pick the
adapter up automatically. The tool only has to honour the common contract:
`<tool> check --dir <path> --format json --output <file>`.

## 2. Bundle the tool in the image

The image copies each tool binary out of its published image. Add two lines to
the `Dockerfile`:

```dockerfile
FROM ghcr.io/soundmatt/rust-fusa:latest AS rustfusa   # tool stage
...
COPY --from=rustfusa /usr/local/bin/rustfusa /usr/local/bin/rustfusa
```

Requirements for a tool image to be bundlable:

- it must place its binary at a predictable path (`/usr/local/bin/<tool>`);
- its base must be runtime-compatible with FuSaOps's `alpine` base (static, or
  musl + any libs added via `apk`). glibc-only images cannot be copied in — give
  the tool an alpine-based image first.

## 3. Keep it fresh automatically

Add the tool to the monitor and tell its repo to notify FuSaOps on release.

**FuSaOps side** — nothing extra: `tools-monitor.yml` rebuilds the whole image
with `pull: true`, so the new tool stage is included on the next refresh.

**Tool repo side** — add a step to the tool's release / docker-publish workflow
so a new release triggers an immediate FuSaOps rebuild:

```yaml
- name: Notify FuSaOps to rebuild
  run: |
    curl -sf -X POST \
      -H "Authorization: Bearer ${{ secrets.FUSAOPS_DISPATCH_TOKEN }}" \
      -H "Accept: application/vnd.github+json" \
      https://api.github.com/repos/SoundMatt/FuSaOps/dispatches \
      -d '{"event_type":"xfusa-released","client_payload":{"tool":"rust-FuSa","version":"${{ github.ref_name }}"}}'
```

`FUSAOPS_DISPATCH_TOKEN` is a fine-grained PAT with **Contents: read & write**
(or classic `repo`) scope on `SoundMatt/FuSaOps`, stored as a secret in the tool
repo. Without it, the weekly scheduled rebuild in `tools-monitor.yml` still
picks the tool up — just less promptly.

## Checklist

- [ ] `adapter/<tool>.go` + `Lang<X>` constant + `scan` extension + test
- [ ] `Dockerfile`: tool `FROM` stage + `COPY --from` line
- [ ] tool image is alpine-compatible and exposes `/usr/local/bin/<tool>`
- [ ] tool repo sends the `xfusa-released` dispatch on release
- [ ] update `README.md` supported-languages table
