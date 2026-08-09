package domain

import "time"

// SystemInfo is a trimmed Engine info view.
type SystemInfo struct {
	ID                 string `json:"id,omitempty"`
	Name               string `json:"name,omitempty"`
	ServerVersion      string `json:"serverVersion"`
	APIVersion         string `json:"apiVersion,omitempty"`
	OS                 string `json:"os"`
	OSVersion          string `json:"osVersion,omitempty"`
	OSType             string `json:"osType,omitempty"`
	Architecture       string `json:"architecture"`
	KernelVersion      string `json:"kernelVersion,omitempty"`
	NCPU               int    `json:"cpus"`
	MemTotalBytes      int64  `json:"memoryBytes"`
	Driver             string `json:"driver,omitempty"`
	DockerRootDir      string `json:"dockerRootDir,omitempty"`
	Containers         int    `json:"containers"`
	ContainersRunning  int    `json:"containersRunning"`
	ContainersPaused   int    `json:"containersPaused"`
	ContainersStopped  int    `json:"containersStopped"`
	Images             int    `json:"images"`
}

// SystemResources is the PowerShell "grand totals" style rollup.
type SystemResources struct {
	ResourceSummary
	NetworkRxBytes  int64 `json:"networkRxBytes"`
	NetworkTxBytes  int64 `json:"networkTxBytes"`
	BlockReadBytes  int64 `json:"blockReadBytes"`
	BlockWriteBytes int64 `json:"blockWriteBytes"`
}

// ConnectionStatus describes Docker endpoint connectivity (Phase 1+).
type ConnectionStatus struct {
	Connected bool      `json:"connected"`
	Host      string    `json:"host"`
	Source    string    `json:"source"` // explicit|docker_host|context|default
	Context   string    `json:"context,omitempty"`
	APIVersion string   `json:"apiVersion,omitempty"`
	OSType    string    `json:"osType,omitempty"`
	CheckedAt time.Time `json:"checkedAt"`
	Error     string    `json:"error,omitempty"`
}
