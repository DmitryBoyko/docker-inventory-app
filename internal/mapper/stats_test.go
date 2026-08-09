package mapper

import (
	"testing"
	"time"

	"github.com/moby/moby/api/types/container"
)

func TestFromStatsResponse_Linux(t *testing.T) {
	v := container.StatsResponse{
		OSType: "linux",
		Read:   time.Unix(100, 0).UTC(),
		PreCPUStats: container.CPUStats{
			CPUUsage:    container.CPUUsage{TotalUsage: 1000},
			SystemUsage: 10_000,
		},
		CPUStats: container.CPUStats{
			CPUUsage:    container.CPUUsage{TotalUsage: 2000},
			SystemUsage: 20_000,
			OnlineCPUs:  2,
		},
		MemoryStats: container.MemoryStats{
			Usage: 1000,
			Limit: 4000,
			Stats: map[string]uint64{"inactive_file": 200},
		},
		Networks: map[string]container.NetworkStats{
			"eth0": {RxBytes: 10, TxBytes: 20},
			"eth1": {RxBytes: 5, TxBytes: 7},
		},
		BlkioStats: container.BlkioStats{
			IoServiceBytesRecursive: []container.BlkioStatEntry{
				{Op: "Read", Value: 100},
				{Op: "Write", Value: 50},
			},
		},
	}
	st := FromStatsResponse(v)
	if st.CPUPercent != 20 {
		t.Fatalf("cpu=%v", st.CPUPercent)
	}
	if st.MemoryBytes != 800 {
		t.Fatalf("mem=%d", st.MemoryBytes)
	}
	if st.MemoryPercent != 20 {
		t.Fatalf("mem%%=%v", st.MemoryPercent)
	}
	if st.NetworkRxBytes != 15 || st.NetworkTxBytes != 27 {
		t.Fatalf("net rx=%d tx=%d", st.NetworkRxBytes, st.NetworkTxBytes)
	}
	if st.BlockReadBytes != 100 || st.BlockWriteBytes != 50 {
		t.Fatalf("blk r=%d w=%d", st.BlockReadBytes, st.BlockWriteBytes)
	}
}
