package config

import (
	"net"
	"strings"
)

// IsLoopbackListen reports whether addr binds only to loopback (ADR-009).
// Bare ":8080" / "0.0.0.0:8080" are not loopback.
func IsLoopbackListen(addr string) bool {
	host, _, err := net.SplitHostPort(strings.TrimSpace(addr))
	if err != nil {
		host = strings.TrimSpace(addr)
	}
	switch strings.ToLower(host) {
	case "127.0.0.1", "::1", "localhost":
		return true
	default:
		return false
	}
}
