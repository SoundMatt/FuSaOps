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
FROM ghcr.io/soundmatt/go-fusa:latest   AS gofusa
FROM ghcr.io/soundmatt/cpp-fusa:latest  AS cpfusa
FROM ghcr.io/soundmatt/c-fusa:latest    AS cfusa
FROM ghcr.io/soundmatt/rust-fusa:latest AS rsfusa
FROM ghcr.io/soundmatt/py-fusa:latest   AS pyfusa
FROM ghcr.io/soundmatt/java-fusa:latest AS jfusa

# ── Build fusaops ─────────────────────────────────────────────────────────────
FROM golang:1.22-alpine AS build
WORKDIR /build
COPY go.mod ./
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w -extldflags=-static" \
    -o /bin/fusaops ./cmd/fusaops

# ── Runtime ───────────────────────────────────────────────────────────────────
# python:3.12-alpine (= alpine:3.20 + Python 3.12) provides the Python runtime
# needed by pyfusa.  Java is added via openjdk21-jre-headless for jfusa.
FROM python:3.12-alpine

# git + ca-certificates back the tools' provenance / vulnerability features.
# libstdc++ is pre-staged so cpp-FuSa drops in without a base change.
# openjdk21-jre-headless provides the `java` binary required by jfusa.
RUN apk add --no-cache git ca-certificates libstdc++ openjdk21-jre-headless

COPY --from=build  /bin/fusaops          /usr/local/bin/fusaops
COPY --from=gofusa /usr/local/bin/gofusa /usr/local/bin/gofusa
COPY --from=cpfusa /usr/local/bin/cpfusa /usr/local/bin/cpfusa
COPY --from=cfusa  /usr/local/bin/cfusa  /usr/local/bin/cfusa
COPY --from=rsfusa /usr/local/bin/rsfusa /usr/local/bin/rsfusa

# py-FuSa: entry point + installed packages (same python:3.12-alpine base → ABI compatible)
COPY --from=pyfusa /usr/local/bin/pyfusa /usr/local/bin/pyfusa
COPY --from=pyfusa /usr/local/lib/python3.12/site-packages /usr/local/lib/python3.12/site-packages

# java-FuSa: JAR + shell wrapper that calls `java -jar`
COPY --from=jfusa /usr/local/lib/jfusa.jar /usr/local/lib/jfusa.jar
COPY --from=jfusa /usr/local/bin/jfusa     /usr/local/bin/jfusa

WORKDIR /project
EXPOSE 8080
ENTRYPOINT ["fusaops"]
CMD ["help"]
