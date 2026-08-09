package domain

// VolumeUsage is per-volume disk usage with availability (ADR-011).
type VolumeUsage struct {
	ByteMetric
	Links *int64 `json:"links"` // null if RefCount unavailable (-1)
}

// Volume is a Docker volume plus optional usage and reverse links.
type Volume struct {
	Name       string            `json:"name"`
	Driver     string            `json:"driver"`
	Mountpoint string            `json:"mountpoint,omitempty"`
	Scope      string            `json:"scope,omitempty"`
	Labels     map[string]string `json:"labels,omitempty"`
	Usage      VolumeUsage       `json:"usage"`
	Containers []string          `json:"containers,omitempty"`
	Stacks     []string          `json:"stacks,omitempty"`
	Shared     bool              `json:"shared,omitempty"`
}
