# ADR-008: Stack derived from Compose labels

## Status

Accepted (Phase 0)

## Context

Docker Engine has no first-class Compose “stack” object. The PowerShell source groups by `com.docker.compose.project`.

## Decision

Derive Stack from container labels:

- `com.docker.compose.project` → stack name (default `standalone`)
- `com.docker.compose.service` → service (API may use `null`; UI may show `-`)
- `com.docker.compose.container-number` → replica ordinal when present

Do not require compose files on disk.

## Alternatives

- Parse compose YAML from the filesystem
- Support only Swarm `com.docker.stack.namespace` (deferred)

## Why

Matches existing PowerShell behavior and works wherever Compose-created containers already run.

## Consequences

Undeployed services in compose files are invisible. Swarm label alias is V2 optional.
