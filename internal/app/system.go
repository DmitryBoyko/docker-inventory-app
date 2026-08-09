package app

import (
	"time"

	"github.com/epm-games/docker-visualizer/internal/domain"
	"github.com/epm-games/docker-visualizer/internal/store"
)

// SystemService exposes df + resource rollups.
type SystemService struct {
	Store *store.Store
}

// ResourcesResult wraps grand totals.
type ResourcesResult struct {
	Resources  domain.SystemResources
	SnapshotAt time.Time
	StatsAt    time.Time
	SystemAt   time.Time
}

// Resources returns PowerShell-style ALL totals.
func (s *SystemService) Resources() ResourcesResult {
	snap := s.Store.Load()
	vols := domain.LinkVolumes(snap.Volumes, snap.Containers)
	return ResourcesResult{
		Resources:  domain.BuildSystemResources(snap.Containers, vols),
		SnapshotAt: snap.CollectedAt,
		StatsAt:    snap.StatsAt,
		SystemAt:   snap.SystemAt,
	}
}

// DiskUsageResult wraps /system/df view.
type DiskUsageResult struct {
	DiskUsage  *domain.DiskUsageView
	SnapshotAt time.Time
	SystemAt   time.Time
}

// DiskUsage returns the last collected df summary (may be nil before first system collect).
func (s *SystemService) DiskUsage() DiskUsageResult {
	snap := s.Store.Load()
	return DiskUsageResult{
		DiskUsage:  snap.DiskUsage,
		SnapshotAt: snap.CollectedAt,
		SystemAt:   snap.SystemAt,
	}
}

// InfoResult wraps Engine info.
type InfoResult struct {
	Info       *domain.SystemInfo
	SnapshotAt time.Time
	SystemAt   time.Time
}

// Info returns cached Engine /info.
func (s *SystemService) Info() InfoResult {
	snap := s.Store.Load()
	return InfoResult{
		Info:       snap.SystemInfo,
		SnapshotAt: snap.CollectedAt,
		SystemAt:   snap.SystemAt,
	}
}
