package config

import (
	"fmt"
	"strings"
)

const (
	EnvDockerHosts = "DOCKER_VISUALIZER_DOCKER_HOSTS"
	DefaultHostName = "default"
)

// HostSpec is one configured Docker endpoint.
type HostSpec struct {
	Name string
	URL  string // empty ⇒ ADR-010 discovery (only valid for a single default host)
}

// ParseDockerHosts parses "name=url,name2=url2" (comma or semicolon separated).
// Empty input yields a single default host with URL from dockerHostFallback.
func ParseDockerHosts(raw, dockerHostFallback string) ([]HostSpec, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return []HostSpec{{Name: DefaultHostName, URL: strings.TrimSpace(dockerHostFallback)}}, nil
	}
	parts := strings.FieldsFunc(raw, func(r rune) bool { return r == ',' || r == ';' })
	out := make([]HostSpec, 0, len(parts))
	seen := map[string]struct{}{}
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		name, url, ok := strings.Cut(p, "=")
		name = strings.TrimSpace(name)
		url = strings.TrimSpace(url)
		if !ok || name == "" || url == "" {
			return nil, fmt.Errorf("invalid --docker-hosts entry %q (want name=url)", p)
		}
		if !validHostName(name) {
			return nil, fmt.Errorf("invalid host name %q (use [a-zA-Z0-9_.-])", name)
		}
		key := strings.ToLower(name)
		if _, dup := seen[key]; dup {
			return nil, fmt.Errorf("duplicate host name %q", name)
		}
		seen[key] = struct{}{}
		out = append(out, HostSpec{Name: name, URL: url})
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("empty --docker-hosts")
	}
	return out, nil
}

func validHostName(s string) bool {
	if s == "" || len(s) > 64 {
		return false
	}
	for i, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
		case r == '_' || r == '-' || r == '.':
			if i == 0 {
				return false
			}
		default:
			return false
		}
	}
	return true
}
