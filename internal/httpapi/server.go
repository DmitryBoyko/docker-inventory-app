package httpapi

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/epm-games/docker-visualizer/internal/app"
	"github.com/epm-games/docker-visualizer/internal/docker"
	"github.com/epm-games/docker-visualizer/internal/ws"
)

// Server is the HTTP API surface.
type Server struct {
	Docker      *docker.Client
	Containers  *app.ContainersService
	Live        *app.ContainerLiveService
	Stacks      *app.StacksService
	Networks    *app.NetworksService
	Volumes     *app.VolumesService
	Images      *app.ImagesService
	System      *app.SystemService
	Graph       *app.GraphService
	Diagnostics *app.DiagnosticsService
	Hub         *ws.Hub
	Events      EventsStatus
	Version     string
}

// Handler returns the root mux. API lives under /api/v1.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/health", s.handleHealth)
	mux.HandleFunc("GET /api/v1/ready", s.handleReady)
	mux.HandleFunc("GET /health", s.handleHealth)
	mux.HandleFunc("GET /ready", s.handleReady)

	mux.HandleFunc("GET /api/v1/containers", s.handleListContainers)
	mux.HandleFunc("GET /api/v1/containers/{id}/stats", s.handleGetContainerStats)
	mux.HandleFunc("GET /api/v1/containers/{id}/inspect", s.handleGetContainerInspect)
	mux.HandleFunc("GET /api/v1/containers/{id}/logs", s.handleGetContainerLogs)
	mux.HandleFunc("GET /api/v1/containers/{id}", s.handleGetContainer)

	mux.HandleFunc("GET /api/v1/stacks", s.handleListStacks)
	mux.HandleFunc("GET /api/v1/stacks/{name}/volumes", s.handleStackVolumes)
	mux.HandleFunc("GET /api/v1/stacks/{name}", s.handleGetStack)

	mux.HandleFunc("GET /api/v1/networks", s.handleListNetworks)
	mux.HandleFunc("GET /api/v1/networks/{id}", s.handleGetNetwork)

	mux.HandleFunc("GET /api/v1/volumes", s.handleListVolumes)
	mux.HandleFunc("GET /api/v1/volumes/{name}", s.handleGetVolume)

	mux.HandleFunc("GET /api/v1/images", s.handleListImages)
	mux.HandleFunc("GET /api/v1/images/{id}", s.handleGetImage)

	mux.HandleFunc("GET /api/v1/graph", s.handleGraph)

	mux.HandleFunc("GET /api/v1/system/df", s.handleSystemDF)
	mux.HandleFunc("GET /api/v1/system/resources", s.handleSystemResources)
	mux.HandleFunc("GET /api/v1/system/info", s.handleSystemInfo)
	mux.HandleFunc("GET /api/v1/system/settings", s.handleSystemSettings)
	mux.HandleFunc("GET /api/v1/system/diagnostics", s.handleDiagnostics)
	mux.HandleFunc("GET /api/v1/ws", s.handleWS)
	return mux
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status":    "ok",
		"version":   s.Version,
		"timestamp": time.Now().UTC().Format(time.RFC3339Nano),
	})
}

func (s *Server) handleReady(w http.ResponseWriter, r *http.Request) {
	st := s.Docker.Ping(r.Context())
	ts := time.Now().UTC().Format(time.RFC3339Nano)
	eventsConnected := false
	eventsErr := ""
	if s.Events != nil {
		eventsConnected = s.Events.Connected()
		eventsErr = s.Events.LastError()
	}
	eventsPayload := map[string]any{
		"connected": eventsConnected,
		"error":     nullIfEmpty(eventsErr),
	}
	if !st.Connected {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{
			"error": map[string]any{
				"code":    "docker_unavailable",
				"message": st.Error,
				"details": map[string]any{
					"host":    st.Host,
					"source":  st.Source,
					"context": st.Context,
				},
			},
			"ready":     false,
			"docker":    st,
			"events":    eventsPayload,
			"timestamp": ts,
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ready":     true,
		"docker":    st,
		"events":    eventsPayload,
		"timestamp": ts,
	})
}

func (s *Server) handleListContainers(w http.ResponseWriter, r *http.Request) {
	if s.Containers == nil {
		writeErr(w, http.StatusServiceUnavailable, "not_ready", "container inventory not initialized")
		return
	}
	q := app.ContainerQuery{
		Stack:  r.URL.Query().Get("stack"),
		State:  r.URL.Query().Get("state"),
		Health: r.URL.Query().Get("health"),
		Q:      r.URL.Query().Get("q"),
	}
	res := s.Containers.List(q)
	writeJSON(w, http.StatusOK, map[string]any{
		"timestamp":     time.Now().UTC().Format(time.RFC3339Nano),
		"snapshotAt":    formatTimePtr(res.SnapshotAt),
		"snapshotAgeMs": res.SnapshotAge.Milliseconds(),
		"collectError":  nullIfEmpty(res.CollectErr),
		"data":          res.Containers,
	})
}

func (s *Server) handleGetContainer(w http.ResponseWriter, r *http.Request) {
	if s.Containers == nil {
		writeErr(w, http.StatusServiceUnavailable, "not_ready", "container inventory not initialized")
		return
	}
	c, snapAt, ok := s.Containers.Get(r.PathValue("id"))
	if !ok {
		writeErr(w, http.StatusNotFound, "not_found", "container not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"timestamp":  time.Now().UTC().Format(time.RFC3339Nano),
		"snapshotAt": formatTimePtr(snapAt),
		"data":       c,
	})
}

func (s *Server) handleGetContainerStats(w http.ResponseWriter, r *http.Request) {
	if s.Containers == nil {
		writeErr(w, http.StatusServiceUnavailable, "not_ready", "container inventory not initialized")
		return
	}
	res := s.Containers.GetStats(r.PathValue("id"))
	if !res.Found {
		writeErr(w, http.StatusNotFound, "not_found", "container not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"timestamp":  time.Now().UTC().Format(time.RFC3339Nano),
		"snapshotAt": formatTimePtr(res.SnapshotAt),
		"statsAt":    formatTimePtr(res.StatsAt),
		"running":    res.Running,
		"data":       res.Stats,
	})
}

func (s *Server) handleGetContainerInspect(w http.ResponseWriter, r *http.Request) {
	if s.Live == nil {
		writeErr(w, http.StatusServiceUnavailable, "not_ready", "live container API not initialized")
		return
	}
	doRedact := parseBoolDefaultTrue(r.URL.Query().Get("redact"))
	res, err := s.Live.Inspect(r.Context(), r.PathValue("id"), doRedact)
	if err != nil {
		if app.IsNotFound(err) {
			writeErr(w, http.StatusNotFound, "not_found", "container not found")
			return
		}
		writeErr(w, http.StatusBadGateway, "docker_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"timestamp": time.Now().UTC().Format(time.RFC3339Nano),
		"data": map[string]any{
			"id":             res.ID,
			"name":           res.Name,
			"redacted":       res.Redacted,
			"redactedFields": res.RedactedFields,
			"inspect":        json.RawMessage(res.Inspect),
		},
	})
}

func (s *Server) handleGetContainerLogs(w http.ResponseWriter, r *http.Request) {
	if s.Live == nil {
		writeErr(w, http.StatusServiceUnavailable, "not_ready", "live container API not initialized")
		return
	}
	q := r.URL.Query()
	tail := 0
	if v := q.Get("tail"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			tail = n
		}
	}
	res, err := s.Live.Logs(r.Context(), r.PathValue("id"), app.LogsOptions{
		Tail:       tail,
		Since:      q.Get("since"),
		Timestamps: parseBoolDefaultFalse(q.Get("timestamps")),
	})
	if err != nil {
		if app.IsNotFound(err) {
			writeErr(w, http.StatusNotFound, "not_found", "container not found")
			return
		}
		writeErr(w, http.StatusBadGateway, "docker_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"timestamp": time.Now().UTC().Format(time.RFC3339Nano),
		"data": map[string]any{
			"id":         res.ID,
			"name":       res.Name,
			"tail":       res.Tail,
			"since":      nullIfEmpty(res.Since),
			"timestamps": res.Timestamps,
			"truncated":  res.Truncated,
			"text":       res.Text,
			"warning":    "Logs are fetched on demand and are not persisted by Docker Visualizer.",
		},
	})
}

func (s *Server) handleListStacks(w http.ResponseWriter, r *http.Request) {
	if s.Stacks == nil {
		writeErr(w, http.StatusServiceUnavailable, "not_ready", "stacks not initialized")
		return
	}
	res := s.Stacks.List()
	writeJSON(w, http.StatusOK, map[string]any{
		"timestamp":     time.Now().UTC().Format(time.RFC3339Nano),
		"snapshotAt":    formatTimePtr(res.SnapshotAt),
		"statsAt":       formatTimePtr(res.StatsAt),
		"systemAt":      formatTimePtr(res.SystemAt),
		"snapshotAgeMs": res.SnapshotAge.Milliseconds(),
		"data":          res.Stacks,
	})
}

func (s *Server) handleGetStack(w http.ResponseWriter, r *http.Request) {
	if s.Stacks == nil {
		writeErr(w, http.StatusServiceUnavailable, "not_ready", "stacks not initialized")
		return
	}
	st, meta, ok := s.Stacks.Get(r.PathValue("name"))
	if !ok {
		writeErr(w, http.StatusNotFound, "not_found", "stack not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"timestamp":  time.Now().UTC().Format(time.RFC3339Nano),
		"snapshotAt": formatTimePtr(meta.SnapshotAt),
		"statsAt":    formatTimePtr(meta.StatsAt),
		"systemAt":   formatTimePtr(meta.SystemAt),
		"data":       st,
	})
}

func (s *Server) handleStackVolumes(w http.ResponseWriter, r *http.Request) {
	if s.Stacks == nil {
		writeErr(w, http.StatusServiceUnavailable, "not_ready", "stacks not initialized")
		return
	}
	vols, ok := s.Stacks.StackVolumes(r.PathValue("name"))
	if !ok {
		writeErr(w, http.StatusNotFound, "not_found", "stack not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"timestamp": time.Now().UTC().Format(time.RFC3339Nano),
		"data":      vols,
	})
}

func (s *Server) handleGraph(w http.ResponseWriter, r *http.Request) {
	if s.Graph == nil {
		writeErr(w, http.StatusServiceUnavailable, "not_ready", "graph not initialized")
		return
	}
	res, err := s.Graph.Get(r.URL.Query().Get("scope"), r.URL.Query().Get("stack"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"timestamp":     time.Now().UTC().Format(time.RFC3339Nano),
		"snapshotAt":    formatTimePtr(res.SnapshotAt),
		"statsAt":       formatTimePtr(res.StatsAt),
		"snapshotAgeMs": res.SnapshotAge.Milliseconds(),
		"data":          res.Graph,
	})
}

func (s *Server) handleListNetworks(w http.ResponseWriter, r *http.Request) {
	if s.Networks == nil {
		writeErr(w, http.StatusServiceUnavailable, "not_ready", "networks not initialized")
		return
	}
	res := s.Networks.List(r.URL.Query().Get("q"), r.URL.Query().Get("driver"))
	writeJSON(w, http.StatusOK, map[string]any{
		"timestamp":     time.Now().UTC().Format(time.RFC3339Nano),
		"snapshotAt":    formatTimePtr(res.SnapshotAt),
		"snapshotAgeMs": res.SnapshotAge.Milliseconds(),
		"data":          res.Networks,
	})
}

func (s *Server) handleGetNetwork(w http.ResponseWriter, r *http.Request) {
	if s.Networks == nil {
		writeErr(w, http.StatusServiceUnavailable, "not_ready", "networks not initialized")
		return
	}
	n, snapAt, ok := s.Networks.Get(r.PathValue("id"))
	if !ok {
		writeErr(w, http.StatusNotFound, "not_found", "network not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"timestamp":  time.Now().UTC().Format(time.RFC3339Nano),
		"snapshotAt": formatTimePtr(snapAt),
		"data":       n,
	})
}

func (s *Server) handleListVolumes(w http.ResponseWriter, r *http.Request) {
	if s.Volumes == nil {
		writeErr(w, http.StatusServiceUnavailable, "not_ready", "volumes not initialized")
		return
	}
	res := s.Volumes.List(r.URL.Query().Get("stack"), r.URL.Query().Get("q"))
	writeJSON(w, http.StatusOK, map[string]any{
		"timestamp":     time.Now().UTC().Format(time.RFC3339Nano),
		"snapshotAt":    formatTimePtr(res.SnapshotAt),
		"systemAt":      formatTimePtr(res.SystemAt),
		"snapshotAgeMs": res.SnapshotAge.Milliseconds(),
		"data":          res.Volumes,
	})
}

func (s *Server) handleGetVolume(w http.ResponseWriter, r *http.Request) {
	if s.Volumes == nil {
		writeErr(w, http.StatusServiceUnavailable, "not_ready", "volumes not initialized")
		return
	}
	v, systemAt, ok := s.Volumes.Get(r.PathValue("name"))
	if !ok {
		writeErr(w, http.StatusNotFound, "not_found", "volume not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"timestamp": time.Now().UTC().Format(time.RFC3339Nano),
		"systemAt":  formatTimePtr(systemAt),
		"data":      v,
	})
}

func (s *Server) handleListImages(w http.ResponseWriter, r *http.Request) {
	if s.Images == nil {
		writeErr(w, http.StatusServiceUnavailable, "not_ready", "images not initialized")
		return
	}
	res := s.Images.List(r.URL.Query().Get("q"), r.URL.Query().Get("dangling"))
	writeJSON(w, http.StatusOK, map[string]any{
		"timestamp":     time.Now().UTC().Format(time.RFC3339Nano),
		"snapshotAt":    formatTimePtr(res.SnapshotAt),
		"snapshotAgeMs": res.SnapshotAge.Milliseconds(),
		"data":          res.Images,
	})
}

func (s *Server) handleGetImage(w http.ResponseWriter, r *http.Request) {
	if s.Images == nil {
		writeErr(w, http.StatusServiceUnavailable, "not_ready", "images not initialized")
		return
	}
	img, snapAt, ok := s.Images.Get(r.PathValue("id"))
	if !ok {
		writeErr(w, http.StatusNotFound, "not_found", "image not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"timestamp":  time.Now().UTC().Format(time.RFC3339Nano),
		"snapshotAt": formatTimePtr(snapAt),
		"data":       img,
	})
}

func (s *Server) handleSystemDF(w http.ResponseWriter, r *http.Request) {
	if s.System == nil {
		writeErr(w, http.StatusServiceUnavailable, "not_ready", "system not initialized")
		return
	}
	res := s.System.DiskUsage()
	writeJSON(w, http.StatusOK, map[string]any{
		"timestamp":  time.Now().UTC().Format(time.RFC3339Nano),
		"snapshotAt": formatTimePtr(res.SnapshotAt),
		"systemAt":   formatTimePtr(res.SystemAt),
		"data":       res.DiskUsage,
	})
}

func (s *Server) handleSystemResources(w http.ResponseWriter, r *http.Request) {
	if s.System == nil {
		writeErr(w, http.StatusServiceUnavailable, "not_ready", "system not initialized")
		return
	}
	res := s.System.Resources()
	writeJSON(w, http.StatusOK, map[string]any{
		"timestamp":  time.Now().UTC().Format(time.RFC3339Nano),
		"snapshotAt": formatTimePtr(res.SnapshotAt),
		"statsAt":    formatTimePtr(res.StatsAt),
		"systemAt":   formatTimePtr(res.SystemAt),
		"data":       res.Resources,
	})
}

func (s *Server) handleSystemInfo(w http.ResponseWriter, r *http.Request) {
	if s.System == nil {
		writeErr(w, http.StatusServiceUnavailable, "not_ready", "system not initialized")
		return
	}
	res := s.System.Info()
	if res.Info == nil {
		writeErr(w, http.StatusServiceUnavailable, "not_ready", "system info not collected yet")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"timestamp":  time.Now().UTC().Format(time.RFC3339Nano),
		"snapshotAt": formatTimePtr(res.SnapshotAt),
		"systemAt":   formatTimePtr(res.SystemAt),
		"data":       res.Info,
	})
}

func (s *Server) handleSystemSettings(w http.ResponseWriter, r *http.Request) {
	if s.Diagnostics == nil {
		writeErr(w, http.StatusServiceUnavailable, "not_ready", "settings not initialized")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"timestamp": time.Now().UTC().Format(time.RFC3339Nano),
		"data":      s.Diagnostics.Settings(),
	})
}

func (s *Server) handleDiagnostics(w http.ResponseWriter, r *http.Request) {
	if !IsLoopbackAddr(r.RemoteAddr) {
		writeErr(w, http.StatusForbidden, "forbidden", "diagnostics is available from localhost only")
		return
	}
	if s.Diagnostics == nil {
		writeErr(w, http.StatusServiceUnavailable, "not_ready", "diagnostics not initialized")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"timestamp": time.Now().UTC().Format(time.RFC3339Nano),
		"data":      s.Diagnostics.Get(),
	})
}

func writeErr(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{
		"error": map[string]any{
			"code":    code,
			"message": message,
		},
		"timestamp": time.Now().UTC().Format(time.RFC3339Nano),
	})
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func formatTimePtr(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t.UTC().Format(time.RFC3339Nano)
}

func parseBoolDefaultTrue(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "0", "false", "no", "off":
		return false
	default:
		return true
	}
}

func parseBoolDefaultFalse(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(true)
	_ = enc.Encode(v)
}
