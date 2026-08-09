package docker

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/moby/moby/client"
)

func TestResolveEndpoint_Explicit(t *testing.T) {
	t.Setenv(client.EnvOverrideHost, "unix:///should/not/use.sock")

	ep, err := ResolveEndpoint("tcp://127.0.0.1:2375", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if ep.Source != SourceExplicit {
		t.Fatalf("source=%s want %s", ep.Source, SourceExplicit)
	}
	if ep.Host != "tcp://127.0.0.1:2375" {
		t.Fatalf("host=%s", ep.Host)
	}
}

func TestResolveEndpoint_DockerHost(t *testing.T) {
	t.Setenv(client.EnvOverrideHost, "unix:///custom/docker.sock")

	ep, err := ResolveEndpoint("", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if ep.Source != SourceDockerHost {
		t.Fatalf("source=%s want %s", ep.Source, SourceDockerHost)
	}
	if ep.Host != "unix:///custom/docker.sock" {
		t.Fatalf("host=%s", ep.Host)
	}
}

func TestResolveEndpoint_Context(t *testing.T) {
	t.Setenv(client.EnvOverrideHost, "")
	dir := t.TempDir()

	cfg := map[string]any{"currentContext": "desktop-linux"}
	writeJSON(t, filepath.Join(dir, "config.json"), cfg)

	metaDir := filepath.Join(dir, "contexts", "meta", "abc123")
	if err := os.MkdirAll(metaDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeJSON(t, filepath.Join(metaDir, "meta.json"), map[string]any{
		"Name": "desktop-linux",
		"Endpoints": map[string]any{
			"docker": map[string]any{
				"Host": "unix:///home/user/.docker/desktop/docker.sock",
			},
		},
	})

	ep, err := ResolveEndpoint("", dir)
	if err != nil {
		t.Fatal(err)
	}
	if ep.Source != SourceContext {
		t.Fatalf("source=%s want %s", ep.Source, SourceContext)
	}
	if ep.Context != "desktop-linux" {
		t.Fatalf("context=%s", ep.Context)
	}
	if ep.Host != "unix:///home/user/.docker/desktop/docker.sock" {
		t.Fatalf("host=%s", ep.Host)
	}
}

func TestResolveEndpoint_DefaultContextFallsThrough(t *testing.T) {
	t.Setenv(client.EnvOverrideHost, "")
	dir := t.TempDir()
	writeJSON(t, filepath.Join(dir, "config.json"), map[string]any{"currentContext": "default"})

	ep, err := ResolveEndpoint("", dir)
	if err != nil {
		t.Fatal(err)
	}
	if ep.Source != SourceDefault {
		t.Fatalf("source=%s want %s", ep.Source, SourceDefault)
	}
	if ep.Host != client.DefaultDockerHost {
		t.Fatalf("host=%s want %s", ep.Host, client.DefaultDockerHost)
	}
}

func TestResolveEndpoint_InvalidExplicit(t *testing.T) {
	_, err := ResolveEndpoint("not-a-url", "")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestClassifyError_NotFound(t *testing.T) {
	msg := ClassifyError("unix:///missing.sock", os.ErrNotExist)
	if msg == "" {
		t.Fatal("empty message")
	}
}

func writeJSON(t *testing.T, path string, v any) {
	t.Helper()
	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
}
