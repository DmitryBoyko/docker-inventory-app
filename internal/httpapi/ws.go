package httpapi

import (
	"net/http"

	"github.com/coder/websocket"
	"github.com/epm-games/docker-visualizer/internal/ws"
)

// EventsStatus provides Docker events stream connectivity for WS seeding /ready.
type EventsStatus interface {
	Connected() bool
	LastError() string
}

func (s *Server) handleWS(w http.ResponseWriter, r *http.Request) {
	if s.Hub == nil {
		writeErr(w, http.StatusServiceUnavailable, "not_ready", "websocket hub not initialized")
		return
	}
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		// Local ops tool; Vite dev proxy may change Origin.
		InsecureSkipVerify: true,
	})
	if err != nil {
		return
	}
	conn.SetReadLimit(1 << 20)

	ctx := r.Context()
	client := ws.NewClient(s.Hub, conn)
	s.Hub.Register(client)

	// Seed after register so the client receives initial status.
	if s.Docker != nil {
		s.Hub.PublishConnectionStatus(s.Docker.Ping(ctx))
	}
	if s.Events != nil {
		s.Hub.PublishEventsStatus(s.Events.Connected(), s.Events.LastError())
	}

	go client.WritePump(ctx)
	client.ReadPump(ctx)
}
