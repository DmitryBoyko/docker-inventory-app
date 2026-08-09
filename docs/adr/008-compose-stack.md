# ADR-008: Stack derived from Compose (and Swarm) labels

## Status

Accepted (Phase 0); Swarm alias added in V2.

## Context

Docker Engine has no first-class Compose “stack” object. The PowerShell source groups by `com.docker.compose.project`. Swarm stack deploy uses `com.docker.stack.namespace`.

## Decision

Derive Stack from container labels:

1. `com.docker.compose.project` → stack name (Compose, PowerShell parity)
2. Else `com.docker.stack.namespace` → stack name (Swarm)
3. Else `standalone`

Service:

- Compose: `com.docker.compose.service` (API may use `null`; UI may show `-`)
- Swarm: `com.docker.swarm.service.name`, with `{namespace}_` prefix stripped when present
- Compose: `com.docker.compose.container-number` → replica ordinal when present

Do not require compose files on disk. Compose labels win when both are present.

## Alternatives

- Parse compose YAML from the filesystem
- Swarm-only grouping

## Why

Matches existing PowerShell behavior for Compose and covers Swarm stack deploy without a separate inventory path.

## Consequences

Undeployed services in compose files are invisible. Mixed Compose+Swarm hosts use Compose identity when the compose project label exists.
