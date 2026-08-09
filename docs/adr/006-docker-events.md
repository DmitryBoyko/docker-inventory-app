# ADR-006: Docker events for invalidation

## Status

Accepted (Phase 0)

## Context

Pure polling either lags or hammers the daemon.

## Decision

Consume Docker Events to invalidate / refresh inventory and system collectors. Stats remain on a short poll loop of running containers.

## Alternatives

- Polling only
- Per-container long-lived stats streams without a central collector

## Why

Fresher UI without N×inspect every second.

## Consequences

Need reconnect/backoff when the event stream dies; fall back to polling and expose `eventsConnected=false`. Not implemented in Phase 1.
