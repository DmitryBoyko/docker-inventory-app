# ADR-004: React embedded into Go binary

## Status

Accepted (Phase 0)

## Context

Distribution target is one executable per platform that serves both API and UI.

## Decision

Build the React (Vite) app into `web/dist` and embed it with `go:embed` from the Go process. SPA fallback serves `index.html` for non-API routes.

## Alternatives

- Separate static hosting
- Wails / Tauri desktop shells
- Server-rendered HTML only

## Why

Simplest operator experience: download binary, run, open browser.

## Consequences

Release pipeline needs Node for the frontend build. Phase 1 does not yet embed UI (starts at packaging / later phases); contract is locked now.
