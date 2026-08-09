package httpapi

import (
	"net/http"

	"github.com/epm-games/docker-visualizer/internal/hosts"
)

// runtime resolves ?host= (empty → default). Writes 400 on unknown host.
func (s *Server) runtime(w http.ResponseWriter, r *http.Request) *hosts.Runtime {
	if s.Hosts == nil {
		writeErr(w, http.StatusServiceUnavailable, "not_ready", "hosts not initialized")
		return nil
	}
	rt, err := s.Hosts.Get(r.URL.Query().Get("host"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "unknown_host", err.Error())
		return nil
	}
	return rt
}
