package config

import (
	"testing"
	"time"
)

func TestLoad_Defaults(t *testing.T) {
	t.Setenv(EnvListen, "")
	t.Setenv(EnvDockerHost, "")
	t.Setenv(EnvTimeout, "")

	cfg, err := Load(nil, "dev")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ListenAddr != "127.0.0.1:8080" {
		t.Fatalf("listen=%s", cfg.ListenAddr)
	}
	if cfg.DockerTimeout != 5*time.Second {
		t.Fatalf("timeout=%s", cfg.DockerTimeout)
	}
	if cfg.InventoryInterval != 10*time.Second {
		t.Fatalf("inventory=%s", cfg.InventoryInterval)
	}
	if cfg.StatsInterval != time.Second {
		t.Fatalf("stats=%s", cfg.StatsInterval)
	}
	if cfg.SystemInterval != 15*time.Second {
		t.Fatalf("system=%s", cfg.SystemInterval)
	}
	if cfg.MetricsDBPath != "data/metrics.db" {
		t.Fatalf("metrics-db=%s", cfg.MetricsDBPath)
	}
	if !cfg.MetricsEnabled() {
		t.Fatal("metrics should be enabled by default")
	}
	if cfg.MetricsInterval != 10*time.Second || cfg.MetricsRetention != 24*time.Hour {
		t.Fatalf("metrics interval/retention=%s/%s", cfg.MetricsInterval, cfg.MetricsRetention)
	}
}

func TestLoad_MetricsOff(t *testing.T) {
	cfg, err := Load([]string{"-metrics-db", "off"}, "dev")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.MetricsEnabled() {
		t.Fatal("expected disabled")
	}
}

func TestLoad_Flags(t *testing.T) {
	cfg, err := Load([]string{
		"-listen", "127.0.0.1:9090",
		"-docker-host", "npipe:////./pipe/docker_engine",
		"-docker-timeout", "2s",
		"-auth-token", "secret",
	}, "dev")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ListenAddr != "127.0.0.1:9090" {
		t.Fatalf("listen=%s", cfg.ListenAddr)
	}
	if cfg.DockerHost != "npipe:////./pipe/docker_engine" {
		t.Fatalf("host=%s", cfg.DockerHost)
	}
	if cfg.DockerTimeout != 2*time.Second {
		t.Fatalf("timeout=%s", cfg.DockerTimeout)
	}
	if !cfg.AuthEnabled() || cfg.AuthToken != "secret" {
		t.Fatalf("auth=%q", cfg.AuthToken)
	}
}

func TestLoad_NonLoopbackRequiresToken(t *testing.T) {
	t.Setenv(EnvAuthToken, "")
	_, err := Load([]string{"-listen", "0.0.0.0:8080"}, "dev")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestLoad_NonLoopbackWithToken(t *testing.T) {
	cfg, err := Load([]string{"-listen", "0.0.0.0:8080", "-auth-token", "x"}, "dev")
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.AuthEnabled() {
		t.Fatal("expected auth")
	}
}
