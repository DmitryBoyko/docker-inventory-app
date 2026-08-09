package domain

import (
	"regexp"
	"strings"
)

var statusHealthRe = regexp.MustCompile(`\(([^)]+)\)`)

// NormalizeHealth maps Engine health status strings to HealthState.
func NormalizeHealth(raw string) HealthState {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", "none", "-":
		return HealthNone
	case "healthy":
		return HealthHealthy
	case "unhealthy":
		return HealthUnhealthy
	case "starting", "health: starting":
		return HealthStarting
	default:
		return HealthUnknown
	}
}

// HealthFromStatusFallback extracts health from docker ps Status text
// (e.g. "Up 3 minutes (healthy)"), matching PowerShell behavior.
func HealthFromStatusFallback(status string) HealthState {
	m := statusHealthRe.FindStringSubmatch(status)
	if len(m) < 2 {
		return HealthNone
	}
	inner := strings.ToLower(strings.TrimSpace(m[1]))
	switch inner {
	case "healthy":
		return HealthHealthy
	case "unhealthy":
		return HealthUnhealthy
	case "health: starting", "starting":
		return HealthStarting
	default:
		return HealthNone
	}
}

// ResolveHealth prefers inspect/list health; falls back to Status text when none/unknown.
func ResolveHealth(primary HealthState, statusText string) HealthState {
	if primary == HealthHealthy || primary == HealthUnhealthy || primary == HealthStarting {
		return primary
	}
	fb := HealthFromStatusFallback(statusText)
	if fb != HealthNone {
		return fb
	}
	if primary == HealthNone || primary == "" {
		return HealthNone
	}
	return primary
}
