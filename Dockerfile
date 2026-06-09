# syntax=docker/dockerfile:1
#
# Multi-stage build for FuSaOps.
# Stage 1 compiles the fusaops binary and installs the go-FuSa adapter tool;
# Stage 2 produces a minimal runtime image.
#
# Build:
#   docker build -t fusaops .
#
# Run (mount your repo at /project):
#   docker run --rm -v "$(pwd)":/project fusaops scan
#   docker run --rm -v "$(pwd)":/project fusaops check
#   docker run --rm -p 8080:8080 -v "$(pwd)":/project fusaops serve --addr :8080
#
# The image bundles gofusa so Go projects work out of the box. To scan C/C++
# mount cfusa / cpfusa binaries onto PATH, or use the language-specific images.

# ── Stage 1: build ────────────────────────────────────────────────────────────
FROM golang:1.22-alpine AS builder

WORKDIR /build
COPY go.mod ./
COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w -extldflags=-static" \
    -o /bin/fusaops ./cmd/fusaops

# Bundle the go-FuSa adapter tool.
RUN CGO_ENABLED=0 GOBIN=/bin go install github.com/SoundMatt/go-FuSa/cmd/gofusa@latest

# ── Stage 2: runtime ─────────────────────────────────────────────────────────
FROM alpine:3.20

RUN apk add --no-cache git ca-certificates

COPY --from=builder /bin/fusaops /usr/local/bin/fusaops
COPY --from=builder /bin/gofusa /usr/local/bin/gofusa

WORKDIR /project
EXPOSE 8080

ENTRYPOINT ["fusaops"]
CMD ["help"]
