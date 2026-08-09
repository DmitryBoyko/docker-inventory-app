# ADR-016: CLI Companion — Command Registry, Diagnostics, Snapshots

- Status: Accepted
- Date: 2026-08-09

## Context

The Visualizer already exposes a read-only Docker inventory. Users still need a bridge from
UI values to exact Docker CLI / Engine API investigation steps without turning the app into
a remote shell.

## Decision

1. **Command Registry** (`internal/commands`) is the single source of CLI templates.
   Generation is context-aware (`--context` / `-H`) for Bash, PowerShell, and CMD.
2. Commands are **generate + explain + copy only**. No `POST /exec`, no PTY in this phase.
3. **Risk levels** (`READ_ONLY` | `INTERACTIVE` | `STATE_CHANGING` | `DESTRUCTIVE`) are
   attached to every definition for future safe execution policies.
4. **Anomaly diagnostics** live at `GET /api/v1/diagnostics` (findings). The existing
   `GET /api/v1/system/diagnostics` remains the localhost support dump.
5. **Provenance** catalog documents Engine API fields and transforms for key UI metrics.
6. **Persisted inventory snapshots** are stored under `--snapshots-dir` (default
   `data/snapshots`), sanitized (no Env/secrets). Diff compares A↔B or A↔current.
7. UI copy is localized via frontend i18n (EN/RU); Docker commands and API field names
   are never translated.

## Consequences

- Frontend must not hardcode docker CLI strings.
- Snapshots must not persist inspect Env or credential-like labels.
- Future terminal execution can reuse registry + risk gates without redesigning the API.
