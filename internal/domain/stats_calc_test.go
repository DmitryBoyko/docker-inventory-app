package domain

import (
	"testing"
	"time"
)

func TestCalculateCPUPercentUnix(t *testing.T) {
	prev := CPUSample{TotalUsage: 1000, SystemUsage: 10_000}
	cur := CPUSample{TotalUsage: 2000, SystemUsage: 20_000, OnlineCPUs: 2}
	// cpuDelta=1000, systemDelta=10000 → (0.1)*2*100 = 20
	got := CalculateCPUPercentUnix(prev, cur)
	if got != 20.0 {
		t.Fatalf("got %v want 20", got)
	}
}

func TestCalculateCPUPercentUnix_PercpuFallback(t *testing.T) {
	prev := CPUSample{TotalUsage: 0, SystemUsage: 0}
	cur := CPUSample{
		TotalUsage:  500,
		SystemUsage: 1000,
		PercpuUsage: []uint64{1, 2, 3, 4},
	}
	// (500/1000)*4*100 = 200
	got := CalculateCPUPercentUnix(prev, cur)
	if got != 200.0 {
		t.Fatalf("got %v want 200", got)
	}
}

func TestCalculateCPUPercentUnix_NoDelta(t *testing.T) {
	if CalculateCPUPercentUnix(CPUSample{}, CPUSample{}) != 0 {
		t.Fatal("expected 0")
	}
}

func TestCalculateMemUsageUnixNoCache_CgroupV1(t *testing.T) {
	got := CalculateMemUsageUnixNoCache(MemorySample{
		Usage: 1000,
		Stats: map[string]uint64{"total_inactive_file": 200},
	})
	if got != 800 {
		t.Fatalf("got %v", got)
	}
}

func TestCalculateMemUsageUnixNoCache_CgroupV2(t *testing.T) {
	got := CalculateMemUsageUnixNoCache(MemorySample{
		Usage: 1000,
		Stats: map[string]uint64{"inactive_file": 300},
	})
	if got != 700 {
		t.Fatalf("got %v", got)
	}
}

func TestCalculateMemPercent(t *testing.T) {
	if CalculateMemPercentUnixNoCache(1000, 250) != 25 {
		t.Fatal()
	}
	if CalculateMemPercentUnixNoCache(0, 250) != 0 {
		t.Fatal()
	}
}

func TestCalculateBlockIO(t *testing.T) {
	r, w := CalculateBlockIO([]BlkioEntry{
		{Op: "Read", Value: 10},
		{Op: "Write", Value: 20},
		{Op: "read", Value: 5},
		{Op: "", Value: 99},
	})
	if r != 15 || w != 20 {
		t.Fatalf("r=%d w=%d", r, w)
	}
}

func TestCalculateCPUPercentWindows(t *testing.T) {
	pre := time.Unix(0, 0).UTC()
	read := pre.Add(time.Second) // 1e9 ns → 1e7 intervals of 100ns
	// possIntervals = 1e7 * 2 procs = 2e7
	// used = 1e6 → percent = 1e6/2e7*100 = 5
	got := CalculateCPUPercentWindows(read, pre, 2, 1_000_000, 0)
	if got != 5.0 {
		t.Fatalf("got %v want 5", got)
	}
}
