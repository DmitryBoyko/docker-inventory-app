package app

import (
	"strings"
	"time"

	"github.com/epm-games/docker-visualizer/internal/domain"
	"github.com/epm-games/docker-visualizer/internal/store"
)

// ContainerQuery filters inventory listings.
type ContainerQuery struct {
	Stack  string
	State  string
	Health string
	Q      string // substring match on name, image, id, stack, service
}

// ContainersService reads container inventory from the snapshot store.
type ContainersService struct {
	Store *store.Store
}

// ListResult is a list response payload.
type ListResult struct {
	Containers  []domain.Container
	SnapshotAt  time.Time
	SnapshotAge time.Duration
	CollectErr  string
}

// List returns filtered containers from the latest snapshot.
func (s *ContainersService) List(q ContainerQuery) ListResult {
	snap := s.Store.Load()
	out := make([]domain.Container, 0, len(snap.Containers))
	for _, c := range snap.Containers {
		if q.Stack != "" && !strings.EqualFold(c.Stack, q.Stack) {
			continue
		}
		if q.State != "" && !strings.EqualFold(string(c.State), q.State) {
			continue
		}
		if q.Health != "" && !strings.EqualFold(string(c.Health), q.Health) {
			continue
		}
		if q.Q != "" && !matchContainer(c, q.Q) {
			continue
		}
		out = append(out, c)
	}
	return ListResult{
		Containers:  out,
		SnapshotAt:  snap.CollectedAt,
		SnapshotAge: snap.Age(),
		CollectErr:  snap.Err,
	}
}

// Get returns one container by full or short ID.
func (s *ContainersService) Get(id string) (*domain.Container, time.Time, bool) {
	snap := s.Store.Load()
	c, ok := snap.GetContainer(id)
	if !ok {
		return nil, snap.CollectedAt, false
	}
	cp := *c
	return &cp, snap.CollectedAt, true
}

// StatsResult is a single stats sample lookup.
type StatsResult struct {
	Stats      *domain.ContainerStats
	SnapshotAt time.Time
	StatsAt    time.Time
	Found      bool
	Running    bool
}

// GetStats returns the latest cached stats for a container.
func (s *ContainersService) GetStats(id string) StatsResult {
	snap := s.Store.Load()
	c, ok := snap.GetContainer(id)
	if !ok {
		return StatsResult{Found: false}
	}
	res := StatsResult{
		Found:      true,
		Running:    c.State == domain.ContainerStateRunning,
		SnapshotAt: snap.CollectedAt,
		StatsAt:    snap.StatsAt,
	}
	if c.Stats != nil {
		st := *c.Stats
		res.Stats = &st
	}
	return res
}

func matchContainer(c domain.Container, q string) bool {
	q = strings.ToLower(q)
	if strings.Contains(strings.ToLower(c.Name), q) {
		return true
	}
	if strings.Contains(strings.ToLower(c.Image), q) {
		return true
	}
	if strings.Contains(strings.ToLower(c.ID), q) || strings.Contains(strings.ToLower(c.IDShort), q) {
		return true
	}
	if strings.Contains(strings.ToLower(c.Stack), q) {
		return true
	}
	if c.Service != nil && strings.Contains(strings.ToLower(*c.Service), q) {
		return true
	}
	return false
}
