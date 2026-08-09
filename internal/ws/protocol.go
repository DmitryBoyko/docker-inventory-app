package ws

import "github.com/epm-games/docker-visualizer/internal/domain"

// Server → client envelope types.
const (
	TypeContainerStats   = "container.stats"
	TypeDockerEvent      = "docker.event"
	TypeSnapshotUpdated  = "snapshot.updated"
	TypeConnectionStatus = "connection.status"
	TypeEventsStatus     = "events.status"
	TypePing             = "ping"
	TypePong             = "pong"
	TypeError            = "error"
)

// Client → server actions.
const (
	ActionSubscribe   = "subscribe"
	ActionUnsubscribe = "unsubscribe"
	ActionPing        = "ping"
)

// Channels clients can subscribe to.
const (
	ChannelStats     = "stats"
	ChannelEvents    = "events"
	ChannelSnapshots = "snapshots"
	ChannelConnection = "connection"
)

// Envelope is the server→client frame.
type Envelope struct {
	Type      string `json:"type"`
	Host      string `json:"host,omitempty"` // ADR-014 host name
	Timestamp string `json:"timestamp"`
	Data      any    `json:"data,omitempty"`
}

// ClientMessage is the client→server frame.
type ClientMessage struct {
	Action  string         `json:"action"`
	Channel string         `json:"channel,omitempty"`
	Host    string         `json:"host,omitempty"` // select_host / subscribe host filter
	Filters *StatsFilters  `json:"filters,omitempty"`
}

// StatsFilters narrows container.stats fan-out.
type StatsFilters struct {
	Stack        string   `json:"stack,omitempty"`
	ContainerIDs []string `json:"containerIds,omitempty"`
	Host         string   `json:"host,omitempty"`
}

// StatsItem is one container sample on the wire.
type StatsItem struct {
	ID      string                 `json:"id"`
	IDShort string                 `json:"idShort"`
	Name    string                 `json:"name"`
	Stack   string                 `json:"stack"`
	Stats   *domain.ContainerStats `json:"stats"`
}

// SnapshotNotice is a lightweight inventory change notice (not a full dump).
type SnapshotNotice struct {
	Version    uint64 `json:"version"`
	SnapshotAt string `json:"snapshotAt,omitempty"`
	StatsAt    string `json:"statsAt,omitempty"`
	SystemAt   string `json:"systemAt,omitempty"`
	Kind       string `json:"kind"` // inventory|stats|system
}

// DockerEvent is a normalized Engine event.
type DockerEvent struct {
	Type     string            `json:"type"`
	Action   string            `json:"action"`
	ActorID  string            `json:"actorId"`
	ActorName string            `json:"actorName,omitempty"`
	Attrs    map[string]string `json:"attrs,omitempty"`
	Scope    string            `json:"scope,omitempty"`
	Time     string            `json:"time"`
}

// EventsStatus reports whether the Docker events stream is live.
type EventsStatus struct {
	Connected bool   `json:"connected"`
	Error     string `json:"error,omitempty"`
	Host      string `json:"host,omitempty"`
}

// ActionSelectHost switches the client's host filter (ADR-014).
const ActionSelectHost = "select_host"
