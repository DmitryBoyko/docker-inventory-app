# Docker Visualizer

Cross-platform, read-only Docker inventory utility — **one binary** serves API + embedded React UI.

**Status:** Phase 0–12 + V2 + **CLI Companion** (Command Registry, provenance, anomaly diagnostics, inventory snapshots, i18n EN/RU, Ctrl/Cmd+K palette). Still read-only — no arbitrary shell execution.

- Architecture: [`docs/implementation-plan.md`](docs/implementation-plan.md)
- Hardening: [`docs/hardening.md`](docs/hardening.md)
- Companion: [`docs/adr/016-cli-companion.md`](docs/adr/016-cli-companion.md), [`docs/cli-companion.md`](docs/cli-companion.md)
- ADRs: [`docs/adr/`](docs/adr/) (incl. [ADR-013](docs/adr/013-auth-token.md)–[016](docs/adr/016-cli-companion.md))
- Open TODOs: [`docs/todos.md`](docs/todos.md)
- OpenAPI: [`openapi.yaml`](openapi.yaml)
- UI: [`web/`](web/)
- Legacy PowerShell source of truth: [`scripts/docker-stack-inventory.ps1`](scripts/docker-stack-inventory.ps1)

## Quick start (dev)

```bash
# terminal 1 — API (serves disk web/dist if present, else embedded UI)
go run ./cmd/docker-visualizer

# terminal 2 — Vite with /api proxy
cd web && npm run dev
```

Defaults:

- Listen: `127.0.0.1:8080`
- Docker host discovery: `--docker-host` / `DOCKER_VISUALIZER_DOCKER_HOST` → `DOCKER_HOST` → Docker context → platform default

## Single binary (Phase 11)

```bash
# Linux/macOS
make build          # npm build → sync embed → CGO=0 go build → bin/

# Windows (PowerShell)
.\scripts\build.ps1
.\scripts\build.ps1 -Cross   # windows/linux/darwin amd64+arm64
```

Then:

```bash
./bin/docker-visualizer
# open http://127.0.0.1:8080
```

UI resolution order:

1. On-disk `web/dist` (dev override after `npm run build`)
2. Embedded `internal/uiembed/dist` (release)
3. API-only if neither is available

Release with GoReleaser (optional):

```bash
make release-snapshot   # requires goreleaser
```

## API surface

```text
GET /api/v1/health
GET /api/v1/ready                 (+ events.connected; ?host=)
GET /api/v1/hosts                 (named Docker endpoints, ADR-014)
GET /api/v1/containers            (?host=&stack=&state=&health=&q=)
GET /api/v1/containers/{id}
GET /api/v1/containers/{id}/stats
GET /api/v1/containers/{id}/inspect  (?redact=true default)
GET /api/v1/containers/{id}/logs     (?tail=&since=&timestamps=)
GET /api/v1/containers/{id}/logs/ws  WebSocket follow stream
GET /api/v1/stacks
GET /api/v1/stacks/{name}
GET /api/v1/stacks/{name}/volumes
GET /api/v1/networks              (?q=&driver=)
GET /api/v1/networks/{id}
GET /api/v1/volumes               (?stack=&q=)
GET /api/v1/volumes/{name}
GET /api/v1/images                (?q=&dangling=)
GET /api/v1/images/{id}
GET /api/v1/graph                 (?scope=all|stack&stack=name)
GET /api/v1/export                (?format=json|csv&scope=all|containers|stacks)
GET /api/v1/metrics/history       (?host=&scope=host|container&id=&from=&to=&step=)
GET /api/v1/system/df
GET /api/v1/system/resources
GET /api/v1/system/info
GET /api/v1/system/settings
GET /api/v1/system/diagnostics    (localhost only — support dump)
GET /api/v1/commands              (Command Registry definitions)
GET /api/v1/commands/{id}
GET /api/v1/entities/{kind}/commands (?ref=&shell=bash|powershell|cmd)
GET /api/v1/diagnostics           (anomaly findings)
GET /api/v1/diagnostics/{id}
GET /api/v1/provenance
GET /api/v1/provenance/{id}
GET /api/v1/snapshots
POST /api/v1/snapshots            ({"label":"..."})
GET /api/v1/snapshots/{id}
GET /api/v1/snapshots/{id}/diff   (?against=current|<id>)
DELETE /api/v1/snapshots/{id}
GET /api/v1/ws                    WebSocket hub
```

WebSocket types: `container.stats`, `docker.event`, `snapshot.updated`, `connection.status`, `events.status`.

Auth (ADR-013): `--auth-token` required when listen is not loopback. Protects `/api/v1/*` except `GET /api/v1/health`. WS: `Authorization: Bearer` or `?access_token=`. UI: Settings → token in `localStorage`.

Flags:

```text
--listen 127.0.0.1:8080
--auth-token <secret>              # required for 0.0.0.0 / LAN binds
--docker-host unix:///var/run/docker.sock
--docker-hosts name=url,name2=url  # multi-host (ADR-014); empty ⇒ single "default"
--docker-config C:\Users\you\.docker
--docker-timeout 5s
--inventory-interval 10s
--stats-interval 1s
--system-interval 15s
--metrics-db data/metrics.db       # SQLite history (ADR-015); off disables
--metrics-interval 10s
--metrics-retention 24h
--snapshots-dir data/snapshots     # inventory snapshots (ADR-016); off disables
```

Multi-host example:

```bash
go run ./cmd/docker-visualizer --docker-hosts "local=npipe:////./pipe/docker_engine,lab=tcp://192.168.1.10:2376"
```

UI Host picker appears when more than one host is configured. REST/WS take `?host=<name>` (omit → default).

## Tests

```bash
go test ./...
go test -tags=integration ./internal/docker/ -v
cd web && npm test

# Live Go ↔ PowerShell parity (needs Docker + pwsh/powershell)
go run ./cmd/parity-check -skip-stats
```
