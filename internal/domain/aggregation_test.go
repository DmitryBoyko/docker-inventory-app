package domain

import "testing"

func TestBuildStacks_ParityTotals(t *testing.T) {
	svc := "web"
	containers := []Container{
		{
			ID: "1", IDShort: "1", Name: "web-1", Stack: "prod", Service: &svc,
			State: ContainerStateRunning, Health: HealthHealthy, RestartCount: 1,
			WritableLayer: AvailableBytes(100),
			Stats:         &ContainerStats{CPUPercent: 2.5, MemoryBytes: 1000},
			Mounts:        []Mount{{Type: MountTypeVolume, Name: "data"}},
		},
		{
			ID: "2", IDShort: "2", Name: "web-2", Stack: "prod", Service: &svc,
			State: ContainerStateExited, Health: HealthUnhealthy, RestartCount: 0,
			WritableLayer: AvailableBytes(50),
			Mounts:        []Mount{{Type: MountTypeVolume, Name: "data"}},
		},
		{
			ID: "3", IDShort: "3", Name: "alone", Stack: StandaloneStack,
			State: ContainerStateRunning,
			WritableLayer: AvailableBytes(10),
			Stats:         &ContainerStats{CPUPercent: 1, MemoryBytes: 100},
		},
	}
	vols := map[string]Volume{
		"data": {Name: "data", Usage: VolumeUsage{ByteMetric: AvailableBytes(500), Links: int64Ptr(2)}},
	}

	stacks := BuildStacks(containers, vols)
	if len(stacks) != 2 {
		t.Fatalf("stacks=%d", len(stacks))
	}
	prod := stacks[0]
	if prod.Name != "prod" {
		prod = stacks[1]
	}
	if prod.Resources.RunningCount != 1 || prod.Resources.CPUPercent != 2.5 || prod.Resources.MemoryBytes != 1000 {
		t.Fatalf("resources=%+v", prod.Resources)
	}
	if prod.Resources.WritableLayer.Bytes == nil || *prod.Resources.WritableLayer.Bytes != 150 {
		t.Fatalf("writable=%+v", prod.Resources.WritableLayer)
	}
	if prod.VolumeUsage.Bytes == nil || *prod.VolumeUsage.Bytes != 500 {
		t.Fatalf("vol=%+v", prod.VolumeUsage)
	}
	if prod.UnhealthyCount != 1 || prod.RestartedCount != 1 {
		t.Fatalf("counts unhealthy=%d restarted=%d", prod.UnhealthyCount, prod.RestartedCount)
	}
	if len(prod.TopRAM) != 1 || prod.TopRAM[0].MemoryBytes != 1000 {
		t.Fatalf("top=%+v", prod.TopRAM)
	}
}

func TestBuildStacks_UnknownVolumePartial(t *testing.T) {
	containers := []Container{
		{
			ID: "1", Name: "c", Stack: "s", State: ContainerStateRunning,
			WritableLayer: AvailableBytes(1),
			Mounts:        []Mount{{Type: MountTypeVolume, Name: "missing"}},
		},
	}
	st := BuildStacks(containers, map[string]Volume{})[0]
	if st.VolumeUsage.Available || st.VolumeUsage.Bytes != nil || !st.VolumeUsage.Partial || st.VolumeUsage.UnknownCount != 1 {
		t.Fatalf("%+v", st.VolumeUsage)
	}
}

func TestBuildSystemResources_UniqueVolumes(t *testing.T) {
	containers := []Container{
		{Name: "a", State: ContainerStateRunning, WritableLayer: AvailableBytes(10),
			Stats: &ContainerStats{CPUPercent: 1, MemoryBytes: 100, NetworkRxBytes: 1, NetworkTxBytes: 2, BlockReadBytes: 3, BlockWriteBytes: 4},
			Mounts: []Mount{{Type: MountTypeVolume, Name: "v1"}}},
		{Name: "b", State: ContainerStateRunning, WritableLayer: AvailableBytes(20),
			Stats:  &ContainerStats{CPUPercent: 2, MemoryBytes: 200},
			Mounts: []Mount{{Type: MountTypeVolume, Name: "v1"}}},
	}
	vols := []Volume{{Name: "v1", Usage: VolumeUsage{ByteMetric: AvailableBytes(999)}}}
	res := BuildSystemResources(containers, vols)
	if res.CPUPercent != 3 || res.MemoryBytes != 300 {
		t.Fatalf("%+v", res.ResourceSummary)
	}
	if res.VolumeData.Bytes == nil || *res.VolumeData.Bytes != 999 {
		t.Fatalf("vol unique failed: %+v", res.VolumeData)
	}
	if res.WritableLayer.Bytes == nil || *res.WritableLayer.Bytes != 30 {
		t.Fatalf("writable=%+v", res.WritableLayer)
	}
}

func TestLinkVolumes_Shared(t *testing.T) {
	vols := []Volume{{Name: "shared"}, {Name: "solo"}}
	containers := []Container{
		{Name: "a", Stack: "s1", Mounts: []Mount{{Type: MountTypeVolume, Name: "shared"}}},
		{Name: "b", Stack: "s2", Mounts: []Mount{{Type: MountTypeVolume, Name: "shared"}}},
		{Name: "c", Stack: "s1", Mounts: []Mount{{Type: MountTypeVolume, Name: "solo"}}},
	}
	out := LinkVolumes(vols, containers)
	var shared, solo Volume
	for _, v := range out {
		switch v.Name {
		case "shared":
			shared = v
		case "solo":
			solo = v
		}
	}
	if !shared.Shared || len(shared.Stacks) != 2 {
		t.Fatalf("shared=%+v", shared)
	}
	if solo.Shared || len(solo.Stacks) != 1 {
		t.Fatalf("solo=%+v", solo)
	}
}

func int64Ptr(v int64) *int64 { return &v }
