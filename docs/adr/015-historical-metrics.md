# ADR-015: Historical metrics (SQLite)

## Status

Accepted (V2)

## Context

Live stats are 1 Hz via WebSocket with only an in-browser ring buffer (~60 points). Operators need longer CPU/RAM trends without an external TSDB. Volume growth history remains out of scope (external TSDB).

## Decision

- Persist **downsampled** host rollups and per-container samples in a local **SQLite** file (`modernc.org/sqlite`, CGO-free).
- Flags: `--metrics-db` (default `data/metrics.db`; `off`/empty disables), `--metrics-interval` (default `10s`), `--metrics-retention` (default `24h`).
- One DB file for the process; rows are tagged with **host** name (ADR-014).
- Writer runs after each stats collect when the sample interval has elapsed for that host.
- REST: `GET /api/v1/metrics/history?host=&scope=host|container&id=&from=&to=&step=`.
- In-memory snapshot store (ADR-012) stays hot-path only — history is not merged into inventory snapshots.

## Alternatives

- Prometheus / remote TSDB (ops overhead for a local tool)
- Per-host DB files (more paths; same queries)
- Keep every 1s sample (disk growth / write amplification)

## Why

Fits single-binary, offline-friendly deployment; enough for 24h dashboards; queryable without federating hosts.

## Consequences

Disk I/O and file under `data/`; UI can seed charts from REST then tip with WS. Disable with `--metrics-db=off` when unwanted.
