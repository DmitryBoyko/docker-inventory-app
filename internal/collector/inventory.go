package collector

import (
	"context"
	"fmt"
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

const defaultInspectConcurrency = 16

// InventoryCollector lists containers (+ size) and enriches via inspect.
type InventoryCollector struct {
	Docker      *docker.Client
	Store       *store.Store
	Hub         *ws.Hub
	Interval    time.Duration
	Concurrency int
	Log         *slog.Logger
	Health      *observability.Registry

	mu        sync.Mutex
	refreshCh chan struct{}
}

// Run polls until ctx is cancelled. Performs an immediate first collect.
func (c *InventoryCollector) Run(ctx context.Context) {
	if c.Interval <= 0 {
		c.Interval = 10 * time.Second
	}
	if c.Concurrency <= 0 {
		c.Concurrency = defaultInspectConcurrency
	}
	if c.Log == nil {
		c.Log = slog.Default()
	}
	c.ensureRefreshCh()

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

// RequestRefresh asks for an out-of-band collect (events). Coalesced (buffer 1).
func (c *InventoryCollector) RequestRefresh() {
	c.ensureRefreshCh()
	select {
	case c.refreshCh <- struct{}{}:
	default:
	}
}

// Refresh triggers a single collection (tests / events).
func (c *InventoryCollector) Refresh(ctx context.Context) error {
	return c.collectOnce(ctx)
}

func (c *InventoryCollector) ensureRefreshCh() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.refreshCh == nil {
		c.refreshCh = make(chan struct{}, 1)
	}
}

func (c *InventoryCollector) collectOnce(ctx context.Context) error {
	start := time.Now()
	api := c.Docker.API()

	listCtx, cancel := context.WithTimeout(ctx, c.DockerTimeout())
	defer cancel()

	res, err := api.ContainerList(listCtx, client.ContainerListOptions{
		All:  true,
		Size: true,
	})
	if err != nil {
		msg := docker.ClassifyError(c.Docker.Endpoint().Host, err)
		c.Log.Warn("inventory list failed", "err", msg)
		// Keep previous snapshot; record error on a touch if empty.
		snap := c.Store.Load()
		if !snap.HasData() {
			c.Store.Replace(nil, time.Now().UTC(), msg)
		}
		if c.Health != nil {
			c.Health.RecordError("inventory", time.Since(start), err)
		}
		return err
	}

	containers := make([]domain.Container, 0, len(res.Items))
	for _, item := range res.Items {
		containers = append(containers, mapper.FromSummary(item))
	}

	if err := c.enrichAll(ctx, containers); err != nil {
		c.Log.Warn("inventory inspect enrich partial", "err", err)
	}

	networks := c.listNetworks(listCtx)
	images := c.listImages(listCtx)

	c.Store.ReplaceInventory(containers, networks, images, time.Now().UTC(), "")
	c.publishSnapshot("inventory")
	if c.Health != nil {
		c.Health.RecordSuccess("inventory", time.Since(start))
	}
	c.Log.Info("inventory collected",
		"containers", len(containers),
		"networks", len(networks),
		"images", len(images),
		"duration_ms", time.Since(start).Milliseconds(),
	)
	return nil
}

func (c *InventoryCollector) listNetworks(ctx context.Context) []domain.Network {
	res, err := c.Docker.API().NetworkList(ctx, client.NetworkListOptions{})
	if err != nil {
		c.Log.Warn("network list failed", "err", docker.ClassifyError(c.Docker.Endpoint().Host, err))
		// Preserve previous networks on transient failure.
		return append([]domain.Network(nil), c.Store.Load().Networks...)
	}
	out := make([]domain.Network, 0, len(res.Items))
	for _, item := range res.Items {
		out = append(out, mapper.FromNetworkSummary(item))
	}
	return out
}

func (c *InventoryCollector) listImages(ctx context.Context) []domain.Image {
	res, err := c.Docker.API().ImageList(ctx, client.ImageListOptions{})
	if err != nil {
		c.Log.Warn("image list failed", "err", docker.ClassifyError(c.Docker.Endpoint().Host, err))
		return append([]domain.Image(nil), c.Store.Load().Images...)
	}
	out := make([]domain.Image, 0, len(res.Items))
	for _, item := range res.Items {
		out = append(out, mapper.FromImageSummary(item))
	}
	return out
}

func (c *InventoryCollector) publishSnapshot(kind string) {
	if c.Hub == nil {
		return
	}
	snap := c.Store.Load()
	c.Hub.PublishSnapshotUpdated(ws.SnapshotNotice{
		Version:    snap.Version,
		SnapshotAt: formatTime(snap.CollectedAt),
		StatsAt:    formatTime(snap.StatsAt),
		SystemAt:   formatTime(snap.SystemAt),
		Kind:       kind,
	})
}

func (c *InventoryCollector) enrichAll(ctx context.Context, containers []domain.Container) error {
	if len(containers) == 0 {
		return nil
	}
	sem := make(chan struct{}, c.Concurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var firstErr error

	for i := range containers {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			select {
			case <-ctx.Done():
				return
			case sem <- struct{}{}:
			}
			defer func() { <-sem }()

			inspCtx, cancel := context.WithTimeout(ctx, c.DockerTimeout())
			defer cancel()
			res, err := c.Docker.API().ContainerInspect(inspCtx, containers[i].ID, client.ContainerInspectOptions{
				Size: true,
			})
			if err != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = err
				}
				mu.Unlock()
				return
			}
			mapper.EnrichFromInspect(&containers[i], res.Container)
		}()
	}
	wg.Wait()
	if firstErr != nil {
		return fmt.Errorf("inspect: %w", firstErr)
	}
	return nil
}

func (c *InventoryCollector) DockerTimeout() time.Duration {
	if t := c.Docker.Timeout(); t > 0 {
		return t
	}
	return 10 * time.Second
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339Nano)
}
