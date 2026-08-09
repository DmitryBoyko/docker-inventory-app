# Parity harness (Go ↔ PowerShell)

Compares the Go inventory snapshot to [`scripts/docker-stack-inventory.ps1`](../scripts/docker-stack-inventory.ps1) using a shared JSON schema (`schemaVersion: 1`).

## Rules (implementation plan §27)

| Field | Rule |
|-------|------|
| Stack / service / state / restartCount | Exact (`service` `-` ↔ null normalized) |
| Health | Normalized (`-` / `none`, `starting`, …) |
| Container set | Same names |
| Writable layer / volume bytes | Exact when both known; PS `?→0` via `-ps-volume-zero` |
| CPU | Absolute tolerance **0.15** percentage points |
| Memory | Exact (prefer `-skip-stats` if samples are far apart) |
| Port exposures | Set of `public` / `localhost` / `specific` / `internal` |
| Unique volumes | Same name set |

## Unit tests (no Docker)

```bash
go test ./internal/parity/ -count=1
```

## Live compare (same host)

Requires Docker Engine access + PowerShell (`powershell.exe` on Windows, `pwsh` elsewhere).

```bash
# From repo root
go run ./cmd/parity-check -skip-stats

# Keep both JSON artifacts
go run ./cmd/parity-check -go-out tmp/go.json -ps-out tmp/ps.json -skip-stats

# PS exporter alone
pwsh -File scripts/docker-stack-inventory.ps1 -JsonOut tmp/ps.json
```

Exit code `0` = pass, `1` = diffs (JSON report on stdout), `2` = tooling/Docker failure.

## Schema

```json
{
  "schemaVersion": 1,
  "source": "go|powershell",
  "capturedAt": "...",
  "containers": [{ "name", "stack", "service", "state", "health", "restartCount", "writableLayerBytes", "volumeNames", "cpuPercent", "memoryBytes", "portExposures" }],
  "stacks": [{ "name", "containerCount", "runningCount", "cpuPercent", "memoryBytes", "volumeNames", "volumeBytes", ... }],
  "totals": { "containerCount", "runningCount", "cpuPercent", "memoryBytes", "writableLayerBytes", "uniqueVolumeNames", "uniqueVolumeBytes" }
}
```

## Documented diffs

- **Volume unknown:** PS sums `?` as **0**; Go keeps `available:false`. Use `-ps-volume-zero` (default on) for PS-compatible compare, or turn it off to require Go/API availability.
- **Stats window:** CLI `docker stats` and Engine API one-shots are not simultaneous — use `-skip-stats` for structural parity, or accept CPU within 0.15.
