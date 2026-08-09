# Open TODOs

Tracked backlog beyond the current implementation.

## V2 remaining

- [ ] **Optional safe mutations** behind explicit `--enable-mutate`  
  Read-only default stays (ADR-009). When enabled: carefully scoped actions only (e.g. container restart/stop with confirmations), never arbitrary Engine calls from the UI. Needs ADR + OpenAPI + UI guards + audit log.

## Deferred / out of scope for now

- Per-volume growth history (external TSDB)
- Federated multi-host inventory merge
- Prometheus scrape endpoint for process metrics
