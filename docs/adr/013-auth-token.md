# ADR-013: Bearer token for non-localhost bind

## Status

Accepted (V2)

## Context

Binding the UI/API to a non-loopback address exposes a privileged Docker view to the network. Phase 12 warned on non-loopback listen; operators still need a supported way to bind LAN/VPN with a minimal gate.

## Decision

- Flag/env: `--auth-token` / `DOCKER_VISUALIZER_AUTH_TOKEN`.
- If listen is **not** loopback and the token is empty → **refuse to start**.
- If the token is set (any listen address) → require auth on `/api/v1/*` except `GET /api/v1/health` (and `/health`).
- HTTP: `Authorization: Bearer <token>` (constant-time compare).
- WebSocket: also accept `?access_token=` (browsers cannot set WS Authorization headers).
- Static UI assets remain public so the SPA can collect/store the token.
- Token is never logged or returned in diagnostics/settings (only `authEnabled`).

## Alternatives

- mTLS / OAuth / reverse-proxy-only auth
- Cookie sessions

## Why

Matches the security table in the implementation plan (optional token for non-localhost) with fail-closed startup and zero new dependencies.

## Consequences

Operators must pass a token when using `0.0.0.0` / LAN binds. SPA stores the token in `localStorage` for API + WS. Proxy deployments should still prefer terminating auth at the edge.
