package domain

import "time"

// ContainerState mirrors Docker container state strings used by the Engine API.
type ContainerState string

const (
	ContainerStateCreated    ContainerState = "created"
	ContainerStateRunning    ContainerState = "running"
	ContainerStatePaused     ContainerState = "paused"
	ContainerStateRestarting ContainerState = "restarting"
	ContainerStateRemoving   ContainerState = "removing"
	ContainerStateExited     ContainerState = "exited"
	ContainerStateDead       ContainerState = "dead"
)

// HealthState is normalized health.
type HealthState string

const (
	HealthNone      HealthState = "none"
	HealthStarting  HealthState = "starting"
	HealthHealthy   HealthState = "healthy"
	HealthUnhealthy HealthState = "unhealthy"
	HealthUnknown   HealthState = "unknown"
)

// PortExposure classifies published ports (parity with PowerShell Split-DockerPorts).
type PortExposure string

const (
	PortExposurePublic    PortExposure = "public"
	PortExposureLocalhost PortExposure = "localhost"
	PortExposureSpecific  PortExposure = "specific"
	PortExposureInternal  PortExposure = "internal"
)

// Port is a structured port binding.
type Port struct {
	HostIP        string       `json:"hostIP,omitempty"`
	HostPort      *uint16      `json:"hostPort,omitempty"`
	ContainerPort uint16       `json:"containerPort"`
	Protocol      string       `json:"protocol"`
	Exposure      PortExposure `json:"exposure"`
}

// NetworkEndpoint is a container attachment to a network.
type NetworkEndpoint struct {
	NetworkID   string `json:"networkId,omitempty"`
	NetworkName string `json:"networkName"`
	IPAddress   string `json:"ipAddress,omitempty"`
	Gateway     string `json:"gateway,omitempty"`
	MacAddress  string `json:"macAddress,omitempty"`
}

// MountType is a Docker mount type.
type MountType string

const (
	MountTypeVolume MountType = "volume"
	MountTypeBind   MountType = "bind"
	MountTypeTmpfs  MountType = "tmpfs"
	MountTypeNpipe  MountType = "npipe"
)

// Mount describes a container mount.
type Mount struct {
	Type        MountType `json:"type"`
	Name        string    `json:"name,omitempty"`
	Source      string    `json:"source,omitempty"`
	Destination string    `json:"destination"`
	RW          bool      `json:"rw"`
}

// ContainerStats is a fast-changing sample.
//
// Units:
//   - CPUPercent: percent (Docker CLI compatible)
//   - Memory*: bytes / percent
//   - Network/Block: cumulative bytes since container start
type ContainerStats struct {
	Timestamp         time.Time `json:"timestamp"`
	CPUPercent        float64   `json:"cpuPercent"`
	MemoryBytes       int64     `json:"memoryBytes"`
	MemoryLimitBytes  int64     `json:"memoryLimitBytes"`
	MemoryPercent     float64   `json:"memoryPercent"`
	NetworkRxBytes    int64     `json:"networkRxBytes"`
	NetworkTxBytes    int64     `json:"networkTxBytes"`
	BlockReadBytes    int64     `json:"blockReadBytes"`
	BlockWriteBytes   int64     `json:"blockWriteBytes"`
	CountersAvailable bool      `json:"countersAvailable"`
}

// Container is the domain inventory row (PowerShell parity + structured fields).
type Container struct {
	ID              string           `json:"id"`
	IDShort         string           `json:"idShort"`
	Name            string           `json:"name"`
	Stack           string           `json:"stack"`
	Service         *string          `json:"service"`
	ContainerNumber *int             `json:"containerNumber,omitempty"`
	Image           string           `json:"image"`
	ImageID         string           `json:"imageId,omitempty"`
	State           ContainerState   `json:"state"`
	Status          string           `json:"status"`
	Health          HealthState      `json:"health"`
	RestartCount    int              `json:"restartCount"`
	StartedAt       *time.Time       `json:"startedAt,omitempty"`
	FinishedAt      *time.Time       `json:"finishedAt,omitempty"`
	UptimeSeconds   *int64           `json:"uptimeSeconds,omitempty"`
	Ports             []Port              `json:"ports"`
	ExternalExposure  []ExternalExposure  `json:"externalExposure"`
	ExposureScope     ExposureScope       `json:"exposureScope"` // widest published scope for badges
	Endpoints         []NetworkEndpoint   `json:"endpoints"`
	Mounts            []Mount             `json:"mounts"`
	WritableLayer     ByteMetric          `json:"writableLayer"`
	Stats             *ContainerStats     `json:"stats,omitempty"`
	Labels            map[string]string   `json:"labels,omitempty"`
}

// ContainerRef is a lightweight pointer used inside stacks/services.
type ContainerRef struct {
	ID      string         `json:"id"`
	IDShort string         `json:"idShort"`
	Name    string         `json:"name"`
	State   ContainerState `json:"state"`
	Health  HealthState    `json:"health"`
}
