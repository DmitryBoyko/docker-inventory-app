# ADR-007: Domain models separate from SDK

## Status

Accepted (Phase 0)

## Context

Moby SDK structs change across releases and include fields unsuitable for a public UI API (and secrets in inspect).

## Decision

Pipeline: Engine API → SDK → mapper → `internal/domain` → DTO/JSON.

## Alternatives

- Return SDK JSON directly to the frontend

## Why

Stable contract, explicit units/nullability, redaction boundary, insulation from SDK churn.

## Consequences

Mapping code must be maintained and golden-tested.
