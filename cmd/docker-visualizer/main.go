package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/epm-games/docker-visualizer/internal/app"
	"github.com/epm-games/docker-visualizer/internal/collector"
	"github.com/epm-games/docker-visualizer/internal/config"
	"github.com/epm-games/docker-visualizer/internal/docker"
	"github.com/epm-games/docker-visualizer/internal/httpapi"
	"github.com/epm-games/docker-visualizer/internal/observability"
	"github.com/epm-games/docker-visualizer/internal/store"
	"github.com/epm-games/docker-visualizer/internal/uiembed"
	"github.com/epm-games/docker-visualizer/internal/ws"
)

// Injected via -ldflags at release time.
var (
	version = "dev"
	commit  = "none"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	startedAt := time.Now().UTC()

	cfg, err := config.Load(os.Args[1:], version)
	if err != nil {
		log.Error("config", "err", err)
		os.Exit(2)
	}

	if !config.IsLoopbackListen(cfg.ListenAddr) {
		log.Warn("listen address is not loopback; auth token required (ADR-013)",
			"addr", cfg.ListenAddr,
			"authEnabled", true,
		)
	} else if cfg.AuthEnabled() {
		log.Info("API auth enabled", "authEnabled", true)
	}

	cli, err := docker.Connect(docker.Options{
		ExplicitHost:    cfg.DockerHost,
		DockerConfigDir: cfg.DockerConfigDir,
		Timeout:         cfg.DockerTimeout,
	})
	if err != nil {
		log.Error("docker connect", "err", err)
		os.Exit(1)
	}
	defer cli.Close()

	ep := cli.Endpoint()
	log.Info("docker endpoint resolved",
		"host", ep.Host,
		"source", ep.Source,
		"context", ep.Context,
	)

	st := cli.Ping(context.Background())
	if st.Connected {
		log.Info("docker ping ok", "apiVersion", st.APIVersion, "osType", st.OSType)
	} else {
		log.Warn("docker ping failed at startup; /ready will return 503 until daemon is available",
			"error", st.Error,
		)
	}

	snapStore := store.New()
	hub := ws.NewHub(log)
	health := observability.NewRegistry()

	containersSvc := &app.ContainersService{Store: snapStore}
	liveSvc := &app.ContainerLiveService{Docker: cli, Store: snapStore}
	stacksSvc := &app.StacksService{Store: snapStore}
	networksSvc := &app.NetworksService{Store: snapStore}
	volumesSvc := &app.VolumesService{Store: snapStore}
	imagesSvc := &app.ImagesService{Store: snapStore}
	systemSvc := &app.SystemService{Store: snapStore}
	graphSvc := &app.GraphService{Store: snapStore}
	diagSvc := &app.DiagnosticsService{
		Store:   snapStore,
		Docker:  cli,
		Hub:     hub,
		Health:  health,
		Version: version,
		Commit:  commit,
		Listen:  cfg.ListenAddr,
		Intervals: map[string]string{
			"inventory": cfg.InventoryInterval.String(),
			"stats":     cfg.StatsInterval.String(),
			"system":    cfg.SystemInterval.String(),
		},
		DockerTimeout: cfg.DockerTimeout.String(),
		AuthEnabled:   cfg.AuthEnabled(),
		StartedAt:     startedAt,
	}

	inv := &collector.InventoryCollector{
		Docker: cli, Store: snapStore, Hub: hub, Interval: cfg.InventoryInterval, Log: log, Health: health,
	}
	statsCol := &collector.StatsCollector{
		Docker: cli, Store: snapStore, Hub: hub, Interval: cfg.StatsInterval, Log: log, Health: health,
	}
	sysCol := &collector.SystemCollector{
		Docker: cli, Store: snapStore, Hub: hub, Interval: cfg.SystemInterval, Log: log, Health: health,
	}
	eventsCol := &collector.EventsCollector{
		Docker: cli, Hub: hub, Inventory: inv, System: sysCol, Log: log, Health: health,
	}
	diagSvc.Events = eventsCol

	runCtx, cancelCollectors := context.WithCancel(context.Background())
	defer cancelCollectors()

	go hub.Run(runCtx.Done())
	go inv.Run(runCtx)
	go statsCol.Run(runCtx)
	go sysCol.Run(runCtx)
	go eventsCol.Run(runCtx)
	go (&collector.ConnectionPublisher{Docker: cli, Hub: hub, Log: log}).Run(runCtx)

	srv := &httpapi.Server{
		Docker:      cli,
		Containers:  containersSvc,
		Live:        liveSvc,
		Stacks:      stacksSvc,
		Networks:    networksSvc,
		Volumes:     volumesSvc,
		Images:      imagesSvc,
		System:      systemSvc,
		Graph:       graphSvc,
		Diagnostics: diagSvc,
		Hub:         hub,
		Events:      eventsCol,
		Version:     version,
	}
	handler := attachUI(srv.Handler(), log)
	handler = httpapi.Chain(handler,
		httpapi.WithAuth(cfg.AuthToken),
		httpapi.WithRecover(log),
		httpapi.WithRequestID,
		httpapi.WithAccessLog(log),
	)

	httpSrv := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		// ReadTimeout/WriteTimeout left unset: long-lived WebSocket on /api/v1/ws.
		IdleTimeout:    120 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1 MiB
	}

	go func() {
		log.Info("http listen",
			"addr", cfg.ListenAddr,
			"version", version,
			"commit", commit,
			"authEnabled", cfg.AuthEnabled(),
			"inventory_interval", cfg.InventoryInterval.String(),
			"stats_interval", cfg.StatsInterval.String(),
			"system_interval", cfg.SystemInterval.String(),
		)
		if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("http server", "err", err)
			os.Exit(1)
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()

	cancelCollectors()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpSrv.Shutdown(shutdownCtx); err != nil {
		log.Error("shutdown", "err", err)
		os.Exit(1)
	}
	log.Info("shutdown complete")
}

// attachUI prefers on-disk web/dist (dev override), else embedded SPA (release).
func attachUI(api http.Handler, log *slog.Logger) http.Handler {
	if staticDir := resolveWebDist(); httpapi.StaticDirExists(staticDir) {
		log.Info("serving web UI from disk", "dir", staticDir)
		return httpapi.WithStatic(api, staticDir)
	}
	if uiembed.Available() {
		fsys, err := uiembed.FS()
		if err != nil {
			log.Warn("embedded UI unavailable", "err", err)
			return api
		}
		log.Info("serving embedded web UI")
		return httpapi.WithFS(api, fsys)
	}
	log.Warn("no web UI found; API-only mode (run: make ui && make build)")
	return api
}

func resolveWebDist() string {
	candidates := []string{"web/dist"}
	if exe, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Join(filepath.Dir(exe), "web", "dist"))
	}
	for _, c := range candidates {
		if httpapi.StaticDirExists(c) {
			return c
		}
	}
	return "web/dist"
}
