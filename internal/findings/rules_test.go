package findings

import (
	"testing"

	"github.com/epm-games/docker-visualizer/internal/domain"
)

func TestRestartCritical(t *testing.T) {
	th := DefaultThresholds()
	c := domain.Container{ID: "abc123456789", IDShort: "abc123456789", Name: "nginx", State: domain.ContainerStateRunning, RestartCount: 14, Health: domain.HealthHealthy}
	fs := Analyze([]domain.Container{c}, nil, nil, nil, th)
	if !hasRule(fs, "container.restarts") {
		t.Fatalf("missing restarts finding: %+v", fs)
	}
}

func TestUnhealthy(t *testing.T) {
	c := domain.Container{ID: "1", IDShort: "1", Name: "api", State: domain.ContainerStateRunning, Health: domain.HealthUnhealthy}
	fs := Analyze([]domain.Container{c}, nil, nil, nil, DefaultThresholds())
	if !hasRule(fs, "container.unhealthy") {
		t.Fatal("expected unhealthy")
	}
}

func TestWritableLayer(t *testing.T) {
	sz := int64(3 << 30)
	c := domain.Container{
		ID: "1", IDShort: "1", Name: "fat", State: domain.ContainerStateRunning, Health: domain.HealthNone,
		WritableLayer: domain.AvailableBytes(sz),
	}
	fs := Analyze([]domain.Container{c}, nil, nil, nil, DefaultThresholds())
	if !hasRule(fs, "container.writable_layer") {
		t.Fatal("expected writable layer finding")
	}
}

func TestDanglingAndUnusedImage(t *testing.T) {
	imgs := []domain.Image{
		{ID: "sha256:aaa", IDShort: "aaa", Dangling: true, SizeBytes: 100},
		{ID: "sha256:bbb", IDShort: "bbb", RepoTags: []string{"app:1"}, ContainerCount: 0, SizeBytes: 100},
	}
	fs := Analyze(nil, imgs, nil, nil, DefaultThresholds())
	if !hasRule(fs, "image.dangling") || !hasRule(fs, "image.unused") {
		t.Fatalf("got %+v", fs)
	}
}

func TestUnusedVolumeAndSizeUnavailable(t *testing.T) {
	vols := []domain.Volume{
		{Name: "data", Driver: "local", Usage: domain.VolumeUsage{ByteMetric: domain.UnavailableBytes(domain.ReasonDaemonOmitted)}},
	}
	fs := Analyze(nil, nil, vols, nil, DefaultThresholds())
	if !hasRule(fs, "volume.unused") || !hasRule(fs, "volume.size_unavailable") {
		t.Fatalf("got %+v", fs)
	}
}

func TestUnusedNetworkSkipsBridge(t *testing.T) {
	nets := []domain.Network{
		{ID: "1", IDShort: "1", Name: "bridge", Driver: "bridge"},
		{ID: "2", IDShort: "2", Name: "appnet", Driver: "bridge"},
	}
	fs := Analyze(nil, nil, nil, nets, DefaultThresholds())
	if hasRule(fs, "network.unused") && findingName(fs, "network.unused") == "bridge" {
		t.Fatal("should not flag bridge")
	}
	if !hasRule(fs, "network.unused") {
		t.Fatal("expected appnet unused")
	}
}

func hasRule(fs []Finding, rule string) bool {
	for _, f := range fs {
		if f.RuleID == rule {
			return true
		}
	}
	return false
}

func findingName(fs []Finding, rule string) string {
	for _, f := range fs {
		if f.RuleID == rule {
			return f.Entity.Name
		}
	}
	return ""
}
