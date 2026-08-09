package ws

import (
	"context"
	"encoding/json"
	"sync"
	"sync/atomic"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

// Client is one WebSocket subscriber.
type Client struct {
	hub  *Hub
	conn *websocket.Conn

	mu      sync.RWMutex
	subs    map[string]*StatsFilters // channel → optional filters (stats only)
	send   chan Envelope
	closed atomic.Bool
	drops  atomic.Int32
}

// NewClient wraps a websocket connection.
func NewClient(hub *Hub, conn *websocket.Conn) *Client {
	c := &Client{
		hub:  hub,
		conn: conn,
		subs: map[string]*StatsFilters{
			// Auto-subscribe lightweight channels; stats requires explicit subscribe.
			ChannelEvents:     {},
			ChannelSnapshots:  {},
			ChannelConnection: {},
		},
		send: make(chan Envelope, clientSendBuffer),
	}
	return c
}

func (c *Client) accepts(env Envelope) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	switch env.Type {
	case TypeContainerStats:
		_, ok := c.subs[ChannelStats]
		return ok
	case TypeDockerEvent:
		_, ok := c.subs[ChannelEvents]
		return ok
	case TypeSnapshotUpdated:
		_, ok := c.subs[ChannelSnapshots]
		return ok
	case TypeConnectionStatus, TypeEventsStatus:
		_, ok := c.subs[ChannelConnection]
		return ok
	case TypePing, TypePong, TypeError:
		return true
	default:
		return true
	}
}

func (c *Client) filterStats(env Envelope) []StatsItem {
	items, ok := env.Data.([]StatsItem)
	if !ok {
		// JSON round-trip path shouldn't happen in-process.
		return nil
	}
	c.mu.RLock()
	f := c.subs[ChannelStats]
	c.mu.RUnlock()
	if f == nil {
		return items
	}
	idSet := map[string]struct{}{}
	for _, id := range f.ContainerIDs {
		if id != "" {
			idSet[id] = struct{}{}
		}
	}
	out := make([]StatsItem, 0, len(items))
	for _, it := range items {
		if f.Stack != "" && it.Stack != f.Stack {
			continue
		}
		if len(idSet) > 0 {
			if _, ok := idSet[it.ID]; !ok {
				if _, ok2 := idSet[it.IDShort]; !ok2 {
					continue
				}
			}
		}
		out = append(out, it)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func (c *Client) trySend(env Envelope) {
	if c.closed.Load() {
		return
	}
	select {
	case c.send <- env:
	default:
		// Drop oldest then enqueue newest.
		select {
		case <-c.send:
		default:
		}
		select {
		case c.send <- env:
		default:
		}
		n := c.drops.Add(1)
		if n >= maxDrops {
			c.hub.log.Warn("ws client too slow; disconnecting", "drops", n)
			_ = c.conn.Close(websocket.StatusPolicyViolation, "too many dropped messages")
			c.hub.Unregister(c)
		}
	}
}

func (c *Client) closeSend() {
	if c.closed.Swap(true) {
		return
	}
	close(c.send)
}

// ReadPump handles client→server messages until disconnect.
func (c *Client) ReadPump(ctx context.Context) {
	defer c.hub.Unregister(c)
	for {
		var msg ClientMessage
		if err := wsjson.Read(ctx, c.conn, &msg); err != nil {
			return
		}
		c.handleClientMessage(msg)
	}
}

// WritePump drains the outbound queue.
func (c *Client) WritePump(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			_ = c.conn.Close(websocket.StatusGoingAway, "shutdown")
			return
		case env, ok := <-c.send:
			if !ok {
				_ = c.conn.Close(websocket.StatusNormalClosure, "")
				return
			}
			wctx, cancel := context.WithTimeout(ctx, 5*time.Second)
			err := wsjson.Write(wctx, c.conn, env)
			cancel()
			if err != nil {
				return
			}
		}
	}
}

func (c *Client) handleClientMessage(msg ClientMessage) {
	switch msg.Action {
	case ActionPing:
		c.trySend(Envelope{Type: TypePong, Timestamp: nowTS()})
	case ActionSubscribe:
		c.subscribe(msg.Channel, msg.Filters)
	case ActionUnsubscribe:
		c.unsubscribe(msg.Channel)
	default:
		c.trySend(Envelope{
			Type:      TypeError,
			Timestamp: nowTS(),
			Data:      map[string]string{"message": "unknown action"},
		})
	}
}

func (c *Client) subscribe(channel string, filters *StatsFilters) {
	switch channel {
	case ChannelStats, ChannelEvents, ChannelSnapshots, ChannelConnection:
	default:
		c.trySend(Envelope{
			Type:      TypeError,
			Timestamp: nowTS(),
			Data:      map[string]string{"message": "unknown channel"},
		})
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if channel == ChannelStats {
		c.subs[channel] = filters
	} else {
		c.subs[channel] = &StatsFilters{}
	}
}

func (c *Client) unsubscribe(channel string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.subs, channel)
}

// MarshalClientMessage is a test helper.
func MarshalClientMessage(msg ClientMessage) ([]byte, error) {
	return json.Marshal(msg)
}
