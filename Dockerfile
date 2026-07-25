# syntax=docker/dockerfile:1
#
# FuSaOps — all-in-one multi-language functional safety image.
#
# Tools are NOT built here. Each x-FuSa tool's published image is the single
# source of truth; this image copies the tool binary straight out of it. A tool
# release is therefore picked up by *rebuilding* this image (automatically — see
# .github/workflows/tools-monitor.yml), never by editing anything here.
#
# Adding a future x-FuSa is a one-liner: add a `FROM ... AS <tool>` stage and a
# matching COPY line, register the adapter in Go, and add the tool to the
# tools-monitor matrix. See docs/extending.md.
#
# NOTE: the tool images are linux/amd64; build this image for amd64 too:
#   docker build --platform linux/amd64 -t fusaops .
#
# Run (mount your repo at /project):
#   docker run --rm -v "$(pwd)":/project ghcr.io/soundmatt/fusaops scan
#   docker run --rm -v "$(pwd)":/project ghcr.io/soundmatt/fusaops check
#   docker run --rm -p 8080:8080 -v "$(pwd)":/project ghcr.io/soundmatt/fusaops serve --addr :8080

# ── Tool stages (source = each x-FuSa's published image) ──────────────────────
FROM ghcr.io/soundmatt/go-fusa:latest AS gofusa
FROM ghcr.io/soundmatt/cpp-fusa:latest AS cpfusa
# c-FuSa v0.5.33 — awaiting docker-publish.yml (c-FuSa#44); enable once published
# FROM ghcr.io/soundmatt/c-fusa:latest   AS cfusa
# rust-FuSa v0.2.8 — awaiting docker-publish.yml (rust-FuSa#15); enable once published
# FROM ghcr.io/soundmatt/rust-fusa:latest AS rsfusa
# py-FuSa v0.1.8 — awaiting docker-publish.yml (py-FuSa#5); enable once published
# FROM ghcr.io/soundmatt/py-fusa:latest   AS pyfusa
# java-FuSa v0.3.1 — awaiting Dockerfile+docker-publish.yml (java-FuSa#9); also needs openjdk21-jre below
# FROM ghcr.io/soundmatt/java-fusa:latest AS jfusa

# ── Build fusaops ─────────────────────────────────────────────────────────────
FROM golang:1.22-alpine AS build
WORKDIR /build
COPY go.mod ./
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w -extldflags=-static" \
    -o /bin/fusaops ./cmd/fusaops

# ── Runtime ───────────────────────────────────────────────────────────────────
FROM alpine:3.20

# git + ca-certificates back the tools' provenance / vulnerability features.
# libstdc++ is pre-staged so cpp-FuSa drops in without a base change.
RUN apk add --no-cache git ca-certificates libstdc++

COPY --from=build  /bin/fusaops          /usr/local/bin/fusaops
COPY --from=gofusa /usr/local/bin/gofusa /usr/local/bin/gofusa
COPY --from=cpfusa /usr/local/bin/cpfusa /usr/local/bin/cpfusa
# COPY --from=cfusa  /usr/local/bin/cfusa  /usr/local/bin/cfusa   # enable when c-FuSa#44 is resolved
# COPY --from=rsfusa /usr/local/bin/rsfusa /usr/local/bin/rsfusa  # enable when rust-FuSa#15 is resolved
# COPY --from=pyfusa /usr/local/bin/pyfusa /usr/local/bin/pyfusa  # enable when py-FuSa#5 is resolved
# COPY --from=jfusa  /usr/local/bin/jfusa  /usr/local/bin/jfusa   # enable when java-FuSa#9 is resolved

WORKDIR /project
EXPOSE 8080
ENTRYPOINT ["fusaops"]
CMD ["help"]
