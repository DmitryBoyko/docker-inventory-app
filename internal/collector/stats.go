package collector

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"sync"
	"time"

	"github.com/epm-games/docker-visualizer/internal/docker"
	"github.com/epm-games/docker-visualizer/internal/domain"
	"github.com/epm-games/docker-visualizer/internal/mapper"
	"github.com/epm-games/docker-visualizer/internal/observability"
	"github.com/epm-games/docker-visualizer/internal/store"
	"github.com/epm-games/docker-visualizer/internal/ws"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
)

const defaultStatsConcurrency = 16

// MetricsRecorder persists downsampled historical samples (ADR-015). Optional.
type MetricsRecorder interface {
	Record(host string, at time.Time, containers []MetricsContainerSample) error
}

// MetricsContainerSample is the subset of stats written to history.
type MetricsContainerSample struct {
	ID    string
	Name  string
	Stack string
	CPU   float64
	Mem   int64
	NetRx int64
	NetTx int64
}

// StatsCollector samples running-container stats (CLI-compatible one-shot).
type StatsCollector struct {
	Docker      *docker.Client
	Store       *store.Store
	Hub         ws.Bus
	HostName    string
	Metrics     MetricsRecorder
	Interval    time.Duration
	Concurrency int
	Log         *slog.Logger
	Health      *observability.Registry
}

// Run polls until ctx is cancelled.
func (c *StatsCollector) Run(ctx context.Context) {
	if c.Interval <= 0 {
		c.Interval = time.Second
	}
	if c.Concurrency <= 0 {
		c.Concurrency = defaultStatsConcurrency
	}
	if c.Log == nil {
		c.Log = slog.Default()
	}

	// Wait briefly for first inventory if empty.
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
		}
	}
}

func (c *StatsCollector) waitForInventory(ctx context.Context) {
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

// Refresh performs a single stats sample (parity harness / tests).
func (c *StatsCollector) Refresh(ctx context.Context) {
	if c.Log == nil {
		c.Log = slog.Default()
	}
	if c.Concurrency <= 0 {
		c.Concurrency = defaultStatsConcurrency
	}
	c.collectOnce(ctx)
}

func (c *StatsCollector) collectOnce(ctx context.Context) {
	start := time.Now()
	snap := c.Store.Load()
	if !snap.HasData() {
		return
	}

	var running []string
	for _, ctr := range snap.Containers {
		if ctr.State == domain.ContainerStateRunning {
			running = append(running, ctr.ID)
		}
	}
	if len(running) == 0 {
		c.Store.MergeStats(map[string]*domain.ContainerStats{}, time.Now().UTC())
		c.publishStats()
		return
	}

	stats := make(map[string]*domain.ContainerStats, len(running))
	var mu sync.Mutex
	sem := make(chan struct{}, c.Concurrency)
	var wg sync.WaitGroup
	var okCount int

	for _, id := range running {
		id := id
		wg.Add(1)
		go func() {
			defer wg.Done()
			select {
			case <-ctx.Done():
				return
			case sem <- struct{}{}:
			}
			defer func() { <-sem }()

			st, err := c.fetchOne(ctx, id)
			if err != nil {
				c.Log.Debug("stats sample failed", "id", domain.ShortID(id), "err", err)
				return
			}
			mu.Lock()
			stats[id] = &st
			okCount++
			mu.Unlock()
		}()
	}
	wg.Wait()

	at := time.Now().UTC()
	c.Store.MergeStats(stats, at)
	c.publishStats()
	c.recordMetrics(at)
	if c.Health != nil {
		c.Health.RecordSuccess("stats", time.Since(start))
	}
	c.Log.Info("stats collected",
		"running", len(running),
		"ok", okCount,
		"duration_ms", time.Since(start).Milliseconds(),
	)
}

func (c *StatsCollector) recordMetrics(at time.Time) {
	if c.Metrics == nil {
		return
	}
	snap := c.Store.Load()
	samples := make([]MetricsContainerSample, 0, len(snap.Containers))
	for _, ctr := range snap.Containers {
		if ctr.Stats == nil {
			continue
		}
		samples = append(samples, MetricsContainerSample{
			ID:    ctr.ID,
			Name:  ctr.Name,
			Stack: ctr.Stack,
			CPU:   ctr.Stats.CPUPercent,
			Mem:   ctr.Stats.MemoryBytes,
			NetRx: ctr.Stats.NetworkRxBytes,
			NetTx: ctr.Stats.NetworkTxBytes,
		})
	}
	host := c.HostName
	if host == "" {
		host = "default"
	}
	if err := c.Metrics.Record(host, at, samples); err != nil {
		c.Log.Warn("metrics record failed", "host", host, "err", err)
	}
}

func (c *StatsCollector) publishStats() {
	if c.Hub == nil {
		return
	}
	snap := c.Store.Load()
	items := make([]ws.StatsItem, 0, len(snap.Containers))
	for _, ctr := range snap.Containers {
		if ctr.Stats == nil {
			continue
		}
		st := *ctr.Stats
		items = append(items, ws.StatsItem{
			ID:      ctr.ID,
			IDShort: ctr.IDShort,
			Name:    ctr.Name,
			Stack:   ctr.Stack,
			Stats:   &st,
		})
	}
	c.Hub.PublishStats(items)
	c.Hub.PublishSnapshotUpdated(ws.SnapshotNotice{
		Version:    snap.Version,
		SnapshotAt: formatTime(snap.CollectedAt),
		StatsAt:    formatTime(snap.StatsAt),
		SystemAt:   formatTime(snap.SystemAt),
		Kind:       "stats",
	})
}

func (c *StatsCollector) fetchOne(ctx context.Context, id string) (domain.ContainerStats, error) {
	// IncludePreviousSample waits ~1s on the daemon for PreCPUStats (same as docker stats --no-stream).
	timeout := 15 * time.Second
	if t := c.Docker.Timeout(); t > timeout {
		timeout = t
	}
	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	res, err := c.Docker.API().ContainerStats(reqCtx, id, client.ContainerStatsOptions{
		Stream:                false,
		IncludePreviousSample: true,
	})
	if err != nil {
		return domain.ContainerStats{}, err
	}
	defer res.Body.Close()

	var sample container.StatsResponse
	dec := json.NewDecoder(res.Body)
	if err := dec.Decode(&sample); err != nil {
		if err == io.EOF {
			return domain.ContainerStats{}, io.ErrUnexpectedEOF
		}
		return domain.ContainerStats{}, err
	}
	return mapper.FromStatsResponse(sample), nil
}
