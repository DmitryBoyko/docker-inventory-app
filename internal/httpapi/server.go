package httpapi

import (
	"encoding/json"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/epm-games/docker-visualizer/internal/app"
	"github.com/epm-games/docker-visualizer/internal/hosts"
	"github.com/epm-games/docker-visualizer/internal/uiembed"
	"github.com/epm-games/docker-visualizer/internal/ws"
)

// Server is the HTTP API surface (multi-host via Hosts registry, ADR-014).
type Server struct {
	Hosts         *hosts.Registry
	Hub           *ws.Hub
	Version       string
	Commit        string
	Listen        string
	AuthEnabled   bool
	DockerTimeout string
	Intervals     map[string]string
	StartedAt     time.Time
}

// Handler returns the root mux. API lives under /api/v1.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/health", s.handleHealth)
	mux.HandleFunc("GET /api/v1/ready", s.handleReady)
	mux.HandleFunc("GET /health", s.handleHealth)
	mux.HandleFunc("GET /ready", s.handleReady)

	mux.HandleFunc("GET /api/v1/hosts", s.handleListHosts)

	mux.HandleFunc("GET /api/v1/containers", s.handleListContainers)
	mux.HandleFunc("GET /api/v1/containers/{id}/stats", s.handleGetContainerStats)
	mux.HandleFunc("GET /api/v1/containers/{id}/inspect", s.handleGetContainerInspect)
	mux.HandleFunc("GET /api/v1/containers/{id}/logs/ws", s.handleContainerLogsWS)
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
	mux.HandleFunc("GET /api/v1/export", s.handleExport)

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

func (s *Server) handleListHosts(w http.ResponseWriter, r *http.Request) {
	if s.Hosts == nil {
		writeErr(w, http.StatusServiceUnavailable, "not_ready", "hosts not initialized")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"timestamp":   time.Now().UTC().Format(time.RFC3339Nano),
		"defaultHost": s.Hosts.Default,
		"data":        s.Hosts.List(),
	})
}

func (s *Server) handleReady(w http.ResponseWriter, r *http.Request) {
	rt := s.runtime(w, r)
	if rt == nil {
		return
	}
	st := rt.Docker.Ping(r.Context())
	st.Name = rt.Name
	ts := time.Now().UTC().Format(time.RFC3339Nano)
	eventsConnected := false
	eventsErr := ""
	if rt.Events != nil {
		eventsConnected = rt.Events.Connected()
		eventsErr = rt.Events.LastError()
	}
	eventsPayload := map[string]any{
		"connected": eventsConnected,
		"error":     nullIfEmpty(eventsErr),
		"host":      rt.Name,
	}
	if !st.Connected {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{
			"error": map[string]any{
				"code":    "docker_unavailable",
				"message": st.Error,
				"details": map[string]any{
					"name":    rt.Name,
					"host":    st.Host,
					"source":  st.Source,
					"context": st.Context,
				},
			},
			"ready":     false,
			"host":      rt.Name,
			"docker":    st,
			"events":    eventsPayload,
			"timestamp": ts,
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ready":     true,
		"host":      rt.Name,
		"docker":    st,
		"events":    eventsPayload,
		"timestamp": ts,
	})
}

func (s *Server) handleListContainers(w http.ResponseWriter, r *http.Request) {
	rt := s.runtime(w, r)
	if rt == nil {
		return
	}
	q := app.ContainerQuery{
		Stack:  r.URL.Query().Get("stack"),
		State:  r.URL.Query().Get("state"),
		Health: r.URL.Query().Get("health"),
		Q:      r.URL.Query().Get("q"),
	}
	res := rt.Containers.List(q)
	writeJSON(w, http.StatusOK, map[string]any{
		"timestamp":     time.Now().UTC().Format(time.RFC3339Nano),
		"host":          rt.Name,
		"snapshotAt":    formatTimePtr(res.SnapshotAt),
		"snapshotAgeMs": res.SnapshotAge.Milliseconds(),
		"collectError":  nullIfEmpty(res.CollectErr),
		"data":          res.Containers,
	})
}

func (s *Server) handleGetContainer(w http.ResponseWriter, r *http.Request) {
	rt := s.runtime(w, r)
	if rt == nil {
		return
	}
	c, snapAt, ok := rt.Containers.Get(r.PathValue("id"))
	if !ok {
		writeErr(w, http.StatusNotFound, "not_found", "container not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"timestamp":  time.Now().UTC().Format(time.RFC3339Nano),
		"host":       rt.Name,
		"snapshotAt": formatTimePtr(snapAt),
		"data":       c,
	})
}

func (s *Server) handleGetContainerStats(w http.ResponseWriter, r *http.Request) {
	rt := s.runtime(w, r)
	if rt == nil {
		return
	}
	res := rt.Containers.GetStats(r.PathValue("id"))
	if !res.Found {
		writeErr(w, http.StatusNotFound, "not_found", "container not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"timestamp":  time.Now().UTC().Format(time.RFC3339Nano),
		"host":       rt.Name,
		"snapshotAt": formatTimePtr(res.SnapshotAt),
		"statsAt":    formatTimePtr(res.StatsAt),
		"running":    res.Running,
		"data":       res.Stats,
	})
}

func (s *Server) handleGetContainerInspect(w http.ResponseWriter, r *http.Request) {
	rt := s.runtime(w, r)
	if rt == nil {
		return
	}
	doRedact := parseBoolDefaultTrue(r.URL.Query().Get("redact"))
	res, err := rt.Live.Inspect(r.Context(), r.PathValue("id"), doRedact)
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
		"host":      rt.Name,
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
	rt := s.runtime(w, r)
	if rt == nil {
		return
	}
	q := r.URL.Query()
	tail := 0
	if v := q.Get("tail"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			tail = n
		}
	}
	res, err := rt.Live.Logs(r.Context(), r.PathValue("id"), app.LogsOptions{
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
		"host":      rt.Name,
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
	rt := s.runtime(w, r)
	if rt == nil {
		return
	}
	res := rt.Stacks.List()
	writeJSON(w, http.StatusOK, map[string]any{
		"timestamp":     time.Now().UTC().Format(time.RFC3339Nano),
		"host":          rt.Name,
		"snapshotAt":    formatTimePtr(res.SnapshotAt),
		"statsAt":       formatTimePtr(res.StatsAt),
		"systemAt":      formatTimePtr(res.SystemAt),
		"snapshotAgeMs": res.SnapshotAge.Milliseconds(),
		"data":          res.Stacks,
	})
}

func (s *Server) handleGetStack(w http.ResponseWriter, r *http.Request) {
	rt := s.runtime(w, r)
	if rt == nil {
		return
	}
	st, meta, ok := rt.Stacks.Get(r.PathValue("name"))
	if !ok {
		writeErr(w, http.StatusNotFound, "not_found", "stack not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"timestamp":  time.Now().UTC().Format(time.RFC3339Nano),
		"host":       rt.Name,
		"snapshotAt": formatTimePtr(meta.SnapshotAt),
		"statsAt":    formatTimePtr(meta.StatsAt),
		"systemAt":   formatTimePtr(meta.SystemAt),
		"data":       st,
	})
}

func (s *Server) handleStackVolumes(w http.ResponseWriter, r *http.Request) {
	rt := s.runtime(w, r)
	if rt == nil {
		return
	}
	vols, ok := rt.Stacks.StackVolumes(r.PathValue("name"))
	if !ok {
		writeErr(w, http.StatusNotFound, "not_found", "stack not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"timestamp": time.Now().UTC().Format(time.RFC3339Nano),
		"host":      rt.Name,
		"data":      vols,
	})
}

func (s *Server) handleGraph(w http.ResponseWriter, r *http.Request) {
	rt := s.runtime(w, r)
	if rt == nil {
		return
	}
	res, err := rt.Graph.Get(r.URL.Query().Get("scope"), r.URL.Query().Get("stack"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"timestamp":     time.Now().UTC().Format(time.RFC3339Nano),
		"host":          rt.Name,
		"snapshotAt":    formatTimePtr(res.SnapshotAt),
		"statsAt":       formatTimePtr(res.StatsAt),
		"snapshotAgeMs": res.SnapshotAge.Milliseconds(),
		"data":          res.Graph,
	})
}

func (s *Server) handleListNetworks(w http.ResponseWriter, r *http.Request) {
	rt := s.runtime(w, r)
	if rt == nil {
		return
	}
	res := rt.Networks.List(r.URL.Query().Get("q"), r.URL.Query().Get("driver"))
	writeJSON(w, http.StatusOK, map[string]any{
		"timestamp":     time.Now().UTC().Format(time.RFC3339Nano),
		"host":          rt.Name,
		"snapshotAt":    formatTimePtr(res.SnapshotAt),
		"snapshotAgeMs": res.SnapshotAge.Milliseconds(),
		"data":          res.Networks,
	})
}

func (s *Server) handleGetNetwork(w http.ResponseWriter, r *http.Request) {
	rt := s.runtime(w, r)
	if rt == nil {
		return
	}
	n, snapAt, ok := rt.Networks.Get(r.PathValue("id"))
	if !ok {
		writeErr(w, http.StatusNotFound, "not_found", "network not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"timestamp":  time.Now().UTC().Format(time.RFC3339Nano),
		"host":       rt.Name,
		"snapshotAt": formatTimePtr(snapAt),
		"data":       n,
	})
}

func (s *Server) handleListVolumes(w http.ResponseWriter, r *http.Request) {
	rt := s.runtime(w, r)
	if rt == nil {
		return
	}
	res := rt.Volumes.List(r.URL.Query().Get("stack"), r.URL.Query().Get("q"))
	writeJSON(w, http.StatusOK, map[string]any{
		"timestamp":     time.Now().UTC().Format(time.RFC3339Nano),
		"host":          rt.Name,
		"snapshotAt":    formatTimePtr(res.SnapshotAt),
		"systemAt":      formatTimePtr(res.SystemAt),
		"snapshotAgeMs": res.SnapshotAge.Milliseconds(),
		"data":          res.Volumes,
	})
}

func (s *Server) handleGetVolume(w http.ResponseWriter, r *http.Request) {
	rt := s.runtime(w, r)
	if rt == nil {
		return
	}
	v, systemAt, ok := rt.Volumes.Get(r.PathValue("name"))
	if !ok {
		writeErr(w, http.StatusNotFound, "not_found", "volume not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"timestamp": time.Now().UTC().Format(time.RFC3339Nano),
		"host":      rt.Name,
		"systemAt":  formatTimePtr(systemAt),
		"data":      v,
	})
}

func (s *Server) handleListImages(w http.ResponseWriter, r *http.Request) {
	rt := s.runtime(w, r)
	if rt == nil {
		return
	}
	res := rt.Images.List(r.URL.Query().Get("q"), r.URL.Query().Get("dangling"))
	writeJSON(w, http.StatusOK, map[string]any{
		"timestamp":     time.Now().UTC().Format(time.RFC3339Nano),
		"host":          rt.Name,
		"snapshotAt":    formatTimePtr(res.SnapshotAt),
		"snapshotAgeMs": res.SnapshotAge.Milliseconds(),
		"data":          res.Images,
	})
}

func (s *Server) handleGetImage(w http.ResponseWriter, r *http.Request) {
	rt := s.runtime(w, r)
	if rt == nil {
		return
	}
	img, snapAt, ok := rt.Images.Get(r.PathValue("id"))
	if !ok {
		writeErr(w, http.StatusNotFound, "not_found", "image not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"timestamp":  time.Now().UTC().Format(time.RFC3339Nano),
		"host":       rt.Name,
		"snapshotAt": formatTimePtr(snapAt),
		"data":       img,
	})
}

func (s *Server) handleSystemDF(w http.ResponseWriter, r *http.Request) {
	rt := s.runtime(w, r)
	if rt == nil {
		return
	}
	res := rt.System.DiskUsage()
	writeJSON(w, http.StatusOK, map[string]any{
		"timestamp":  time.Now().UTC().Format(time.RFC3339Nano),
		"host":       rt.Name,
		"snapshotAt": formatTimePtr(res.SnapshotAt),
		"systemAt":   formatTimePtr(res.SystemAt),
		"data":       res.DiskUsage,
	})
}

func (s *Server) handleSystemResources(w http.ResponseWriter, r *http.Request) {
	rt := s.runtime(w, r)
	if rt == nil {
		return
	}
	res := rt.System.Resources()
	writeJSON(w, http.StatusOK, map[string]any{
		"timestamp":  time.Now().UTC().Format(time.RFC3339Nano),
		"host":       rt.Name,
		"snapshotAt": formatTimePtr(res.SnapshotAt),
		"statsAt":    formatTimePtr(res.StatsAt),
		"systemAt":   formatTimePtr(res.SystemAt),
		"data":       res.Resources,
	})
}

func (s *Server) handleSystemInfo(w http.ResponseWriter, r *http.Request) {
	rt := s.runtime(w, r)
	if rt == nil {
		return
	}
	res := rt.System.Info()
	if res.Info == nil {
		writeErr(w, http.StatusServiceUnavailable, "not_ready", "system info not collected yet")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"timestamp":  time.Now().UTC().Format(time.RFC3339Nano),
		"host":       rt.Name,
		"snapshotAt": formatTimePtr(res.SnapshotAt),
		"systemAt":   formatTimePtr(res.SystemAt),
		"data":       res.Info,
	})
}

func (s *Server) handleExport(w http.ResponseWriter, r *http.Request) {
	rt := s.runtime(w, r)
	if rt == nil {
		return
	}
	res, err := rt.Export.Export(r.URL.Query().Get("format"), r.URL.Query().Get("scope"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	w.Header().Set("Content-Type", res.ContentType)
	w.Header().Set("Content-Disposition", res.ContentDisposition)
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Docker-Host", rt.Name)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(res.Body)
}

func (s *Server) handleSystemSettings(w http.ResponseWriter, r *http.Request) {
	intervals := s.Intervals
	if intervals == nil {
		intervals = map[string]string{}
	}
	var hostList []hosts.Info
	defaultHost := ""
	if s.Hosts != nil {
		hostList = s.Hosts.List()
		defaultHost = s.Hosts.Default
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"timestamp": time.Now().UTC().Format(time.RFC3339Nano),
		"data": map[string]any{
			"listen":         s.Listen,
			"listenLoopback": isListenLoopbackHTTP(s.Listen),
			"authEnabled":    s.AuthEnabled,
			"dockerTimeout":  s.DockerTimeout,
			"intervals":      intervals,
			"version":        s.Version,
			"commit":         s.Commit,
			"uiEmbedded":     uiembed.Available(),
			"defaultHost":    defaultHost,
			"hosts":          hostList,
			"defaults":       map[string]any{"inspectRedact": true},
		},
	})
}

func (s *Server) handleDiagnostics(w http.ResponseWriter, r *http.Request) {
	if !IsLoopbackAddr(r.RemoteAddr) {
		writeErr(w, http.StatusForbidden, "forbidden", "diagnostics is available from localhost only")
		return
	}
	rt := s.runtime(w, r)
	if rt == nil {
		return
	}
	diag := &app.DiagnosticsService{
		Store:         rt.Store,
		Docker:        rt.Docker,
		Hub:           s.Hub,
		Events:        rt.Events,
		Health:        rt.Health,
		Version:       s.Version,
		Commit:        s.Commit,
		Listen:        s.Listen,
		Intervals:     s.Intervals,
		DockerTimeout: s.DockerTimeout,
		AuthEnabled:   s.AuthEnabled,
		StartedAt:     s.StartedAt,
	}
	data := diag.Get()
	writeJSON(w, http.StatusOK, map[string]any{
		"timestamp": time.Now().UTC().Format(time.RFC3339Nano),
		"host":      rt.Name,
		"hosts":     s.Hosts.List(),
		"data":      data,
	})
}

func isListenLoopbackHTTP(addr string) bool {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		host = addr
	}
	switch host {
	case "127.0.0.1", "::1", "localhost":
		return true
	default:
		return false
	}
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
