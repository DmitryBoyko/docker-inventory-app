# Docker Visualizer

[![CI](https://github.com/DmitryBoyko/docker-inventory-app/actions/workflows/ci.yml/badge.svg)](https://github.com/DmitryBoyko/docker-inventory-app/actions/workflows/ci.yml)
[![Go](https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go&logoColor=white)](go.mod)
[![License](https://img.shields.io/badge/license-proprietary-lightgrey)](#license)

Cross-platform, **read-only** Docker inventory: one Go binary serves the API and an embedded React UI. Browse containers, stacks, networks, volumes, images, and system usage — plus CLI companion helpers (commands, diagnostics, snapshots). No arbitrary shell execution.

## Features

- **Live inventory** — containers, Compose stacks, networks, volumes, images, graph, export (JSON/CSV)
- **Multi-host** — named Docker endpoints (`--docker-hosts`); UI host picker
- **Observability** — stats, log follow (WS), optional SQLite metrics history
- **CLI Companion** — Command Registry (Bash / PowerShell / CMD), provenance tooltips, anomaly diagnostics, sanitized inventory snapshots
- **i18n** — English and Russian UI (Docker commands and API field names stay English)
- **Single binary** — UI embedded at build time; optional on-disk `web/dist` for local UI iteration
- **Auth** — Bearer token required when binding outside loopback (ADR-013)

## Requirements

| Component | Version |
|-----------|---------|
| Go | 1.25+ (see `go.mod`) |
| Node.js | 24+ (UI build / Vite) |
| Docker Engine | reachable via socket, named pipe, or TCP |

## Quick start

### Windows EXE (recommended on Windows)

Prerequisites: [Go](https://go.dev/dl/) 1.25+, [Node.js](https://nodejs.org/) 24+, Docker Desktop running.

```powershell
# From repo root — build EXE with embedded UI
.\scripts\build-exe.ps1

# Run (builds automatically if bin\docker-visualizer.exe is missing)
.\scripts\run-exe.ps1
.\scripts\run-exe.ps1 -OpenBrowser
```

Then open [http://127.0.0.1:8080](http://127.0.0.1:8080).

| Output | Path |
|--------|------|
| Local run | `bin\docker-visualizer.exe` |
| Distribution name | `bin\docker-visualizer-windows-amd64.exe` |

Common run options:

```powershell
.\bin\docker-visualizer.exe -h
.\bin\docker-visualizer.exe --listen 127.0.0.1:8080
.\bin\docker-visualizer.exe --listen 0.0.0.0:8080 --auth-token "your-secret"
.\scripts\run-exe.ps1 -Listen 0.0.0.0:8080 -AuthToken "your-secret"
```

If PowerShell blocks scripts: `Set-ExecutionPolicy -Scope CurrentUser RemoteSigned`.

### Development (hot UI)

```bash
# terminal 1 — API (uses web/dist if present, else embedded UI)
go run ./cmd/docker-visualizer

# terminal 2 — Vite with /api proxy
cd web && npm install && npm run dev
```

Open the Vite URL (usually `http://127.0.0.1:5173`). API defaults to `http://127.0.0.1:8080`.

### Linux / macOS binary

```bash
make build
./bin/docker-visualizer
```

Cross-compile / release:

```bash
make build-all                 # or: .\scripts\build.ps1 -Cross
make release-snapshot          # optional GoReleaser snapshot
```

### UI resolution order

1. On-disk `web/dist` (after `npm run build`)
2. Embedded `internal/uiembed/dist` (release builds)
3. API-only if neither is available

## Configuration

Docker host discovery: `--docker-host` / `DOCKER_VISUALIZER_DOCKER_HOST` → `DOCKER_HOST` → Docker context → platform default.

| Flag | Default | Notes |
|------|---------|--------|
| `--listen` | `127.0.0.1:8080` | Bind address |
| `--auth-token` | _(empty)_ | **Required** when listen is not loopback |
| `--docker-host` | _(auto)_ | Single Engine endpoint |
| `--docker-hosts` | _(empty)_ | `name=url,name2=url` multi-host; empty ⇒ single `default` |
| `--docker-config` | Docker default | Client config dir |
| `--docker-timeout` | `5s` | Engine API timeout |
| `--inventory-interval` | `10s` | Inventory refresh |
| `--stats-interval` | `1s` | Live stats cadence |
| `--system-interval` | `15s` | System info refresh |
| `--metrics-db` | `data/metrics.db` | SQLite history; `off` disables |
| `--metrics-interval` | `10s` | History sample interval |
| `--metrics-retention` | `24h` | History retention |
| `--snapshots-dir` | `data/snapshots` | Inventory snapshots; `off` disables |

Multi-host example:

```bash
go run ./cmd/docker-visualizer \
  --docker-hosts "local=npipe:////./pipe/docker_engine,lab=tcp://192.168.1.10:2376"
```

REST and WebSocket accept `?host=<name>` (omit → default host). Auth: `Authorization: Bearer <token>` (or `?access_token=` on WS). UI: Settings → store token in `localStorage`.

## API

REST under `/api/v1/*`. Full contract: [`openapi.yaml`](openapi.yaml).

| Area | Examples |
|------|----------|
| Health / hosts | `/health`, `/ready`, `/hosts` |
| Inventory | `/containers`, `/stacks`, `/networks`, `/volumes`, `/images`, `/graph`, `/export` |
| Live | `/containers/{id}/stats`, `/logs`, `/logs/ws`, `/ws` |
| System | `/system/df`, `/resources`, `/info`, `/settings`, `/diagnostics` (localhost support dump) |
| Companion | `/commands`, `/diagnostics` (findings), `/provenance`, `/snapshots` |
| Metrics | `/metrics/history` |

WebSocket event types: `container.stats`, `docker.event`, `snapshot.updated`, `connection.status`, `events.status`.

## Documentation

| Doc | Purpose |
|-----|---------|
| [`docs/implementation-plan.md`](docs/implementation-plan.md) | Architecture & phases |
| [`docs/hardening.md`](docs/hardening.md) | Security hardening |
| [`docs/cli-companion.md`](docs/cli-companion.md) | CLI Companion overview |
| [`docs/adr/`](docs/adr/) | Architecture Decision Records (013–016+) |
| [`docs/todos.md`](docs/todos.md) | Open TODOs |
| [`web/`](web/) | React SPA source |
| [`scripts/docker-stack-inventory.ps1`](scripts/docker-stack-inventory.ps1) | Legacy PowerShell inventory |

## Development

```bash
go test ./...
go test -tags=integration ./internal/docker/ -v
cd web && npm test && npm run build

# Live Go ↔ PowerShell parity (needs Docker + pwsh/powershell)
go run ./cmd/parity-check -skip-stats
```

CI runs on `main` / PRs: sync UI embed → `go test` → host build → cross-compile matrix (see [`.github/workflows/ci.yml`](.github/workflows/ci.yml)).

## Security

- **Read-only** by design: inventory and CLI *generation* only — no `POST /exec` / PTY.
- Snapshots are sanitized (no Env / secrets).
- Non-loopback binds require `--auth-token`.
- `/api/v1/system/diagnostics` is localhost-only.

## License

No open-source license is published in this repository yet. Treat the code as proprietary unless a `LICENSE` file is added.
