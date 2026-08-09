# ADR-003: Moby Go client module

## Status

Accepted (Phase 0)

## Context

`github.com/docker/docker` is deprecated as of Docker Engine v29. Docker documents `github.com/moby/moby/client` and `github.com/moby/moby/api` as the supported public modules.

## Decision

Use `github.com/moby/moby/client` (and `github.com/moby/moby/api` types internally in adapters/mappers only).

## Alternatives

- Legacy `github.com/docker/docker`
- `github.com/docker/go-sdk` high-level wrapper
- Raw HTTP to Engine API

## Why

Official low-level Engine client with API version negotiation and maintained platform transports (unix socket, Windows named pipe, TCP/TLS).

## Consequences

Track client/api module version churn (v29+ renames). Never expose SDK types on the public REST surface (see ADR-007).
