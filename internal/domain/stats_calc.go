package domain

import "time"

// CPUSample is a platform-neutral CPU counter snapshot for CLI-compatible math.
type CPUSample struct {
	TotalUsage  uint64
	SystemUsage uint64
	OnlineCPUs  uint32
	PercpuUsage []uint64
}

// MemorySample is a platform-neutral memory snapshot.
type MemorySample struct {
	Usage uint64
	Limit uint64
	Stats map[string]uint64
}

// BlkioEntry is one blkio counter row.
type BlkioEntry struct {
	Op    string
	Value uint64
}

// CalculateCPUPercentUnix mirrors docker/cli calculateCPUPercentUnix.
func CalculateCPUPercentUnix(previous, current CPUSample) float64 {
	cpuDelta := float64(current.TotalUsage) - float64(previous.TotalUsage)
	systemDelta := float64(current.SystemUsage) - float64(previous.SystemUsage)
	onlineCPUs := float64(current.OnlineCPUs)
	if onlineCPUs == 0.0 {
		onlineCPUs = float64(len(current.PercpuUsage))
	}
	if systemDelta > 0.0 && cpuDelta > 0.0 {
		return (cpuDelta / systemDelta) * onlineCPUs * 100.0
	}
	return 0
}

// CalculateCPUPercentWindows mirrors docker/cli calculateCPUPercentWindows.
func CalculateCPUPercentWindows(read, preRead time.Time, numProcs uint32, totalUsage, preTotalUsage uint64) float64 {
	possIntervals := uint64(read.Sub(preRead).Nanoseconds())
	possIntervals /= 100
	possIntervals *= uint64(numProcs)
	if possIntervals > 0 {
		intervalsUsed := totalUsage - preTotalUsage
		return float64(intervalsUsed) / float64(possIntervals) * 100.0
	}
	return 0
}

// CalculateMemUsageUnixNoCache mirrors docker/cli (exclude page cache).
func CalculateMemUsageUnixNoCache(mem MemorySample) float64 {
	if v, isCgroup1 := mem.Stats["total_inactive_file"]; isCgroup1 && v < mem.Usage {
		return float64(mem.Usage - v)
	}
	if v := mem.Stats["inactive_file"]; v < mem.Usage {
		return float64(mem.Usage - v)
	}
	return float64(mem.Usage)
}

// CalculateMemPercentUnixNoCache mirrors docker/cli.
func CalculateMemPercentUnixNoCache(limit float64, usedNoCache float64) float64 {
	if limit != 0 {
		return usedNoCache / limit * 100.0
	}
	return 0
}

// CalculateBlockIO sums read/write bytes from IoServiceBytesRecursive.
func CalculateBlockIO(entries []BlkioEntry) (read, write uint64) {
	for _, e := range entries {
		if len(e.Op) == 0 {
			continue
		}
		switch e.Op[0] {
		case 'r', 'R':
			read += e.Value
		case 'w', 'W':
			write += e.Value
		}
	}
	return read, write
}
