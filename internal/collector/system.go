package collector

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/epm-games/docker-visualizer/internal/docker"
	"github.com/epm-games/docker-visualizer/internal/domain"
	"github.com/epm-games/docker-visualizer/internal/mapper"
	"github.com/epm-games/docker-visualizer/internal/observability"
	"github.com/epm-games/docker-visualizer/internal/store"
	"github.com/epm-games/docker-visualizer/internal/ws"
	"github.com/moby/moby/client"
)

// SystemCollector refreshes volumes + disk usage (expensive; default 15s).
type SystemCollector struct {
	Docker   *docker.Client
	Store    *store.Store
	Hub      *ws.Hub
	Interval time.Duration
	Log      *slog.Logger
	Health   *observability.Registry

	mu        sync.Mutex
	refreshCh chan struct{}
}

// Run polls until ctx is cancelled.
func (c *SystemCollector) Run(ctx context.Context) {
	if c.Interval <= 0 {
		c.Interval = 15 * time.Second
	}
	if c.Log == nil {
		c.Log = slog.Default()
	}
	c.ensureRefreshCh()

	c.waitForInventory(ctx)
	c.collectOnce(ctx)

	t := time.NewTicker(c.Interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			c.collectOnce(ctx)
		case <-c.refreshCh:
			c.collectOnce(ctx)
		}
	}
}

// RequestRefresh asks for an out-of-band system collect (events). Coalesced.
func (c *SystemCollector) RequestRefresh() {
	c.ensureRefreshCh()
	select {
	case c.refreshCh <- struct{}{}:
	default:
	}
}

func (c *SystemCollector) ensureRefreshCh() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.refreshCh == nil {
		c.refreshCh = make(chan struct{}, 1)
	}
}

func (c *SystemCollector) waitForInventory(ctx context.Context) {
	deadline := time.Now().Add(15 * time.Second)
	for {
		if c.Store.Load().HasData() {
			return
		}
		if time.Now().After(deadline) {
			return
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(100 * time.Millisecond):
		}
	}
}

// Refresh performs a single system collect (parity harness / tests).
func (c *SystemCollector) Refresh(ctx context.Context) {
	if c.Log == nil {
		c.Log = slog.Default()
	}
	c.collectOnce(ctx)
}

func (c *SystemCollector) collectOnce(ctx context.Context) {
	start := time.Now()
	api := c.Docker.API()

	timeout := 60 * time.Second
	if t := c.Docker.Timeout(); t > timeout {
		timeout = t
	}
	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var info *domain.SystemInfo
	if infoRes, err := api.Info(reqCtx, client.InfoOptions{}); err != nil {
		c.Log.Warn("system info failed", "err", docker.ClassifyError(c.Docker.Endpoint().Host, err))
	} else {
		apiVer := ""
		if st := c.Docker.Status(); st.APIVersion != "" {
			apiVer = st.APIVersion
		}
		mapped := mapper.FromInfo(infoRes.Info, apiVer)
		info = &mapped
	}

	listed, err := api.VolumeList(reqCtx, client.VolumeListOptions{})
	if err != nil {
		c.Log.Warn("volume list failed", "err", docker.ClassifyError(c.Docker.Endpoint().Host, err))
		if info != nil {
			prev := c.Store.Load()
			c.Store.MergeSystem(prev.Volumes, prev.DiskUsage, info, time.Now().UTC())
			c.publishSnapshot()
		}
		if c.Health != nil {
			c.Health.RecordError("system", time.Since(start), err)
		}
		return
	}

	// Verbose required on API >= 1.52 to receive per-volume UsageData items.
	// Keep Verbose scoped to volumes to avoid pulling full image layer lists.
	dfVol, err := api.DiskUsage(reqCtx, client.DiskUsageOptions{
		Volumes: true,
		Verbose: true,
	})
	if err != nil {
		c.Log.Warn("disk usage (volumes) failed", "err", docker.ClassifyError(c.Docker.Endpoint().Host, err))
		vols := mapper.MergeVolumeLists(listed.Items, nil)
		c.Store.MergeSystem(vols, nil, info, time.Now().UTC())
		c.publishSnapshot()
		return
	}

	vols := mapper.MergeVolumeLists(listed.Items, dfVol.Volumes.Items)
	du := &domain.DiskUsageView{
		Volumes: domain.DiskUsageCategory{
			ActiveCount: dfVol.Volumes.ActiveCount,
			TotalCount:  dfVol.Volumes.TotalCount,
			Reclaimable: dfVol.Volumes.Reclaimable,
			TotalSize:   dfVol.Volumes.TotalSize,
			ItemsKnown:  len(dfVol.Volumes.Items) > 0 || dfVol.Volumes.TotalCount == 0,
		},
	}

	dfSum, err := api.DiskUsage(reqCtx, client.DiskUsageOptions{
		Containers: true,
		Images:     true,
		BuildCache: true,
	})
	if err != nil {
		c.Log.Warn("disk usage (summary) failed", "err", docker.ClassifyError(c.Docker.Endpoint().Host, err))
	} else {
		du.Containers = domain.DiskUsageCategory{
			ActiveCount: dfSum.Containers.ActiveCount,
			TotalCount:  dfSum.Containers.TotalCount,
			Reclaimable: dfSum.Containers.Reclaimable,
			TotalSize:   dfSum.Containers.TotalSize,
			ItemsKnown:  true,
		}
		du.Images = domain.DiskUsageCategory{
			ActiveCount: dfSum.Images.ActiveCount,
			TotalCount:  dfSum.Images.TotalCount,
			Reclaimable: dfSum.Images.Reclaimable,
			TotalSize:   dfSum.Images.TotalSize,
			ItemsKnown:  true,
		}
		du.BuildCache = domain.DiskUsageCategory{
			ActiveCount: dfSum.BuildCache.ActiveCount,
			TotalCount:  dfSum.BuildCache.TotalCount,
			Reclaimable: dfSum.BuildCache.Reclaimable,
			TotalSize:   dfSum.BuildCache.TotalSize,
			ItemsKnown:  true,
		}
	}

	c.Store.MergeSystem(vols, du, info, time.Now().UTC())
	c.publishSnapshot()
	if c.Health != nil {
		c.Health.RecordSuccess("system", time.Since(start))
	}
	c.Log.Info("system collected",
		"volumes", len(vols),
		"duration_ms", time.Since(start).Milliseconds(),
	)
}

func (c *SystemCollector) publishSnapshot() {
	if c.Hub == nil {
		return
	}
	snap := c.Store.Load()
	c.Hub.PublishSnapshotUpdated(ws.SnapshotNotice{
		Version:    snap.Version,
		SnapshotAt: formatTime(snap.CollectedAt),
		StatsAt:    formatTime(snap.StatsAt),
		SystemAt:   formatTime(snap.SystemAt),
		Kind:       "system",
	})
}
