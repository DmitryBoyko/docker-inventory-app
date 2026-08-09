package docker

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/epm-games/docker-visualizer/internal/domain"
	"github.com/moby/moby/client"
)

// Client wraps the Moby Engine client with discovery metadata and ping helpers.
type Client struct {
	api      *client.Client
	endpoint Endpoint
	timeout  time.Duration

	mu     sync.RWMutex
	status domain.ConnectionStatus
}

// Options configures Engine connection.
type Options struct {
	ExplicitHost    string
	DockerConfigDir string
	Timeout         time.Duration
}

// Connect resolves the endpoint and creates an Engine API client (ADR-003, ADR-010).
func Connect(opts Options) (*Client, error) {
	if opts.Timeout <= 0 {
		opts.Timeout = 5 * time.Second
	}

	ep, err := ResolveEndpoint(opts.ExplicitHost, opts.DockerConfigDir)
	if err != nil {
		return nil, err
	}

	api, err := client.New(
		client.WithTLSClientConfigFromEnv(),
		client.WithAPIVersionFromEnv(),
		client.WithTimeout(opts.Timeout),
		client.WithHost(ep.Host),
	)
	if err != nil {
		return nil, fmt.Errorf("create docker client for %s: %w", ep.Host, err)
	}

	c := &Client{
		api:      api,
		endpoint: ep,
		timeout:  opts.Timeout,
		status: domain.ConnectionStatus{
			Connected: false,
			Host:      ep.Host,
			Source:    ep.Source,
			Context:   ep.Context,
			CheckedAt: time.Now().UTC(),
		},
	}
	return c, nil
}

// API returns the underlying Moby client for later collectors.
func (c *Client) API() *client.Client { return c.api }

// Endpoint returns the resolved endpoint.
func (c *Client) Endpoint() Endpoint { return c.endpoint }

// Timeout returns the configured Engine request timeout.
func (c *Client) Timeout() time.Duration { return c.timeout }

// Close releases idle connections.
func (c *Client) Close() error {
	if c == nil || c.api == nil {
		return nil
	}
	return c.api.Close()
}

// Ping checks Engine liveness and negotiates API version.
func (c *Client) Ping(ctx context.Context) domain.ConnectionStatus {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	now := time.Now().UTC()
	st := domain.ConnectionStatus{
		Host:      c.endpoint.Host,
		Source:    c.endpoint.Source,
		Context:   c.endpoint.Context,
		CheckedAt: now,
	}

	res, err := c.api.Ping(ctx, client.PingOptions{NegotiateAPIVersion: true})
	if err != nil {
		st.Connected = false
		st.Error = ClassifyError(c.endpoint.Host, err)
		c.setStatus(st)
		return st
	}

	st.Connected = true
	st.APIVersion = res.APIVersion
	st.OSType = res.OSType
	c.setStatus(st)
	return st
}

// Status returns the last ping result (or initial unresolved state).
func (c *Client) Status() domain.ConnectionStatus {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.status
}

func (c *Client) setStatus(st domain.ConnectionStatus) {
	c.mu.Lock()
	c.status = st
	c.mu.Unlock()
}
