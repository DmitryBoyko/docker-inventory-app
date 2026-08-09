package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/epm-games/docker-visualizer/internal/domain"
	"github.com/epm-games/docker-visualizer/internal/store"
)

func TestLoad_ListContainersAndStacks200(t *testing.T) {
	const n = 200
	st := store.New()
	containers := make([]domain.Container, 0, n)
	vols := make([]domain.Volume, 0, 20)
	volSeen := map[string]bool{}
	for i := 0; i < n; i++ {
		stack := fmt.Sprintf("stack-%02d", i%10)
		svc := fmt.Sprintf("svc-%d", i%3)
		vol := fmt.Sprintf("v-%d", i%20)
		if !volSeen[vol] {
			volSeen[vol] = true
			vols = append(vols, domain.Volume{
				Name: vol, Driver: "local",
				Usage: domain.VolumeUsage{ByteMetric: domain.AvailableBytes(1024)},
			})
		}
		id := fmt.Sprintf("%064x", i+1)
		containers = append(containers, domain.Container{
			ID: id, IDShort: domain.ShortID(id), Name: fmt.Sprintf("c-%03d", i),
			Stack: stack, Service: &svc, State: domain.ContainerStateRunning,
			Health: domain.HealthHealthy, WritableLayer: domain.AvailableBytes(10),
			Mounts: []domain.Mount{{Type: domain.MountTypeVolume, Name: vol}},
			Stats:  &domain.ContainerStats{CPUPercent: 1, MemoryBytes: 1024},
		})
	}
	st.Replace(containers, time.Now().UTC(), "")
	st.MergeSystem(vols, nil, nil, time.Now().UTC())

	s := testServer(t, st)
	h := s.Handler()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/containers", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("containers status=%d", rr.Code)
	}
	var list map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &list); err != nil {
		t.Fatal(err)
	}
	data, _ := list["data"].([]any)
	if len(data) != n {
		t.Fatalf("containers=%d", len(data))
	}

	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/stacks", nil)
	rr2 := httptest.NewRecorder()
	h.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusOK {
		t.Fatalf("stacks status=%d", rr2.Code)
	}
	var stacksBody map[string]any
	if err := json.Unmarshal(rr2.Body.Bytes(), &stacksBody); err != nil {
		t.Fatal(err)
	}
	stacks, _ := stacksBody["data"].([]any)
	if len(stacks) != 10 {
		t.Fatalf("stacks=%d", len(stacks))
	}

	req3 := httptest.NewRequest(http.MethodGet, "/api/v1/graph?scope=all", nil)
	rr3 := httptest.NewRecorder()
	h.ServeHTTP(rr3, req3)
	if rr3.Code != http.StatusOK {
		t.Fatalf("graph status=%d body=%s", rr3.Code, rr3.Body.String())
	}
}

func BenchmarkHTTPListContainers200(b *testing.B) {
	const n = 200
	st := store.New()
	containers := make([]domain.Container, 0, n)
	for i := 0; i < n; i++ {
		id := fmt.Sprintf("%064x", i+1)
		stack := fmt.Sprintf("s-%d", i%5)
		containers = append(containers, domain.Container{
			ID: id, IDShort: domain.ShortID(id), Name: fmt.Sprintf("c-%d", i),
			Stack: stack, State: domain.ContainerStateRunning,
		})
	}
	st.Replace(containers, time.Now().UTC(), "")
	s := testServer(b, st)
	h := s.Handler()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/v1/containers", nil))
		if rr.Code != http.StatusOK {
			b.Fatalf("status=%d", rr.Code)
		}
	}
}
