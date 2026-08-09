Да — идея с **«командным реестром»** очень хорошая. Более того, я бы сделал её одной из ключевых фич приложения.

Но я бы разделил две вещи:

1. **Command Registry** — генерирует команды для CLI и объясняет, что именно они делают.
2. **Terminal integration** — опционально позволяет реально выполнить команду, но не через браузер напрямую.

Это даст тебе очень сильную концепцию: **UI для быстрого понимания + CLI для полного контроля**.

### Что я бы добавил

#### 1. Command Registry — обязательно

Для каждой сущности:

```text
Container
├── docker ps
├── docker inspect
├── docker stats
├── docker logs
├── docker top
├── docker exec
└── docker diff

Network
├── docker network inspect
└── docker network ls

Volume
├── docker volume inspect
└── docker system df -v

Image
├── docker image inspect
├── docker image history
└── docker image ls
```

Но не просто показывать текст команды.

Например, на странице контейнера:

> **CLI commands**

```bash
docker inspect my-container
```

```powershell
docker inspect my-container
```

Кнопки:

**Copy Bash** | **Copy PowerShell**

И рядом:

> Что делает команда
> Какие данные покажет
> Почему эта команда полезна

Это особенно хорошо подходит твоей идее "детализации, которой нет в UI".

---

## 2. Генерация команд должна учитывать Docker context

Это важный нюанс.

Если приложение подключено к remote Docker context, команда должна быть не просто:

```bash
docker inspect nginx
```

а, например:

```bash
docker --context production inspect nginx
```

Docker официально поддерживает `--context` для отдельных команд и `DOCKER_CONTEXT` для текущей shell-сессии. ([Docker Documentation][1])

Поэтому я бы сделал объект:

```ts
interface Command {
  id: string;
  title: string;
  description: string;

  bash?: string;
  powershell?: string;
  cmd?: string;

  risk: "safe" | "destructive" | "interactive";

  requiresTTY: boolean;
  requiresDockerCLI: boolean;

  category:
    | "inspect"
    | "logs"
    | "stats"
    | "network"
    | "storage"
    | "debug"
    | "lifecycle";
}
```

И command generator:

```text
Docker context
      +
Entity
      +
Command definition
      ↓
Command Generator
      ↓
Bash / PowerShell / CMD
```

---

# 3. А вот настоящий terminal я бы не делал первой версией

Ты правильно почувствовал проблему:

> "из браузера не получится"

**Сам браузер не может просто открыть локальный PowerShell/Bash и дать ему произвольную команду.**

Но есть хорошие варианты.

### Вариант A — лучший

**Copy command → пользователь вставляет в terminal.**

Простейший, безопасный и кроссплатформенный вариант.

### Вариант B — "Open in Terminal"

Можно сделать desktop integration:

```text
Docker Visualizer
       ↓
Open Terminal
       ↓
PowerShell / Windows Terminal / Terminal.app / xterm
```

Но это уже требует OS-specific integration.

### Вариант C — встроенный Terminal

Технически можно сделать:

```text
Browser
   ↓ WebSocket
Go backend
   ↓
PTY
   ↓
PowerShell / Bash
```

И получить что-то вроде:

```text
┌──────────────────────────────────────┐
│ Docker Visualizer                    │
│                                      │
│  Containers                          │
│  Networks                            │
│  Volumes                             │
│                                      │
│  ┌────────────────────────────────┐  │
│  │ $ docker inspect nginx         │  │
│  │ {                              │  │
│  │   "Id": "...",                 │  │
│  │   ...                          │  │
│  │ }                              │  │
│  │ $                             │  │
│  └────────────────────────────────┘  │
└──────────────────────────────────────┘
```

Это возможно.

Но **безопасность здесь становится принципиально другой**.

Docker сам по себе предупреждает, что доступ к daemon очень привилегированный; веб-приложение, способное выполнять произвольные Docker-команды, нужно защищать особенно тщательно. ([Docker Documentation][2])

Поэтому я бы сделал:

```text
MVP:
Command Registry
       ↓
Generate
       ↓
Copy
```

а позже:

```text
Advanced:
Embedded Terminal
```

и только с явным включением пользователем.

---

# 4. Очень интересная фича — "Explain this"

Я бы пошёл даже дальше.

Например пользователь открывает Container:

```text
nginx-prod
```

и видит:

### CLI

```bash
docker inspect nginx-prod
```

**Explain**

> Показывает низкоуровневую конфигурацию контейнера: mounts, networks, environment, runtime configuration и т.д.

### Raw

```json
{
  ...
}
```

То есть:

**UI → Command → Raw Docker data**

Это станет одним из главных преимуществ твоей программы.

---

# 5. "Show me how UI got this value"

Это вообще может быть killer feature.

Допустим UI показывает:

```text
RAM
1.82 GB
```

При клике:

```text
Where does this come from?
```

открывается:

```text
Docker Engine API
GET /containers/{id}/stats

        ↓

memory_stats.usage

        ↓

Visualizer normalization

        ↓

1.82 GB
```

И рядом:

```bash
docker stats nginx-prod --no-stream
```

Это делает приложение **прозрачным и диагностическим**, а не просто красивой оболочкой.

---

# 6. "CLI parity mode"

Я бы добавил специальный режим:

> **CLI View**

Любой экран можно переключить:

```text
Visual     |     CLI
```

Например Containers:

```text
┌─────────────────────────────────────┐
│ Containers                          │
│                                     │
│ nginx-prod     running    2.1% CPU  │
│ postgres       running    4.8% CPU  │
│ redis          running    0.4% CPU  │
└─────────────────────────────────────┘
```

переключается в:

```bash
docker ps -a --size
```

с кнопкой:

**Copy**

А детали:

```bash
docker inspect nginx-prod
docker stats nginx-prod --no-stream
docker logs --tail 200 nginx-prod
docker network inspect frontend
```

Это очень хорошо соответствует твоей исходной идее.

---

# 7. Command Palette

Ещё одна фича, которую я бы добавил обязательно:

**Ctrl+K**

```text
> inspect nginx-prod

> logs nginx-prod

> stats nginx-prod

> network frontend

> volume postgres-data

> docker system df

> generate docker inspect command
```

То есть приложение превращается не только в dashboard, но и в **операционный инструмент**.

---

# 8. История команд

Можно хранить локально:

```text
Recent commands

docker inspect nginx
docker logs postgres --tail 500
docker network inspect proxy
docker system df -v
```

И:

**Copy again**

Но я бы **не сохранял stdout автоматически**, если там потенциально могут быть secrets.

---

# 9. "Danger level"

У каждой команды:

🟢 Read-only

🟡 Potentially disruptive

🔴 Destructive

Например:

```text
docker inspect
🟢 READ ONLY
```

```text
docker logs
🟢 READ ONLY
```

```text
docker exec
🟡 INTERACTIVE
```

```text
docker restart
🟠 CHANGES STATE
```

```text
docker rm
🔴 DESTRUCTIVE
```

Это особенно полезно, если позже ты добавишь возможность выполнять команды.

---

# 10. Мультиязычность — да, но не только UI

Я бы сделал i18n сразу правильно.

Не:

```text
if language == "ru"
```

а:

```text
i18n
 ├── ru
 ├── en
 ├── de
 └── ...
```

И локализовать:

* UI;
* ошибки;
* descriptions;
* tooltips;
* command explanations;
* onboarding;
* documentation внутри приложения.

**Но сами Docker-команды НЕ переводить.**

Например:

```text
Русский:
"Показать подробную конфигурацию контейнера"

Command:
docker inspect nginx
```

Команда остаётся стандартной.

---

# 11. Ещё фича, которую я бы очень рекомендовал — Docker Context Manager

Раз у тебя приложение уже умеет подключаться к Docker Engine, логично сделать:

```text
Docker Host

● Local
  Docker Desktop

○ Production
  ssh://user@server

○ Staging
  ssh://user@staging
```

И сверху:

```text
Docker: production
```

Все данные UI и сгенерированные команды автоматически используют этот context.

Docker contexts как раз предназначены для управления несколькими Docker daemons из одного клиента. ([Docker Documentation][3])

---

# 12. "Compare"

Очень полезная профессиональная функция:

```text
Compare containers
```

или:

```text
Compare stacks
```

Например:

```text
production       staging

Image            nginx:1.27    nginx:1.26
CPU              3.2%         4.1%
Memory           512 MB       620 MB
Networks         2            2
Volumes          3            3
Environment      12 vars      14 vars
```

А ещё интереснее:

### Compare Docker hosts

```text
Production       Staging

Containers       42             39
Images           81             74
Volumes          26             21
RAM              32 GB          16 GB
Docker           28.3           28.2
```

Это уже превращает программу из "Docker dashboard" в **Docker diagnostics tool**.

---

# 13. Snapshot / Diff

Ещё сильнее:

```text
Take snapshot
```

Сохраняешь inventory:

```text
docker-state-2026-08-09.json
```

Потом:

```text
Compare with current
```

Получаешь:

```text
+ container nginx-2
- container old-api
~ postgres image changed
~ volume size +4.2 GB
~ network configuration changed
```

Для диагностики production это реально полезно.

---

# 14. Health / Anomaly Center

Можно сделать:

```text
⚠ 7 issues detected
```

Например:

```text
🔴 Container restarting repeatedly
🟠 Healthcheck failing
🟠 Volume usage unavailable
🟡 Container has no healthcheck
🟡 Image has no tag
🟡 Unused volume
🔵 Container has writable layer > 2 GB
```

Причём каждое предупреждение должно иметь:

```text
Why?
How to investigate?
CLI command
```

Например:

```text
Container restarted 14 times.

Investigate:

docker inspect api
docker logs api --tail 500
```

---

# И я бы объединил всё это в одну концепцию

Получается не просто:

> Docker Visualizer

а:

> **Docker Visualizer + Diagnostics + CLI Companion**

Архитектурно:

```text
                    Docker Visualizer
                           │
          ┌────────────────┼─────────────────┐
          │                │                 │
          ▼                ▼                 ▼
      Dashboard       Diagnostics        Command
          │                │             Registry
          │                │                 │
          ▼                ▼                 ▼
       Graph          Anomalies        Bash / PowerShell
       Metrics        Health           Copy / Explain
       Inventory      Diff             Context-aware
          │
          └────────────────┬────────────────┘
                           ▼
                    Docker Engine API
```

А **embedded terminal я бы оставил как Phase 2/3**.

Причём если делать его, я бы не давал браузеру "выполнять shell". Лучше архитектура:

```text
Browser
   │
   │ WebSocket
   ▼
Go backend
   │
   ├── PTY
   │
   └── local shell
          ├── Bash
          ├── PowerShell
          └── CMD
```

с **явным opt-in**, подтверждением и очень чётким разграничением `read-only` / `interactive` / `destructive`.

А для большинства случаев **генератор команд + Copy Bash / Copy PowerShell будет даже лучше**: он не требует привилегированного shell bridge и сохраняет главный принцип твоего приложения — UI помогает понять Docker, но не скрывает стандартные Docker-инструменты.

И ещё важный момент: поскольку Docker CLI сам является клиентом Docker Engine API, твоя идея "UI → показать эквивалентную CLI-команду" концептуально очень чистая. ([Docker Documentation][4])

**Если выбирать только 5 следующих фич, я бы сделал именно так:**

1. 🌍 **i18n RU/EN**
2. 🧩 **Command Registry + Bash/PowerShell/CMD generator**
3. 🔍 **"How was this value calculated?" / API provenance**
4. ⚠️ **Diagnostics / Anomaly Center**
5. 🔄 **Snapshots + Diff**

А **Command Palette (Ctrl+K)** я бы добавил одновременно с Command Registry — они очень хорошо дополняют друг друга.

[1]: https://docs.docker.com/reference/cli/docker/context/use/?utm_source=chatgpt.com "docker context use | Docker Docs"
[2]: https://docs.docker.com/engine/security/?utm_source=chatgpt.com "Docker Engine security | Docker Docs"
[3]: https://docs.docker.com/engine/manage-resources/contexts/?utm_source=chatgpt.com "Docker contexts | Docker Docs"
[4]: https://docs.docker.com/engine/?utm_source=chatgpt.com "Docker Engine | Docker Docs"

Да. Я бы **не пытался реализовать всё сразу**. Если Cursor уже реализовал базовый план Docker Visualizer, следующий этап стоит сфокусировать на функциях, которые действительно отличат приложение от обычного Portainer-подобного UI.

Я бы оставил **5 вещей**:

1. **Command Registry** — главный приоритет.
2. **Command Palette (`Ctrl+K`)** — быстрый доступ к Registry.
3. **CLI provenance** — показать, откуда UI взял конкретное значение.
4. **Diagnostics / Anomaly Center** — автоматический поиск проблем.
5. **Snapshots + Diff** — сравнение состояния Docker во времени.

**i18n тоже добавить сейчас**, но как инфраструктурную возможность, а не как отдельную большую фичу.

А вот **embedded terminal пока не делать**. Сначала сделать идеальную генерацию `bash` / `PowerShell` / `CMD` команд с Copy. Терминал можно добавить позже, когда станет понятна модель безопасности.

Ниже я бы дал Cursor именно такой промпт.

# Task: Upgrade Docker Visualizer into a Docker Diagnostics & CLI Companion

## Context

You are working inside an existing Docker Visualizer project.

The project has already implemented the main architecture and functionality based on:

* the original PowerShell script provided in the repository;
* the existing Docker Visualizer architecture;
* Go backend;
* React + TypeScript frontend;
* Docker Engine API;
* REST API;
* WebSocket/live statistics;
* container, stack, network, volume, image and system views.

Before changing anything, inspect the entire existing repository and understand what has already been implemented.

Do NOT rewrite working functionality unnecessarily.

Your goal is to evolve the existing application into a professional:

> **Docker Visualizer + Diagnostics + CLI Companion**

The application should remain cross-platform and should continue to work with Docker Engine on:

* Windows
* Linux
* macOS
* Docker Desktop

---

# 1. First: Analyze the Existing Project

Before implementing anything:

1. Inspect the repository structure.
2. Inspect the Go backend.
3. Inspect the React frontend.
4. Inspect existing API contracts/OpenAPI.
5. Inspect Docker Engine integration.
6. Inspect existing models.
7. Inspect WebSocket implementation.
8. Inspect current routing/navigation.
9. Inspect current configuration.
10. Inspect existing tests.
11. Identify functionality that already exists.
12. Identify architectural problems or duplication.
13. Identify where the new features should integrate naturally.

Do NOT create duplicate abstractions if equivalent functionality already exists.

Create a short implementation assessment before modifying code:

```text
Existing architecture
Already implemented
Missing functionality
Recommended extension points
Potential risks
```

Then implement the plan.

---

# 2. Product Direction

The application should not become just another Docker dashboard.

Its main differentiator should be:

> The UI gives a high-level visual understanding of Docker, while always allowing the user to descend into the exact Docker CLI/API details behind the UI.

The user should be able to move naturally between:

```text
Visual UI
   ↓
Entity
   ↓
Detailed diagnostics
   ↓
Docker API information
   ↓
Equivalent CLI command
   ↓
Copy command
```

Example:

```text
Container: nginx-prod

CPU       3.2%
Memory    512 MB
Network   1.2 GB
Restarts  14
```

The user can click a metric and understand:

```text
Where did this value come from?
```

and then see:

```text
Docker Engine API
GET /containers/{id}/stats

memory_stats.usage

→ normalized by backend

→ displayed as 512 MB
```

and also:

```bash
docker stats nginx-prod --no-stream
```

---

# 3. Feature #1 — Command Registry

This is the highest-priority feature.

Create a reusable Command Registry abstraction.

Commands should be associated with entities and operations.

Examples:

## Container

```text
inspect
stats
logs
top
diff
port
exec
```

## Network

```text
inspect
```

## Volume

```text
inspect
```

## Image

```text
inspect
history
```

## System

```text
system df
info
version
```

The registry must NOT simply contain hardcoded UI strings.

Create a structured command model.

Conceptually:

```go
type CommandDefinition struct {
    ID                  string
    Title               string
    Description         string
    Category            string
    RiskLevel           string
    RequiresTTY         bool
    RequiresDockerCLI   bool

    SupportsBash        bool
    SupportsPowerShell  bool
    SupportsCMD         bool
}
```

The exact model should follow the existing project architecture.

Commands should be generated dynamically from the current Docker entity.

For example:

```text
Container:
nginx-prod
```

generates:

```bash
docker inspect nginx-prod
```

and:

```powershell
docker inspect nginx-prod
```

and:

```cmd
docker inspect nginx-prod
```

Do not duplicate command-generation logic throughout the frontend.

Create a centralized command-generation service.

---

# 4. Context-Aware Command Generation

Commands must respect the currently selected Docker host/context.

If the application is connected to a remote Docker Engine, generated commands should contain the necessary context information where appropriate.

For example:

```bash
docker --context production inspect nginx-prod
```

or an equivalent environment-based invocation when appropriate.

Do NOT assume that every Docker host is local.

The command generator must have access to the current connection/context.

---

# 5. Command Risk Classification

Every command must have a risk classification.

Use at least:

```text
READ_ONLY
INTERACTIVE
STATE_CHANGING
DESTRUCTIVE
```

Examples:

```text
docker inspect
READ_ONLY
```

```text
docker logs
READ_ONLY
```

```text
docker exec
INTERACTIVE
```

```text
docker restart
STATE_CHANGING
```

```text
docker rm
DESTRUCTIVE
```

This is important for future terminal/command execution support.

For the current implementation, the application should NOT execute arbitrary generated commands.

---

# 6. Command UI

Every relevant entity detail page should have a:

> CLI / Commands

section.

Example:

```text
CLI Commands

Inspect container
Show logs
Show statistics
Show network configuration
Show filesystem changes
```

Each command should display:

* title
* short explanation
* risk level
* generated command
* shell selector
* Copy button

Shell selector:

```text
Bash | PowerShell | CMD
```

Example:

```bash
docker inspect nginx-prod
```

Buttons:

```text
Copy
```

Do not make the command execution mechanism part of this phase.

---

# 7. Explain Commands

Every command should have a human-readable explanation.

Example:

```text
docker inspect nginx-prod
```

Explanation:

```text
Shows the low-level Docker configuration and runtime state
of the container, including mounts, networks, environment,
runtime configuration and metadata.
```

Descriptions should be localizable.

Do not hardcode explanations directly into React components.

---

# 8. Feature #2 — Command Palette

Implement a global command palette.

Keyboard shortcut:

```text
Ctrl+K
```

On macOS also support:

```text
Cmd+K
```

The palette should allow searching:

```text
inspect nginx
logs nginx
stats postgres
network proxy
volume postgres-data
system df
```

Results should include both application navigation and commands.

For example:

```text
Search

> inspect nginx-prod

Commands
  Inspect nginx-prod
  Logs nginx-prod
  Stats nginx-prod

Navigation
  Open nginx-prod
  Open Containers
```

The palette must be fast and keyboard-friendly.

Escape closes it.

Enter executes the selected UI action, such as navigating to the entity or opening the command detail.

Do not execute shell commands.

---

# 9. Feature #3 — Data Provenance / "Where did this value come from?"

This is a major differentiating feature.

For important values shown by the UI, provide a way to inspect their origin.

Examples:

```text
CPU
Memory
Network I/O
Block I/O
Restart count
Health
Writable layer size
Volume size
Container IP
```

Add a UI affordance such as:

```text
ⓘ
```

or:

```text
Where does this come from?
```

When clicked, show:

```text
Source

Docker Engine API

Endpoint:
GET /containers/{id}/stats

Docker field:
memory_stats.usage

Visualizer transformation:
bytes → human-readable MB

Displayed value:
512 MB
```

For values derived from multiple Docker API calls, show the full chain.

Example:

```text
Volume size

docker system df
        ↓
Volume usage
        ↓
volume association
        ↓
stack aggregation
        ↓
UI
```

The exact implementation should follow the existing backend architecture.

Do not expose internal implementation details that are meaningless to users.

The goal is transparency and diagnostics.

---

# 10. Feature #4 — Diagnostics / Anomaly Center

Add a diagnostics subsystem.

The application should analyze Docker state and identify useful problems.

Initial rules should include at least:

### Container

* restarting repeatedly
* unhealthy
* healthcheck failing
* stopped container
* very large writable layer
* unusually high memory usage
* unusually high CPU usage
* container without healthcheck

### Images

* dangling image
* unused image
* very large image

### Volumes

* unused volume
* unusually large volume
* volume size unavailable

### Networks

* unused network
* unusual configuration where detectable

### System

* Docker daemon information
* resource pressure where data is available

Do NOT invent metrics that Docker Engine does not provide.

Every diagnostic should contain:

```text
id
severity
entity
title
description
reason
recommendation
related commands
```

Severity:

```text
INFO
WARNING
CRITICAL
```

Example:

```text
CRITICAL

Container nginx-prod has restarted 14 times.

Why:
The restart count indicates repeated container termination.

Investigate:

docker inspect nginx-prod
docker logs nginx-prod --tail 500
```

The diagnostic should link directly to the entity and to the relevant Command Registry entries.

---

# 11. Diagnostics Must Be Explainable

Do not create opaque "AI-like" warnings.

Every diagnostic must explain:

```text
What was detected?
Why is it potentially a problem?
Which data caused the detection?
How can the user investigate it?
```

Example:

```text
Writable layer: 4.8 GB

Threshold: 2 GB

Source:
Docker container size information

Recommendation:
Investigate filesystem changes with:

docker diff nginx-prod
```

Thresholds should eventually be configurable.

For now, define sensible constants/configuration values in the backend rather than scattering magic numbers across the code.

---

# 12. Feature #5 — Snapshots and Diff

Implement Docker state snapshots.

A snapshot should represent the application-visible Docker inventory at a point in time.

It should contain enough information to compare:

* containers
* images
* networks
* volumes
* stacks
* relevant metrics/configuration

Do NOT store sensitive information unnecessarily.

Especially avoid blindly persisting:

* environment secrets
* credentials
* tokens
* container logs

Snapshot storage should be local and configurable.

Possible model:

```text
Snapshot
├── ID
├── timestamp
├── Docker host/context
├── Docker version
├── containers
├── images
├── networks
├── volumes
└── stacks
```

Implement:

```text
Create snapshot
List snapshots
View snapshot
Compare snapshot with current state
Compare snapshot A with snapshot B
```

Diff should show:

```text
+ container added
- container removed
~ container changed
+ image added
- image removed
~ image changed
+ volume added
~ volume size changed
~ network changed
```

Example:

```text
Production

Containers

+ api-2
- old-api
~ postgres
    image changed
    memory limit changed

Volumes

~ postgres-data
    +4.2 GB
```

Make the diff readable and useful.

---

# 13. Internationalization

Introduce proper i18n infrastructure.

Initial languages:

```text
English
Russian
```

The architecture must allow adding more languages later.

Translate:

* navigation
* UI labels
* diagnostics
* command descriptions
* tooltips
* errors
* onboarding/help text

Do NOT translate:

* Docker commands
* Docker API field names
* container/image/network/volume names
* user-provided values

Do not implement language checks such as:

```typescript
if (language === "ru") ...
```

Use a proper localization abstraction.

Persist the selected language locally.

Default to the browser/system language where appropriate, with English fallback.

---

# 14. Embedded Terminal — DO NOT IMPLEMENT YET

Do NOT add arbitrary shell execution in this phase.

Do NOT create:

```text
Browser → arbitrary shell
```

Do NOT allow the browser to execute generated commands automatically.

Instead, prepare the architecture so a future version could support:

```text
Browser
   ↓ WebSocket
Go backend
   ↓
PTY
   ↓
Bash / PowerShell / CMD
```

But leave actual terminal execution out of this implementation.

The current workflow is:

```text
Generate command
→ Explain command
→ Copy command
→ User executes it manually
```

This keeps the current security model simple.

---

# 15. Security Requirements

Treat Docker access as privileged.

Do not expose Docker Engine credentials or sockets to the frontend.

The React application must never communicate directly with Docker Engine.

Architecture remains:

```text
React
   ↓
Go API
   ↓
Docker Engine
```

Do not expose arbitrary command execution endpoints.

Do not add an endpoint such as:

```text
POST /exec
{
    "command": "..."
}
```

in this phase.

Do not persist secrets from Docker inspect/environment data in snapshots.

Be careful when displaying environment variables.

Sensitive-looking values should be masked where appropriate.

---

# 16. API Design

Extend the existing API instead of replacing it.

Potential new endpoints:

```text
GET /api/v1/commands
GET /api/v1/commands/{id}

GET /api/v1/diagnostics
GET /api/v1/diagnostics/{id}

POST /api/v1/snapshots
GET /api/v1/snapshots
GET /api/v1/snapshots/{id}
GET /api/v1/snapshots/{id}/diff
```

These are examples, not mandatory exact routes.

Follow the project's existing API conventions.

Update OpenAPI accordingly.

Keep domain logic out of HTTP handlers.

---

# 17. Backend Architecture

Maintain separation between:

```text
API
Domain/Application logic
Docker integration
Models
Storage
Command generation
Diagnostics
```

Suggested conceptual structure:

```text
internal/
├── api/
├── docker/
├── commands/
├── diagnostics/
├── snapshots/
├── provenance/
├── models/
├── storage/
├── ws/
└── config/
```

Adapt this to the existing project rather than blindly creating these directories.

Command generation should be independently testable.

Diagnostics should be independently testable.

Snapshot diffing should be independently testable.

---

# 18. Frontend UX

Integrate the features into the existing UI.

Do NOT create five unrelated pages.

The main UX should be:

```text
Dashboard
Containers
Stacks
Networks
Volumes
Images
System
Diagnostics
Snapshots
```

Entity detail pages should expose:

```text
Overview
Metrics
Networks
Volumes
Logs
Raw / Inspect
CLI Commands
Data Source
```

Diagnostics should link to entities.

Commands should link back to entities.

Provenance should link to the API/data source.

Snapshots should link to the changed entities.

Everything should feel interconnected.

---

# 19. Visual Design

Keep the existing visual language of the application.

Do not redesign the entire application unless necessary.

The new UI should be:

* professional
* compact
* technical
* readable
* dark-mode friendly
* keyboard friendly
* suitable for developers and DevOps engineers

Risk levels should be visually distinguishable but not excessively colorful.

Use consistent badges:

```text
READ ONLY
INTERACTIVE
STATE CHANGING
DESTRUCTIVE
```

Diagnostics:

```text
INFO
WARNING
CRITICAL
```

---

# 20. Testing

Add tests for all new core logic.

At minimum:

## Command generator

Test:

```text
container inspect
container logs
container stats
network inspect
volume inspect
image inspect
system df
```

for:

```text
Bash
PowerShell
CMD
```

Test context-aware generation.

Test proper quoting/escaping for entity names.

## Diagnostics

Test each diagnostic rule.

## Snapshots

Test:

```text
empty → populated
added entity
removed entity
changed entity
unchanged entity
```

## i18n

Test fallback behavior.

## API

Test new endpoints.

Do not rely only on frontend tests.

---

# 21. Documentation

Update the project documentation.

Document:

* Command Registry
* command generation
* supported shells
* command risk levels
* diagnostics
* snapshots
* i18n
* security model
* why arbitrary command execution is intentionally not implemented

Add developer documentation where architecture is non-obvious.

---

# 22. Important Engineering Constraints

Do not:

* rewrite the existing backend unnecessarily;
* replace working Docker integration without a reason;
* introduce unnecessary dependencies;
* duplicate Docker API calls;
* put Docker business logic into React;
* hardcode commands in UI components;
* hardcode localization strings throughout the application;
* store secrets in snapshots;
* introduce arbitrary shell execution;
* create fake Docker metrics;
* make assumptions about Linux-only Docker behavior.

Prefer:

* existing dependencies;
* existing abstractions;
* Docker Engine API;
* typed models;
* deterministic logic;
* testable services;
* clear separation of concerns.

---

# 23. Implementation Order

Implement in this order:

### Phase 1

Repository analysis and architecture assessment.

### Phase 2

i18n foundation.

### Phase 3

Command Registry.

### Phase 4

Context-aware Bash / PowerShell / CMD command generation.

### Phase 5

CLI UI integrated into entity pages.

### Phase 6

Command Palette.

### Phase 7

Data provenance.

### Phase 8

Diagnostics engine.

### Phase 9

Diagnostics UI.

### Phase 10

Snapshot storage.

### Phase 11

Snapshot diff engine.

### Phase 12

Snapshot UI.

### Phase 13

Tests.

### Phase 14

Documentation and final architectural cleanup.

---

# 24. Definition of Done

The implementation is complete when:

1. Existing functionality still works.
2. English and Russian UI are supported.
3. Every important Docker entity has useful CLI commands.
4. Commands can be generated for Bash, PowerShell and CMD.
5. Commands respect the current Docker host/context.
6. Commands have explanations and risk classifications.
7. Commands can be copied with one click.
8. Ctrl/Cmd+K provides a global command palette.
9. Important UI values can expose their data provenance.
10. Diagnostics detect meaningful Docker problems.
11. Diagnostics explain why they were detected.
12. Diagnostics provide investigation commands.
13. Docker state snapshots can be created.
14. Snapshots can be compared.
15. Changes are displayed clearly.
16. No arbitrary shell execution has been introduced.
17. Secrets are not unnecessarily persisted.
18. OpenAPI is updated.
19. Backend tests exist for core logic.
20. Frontend builds successfully.
21. Go tests pass.
22. The application remains cross-platform.
23. Documentation is updated.

---

# Final instruction

Do not treat this as a greenfield project.

This is an evolution of an already implemented Docker Visualizer.

First understand the current codebase.

Then implement the smallest clean architectural extension that provides the functionality above.

Prefer correctness, maintainability and security over adding more features.

The central product principle should remain:

> **Visualize Docker at a high level, but never hide the underlying Docker reality.**

The user should always be able to go from:

```text
What is happening?
        ↓
Why is it happening?
        ↓
Where did this value come from?
        ↓
How do I investigate it?
        ↓
What exact Docker command should I run?
```

without leaving the application.

Я бы **именно этот промпт** отдал Cursor, а не добавлял туда embedded terminal. Он задаёт хорошую границу: сначала сделать приложение сильным **инструментом диагностики и мостом между UI и Docker CLI**, а уже после этого решать, нужен ли полноценный terminal.

Особенно важны три фичи, которые я считаю наиболее удачными: **Command Registry + Provenance + Diagnostics**. Вместе они превращают твой проект из очередного Docker dashboard в инструмент, который помогает понять *почему* UI показывает именно это и *какой низкоуровневой командой это проверить*.

