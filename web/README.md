# Docker Visualizer UI

React + TypeScript + Vite shell for the Go API.

## Dev

```bash
# terminal 1 — API
go run ./cmd/docker-visualizer

# terminal 2 — UI (proxies /api to :8080)
cd web && npm run dev
```

Open http://127.0.0.1:5173

Vite proxies `/api` (including WebSocket `/api/v1/ws`) to the Go backend.

With WS connected, REST polling slows down; live CPU/RAM come from `container.stats`.

## Production build (served by Go)

```bash
cd web && npm run build
go run ./cmd/docker-visualizer
```

If `web/dist` exists, the API process serves the SPA at http://127.0.0.1:8080
