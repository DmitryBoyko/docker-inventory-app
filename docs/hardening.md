# Hardening (Phase 12)

Operational defaults and mitigations for the Docker Visualizer single binary.

## Safe defaults

| Setting | Default | Notes |
|---------|---------|--------|
| Listen | `127.0.0.1:8080` | ADR-009 |
| Auth token | unset | ADR-013: required for non-loopback listen (`--auth-token`) |
| Mutations | none | Read-only product stance |
| Inspect | `redact=true` | Env / secret-like labels stripped |
| Logs | on-demand | Not persisted; response truncated (~512 KiB) |
| Docker timeout | `5s` | Per-request Engine call budget |

## HTTP server limits

| Limit | Value | Rationale |
|-------|-------|-----------|
| `ReadHeaderTimeout` | 5s | Slowloris / stuck clients |
| `IdleTimeout` | 120s | Keep-alive cleanup |
| `MaxHeaderBytes` | 1 MiB | Oversized headers |
| `ReadTimeout` / `WriteTimeout` | unset | Long-lived WebSocket on `/api/v1/ws` |

Per-Docker work uses request contexts (`--docker-timeout`, collector-specific caps). Middleware: auth (ADR-013), panic recover, `X-Request-Id`, debug access log.

## Auth (ADR-013)

- Non-loopback listen without `--auth-token` / `DOCKER_VISUALIZER_AUTH_TOKEN` → process exits.
- When token is set: all `/api/v1/*` except `GET /health` require `Authorization: Bearer` (WS also accepts `?access_token=`).
- Static UI remains public; Settings page stores the token in `localStorage`.
- Token is never logged or returned by `/system/settings` / diagnostics (`authEnabled` only).

## Diagnostics

```http
GET /api/v1/system/diagnostics
```

- **Localhost only** (`RemoteAddr` loopback). Ignores `X-Forwarded-For` (no proxy trust).
- When auth is enabled, Bearer is still required.
- Payload: version/commit, listen, `authEnabled`, Docker status, events stream, WS client count, snapshot ages/counts, collector health, runtime mem/goroutines.

## Collector health

`observability.Registry` records success/error + duration for:

- `inventory`
- `stats`
- `system`
- `events` (connect / reconnect errors)

Surfaced under `data.collectors` in diagnostics.

## Load target (~200 containers)

Synthetic coverage (no live Docker required):

```bash
go test ./internal/domain/ -run Load -count=1
go test ./internal/httpapi/ -run Load -count=1
go test ./internal/domain/ -bench Build -benchtime=2s
```

Mitigations for large N (risk register): inspect/stats concurrency caps (16), immutable snapshot swap, WS backpressure, configurable intervals.

## Risk register checklist

| Risk | Verified mitigation |
|------|---------------------|
| Docker socket privilege | Default loopback listen; non-loopback requires auth token (fail closed) |
| Inspect secrets leak | Redact default on; docs + OpenAPI |
| Volume size `-1` | Availability model (not coerced to 0) |
| Large N containers | Concurrency limits + 200-container load tests |
| High-frequency stats | Configurable `--stats-interval` |
| Process memory growth | Snapshot replace; logs not retained |
| WS backpressure | Hub drop policy (Phase 7) |
| Daemon disconnect | `/ready` 503 + `connection.status` / `events.status` |
| Stale cache | Polling + events coalesce + snapshot age in API |
| Scope creep mutations | No mutate endpoints; V2 gate |

## Troubleshooting

1. `/ready` 503 → Docker discovery/ping; check Desktop context / named pipe / socket perms.
2. Empty inventory on APK-like packaging N/A — this is a host ops tool.
3. Support dump: from the same machine, `curl -s http://127.0.0.1:8080/api/v1/system/diagnostics`.
