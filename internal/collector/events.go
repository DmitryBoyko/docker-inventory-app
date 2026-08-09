package collector

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"sync/atomic"
	"time"

	"github.com/epm-games/docker-visualizer/internal/docker"
	"github.com/epm-games/docker-visualizer/internal/observability"
	"github.com/epm-games/docker-visualizer/internal/ws"
	"github.com/moby/moby/api/types/events"
	"github.com/moby/moby/client"
)

// EventsCollector streams Docker events for cache invalidation (ADR-006).
type EventsCollector struct {
	Docker    *docker.Client
	Hub       *ws.Hub
	Inventory *InventoryCollector
	System    *SystemCollector
	Coalesce  time.Duration
	Log       *slog.Logger
	Health    *observability.Registry

	connected atomic.Bool
	lastErr   atomic.Value // string
}

// Connected reports whether the events stream is currently live.
func (c *EventsCollector) Connected() bool { return c.connected.Load() }

// LastError returns the last stream error (empty when healthy).
func (c *EventsCollector) LastError() string {
	if v := c.lastErr.Load(); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// Run reconnects with backoff until ctx is cancelled.
func (c *EventsCollector) Run(ctx context.Context) {
	if c.Coalesce <= 0 {
		c.Coalesce = 250 * time.Millisecond
	}
	if c.Log == nil {
		c.Log = slog.Default()
	}

	backoff := time.Second
	for {
		if ctx.Err() != nil {
			c.setConnected(false, "stopped")
			return
		}
		err := c.streamOnce(ctx)
		if ctx.Err() != nil {
			c.setConnected(false, "stopped")
			return
		}
		msg := docker.ClassifyError(c.Docker.Endpoint().Host, err)
		c.setConnected(false, msg)
		c.Log.Warn("docker events stream ended; reconnecting", "err", msg, "backoff", backoff.String())
		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}
		if backoff < 30*time.Second {
			backoff *= 2
			if backoff > 30*time.Second {
				backoff = 30 * time.Second
			}
		}
	}
}

func (c *EventsCollector) streamOnce(ctx context.Context) error {
	filters := make(client.Filters).
		Add("type",
			string(events.ContainerEventType),
			string(events.NetworkEventType),
			string(events.VolumeEventType),
			string(events.ImageEventType),
		)

	res := c.Docker.API().Events(ctx, client.EventsListOptions{Filters: filters})
	c.setConnected(true, "")
	c.Log.Info("docker events stream connected")

	var (
		dirtyInventory bool
		dirtySystem    bool
		timer          *time.Timer
		timerC         <-chan time.Time
	)
	stopTimer := func() {
		if timer != nil {
			timer.Stop()
			timer = nil
			timerC = nil
		}
	}
	defer stopTimer()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err, ok := <-res.Err:
			if !ok {
				return errors.New("events error channel closed")
			}
			if err == nil || errors.Is(err, io.EOF) || errors.Is(err, context.Canceled) {
				return err
			}
			return err
		case msg, ok := <-res.Messages:
			if !ok {
				return errors.New("events message channel closed")
			}
			c.handleMessage(msg, &dirtyInventory, &dirtySystem)
			if timer == nil {
				timer = time.NewTimer(c.Coalesce)
				timerC = timer.C
			} else {
				if !timer.Stop() {
					select {
					case <-timer.C:
					default:
					}
				}
				timer.Reset(c.Coalesce)
			}
		case <-timerC:
			stopTimer()
			if dirtyInventory && c.Inventory != nil {
				c.Inventory.RequestRefresh()
			}
			if dirtySystem && c.System != nil {
				c.System.RequestRefresh()
			}
			dirtyInventory = false
			dirtySystem = false
		}
	}
}

func (c *EventsCollector) handleMessage(msg events.Message, dirtyInv, dirtySys *bool) {
	ev := normalizeEvent(msg)
	if c.Hub != nil {
		c.Hub.PublishDockerEvent(ev)
	}

	switch events.Type(ev.Type) {
	case events.ContainerEventType:
		if interestingContainerAction(ev.Action) {
			*dirtyInv = true
		}
	case events.NetworkEventType:
		if interestingNetworkAction(ev.Action) {
			*dirtyInv = true
		}
	case events.VolumeEventType:
		*dirtySys = true
		*dirtyInv = true
	case events.ImageEventType:
		// Optional: inventory image refs may change after tag/delete.
		if ev.Action == string(events.ActionDelete) || ev.Action == string(events.ActionTag) || ev.Action == string(events.ActionUnTag) {
			*dirtyInv = true
		}
	}
}

func (c *EventsCollector) setConnected(ok bool, errMsg string) {
	c.connected.Store(ok)
	c.lastErr.Store(errMsg)
	if c.Hub != nil {
		c.Hub.PublishEventsStatus(ok, errMsg)
	}
	if c.Health == nil {
		return
	}
	if ok {
		c.Health.RecordSuccess("events", 0)
		return
	}
	if errMsg == "" || errMsg == "stopped" {
		return
	}
	c.Health.RecordError("events", 0, errors.New(errMsg))
}

func normalizeEvent(msg events.Message) ws.DockerEvent {
	attrs := msg.Actor.Attributes
	name := ""
	if attrs != nil {
		if n, ok := attrs["name"]; ok {
			name = n
		}
	}
	ts := time.Unix(msg.Time, 0).UTC()
	if msg.TimeNano > 0 {
		ts = time.Unix(0, msg.TimeNano).UTC()
	}
	return ws.DockerEvent{
		Type:      string(msg.Type),
		Action:    string(msg.Action),
		ActorID:   msg.Actor.ID,
		ActorName: name,
		Attrs:     attrs,
		Scope:     msg.Scope,
		Time:      ts.Format(time.RFC3339Nano),
	}
}

func interestingContainerAction(action string) bool {
	a := strings.ToLower(action)
	switch {
	case strings.HasPrefix(a, "health_status"):
		return true
	case a == "start", a == "stop", a == "die", a == "destroy", a == "kill",
		a == "pause", a == "unpause", a == "restart", a == "rename",
		a == "create", a == "update", a == "oom":
		return true
	default:
		return false
	}
}

func interestingNetworkAction(action string) bool {
	switch strings.ToLower(action) {
	case "connect", "disconnect", "create", "destroy", "remove":
		return true
	default:
		return false
	}
}
