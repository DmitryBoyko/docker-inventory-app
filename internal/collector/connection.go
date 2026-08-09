package collector

import (
	"context"
	"log/slog"
	"time"

	"github.com/epm-games/docker-visualizer/internal/docker"
	"github.com/epm-games/docker-visualizer/internal/ws"
)

// ConnectionPublisher periodically pings the Engine and pushes connection.status.
type ConnectionPublisher struct {
	Docker   *docker.Client
	Hub      *ws.Hub
	Interval time.Duration
	Log      *slog.Logger
}

// Run until ctx cancelled.
func (c *ConnectionPublisher) Run(ctx context.Context) {
	if c.Interval <= 0 {
		c.Interval = 5 * time.Second
	}
	if c.Log == nil {
		c.Log = slog.Default()
	}
	c.publish(ctx)
	t := time.NewTicker(c.Interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			c.publish(ctx)
		}
	}
}

func (c *ConnectionPublisher) publish(ctx context.Context) {
	if c.Hub == nil {
		return
	}
	st := c.Docker.Ping(ctx)
	c.Hub.PublishConnectionStatus(st)
}
