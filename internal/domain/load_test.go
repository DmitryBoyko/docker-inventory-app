package domain

import (
	"fmt"
	"testing"
	"time"
)

// Synthetic load target from Phase 12 acceptance (~200 containers).
const loadContainerCount = 200

func synthInventory(n int) (containers []Container, networks []Network, volumes []Volume, images []Image, volsByName map[string]Volume) {
	containers = make([]Container, 0, n)
	networks = []Network{
		{ID: "net-bridge", Name: "bridge", Driver: "bridge"},
		{ID: "net-a", Name: "stack_a_default", Driver: "bridge"},
		{ID: "net-b", Name: "stack_b_default", Driver: "bridge"},
	}
	images = []Image{
		{ID: "sha256:img1", RepoTags: []string{"nginx:latest"}, SizeBytes: 50 << 20},
		{ID: "sha256:img2", RepoTags: []string{"redis:7"}, SizeBytes: 30 << 20},
	}
	volsByName = make(map[string]Volume, n/4+1)
	volumes = make([]Volume, 0, n/4+1)

	for i := 0; i < n; i++ {
		stack := fmt.Sprintf("stack-%02d", i%10)
		svc := fmt.Sprintf("svc-%d", i%5)
		volName := fmt.Sprintf("vol-%d", i%40)
		if _, ok := volsByName[volName]; !ok {
			v := Volume{
				Name:   volName,
				Driver: "local",
				Usage:  VolumeUsage{ByteMetric: AvailableBytes(int64(i%7+1) * 1024 * 1024)},
			}
			volsByName[volName] = v
			volumes = append(volumes, v)
		}
		id := fmt.Sprintf("%064x", i+1)
		netName := "stack_a_default"
		if i%2 == 0 {
			netName = "stack_b_default"
		}
		cpu := float64(i%100) / 10
		mem := int64((i%50 + 1) * 1024 * 1024)
		containers = append(containers, Container{
			ID: id, IDShort: ShortID(id), Name: fmt.Sprintf("c-%03d", i),
			Stack: stack, Service: &svc,
			Image: images[i%2].RepoTags[0], ImageID: images[i%2].ID,
			State: ContainerStateRunning, Health: HealthHealthy,
			WritableLayer: AvailableBytes(int64(i%9) * 1024),
			Endpoints: []NetworkEndpoint{
				{NetworkID: networks[1+(i%2)].ID, NetworkName: netName, IPAddress: fmt.Sprintf("172.18.0.%d", (i%250)+2)},
			},
			Mounts: []Mount{{Type: MountTypeVolume, Name: volName, Destination: "/data", RW: true}},
			Stats: &ContainerStats{
				Timestamp: time.Now().UTC(), CPUPercent: cpu,
				MemoryBytes: mem, MemoryLimitBytes: 512 << 20, MemoryPercent: float64(mem) / float64(512<<20) * 100,
				CountersAvailable: true,
			},
		})
	}
	return containers, networks, volumes, images, volsByName
}

func TestLoad_BuildStacks200(t *testing.T) {
	containers, _, _, _, vols := synthInventory(loadContainerCount)
	stacks := BuildStacks(containers, vols)
	if len(stacks) != 10 {
		t.Fatalf("stacks=%d want 10", len(stacks))
	}
	totalCtr := 0
	for _, st := range stacks {
		totalCtr += st.Resources.ContainerCount
	}
	if totalCtr != loadContainerCount {
		t.Fatalf("container refs=%d", totalCtr)
	}
}

func TestLoad_BuildGraph200(t *testing.T) {
	containers, networks, volumes, images, _ := synthInventory(loadContainerCount)
	linkedN := LinkNetworks(networks, containers)
	linkedI := LinkImages(images, containers)
	g := BuildGraph(containers, linkedN, volumes, linkedI, GraphOptions{Scope: "all"})
	if len(g.Nodes) < loadContainerCount {
		t.Fatalf("nodes=%d", len(g.Nodes))
	}
	if len(g.Edges) == 0 {
		t.Fatal("expected edges")
	}
}

func BenchmarkBuildStacks200(b *testing.B) {
	containers, _, _, _, vols := synthInventory(loadContainerCount)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = BuildStacks(containers, vols)
	}
}

func BenchmarkBuildGraph200(b *testing.B) {
	containers, networks, volumes, images, _ := synthInventory(loadContainerCount)
	linkedN := LinkNetworks(networks, containers)
	linkedI := LinkImages(images, containers)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = BuildGraph(containers, linkedN, volumes, linkedI, GraphOptions{Scope: "all"})
	}
}
