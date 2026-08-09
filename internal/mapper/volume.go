package mapper

import (
	"github.com/epm-games/docker-visualizer/internal/domain"
	"github.com/moby/moby/api/types/volume"
)

// FromVolume maps a Docker volume (list or df item) into domain.Volume.
// UsageData.Size == -1 means unavailable (ADR-011).
func FromVolume(v volume.Volume) domain.Volume {
	out := domain.Volume{
		Name:       v.Name,
		Driver:     v.Driver,
		Mountpoint: v.Mountpoint,
		Scope:      v.Scope,
		Labels:     cloneLabels(v.Labels),
		Usage: domain.VolumeUsage{
			ByteMetric: domain.UnavailableBytes(domain.ReasonDaemonOmitted),
		},
	}
	if v.UsageData == nil {
		return out
	}
	out.Usage = mapUsageData(v.UsageData, v.Driver)
	return out
}

// ApplyUsage overlays /system/df UsageData onto an existing volume.
func ApplyUsage(dst *domain.Volume, src volume.Volume) {
	if dst == nil || src.UsageData == nil {
		return
	}
	dst.Usage = mapUsageData(src.UsageData, src.Driver)
}

func mapUsageData(u *volume.UsageData, driver string) domain.VolumeUsage {
	vu := domain.VolumeUsage{}
	if u.RefCount >= 0 {
		links := u.RefCount
		vu.Links = &links
	}
	if u.Size < 0 {
		reason := domain.ReasonUnsupported
		if driver != "" && driver != "local" {
			reason = domain.ReasonNotLocalDriver
		}
		vu.ByteMetric = domain.UnavailableBytes(reason)
		return vu
	}
	vu.ByteMetric = domain.AvailableBytes(u.Size)
	return vu
}

// MergeVolumeLists unions VolumeList metadata with DiskUsage volume items (usage wins).
func MergeVolumeLists(listed []volume.Volume, usageItems []volume.Volume) []domain.Volume {
	byName := map[string]domain.Volume{}
	order := make([]string, 0, len(listed)+len(usageItems))

	for _, v := range listed {
		byName[v.Name] = FromVolume(v)
		order = append(order, v.Name)
	}
	for _, v := range usageItems {
		if cur, ok := byName[v.Name]; ok {
			ApplyUsage(&cur, v)
			byName[v.Name] = cur
			continue
		}
		byName[v.Name] = FromVolume(v)
		order = append(order, v.Name)
	}

	out := make([]domain.Volume, 0, len(byName))
	seen := map[string]struct{}{}
	for _, name := range order {
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, byName[name])
	}
	return out
}
