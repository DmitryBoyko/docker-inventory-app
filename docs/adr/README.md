# Architecture Decision Records

Canonical decisions for Docker Visualizer. Source narrative: [`../implementation-plan.md`](../implementation-plan.md).

| ID | Title |
|----|-------|
| [ADR-001](001-go.md) | Go as implementation language |
| [ADR-002](002-docker-engine-api.md) | Docker Engine API instead of CLI |
| [ADR-003](003-moby-client.md) | Moby Go client module |
| [ADR-004](004-react-embed.md) | React embedded into Go binary |
| [ADR-005](005-rest-websocket.md) | REST + limited WebSocket |
| [ADR-006](006-docker-events.md) | Docker events for invalidation |
| [ADR-007](007-domain-models.md) | Domain models separate from SDK |
| [ADR-008](008-compose-stack.md) | Stack derived from Compose labels |
| [ADR-009](009-read-only.md) | Read-only by default |
| [ADR-010](010-endpoint-discovery.md) | Docker endpoint discovery |
| [ADR-011](011-volume-usage-availability.md) | Volume usage availability semantics |
| [ADR-012](012-snapshot-store.md) | In-memory snapshot store |
| [ADR-013](013-auth-token.md) | Bearer token for non-localhost bind |

To change a decision: add a superseding ADR and update the implementation plan.
