package mapper

import (
	"strings"
	"time"

	"github.com/epm-games/docker-visualizer/internal/domain"
	"github.com/moby/moby/api/types/container"
)

// FromStatsResponse maps Engine stats using Docker CLI-compatible formulas.
func FromStatsResponse(v container.StatsResponse) domain.ContainerStats {
	osType := strings.ToLower(v.OSType)
	st := domain.ContainerStats{
		Timestamp:         v.Read.UTC(),
		CountersAvailable: true,
	}
	if st.Timestamp.IsZero() {
		st.Timestamp = time.Now().UTC()
	}

	if osType == "windows" {
		st.CPUPercent = domain.CalculateCPUPercentWindows(
			v.Read, v.PreRead, v.NumProcs,
			v.CPUStats.CPUUsage.TotalUsage, v.PreCPUStats.CPUUsage.TotalUsage,
		)
		st.MemoryBytes = int64(v.MemoryStats.PrivateWorkingSet)
		st.BlockReadBytes = int64(v.StorageStats.ReadSizeBytes)
		st.BlockWriteBytes = int64(v.StorageStats.WriteSizeBytes)
	} else {
		memUsage := domain.CalculateMemUsageUnixNoCache(domain.MemorySample{
			Usage: v.MemoryStats.Usage,
			Limit: v.MemoryStats.Limit,
			Stats: v.MemoryStats.Stats,
		})
		st.CPUPercent = domain.CalculateCPUPercentUnix(
			toCPUSample(v.PreCPUStats),
			toCPUSample(v.CPUStats),
		)
		st.MemoryBytes = int64(memUsage)
		st.MemoryLimitBytes = int64(v.MemoryStats.Limit)
		st.MemoryPercent = domain.CalculateMemPercentUnixNoCache(float64(v.MemoryStats.Limit), memUsage)
		br, bw := domain.CalculateBlockIO(toBlkio(v.BlkioStats.IoServiceBytesRecursive))
		st.BlockReadBytes = int64(br)
		st.BlockWriteBytes = int64(bw)
	}

	var rx, tx uint64
	for _, n := range v.Networks {
		rx += n.RxBytes
		tx += n.TxBytes
	}
	st.NetworkRxBytes = int64(rx)
	st.NetworkTxBytes = int64(tx)
	return st
}

func toCPUSample(c container.CPUStats) domain.CPUSample {
	return domain.CPUSample{
		TotalUsage:  c.CPUUsage.TotalUsage,
		SystemUsage: c.SystemUsage,
		OnlineCPUs:  c.OnlineCPUs,
		PercpuUsage: c.CPUUsage.PercpuUsage,
	}
}

func toBlkio(entries []container.BlkioStatEntry) []domain.BlkioEntry {
	out := make([]domain.BlkioEntry, 0, len(entries))
	for _, e := range entries {
		out = append(out, domain.BlkioEntry{Op: e.Op, Value: e.Value})
	}
	return out
}
