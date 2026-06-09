# `fusaops serve`

Launch the web reporting dashboard.

```bash
fusaops serve [--dir <path>] [--only <tools>] [--addr <addr>]
```

## Flags

| Flag | Default | Description |
|---|---|---|
| `--dir` | `.` | Project root |
| `--only` | all applicable | Restrict to specific tools |
| `--addr` | `:8080` | Listen address |

## Routes

| Path | Description |
|---|---|
| `/` | HTML dashboard: status badge, per-language cards, filterable findings |
| `/api/report` | The aggregate report as JSON |
| `/refresh` | Recompute the report, then redirect to `/` |
| `/healthz` | Liveness probe |

## Safety note

The dashboard has **no authentication**. Bind to localhost or place it behind an
authenticating proxy; do not expose it to untrusted networks. Always `/refresh`
before relying on the dashboard for a gating or audit decision (hazard H-005).

## Docker

```bash
docker run --rm -p 8080:8080 -v "$(pwd)":/project \
  ghcr.io/soundmatt/fusaops serve --addr :8080
```
