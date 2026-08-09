package parity

import (
	"fmt"
	"math"
	"sort"
	"strings"
)

const (
	// CPUAbsTolerance is absolute percent points (plan §27).
	CPUAbsTolerance = 0.15
)

// Options controls comparison strictness.
type Options struct {
	// SkipStats skips CPU/memory comparisons (different sample windows).
	SkipStats bool
	// TreatMissingVolumeBytesAsZero matches PS behavior (? → 0 in sums).
	// When true, null volume/writable bytes on either side are compared as 0.
	TreatMissingVolumeBytesAsZero bool
}

// Diff is one mismatch.
type Diff struct {
	Path string `json:"path"`
	Want string `json:"want"` // reference (usually powershell)
	Got  string `json:"got"`  // candidate (usually go)
}

// Report is the full compare result.
type Report struct {
	OK          bool     `json:"ok"`
	Diffs       []Diff   `json:"diffs"`
	Notes       []string `json:"notes,omitempty"`
	RefSource   string   `json:"refSource"`
	CandSource  string   `json:"candSource"`
	RefCount    int      `json:"refContainerCount"`
	CandCount   int      `json:"candContainerCount"`
}

// Compare checks candidate against reference using MVP parity rules.
func Compare(ref, cand Snapshot, opts Options) Report {
	rep := Report{
		OK:         true,
		RefSource:  ref.Source,
		CandSource: cand.Source,
		RefCount:   len(ref.Containers),
		CandCount:  len(cand.Containers),
	}
	if opts.SkipStats {
		rep.Notes = append(rep.Notes, "stats (cpu/memory) skipped")
	}
	if opts.TreatMissingVolumeBytesAsZero {
		rep.Notes = append(rep.Notes, "null volume/writable bytes treated as 0 (PS parity mode)")
	}

	refBy := indexContainers(ref.Containers)
	candBy := indexContainers(cand.Containers)

	for key := range refBy {
		if _, ok := candBy[key]; !ok {
			add(&rep, "containers."+key, "present", "missing")
		}
	}
	for key := range candBy {
		if _, ok := refBy[key]; !ok {
			add(&rep, "containers."+key, "absent", "extra")
		}
	}

	for key, r := range refBy {
		c, ok := candBy[key]
		if !ok {
			continue
		}
		prefix := "containers." + key
		eqStr(&rep, prefix+".stack", r.Stack, c.Stack)
		eqStr(&rep, prefix+".service", r.Service, c.Service)
		eqStr(&rep, prefix+".state", r.State, c.State)
		eqStr(&rep, prefix+".health", NormalizeHealth(r.Health), NormalizeHealth(c.Health))
		eqInt(&rep, prefix+".restartCount", r.RestartCount, c.RestartCount)
		eqStringSet(&rep, prefix+".volumeNames", r.VolumeNames, c.VolumeNames)
		eqStringSet(&rep, prefix+".portExposures", r.PortExposures, c.PortExposures)
		eqBytes(&rep, prefix+".writableLayerBytes", r.WritableLayerBytes, c.WritableLayerBytes, opts)
		if !opts.SkipStats {
			eqCPU(&rep, prefix+".cpuPercent", r.CPUPercent, c.CPUPercent)
			eqInt64(&rep, prefix+".memoryBytes", r.MemoryBytes, c.MemoryBytes)
		}
	}

	refStacks := indexStacks(ref.Stacks)
	candStacks := indexStacks(cand.Stacks)
	for name := range refStacks {
		if _, ok := candStacks[name]; !ok {
			add(&rep, "stacks."+name, "present", "missing")
		}
	}
	for name := range candStacks {
		if _, ok := refStacks[name]; !ok {
			add(&rep, "stacks."+name, "absent", "extra")
		}
	}
	for name, r := range refStacks {
		c, ok := candStacks[name]
		if !ok {
			continue
		}
		prefix := "stacks." + name
		eqInt(&rep, prefix+".containerCount", r.ContainerCount, c.ContainerCount)
		eqInt(&rep, prefix+".runningCount", r.RunningCount, c.RunningCount)
		eqInt(&rep, prefix+".unhealthyCount", r.UnhealthyCount, c.UnhealthyCount)
		eqInt(&rep, prefix+".restartedCount", r.RestartedCount, c.RestartedCount)
		eqStringSet(&rep, prefix+".volumeNames", r.VolumeNames, c.VolumeNames)
		eqBytes(&rep, prefix+".writableLayerBytes", r.WritableLayerBytes, c.WritableLayerBytes, opts)
		eqBytes(&rep, prefix+".volumeBytes", r.VolumeBytes, c.VolumeBytes, opts)
		if !opts.SkipStats {
			eqCPU(&rep, prefix+".cpuPercent", r.CPUPercent, c.CPUPercent)
			eqInt64(&rep, prefix+".memoryBytes", r.MemoryBytes, c.MemoryBytes)
		}
	}

	eqInt(&rep, "totals.containerCount", ref.Totals.ContainerCount, cand.Totals.ContainerCount)
	eqInt(&rep, "totals.runningCount", ref.Totals.RunningCount, cand.Totals.RunningCount)
	eqStringSet(&rep, "totals.uniqueVolumeNames", ref.Totals.UniqueVolumeNames, cand.Totals.UniqueVolumeNames)
	eqBytes(&rep, "totals.writableLayerBytes", ref.Totals.WritableLayerBytes, cand.Totals.WritableLayerBytes, opts)
	eqBytes(&rep, "totals.uniqueVolumeBytes", ref.Totals.UniqueVolumeBytes, cand.Totals.UniqueVolumeBytes, opts)
	if !opts.SkipStats {
		eqCPU(&rep, "totals.cpuPercent", ref.Totals.CPUPercent, cand.Totals.CPUPercent)
		eqInt64(&rep, "totals.memoryBytes", ref.Totals.MemoryBytes, cand.Totals.MemoryBytes)
	}

	sort.Slice(rep.Diffs, func(i, j int) bool { return rep.Diffs[i].Path < rep.Diffs[j].Path })
	return rep
}

func indexContainers(rows []ContainerRow) map[string]ContainerRow {
	out := make(map[string]ContainerRow, len(rows))
	for _, r := range rows {
		key := r.Name
		if key == "" {
			key = r.IDShort
		}
		out[key] = r
	}
	return out
}

func indexStacks(rows []StackRow) map[string]StackRow {
	out := make(map[string]StackRow, len(rows))
	for _, r := range rows {
		out[r.Name] = r
	}
	return out
}

func add(rep *Report, path, want, got string) {
	rep.OK = false
	rep.Diffs = append(rep.Diffs, Diff{Path: path, Want: want, Got: got})
}

func eqStr(rep *Report, path, want, got string) {
	if want != got {
		add(rep, path, want, got)
	}
}

func eqInt(rep *Report, path string, want, got int) {
	if want != got {
		add(rep, path, fmt.Sprintf("%d", want), fmt.Sprintf("%d", got))
	}
}

func eqInt64(rep *Report, path string, want, got int64) {
	if want != got {
		add(rep, path, fmt.Sprintf("%d", want), fmt.Sprintf("%d", got))
	}
}

func eqCPU(rep *Report, path string, want, got float64) {
	if math.Abs(want-got) > CPUAbsTolerance {
		add(rep, path, fmt.Sprintf("%.4f", want), fmt.Sprintf("%.4f", got))
	}
}

func eqBytes(rep *Report, path string, want, got *int64, opts Options) {
	w, g := want, got
	if opts.TreatMissingVolumeBytesAsZero {
		z := int64(0)
		if w == nil {
			w = &z
		}
		if g == nil {
			g = &z
		}
	}
	if w == nil && g == nil {
		return
	}
	if w == nil || g == nil {
		add(rep, path, fmtPtr(w), fmtPtr(g))
		return
	}
	if *w != *g {
		add(rep, path, fmt.Sprintf("%d", *w), fmt.Sprintf("%d", *g))
	}
}

func eqStringSet(rep *Report, path string, want, got []string) {
	ws := normalizeSet(want)
	gs := normalizeSet(got)
	if strings.Join(ws, ",") != strings.Join(gs, ",") {
		add(rep, path, strings.Join(ws, ","), strings.Join(gs, ","))
	}
}

func normalizeSet(in []string) []string {
	seen := map[string]struct{}{}
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		seen[s] = struct{}{}
	}
	out := make([]string, 0, len(seen))
	for s := range seen {
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}

func fmtPtr(p *int64) string {
	if p == nil {
		return "null"
	}
	return fmt.Sprintf("%d", *p)
}
