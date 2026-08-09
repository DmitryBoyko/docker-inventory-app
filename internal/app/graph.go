package app

import (
	"strings"
	"time"

	"github.com/epm-games/docker-visualizer/internal/domain"
	"github.com/epm-games/docker-visualizer/internal/store"
)

// GraphService builds topology views from the snapshot.
type GraphService struct {
	Store *store.Store
}

// GraphResult wraps a graph with snapshot metadata.
type GraphResult struct {
	Graph       domain.Graph
	SnapshotAt  time.Time
	StatsAt     time.Time
	SnapshotAge time.Duration
}

// Get builds a graph for scope=all|stack.
func (s *GraphService) Get(scope, stack string) (GraphResult, error) {
	snap := s.Store.Load()
	scope = strings.ToLower(strings.TrimSpace(scope))
	if scope == "" {
		scope = "all"
	}
	if scope != "all" && scope != "stack" {
		return GraphResult{}, errBadScope
	}
	if scope == "stack" && strings.TrimSpace(stack) == "" {
		return GraphResult{}, errStackRequired
	}
	g := domain.BuildGraph(snap.Containers, snap.Networks, snap.Volumes, snap.Images, domain.GraphOptions{
		Scope: scope,
		Stack: stack,
	})
	return GraphResult{
		Graph:       g,
		SnapshotAt:  snap.CollectedAt,
		StatsAt:     snap.StatsAt,
		SnapshotAge: snap.Age(),
	}, nil
}

type graphError string

func (e graphError) Error() string { return string(e) }

const (
	errBadScope      graphError = "scope must be all or stack"
	errStackRequired graphError = "stack query parameter is required when scope=stack"
)
