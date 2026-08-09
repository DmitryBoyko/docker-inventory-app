# ADR-002: Docker Engine API instead of CLI

## Status

Accepted (Phase 0)

## Context

The legacy PowerShell inventory shells out to `docker ps/stats/inspect/system df`. CLI text/JSON formatting is fragile and requires a Docker CLI binary on PATH.

## Decision

At runtime, talk only to the Docker Engine API (via the Go client). Do not exec `docker`.

## Alternatives

- Keep `exec("docker …")` wrappers
- Mix API + CLI for awkward fields

## Why

Structured numeric fields (bytes, counters), no CLI dependency, works in constrained environments where only the engine socket/pipe is available.

## Consequences

Must reimplement CLI-compatible CPU/memory formatting math for parity tests. Human strings like `RunningFor` are computed, not copied from CLI.
