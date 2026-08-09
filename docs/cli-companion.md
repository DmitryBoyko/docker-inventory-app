# CLI Companion (ADR-016)

## Command Registry

Package `internal/commands` owns structured command definitions and shell rendering.

- Shells: Bash, PowerShell, CMD (quoting differs; docker tokens do not).
- Context: `--context <name>` when Engine was resolved via Docker context; otherwise `-H <endpoint>` for explicit/DOCKER_HOST/remote.
- Risk: `READ_ONLY` | `INTERACTIVE` | `STATE_CHANGING` | `DESTRUCTIVE`.
- **No execution** — generate, explain, copy only. Do not add `POST /exec` without a new ADR.

API:

- `GET /api/v1/commands`
- `GET /api/v1/entities/{kind}/commands?ref=&shell=`

## Provenance

`internal/provenance` documents Engine API endpoints/fields and visualizer transforms for key UI metrics (`ⓘ` in the UI).

## Diagnostics (Anomaly Center)

`GET /api/v1/diagnostics` returns explainable findings from the live inventory snapshot.
Distinct from `GET /api/v1/system/diagnostics` (support dump, localhost-only).

Thresholds live in `internal/findings.DefaultThresholds()` (backend constants; configurable later).

## Snapshots

Persisted sanitized inventory under `--snapshots-dir` / `DOCKER_VISUALIZER_SNAPSHOTS_DIR` (default `data/snapshots`; `off` disables).

- Create / list / get / diff (`?against=current|<id>`) / delete
- No Env / secrets; compose labels only

## i18n

Frontend catalogs EN/RU (`web/src/i18n`). Docker commands, API field names, and entity names are never translated.
Language preference: `localStorage` key `dv.locale` (browser default with English fallback).

## Terminal (future)

A future PTY bridge may reuse the Command Registry + risk gates. It is intentionally **not** implemented now.
