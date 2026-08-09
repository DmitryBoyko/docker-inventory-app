package docker

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/moby/moby/client"
)

// Host source labels (ADR-010 / OpenAPI ConnectionStatus.source).
const (
	SourceExplicit   = "explicit"
	SourceDockerHost = "docker_host"
	SourceContext    = "context"
	SourceDefault    = "default"
)

// Endpoint is a resolved Docker Engine host.
type Endpoint struct {
	Host    string // e.g. unix:///var/run/docker.sock
	Source  string
	Context string // docker context name when SourceContext
}

type dockerConfigJSON struct {
	CurrentContext string `json:"currentContext"`
}

type contextMeta struct {
	Name      string `json:"Name"`
	Endpoints map[string]struct {
		Host          string `json:"Host"`
		SkipTLSVerify bool   `json:"SkipTLSVerify"`
	} `json:"Endpoints"`
}

// ResolveEndpoint implements ADR-010 discovery order.
//
//  1. explicitHost (flag / DOCKER_VISUALIZER_DOCKER_HOST)
//  2. DOCKER_HOST
//  3. current Docker context endpoint (if not "default" and Host set)
//  4. SDK platform default
func ResolveEndpoint(explicitHost, dockerConfigDir string) (Endpoint, error) {
	if h := strings.TrimSpace(explicitHost); h != "" {
		if err := validateHost(h); err != nil {
			return Endpoint{}, fmt.Errorf("explicit docker host: %w", err)
		}
		return Endpoint{Host: h, Source: SourceExplicit}, nil
	}

	if h := strings.TrimSpace(os.Getenv(client.EnvOverrideHost)); h != "" {
		if err := validateHost(h); err != nil {
			return Endpoint{}, fmt.Errorf("DOCKER_HOST: %w", err)
		}
		return Endpoint{Host: h, Source: SourceDockerHost}, nil
	}

	cfgDir, err := resolveDockerConfigDir(dockerConfigDir)
	if err != nil {
		// Missing config dir is fine — fall through to default.
		if !errors.Is(err, os.ErrNotExist) {
			return Endpoint{}, err
		}
	} else {
		ep, ok, err := endpointFromContext(cfgDir)
		if err != nil {
			return Endpoint{}, err
		}
		if ok {
			return ep, nil
		}
	}

	return Endpoint{Host: client.DefaultDockerHost, Source: SourceDefault}, nil
}

func validateHost(host string) error {
	_, err := client.ParseHostURL(host)
	if err != nil {
		return fmt.Errorf("invalid host URL %q: %w", host, err)
	}
	return nil
}

func resolveDockerConfigDir(override string) (string, error) {
	if override != "" {
		return override, nil
	}
	if v := os.Getenv("DOCKER_CONFIG"); v != "" {
		return v, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home for docker config: %w", err)
	}
	dir := filepath.Join(home, ".docker")
	if st, err := os.Stat(dir); err != nil {
		return "", err
	} else if !st.IsDir() {
		return "", fmt.Errorf("%s is not a directory", dir)
	}
	return dir, nil
}

func endpointFromContext(configDir string) (Endpoint, bool, error) {
	cfgPath := filepath.Join(configDir, "config.json")
	raw, err := os.ReadFile(cfgPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Endpoint{}, false, nil
		}
		return Endpoint{}, false, fmt.Errorf("read docker config: %w", err)
	}

	var cfg dockerConfigJSON
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return Endpoint{}, false, fmt.Errorf("parse docker config.json: %w", err)
	}

	ctxName := strings.TrimSpace(cfg.CurrentContext)
	if ctxName == "" || ctxName == "default" {
		return Endpoint{}, false, nil
	}

	host, err := findContextHost(configDir, ctxName)
	if err != nil {
		return Endpoint{}, false, err
	}
	if host == "" {
		return Endpoint{}, false, nil
	}
	if err := validateHost(host); err != nil {
		return Endpoint{}, false, fmt.Errorf("docker context %q: %w", ctxName, err)
	}
	return Endpoint{Host: host, Source: SourceContext, Context: ctxName}, true, nil
}

func findContextHost(configDir, contextName string) (string, error) {
	metaRoot := filepath.Join(configDir, "contexts", "meta")
	entries, err := os.ReadDir(metaRoot)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", nil
		}
		return "", fmt.Errorf("read docker contexts: %w", err)
	}

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		metaPath := filepath.Join(metaRoot, e.Name(), "meta.json")
		raw, err := os.ReadFile(metaPath)
		if err != nil {
			continue
		}
		var meta contextMeta
		if err := json.Unmarshal(raw, &meta); err != nil {
			continue
		}
		if meta.Name != contextName {
			continue
		}
		if ep, ok := meta.Endpoints["docker"]; ok {
			return strings.TrimSpace(ep.Host), nil
		}
		return "", nil
	}
	return "", nil
}
