package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/epm-games/docker-visualizer/internal/app"
	"github.com/epm-games/docker-visualizer/internal/docker"
	"github.com/epm-games/docker-visualizer/internal/domain"
	"github.com/epm-games/docker-visualizer/internal/store"
)

func testServer(t testing.TB, st *store.Store) *Server {
	t.Helper()
	cli, err := docker.Connect(docker.Options{
		ExplicitHost: "unix:///definitely/missing-docker-visualizer.sock",
		Timeout:      time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cli.Close() })
	return &Server{
		Docker:     cli,
		Containers: &app.ContainersService{Store: st},
		Stacks:     &app.StacksService{Store: st},
		Networks:   &app.NetworksService{Store: st},
		Volumes:    &app.VolumesService{Store: st},
		Images:     &app.ImagesService{Store: st},
		System:     &app.SystemService{Store: st},
		Graph:      &app.GraphService{Store: st},
		Export:     &app.ExportService{Store: st},
		Version:    "test",
	}
}

func TestHealth(t *testing.T) {
	s := testServer(t, store.New())
	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestReady_Unavailable(t *testing.T) {
	s := testServer(t, store.New())
	req := httptest.NewRequest(http.MethodGet, "/api/v1/ready", nil)
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestListContainersAndStacks(t *testing.T) {
	st := store.New()
	svc := "web"
	st.Replace([]domain.Container{
		{
			ID: "abc123abc123", IDShort: "abc123abc123", Name: "web", Stack: "prod", Service: &svc,
			State: domain.ContainerStateRunning, Health: domain.HealthHealthy,
			WritableLayer: domain.AvailableBytes(100),
			Stats:         &domain.ContainerStats{CPUPercent: 2.5, MemoryBytes: 1000},
			Mounts:        []domain.Mount{{Type: domain.MountTypeVolume, Name: "data"}},
		},
	}, time.Now().UTC(), "")
	st.MergeSystem([]domain.Volume{
		{Name: "data", Driver: "local", Usage: domain.VolumeUsage{ByteMetric: domain.AvailableBytes(500)}},
	}, &domain.DiskUsageView{
		Volumes: domain.DiskUsageCategory{TotalCount: 1, TotalSize: 500, ItemsKnown: true},
	}, &domain.SystemInfo{ServerVersion: "29.0.0", NCPU: 4, MemTotalBytes: 8 << 30}, time.Now().UTC())

	s := testServer(t, st)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/containers?stack=prod", nil)
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("containers status=%d body=%s", rr.Code, rr.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/stacks", nil)
	rr = httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("stacks status=%d body=%s", rr.Code, rr.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	data := body["data"].([]any)
	if len(data) != 1 {
		t.Fatalf("stacks=%v", data)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/volumes", nil)
	rr = httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("volumes status=%d", rr.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/system/resources", nil)
	rr = httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("resources status=%d body=%s", rr.Code, rr.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/system/df", nil)
	rr = httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("df status=%d", rr.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/system/info", nil)
	rr = httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("info status=%d body=%s", rr.Code, rr.Body.String())
	}

	st.ReplaceInventory(
		[]domain.Container{
			{
				ID: "abc123abc123", IDShort: "abc123abc123", Name: "web", Stack: "prod", Service: &svc,
				State: domain.ContainerStateRunning, Health: domain.HealthHealthy,
				Image: "nginx:1.25", ImageID: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				WritableLayer: domain.AvailableBytes(100),
				Endpoints:     []domain.NetworkEndpoint{{NetworkID: "nid1234567890ab", NetworkName: "frontend"}},
				Mounts:        []domain.Mount{{Type: domain.MountTypeVolume, Name: "data"}},
			},
		},
		[]domain.Network{{ID: "nid1234567890ab", IDShort: "nid123456789", Name: "frontend", Driver: "bridge"}},
		[]domain.Image{{ID: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", IDShort: "aaaaaaaaaaaa", RepoTags: []string{"nginx:1.25"}, SizeBytes: 100}},
		time.Now().UTC(), "",
	)

	req = httptest.NewRequest(http.MethodGet, "/api/v1/networks", nil)
	rr = httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("networks status=%d body=%s", rr.Code, rr.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/networks/frontend", nil)
	rr = httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("network detail status=%d body=%s", rr.Code, rr.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/images?q=nginx", nil)
	rr = httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("images status=%d body=%s", rr.Code, rr.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/images/nginx:1.25", nil)
	rr = httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("image detail status=%d body=%s", rr.Code, rr.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/graph?scope=stack&stack=prod", nil)
	rr = httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("graph status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestGetContainerStats(t *testing.T) {
	st := store.New()
	st.Replace([]domain.Container{
		{
			ID: "abc123abc123", IDShort: "abc123abc123", Name: "web",
			State: domain.ContainerStateRunning,
			Stats: &domain.ContainerStats{CPUPercent: 2.5, MemoryBytes: 1000, CountersAvailable: true},
		},
	}, time.Now().UTC(), "")
	s := testServer(t, st)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/containers/abc123abc123/stats", nil)
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}
