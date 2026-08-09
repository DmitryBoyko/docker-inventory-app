package ws

import "github.com/epm-games/docker-visualizer/internal/domain"

// Bus is the collector → WebSocket publish surface (optionally host-scoped).
type Bus interface {
	PublishStats(items []StatsItem)
	PublishSnapshotUpdated(n SnapshotNotice)
	PublishDockerEvent(ev DockerEvent)
	PublishConnectionStatus(st domain.ConnectionStatus)
	PublishEventsStatus(connected bool, errMsg string)
}

// Bind returns a Bus that tags envelopes with host (ADR-014).
func (h *Hub) Bind(host string) Bus {
	if h == nil {
		return nopBus{}
	}
	if host == "" {
		return h
	}
	return &boundBus{hub: h, host: host}
}

type boundBus struct {
	hub  *Hub
	host string
}

func (b *boundBus) PublishStats(items []StatsItem) {
	b.hub.Publish(Envelope{Type: TypeContainerStats, Host: b.host, Timestamp: nowTS(), Data: items})
}

func (b *boundBus) PublishSnapshotUpdated(n SnapshotNotice) {
	b.hub.Publish(Envelope{Type: TypeSnapshotUpdated, Host: b.host, Timestamp: nowTS(), Data: n})
}

func (b *boundBus) PublishDockerEvent(ev DockerEvent) {
	b.hub.Publish(Envelope{Type: TypeDockerEvent, Host: b.host, Timestamp: nowTS(), Data: ev})
}

func (b *boundBus) PublishConnectionStatus(st domain.ConnectionStatus) {
	st.Name = b.host
	b.hub.Publish(Envelope{Type: TypeConnectionStatus, Host: b.host, Timestamp: nowTS(), Data: st})
}

func (b *boundBus) PublishEventsStatus(connected bool, errMsg string) {
	b.hub.Publish(Envelope{Type: TypeEventsStatus, Host: b.host, Timestamp: nowTS(), Data: EventsStatus{
		Connected: connected,
		Error:     errMsg,
		Host:      b.host,
	}})
}

type nopBus struct{}

func (nopBus) PublishStats([]StatsItem)                            {}
func (nopBus) PublishSnapshotUpdated(SnapshotNotice)               {}
func (nopBus) PublishDockerEvent(DockerEvent)                      {}
func (nopBus) PublishConnectionStatus(domain.ConnectionStatus)     {}
func (nopBus) PublishEventsStatus(connected bool, errMsg string)   {}
