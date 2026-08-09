package app

import (
	"strings"
	"testing"
	"time"

	"github.com/epm-games/docker-visualizer/internal/domain"
	"github.com/epm-games/docker-visualizer/internal/store"
)

func TestExport_JSONAndCSV(t *testing.T) {
	st := store.New()
	svc := "api"
	st.Replace([]domain.Container{
		{
			ID: "abcdabcdabcd", IDShort: "abcdabcdabcd", Name: "api-1",
			Stack: "demo", Service: &svc, State: domain.ContainerStateRunning,
			Health: domain.HealthHealthy, RestartCount: 0,
			WritableLayer: domain.AvailableBytes(42),
			Mounts:        []domain.Mount{{Type: domain.MountTypeVolume, Name: "data"}},
			Stats:         &domain.ContainerStats{CPUPercent: 1.5, MemoryBytes: 2048},
		},
	}, time.Now().UTC(), "")
	st.MergeSystem([]domain.Volume{
		{Name: "data", Usage: domain.VolumeUsage{ByteMetric: domain.AvailableBytes(100)}},
	}, nil, nil, time.Now().UTC())

	ex := &ExportService{Store: st}

	j, err := ex.Export("json", "all")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(j.Body), `"name": "api-1"`) {
		t.Fatalf("json=%s", j.Body)
	}
	if !strings.Contains(j.ContentDisposition, ".json") {
		t.Fatalf("disp=%s", j.ContentDisposition)
	}

	c, err := ex.Export("csv", "containers")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(c.Body), "api-1") || !strings.Contains(string(c.Body), "demo") {
		t.Fatalf("csv=%s", c.Body)
	}

	s, err := ex.Export("csv", "stacks")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(s.Body), "demo") {
		t.Fatalf("stacks csv=%s", s.Body)
	}

	if _, err := ex.Export("xml", "all"); err == nil {
		t.Fatal("expected bad format")
	}
}
