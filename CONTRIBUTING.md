# Contributing to FuSaOps

Thanks for helping improve FuSaOps.

## Developer Certificate of Origin (DCO)

Every commit must carry a `Signed-off-by` trailer certifying the
[DCO](https://developercertificate.org/). Add one automatically with:

```bash
git commit -s
```

CI rejects pull requests containing commits without a sign-off.

## Development

```bash
make build     # build the fusaops binary
make test      # go test -race ./...
make cover     # coverage summary (CI gates at >= 80%)
make vet lint  # go vet + golangci-lint
make selfcheck # gate FuSaOps's own Go source with gofusa
```

## Standards

- Keep coverage at or above **80%** — the CI gate matches go-FuSa.
- Annotate new exported behaviour with `//fusa:req REQ-FO-...` so the go-FuSa
  self-check can trace it.
- Run `gofmt`, `go vet`, and `golangci-lint` before pushing.
- FuSaOps has **no external runtime dependencies** (standard library only).
  Keep it that way; the web UI is server-rendered Go with inlined assets.

## Adding a language adapter

See the "Adding a language adapter" section of [ROADMAP.md](ROADMAP.md).
Implement `adapter.Adapter`, register it, and add tests with a fake runner so
the adapter is covered without requiring the real tool binary.
