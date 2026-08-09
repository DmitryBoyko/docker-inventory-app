package domain

import "testing"

func TestBuildGraphStackScope(t *testing.T) {
	svc := "web"
	containers := []Container{
		{
			ID: "abc123abc123", IDShort: "abc123abc123", Name: "web_1", Stack: "prod", Service: &svc,
			State: ContainerStateRunning, Health: HealthHealthy, Image: "nginx:1.25",
			ImageID: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			Endpoints: []NetworkEndpoint{{NetworkName: "frontend", NetworkID: "nid1"}},
			Mounts:    []Mount{{Type: MountTypeVolume, Name: "data"}},
		},
		{
			ID: "def456def456", IDShort: "def456def456", Name: "other", Stack: "dev",
			State: ContainerStateExited, Image: "redis:7",
		},
	}
	g := BuildGraph(containers,
		[]Network{{ID: "nid1", IDShort: "nid1", Name: "frontend", Driver: "bridge"}},
		[]Volume{{Name: "data", Driver: "local", Usage: VolumeUsage{ByteMetric: AvailableBytes(10)}}},
		[]Image{{ID: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", IDShort: "aaaaaaaaaaaa", RepoTags: []string{"nginx:1.25"}}},
		GraphOptions{Scope: "stack", Stack: "prod"},
	)
	if g.Scope != "stack" || g.Stack != "prod" {
		t.Fatalf("%+v", g)
	}
	types := map[string]int{}
	for _, n := range g.Nodes {
		types[n.Type]++
	}
	if types["stack"] != 1 || types["service"] != 1 || types["container"] != 1 {
		t.Fatalf("nodes=%v types=%v", g.Nodes, types)
	}
	if types["network"] != 1 || types["volume"] != 1 || types["image"] != 1 {
		t.Fatalf("entity types=%v", types)
	}
	edgeTypes := map[string]int{}
	for _, e := range g.Edges {
		edgeTypes[e.Type]++
	}
	for _, want := range []string{"contains", "runs", "attached", "mounts", "uses_image"} {
		if edgeTypes[want] < 1 {
			t.Fatalf("missing edge %s in %v", want, edgeTypes)
		}
	}
}

func TestBuildGraphAll(t *testing.T) {
	g := BuildGraph([]Container{
		{ID: "a", IDShort: "a", Name: "a", Stack: "s1", State: ContainerStateRunning},
		{ID: "b", IDShort: "b", Name: "b", Stack: "s2", State: ContainerStateRunning},
	}, nil, nil, nil, GraphOptions{Scope: "all"})
	stacks := 0
	for _, n := range g.Nodes {
		if n.Type == "stack" {
			stacks++
		}
	}
	if stacks != 2 {
		t.Fatalf("stacks=%d nodes=%v", stacks, g.Nodes)
	}
}
