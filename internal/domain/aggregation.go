package domain

import (
	"sort"
	"strings"
)

// BuildStacks aggregates containers into Compose stacks (PowerShell Group-Object Stack).
func BuildStacks(containers []Container, volumesByName map[string]Volume) []Stack {
	groups := map[string][]Container{}
	order := make([]string, 0)
	for _, c := range containers {
		name := c.Stack
		if name == "" {
			name = StandaloneStack
		}
		if _, ok := groups[name]; !ok {
			order = append(order, name)
		}
		groups[name] = append(groups[name], c)
	}
	sort.Strings(order)

	out := make([]Stack, 0, len(order))
	for _, name := range order {
		out = append(out, buildOneStack(name, groups[name], volumesByName))
	}
	return out
}

func buildOneStack(name string, containers []Container, volumesByName map[string]Volume) Stack {
	st := Stack{
		Name:       name,
		Containers: make([]ContainerRef, 0, len(containers)),
	}

	var writable []ByteMetric
	var cpu float64
	var mem int64
	running := 0
	volNamesSet := map[string]struct{}{}

	for _, c := range containers {
		st.Containers = append(st.Containers, ContainerRef{
			ID: c.ID, IDShort: c.IDShort, Name: c.Name, State: c.State, Health: c.Health,
		})
		writable = append(writable, c.WritableLayer)

		if c.State == ContainerStateRunning {
			running++
			if c.Stats != nil {
				cpu += c.Stats.CPUPercent
				mem += c.Stats.MemoryBytes
			}
		}
		if c.Health == HealthUnhealthy {
			st.UnhealthyCount++
		}
		if c.RestartCount > 0 {
			st.RestartedCount++
		}
		for _, m := range c.Mounts {
			if m.Type == MountTypeVolume && m.Name != "" {
				volNamesSet[m.Name] = struct{}{}
			}
		}
	}

	volNames := make([]string, 0, len(volNamesSet))
	for n := range volNamesSet {
		volNames = append(volNames, n)
	}
	sort.Strings(volNames)

	st.VolumeNames = volNames
	st.VolumeUsage = SumUniqueVolumeUsage(volNames, volumesByName)
	st.Resources = ResourceSummary{
		CPUPercent:     cpu,
		MemoryBytes:    mem,
		WritableLayer:  SumByteMetrics(writable),
		VolumeData:     st.VolumeUsage,
		ContainerCount: len(containers),
		RunningCount:   running,
	}
	st.Services = groupServices(name, containers)
	st.TopRAM = topRAM(containers, 3)
	return st
}

func groupServices(stack string, containers []Container) []Service {
	bySvc := map[string][]ContainerRef{}
	var order []string
	for _, c := range containers {
		svc := "-"
		if c.Service != nil && *c.Service != "" {
			svc = *c.Service
		}
		if _, ok := bySvc[svc]; !ok {
			order = append(order, svc)
		}
		bySvc[svc] = append(bySvc[svc], ContainerRef{
			ID: c.ID, IDShort: c.IDShort, Name: c.Name, State: c.State, Health: c.Health,
		})
	}
	sort.Strings(order)
	out := make([]Service, 0, len(order))
	for _, svc := range order {
		name := svc
		if name == "-" {
			name = ""
		}
		out = append(out, Service{
			Name:       name,
			Stack:      stack,
			Containers: bySvc[svc],
		})
	}
	return out
}

func topRAM(containers []Container, n int) []TopConsumer {
	type row struct {
		c    Container
		mem  int64
	}
	rows := make([]row, 0, len(containers))
	for _, c := range containers {
		if c.Stats == nil || c.Stats.MemoryBytes <= 0 {
			continue
		}
		rows = append(rows, row{c: c, mem: c.Stats.MemoryBytes})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].mem > rows[j].mem })
	if len(rows) > n {
		rows = rows[:n]
	}
	out := make([]TopConsumer, 0, len(rows))
	for _, r := range rows {
		out = append(out, TopConsumer{
			ContainerID:   r.c.ID,
			ContainerName: r.c.Name,
			MemoryBytes:   r.mem,
		})
	}
	return out
}

// LinkVolumes attaches reverse container/stack references and Shared flag.
func LinkVolumes(volumes []Volume, containers []Container) []Volume {
	type hit struct {
		containers map[string]struct{}
		stacks     map[string]struct{}
	}
	hits := map[string]*hit{}
	for _, c := range containers {
		for _, m := range c.Mounts {
			if m.Type != MountTypeVolume || m.Name == "" {
				continue
			}
			h, ok := hits[m.Name]
			if !ok {
				h = &hit{containers: map[string]struct{}{}, stacks: map[string]struct{}{}}
				hits[m.Name] = h
			}
			h.containers[c.Name] = struct{}{}
			stack := c.Stack
			if stack == "" {
				stack = StandaloneStack
			}
			h.stacks[stack] = struct{}{}
		}
	}

	out := make([]Volume, len(volumes))
	copy(out, volumes)
	for i := range out {
		h := hits[out[i].Name]
		if h == nil {
			continue
		}
		out[i].Containers = sortedKeys(h.containers)
		out[i].Stacks = sortedKeys(h.stacks)
		out[i].Shared = len(h.stacks) > 1
	}
	sort.Slice(out, func(i, j int) bool {
		return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
	})
	return out
}

// BuildSystemResources computes PowerShell-style grand totals.
func BuildSystemResources(containers []Container, volumes []Volume) SystemResources {
	byVol := map[string]Volume{}
	for _, v := range volumes {
		byVol[v.Name] = v
	}

	var writable []ByteMetric
	var cpu float64
	var mem int64
	var rx, tx, br, bw int64
	running := 0
	var volNames []string
	seenVol := map[string]struct{}{}

	for _, c := range containers {
		writable = append(writable, c.WritableLayer)
		if c.State == ContainerStateRunning {
			running++
			if c.Stats != nil {
				cpu += c.Stats.CPUPercent
				mem += c.Stats.MemoryBytes
				rx += c.Stats.NetworkRxBytes
				tx += c.Stats.NetworkTxBytes
				br += c.Stats.BlockReadBytes
				bw += c.Stats.BlockWriteBytes
			}
		}
		for _, m := range c.Mounts {
			if m.Type == MountTypeVolume && m.Name != "" {
				if _, ok := seenVol[m.Name]; !ok {
					seenVol[m.Name] = struct{}{}
					volNames = append(volNames, m.Name)
				}
			}
		}
	}

	return SystemResources{
		ResourceSummary: ResourceSummary{
			CPUPercent:     cpu,
			MemoryBytes:    mem,
			WritableLayer:  SumByteMetrics(writable),
			VolumeData:     SumUniqueVolumeUsage(volNames, byVol),
			ContainerCount: len(containers),
			RunningCount:   running,
		},
		NetworkRxBytes:  rx,
		NetworkTxBytes:  tx,
		BlockReadBytes:  br,
		BlockWriteBytes: bw,
	}
}

func sortedKeys(m map[string]struct{}) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
