# ADR-012: In-memory snapshot store

## Status

Accepted (Phase 0)

## Context

Per-request Docker fan-out (`N` inspect + stats) does not scale and races with the UI.

## Decision

Background collectors publish an atomic in-memory `Snapshot`. REST handlers read the snapshot (except on-demand logs/inspect).

## Alternatives

- Fetch from Docker on every HTTP request
- External cache (Redis) — unnecessary for a local utility

## Why

Consistent aggregates, bounded daemon load, simple deployment.

## Consequences

Data can be stale up to collector intervals; expose snapshot age to the UI. Collectors are not implemented in Phase 1.
