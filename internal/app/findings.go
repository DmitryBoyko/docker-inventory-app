package app

import (
	"github.com/epm-games/docker-visualizer/internal/findings"
	"github.com/epm-games/docker-visualizer/internal/store"
)

// FindingsService runs explainable anomaly diagnostics against the live snapshot.
type FindingsService struct {
	Store      *store.Store
	Thresholds findings.Thresholds
}

// List returns current findings.
func (s *FindingsService) List() []findings.Finding {
	if s == nil || s.Store == nil {
		return nil
	}
	th := s.Thresholds
	if th.RestartCountCritical == 0 {
		th = findings.DefaultThresholds()
	}
	snap := s.Store.Load()
	return findings.Analyze(snap.Containers, snap.Images, snap.Volumes, snap.Networks, th)
}

// Get returns a finding by id.
func (s *FindingsService) Get(id string) (findings.Finding, bool) {
	for _, f := range s.List() {
		if f.ID == id {
			return f, true
		}
	}
	return findings.Finding{}, false
}
