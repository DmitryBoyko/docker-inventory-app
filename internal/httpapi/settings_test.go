package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/epm-games/docker-visualizer/internal/store"
)

func TestSystemSettings(t *testing.T) {
	st := store.New()
	s := testServer(t, st)
	s.Listen = "127.0.0.1:8080"
	s.Version = "test"
	s.Commit = "abc"
	s.DockerTimeout = "5s"
	s.AuthEnabled = false
	s.Intervals = map[string]string{
		"inventory": "10s",
		"stats":     "1s",
		"system":    "15s",
	}
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/v1/system/settings", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	data := body["data"].(map[string]any)
	if data["listen"] != "127.0.0.1:8080" {
		t.Fatalf("%v", data)
	}
	if data["authEnabled"] != false {
		t.Fatalf("authEnabled=%v", data["authEnabled"])
	}
	if data["defaultHost"] != "default" {
		t.Fatalf("defaultHost=%v", data["defaultHost"])
	}
	defaults := data["defaults"].(map[string]any)
	if defaults["inspectRedact"] != true {
		t.Fatalf("defaults=%v", defaults)
	}
}

func TestListHosts(t *testing.T) {
	s := testServer(t, store.New())
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/v1/hosts", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["defaultHost"] != "default" {
		t.Fatalf("%v", body)
	}
	data := body["data"].([]any)
	if len(data) != 1 {
		t.Fatalf("hosts=%v", data)
	}
}

func TestUnknownHostQuery(t *testing.T) {
	s := testServer(t, store.New())
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/v1/containers?host=nope", nil))
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}
