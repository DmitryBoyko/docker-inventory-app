package httpapi

import (
	"net/http"

	"github.com/coder/websocket"
	"github.com/epm-games/docker-visualizer/internal/ws"
)

func (s *Server) handleWS(w http.ResponseWriter, r *http.Request) {
	if s.Hub == nil || s.Hosts == nil {
		writeErr(w, http.StatusServiceUnavailable, "not_ready", "websocket hub not initialized")
		return
	}
	rt, err := s.Hosts.Get(r.URL.Query().Get("host"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "unknown_host", err.Error())
		return
	}

	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true,
	})
	if err != nil {
		return
	}
	conn.SetReadLimit(1 << 20)

	ctx := r.Context()
	client := ws.NewClient(s.Hub, conn, rt.Name)
	s.Hub.Register(client)

	st := rt.Docker.Ping(ctx)
	st.Name = rt.Name
	rt.Bus.PublishConnectionStatus(st)
	if rt.Events != nil {
		rt.Bus.PublishEventsStatus(rt.Events.Connected(), rt.Events.LastError())
	}

	go client.WritePump(ctx)
	client.ReadPump(ctx)
}
