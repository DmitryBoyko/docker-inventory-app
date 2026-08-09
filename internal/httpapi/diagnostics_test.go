package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/epm-games/docker-visualizer/internal/app"
	"github.com/epm-games/docker-visualizer/internal/domain"
	"github.com/epm-games/docker-visualizer/internal/observability"
	"github.com/epm-games/docker-visualizer/internal/store"
)

func TestDiagnostics_LocalhostOnly(t *testing.T) {
	st := store.New()
	st.Replace([]domain.Container{
		{ID: "abc", IDShort: "abc", Name: "c1", Stack: "s", State: domain.ContainerStateRunning},
	}, time.Now().UTC(), "")
	health := observability.NewRegistry()
	health.RecordSuccess("inventory", time.Millisecond)

	s := testServer(t, st)
	s.Diagnostics = &app.DiagnosticsService{
		Store:         st,
		Health:        health,
		Version:       "test",
		Commit:        "deadbeef",
		Listen:        "127.0.0.1:8080",
		Intervals:     map[string]string{"inventory": "10s"},
		DockerTimeout: "5s",
		AuthEnabled:   true,
		StartedAt:     time.Now().UTC().Add(-time.Minute),
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/system/diagnostics", nil)
	req.RemoteAddr = "127.0.0.1:54321"
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	data, ok := body["data"].(map[string]any)
	if !ok {
		t.Fatalf("data=%T", body["data"])
	}
	if data["listenLoopback"] != true {
		t.Fatalf("listenLoopback=%v", data["listenLoopback"])
	}
	if data["authEnabled"] != true {
		t.Fatalf("authEnabled=%v", data["authEnabled"])
	}
	snap, _ := data["snapshot"].(map[string]any)
	if snap["containerCount"].(float64) != 1 {
		t.Fatalf("snapshot=%v", snap)
	}

	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/system/diagnostics", nil)
	req2.RemoteAddr = "203.0.113.9:9999"
	rr2 := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rr2.Code)
	}
}
