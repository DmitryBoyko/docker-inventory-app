package parity

import (
	"sort"
	"strings"
	"time"

	"github.com/epm-games/docker-visualizer/internal/domain"
	"github.com/epm-games/docker-visualizer/internal/store"
)

const SchemaVersion = 1

// Snapshot is the machine-readable parity view shared by Go and the PS exporter.
type Snapshot struct {
	SchemaVersion int               `json:"schemaVersion"`
	Source        string            `json:"source"` // "go" | "powershell"
	CapturedAt    string            `json:"capturedAt"`
	Containers    []ContainerRow    `json:"containers"`
	Stacks        []StackRow        `json:"stacks"`
	Totals        Totals            `json:"totals"`
}

// ContainerRow is one inventory row for comparison.
type ContainerRow struct {
	IDShort           string   `json:"idShort"`
	Name              string   `json:"name"`
	Stack             string   `json:"stack"`
	Service           string   `json:"service"` // "-" when absent
	State             string   `json:"state"`
	Health            string   `json:"health"` // normalized: none|starting|healthy|unhealthy|unknown
	RestartCount      int      `json:"restartCount"`
	WritableLayerBytes *int64  `json:"writableLayerBytes"` // null if unavailable (Go); PS always fills
	VolumeNames       []string `json:"volumeNames"`
	CPUPercent        float64  `json:"cpuPercent"`
	MemoryBytes       int64    `json:"memoryBytes"`
	PortExposures     []string `json:"portExposures"` // sorted unique exposure classes
}

// StackRow is stack-level aggregation.
type StackRow struct {
	Name               string   `json:"name"`
	ContainerCount     int      `json:"containerCount"`
	RunningCount       int      `json:"runningCount"`
	UnhealthyCount     int      `json:"unhealthyCount"`
	RestartedCount     int      `json:"restartedCount"`
	CPUPercent         float64  `json:"cpuPercent"`
	MemoryBytes        int64    `json:"memoryBytes"`
	WritableLayerBytes *int64   `json:"writableLayerBytes"`
	VolumeNames        []string `json:"volumeNames"`
	VolumeBytes        *int64   `json:"volumeBytes"` // null if partial/unavailable
}

// Totals are PowerShell-style grand totals.
type Totals struct {
	ContainerCount     int      `json:"containerCount"`
	RunningCount       int      `json:"runningCount"`
	CPUPercent         float64  `json:"cpuPercent"`
	MemoryBytes        int64    `json:"memoryBytes"`
	WritableLayerBytes *int64   `json:"writableLayerBytes"`
	UniqueVolumeNames  []string `json:"uniqueVolumeNames"`
	UniqueVolumeBytes  *int64   `json:"uniqueVolumeBytes"`
}

// FromStore builds a canonical snapshot from the in-memory store.
func FromStore(st *store.Store, source string) Snapshot {
	snap := st.Load()
	return FromDomain(snap.Containers, snap.VolumesByName(), source, time.Now().UTC())
}

// FromDomain builds a canonical snapshot from domain slices.
func FromDomain(containers []domain.Container, volumesByName map[string]domain.Volume, source string, at time.Time) Snapshot {
	rows := make([]ContainerRow, 0, len(containers))
	for _, c := range containers {
		rows = append(rows, containerRow(c))
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Name == rows[j].Name {
			return rows[i].IDShort < rows[j].IDShort
		}
		return rows[i].Name < rows[j].Name
	})

	stacks := domain.BuildStacks(containers, volumesByName)
	stackRows := make([]StackRow, 0, len(stacks))
	for _, st := range stacks {
		stackRows = append(stackRows, stackRow(st))
	}

	res := domain.BuildSystemResources(containers, volumesFromMap(volumesByName))
	uniqueNames := uniqueVolumeNames(containers)
	var uniqueBytes *int64
	vu := domain.SumUniqueVolumeUsage(uniqueNames, volumesByName)
	if vu.Available && vu.Bytes != nil {
		b := *vu.Bytes
		uniqueBytes = &b
	}

	var writable *int64
	if res.WritableLayer.Available && res.WritableLayer.Bytes != nil {
		b := *res.WritableLayer.Bytes
		writable = &b
	}

	return Snapshot{
		SchemaVersion: SchemaVersion,
		Source:        source,
		CapturedAt:    at.Format(time.RFC3339Nano),
		Containers:    rows,
		Stacks:        stackRows,
		Totals: Totals{
			ContainerCount:     res.ContainerCount,
			RunningCount:       res.RunningCount,
			CPUPercent:         res.CPUPercent,
			MemoryBytes:        res.MemoryBytes,
			WritableLayerBytes: writable,
			UniqueVolumeNames:  uniqueNames,
			UniqueVolumeBytes:  uniqueBytes,
		},
	}
}

func containerRow(c domain.Container) ContainerRow {
	svc := "-"
	if c.Service != nil && *c.Service != "" {
		svc = *c.Service
	}
	stack := c.Stack
	if stack == "" {
		stack = domain.StandaloneStack
	}
	vols := make([]string, 0)
	seen := map[string]struct{}{}
	for _, m := range c.Mounts {
		if m.Type == domain.MountTypeVolume && m.Name != "" {
			if _, ok := seen[m.Name]; !ok {
				seen[m.Name] = struct{}{}
				vols = append(vols, m.Name)
			}
		}
	}
	sort.Strings(vols)

	expSet := map[string]struct{}{}
	for _, p := range c.Ports {
		if p.Exposure != "" {
			expSet[string(p.Exposure)] = struct{}{}
		}
	}
	exps := sortedKeys(expSet)

	var wl *int64
	if c.WritableLayer.Available && c.WritableLayer.Bytes != nil {
		b := *c.WritableLayer.Bytes
		wl = &b
	}

	var cpu float64
	var mem int64
	if c.State == domain.ContainerStateRunning && c.Stats != nil {
		cpu = c.Stats.CPUPercent
		mem = c.Stats.MemoryBytes
	}

	return ContainerRow{
		IDShort:            c.IDShort,
		Name:               c.Name,
		Stack:              stack,
		Service:            svc,
		State:              string(c.State),
		Health:             NormalizeHealth(string(c.Health)),
		RestartCount:       c.RestartCount,
		WritableLayerBytes: wl,
		VolumeNames:        vols,
		CPUPercent:         cpu,
		MemoryBytes:        mem,
		PortExposures:      exps,
	}
}

func stackRow(st domain.Stack) StackRow {
	var wl *int64
	if st.Resources.WritableLayer.Available && st.Resources.WritableLayer.Bytes != nil {
		b := *st.Resources.WritableLayer.Bytes
		wl = &b
	}
	var vb *int64
	if st.VolumeUsage.Available && st.VolumeUsage.Bytes != nil {
		b := *st.VolumeUsage.Bytes
		vb = &b
	}
	return StackRow{
		Name:               st.Name,
		ContainerCount:     st.Resources.ContainerCount,
		RunningCount:       st.Resources.RunningCount,
		UnhealthyCount:     st.UnhealthyCount,
		RestartedCount:     st.RestartedCount,
		CPUPercent:         st.Resources.CPUPercent,
		MemoryBytes:        st.Resources.MemoryBytes,
		WritableLayerBytes: wl,
		VolumeNames:        append([]string(nil), st.VolumeNames...),
		VolumeBytes:        vb,
	}
}

// NormalizeHealth maps PS "-" / Engine strings to domain-like tokens.
func NormalizeHealth(raw string) string {
	s := strings.TrimSpace(strings.ToLower(raw))
	switch s {
	case "", "-", "none":
		return "none"
	case "healthy":
		return "healthy"
	case "unhealthy":
		return "unhealthy"
	case "starting", "health: starting":
		return "starting"
	default:
		if strings.Contains(s, "unhealthy") {
			return "unhealthy"
		}
		if strings.Contains(s, "healthy") {
			return "healthy"
		}
		if strings.Contains(s, "starting") {
			return "starting"
		}
		return "unknown"
	}
}

func uniqueVolumeNames(containers []domain.Container) []string {
	seen := map[string]struct{}{}
	for _, c := range containers {
		for _, m := range c.Mounts {
			if m.Type == domain.MountTypeVolume && m.Name != "" {
				seen[m.Name] = struct{}{}
			}
		}
	}
	return sortedKeys(seen)
}

func volumesFromMap(m map[string]domain.Volume) []domain.Volume {
	out := make([]domain.Volume, 0, len(m))
	for _, v := range m {
		out = append(out, v)
	}
	return out
}

func sortedKeys(m map[string]struct{}) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
