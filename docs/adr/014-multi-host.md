# ADR-014: Multi-host Docker endpoints

## Status

Accepted (V2)

## Context

Operators often have more than one Engine (local Desktop + remote/lab). V1 assumed a single endpoint (ADR-010 / ADR-012).

## Decision

- Configure a **named host registry** at process start (`--docker-hosts name=url,...`).
- If omitted, one host named `default` is created via ADR-010 discovery (`--docker-host` / env / context).
- Each host has its own Engine client, snapshot store, and collector set.
- REST selects host with `?host=<name>` (omit → default). No cross-host merged inventory.
- WebSocket envelopes carry `host`; clients filter to their selected host (`?host=` on `/api/v1/ws`).
- Hosts are **not** added dynamically from the UI (SSRF); only configured endpoints.
- Remote `tcp://` uses standard Docker TLS env (`DOCKER_CERT_PATH` / `DOCKER_TLS_VERIFY` / …) via the Moby client — **one TLS env set per process** (different cert directories ⇒ separate processes or a shared CA).

## Alternatives

- Path prefix `/api/v1/hosts/{name}/...` (more mux churn)
- Federated single inventory (rejected for MVP multi-host)
- Per-host cert paths in the flag string (deferred; use multiple processes for now)

## Why

Smallest change to the existing flat `/api/v1/*` API while keeping snapshots isolated and fail-closed on unknown host names.

## Consequences

UI must send `host` on REST/WS. Diagnostics/settings expose the host list. Memory/CPU scale roughly linearly with host count × collector load.

**Operator how-to (multi-host examples + VPS checklist):** [`../../MANUAL.md`](../../MANUAL.md) §3–§4.
