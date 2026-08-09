# ADR-005: REST + limited WebSocket

## Status

Accepted (Phase 0)

## Context

Early proposals streamed the full inventory over WebSocket every second.

## Decision

- REST (`/api/v1`) for inventory, metadata, details, inspect, log snapshots
- WebSocket for live stats, docker connection status, and lightweight snapshot notifications

## Alternatives

- WebSocket-everything
- SSE-only
- REST polling only

## Why

Inventory is cacheable and relatively slow-changing; stats are high-frequency. Splitting transports reduces bandwidth and backpressure risk.

## Consequences

Two transports to test. Phase 1 ships only REST `/health` and `/ready`; WebSocket arrives in a later phase.
