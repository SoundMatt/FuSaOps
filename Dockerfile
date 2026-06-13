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
# cpp-FuSa v0.12.5 is spec v1.10 aligned; enable once ghcr.io/soundmatt/cpp-fusa is published
# FROM ghcr.io/soundmatt/cpp-fusa:latest AS cpfusa
# c-FuSa v0.5.16 is spec v1.10 aligned; enable once ghcr.io/soundmatt/c-fusa is published
# FROM ghcr.io/soundmatt/c-fusa:latest   AS cfusa
# rust-FuSa v0.2.6 is spec v1.10 aligned; enable once ghcr.io/soundmatt/rust-fusa is published
# FROM ghcr.io/soundmatt/rust-fusa:latest AS rsfusa
# py-FuSa v0.1.4 is spec v1.10 aligned; enable once ghcr.io/soundmatt/py-fusa is published
# FROM ghcr.io/soundmatt/py-fusa:latest   AS pyfusa
# java-FuSa v0.2.0 is spec v1.10 aligned; enable once ghcr.io/soundmatt/java-fusa is published
# Note: enabling this stage also requires adding `openjdk21-jre` to the apk install line below.
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
# COPY --from=cpfusa /usr/local/bin/cpfusa /usr/local/bin/cpfusa  # uncomment with FROM stage above
# COPY --from=cfusa  /usr/local/bin/cfusa  /usr/local/bin/cfusa   # uncomment with FROM stage above
# COPY --from=rsfusa /usr/local/bin/rsfusa /usr/local/bin/rsfusa  # uncomment with FROM stage above
# COPY --from=pyfusa /usr/local/bin/pyfusa /usr/local/bin/pyfusa  # uncomment with FROM stage above
# COPY --from=jfusa  /usr/local/bin/jfusa  /usr/local/bin/jfusa   # uncomment with FROM stage above

WORKDIR /project
EXPOSE 8080
ENTRYPOINT ["fusaops"]
CMD ["help"]
