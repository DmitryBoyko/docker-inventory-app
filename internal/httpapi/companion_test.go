package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/epm-games/docker-visualizer/internal/app"
	"github.com/epm-games/docker-visualizer/internal/commands"
	"github.com/epm-games/docker-visualizer/internal/domain"
	"github.com/epm-games/docker-visualizer/internal/hosts"
	"github.com/epm-games/docker-visualizer/internal/store"
)

func TestEntityCommandsAndDiagnostics(t *testing.T) {
	st := store.New()
	st.Replace([]domain.Container{{
		ID: "abc1234567890", IDShort: "abc123456789", Name: "nginx-prod",
		State: domain.ContainerStateRunning, Health: domain.HealthUnhealthy, RestartCount: 14,
	}}, time.Now().UTC(), "")

	rt := &hosts.Runtime{
		Name:     "default",
		Store:    st,
		Commands: &app.CommandsService{Store: st, Host: "default"},
		Findings: &app.FindingsService{Store: st},
	}
	srv := &Server{Hosts: hosts.ForTest("default", rt)}
	h := srv.Handler()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/entities/container/commands?ref=nginx-prod&shell=bash", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("commands status=%d body=%s", rec.Code, rec.Body.String())
	}
	var cmdBody struct {
		Data []commands.Rendered `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &cmdBody); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, c := range cmdBody.Data {
		if c.DefinitionID == "container.inspect" && c.Command == "docker inspect nginx-prod" {
			found = true
		}
	}
	if !found {
		t.Fatalf("inspect missing: %+v", cmdBody.Data)
	}

	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/diagnostics", nil)
	rec2 := httptest.NewRecorder()
	h.ServeHTTP(rec2, req2)
	if rec2.Code != 200 {
		t.Fatalf("diagnostics status=%d", rec2.Code)
	}
	var findBody struct {
		Count int `json:"count"`
	}
	_ = json.Unmarshal(rec2.Body.Bytes(), &findBody)
	if findBody.Count < 1 {
		t.Fatalf("expected findings, got %d", findBody.Count)
	}

	req3 := httptest.NewRequest(http.MethodGet, "/api/v1/provenance/container.cpuPercent", nil)
	rec3 := httptest.NewRecorder()
	h.ServeHTTP(rec3, req3)
	if rec3.Code != 200 {
		t.Fatalf("provenance status=%d", rec3.Code)
	}
}
