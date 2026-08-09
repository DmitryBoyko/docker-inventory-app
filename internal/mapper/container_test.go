package mapper

import (
	"net/netip"
	"testing"

	"github.com/epm-games/docker-visualizer/internal/domain"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/mount"
	"github.com/moby/moby/api/types/network"
)

func TestFromSummary_ComposeAndPorts(t *testing.T) {
	ip := netip.MustParseAddr("0.0.0.0")
	s := container.Summary{
		ID:      "abcdefghijklmnopqrstuvwxyz0123456789",
		Names:   []string{"/webapp"},
		Image:   "nginx:1.25",
		ImageID: "sha256:deadbeef",
		State:   container.StateRunning,
		Status:  "Up 3 minutes (healthy)",
		Labels: map[string]string{
			domain.LabelComposeProject: "prod",
			domain.LabelComposeService: "frontend",
		},
		SizeRw: 1048576,
		Ports: []container.PortSummary{
			{IP: ip, PrivatePort: 80, PublicPort: 8080, Type: "tcp"},
		},
		Mounts: []container.MountPoint{
			{Type: mount.TypeVolume, Name: "webapp-data", Destination: "/data", RW: true},
		},
		NetworkSettings: &container.NetworkSettingsSummary{
			Networks: map[string]*network.EndpointSettings{
				"frontend-net": {
					IPAddress: netip.MustParseAddr("172.20.0.5"),
					NetworkID: "netid1",
				},
			},
		},
	}

	c := FromSummary(s)
	if c.IDShort != "abcdefghijkl" {
		t.Fatalf("short=%s", c.IDShort)
	}
	if c.Name != "webapp" || c.Stack != "prod" {
		t.Fatalf("name/stack=%s/%s", c.Name, c.Stack)
	}
	if c.Service == nil || *c.Service != "frontend" {
		t.Fatalf("service=%v", c.Service)
	}
	if c.Health != domain.HealthHealthy {
		t.Fatalf("health=%s", c.Health)
	}
	if !c.WritableLayer.Available || c.WritableLayer.Bytes == nil || *c.WritableLayer.Bytes != 1048576 {
		t.Fatalf("disk=%+v", c.WritableLayer)
	}
	if len(c.Ports) != 1 || c.Ports[0].Exposure != domain.PortExposurePublic {
		t.Fatalf("ports=%+v", c.Ports)
	}
	if len(c.Mounts) != 1 || c.Mounts[0].Name != "webapp-data" {
		t.Fatalf("mounts=%+v", c.Mounts)
	}
	if len(c.Endpoints) != 1 || c.Endpoints[0].IPAddress != "172.20.0.5" {
		t.Fatalf("endpoints=%+v", c.Endpoints)
	}
}

func TestEnrichFromInspect_RestartsAndUptime(t *testing.T) {
	c := domain.Container{
		ID:     "abc",
		Status: "Up 1 minute",
		State:  domain.ContainerStateRunning,
		Health: domain.HealthNone,
	}
	insp := container.InspectResponse{
		Name:         "/abc",
		RestartCount: 3,
		State: &container.State{
			Status:    container.StateRunning,
			StartedAt: "2026-08-09T00:00:00Z",
			Health:    &container.Health{Status: container.Healthy},
		},
		Mounts: []container.MountPoint{
			{Type: mount.TypeVolume, Name: "v1", Destination: "/v", RW: true},
		},
	}
	EnrichFromInspect(&c, insp)
	if c.RestartCount != 3 {
		t.Fatalf("restarts=%d", c.RestartCount)
	}
	if c.Health != domain.HealthHealthy {
		t.Fatalf("health=%s", c.Health)
	}
	if c.StartedAt == nil || c.UptimeSeconds == nil {
		t.Fatalf("started/uptime missing")
	}
	if len(c.Mounts) != 1 || c.Mounts[0].Name != "v1" {
		t.Fatalf("mounts=%+v", c.Mounts)
	}
}

func TestFromSummary_Standalone(t *testing.T) {
	c := FromSummary(container.Summary{
		ID:     "ffffffffffffffffffffffffffffffff",
		Names:  []string{"/lonely"},
		Image:  "sha256:0123456789abcdef0123456789abcdef",
		State:  container.StateExited,
		Status: "Exited (0) 2 days ago",
	})
	if c.Stack != domain.StandaloneStack {
		t.Fatalf("stack=%s", c.Stack)
	}
	if c.Service != nil {
		t.Fatalf("service=%v", c.Service)
	}
	if c.Image != "sha256:0123456789ab..." {
		t.Fatalf("image=%s", c.Image)
	}
}
