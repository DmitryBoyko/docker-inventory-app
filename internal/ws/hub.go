package ws

import (
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"github.com/epm-games/docker-visualizer/internal/domain"
)

const (
	clientSendBuffer = 16
	maxDrops         = 64
	heartbeatEvery   = 15 * time.Second
)

// Hub fans out realtime messages to WebSocket clients (ADR-005).
type Hub struct {
	mu      sync.RWMutex
	clients map[*Client]struct{}
	log     *slog.Logger

	broadcast chan Envelope
}

// NewHub creates an idle hub. Call Run.
func NewHub(log *slog.Logger) *Hub {
	if log == nil {
		log = slog.Default()
	}
	return &Hub{
		clients:   make(map[*Client]struct{}),
		log:       log,
		broadcast: make(chan Envelope, 64),
	}
}

// Run processes broadcast + heartbeat until ctx is done.
func (h *Hub) Run(ctxDone <-chan struct{}) {
	ticker := time.NewTicker(heartbeatEvery)
	defer ticker.Stop()
	for {
		select {
		case <-ctxDone:
			h.closeAll()
			return
		case env := <-h.broadcast:
			h.fanOut(env)
		case <-ticker.C:
			h.fanOut(Envelope{Type: TypePing, Timestamp: nowTS()})
		}
	}
}

// Register attaches a client synchronously (so seed publishes are not lost).
func (h *Hub) Register(c *Client) {
	h.mu.Lock()
	h.clients[c] = struct{}{}
	h.mu.Unlock()
}

// Unregister detaches a client.
func (h *Hub) Unregister(c *Client) {
	h.mu.Lock()
	if _, ok := h.clients[c]; ok {
		delete(h.clients, c)
		c.closeSend()
	}
	h.mu.Unlock()
}

// Publish enqueues a server envelope (non-blocking for collectors).
func (h *Hub) Publish(env Envelope) {
	if env.Timestamp == "" {
		env.Timestamp = nowTS()
	}
	select {
	case h.broadcast <- env:
	default:
		select {
		case <-h.broadcast:
		default:
		}
		select {
		case h.broadcast <- env:
		default:
			h.log.Warn("ws hub broadcast buffer full; dropping message", "type", env.Type)
		}
	}
}

// PublishStats sends container.stats to subscribers (filtered per client).
func (h *Hub) PublishStats(items []StatsItem) {
	h.Publish(Envelope{Type: TypeContainerStats, Timestamp: nowTS(), Data: items})
}

// PublishSnapshotUpdated notifies inventory/stats/system refresh.
func (h *Hub) PublishSnapshotUpdated(n SnapshotNotice) {
	h.Publish(Envelope{Type: TypeSnapshotUpdated, Timestamp: nowTS(), Data: n})
}

// PublishDockerEvent broadcasts a normalized event.
func (h *Hub) PublishDockerEvent(ev DockerEvent) {
	h.Publish(Envelope{Type: TypeDockerEvent, Timestamp: nowTS(), Data: ev})
}

// PublishConnectionStatus pushes docker connectivity.
func (h *Hub) PublishConnectionStatus(st domain.ConnectionStatus) {
	h.Publish(Envelope{Type: TypeConnectionStatus, Timestamp: nowTS(), Data: st})
}

// PublishEventsStatus pushes events stream connectivity.
func (h *Hub) PublishEventsStatus(connected bool, errMsg string) {
	h.Publish(Envelope{Type: TypeEventsStatus, Timestamp: nowTS(), Data: EventsStatus{
		Connected: connected,
		Error:     errMsg,
	}})
}

// Ensure *Hub implements Bus.
var _ Bus = (*Hub)(nil)

func (h *Hub) fanOut(env Envelope) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for c := range h.clients {
		if env.Host != "" && c.hostFilter() != "" && env.Host != c.hostFilter() {
			continue
		}
		if !c.accepts(env) {
			continue
		}
		payload := env
		if env.Type == TypeContainerStats {
			filtered := c.filterStats(env)
			if filtered == nil {
				continue
			}
			payload.Data = filtered
		}
		c.trySend(payload)
	}
}

func (h *Hub) closeAll() {
	h.mu.Lock()
	defer h.mu.Unlock()
	for c := range h.clients {
		c.closeSend()
		delete(h.clients, c)
	}
}

// ClientCount is for tests/diagnostics.
func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

func nowTS() string {
	return time.Now().UTC().Format(time.RFC3339Nano)
}

// Encode is a small helper for tests.
func Encode(env Envelope) ([]byte, error) {
	return json.Marshal(env)
}
