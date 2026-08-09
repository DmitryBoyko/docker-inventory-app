package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/epm-games/docker-visualizer/internal/app"
	"github.com/epm-games/docker-visualizer/internal/metricsdb"
	"github.com/epm-games/docker-visualizer/internal/store"
)

func TestMetricsHistory(t *testing.T) {
	st := store.New()
	s := testServer(t, st)
	db, err := metricsdb.Open(metricsdb.Options{
		Path:           filepath.Join(t.TempDir(), "h.db"),
		Retention:      time.Hour,
		SampleInterval: time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	rt, _ := s.Hosts.Get("")
	rt.Metrics = &app.MetricsService{DB: db, Host: "default"}
	s.MetricsEnabled = true

	base := time.Now().UTC().Add(-5 * time.Minute)
	for i := 0; i < 3; i++ {
		_ = db.Record("default", base.Add(time.Duration(i)*time.Second), []metricsdb.ContainerSample{
			{ID: "c1", Name: "web", CPU: float64(i + 1), Mem: 1000},
		})
	}

	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/v1/metrics/history?scope=host", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	data := body["data"].(map[string]any)
	pts := data["points"].([]any)
	if len(pts) < 3 {
		t.Fatalf("points=%v", pts)
	}
}

func TestMetricsHistory_Disabled(t *testing.T) {
	s := testServer(t, store.New())
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/v1/metrics/history", nil))
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d", rr.Code)
	}
}
