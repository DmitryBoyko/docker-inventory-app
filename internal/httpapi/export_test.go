package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/epm-games/docker-visualizer/internal/domain"
	"github.com/epm-games/docker-visualizer/internal/store"
)

func TestExportEndpoint(t *testing.T) {
	st := store.New()
	st.Replace([]domain.Container{
		{
			ID: "abcdabcdabcd", IDShort: "abcdabcdabcd", Name: "c1",
			Stack: "s", State: domain.ContainerStateRunning, Health: domain.HealthNone,
			WritableLayer: domain.AvailableBytes(1),
		},
	}, time.Now().UTC(), "")
	s := testServer(t, st)

	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/v1/export?format=json&scope=all", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Header().Get("Content-Disposition"), ".json") {
		t.Fatalf("disp=%s", rr.Header().Get("Content-Disposition"))
	}
	if !strings.Contains(rr.Body.String(), "c1") {
		t.Fatalf("body=%s", rr.Body.String())
	}

	rr2 := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr2, httptest.NewRequest(http.MethodGet, "/api/v1/export?format=csv&scope=containers", nil))
	if rr2.Code != http.StatusOK || !strings.Contains(rr2.Body.String(), "c1") {
		t.Fatalf("csv status=%d body=%s", rr2.Code, rr2.Body.String())
	}
}
