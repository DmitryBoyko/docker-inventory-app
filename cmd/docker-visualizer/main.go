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

	"github.com/epm-games/docker-visualizer/internal/config"
	"github.com/epm-games/docker-visualizer/internal/hosts"
	"github.com/epm-games/docker-visualizer/internal/httpapi"
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

	hub := ws.NewHub(log)
	reg, err := hosts.Build(hosts.Options{
		Hosts:             cfg.DockerHosts,
		DockerConfigDir:   cfg.DockerConfigDir,
		DockerTimeout:     cfg.DockerTimeout,
		InventoryInterval: cfg.InventoryInterval,
		StatsInterval:     cfg.StatsInterval,
		SystemInterval:    cfg.SystemInterval,
		Hub:               hub,
		Log:               log,
	})
	if err != nil {
		log.Error("docker hosts", "err", err)
		os.Exit(1)
	}
	defer reg.Close()

	for _, name := range reg.Names() {
		rt, _ := reg.Get(name)
		st := rt.Docker.Ping(context.Background())
		if st.Connected {
			log.Info("docker ping ok", "name", name, "apiVersion", st.APIVersion, "osType", st.OSType)
		} else {
			log.Warn("docker ping failed at startup", "name", name, "error", st.Error)
		}
	}

	runCtx, cancelCollectors := context.WithCancel(context.Background())
	defer cancelCollectors()

	go hub.Run(runCtx.Done())
	for _, name := range reg.Names() {
		rt, _ := reg.Get(name)
		rt.StartCollectors(runCtx)
	}

	srv := &httpapi.Server{
		Hosts:         reg,
		Hub:           hub,
		Version:       version,
		Commit:        commit,
		Listen:        cfg.ListenAddr,
		AuthEnabled:   cfg.AuthEnabled(),
		DockerTimeout: cfg.DockerTimeout.String(),
		Intervals: map[string]string{
			"inventory": cfg.InventoryInterval.String(),
			"stats":     cfg.StatsInterval.String(),
			"system":    cfg.SystemInterval.String(),
		},
		StartedAt: startedAt,
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
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	go func() {
		log.Info("http listen",
			"addr", cfg.ListenAddr,
			"version", version,
			"commit", commit,
			"authEnabled", cfg.AuthEnabled(),
			"hosts", reg.Names(),
			"defaultHost", reg.Default,
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
