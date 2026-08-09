# ADR-010: Docker endpoint discovery

## Status

Accepted (Phase 0)

## Context

Hardcoding `/var/run/docker.sock` or only the Windows named pipe breaks Docker Desktop on macOS/Linux and non-default contexts.

## Decision

Resolve the Engine endpoint in this order:

1. Explicit `--docker-host` / `DOCKER_VISUALIZER_DOCKER_HOST`
2. `DOCKER_HOST`
3. Current Docker context endpoint (`~/.docker/config.json` + context meta)
4. SDK / platform default (`unix:///var/run/docker.sock` or `npipe:////./pipe/docker_engine`)

Use API version negotiation. Surface actionable errors (not found, permission denied, daemon down, TLS).

Remote `tcp://` TLS uses standard Docker CLI env via Moby `WithTLSClientConfigFromEnv()` (`DOCKER_CERT_PATH`, `DOCKER_TLS_VERIFY`, …).

## Alternatives

- GOOS switch with fixed paths only
- Require users to always set `DOCKER_HOST`

## Why

Matches how Docker CLI finds the engine across Desktop and Engine installs.

## Consequences

Context store parsing must be unit-tested.

**Operator how-to (VPS walkthrough, examples, troubleshooting):** [`../../MANUAL.md`](../../MANUAL.md) §2–§5.
