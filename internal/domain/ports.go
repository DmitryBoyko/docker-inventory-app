package domain

import (
	"fmt"
	"net/netip"
	"sort"
	"strings"
)

// PortBindingInput is a transport-neutral port row before classification.
type PortBindingInput struct {
	HostIP        string
	HostPort      uint16
	ContainerPort uint16
	Protocol      string
	Published     bool // true when host port mapping exists
}

// ClassifyHostIP maps a host bind address to PortExposure (PS Split-DockerPorts).
func ClassifyHostIP(hostIP string) PortExposure {
	ip := strings.TrimSpace(hostIP)
	switch ip {
	case "", "0.0.0.0", "*", "::", "[::]":
		return PortExposurePublic
	case "127.0.0.1", "::1", "[::1]":
		return PortExposureLocalhost
	}
	// Normalize bracketed IPv6
	if strings.HasPrefix(ip, "[") && strings.HasSuffix(ip, "]") {
		ip = strings.TrimSuffix(strings.TrimPrefix(ip, "["), "]")
	}
	addr, err := netip.ParseAddr(ip)
	if err != nil {
		return PortExposureSpecific
	}
	if addr.IsUnspecified() {
		return PortExposurePublic
	}
	if addr.IsLoopback() {
		return PortExposureLocalhost
	}
	return PortExposureSpecific
}

// MapPortBindings converts raw bindings into domain Ports with exposure.
// When the same containerPort/protocol is published on multiple host IPs,
// each binding becomes its own Port row (structured API). Exposure is per row.
func MapPortBindings(in []PortBindingInput) []Port {
	if len(in) == 0 {
		return []Port{}
	}
	out := make([]Port, 0, len(in))
	for _, p := range in {
		proto := strings.ToLower(p.Protocol)
		if proto == "" {
			proto = "tcp"
		}
		row := Port{
			ContainerPort: p.ContainerPort,
			Protocol:      proto,
		}
		if !p.Published || p.HostPort == 0 {
			row.Exposure = PortExposureInternal
			out = append(out, row)
			continue
		}
		hp := p.HostPort
		row.HostPort = &hp
		row.HostIP = normalizeHostIPDisplay(p.HostIP)
		row.Exposure = ClassifyHostIP(p.HostIP)
		out = append(out, row)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].ContainerPort != out[j].ContainerPort {
			return out[i].ContainerPort < out[j].ContainerPort
		}
		return out[i].Protocol < out[j].Protocol
	})
	return out
}

func normalizeHostIPDisplay(hostIP string) string {
	ip := strings.TrimSpace(hostIP)
	if ip == "" {
		return "0.0.0.0"
	}
	return ip
}

// FormatPortsPSStyle builds PowerShell-like External/Internal display strings.
// Used for parity tests / optional UI; API primary form is structured Ports.
func FormatPortsPSStyle(ports []Port) (external, internal string) {
	type key struct {
		hostPort uint16
		dest     string
	}
	extIPs := map[key][]string{}
	var extOrder []key
	var internals []string
	seenInt := map[string]struct{}{}

	for _, p := range ports {
		if p.Exposure == PortExposureInternal || p.HostPort == nil {
			s := fmt.Sprintf("%d/%s", p.ContainerPort, p.Protocol)
			if _, ok := seenInt[s]; !ok {
				seenInt[s] = struct{}{}
				internals = append(internals, s)
			}
			continue
		}
		dest := fmt.Sprintf("%d/%s", p.ContainerPort, p.Protocol)
		k := key{hostPort: *p.HostPort, dest: dest}
		if _, ok := extIPs[k]; !ok {
			extOrder = append(extOrder, k)
		}
		extIPs[k] = appendUnique(extIPs[k], p.HostIP)
	}

	var extParts []string
	for _, k := range extOrder {
		ips := extIPs[k]
		mapping := fmt.Sprintf("%d->%s", k.hostPort, k.dest)
		extParts = append(extParts, formatExternalGroup(ips, mapping))
	}

	external = "-"
	if len(extParts) > 0 {
		external = strings.Join(extParts, " | ")
	}
	internal = "-"
	if len(internals) > 0 {
		internal = strings.Join(internals, "; ")
	}
	return external, internal
}

func formatExternalGroup(ips []string, mapping string) string {
	if len(ips) == 0 {
		return mapping
	}
	allIfaces := false
	onlyLocal := true
	extra := false
	for _, ip := range ips {
		switch ClassifyHostIP(ip) {
		case PortExposurePublic:
			allIfaces = true
			onlyLocal = false
		case PortExposureLocalhost:
			// still only-local so far
		default:
			onlyLocal = false
			extra = true
		}
	}
	if allIfaces && !extra && !onlyLocal {
		return "*:" + mapping + " [наружу]"
	}
	if onlyLocal {
		return "127.0.0.1:" + mapping + " [localhost]"
	}
	parts := make([]string, 0, len(ips))
	for _, ip := range ips {
		parts = append(parts, ip+":"+mapping)
	}
	return strings.Join(parts, "; ")
}

func appendUnique(ss []string, v string) []string {
	for _, s := range ss {
		if s == v {
			return ss
		}
	}
	return append(ss, v)
}
