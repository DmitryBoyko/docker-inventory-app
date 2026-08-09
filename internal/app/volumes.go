package app

import (
	"strings"
	"time"

	"github.com/epm-games/docker-visualizer/internal/domain"
	"github.com/epm-games/docker-visualizer/internal/store"
)

// VolumesService reads volumes from the snapshot.
type VolumesService struct {
	Store *store.Store
}

// VolumesResult is a list response.
type VolumesResult struct {
	Volumes     []domain.Volume
	SnapshotAt  time.Time
	SystemAt    time.Time
	SnapshotAge time.Duration
}

// List returns linked volumes (optionally filtered by stack / q).
func (s *VolumesService) List(stack, q string) VolumesResult {
	snap := s.Store.Load()
	vols := domain.LinkVolumes(snap.Volumes, snap.Containers)
	out := make([]domain.Volume, 0, len(vols))
	for _, v := range vols {
		if stack != "" && !volumeInStack(v, stack) {
			continue
		}
		if q != "" && !strings.Contains(strings.ToLower(v.Name), strings.ToLower(q)) {
			continue
		}
		out = append(out, v)
	}
	return VolumesResult{
		Volumes:     out,
		SnapshotAt:  snap.CollectedAt,
		SystemAt:    snap.SystemAt,
		SnapshotAge: snap.Age(),
	}
}

// Get returns one volume by name.
func (s *VolumesService) Get(name string) (*domain.Volume, time.Time, bool) {
	snap := s.Store.Load()
	vols := domain.LinkVolumes(snap.Volumes, snap.Containers)
	for i := range vols {
		if vols[i].Name == name {
			v := vols[i]
			return &v, snap.SystemAt, true
		}
	}
	return nil, snap.SystemAt, false
}

func volumeInStack(v domain.Volume, stack string) bool {
	for _, s := range v.Stacks {
		if strings.EqualFold(s, stack) {
			return true
		}
	}
	return false
}
