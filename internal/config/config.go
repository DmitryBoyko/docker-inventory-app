package config

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	EnvDockerHost         = "DOCKER_VISUALIZER_DOCKER_HOST"
	EnvListen             = "DOCKER_VISUALIZER_LISTEN"
	EnvTimeout            = "DOCKER_VISUALIZER_DOCKER_TIMEOUT"
	EnvConfigDir          = "DOCKER_VISUALIZER_DOCKER_CONFIG"
	EnvInventoryInterval  = "DOCKER_VISUALIZER_INVENTORY_INTERVAL"
	EnvStatsInterval      = "DOCKER_VISUALIZER_STATS_INTERVAL"
	EnvSystemInterval     = "DOCKER_VISUALIZER_SYSTEM_INTERVAL"
	EnvAuthToken          = "DOCKER_VISUALIZER_AUTH_TOKEN"
)

// Config holds process configuration.
type Config struct {
	ListenAddr        string
	DockerHost        string        // explicit override (flag / DOCKER_VISUALIZER_DOCKER_HOST)
	DockerConfigDir   string        // optional ~/.docker override
	DockerTimeout     time.Duration // dial/request timeout for Engine calls
	InventoryInterval time.Duration // container inventory poll (default 10s)
	StatsInterval     time.Duration // running stats poll (default 1s)
	SystemInterval    time.Duration // volumes/df poll (default 15s)
	AuthToken         string        // Bearer token; required for non-loopback listen (ADR-013)
	Version           string
}

// Load parses flags and environment. Flags win over env for the same key.
func Load(args []string, version string) (*Config, error) {
	fs := flag.NewFlagSet("docker-visualizer", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	listen := fs.String("listen", envOr(EnvListen, "127.0.0.1:8080"), "HTTP listen address (ADR-009 default 127.0.0.1)")
	dockerHost := fs.String("docker-host", envOr(EnvDockerHost, ""), "Docker host URL (overrides DOCKER_HOST and context)")
	dockerConfig := fs.String("docker-config", envOr(EnvConfigDir, ""), "Docker config directory (default: ~/.docker)")
	timeoutStr := fs.String("docker-timeout", envOr(EnvTimeout, "5s"), "Timeout for Docker Engine requests")
	invStr := fs.String("inventory-interval", envOr(EnvInventoryInterval, "10s"), "Container inventory refresh interval")
	statsStr := fs.String("stats-interval", envOr(EnvStatsInterval, "1s"), "Running container stats refresh interval")
	sysStr := fs.String("system-interval", envOr(EnvSystemInterval, "15s"), "Volumes / disk-usage refresh interval")
	authToken := fs.String("auth-token", envOr(EnvAuthToken, ""), "Bearer token for API/WS (required when listen is not loopback)")

	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	timeout, err := parseDuration(*timeoutStr)
	if err != nil {
		return nil, fmt.Errorf("invalid --docker-timeout %q: %w", *timeoutStr, err)
	}
	if timeout <= 0 {
		return nil, fmt.Errorf("docker-timeout must be positive")
	}
	inv, err := parseDuration(*invStr)
	if err != nil {
		return nil, fmt.Errorf("invalid --inventory-interval %q: %w", *invStr, err)
	}
	if inv <= 0 {
		return nil, fmt.Errorf("inventory-interval must be positive")
	}
	stats, err := parseDuration(*statsStr)
	if err != nil {
		return nil, fmt.Errorf("invalid --stats-interval %q: %w", *statsStr, err)
	}
	if stats <= 0 {
		return nil, fmt.Errorf("stats-interval must be positive")
	}
	sys, err := parseDuration(*sysStr)
	if err != nil {
		return nil, fmt.Errorf("invalid --system-interval %q: %w", *sysStr, err)
	}
	if sys <= 0 {
		return nil, fmt.Errorf("system-interval must be positive")
	}

	cfg := &Config{
		ListenAddr:        *listen,
		DockerHost:        *dockerHost,
		DockerConfigDir:   *dockerConfig,
		DockerTimeout:     timeout,
		InventoryInterval: inv,
		StatsInterval:     stats,
		SystemInterval:    sys,
		AuthToken:         strings.TrimSpace(*authToken),
		Version:           version,
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

// AuthEnabled reports whether API/WS auth middleware should enforce a token.
func (c *Config) AuthEnabled() bool {
	return c != nil && c.AuthToken != ""
}

// Validate enforces ADR-013: non-loopback listen requires an auth token.
func (c *Config) Validate() error {
	if c == nil {
		return fmt.Errorf("nil config")
	}
	if !IsLoopbackListen(c.ListenAddr) && c.AuthToken == "" {
		return fmt.Errorf("non-loopback listen %q requires --auth-token / %s (ADR-013)", c.ListenAddr, EnvAuthToken)
	}
	return nil
}

func parseDuration(s string) (time.Duration, error) {
	d, err := time.ParseDuration(s)
	if err == nil {
		return d, nil
	}
	if sec, err2 := strconv.Atoi(s); err2 == nil {
		return time.Duration(sec) * time.Second, nil
	}
	return 0, err
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
