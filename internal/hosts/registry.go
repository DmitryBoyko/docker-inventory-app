package hosts

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/epm-games/docker-visualizer/internal/app"
	"github.com/epm-games/docker-visualizer/internal/collector"
	"github.com/epm-games/docker-visualizer/internal/config"
	"github.com/epm-games/docker-visualizer/internal/docker"
	"github.com/epm-games/docker-visualizer/internal/observability"
	"github.com/epm-games/docker-visualizer/internal/store"
	"github.com/epm-games/docker-visualizer/internal/ws"
)

// Info is a public host list entry.
type Info struct {
	Name      string `json:"name"`
	Endpoint  string `json:"endpoint"`
	Source    string `json:"source"`
	Context   string `json:"context,omitempty"`
	IsDefault bool   `json:"isDefault"`
	Connected bool   `json:"connected"`
	Error     string `json:"error,omitempty"`
}

// Runtime is one Engine + snapshot + services.
type Runtime struct {
	Name       string
	Docker     *docker.Client
	Store      *store.Store
	Containers *app.ContainersService
	Live       *app.ContainerLiveService
	Stacks     *app.StacksService
	Networks   *app.NetworksService
	Volumes    *app.VolumesService
	Images     *app.ImagesService
	System     *app.SystemService
	Graph      *app.GraphService
	Export     *app.ExportService
	Events     interface {
		Connected() bool
		LastError() string
	}
	Health *observability.Registry
	Bus    ws.Bus

	startFn func(ctx context.Context)
}

// StartCollectors launches per-host collectors until ctx is cancelled.
func (rt *Runtime) StartCollectors(ctx context.Context) {
	if rt != nil && rt.startFn != nil {
		rt.startFn(ctx)
	}
}

// Registry holds named host runtimes.
type Registry struct {
	Default string
	order   []string
	byName  map[string]*Runtime
}

// Get resolves host name (empty → default).
func (r *Registry) Get(name string) (*Runtime, error) {
	if r == nil || len(r.byName) == 0 {
		return nil, fmt.Errorf("no hosts configured")
	}
	name = strings.TrimSpace(name)
	if name == "" {
		name = r.Default
	}
	if rt, ok := r.byName[name]; ok {
		return rt, nil
	}
	for k, v := range r.byName {
		if strings.EqualFold(k, name) {
			return v, nil
		}
	}
	return nil, fmt.Errorf("unknown host %q", name)
}

// Names returns host names in config order.
func (r *Registry) Names() []string {
	return append([]string(nil), r.order...)
}

// List returns public host infos (connection from last ping cache).
func (r *Registry) List() []Info {
	out := make([]Info, 0, len(r.order))
	for _, name := range r.order {
		rt := r.byName[name]
		st := rt.Docker.Status()
		ep := rt.Docker.Endpoint()
		out = append(out, Info{
			Name:      name,
			Endpoint:  ep.Host,
			Source:    ep.Source,
			Context:   ep.Context,
			IsDefault: name == r.Default,
			Connected: st.Connected,
			Error:     st.Error,
		})
	}
	return out
}

// Close closes all Engine clients.
func (r *Registry) Close() {
	if r == nil {
		return
	}
	for _, rt := range r.byName {
		_ = rt.Docker.Close()
	}
}

// ForTest builds a single-host registry for unit tests (no collectors).
func ForTest(name string, rt *Runtime) *Registry {
	if name == "" {
		name = "default"
	}
	if rt == nil {
		rt = &Runtime{}
	}
	rt.Name = name
	return &Registry{
		Default: name,
		order:   []string{name},
		byName:  map[string]*Runtime{name: rt},
	}
}

// Options for building a registry.
type Options struct {
	Hosts             []config.HostSpec
	DockerConfigDir   string
	DockerTimeout     time.Duration
	InventoryInterval time.Duration
	StatsInterval     time.Duration
	SystemInterval    time.Duration
	Hub               *ws.Hub
	Log               *slog.Logger
}

// Build connects each configured host and prepares collectors (call StartCollectors).
func Build(opts Options) (*Registry, error) {
	if opts.Log == nil {
		opts.Log = slog.Default()
	}
	if len(opts.Hosts) == 0 {
		return nil, fmt.Errorf("no hosts")
	}
	reg := &Registry{
		Default: opts.Hosts[0].Name,
		order:   make([]string, 0, len(opts.Hosts)),
		byName:  map[string]*Runtime{},
	}
	for _, spec := range opts.Hosts {
		cli, err := docker.Connect(docker.Options{
			ExplicitHost:    spec.URL,
			DockerConfigDir: opts.DockerConfigDir,
			Timeout:         opts.DockerTimeout,
		})
		if err != nil {
			reg.Close()
			return nil, fmt.Errorf("host %s: %w", spec.Name, err)
		}
		bus := opts.Hub.Bind(spec.Name)
		st := store.New()
		health := observability.NewRegistry()
		inv := &collector.InventoryCollector{
			Docker: cli, Store: st, Hub: bus, Interval: opts.InventoryInterval, Log: opts.Log, Health: health,
		}
		statsCol := &collector.StatsCollector{
			Docker: cli, Store: st, Hub: bus, Interval: opts.StatsInterval, Log: opts.Log, Health: health,
		}
		sysCol := &collector.SystemCollector{
			Docker: cli, Store: st, Hub: bus, Interval: opts.SystemInterval, Log: opts.Log, Health: health,
		}
		eventsCol := &collector.EventsCollector{
			Docker: cli, Hub: bus, Inventory: inv, System: sysCol, Log: opts.Log, Health: health,
		}
		connPub := &collector.ConnectionPublisher{Docker: cli, Hub: bus, Log: opts.Log}
		name := spec.Name
		rt := &Runtime{
			Name:       name,
			Docker:     cli,
			Store:      st,
			Containers: &app.ContainersService{Store: st},
			Live:       &app.ContainerLiveService{Docker: cli, Store: st},
			Stacks:     &app.StacksService{Store: st},
			Networks:   &app.NetworksService{Store: st},
			Volumes:    &app.VolumesService{Store: st},
			Images:     &app.ImagesService{Store: st},
			System:     &app.SystemService{Store: st},
			Graph:      &app.GraphService{Store: st},
			Export:     &app.ExportService{Store: st},
			Events:     eventsCol,
			Health:     health,
			Bus:        bus,
			startFn: func(ctx context.Context) {
				go inv.Run(ctx)
				go statsCol.Run(ctx)
				go sysCol.Run(ctx)
				go eventsCol.Run(ctx)
				go connPub.Run(ctx)
			},
		}
		reg.order = append(reg.order, name)
		reg.byName[name] = rt
		opts.Log.Info("docker host registered",
			"name", name,
			"endpoint", cli.Endpoint().Host,
			"source", cli.Endpoint().Source,
		)
	}
	return reg, nil
}
