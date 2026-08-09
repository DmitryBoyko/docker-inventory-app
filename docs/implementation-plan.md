# Docker Visualizer — Implementation Plan / Technical Design

**Status:** Architecture approved for implementation (no production code yet)  
**Date:** 2026-08-09  
**Source of truth (behavior):** `scripts/docker-stack-inventory.ps1`  
**Secondary parity reference:** `scripts/docker-stack-inventory.sh`  
**Prior proposals (non-authoritative):** `docs/idea.md`, `docs/correct-idea.md`

---

# 1. Executive Summary

Создаём **read-only**, **self-contained**, **cross-platform** утилиту `docker-visualizer`:

- один executable на платформу (Windows / Linux / macOS; amd64 + arm64);
- backend на **Go** + официальный **`github.com/moby/moby/client`**;
- React+TS UI, встраиваемый через `embed.FS`;
- данные из **Docker Engine API** (не через `docker` CLI);
- Compose **Stack** — derived entity из labels;
- REST для inventory/details; WebSocket только для live stats и Docker events.

Исходный PowerShell делает **ровно 4 batch-вызова** Docker CLI и агрегирует результат по `com.docker.compose.project`. Go-версия должна сначала достичь **functional parity** с этим выводом, затем расширить сущности (networks/volumes/images как first-class, graph, realtime).

Ключевые исправления относительно `docs/idea.md`:

1. Не хардкодить socket paths — discovery через config → `DOCKER_HOST` → Docker context → SDK defaults.
2. Volume size — nullable + `available`/`reason`; Docker `UsageData.Size == -1` ≠ `0`.
3. Не отдавать SDK types в API; domain + DTO.
4. WebSocket не для полного inventory.
5. Bind по умолчанию `127.0.0.1`; inspect/logs с redaction.
6. Актуальный SDK: `github.com/moby/moby/client` (не deprecated `github.com/docker/docker`).

---

# 2. What the Existing PowerShell Actually Does

Файл: `scripts/docker-stack-inventory.ps1` (~391 строка). Bash-аналог: `scripts/docker-stack-inventory.sh` (parity поведения).

## 2.1 Pipeline (без N+1)

| Step | Command | Purpose |
|------|---------|---------|
| 1 | `docker ps -a --size --format '{{json .}}'` | Все контейнеры + SizeRw (writable) + labels/ports/state |
| 2 | `docker stats --no-stream --format '{{json .}}'` | CPU/Mem/NetIO/BlockIO только для running |
| 3 | `docker inspect $(docker ps -aq)` | Batch inspect: mounts, networks, IPs, health, restarts |
| 4 | `docker system df -v` | Text parse секции VOLUME NAME / LINKS / SIZE |

Прогресс: `Write-Progress` на 5% / 30% / 55% / 75% / 90%.

## 2.2 Docker CLI operations inventory

### A. `docker ps -a --size --format '{{json .}}'`

| Aspect | Detail |
|--------|--------|
| Why | Inventory + writable layer size без per-container inspect size |
| Params | `-a` (all), `--size`, JSON format |
| Fields used | `ID`, `Names`, `Image`, `Labels`, `Ports`, `Size`, `Status`, `State`, `RunningFor` |
| Transform | Labels regex → stack/service; Size strip `(virtual …)`; Ports → Split-DockerPorts; State/Status → uptime/health fallback |
| Aggregation input | Primary row source for stack grouping |

### B. `docker stats --no-stream --format '{{json .}}'`

| Aspect | Detail |
|--------|--------|
| Why | Live resource snapshot |
| Params | `--no-stream` (one sample), JSON |
| Fields used | `ID`, `MemUsage`, `CPUPerc`, `NetIO`, `BlockIO` |
| Transform | MemUsage left of `/` → bytes; CPUPerc → float; NetIO/BlockIO kept as **display strings** |
| Edge | Stopped → Memory=`Stopped`, CPU=`-`, MemBytes=0, CPUVal=0; no stats entry |

### C. `docker ps -aq` + `docker inspect $ids`

| Aspect | Detail |
|--------|--------|
| Why | Networks, volume names, health, restart count (не в compact ps) |
| Lookup key | Short ID = `Id[0:12]` matched to `ps` `ID` |
| Fields used | `Mounts` (Type=`volume`, Name), `NetworkSettings.Networks` keys + `IPAddress`, `RestartCount`, `State.Health.Status` |
| Transform | Volumes list; nets joined; IPs as `"ip (network)"`; health or `-` |

### D. `docker system df -v` (text)

| Aspect | Detail |
|--------|--------|
| Why | Per-volume size + link count |
| Parse | After header `VOLUME NAME LINKS SIZE` until blank / `Build cache` |
| Regex | `^\s*(\S+)\s+(\d+)\s+(\S+)\s*$` |
| Transform | Human size → bytes via `Convert-DockerSizeToBytes` (SI + IEC) |
| Edge | Missing volume → display `?`, bytes treated as **0** in sums (parity bug to fix in Go) |

### E. Other CLI

Нет: `docker network`, `docker volume`, `docker image`, events, logs. Скрипт **не** first-class inventory для networks/volumes/images — только join через container mounts/networks.

## 2.3 Data extraction (entities today)

| Entity | Present? | How |
|--------|----------|-----|
| Containers | Yes | ps + stats + inspect |
| Images | Partial | name/tag string only (`Get-ShortImage`) |
| Networks | Partial | names + IPs from inspect |
| Volumes | Partial | names from mounts + size from df |
| Compose stacks | Yes | label `com.docker.compose.project`, default `standalone` |
| Services | Yes | label `com.docker.compose.service`, default `-` |
| CPU / Memory | Yes | stats |
| Network I/O / Block I/O | Yes | stats strings |
| Writable layer | Yes | SizeRw from `--size` |
| Volume usage | Yes | df -v |
| Restart count | Yes | inspect |
| Health | Yes | inspect + Status fallback |
| Uptime | Yes | RunningFor / Status for exited\|created |
| Ports | Yes | custom Split-DockerPorts |
| Engine info | No | — |

## 2.4 Aggregation logic (must preserve)

### Stack identity

```text
Labels string contains com.docker.compose.project=<name>  → Stack = name
else → Stack = "standalone"

Labels string contains com.docker.compose.service=<name> → Service = name
else → Service = "-"
```

Labels приходят из `ps --format` как comma-separated `k=v` (не map). Regex: `com\.docker\.compose\.project=([^,]+)`.

### Grouping

`Group-Object Stack | Sort-Object Name`

Per stack output:

1. Table A (sort MemBytes desc): Container, Service, Image, Health, State, Restarts, Uptime, CPU, Memory, Disk  
2. Table B (sort Container): External, Internal, IP, Networks, Volumes, NetIO, BlockIO  
3. Counters: containers, with volumes, unhealthy, restarts>0  
4. Stack RAM = Σ MemBytes where State==`running`  
5. Stack CPU = Σ CPUVal where State==`running`  
6. Stack Disk writable = Σ DiskBytes (all states)  
7. Top RAM = top 3 MemBytes>0  
8. Stack volumes = unique VolumeNames; size/links from map; sum bytes  

### Grand totals

- ALL containers count  
- ALL RAM / CPU (running only)  
- ALL writable layers (all)  
- ALL volume data (**unique** volume names across all containers)

### Port semantics (preserve UX)

- External (has `->`): group by `hostPort->dest`; annotate `*:… [наружу]` vs `127.0.0.1:… [localhost]` vs explicit IPs  
- Internal (no `->`): container-only ports  
- IPv6 hosts `[::]`, `[::1]` supported  

### Size parsing quirks (parity note)

`Convert-DockerSizeToBytes` accepts both decimal (`kB/MB/GB`) and binary (`KiB/MiB/GiB`). PowerShell `1KB`/`1MB` = binary (1024). Unparseable → **0** (same bug as missing volume).

### Anonymous volumes

64-char hex name → display `anon:{first12}...`

### Image display

`sha256:xxxxxxxxxxxx…` → `sha256:{12}...`

### Health fallback

If inspect health missing and Status matches `(healthy|unhealthy|health: starting|starting)`.

---

# 3. PowerShell → Docker Engine API Mapping

## 3.1 Containers list + size

```text
docker ps -a --size
        ↓
GET /containers/json?all=true&size=true
        ↓
client.ContainerList(ctx, options{All:true, Size:true})
        ↓
[]container.Summary  (ID, Names, Image, Labels map, Ports, SizeRw, SizeRootFs, State, Status, …)
        ↓
domain.Container
```

| Question | Answer |
|----------|--------|
| Full CLI replace? | **Yes** |
| Differences | API gives Labels as `map[string]string` (better than regex); SizeRw as int64 bytes (no string parse); Ports as structured `[]Port`; no `RunningFor` human string — compute from `StartedAt` via inspect or Status |
| Must compute | Human uptime string (optional); port display labels; stack/service from Labels map |
| Platform dependent | SizeRw accuracy depends on storage driver |

## 3.2 Stats

```text
docker stats --no-stream
        ↓
GET /containers/{id}/stats?stream=false   (per running container)
  OR multiplexed collector with stream=true once
        ↓
client.ContainerStats / ContainerStatsOneShot
        ↓
types.StatsJSON
        ↓
domain.ContainerStats
```

| Question | Answer |
|----------|--------|
| Full CLI replace? | **Yes**, but CPU%/NetIO/BlockIO are **computed** from raw counters (CLI already formats them) |
| Differences | API returns cumulative counters + previous sample needed for CPU%; CLI returns preformatted strings |
| Must compute | `cpuPercent`, memory usage/limit/percent, Σ networks rx/tx, Σ blkio read/write |
| Platform dependent | Windows stats fields differ (use platform-aware formulas; Moby docs / Docker CLI source) |

**Parity rule:** CPU formula must match Docker CLI (`calculateCPUPercentUnix` / Windows equivalent). Memory usage = `usage - cache` on cgroup v1 semantics as CLI does (verify against current moby calculator).

## 3.3 Inspect

```text
docker inspect <ids>
        ↓
GET /containers/{id}/json   (batch in parallel with worker pool; or one-by-one)
        ↓
client.ContainerInspect
        ↓
container.InspectResponse
        ↓
domain.Container (enrich) + raw inspect DTO (redacted)
```

| Question | Answer |
|----------|--------|
| Full CLI replace? | **Yes** |
| Differences | Full ID always; no short-ID join fragility |
| Sensitive | Env, Labels may contain secrets → redaction policy on public inspect endpoint |

**Optimization vs PS:** PS inspects all containers every run. Go should cache inspect metadata and invalidate on events; list endpoint can use Summary + cached enrich.

## 3.4 Volume sizes

```text
docker system df -v   (text parse)
        ↓
GET /system/df
        ↓
client.DiskUsage
        ↓
[]*volume.Volume with UsageData.{Size, RefCount}
        ↓
domain.VolumeUsage { Bytes *int64, Available bool, Reason string }
```

| Question | Answer |
|----------|--------|
| Full CLI replace? | **Yes — and better** (structured JSON, no text parse) |
| Differences | `Size == -1` means unavailable (local driver only otherwise); RefCount similarly |
| Must NOT | Treat `-1` or missing as `0` in aggregates |
| Platform dependent | Non-local drivers, some Desktop mounts, cluster volumes |

## 3.5 Networks / Volumes / Images (extension beyond PS)

| CLI (not in PS) | API | Go client | Domain |
|-----------------|-----|-----------|--------|
| `docker network ls/inspect` | `GET /networks`, `GET /networks/{id}` | NetworkList/Inspect | Network |
| `docker volume ls/inspect` | `GET /volumes`, `GET /volumes/{name}` | VolumeList/Inspect | Volume |
| `docker images` | `GET /images/json` | ImageList | Image |
| `docker info` | `GET /info` | Info | SystemInfo |
| `docker events` | `GET /events` | Events | cache invalidation |
| `docker logs` | `GET /containers/{id}/logs` | ContainerLogs | Logs DTO |

## 3.6 Field-level mapping (PS row → domain)

| PS field | Domain | Source |
|----------|--------|--------|
| Stack | `stack` | Labels[`com.docker.compose.project`] or `standalone` |
| Container | `name` | Names[0] trim `/` |
| Service | `service` | Labels[`com.docker.compose.service`] or null/`-` |
| Image | `image` | Image / ImageID |
| External/Internal | `ports` | Port bindings structured + display helpers |
| IP | `endpoints[].ipAddress` | Inspect Networks |
| Networks | `networks[]` | Inspect |
| Volumes | `mounts` / volume refs | Inspect Mounts Type=volume |
| CPU | `stats.cpuPercent` | Stats calc |
| Memory / MemBytes | `stats.memory.usageBytes` | Stats |
| NetIO | `stats.network.{rx,tx}Bytes` | Stats (structured; display optional) |
| BlockIO | `stats.block.{read,write}Bytes` | Stats |
| Disk / DiskBytes | `writableLayerBytes` | SizeRw |
| Health | `health` | Inspect State.Health / Status parse |
| Restarts | `restartCount` | Inspect |
| Uptime | `uptime` / `startedAt` | Prefer RFC3339 `startedAt` + computed duration |
| State | `state` | Summary.State |

---

# 4. Missing / Impossible / Platform-Dependent Data

## Docker Engine API limitations

| Field | Source | Availability | Calc required? | Platform dependent? | Fallback |
|-------|--------|--------------|----------------|---------------------|----------|
| Writable layer (SizeRw) | ContainerList `size=true` | Usually yes | No | Storage driver | `available:false` if omitted |
| RootFS size | SizeRootFs | Usually yes | No | Same | optional field |
| Volume size | `/system/df` UsageData.Size | **local driver only**; `-1` otherwise | No | Driver/OS/Desktop | `{bytes:null, available:false, reason:"unsupported"}` |
| Volume RefCount | UsageData.RefCount | Often; `-1` if N/A | No | — | unknown |
| CPU % | Stats | Running only | **Yes** (delta) | Linux vs Windows formulas | null if not running |
| Memory usage | Stats | Running only | Yes (cache subtract rules) | cgroup v1/v2 / Windows | null if not running |
| Memory limit | Stats / HostConfig | Yes | No | May be host total if unlimited | expose limit + percent |
| Network I/O | Stats.Networks | Running; per-iface | Sum | Interface names vary | 0 only if truly zero counters |
| Block I/O | Stats.BlkioStats / StorageStats | Running | Sum read/write ops | Driver; Windows different | availability flag |
| Health | Inspect | Only if HEALTHCHECK defined | No | — | `none` vs unknown |
| Human RunningFor | CLI only | N/A in API | Yes from StartedAt | — | compute |
| Compose project | Labels | If Compose-created | No | Swarm stack labels differ (`com.docker.stack.namespace`) | standalone / detect swarm |
| Image shared size | `/system/df` | Partial | No | — | expose as optional |
| Per-volume growth history | None | No | External TSDB | — | V2 feature |
| Exact physical disk for bind mounts | None | No | Host FS (out of scope / privileged) | Yes | unsupported |

### Volume size policy (mandatory)

```json
{
  "usage": {
    "bytes": null,
    "available": false,
    "reason": "unsupported",
    "links": null
  }
}
```

Reasons enum: `unsupported` | `not_local_driver` | `daemon_omitted` | `collect_error` | `pending`.

Aggregates (`totalVolumeBytes`) must expose:

```json
{
  "bytes": 123,
  "available": true,
  "partial": false,
  "unknownCount": 0
}
```

If any volume unknown → `partial:true`, never silently treat as 0.

### Swarm note

PowerShell ignores Swarm. V1 may map `com.docker.stack.namespace` → stack name as optional alias; MVP = Compose labels only + `standalone`.

---

# 5. Recommended Architecture

Pragmatic layered design (not enterprise theater):

```text
Docker Engine API
      ↓
moby/moby/client
      ↓
internal/docker          (adapter: connection, raw calls)
      ↓
internal/mapper          (SDK → domain)
      ↓
internal/domain          (models + pure aggregation)
      ↓
internal/collector       (inventory / stats / system / events)
      ↓
internal/store           (in-memory snapshot cache)
      ↓
internal/app             (use cases)
      ↓
internal/httpapi + ws    (DTO / transport)
      ↓
embed React SPA
```

Principles:

- **One in-memory Snapshot** as source of truth for REST reads.
- Collectors write Snapshot under RWMutex (or immutable swap).
- Handlers never call Docker directly except logs/inspect-on-demand (optional).
- Aggregation is pure functions over Snapshot (unit-testable).
- Prefer stdlib `net/http` + small router (chi/echo optional); avoid heavy frameworks.

---

# 6. Domain Model

Units: bytes = int64; percents = float64 0..N*100 for CPU (Docker CLI style, can exceed 100 on multi-core); timestamps = UTC RFC3339.

### Nullable / availability pattern

```go
type ByteMetric struct {
    Bytes     *int64 `json:"bytes"`               // null if unavailable
    Available bool   `json:"available"`
    Reason    string `json:"reason,omitempty"`    // enum
}
```

### Container

| Field | Type | Units | Nullable | Semantics | Source |
|-------|------|-------|----------|-----------|--------|
| ID | string | — | no | full ID | List/Inspect |
| IDShort | string | — | no | 12 chars | derived |
| Name | string | — | no | without leading `/` | Names |
| Stack | string | — | no | project or `standalone` | label |
| Service | *string | — | yes | compose service | label |
| ContainerNumber | *int | — | yes | compose replica index | label |
| Image | string | — | no | ref as shown by Docker | List |
| ImageID | string | — | no | | List |
| State | enum | — | no | created/running/paused/restarting/removing/exited/dead | List |
| Status | string | — | no | raw Status text | List |
| Health | enum | — | no | none/starting/healthy/unhealthy/unknown | Inspect+fallback |
| RestartCount | int | — | no | | Inspect |
| StartedAt | *time | — | yes | | Inspect |
| FinishedAt | *time | — | yes | | Inspect |
| UptimeSeconds | *int64 | s | yes | running only | computed |
| Ports | []Port | — | no | | List |
| Endpoints | []NetworkEndpoint | — | no | | Inspect |
| Mounts | []Mount | — | no | | Inspect |
| WritableLayer | ByteMetric | B | | SizeRw | List size=true |
| Stats | *ContainerStats | — | yes | nil if not running / no sample | Stats |
| Labels | map | — | no | filtered public subset | List |

### ContainerStats

| Field | Type | Units | Semantics |
|-------|------|-------|-----------|
| Timestamp | time | UTC | sample time |
| CPUPercent | float64 | % | Docker CLI compatible |
| MemoryBytes | int64 | B | usage (CLI-compatible) |
| MemoryLimitBytes | int64 | B | limit |
| MemoryPercent | float64 | % | usage/limit*100 |
| Network.RxBytes / TxBytes | int64 | B | cumulative since container start |
| Block.ReadBytes / WriteBytes | int64 | B | cumulative |
| RawAvailable | bool | | false if platform missing counters |

### Stack

| Field | Type | Semantics |
|-------|------|-----------|
| Name | string | compose project or `standalone` |
| Containers | []ContainerRef | |
| Services | []Service | grouped by service name |
| Resources | ResourceSummary | |
| VolumeNames | []string | unique owned/used |
| VolumeUsage | AggregateBytes | unique volumes with partial flag |
| UnhealthyCount | int | health contains unhealthy |
| RestartedCount | int | restartCount > 0 (parity with PS `restarts>0`) |
| TopRAM | []TopConsumer | top 3 |

### Service

Name, Stack, ContainerRefs[], ResourceSummary (optional V1).

### Network / NetworkEndpoint / Volume / VolumeUsage / Image / SystemInfo / SystemResources / Port / Mount / Health / ResourceSummary

Design as in Phase 4 prompt; critical:

- **ResourceSummary:** `CPUPercent`, `MemoryBytes` (running only), `WritableLayer` AggregateBytes, `VolumeData` AggregateBytes, `ContainerCount`, `RunningCount`.
- **Port:** HostIP, HostPort, ContainerPort, Protocol, PublishMode; plus `Exposure` enum: `public|localhost|specific|internal`.
- **Mount:** Type (volume/bind/tmpfs/npipe), Name/Source, Destination, RW.
- **Volume.Usage:** ByteMetric + Links *int64.
- **SystemResources:** host-level from Info + sum of running container stats (document that Docker doesn't give host CPU% via Engine API alone — container sum is what PS does).

Do **not** export moby types from `internal/httpapi`.

---

# 7. Docker Connection Strategy

## Discovery order

```text
1. Explicit flag/env: --docker-host / DOCKER_VISUALIZER_DOCKER_HOST
2. DOCKER_HOST (standard)
3. Docker context current endpoint (parse ~/.docker/config.json + contexts) — recommended for Desktop
4. client.FromEnv / SDK platform defaults
5. Fail with actionable error
```

**Do not** hardcode only `/var/run/docker.sock`.

| Platform | Typical endpoints |
|----------|-------------------|
| Linux Engine | `unix:///var/run/docker.sock` |
| Docker Desktop Linux | `unix://~/.docker/desktop/docker.sock` (via context) |
| macOS Desktop | `unix://~/.docker/run/docker.sock` (± symlink `/var/run/docker.sock`) |
| Windows Desktop | `npipe:////./pipe/docker_engine` |
| Remote | `tcp://host:2376` + TLS; optional `ssh://` if SDK/transport supports |

## Connection behavior

| Concern | Policy |
|---------|--------|
| API version | Negotiate (`WithAPIVersionNegotiation`) |
| Dial timeout | 5s configurable |
| Ping on startup | `GET /_ping` + `Info` → ready state |
| Retry | Exponential backoff on collector errors; process stays up |
| Reconnect | Events stream reconnect with backoff; mark `docker.connected=false` |
| Errors | Distinguish: socket not found, permission denied, daemon down, TLS, context missing |
| CGO | **CGO_ENABLED=0** for release binaries |

Use `client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())` as baseline; wrap with context-aware host resolution helper.

---

# 8. Data Collection & Caching Strategy

## Anti-pattern (forbidden)

```text
N containers × 1s × (inspect + stats)
```

## Collectors

| Collector | Frequency (default) | Work | Trigger refresh early |
|-----------|---------------------|------|------------------------|
| Inventory | 10s | ContainerList(size), NetworkList, VolumeList, ImageList; enrich from cache; selective Inspect for new/changed IDs | Events |
| Stats | 1s | StatsOneShot for **running** IDs only; concurrency limit (e.g. 16) | — |
| System | 15s | Info + DiskUsage (`/system/df`) | Events volume/image |
| Events | stream | invalidate + coalesce refresh (debounce 250–500ms) | — |

Trade-offs:

- Inventory 5s → fresher UI, more daemon load; 15s → cheaper. **10s default**, configurable.
- Stats 1s matches interactive dashboards; for 500+ containers raise to 2–5s or sample subset (subscribed only).
- `/system/df` can be expensive → 15s + event invalidation; never on every HTTP request.

## Cache / Store

```text
Snapshot {
  CollectedAt
  Containers map[id]*Container
  Stacks map[name]*Stack
  Networks, Volumes, Images
  SystemInfo, DiskUsage
  Totals ResourceSummary
  DockerConnected bool
  Errors []CollectorError
}
```

- Atomic pointer swap after each successful collector merge.
- Readers: `Load()` immutable view.
- Stale: REST returns `X-Snapshot-Age` / body `snapshotAgeMs`; UI shows banner if age > threshold.
- Logs & full inspect: **not cached long**; on-demand with short TTL (e.g. 5s) optional.

---

# 9. Compose Stack Aggregation

## Canonical labels (Compose V2)

| Label | Use |
|-------|-----|
| `com.docker.compose.project` | Stack name |
| `com.docker.compose.service` | Service name |
| `com.docker.compose.container-number` | Replica ordinal |
| `com.docker.compose.project.working_dir` | Optional metadata |
| `com.docker.compose.project.config_files` | Optional metadata |
| `com.docker.compose.version` | Optional |

## Deterministic rules

| Case | Behavior |
|------|----------|
| No compose labels | Stack=`standalone`, Service=null |
| Project only | Stack=project, Service=null |
| Same service name in different projects | Distinct: key `(project, service)` |
| Orphans (project label but no peers) | Still belong to that stack |
| Multiple projects | Separate Stack entities; sort by name |
| Volume “ownership” | Volume used by containers of stack S; if shared across stacks → list all stacks, `shared:true` |
| Network ownership | Same as volumes via endpoints |

Parity with PS: default stack name string **`standalone`**, default service display **`-`** in UI (API may use `null`).

Aggregation formulas — **identical to §2.4**, with volume totals using AggregateBytes partial semantics (improvement over PS).

---

# 10. REST API

Base: `/api/v1`  
Envelope (recommended):

```json
{
  "timestamp": "2026-08-09T10:40:00Z",
  "snapshotAt": "2026-08-09T10:39:58Z",
  "data": {}
}
```

Errors:

```json
{
  "error": {
    "code": "not_found",
    "message": "container not found",
    "details": {}
  },
  "timestamp": "..."
}
```

Statuses: 400 validation, 404 missing, 503 docker disconnected / not ready, 500 unexpected.

### Endpoints

| Method | Path | Purpose | Expensive? | Cache |
|--------|------|---------|------------|-------|
| GET | `/containers` | List (+ query stack, state, health, q) | No | Snapshot |
| GET | `/containers/{id}` | Detail | No | Snapshot |
| GET | `/containers/{id}/stats` | One stats sample from store | No | last stats |
| GET | `/containers/{id}/logs` | Snapshot logs `?tail=&since=&timestamps=` | Medium | no / 2s |
| GET | `/containers/{id}/inspect` | Inspect JSON `?redact=true` default | Medium | short TTL |
| GET | `/stacks` | List stacks + ResourceSummary | No | Snapshot |
| GET | `/stacks/{name}` | Detail + services + topRam | No | Snapshot |
| GET | `/stacks/{name}/volumes` | Volumes linked to stack | No | Snapshot |
| GET | `/networks` | List | No | Snapshot |
| GET | `/networks/{id}` | Detail (id or name) | No | Snapshot |
| GET | `/volumes` | List + usage availability | No | Snapshot |
| GET | `/volumes/{name}` | Detail | No | Snapshot |
| GET | `/images` | List | No | Snapshot |
| GET | `/images/{id}` | Detail | No | Snapshot |
| GET | `/system/info` | Engine info | No | Snapshot |
| GET | `/system/df` | Disk usage normalized | From collector | Snapshot |
| GET | `/system/resources` | Aggregated totals (PS grand totals) | No | Snapshot |
| GET | `/graph` | Normalized nodes/edges | CPU light | Snapshot |
| GET | `/health` | Process alive | No | — |
| GET | `/ready` | Docker ping OK + initial snapshot | No | — |

**Drop from original proposal as redundant:** separate verbose inspect always embedded in ContainerDetail.

**Metrics naming (fix OpenAPI):** `cpuPercent`, `memoryBytes`, `memoryLimitBytes`, `memoryPercent`, `writableLayerBytes`, `network.rxBytes`, `network.txBytes`, `block.readBytes`, `block.writeBytes` — never ambiguous `cpu`/`diskBytes` without units docs.

---

# 11. WebSocket API

Path: `GET /api/v1/ws` (single hub).

### Use WS for

- `container.stats` (filtered)
- `docker.event` (normalized)
- `snapshot.updated` (lightweight notice: version + age — **not** full dump)
- `connection.status`

### Do not use WS for

Full inventory every second; images/networks dumps; logs streaming in MVP (V1 optional).

### Envelope

```json
{
  "type": "container.stats",
  "timestamp": "2026-08-09T10:40:00Z",
  "data": {}
}
```

### Client protocol

```json
{ "action": "subscribe", "channel": "stats", "filters": { "stack": "prod", "containerIds": [] } }
{ "action": "unsubscribe", "channel": "stats" }
{ "action": "ping" }
```

### Server policies

| Topic | Policy |
|-------|--------|
| Heartbeat | server ping every 15s |
| Backpressure | per-client buffer 8–32; drop oldest stats; never block collectors |
| Slow clients | disconnect after N drops with `policy_violation` |
| Update interval | align with Stats collector (1s); coalesced |
| Shutdown | close with code 1001; collectors stop |
| Concurrency | Hub goroutine + per-conn read/write pumps |

---

# 12. Event Architecture

Use Docker Events API (`GET /events`).

Subscribe types (MVP):

- container: start, stop, die, destroy, pause, unpause, restart, health_status, rename  
- network: connect, disconnect, create, destroy  
- volume: create, destroy  
- image: delete, tag (optional)

Flow:

```text
Docker Events → coalesce (250ms) → Inventory/System invalidate
             → Snapshot bump → WS snapshot.updated + docker.event
```

Stats collector always loops on current running set from Snapshot (no need to event every CPU tick).

If events stream dies → fallback to pure polling; surface `eventsConnected:false`.

---

# 13. React Architecture

Stack: React 18+ / TypeScript / Vite.  
Server state: **TanStack Query** for REST.  
WS: small store (Zustand or React context) for live stats overlay.  
Router: React Router.  
Tables: TanStack Table + virtualization (`@tanstack/react-virtual`) for 100+ rows.  
Charts: ECharts.  
Graph: Cytoscape.js.  
UI: pragmatic CSS variables + light component set (avoid shipping half of MUI unless needed). Theme: dark/light.

### Routes

```text
/                 Dashboard
/containers       List
/containers/:id   Detail (tabs: overview, ports, networks, volumes, stats, logs, inspect)
/stacks           List
/stacks/:name     Detail + graph
/networks
/networks/:id
/volumes
/volumes/:name
/images
/system
/settings
```

### States

Loading / empty / error / docker-disconnected / stale-snapshot banners on all inventory pages.

### Settings

Bind address display, refresh intervals (read-only from server config in MVP), theme, redact default, WS reconnect.

Parity UI must show the same columns as PS tables (two conceptual tables → one rich table with column toggle).

---

# 14. Graph Architecture

Backend `GET /api/v1/graph?scope=stack|all&stack=name` returns renderer-agnostic model:

```json
{
  "nodes": [
    { "id": "stack:prod", "type": "stack", "label": "prod", "data": { "health": "degraded" } },
    { "id": "service:prod:web", "type": "service", "label": "web", "data": {} },
    { "id": "container:abc...", "type": "container", "label": "webapp", "data": { "state": "running", "cpuPercent": 2.1 } },
    { "id": "network:frontend", "type": "network", "label": "frontend", "data": {} },
    { "id": "volume:webapp-data", "type": "volume", "label": "webapp-data", "data": { "usage": {} } }
  ],
  "edges": [
    { "id": "...", "type": "contains", "source": "stack:prod", "target": "service:prod:web" },
    { "id": "...", "type": "runs", "source": "service:prod:web", "target": "container:abc" },
    { "id": "...", "type": "attached", "source": "container:abc", "target": "network:frontend" },
    { "id": "...", "type": "mounts", "source": "container:abc", "target": "volume:webapp-data" }
  ]
}
```

Node types: `stack|service|container|network|volume|image`  
Edge types: `contains|runs|attached|mounts|uses_image`

IDs namespaced to avoid collisions. Frontend Cytoscape maps type→style only.

---

# 15. Security Model

| Control | Default |
|---------|---------|
| Mode | **Read-only** (no start/stop/rm/pull APIs in MVP/V1) |
| Bind | `127.0.0.1:8080` |
| CORS | same-origin / localhost only unless configured |
| Auth | none in MVP; V1 optional token for non-localhost bind |
| Docker socket | user must already have access; document least privilege; prefer `:ro` mount in container docs (note: Docker socket RO mount is imperfect isolation) |
| Inspect | `redact=true` by default: strip `Env`, known secret label keys, RegistryAuth |
| Logs | warning in UI; no persistence; optional redact patterns later |
| Remote DOCKER_HOST | allowlist / explicit opt-in; TLS required for tcp |
| SSRF | do not fetch arbitrary user URLs; only configured Docker host |

---

# 16. Observability

Keep lightweight (stdlib / slog):

- Structured JSON logs: level, msg, request_id, component  
- Request ID middleware  
- Internal counters (expvar or Prometheus optional behind `--metrics`):  
  - docker_connected  
  - snapshot_age_seconds  
  - collector_errors_total  
  - collector_duration_seconds  
  - ws_clients  
  - stats_dropped_total  
- `GET /api/v1/system/diagnostics` (localhost only) for support  

No ELK/Jaeger in-process.

---

# 17. Testing Strategy

| Layer | What |
|-------|------|
| Unit | Compose detection; port exposure classification; CPU/mem formulas (golden vectors from moby); stack aggregation; AggregateBytes partial; graph build; size mapping `-1`→unavailable |
| Mapper | SDK fixtures → domain JSON golden |
| API | httptest against fake Snapshot store |
| WS | subscribe/unsubscribe/heartbeat/backpressure |
| Integration | `DOCKER_HOST` testengine / CI service container; skip if no docker |
| Cross-platform | build matrix + smoke `--ready` on Win/Linux/macOS |
| Parity | see §19 |

---

# 18. Cross-Platform Strategy

| Target | Build | Runtime Docker |
|--------|-------|----------------|
| windows/amd64 | CGO=0 | named pipe / context |
| linux/amd64, arm64 | CGO=0 | socket / Desktop socket |
| darwin/amd64, arm64 | CGO=0 | Desktop user socket |

Container image: distroless/static or scratch + CA certs if tcp/tls; mount docker.sock; run as user in docker group documented.

CI: `go test ./...` + `GOOS/GOARCH` build matrix; optional integration job with Docker-in-Docker.

---

# 19. Single-Binary Build Strategy

```text
web/ (Vite) → npm ci && npm run build → web/dist
go:embed all:web/dist
go build -trimpath -ldflags "-s -w -X main.version=... -X main.commit=..."
```

Artifacts:

```text
docker-visualizer-windows-amd64.exe
docker-visualizer-linux-amd64
docker-visualizer-linux-arm64
docker-visualizer-darwin-amd64
docker-visualizer-darwin-arm64
(+ SHA256SUMS)
```

Makefile / goreleaser. Reproducible: trimpath, pinned module versions, Node LTS pin. CGO disabled.

SPA fallback: all non-API routes → `index.html`.

---

# 20. Repository Structure

```text
docker-visualizer/
├── cmd/docker-visualizer/
│   └── main.go                 # wire config, docker, collectors, http
├── internal/
│   ├── config/                 # flags/env
│   ├── docker/                 # client factory, endpoint discovery, ping
│   ├── mapper/                 # SDK → domain
│   ├── domain/                 # models + aggregation + graph + compose
│   ├── collector/              # inventory, stats, system, events
│   ├── store/                  # snapshot
│   ├── app/                    # use cases (thin)
│   ├── httpapi/                # REST handlers, middleware, DTO
│   ├── ws/                     # hub
│   ├── redact/                 # inspect/env redaction
│   └── observability/          # slog, metrics
├── web/                        # React app (Vite)
├── scripts/                    # existing PS/SH inventory + parity helpers
├── docs/                       # architecture, ADRs
├── testdata/                   # golden docker fixtures
├── openapi.yaml
├── Makefile
├── .goreleaser.yaml
├── go.mod
└── README.md
```

**Changes vs idea.md:** merge `models` into `domain`; add `collector`/`store`/`mapper`/`redact`; drop unused `pkg/openapi` codegen unless needed; no `utils` grab-bag — put helpers next to use.

Responsibilities: transport (`httpapi`/`ws`) → app → store/domain → collector/docker.

---

# 21. Implementation Roadmap

### Phase 0 — Architecture freeze

- **Objective:** ADRs + OpenAPI skeleton + domain types accepted  
- **Modules:** `docs/adr/*`, `openapi.yaml`, `internal/domain` stubs  
- **Tasks:** lock metrics semantics; volume availability; connection discovery  
- **Tests:** none yet  
- **Acceptance:** this document + ADR-001..010 merged  
- **Risks:** scope creep into mutations  

### Phase 1 — Docker connection

- **Objective:** connect + ping + ready on Win/Linux/macOS  
- **Modules:** `internal/docker`, `internal/config`, `cmd/...`  
- **Deps:** Phase 0  
- **Tasks:** discovery order; error messages; `--docker-host`  
- **Tests:** unit discovery parsing; integration ping  
- **Acceptance:** `/ready` 200 with running Docker; clear error without  
- **Risks:** Desktop context paths  

### Phase 2 — Container inventory parity core

- **Objective:** list containers with stack/service/ports/state/health/restarts/writable layer  
- **Modules:** mapper, collector/inventory, store, domain/compose  
- **Deps:** Phase 1  
- **Tasks:** ContainerList size=true; inspect enrich; health fallback; port exposure  
- **Tests:** golden labels/ports; mapper fixtures  
- **Acceptance:** matches PS columns except live stats  
- **Risks:** short vs full ID; Status parsing  

### Phase 3 — Stats

- **Objective:** CLI-compatible CPU/mem/net/block  
- **Modules:** collector/stats, domain stats calc  
- **Deps:** Phase 2  
- **Tasks:** one-shot stats; formulas; running-only  
- **Tests:** golden CPU vectors  
- **Acceptance:** within tolerance vs `docker stats --no-stream`  
- **Risks:** Windows formulas; cgroup v2  

### Phase 4 — Volumes/DF + stack aggregation

- **Objective:** stack resources + volume usage availability  
- **Modules:** system collector, aggregation  
- **Deps:** Phase 2–3  
- **Tasks:** DiskUsage map; AggregateBytes; top RAM; grand totals  
- **Tests:** aggregation fixtures including unknown sizes  
- **Acceptance:** stack totals match PS when all sizes available  
- **Risks:** df cost; `-1` handling  

### Phase 5 — REST API

- **Objective:** serve Snapshot via `/api/v1`  
- **Modules:** httpapi, openapi.yaml  
- **Deps:** Phase 4  
- **Tasks:** endpoints list/detail/system/health/ready; error model  
- **Tests:** httptest  
- **Acceptance:** OpenAPI examples validated  
- **Risks:** over-fetch inspect  

### Phase 6 — React shell + containers UI

- **Objective:** usable UI for inventory  
- **Modules:** `web/`  
- **Deps:** Phase 5  
- **Tasks:** layout, containers table, stacks list, dashboard totals  
- **Tests:** component smoke  
- **Acceptance:** replaces daily use of PS script for viewing  
- **Risks:** table density on phone — primary is desktop ops tool (OK)  

### Phase 7 — Realtime

- **Objective:** WS stats + events invalidation  
- **Modules:** ws, events collector  
- **Deps:** Phase 3,5,6  
- **Tasks:** hub, subscribe, live CPU/RAM charts  
- **Tests:** ws tests  
- **Acceptance:** UI updates without full REST poll storm  
- **Risks:** backpressure  

### Phase 8 — Networks / Volumes / Images pages

- **Objective:** first-class entities beyond PS  
- **Deps:** Phase 5–6  
- **Acceptance:** browse + link to containers  

### Phase 9 — Graph

- **Objective:** `/graph` + Cytoscape stack view  
- **Deps:** Phase 8  
- **Acceptance:** stack→service→container→net/vol navigable  

### Phase 10 — Logs + redacted inspect

- **Objective:** detail tabs  
- **Security:** redact default on  
- **Acceptance:** secrets not shown by default  

### Phase 11 — Packaging

- **Objective:** embed + multi-platform release  
- **Acceptance:** single exe serves UI+API  

### Phase 12 — Hardening ✅

- **Objective:** diagnostics, timeouts, load test 200 containers, docs  
- **Acceptance:** risk mitigations verified — see [`docs/hardening.md`](hardening.md)  
- **Delivered:** `/api/v1/system/diagnostics` (localhost), collector health registry, HTTP header/idle limits + middleware, synthetic 200-container load/bench, ops checklist  


---

# 22. MVP

Must be useful as daily replacement for the PowerShell script:

1. Docker connect (discovery)  
2. Container inventory with Compose stack grouping  
3. Stats (CPU/RAM/IO)  
4. Writable layer + volume usage (with availability)  
5. Stack aggregates + grand totals (parity)  
6. REST + minimal React: Dashboard, Containers, Stacks  
7. Single binary with embedded UI  
8. Read-only, bind 127.0.0.1  

Out of MVP: graph polish, image GC insights, auth, log streaming, mutations, history DB.

---

# 23. V1

- Networks / Volumes / Images full pages  
- WebSocket live stats + event-driven refresh  
- Graph visualization  
- Logs + redacted inspect  
- Settings (intervals, theme) ✅  
- OpenAPI published  
- goreleaser artifacts + checksums  
- Parity test harness vs PS/SH  

---

# 24. V2

- Optional auth when binding non-localhost ✅ (ADR-013, Settings UI token)  
- Settings (intervals display, theme, redact default) ✅  
- Multi-host (multiple Docker endpoints)  
- Historical metrics (SQLite)  
- Log stream WS  
- Swarm stack labels  
- Export JSON/CSV (structured PS replacement)  
- Optional safe mutations behind explicit `--enable-mutate`  
- Mobile-responsive polish  

---

# 25. ADRs

### ADR-001 Go

- **Context:** Need long-running cross-platform daemon + embed UI  
- **Decision:** Go  
- **Alternatives:** Rust, Node, Python, PowerShell 7  
- **Why:** Best practical combo of Docker SDK, cross-compile, concurrency, single binary  
- **Consequences:** Team needs Go familiarity  

### ADR-002 Docker Engine API (not CLI)

- **Context:** PS uses CLI; CLI parsing fragile  
- **Decision:** Engine API only at runtime  
- **Alternatives:** exec docker  
- **Why:** Structured bytes, no docker binary dependency  
- **Consequences:** Must reimplement CLI formatting math for parity  

### ADR-003 Moby client module

- **Context:** `github.com/docker/docker` deprecated (Docker v29+)  
- **Decision:** `github.com/moby/moby/client` (+ `api` types)  
- **Alternatives:** docker/go-sdk high-level, raw HTTP  
- **Why:** Official low-level Engine client  
- **Consequences:** Track v29+ breaking renames  

### ADR-004 React embed in Go

- **Context:** Distribution simplicity  
- **Decision:** Vite build + `embed.FS`  
- **Alternatives:** separate UI deploy, Wails, Tauri  
- **Why:** One artifact; browser UI enough  
- **Consequences:** Release pipeline needs Node  

### ADR-005 REST + limited WebSocket

- **Context:** idea.md overused WS  
- **Decision:** REST inventory; WS stats/events  
- **Alternatives:** WS-everything, SSE-only  
- **Why:** Cacheable reads; less bandwidth  
- **Consequences:** Two transports to test  

### ADR-006 Docker events

- **Context:** Polling alone is laggy/expensive  
- **Decision:** Events invalidate inventory; stats still polled  
- **Alternatives:** poll only  
- **Why:** Balance freshness vs load  
- **Consequences:** Reconnect logic required  

### ADR-007 Domain ≠ SDK

- **Context:** API stability for UI  
- **Decision:** mapper → domain → DTO  
- **Alternatives:** expose SDK JSON  
- **Why:** Insulate from moby type churn; enforce availability semantics  
- **Consequences:** Mapping code to maintain  

### ADR-008 Stack from Compose labels

- **Context:** No Engine Stack entity for Compose  
- **Decision:** Derive Stack from labels; `standalone` bucket  
- **Alternatives:** parse compose files on disk  
- **Why:** Matches PS; works without compose file access  
- **Consequences:** Won't see undeployed compose services  

### ADR-009 Read-only by default

- **Context:** Socket is root-equivalent  
- **Decision:** No mutation APIs initially; localhost bind  
- **Alternatives:** full Portainer-like control  
- **Why:** Risk reduction  
- **Consequences:** Users still use docker CLI to change state  

### ADR-010 Endpoint discovery

- **Context:** Desktop sockets vary  
- **Decision:** config → DOCKER_HOST → context → SDK default  
- **Alternatives:** hardcode paths per GOOS  
- **Why:** Correct on Desktop Linux/macOS/Windows  
- **Consequences:** Need context file parser tests  

### ADR-011 Volume usage availability

- **Context:** Size not always known; PS treats missing as 0  
- **Decision:** ByteMetric availability; partial aggregates  
- **Alternatives:** always int64  
- **Why:** Honest UI  
- **Consequences:** Parity diffs vs PS totals when unknown  

### ADR-012 In-memory snapshot store

- **Context:** Avoid per-request Docker storms  
- **Decision:** Collectors → atomic Snapshot  
- **Alternatives:** request-time fetch  
- **Why:** Performance + consistent aggregates  
- **Consequences:** Stale window equal to interval  

---

# 26. Risk Register

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| API/CLI stats formula drift | Med | High | Golden tests vs `docker stats`; copy moby calculator comments |
| Docker Desktop socket/context mismatch | High | High | Context-aware discovery; docs troubleshooting |
| Windows named pipe permissions | Med | High | Clear errors; run as user with Docker access |
| macOS socket path variance | Med | Med | FromEnv + context; probe known paths only as last resort |
| Linux Desktop vs Engine confusion | Med | High | Prefer active context endpoint |
| Volume size `-1` / unsupported | High | Med | Availability model; UI badges |
| API version skew | Med | Med | Negotiation; min version documented |
| Large N containers (500+) | Med | High | Concurrency limits; stats sampling; virtualization |
| High-frequency stats CPU/network | Med | Med | Configurable interval; WS filters |
| Process memory growth | Low | Med | Immutable snapshot swap; no log retention |
| WS backpressure | Med | Med | Bounded queues; drop policy |
| Inspect secrets leak | High | High | Default redaction; warn in UI |
| Docker socket privilege | High | High | Localhost bind; read-only product stance |
| Stale cache after event miss | Med | Med | Polling fallback; age banner |
| Daemon disconnect mid-run | High | Med | `/ready`, reconnect, UI disconnected state |
| Text df parse abandoned for API surprises | Low | Low | Prefer `/system/df`; integration tests |
| Scope creep to Portainer | High | High | MVP/V1/V2 gates |

---

# 27. Acceptance Criteria

## Functional parity (MVP)

On the same Docker host, Go Snapshot vs PS script:

| Field | Match rule |
|-------|------------|
| Stack name | Exact |
| Service | Exact (`-` ↔ null normalized) |
| Container set | Same IDs/names |
| State | Exact |
| Health | Exact after same fallback rules |
| RestartCount | Exact |
| WritableLayer bytes | Exact (SizeRw) |
| Volume names set | Exact |
| Volume bytes | Exact when API available; **documented diff** when PS used `?→0` |
| Stack Mem/CPU sums | Within float tolerance (0.15% abs for CPU) same sample window |
| Ports exposure class | Same public/localhost classification |
| Grand unique volumes | Same set |

Timing-dependent (stats, uptime strings): compare with simultaneous sampling or tolerance; prefer comparing raw bytes/% over human strings.

## Product acceptance

- [ ] Single binary serves UI at `/` and API at `/api/v1`  
- [ ] Works on Windows + Linux with Docker Desktop/Engine  
- [ ] No docker CLI dependency at runtime  
- [ ] Default listen 127.0.0.1  
- [ ] No mutation endpoints  
- [ ] Volume unknown ≠ 0  
- [ ] `/ready` fails closed when Docker down  

---

# 28. Final Architecture Diagram

```text
                     ┌────────────────────────────────┐
                     │     React SPA (embed.FS)       │
                     │  Query (REST)  │  WS store     │
                     └───────────────┬────────────────┘
                                     │
                      REST /api/v1   │   WS /api/v1/ws
                                     │
                     ┌───────────────▼────────────────┐
                     │            Go Process          │
                     │  httpapi │ ws hub │ redact     │
                     │            │                   │
                     │            ▼                   │
                     │         app (usecases)         │
                     │            │                   │
                     │            ▼                   │
                     │     store.Snapshot (atomic)    │
                     │      ▲         ▲        ▲      │
                     │      │         │        │      │
                     │ inventory   stats    system    │
                     │ collector  collector collector │
                     │      ▲         ▲        ▲      │
                     │      └──── events ──────┘      │
                     │              │                 │
                     │         docker.adapter         │
                     │   discovery │ client │ ping    │
                     └──────────────┬─────────────────┘
                                    │
                     ┌──────────────▼─────────────────┐
                     │       Docker Engine API        │
                     │ containers networks volumes    │
                     │ images stats events system/df  │
                     └────────────────────────────────┘
```

Data flow for a typical view:

```text
UI → GET /stacks → Snapshot.Stacks (pre-aggregated)
UI → WS subscribe stats → hub fans out collector samples
Events → debounce → inventory refresh → snapshot.updated
```

---

# What I Would Change From The Original Proposal

## Correct (keep)

| Item | Why |
|------|-----|
| Go + Docker Engine API + React | Right product shape |
| Single binary via embed | Best distribution for a host utility |
| Stack as virtual Compose aggregation | Matches real PS behavior |
| Read-only + localhost default | Essential safety |
| Domain models ≠ SDK | Prevents API churn breakage |
| REST for inventory, WS for live | Scalable |
| Cytoscape + ECharts | Fit graph/metrics needs |
| Moby client (`moby/moby/client`) | Current official module |

## Questionable

| Item | Why | Better |
|------|-----|--------|
| Polling intervals fixed in prose without load testing | N-dependent | Defaults + adaptive/config |
| `GET .../stats/stream` per container WS path | Many sockets | One hub + filters |
| Material/Ant as assumed UI kit | Heavy | Start CSS + small primitives |
| `pkg/openapi` codegen early | Overhead | Hand DTO + OpenAPI as contract first |
| Human `uptime` as primary API field | Fragile parity | Prefer `startedAt` + `uptimeSeconds`; human optional |

## Wrong

| Item | Why |
|------|-----|
| “PowerShell is not cross-platform” as main reason | PS7 is; real reason = wrong runtime for this app |
| “Go is the only language for one binary” | Rust/C also can |
| Hardcoded `/var/run/docker.sock` / npipe only | Breaks Desktop Linux/macOS contexts |
| `sizeBytes: 0` when unknown | Lies; use availability |
| WebSocket for entire inventory | Wasteful and racy |
| Treating Stack as if Engine had Stack API | It doesn’t for Compose |
| Ambiguous OpenAPI `cpu`/`memory` without units/semantics | Will cause UI bugs |

## Unnecessary (MVP)

| Item | Why defer |
|------|-----------|
| Full Portainer-like mutations | Risk/scope |
| Historical growth charts | Needs storage |
| Multi-host federation | Complexity |
| Auto-generated clients in-repo | Premature |
| Heavy observability stack | Overkill for local tool |

## Missing from original proposal (add)

| Item | Why |
|------|-----|
| Collector/snapshot architecture | Prevents Docker API storms |
| Docker events invalidation | Freshness without heavy poll |
| Endpoint discovery including contexts | Desktop reality |
| Volume usage availability model | API truthfulness |
| Inspect/log redaction | Secret safety |
| `/ready` vs `/health` | Orchestration semantics |
| Graph DTO separate from Docker | Renderer independence |
| Parity harness vs PS/SH | Migration confidence |
| Explicit CPU formula parity tests | Hardest correctness risk |
| `standalone` stack semantics + shared volumes | Real-world compose |
| CGO=0 / goreleaser / checksums | Production releases |
| Backpressure rules for WS | Stability under load |

---

## How to use this document next

Recommended next user prompt:

> Реализуй Phase 0 и Phase 1 согласно `docs/implementation-plan.md`, не меняя архитектурные решения без явного объяснения и ADR-дополнения.

Phase 0 deliverables: ADR files, OpenAPI skeleton, domain type stubs.  
Phase 1 deliverables: working Docker client discovery + `/health` + `/ready`.

**Do not implement application features until Phase 0–1 acceptance is met.**
