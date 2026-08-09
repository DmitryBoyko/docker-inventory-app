package store

import (
	"testing"
	"time"

	"github.com/epm-games/docker-visualizer/internal/domain"
)

func TestReplaceAndGet(t *testing.T) {
	s := New()
	if s.Load().HasData() {
		t.Fatal("expected empty")
	}
	s.Replace([]domain.Container{
		{ID: "fullid1234567890", IDShort: "fullid123456", Name: "n1"},
	}, time.Now().UTC(), "")
	snap := s.Load()
	if !snap.HasData() || len(snap.Containers) != 1 {
		t.Fatalf("%+v", snap)
	}
	if _, ok := snap.GetContainer("fullid123456"); !ok {
		t.Fatal("short lookup failed")
	}
}

func TestReplacePreservesStatsAndVolumes(t *testing.T) {
	s := New()
	st := &domain.ContainerStats{CPUPercent: 12.5, MemoryBytes: 100}
	s.Replace([]domain.Container{
		{ID: "id1", IDShort: "id1", Name: "a", State: domain.ContainerStateRunning, Stats: st},
	}, time.Now().UTC(), "")
	s.MergeSystem([]domain.Volume{{Name: "v1", Usage: domain.VolumeUsage{ByteMetric: domain.AvailableBytes(9)}}}, nil, nil, time.Now().UTC())

	s.Replace([]domain.Container{
		{ID: "id1", IDShort: "id1", Name: "a", State: domain.ContainerStateRunning},
	}, time.Now().UTC(), "")

	snap := s.Load()
	c, _ := snap.GetContainer("id1")
	if c.Stats == nil || c.Stats.CPUPercent != 12.5 {
		t.Fatalf("stats not preserved: %+v", c)
	}
	if len(snap.Volumes) != 1 || snap.Volumes[0].Name != "v1" {
		t.Fatalf("volumes not preserved: %+v", snap.Volumes)
	}
}

func TestMergeStats(t *testing.T) {
	s := New()
	s.Replace([]domain.Container{
		{ID: "run1", IDShort: "run1", Name: "r", State: domain.ContainerStateRunning},
		{ID: "stop1", IDShort: "stop1", Name: "s", State: domain.ContainerStateExited, Stats: &domain.ContainerStats{CPUPercent: 1}},
	}, time.Now().UTC(), "")

	s.MergeStats(map[string]*domain.ContainerStats{
		"run1": {CPUPercent: 3.5, MemoryBytes: 42},
	}, time.Now().UTC())

	snap := s.Load()
	run, _ := snap.GetContainer("run1")
	if run.Stats == nil || run.Stats.CPUPercent != 3.5 {
		t.Fatalf("run stats=%+v", run.Stats)
	}
	stopped, _ := snap.GetContainer("stop1")
	if stopped.Stats != nil {
		t.Fatalf("stopped should clear stats: %+v", stopped.Stats)
	}
}
