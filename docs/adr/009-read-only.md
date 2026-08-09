# ADR-009: Read-only by default

## Status

Accepted (Phase 0)

## Context

Access to the Docker socket/pipe is effectively privileged.

## Decision

MVP/V1 expose only read APIs. Default listen address is `127.0.0.1`. No start/stop/rm/pull endpoints unless a future ADR enables an explicit mutate flag.

## Alternatives

- Full Portainer-like control plane from day one

## Why

Dramatically reduces blast radius of a compromised or mis-bound UI.

## Consequences

Operators still use Docker CLI/other tools to mutate state.
