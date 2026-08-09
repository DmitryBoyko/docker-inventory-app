package parity

import (
	"testing"
	"time"

	"github.com/epm-games/docker-visualizer/internal/domain"
)

func TestCompare_Matching(t *testing.T) {
	svc := "web"
	containers := []domain.Container{
		{
			ID: "aaaaaaaaaaaa", IDShort: "aaaaaaaaaaaa", Name: "web-1",
			Stack: "prod", Service: &svc, State: domain.ContainerStateRunning,
			Health: domain.HealthHealthy, RestartCount: 1,
			WritableLayer: domain.AvailableBytes(100),
			Mounts:        []domain.Mount{{Type: domain.MountTypeVolume, Name: "data"}},
			Ports:         []domain.Port{{ContainerPort: 80, Protocol: "tcp", Exposure: domain.PortExposurePublic}},
			Stats:         &domain.ContainerStats{CPUPercent: 2.5, MemoryBytes: 1000},
		},
	}
	vols := map[string]domain.Volume{
		"data": {Name: "data", Usage: domain.VolumeUsage{ByteMetric: domain.AvailableBytes(500)}},
	}
	a := FromDomain(containers, vols, "go", time.Now().UTC())
	b := FromDomain(containers, vols, "powershell", time.Now().UTC())
	rep := Compare(a, b, Options{})
	if !rep.OK {
		t.Fatalf("diffs=%+v", rep.Diffs)
	}
}

func TestCompare_HealthNormalize(t *testing.T) {
	ref := Snapshot{
		Source: "powershell",
		Containers: []ContainerRow{
			{Name: "c1", IDShort: "abc", Stack: "s", Service: "-", State: "running", Health: "-", RestartCount: 0},
		},
		Stacks: []StackRow{{Name: "s", ContainerCount: 1}},
		Totals: Totals{ContainerCount: 1},
	}
	cand := Snapshot{
		Source: "go",
		Containers: []ContainerRow{
			{Name: "c1", IDShort: "abc", Stack: "s", Service: "-", State: "running", Health: "none", RestartCount: 0},
		},
		Stacks: []StackRow{{Name: "s", ContainerCount: 1}},
		Totals: Totals{ContainerCount: 1},
	}
	rep := Compare(ref, cand, Options{SkipStats: true})
	if !rep.OK {
		t.Fatalf("diffs=%+v", rep.Diffs)
	}
}

func TestCompare_CPUTolerance(t *testing.T) {
	ref := Snapshot{
		Containers: []ContainerRow{{Name: "c", IDShort: "x", Stack: "s", Service: "-", State: "running", Health: "none", CPUPercent: 1.0}},
		Stacks:     []StackRow{{Name: "s", ContainerCount: 1, RunningCount: 1, CPUPercent: 1.0}},
		Totals:     Totals{ContainerCount: 1, RunningCount: 1, CPUPercent: 1.0},
	}
	cand := Snapshot{
		Containers: []ContainerRow{{Name: "c", IDShort: "x", Stack: "s", Service: "-", State: "running", Health: "none", CPUPercent: 1.10}},
		Stacks:     []StackRow{{Name: "s", ContainerCount: 1, RunningCount: 1, CPUPercent: 1.10}},
		Totals:     Totals{ContainerCount: 1, RunningCount: 1, CPUPercent: 1.10},
	}
	if !Compare(ref, cand, Options{}).OK {
		t.Fatal("within 0.15 should pass")
	}
	cand.Containers[0].CPUPercent = 1.20
	cand.Stacks[0].CPUPercent = 1.20
	cand.Totals.CPUPercent = 1.20
	if Compare(ref, cand, Options{}).OK {
		t.Fatal("over tolerance should fail")
	}
}

func TestNormalizeHealth(t *testing.T) {
	cases := map[string]string{
		"-": "none", "NONE": "none", "healthy": "healthy",
		"health: starting": "starting", "Up (unhealthy)": "unhealthy",
	}
	for in, want := range cases {
		if got := NormalizeHealth(in); got != want {
			t.Errorf("%q -> %q want %q", in, got, want)
		}
	}
}
