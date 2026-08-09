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
	EnvDockerHost        = "DOCKER_VISUALIZER_DOCKER_HOST"
	EnvListen            = "DOCKER_VISUALIZER_LISTEN"
	EnvTimeout           = "DOCKER_VISUALIZER_DOCKER_TIMEOUT"
	EnvConfigDir         = "DOCKER_VISUALIZER_DOCKER_CONFIG"
	EnvInventoryInterval = "DOCKER_VISUALIZER_INVENTORY_INTERVAL"
	EnvStatsInterval     = "DOCKER_VISUALIZER_STATS_INTERVAL"
	EnvSystemInterval    = "DOCKER_VISUALIZER_SYSTEM_INTERVAL"
	EnvAuthToken         = "DOCKER_VISUALIZER_AUTH_TOKEN"
	EnvMetricsDB         = "DOCKER_VISUALIZER_METRICS_DB"
	EnvMetricsInterval   = "DOCKER_VISUALIZER_METRICS_INTERVAL"
	EnvMetricsRetention  = "DOCKER_VISUALIZER_METRICS_RETENTION"
	EnvSnapshotsDir      = "DOCKER_VISUALIZER_SNAPSHOTS_DIR"
)

// Config holds process configuration.
type Config struct {
	ListenAddr        string
	DockerHost        string        // explicit override for single-host / default entry
	DockerHosts       []HostSpec    // named registry (ADR-014); always ≥1 after Load
	DockerConfigDir   string        // optional ~/.docker override
	DockerTimeout     time.Duration // dial/request timeout for Engine calls
	InventoryInterval time.Duration // container inventory poll (default 10s)
	StatsInterval     time.Duration // running stats poll (default 1s)
	SystemInterval    time.Duration // volumes/df poll (default 15s)
	AuthToken         string        // Bearer token; required for non-loopback listen (ADR-013)
	MetricsDBPath     string        // SQLite path; off/empty disables (ADR-015)
	MetricsInterval   time.Duration // history write cadence (default 10s)
	MetricsRetention  time.Duration // history retention (default 24h)
	SnapshotsDir      string        // persisted inventory snapshots; off disables
	Version           string
}

// Load parses flags and environment. Flags win over env for the same key.
func Load(args []string, version string) (*Config, error) {
	fs := flag.NewFlagSet("docker-visualizer", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	listen := fs.String("listen", envOr(EnvListen, "127.0.0.1:8080"), "HTTP listen address (ADR-009 default 127.0.0.1)")
	dockerHost := fs.String("docker-host", envOr(EnvDockerHost, ""), "Docker host URL for default host (overrides DOCKER_HOST and context)")
	dockerHosts := fs.String("docker-hosts", envOr(EnvDockerHosts, ""), "Named hosts name=url,name2=url (ADR-014); empty ⇒ single default")
	dockerConfig := fs.String("docker-config", envOr(EnvConfigDir, ""), "Docker config directory (default: ~/.docker)")
	timeoutStr := fs.String("docker-timeout", envOr(EnvTimeout, "5s"), "Timeout for Docker Engine requests")
	invStr := fs.String("inventory-interval", envOr(EnvInventoryInterval, "10s"), "Container inventory refresh interval")
	statsStr := fs.String("stats-interval", envOr(EnvStatsInterval, "1s"), "Running container stats refresh interval")
	sysStr := fs.String("system-interval", envOr(EnvSystemInterval, "15s"), "Volumes / disk-usage refresh interval")
	authToken := fs.String("auth-token", envOr(EnvAuthToken, ""), "Bearer token for API/WS (required when listen is not loopback)")
	metricsDB := fs.String("metrics-db", envOr(EnvMetricsDB, "data/metrics.db"), "SQLite path for historical metrics (ADR-015); off disables")
	metricsIntStr := fs.String("metrics-interval", envOr(EnvMetricsInterval, "10s"), "Historical metrics sample interval")
	metricsRetStr := fs.String("metrics-retention", envOr(EnvMetricsRetention, "24h"), "Historical metrics retention")
	snapshotsDir := fs.String("snapshots-dir", envOr(EnvSnapshotsDir, "data/snapshots"), "Directory for inventory snapshots; off disables")

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
	metricsInt, err := parseDuration(*metricsIntStr)
	if err != nil {
		return nil, fmt.Errorf("invalid --metrics-interval %q: %w", *metricsIntStr, err)
	}
	if metricsInt <= 0 {
		return nil, fmt.Errorf("metrics-interval must be positive")
	}
	metricsRet, err := parseDuration(*metricsRetStr)
	if err != nil {
		return nil, fmt.Errorf("invalid --metrics-retention %q: %w", *metricsRetStr, err)
	}
	if metricsRet <= 0 {
		return nil, fmt.Errorf("metrics-retention must be positive")
	}

	hosts, err := ParseDockerHosts(*dockerHosts, *dockerHost)
	if err != nil {
		return nil, fmt.Errorf("docker-hosts: %w", err)
	}

	cfg := &Config{
		ListenAddr:        *listen,
		DockerHost:        *dockerHost,
		DockerHosts:       hosts,
		DockerConfigDir:   *dockerConfig,
		DockerTimeout:     timeout,
		InventoryInterval: inv,
		StatsInterval:     stats,
		SystemInterval:    sys,
		AuthToken:         strings.TrimSpace(*authToken),
		MetricsDBPath:     strings.TrimSpace(*metricsDB),
		MetricsInterval:   metricsInt,
		MetricsRetention:  metricsRet,
		SnapshotsDir:      strings.TrimSpace(*snapshotsDir),
		Version:           version,
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

// DefaultHostName returns the first configured host name.
func (c *Config) DefaultHostName() string {
	if c == nil || len(c.DockerHosts) == 0 {
		return DefaultHostName
	}
	return c.DockerHosts[0].Name
}

// AuthEnabled reports whether API/WS auth middleware should enforce a token.
func (c *Config) AuthEnabled() bool {
	return c != nil && c.AuthToken != ""
}

// MetricsEnabled reports whether historical metrics SQLite is on (ADR-015).
func (c *Config) MetricsEnabled() bool {
	if c == nil {
		return false
	}
	_, ok := ParseMetricsDB(c.MetricsDBPath)
	return ok
}

// ParseMetricsDB mirrors metricsdb.ParsePath without importing that package.
func ParseMetricsDB(raw string) (path string, enabled bool) {
	s := strings.TrimSpace(raw)
	switch strings.ToLower(s) {
	case "", "off", "false", "0", "none", "disabled":
		return "", false
	default:
		return s, true
	}
}

// SnapshotsEnabled reports whether persisted inventory snapshots are on.
func (c *Config) SnapshotsEnabled() bool {
	if c == nil {
		return false
	}
	_, ok := ParseSnapshotsDir(c.SnapshotsDir)
	return ok
}

// ParseSnapshotsDir returns path and whether snapshots are enabled.
func ParseSnapshotsDir(raw string) (path string, enabled bool) {
	s := strings.TrimSpace(raw)
	switch strings.ToLower(s) {
	case "", "off", "false", "0", "none", "disabled":
		return "", false
	default:
		return s, true
	}
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
