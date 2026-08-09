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
