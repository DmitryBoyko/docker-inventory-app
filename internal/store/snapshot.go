package store

import (
	"sync/atomic"
	"time"

	"github.com/epm-games/docker-visualizer/internal/domain"
)

// Snapshot is an immutable inventory view (ADR-012).
type Snapshot struct {
	Version     uint64
	CollectedAt time.Time
	StatsAt     time.Time
	SystemAt    time.Time
	Containers  []domain.Container
	Networks    []domain.Network
	Volumes     []domain.Volume
	Images      []domain.Image
	DiskUsage   *domain.DiskUsageView
	SystemInfo  *domain.SystemInfo
	byID        map[string]*domain.Container
	byNetwork   map[string]*domain.Network
	byVolume    map[string]*domain.Volume
	byImage     map[string]*domain.Image
	Err         string
}

// Store holds the latest atomic snapshot.
type Store struct {
	val atomic.Pointer[Snapshot]
}

// New creates an empty store.
func New() *Store {
	s := &Store{}
	s.val.Store(&Snapshot{
		byID:      map[string]*domain.Container{},
		byNetwork: map[string]*domain.Network{},
		byVolume:  map[string]*domain.Volume{},
		byImage:   map[string]*domain.Image{},
	})
	return s
}

// Replace publishes containers only, preserving networks/images/volumes/system.
func (s *Store) Replace(containers []domain.Container, collectedAt time.Time, collectErr string) {
	prev := s.Load()
	s.ReplaceInventory(containers, prev.Networks, prev.Images, collectedAt, collectErr)
}

// ReplaceInventory publishes containers + networks + images (Phase 8).
func (s *Store) ReplaceInventory(
	containers []domain.Container,
	networks []domain.Network,
	images []domain.Image,
	collectedAt time.Time,
	collectErr string,
) {
	prev := s.Load()
	cp := make([]domain.Container, len(containers))
	copy(cp, containers)
	for i := range cp {
		if old, ok := prev.byID[cp[i].ID]; ok && old.Stats != nil {
			st := *old.Stats
			cp[i].Stats = &st
		}
	}
	nets := domain.LinkNetworks(networks, cp)
	imgs := domain.LinkImages(images, cp)
	s.val.Store(buildSnapshot(
		prev.Version+1, cp, nets, prev.Volumes, imgs,
		prev.DiskUsage, prev.SystemInfo,
		collectedAt, prev.StatsAt, prev.SystemAt, collectErr,
	))
}

// MergeStats updates stats on the current inventory without dropping metadata.
func (s *Store) MergeStats(stats map[string]*domain.ContainerStats, statsAt time.Time) {
	prev := s.Load()
	cp := make([]domain.Container, len(prev.Containers))
	copy(cp, prev.Containers)
	for i := range cp {
		id := cp[i].ID
		if st, ok := stats[id]; ok && st != nil {
			cloned := *st
			cp[i].Stats = &cloned
			continue
		}
		if cp[i].State != domain.ContainerStateRunning {
			cp[i].Stats = nil
		}
	}
	s.val.Store(buildSnapshot(
		prev.Version+1, cp, prev.Networks, prev.Volumes, prev.Images,
		prev.DiskUsage, prev.SystemInfo,
		prev.CollectedAt, statsAt, prev.SystemAt, prev.Err,
	))
}

// MergeSystem publishes volumes, disk usage, and engine info.
func (s *Store) MergeSystem(volumes []domain.Volume, du *domain.DiskUsageView, info *domain.SystemInfo, systemAt time.Time) {
	prev := s.Load()
	linked := domain.LinkVolumes(volumes, prev.Containers)
	var duCopy *domain.DiskUsageView
	if du != nil {
		tmp := *du
		duCopy = &tmp
	}
	var infoCopy *domain.SystemInfo
	if info != nil {
		tmp := *info
		infoCopy = &tmp
	}
	s.val.Store(buildSnapshot(
		prev.Version+1, prev.Containers, prev.Networks, linked, prev.Images,
		duCopy, infoCopy,
		prev.CollectedAt, prev.StatsAt, systemAt, prev.Err,
	))
}

func buildSnapshot(
	version uint64,
	containers []domain.Container,
	networks []domain.Network,
	volumes []domain.Volume,
	images []domain.Image,
	du *domain.DiskUsageView,
	info *domain.SystemInfo,
	collectedAt, statsAt, systemAt time.Time,
	collectErr string,
) *Snapshot {
	cp := make([]domain.Container, len(containers))
	copy(cp, containers)
	idx := make(map[string]*domain.Container, len(cp)*2)
	for i := range cp {
		c := &cp[i]
		if c.ID != "" {
			idx[c.ID] = c
		}
		if c.IDShort != "" {
			idx[c.IDShort] = c
		}
	}

	np := make([]domain.Network, len(networks))
	copy(np, networks)
	nidx := make(map[string]*domain.Network, len(np)*2)
	for i := range np {
		n := &np[i]
		if n.ID != "" {
			nidx[n.ID] = n
		}
		if n.IDShort != "" {
			nidx[n.IDShort] = n
		}
		if n.Name != "" {
			nidx[n.Name] = n
		}
	}

	vp := make([]domain.Volume, len(volumes))
	copy(vp, volumes)
	vidx := make(map[string]*domain.Volume, len(vp))
	for i := range vp {
		v := &vp[i]
		if v.Name != "" {
			vidx[v.Name] = v
		}
	}

	ip := make([]domain.Image, len(images))
	copy(ip, images)
	iidx := make(map[string]*domain.Image, len(ip)*2)
	for i := range ip {
		im := &ip[i]
		if im.ID != "" {
			iidx[im.ID] = im
		}
		if im.IDShort != "" {
			iidx[im.IDShort] = im
		}
		for _, tag := range im.RepoTags {
			if tag != "" && tag != "<none>:<none>" {
				iidx[tag] = im
			}
		}
	}

	return &Snapshot{
		Version:     version,
		CollectedAt: collectedAt,
		StatsAt:     statsAt,
		SystemAt:    systemAt,
		Containers:  cp,
		Networks:    np,
		Volumes:     vp,
		Images:      ip,
		DiskUsage:   du,
		SystemInfo:  info,
		byID:        idx,
		byNetwork:   nidx,
		byVolume:    vidx,
		byImage:     iidx,
		Err:         collectErr,
	}
}

// Load returns the current snapshot (never nil).
func (s *Store) Load() *Snapshot {
	return s.val.Load()
}

// GetContainer finds by full or short ID.
func (s *Snapshot) GetContainer(id string) (*domain.Container, bool) {
	if s == nil {
		return nil, false
	}
	c, ok := s.byID[id]
	return c, ok
}

// GetNetwork finds by id, short id, or name.
func (s *Snapshot) GetNetwork(idOrName string) (*domain.Network, bool) {
	if s == nil {
		return nil, false
	}
	n, ok := s.byNetwork[idOrName]
	return n, ok
}

// GetVolume finds by name.
func (s *Snapshot) GetVolume(name string) (*domain.Volume, bool) {
	if s == nil {
		return nil, false
	}
	v, ok := s.byVolume[name]
	return v, ok
}

// GetImage finds by id, short id, or repo tag.
func (s *Snapshot) GetImage(idOrTag string) (*domain.Image, bool) {
	if s == nil {
		return nil, false
	}
	img, ok := s.byImage[idOrTag]
	return img, ok
}

// VolumesByName returns a name→volume map for aggregation.
func (s *Snapshot) VolumesByName() map[string]domain.Volume {
	out := make(map[string]domain.Volume, len(s.Volumes))
	for _, v := range s.Volumes {
		out[v.Name] = v
	}
	return out
}

// Age returns time since CollectedAt; zero collected → large age.
func (s *Snapshot) Age() time.Duration {
	if s == nil || s.CollectedAt.IsZero() {
		return time.Hour * 24 * 365
	}
	return time.Since(s.CollectedAt)
}

// HasData reports whether at least one successful collection completed.
func (s *Snapshot) HasData() bool {
	return s != nil && !s.CollectedAt.IsZero()
}
