package app

import (
	"fmt"
	"strings"
	"time"

	"github.com/epm-games/docker-visualizer/internal/domain"
	"github.com/epm-games/docker-visualizer/internal/docker"
	"github.com/epm-games/docker-visualizer/internal/snapshots"
	"github.com/epm-games/docker-visualizer/internal/store"
)

// SnapshotsService creates and compares persisted inventory snapshots.
type SnapshotsService struct {
	Store  *store.Store
	Docker *docker.Client
	Disk   *snapshots.Store
	Host   string
}

// Create captures current inventory into a persisted snapshot.
func (s *SnapshotsService) Create(label string) (snapshots.Snapshot, error) {
	if s == nil || s.Disk == nil || s.Store == nil {
		return snapshots.Snapshot{}, fmt.Errorf("snapshots not configured")
	}
	live := s.Store.Load()
	volMap := map[string]domain.Volume{}
	for _, v := range live.Volumes {
		volMap[v.Name] = v
	}
	stacks := domain.BuildStacks(live.Containers, volMap)
	opts := snapshots.CaptureOptions{
		HostName: s.Host,
		Label:    strings.TrimSpace(label),
		ID:       snapshots.NewID(time.Now().UTC()),
		Now:      time.Now().UTC(),
	}
	if s.Docker != nil {
		ep := s.Docker.Endpoint()
		opts.Endpoint = ep.Host
		opts.Context = ep.Context
		if live.SystemInfo != nil {
			opts.DockerVersion = live.SystemInfo.ServerVersion
		}
	}
	snap := snapshots.Capture(live.Containers, live.Images, live.Networks, live.Volumes, stacks, opts)
	if err := s.Disk.Save(snap); err != nil {
		return snapshots.Snapshot{}, err
	}
	return snap, nil
}

// List returns snapshot metas.
func (s *SnapshotsService) List() ([]snapshots.Meta, error) {
	if s == nil || s.Disk == nil {
		return nil, fmt.Errorf("snapshots not configured")
	}
	return s.Disk.List()
}

// Get returns a full snapshot.
func (s *SnapshotsService) Get(id string) (snapshots.Snapshot, error) {
	if s == nil || s.Disk == nil {
		return snapshots.Snapshot{}, fmt.Errorf("snapshots not configured")
	}
	return s.Disk.Get(id)
}

// Diff compares two snapshots. If rightID is "current" or empty, compares left to live state.
func (s *SnapshotsService) Diff(leftID, rightID string) (snapshots.Diff, error) {
	if s == nil || s.Disk == nil {
		return snapshots.Diff{}, fmt.Errorf("snapshots not configured")
	}
	left, err := s.Disk.Get(leftID)
	if err != nil {
		return snapshots.Diff{}, err
	}
	rightID = strings.TrimSpace(rightID)
	var right snapshots.Snapshot
	if rightID == "" || strings.EqualFold(rightID, "current") {
		right, err = s.currentAsSnapshot()
		if err != nil {
			return snapshots.Diff{}, err
		}
		right.ID = "current"
	} else {
		right, err = s.Disk.Get(rightID)
		if err != nil {
			return snapshots.Diff{}, err
		}
	}
	return snapshots.Compare(left, right), nil
}

func (s *SnapshotsService) currentAsSnapshot() (snapshots.Snapshot, error) {
	live := s.Store.Load()
	volMap := map[string]domain.Volume{}
	for _, v := range live.Volumes {
		volMap[v.Name] = v
	}
	stacks := domain.BuildStacks(live.Containers, volMap)
	opts := snapshots.CaptureOptions{
		HostName: s.Host,
		ID:       "current",
		Now:      time.Now().UTC(),
	}
	if s.Docker != nil {
		ep := s.Docker.Endpoint()
		opts.Endpoint = ep.Host
		opts.Context = ep.Context
	}
	if live.SystemInfo != nil {
		opts.DockerVersion = live.SystemInfo.ServerVersion
	}
	return snapshots.Capture(live.Containers, live.Images, live.Networks, live.Volumes, stacks, opts), nil
}

// Delete removes a persisted snapshot.
func (s *SnapshotsService) Delete(id string) error {
	if s == nil || s.Disk == nil {
		return fmt.Errorf("snapshots not configured")
	}
	return s.Disk.Delete(id)
}
