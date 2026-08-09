package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/epm-games/docker-visualizer/internal/app"
	"github.com/epm-games/docker-visualizer/internal/store"
)

func TestSystemSettings(t *testing.T) {
	st := store.New()
	s := testServer(t, st)
	s.Diagnostics = &app.DiagnosticsService{
		Listen:        "127.0.0.1:8080",
		Version:       "test",
		Commit:        "abc",
		DockerTimeout: "5s",
		AuthEnabled:   false,
		Intervals: map[string]string{
			"inventory": "10s",
			"stats":     "1s",
			"system":    "15s",
		},
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
	defaults := data["defaults"].(map[string]any)
	if defaults["inspectRedact"] != true {
		t.Fatalf("defaults=%v", defaults)
	}
}
