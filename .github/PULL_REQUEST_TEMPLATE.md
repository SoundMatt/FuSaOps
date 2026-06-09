## Summary

<!-- What does this PR do? One paragraph. -->

## Motivation

<!-- Why is this change needed? Reference ROADMAP.md items where relevant. -->

## Safety checklist

- [ ] `go test -race -count=1 ./...` passes locally
- [ ] `go vet ./...` and `golangci-lint run ./...` pass
- [ ] New or changed behaviour has `//fusa:req` and `//fusa:test` annotations
- [ ] `gofusa trace` shows requirements fully traced **and** tested
- [ ] Test coverage stays at or above the 80% CI gate
- [ ] If a new adapter was added: `Detect`/`Check` tests use a fake runner, and
      the Dockerfile tool stage + `docs/extending.md` were updated
- [ ] `CHANGELOG.md` updated (add entry under `## [Unreleased]`)
- [ ] All commits are signed off (`Signed-off-by:` trailer present)

## Type of change

- [ ] Bug fix
- [ ] New feature / adapter
- [ ] Refactor / cleanup
- [ ] Documentation
- [ ] CI / tooling

## Related issues / roadmap

<!-- Closes #NNN  |  Part of vX.Y scope in ROADMAP.md -->
