package domain

import (
	"fmt"
	"sort"
	"strings"
)

// Graph is a renderer-agnostic topology model for the UI.
type Graph struct {
	Scope string      `json:"scope"`
	Stack string      `json:"stack,omitempty"`
	Nodes []GraphNode `json:"nodes"`
	Edges []GraphEdge `json:"edges"`
}

// GraphNode is one topology vertex.
type GraphNode struct {
	ID    string         `json:"id"`
	Type  string         `json:"type"` // stack|service|container|network|volume|image
	Label string         `json:"label"`
	Data  map[string]any `json:"data,omitempty"`
}

// GraphEdge is one topology edge.
type GraphEdge struct {
	ID     string `json:"id"`
	Type   string `json:"type"` // contains|runs|attached|mounts|uses_image
	Source string `json:"source"`
	Target string `json:"target"`
}

// GraphOptions controls BuildGraph filtering.
type GraphOptions struct {
	Scope string // all|stack
	Stack string // required when Scope=stack
}

// BuildGraph derives nodes/edges from inventory snapshot slices.
func BuildGraph(containers []Container, networks []Network, volumes []Volume, images []Image, opts GraphOptions) Graph {
	scope := strings.ToLower(strings.TrimSpace(opts.Scope))
	if scope == "" {
		scope = "all"
	}
	stackFilter := strings.TrimSpace(opts.Stack)

	filtered := make([]Container, 0, len(containers))
	for _, c := range containers {
		st := c.Stack
		if st == "" {
			st = StandaloneStack
		}
		if scope == "stack" {
			if stackFilter == "" || !strings.EqualFold(st, stackFilter) {
				continue
			}
		}
		filtered = append(filtered, c)
	}

	g := Graph{Scope: scope, Stack: stackFilter}
	nodes := map[string]GraphNode{}
	edges := map[string]GraphEdge{}

	addNode := func(n GraphNode) {
		if _, ok := nodes[n.ID]; !ok {
			nodes[n.ID] = n
		}
	}
	addEdge := func(typ, src, dst string) {
		id := fmt.Sprintf("%s:%s->%s", typ, src, dst)
		if _, ok := edges[id]; ok {
			return
		}
		edges[id] = GraphEdge{ID: id, Type: typ, Source: src, Target: dst}
	}

	volByName := map[string]Volume{}
	for _, v := range volumes {
		volByName[v.Name] = v
	}
	netByKey := map[string]Network{}
	for _, n := range networks {
		netByKey[n.Name] = n
		netByKey[n.ID] = n
		netByKey[n.IDShort] = n
	}
	imgByKey := map[string]Image{}
	for _, img := range images {
		imgByKey[img.ID] = img
		imgByKey[img.IDShort] = img
		for _, tag := range img.RepoTags {
			if tag != "" && tag != "<none>:<none>" {
				imgByKey[tag] = img
			}
		}
	}

	stackHealth := map[string]string{}
	stackStats := map[string]struct {
		running, total, unhealthy int
	}{}

	for _, c := range filtered {
		stack := c.Stack
		if stack == "" {
			stack = StandaloneStack
		}
		st := stackStats[stack]
		st.total++
		if c.State == ContainerStateRunning {
			st.running++
		}
		if c.Health == HealthUnhealthy {
			st.unhealthy++
		}
		stackStats[stack] = st

		svcName := "_"
		if c.Service != nil && *c.Service != "" {
			svcName = *c.Service
		}

		stackID := "stack:" + stack
		svcID := fmt.Sprintf("service:%s:%s", stack, svcName)
		ctrID := "container:" + c.ID

		addNode(GraphNode{ID: stackID, Type: "stack", Label: stack})
		addNode(GraphNode{ID: svcID, Type: "service", Label: svcName, Data: map[string]any{"stack": stack}})
		ctrData := map[string]any{
			"state":  string(c.State),
			"health": string(c.Health),
			"stack":  stack,
			"idShort": c.IDShort,
		}
		if c.Stats != nil {
			ctrData["cpuPercent"] = c.Stats.CPUPercent
			ctrData["memoryBytes"] = c.Stats.MemoryBytes
		}
		addNode(GraphNode{ID: ctrID, Type: "container", Label: c.Name, Data: ctrData})

		addEdge("contains", stackID, svcID)
		addEdge("runs", svcID, ctrID)

		for _, ep := range c.Endpoints {
			n, ok := netByKey[ep.NetworkName]
			if !ok {
				n, ok = netByKey[ep.NetworkID]
			}
			label := ep.NetworkName
			netID := "network:" + label
			data := map[string]any{}
			if ok {
				label = n.Name
				netID = "network:" + n.Name
				data["driver"] = n.Driver
				data["idShort"] = n.IDShort
			}
			addNode(GraphNode{ID: netID, Type: "network", Label: label, Data: data})
			addEdge("attached", ctrID, netID)
		}

		for _, m := range c.Mounts {
			if m.Type != MountTypeVolume || m.Name == "" {
				continue
			}
			volID := "volume:" + m.Name
			data := map[string]any{}
			if v, ok := volByName[m.Name]; ok {
				if v.Usage.Available && v.Usage.Bytes != nil {
					data["usageBytes"] = *v.Usage.Bytes
				}
				data["available"] = v.Usage.Available
				data["driver"] = v.Driver
			}
			addNode(GraphNode{ID: volID, Type: "volume", Label: m.Name, Data: data})
			addEdge("mounts", ctrID, volID)
		}

		imgKey := c.ImageID
		img, ok := imgByKey[imgKey]
		if !ok {
			img, ok = imgByKey[c.Image]
		}
		if ok {
			imgID := "image:" + img.IDShort
			label := primaryTag(img)
			addNode(GraphNode{
				ID: imgID, Type: "image", Label: label,
				Data: map[string]any{"idShort": img.IDShort, "sizeBytes": img.SizeBytes, "dangling": img.Dangling},
			})
			addEdge("uses_image", ctrID, imgID)
		} else if c.Image != "" {
			imgID := "image:" + c.Image
			addNode(GraphNode{ID: imgID, Type: "image", Label: c.Image})
			addEdge("uses_image", ctrID, imgID)
		}
	}

	for name, st := range stackStats {
		health := "unknown"
		switch {
		case st.unhealthy > 0:
			health = "unhealthy"
		case st.running == 0 && st.total > 0:
			health = "stopped"
		case st.running < st.total:
			health = "degraded"
		case st.running == st.total && st.total > 0:
			health = "healthy"
		}
		stackHealth[name] = health
		id := "stack:" + name
		if n, ok := nodes[id]; ok {
			if n.Data == nil {
				n.Data = map[string]any{}
			}
			n.Data["health"] = health
			n.Data["runningCount"] = st.running
			n.Data["containerCount"] = st.total
			nodes[id] = n
		}
	}

	g.Nodes = make([]GraphNode, 0, len(nodes))
	for _, n := range nodes {
		g.Nodes = append(g.Nodes, n)
	}
	sort.Slice(g.Nodes, func(i, j int) bool {
		if g.Nodes[i].Type == g.Nodes[j].Type {
			return g.Nodes[i].ID < g.Nodes[j].ID
		}
		return g.Nodes[i].Type < g.Nodes[j].Type
	})

	g.Edges = make([]GraphEdge, 0, len(edges))
	for _, e := range edges {
		g.Edges = append(g.Edges, e)
	}
	sort.Slice(g.Edges, func(i, j int) bool { return g.Edges[i].ID < g.Edges[j].ID })
	return g
}
