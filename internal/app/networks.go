package app

import (
	"strings"
	"time"

	"github.com/epm-games/docker-visualizer/internal/domain"
	"github.com/epm-games/docker-visualizer/internal/store"
)

// NetworksService reads networks from the snapshot.
type NetworksService struct {
	Store *store.Store
}

// NetworksResult is a list response.
type NetworksResult struct {
	Networks    []domain.Network
	SnapshotAt  time.Time
	SnapshotAge time.Duration
}

// List returns networks (optional q / driver filters).
func (s *NetworksService) List(q, driver string) NetworksResult {
	snap := s.Store.Load()
	out := make([]domain.Network, 0, len(snap.Networks))
	ql := strings.ToLower(q)
	for _, n := range snap.Networks {
		if driver != "" && !strings.EqualFold(n.Driver, driver) {
			continue
		}
		if ql != "" {
			if !strings.Contains(strings.ToLower(n.Name), ql) &&
				!strings.Contains(strings.ToLower(n.ID), ql) &&
				!strings.Contains(strings.ToLower(n.IDShort), ql) {
				continue
			}
		}
		out = append(out, n)
	}
	return NetworksResult{
		Networks:    out,
		SnapshotAt:  snap.CollectedAt,
		SnapshotAge: snap.Age(),
	}
}

// Get returns one network by id, short id, or name.
func (s *NetworksService) Get(idOrName string) (*domain.Network, time.Time, bool) {
	snap := s.Store.Load()
	n, ok := snap.GetNetwork(idOrName)
	if !ok {
		return nil, snap.CollectedAt, false
	}
	cp := *n
	return &cp, snap.CollectedAt, true
}
