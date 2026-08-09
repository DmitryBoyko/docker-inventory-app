package app

import (
	"strings"
	"time"

	"github.com/epm-games/docker-visualizer/internal/domain"
	"github.com/epm-games/docker-visualizer/internal/store"
)

// StacksService derives stacks from the live snapshot.
type StacksService struct {
	Store *store.Store
}

// StacksResult is a list response.
type StacksResult struct {
	Stacks      []domain.Stack
	SnapshotAt  time.Time
	StatsAt     time.Time
	SystemAt    time.Time
	SnapshotAge time.Duration
}

// List returns all stacks sorted by name.
func (s *StacksService) List() StacksResult {
	snap := s.Store.Load()
	stacks := domain.BuildStacks(snap.Containers, snap.VolumesByName())
	return StacksResult{
		Stacks:      stacks,
		SnapshotAt:  snap.CollectedAt,
		StatsAt:     snap.StatsAt,
		SystemAt:    snap.SystemAt,
		SnapshotAge: snap.Age(),
	}
}

// Get returns one stack by name (case-insensitive).
func (s *StacksService) Get(name string) (*domain.Stack, StacksResult, bool) {
	res := s.List()
	for i := range res.Stacks {
		if strings.EqualFold(res.Stacks[i].Name, name) {
			st := res.Stacks[i]
			return &st, res, true
		}
	}
	return nil, res, false
}

// StackVolumes returns volumes linked to a stack.
func (s *StacksService) StackVolumes(name string) ([]domain.Volume, bool) {
	st, _, ok := s.Get(name)
	if !ok {
		return nil, false
	}
	snap := s.Store.Load()
	byName := snap.VolumesByName()
	out := make([]domain.Volume, 0, len(st.VolumeNames))
	for _, n := range st.VolumeNames {
		if v, ok := byName[n]; ok {
			out = append(out, v)
			continue
		}
		// Volume mounted but missing from daemon volume list.
		out = append(out, domain.Volume{
			Name: n,
			Usage: domain.VolumeUsage{
				ByteMetric: domain.UnavailableBytes(domain.ReasonDaemonOmitted),
			},
			Stacks: []string{st.Name},
		})
	}
	return out, true
}
