package domain

import (
	"sort"
	"strings"
)

// Network is a Docker network as exposed by our API (not SDK types).
type Network struct {
	ID         string            `json:"id"`
	IDShort    string            `json:"idShort"`
	Name       string            `json:"name"`
	Driver     string            `json:"driver"`
	Scope      string            `json:"scope,omitempty"`
	Internal   bool              `json:"internal"`
	Attachable bool              `json:"attachable,omitempty"`
	Ingress    bool              `json:"ingress,omitempty"`
	Labels     map[string]string `json:"labels,omitempty"`
	Containers []string          `json:"containers,omitempty"`
	Stacks     []string          `json:"stacks,omitempty"`
}

// LinkNetworks attaches reverse container/stack refs from inventory endpoints.
func LinkNetworks(networks []Network, containers []Container) []Network {
	type hit struct {
		containers map[string]struct{}
		stacks     map[string]struct{}
	}
	byName := map[string]*hit{}
	byID := map[string]*hit{}

	touch := func(m map[string]*hit, key string) *hit {
		if key == "" {
			return nil
		}
		h, ok := m[key]
		if !ok {
			h = &hit{containers: map[string]struct{}{}, stacks: map[string]struct{}{}}
			m[key] = h
		}
		return h
	}

	for _, c := range containers {
		stack := c.Stack
		if stack == "" {
			stack = StandaloneStack
		}
		for _, ep := range c.Endpoints {
			if h := touch(byName, ep.NetworkName); h != nil {
				h.containers[c.Name] = struct{}{}
				h.stacks[stack] = struct{}{}
			}
			if h := touch(byID, ep.NetworkID); h != nil {
				h.containers[c.Name] = struct{}{}
				h.stacks[stack] = struct{}{}
			}
		}
	}

	out := make([]Network, len(networks))
	copy(out, networks)
	for i := range out {
		h := byName[out[i].Name]
		if h == nil {
			h = byID[out[i].ID]
		}
		if h == nil {
			h = byID[out[i].IDShort]
		}
		if h == nil {
			continue
		}
		out[i].Containers = sortedKeys(h.containers)
		out[i].Stacks = sortedKeys(h.stacks)
	}
	sort.Slice(out, func(i, j int) bool {
		return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
	})
	return out
}
