package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/epm-games/docker-visualizer/internal/commands"
	"github.com/epm-games/docker-visualizer/internal/provenance"
)

func (s *Server) handleListCommandDefs(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"timestamp": time.Now().UTC().Format(time.RFC3339Nano),
		"data":      commands.Registry(),
	})
}

func (s *Server) handleGetCommandDef(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	def, ok := commands.Lookup(id)
	if !ok {
		writeErr(w, http.StatusNotFound, "not_found", "command not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"timestamp": time.Now().UTC().Format(time.RFC3339Nano),
		"data":      def,
	})
}

func (s *Server) handleEntityCommands(w http.ResponseWriter, r *http.Request) {
	rt := s.runtime(w, r)
	if rt == nil {
		return
	}
	if rt.Commands == nil {
		writeErr(w, http.StatusServiceUnavailable, "not_ready", "commands service unavailable")
		return
	}
	kind := commands.EntityKind(strings.TrimSpace(r.PathValue("kind")))
	ref := strings.TrimSpace(r.PathValue("ref"))
	if q := strings.TrimSpace(r.URL.Query().Get("ref")); q != "" {
		ref = q
	}
	shell := commands.Shell(strings.TrimSpace(r.URL.Query().Get("shell")))
	switch kind {
	case commands.EntityContainer, commands.EntityNetwork, commands.EntityVolume,
		commands.EntityImage, commands.EntitySystem, commands.EntityStack:
	default:
		writeErr(w, http.StatusBadRequest, "bad_request", "unknown entity kind")
		return
	}
	if kind != commands.EntitySystem && ref == "" {
		writeErr(w, http.StatusBadRequest, "bad_request", "entity ref required")
		return
	}
	data := rt.Commands.ForEntity(kind, ref, shell)
	writeJSON(w, http.StatusOK, map[string]any{
		"timestamp": time.Now().UTC().Format(time.RFC3339Nano),
		"host":      rt.Name,
		"context":   rt.Commands.ConnectionContext(),
		"data":      data,
	})
}

func (s *Server) handleListFindings(w http.ResponseWriter, r *http.Request) {
	rt := s.runtime(w, r)
	if rt == nil {
		return
	}
	if rt.Findings == nil {
		writeErr(w, http.StatusServiceUnavailable, "not_ready", "findings unavailable")
		return
	}
	data := rt.Findings.List()
	writeJSON(w, http.StatusOK, map[string]any{
		"timestamp": time.Now().UTC().Format(time.RFC3339Nano),
		"host":      rt.Name,
		"count":     len(data),
		"data":      data,
	})
}

func (s *Server) handleGetFinding(w http.ResponseWriter, r *http.Request) {
	rt := s.runtime(w, r)
	if rt == nil {
		return
	}
	f, ok := rt.Findings.Get(r.PathValue("id"))
	if !ok {
		writeErr(w, http.StatusNotFound, "not_found", "finding not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"timestamp": time.Now().UTC().Format(time.RFC3339Nano),
		"host":      rt.Name,
		"data":      f,
	})
}

func (s *Server) handleListProvenance(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"timestamp": time.Now().UTC().Format(time.RFC3339Nano),
		"data":      provenance.Catalog(),
	})
}

func (s *Server) handleGetProvenance(w http.ResponseWriter, r *http.Request) {
	spec, ok := provenance.Get(r.PathValue("id"))
	if !ok {
		writeErr(w, http.StatusNotFound, "not_found", "provenance not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"timestamp": time.Now().UTC().Format(time.RFC3339Nano),
		"data":      spec,
	})
}

func (s *Server) handleListSnapshots(w http.ResponseWriter, r *http.Request) {
	rt := s.runtime(w, r)
	if rt == nil {
		return
	}
	if rt.Snapshots == nil || rt.Snapshots.Disk == nil {
		writeErr(w, http.StatusServiceUnavailable, "snapshots_disabled", "snapshots storage is disabled")
		return
	}
	list, err := rt.Snapshots.List()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"timestamp": time.Now().UTC().Format(time.RFC3339Nano),
		"host":      rt.Name,
		"data":      list,
	})
}

func (s *Server) handleCreateSnapshot(w http.ResponseWriter, r *http.Request) {
	rt := s.runtime(w, r)
	if rt == nil {
		return
	}
	if rt.Snapshots == nil || rt.Snapshots.Disk == nil {
		writeErr(w, http.StatusServiceUnavailable, "snapshots_disabled", "snapshots storage is disabled")
		return
	}
	var body struct {
		Label string `json:"label"`
	}
	if r.Body != nil && r.ContentLength != 0 {
		_ = json.NewDecoder(r.Body).Decode(&body)
	}
	snap, err := rt.Snapshots.Create(body.Label)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"timestamp": time.Now().UTC().Format(time.RFC3339Nano),
		"host":      rt.Name,
		"data":      snap,
	})
}

func (s *Server) handleGetSnapshot(w http.ResponseWriter, r *http.Request) {
	rt := s.runtime(w, r)
	if rt == nil {
		return
	}
	if rt.Snapshots == nil || rt.Snapshots.Disk == nil {
		writeErr(w, http.StatusServiceUnavailable, "snapshots_disabled", "snapshots storage is disabled")
		return
	}
	snap, err := rt.Snapshots.Get(r.PathValue("id"))
	if err != nil {
		writeErr(w, http.StatusNotFound, "not_found", "snapshot not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"timestamp": time.Now().UTC().Format(time.RFC3339Nano),
		"host":      rt.Name,
		"data":      snap,
	})
}

func (s *Server) handleSnapshotDiff(w http.ResponseWriter, r *http.Request) {
	rt := s.runtime(w, r)
	if rt == nil {
		return
	}
	if rt.Snapshots == nil || rt.Snapshots.Disk == nil {
		writeErr(w, http.StatusServiceUnavailable, "snapshots_disabled", "snapshots storage is disabled")
		return
	}
	right := r.URL.Query().Get("against")
	if right == "" {
		right = "current"
	}
	diff, err := rt.Snapshots.Diff(r.PathValue("id"), right)
	if err != nil {
		writeErr(w, http.StatusNotFound, "not_found", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"timestamp": time.Now().UTC().Format(time.RFC3339Nano),
		"host":      rt.Name,
		"data":      diff,
	})
}

func (s *Server) handleDeleteSnapshot(w http.ResponseWriter, r *http.Request) {
	rt := s.runtime(w, r)
	if rt == nil {
		return
	}
	if rt.Snapshots == nil || rt.Snapshots.Disk == nil {
		writeErr(w, http.StatusServiceUnavailable, "snapshots_disabled", "snapshots storage is disabled")
		return
	}
	if err := rt.Snapshots.Delete(r.PathValue("id")); err != nil {
		writeErr(w, http.StatusNotFound, "not_found", "snapshot not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"timestamp": time.Now().UTC().Format(time.RFC3339Nano),
		"ok":        true,
	})
}
