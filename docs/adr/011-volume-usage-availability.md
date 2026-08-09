# ADR-011: Volume usage availability semantics

## Status

Accepted (Phase 0)

## Context

`GET /system/df` reports `UsageData.Size = -1` when size is unavailable (non-local drivers, etc.). The PowerShell script treats missing parse results as `0`, which lies.

## Decision

Model sizes as:

```json
{ "bytes": null, "available": false, "reason": "unsupported" }
```

Aggregates must expose `partial` / `unknownCount` and must not silently coerce unknown to zero.

## Alternatives

- Always `int64` with `0` meaning unknown
- Omit volume sizes entirely

## Why

Honest UI and safe totals.

## Consequences

Parity with PowerShell totals may differ when sizes are unknown; document as intentional improvement.
